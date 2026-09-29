//go:build !linux

package control

func validateSocketDir(string) error { return nil }
