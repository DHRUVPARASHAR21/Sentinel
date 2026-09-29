//go:build !unix

package process

import "os/exec"

func configureCmd(*exec.Cmd) {}
