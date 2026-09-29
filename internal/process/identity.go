package process

import (
	"errors"

	"github.com/sentinel/sentinel/internal/procfs"
)

// Identity binds a PID to its procfs creation tick, preventing a later PID
// reuse from being mistaken for the original process.
type Identity struct {
	PID            int
	StartTimeTicks uint64
}
type Inspector interface {
	Inspect(int) (procfs.Process, error)
}

func CaptureIdentity(inspector Inspector, pid int) (Identity, error) {
	p, err := inspector.Inspect(pid)
	if err != nil {
		return Identity{}, err
	}
	return Identity{PID: p.PID, StartTimeTicks: p.StartTimeTicks}, nil
}
func (i Identity) Verify(inspector Inspector) error {
	p, err := inspector.Inspect(i.PID)
	if err != nil {
		return err
	}
	if p.StartTimeTicks != i.StartTimeTicks {
		return errors.New("process: pid identity changed")
	}
	return nil
}
