//go:build !unix

package process

import (
	"errors"
	"os"
)

var errProcessDone = errors.New("process already exited")

// Windows has no SIGTERM equivalent for an arbitrary child; force termination
// is the only reliable fallback. Linux behavior is defined in signals_unix.go.
func sendTerminate(p *os.Process) error { return p.Kill() }
func sendKill(p *os.Process) error      { return p.Kill() }
