//go:build linux

package control

import (
	"fmt"
	"os"
	"syscall"
)

func validateSocketDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("control: cannot verify socket directory ownership")
	}
	if stat.Uid != uint32(os.Geteuid()) || info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("control: unsafe socket directory %s", path)
	}
	return nil
}
