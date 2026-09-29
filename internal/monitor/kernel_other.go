//go:build !linux

package monitor

import (
	"errors"
	"time"
)

type ProcStat struct{ Path string }

func (ProcStat) TotalCPUTicks() (uint64, error) {
	return 0, errors.New("monitor: Linux procfs required")
}
func (ProcStat) Uptime() (time.Duration, error) {
	return 0, errors.New("monitor: Linux procfs required")
}
