// Package monitor derives process resource samples from procfs observations.
package monitor

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"time"

	"github.com/sentinel/sentinel/internal/procfs"
)

// Inspector is the narrow procfs boundary required by Monitor.
type Inspector interface {
	Inspect(pid int) (procfs.Process, error)
}

// KernelCounters supplies cumulative Linux CPU ticks from /proc/stat.
type KernelCounters interface {
	TotalCPUTicks() (uint64, error)
	Uptime() (time.Duration, error)
}

// Sample is one non-atomic resource observation.
type Sample struct {
	Process         procfs.Process
	At              time.Time
	Uptime          time.Duration
	CPUPercent      float64
	ProcessCPUTicks uint64
	KernelCPUTicks  uint64
}

// Monitor owns no goroutines. Call Sample on the caller's chosen schedule.
type Monitor struct {
	inspector      Inspector
	kernel         KernelCounters
	now            func() time.Time
	cpus           int
	ticksPerSecond uint64
	mu             sync.Mutex
	previous       map[int]Sample
}

func New(inspector Inspector, kernel KernelCounters) *Monitor {
	return NewWithTicks(inspector, kernel, 100)
}

// NewWithTicks sets Linux USER_HZ used by /proc/<pid>/stat starttime. Linux
// deployments commonly use 100; callers on a different ABI must provide its
// actual sysconf(_SC_CLK_TCK) value.
func NewWithTicks(inspector Inspector, kernel KernelCounters, ticksPerSecond uint64) *Monitor {
	if ticksPerSecond == 0 {
		ticksPerSecond = 100
	}
	return &Monitor{inspector: inspector, kernel: kernel, now: time.Now, cpus: runtime.NumCPU(), ticksPerSecond: ticksPerSecond, previous: make(map[int]Sample)}
}

// Sample reads current counters and derives CPU percentage from deltas. See
// docs/architecture.md for the formula and its host-level assumptions.
func (m *Monitor) Sample(pid int) (Sample, error) {
	process, err := m.inspector.Inspect(pid)
	if err != nil {
		return Sample{}, err
	}
	total, err := m.kernel.TotalCPUTicks()
	if err != nil {
		return Sample{}, err
	}
	uptime, err := m.kernel.Uptime()
	if err != nil {
		return Sample{}, err
	}
	now := m.now()
	processTicks := process.UserCPUTicks + process.SystemCPUTicks
	sample := Sample{Process: process, At: now, ProcessCPUTicks: processTicks, KernelCPUTicks: total}
	startedAfterBoot := time.Duration(process.StartTimeTicks) * time.Second / time.Duration(m.ticksPerSecond)
	if uptime > startedAfterBoot {
		sample.Uptime = uptime - startedAfterBoot
	}
	if now.Before(time.Unix(0, 0)) {
		return Sample{}, errors.New("monitor: invalid clock")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if previous, ok := m.previous[pid]; ok && total > previous.KernelCPUTicks && processTicks >= previous.ProcessCPUTicks {
		sample.CPUPercent = float64(processTicks-previous.ProcessCPUTicks) / float64(total-previous.KernelCPUTicks) * float64(m.cpus) * 100
	}
	m.previous[pid] = sample
	return sample, nil
}

// Forget removes retained counters after a process exits.
func (m *Monitor) Forget(pid int) {
	m.mu.Lock()
	delete(m.previous, pid)
	m.mu.Unlock()
}

// Run samples at interval until ctx is cancelled. The callback runs in this
// method's goroutine and must return promptly.
func (m *Monitor) Run(ctx context.Context, pid int, interval time.Duration, callback func(Sample, error)) error {
	if interval <= 0 {
		return errors.New("monitor: interval must be positive")
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		sample, err := m.Sample(pid)
		callback(sample, err)
		select {
		case <-ctx.Done():
			m.Forget(pid)
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
