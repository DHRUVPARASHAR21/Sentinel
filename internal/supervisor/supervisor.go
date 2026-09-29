// Package supervisor owns one configured child lifecycle.
package supervisor

import (
	"context"
	"errors"
	"os"
	"sync"
	"time"

	"github.com/sentinel/sentinel/internal/process"
)

type RestartPolicy string

const (
	RestartNever     RestartPolicy = "never"
	RestartAlways    RestartPolicy = "always"
	RestartOnFailure RestartPolicy = "on-failure"
)

type State string

const (
	StateStopped State = "stopped"
	StateRunning State = "running"
	StateBackoff State = "backoff"
	StateFailed  State = "failed"
)

type Config struct {
	Process            process.Spec
	Restart            RestartPolicy
	MaxRetries         int
	InitialBackoff     time.Duration
	MaxBackoff         time.Duration
	ResetWindow        time.Duration
	TerminationTimeout time.Duration
}
type Status struct {
	State    State
	PID      int
	Restarts int
	Retries  int
	LastExit error
}

type Supervisor struct {
	cfg      Config
	mu       sync.RWMutex
	status   Status
	ctx      context.Context
	cancel   context.CancelFunc
	done     chan struct{}
	stopOnce sync.Once
}

func New(cfg Config) (*Supervisor, error) {
	if cfg.Restart == "" {
		cfg.Restart = RestartNever
	}
	if cfg.Restart != RestartNever && cfg.Restart != RestartAlways && cfg.Restart != RestartOnFailure {
		return nil, errors.New("supervisor: invalid restart policy")
	}
	if cfg.MaxRetries < 0 {
		return nil, errors.New("supervisor: negative retry limit")
	}
	if cfg.InitialBackoff <= 0 {
		cfg.InitialBackoff = time.Millisecond
	}
	if cfg.MaxBackoff <= 0 {
		cfg.MaxBackoff = cfg.InitialBackoff
	}
	if cfg.MaxBackoff < cfg.InitialBackoff {
		return nil, errors.New("supervisor: backoff maximum below initial")
	}
	if cfg.TerminationTimeout <= 0 {
		cfg.TerminationTimeout = time.Second
	}
	return &Supervisor{cfg: cfg, status: Status{State: StateStopped}, done: make(chan struct{})}, nil
}

func (s *Supervisor) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.ctx != nil {
		s.mu.Unlock()
		return errors.New("supervisor: already started")
	}
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.mu.Unlock()
	go s.run()
	return nil
}
func (s *Supervisor) Wait()          { <-s.done }
func (s *Supervisor) Status() Status { s.mu.RLock(); defer s.mu.RUnlock(); return s.status }

func (s *Supervisor) Stop(ctx context.Context) error {
	s.mu.RLock()
	cancel := s.cancel
	s.mu.RUnlock()
	if cancel == nil {
		return nil
	}
	s.stopOnce.Do(cancel)
	select {
	case <-s.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// HandleSignal maps TERM/INT to graceful stop. HUP is intentionally a no-op
// until configuration reload is introduced in milestone 5.
func (s *Supervisor) HandleSignal(ctx context.Context, signal os.Signal) error {
	if isReloadSignal(signal) {
		return nil
	}
	if isStopSignal(signal) {
		return s.Stop(ctx)
	}
	return errors.New("supervisor: unsupported signal")
}

func (s *Supervisor) run() {
	defer close(s.done)
	defer func() {
		status := s.Status()
		if status.State != StateFailed {
			status.State = StateStopped
		}
		status.PID = 0
		s.setStatus(status)
	}()
	var retries int
	for {
		if s.ctx.Err() != nil {
			return
		}
		child, err := process.Start(s.cfg.Process)
		if err != nil {
			if retries >= s.cfg.MaxRetries {
				s.setStatus(Status{State: StateFailed, Restarts: s.Status().Restarts, Retries: retries, LastExit: err})
				return
			}
			retries++
			s.mu.Lock()
			restarts := s.status.Restarts + 1
			s.mu.Unlock()
			s.setStatus(Status{State: StateBackoff, Restarts: restarts, Retries: retries, LastExit: err})
			if !s.sleep(s.backoff(retries)) {
				return
			}
			continue
		}
		s.setStatus(Status{State: StateRunning, PID: child.PID(), Restarts: s.Status().Restarts, Retries: retries})
		started := time.Now()
		exit := s.waitChild(child)
		if s.ctx.Err() != nil {
			return
		}
		failed := exit != nil
		should := s.cfg.Restart == RestartAlways || (s.cfg.Restart == RestartOnFailure && failed)
		if !should {
			s.setStatus(Status{State: StateStopped, Restarts: s.Status().Restarts, Retries: retries, LastExit: exit})
			return
		}
		if s.cfg.ResetWindow > 0 && time.Since(started) >= s.cfg.ResetWindow {
			retries = 0
		}
		if retries >= s.cfg.MaxRetries {
			s.setStatus(Status{State: StateFailed, Restarts: s.Status().Restarts, Retries: retries, LastExit: exit})
			return
		}
		retries++
		s.mu.Lock()
		restarts := s.status.Restarts + 1
		s.mu.Unlock()
		s.setStatus(Status{State: StateBackoff, Restarts: restarts, Retries: retries, LastExit: exit})
		if !s.sleep(s.backoff(retries)) {
			return
		}
	}
}

func (s *Supervisor) waitChild(child *process.Child) error {
	result := make(chan error, 1)
	go func() { result <- child.Wait() }()
	select {
	case err := <-result:
		return err
	case <-s.ctx.Done():
		_ = child.Terminate(context.Background(), s.cfg.TerminationTimeout)
		return <-result
	}
}
func (s *Supervisor) sleep(d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-s.ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
func (s *Supervisor) backoff(retry int) time.Duration {
	d := s.cfg.InitialBackoff
	for i := 1; i < retry && d < s.cfg.MaxBackoff; i++ {
		if d > s.cfg.MaxBackoff/2 {
			return s.cfg.MaxBackoff
		}
		d *= 2
	}
	return d
}
func (s *Supervisor) setStatus(status Status) { s.mu.Lock(); s.status = status; s.mu.Unlock() }
