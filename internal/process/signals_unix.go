//go:build unix

package process

import (
	"errors"
	"os"
	"syscall"
)

var errProcessDone = errors.New("process already exited")

func sendTerminate(p *os.Process) error {
	if err := p.Signal(syscall.SIGTERM); errors.Is(err, os.ErrProcessDone) {
		return errProcessDone
	}
	return err
}
func sendKill(p *os.Process) error {
	if err := p.Signal(syscall.SIGKILL); errors.Is(err, os.ErrProcessDone) {
		return errProcessDone
	}
	return err
}
