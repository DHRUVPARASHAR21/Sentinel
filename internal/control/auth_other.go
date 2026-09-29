//go:build !linux

package control

import "net"

func authorize(net.Conn) error { return nil }
