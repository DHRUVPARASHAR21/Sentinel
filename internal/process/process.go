// Package process owns child-process launch, waiting, and bounded termination.
package process

import (
	"context"
	"errors"
	"os/exec"
	"sync"
	"time"
)

type Spec struct {
	Path string
	Args []string
	Env  []string
	Dir  string
}

type Child struct {
	cmd  *exec.Cmd
	done chan struct{}
	mu   sync.RWMutex
	err  error
}

func Start(spec Spec) (*Child, error) {
	if spec.Path == "" {
		return nil, errors.New("process: executable path is required")
	}
	cmd := exec.Command(spec.Path, spec.Args...)
	cmd.Env, cmd.Dir = spec.Env, spec.Dir
	configureCmd(cmd)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	child := &Child{cmd: cmd, done: make(chan struct{})}
	go func() {
		err := cmd.Wait()
		child.mu.Lock()
		child.err = err
		child.mu.Unlock()
		close(child.done)
	}()
	return child, nil
}

func (c *Child) PID() int { return c.cmd.Process.Pid }
func (c *Child) Wait() error {
	<-c.done
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.err
}

// Terminate sends graceful then forced termination. It never starts a goroutine:
// the already-owned Wait goroutine is the sole waiter.
func (c *Child) Terminate(ctx context.Context, grace time.Duration) error {
	if err := sendTerminate(c.cmd.Process); err != nil && !errors.Is(err, errProcessDone) {
		return err
	}
	if grace <= 0 {
		grace = time.Second
	}
	timer := time.NewTimer(grace)
	defer timer.Stop()
	select {
	case <-c.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		if err := sendKill(c.cmd.Process); err != nil && !errors.Is(err, errProcessDone) {
			return err
		}
		<-c.done
		return nil
	}
}
