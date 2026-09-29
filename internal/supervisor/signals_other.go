//go:build !unix

package supervisor

import "os"

func isStopSignal(s os.Signal) bool { return s == os.Interrupt }
func isReloadSignal(os.Signal) bool { return false }
