//go:build unix

package process

import (
	"errors"
	"os"
	"syscall"
)

var errProcessDone = errors.New("process already exited")

func sendTerminate(p *os.Process) error {
	if err := syscall.Kill(-p.Pid, syscall.SIGTERM); errors.Is(err, os.ErrProcessDone) || err == syscall.ESRCH {
		return errProcessDone
	}
	return err
}
func sendKill(p *os.Process) error {
	if err := syscall.Kill(-p.Pid, syscall.SIGKILL); errors.Is(err, os.ErrProcessDone) || err == syscall.ESRCH {
		return errProcessDone
	}
	return err
}
