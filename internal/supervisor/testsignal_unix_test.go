//go:build unix

package supervisor

import "syscall"

func terminationSignal() syscall.Signal { return syscall.SIGTERM }
