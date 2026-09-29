//go:build unix

package supervisor

import (
	"os"
	"syscall"
)

func isStopSignal(s os.Signal) bool   { return s == syscall.SIGTERM || s == syscall.SIGINT }
func isReloadSignal(s os.Signal) bool { return s == syscall.SIGHUP }
