//go:build linux

package control

import (
	"fmt"
	"net"
	"os"
	"syscall"
)

func authorize(conn net.Conn) error {
	u, ok := conn.(*net.UnixConn)
	if !ok {
		return fmt.Errorf("control: non-Unix connection")
	}
	raw, err := u.SyscallConn()
	if err != nil {
		return err
	}
	var uid uint32
	var credentialErr error
	err = raw.Control(func(fd uintptr) {
		cred, e := syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
		if e != nil {
			credentialErr = e
			return
		}
		uid = cred.Uid
	})
	if err != nil {
		return err
	}
	if credentialErr != nil {
		return credentialErr
	}
	if uid != uint32(os.Geteuid()) {
		return fmt.Errorf("control: unauthorized local caller")
	}
	return nil
}
