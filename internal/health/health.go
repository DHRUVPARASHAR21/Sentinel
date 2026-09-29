// Package health provides bounded process, TCP, and HTTP health checks.
package health

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
	"time"
)

type Kind string

const (
	Process Kind = "process"
	TCP     Kind = "tcp"
	HTTP    Kind = "http"
)

type Check struct {
	Kind                               Kind
	Address                            string
	URL                                string
	Interval, Timeout                  time.Duration
	SuccessThreshold, FailureThreshold int
	Process                            func() bool
}
type Result struct {
	Healthy                                bool
	ConsecutiveSuccess, ConsecutiveFailure int
	LastError                              error
}
type Runner struct {
	check  Check
	mu     sync.RWMutex
	result Result
}

func New(c Check) (*Runner, error) {
	if c.Interval <= 0 || c.Timeout <= 0 {
		return nil, errors.New("health: interval and timeout must be positive")
	}
	if c.SuccessThreshold <= 0 {
		c.SuccessThreshold = 1
	}
	if c.FailureThreshold <= 0 {
		c.FailureThreshold = 1
	}
	if c.Kind != Process && c.Kind != TCP && c.Kind != HTTP {
		return nil, errors.New("health: unsupported check")
	}
	return &Runner{check: c}, nil
}
func (r *Runner) Result() Result { r.mu.RLock(); defer r.mu.RUnlock(); return r.result }
func (r *Runner) Run(ctx context.Context, onChange func(Result)) {
	ticker := time.NewTicker(r.check.Interval)
	defer ticker.Stop()
	for {
		r.runOnce(ctx, onChange)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
func (r *Runner) runOnce(parent context.Context, onChange func(Result)) {
	ctx, cancel := context.WithTimeout(parent, r.check.Timeout)
	defer cancel()
	err := r.probe(ctx)
	r.mu.Lock()
	old := r.result.Healthy
	if err == nil {
		r.result.ConsecutiveSuccess++
		r.result.ConsecutiveFailure = 0
		r.result.LastError = nil
		if r.result.ConsecutiveSuccess >= r.check.SuccessThreshold {
			r.result.Healthy = true
		}
	} else {
		r.result.ConsecutiveFailure++
		r.result.ConsecutiveSuccess = 0
		r.result.LastError = err
		if r.result.ConsecutiveFailure >= r.check.FailureThreshold {
			r.result.Healthy = false
		}
	}
	changed := old != r.result.Healthy
	value := r.result
	r.mu.Unlock()
	if changed && onChange != nil {
		onChange(value)
	}
}
func (r *Runner) probe(ctx context.Context) error {
	switch r.check.Kind {
	case Process:
		if r.check.Process != nil && r.check.Process() {
			return nil
		}
		return errors.New("process is not running")
	case TCP:
		var d net.Dialer
		c, e := d.DialContext(ctx, "tcp", r.check.Address)
		if e == nil {
			c.Close()
		}
		return e
	case HTTP:
		req, e := http.NewRequestWithContext(ctx, http.MethodGet, r.check.URL, nil)
		if e != nil {
			return e
		}
		resp, e := (&http.Client{}).Do(req)
		if e != nil {
			return e
		}
		defer resp.Body.Close()
		if resp.StatusCode/100 != 2 {
			return errors.New("unexpected HTTP status")
		}
		return nil
	}
	return errors.New("unsupported health check")
}
