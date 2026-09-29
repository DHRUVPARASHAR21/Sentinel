//go:build linux

package monitor

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// ProcStat reads the aggregate cpu row from /proc/stat.
type ProcStat struct{ Path string }

func (p ProcStat) TotalCPUTicks() (uint64, error) {
	path := p.Path
	if path == "" {
		path = "/proc/stat"
	}
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	if !s.Scan() {
		if err := s.Err(); err != nil {
			return 0, err
		}
		return 0, fmt.Errorf("monitor: missing cpu counters")
	}
	fields := strings.Fields(s.Text())
	if len(fields) < 2 || fields[0] != "cpu" {
		return 0, fmt.Errorf("monitor: malformed cpu counters")
	}
	var total uint64
	for _, field := range fields[1:] {
		value, err := strconv.ParseUint(field, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("monitor: malformed cpu counter: %w", err)
		}
		if ^uint64(0)-total < value {
			return 0, fmt.Errorf("monitor: cpu counter overflow")
		}
		total += value
	}
	return total, nil
}

func (p ProcStat) Uptime() (time.Duration, error) {
	path := "/proc/uptime"
	if p.Path != "" {
		path = strings.TrimSuffix(p.Path, "stat") + "uptime"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	fields := strings.Fields(string(data))
	if len(fields) < 1 {
		return 0, fmt.Errorf("monitor: malformed uptime")
	}
	seconds, err := strconv.ParseFloat(fields[0], 64)
	if err != nil || seconds < 0 {
		return 0, fmt.Errorf("monitor: malformed uptime")
	}
	return time.Duration(seconds * float64(time.Second)), nil
}
