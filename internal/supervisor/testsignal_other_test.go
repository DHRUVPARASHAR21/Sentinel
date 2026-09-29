//go:build !unix

package supervisor

import "os"

func terminationSignal() os.Signal { return os.Interrupt }
