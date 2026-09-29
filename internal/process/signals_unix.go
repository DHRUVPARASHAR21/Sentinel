//go:build unix

package process

import (
	"errors"
	"os"
	"syscall"
)

var errProcessDone = errors.New("process already exited")

func sendTerminate(p *os.Process) error {
	err := syscall.Kill(-p.Pid, syscall.SIGTERM)
	if errors.Is(err, os.ErrProcessDone) || err == syscall.ESRCH {
		return errProcessDone
	}
	return err
}
func sendKill(p *os.Process) error {
	err := syscall.Kill(-p.Pid, syscall.SIGKILL)
	if errors.Is(err, os.ErrProcessDone) || err == syscall.ESRCH {
		return errProcessDone
	}
	return err
}
