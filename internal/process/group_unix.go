//go:build unix

package process

import "os/exec"
import "syscall"

func configureCmd(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} }
