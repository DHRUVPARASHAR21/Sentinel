package monitor

import (
	"context"
	"testing"
	"time"

	"github.com/sentinel/sentinel/internal/procfs"
)

type fakeInspector struct {
	values []procfs.Process
	i      int
}

func (f *fakeInspector) Inspect(int) (procfs.Process, error) {
	v := f.values[f.i]
	f.i++
	return v, nil
}

type fakeKernel struct {
	values []uint64
	i      int
	uptime time.Duration
}

func (f *fakeKernel) TotalCPUTicks() (uint64, error) { v := f.values[f.i]; f.i++; return v, nil }
func (f *fakeKernel) Uptime() (time.Duration, error) { return f.uptime, nil }

func TestSampleCalculatesCPUFromProcessAndKernelDeltas(t *testing.T) {
	m := New(&fakeInspector{values: []procfs.Process{{PID: 1, UserCPUTicks: 10}, {PID: 1, UserCPUTicks: 15, SystemCPUTicks: 5}}}, &fakeKernel{values: []uint64{100, 120}, uptime: time.Second})
	m.cpus = 2
	if first, err := m.Sample(1); err != nil || first.CPUPercent != 0 {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	second, err := m.Sample(1)
	if err != nil || second.CPUPercent != 100 {
		t.Fatalf("second=%+v err=%v", second, err)
	}
}

func TestRunStopsOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	m := New(&fakeInspector{values: []procfs.Process{{PID: 1}, {PID: 1}}}, &fakeKernel{values: []uint64{1, 2}, uptime: time.Second})
	calls := 0
	err := m.Run(ctx, 1, time.Millisecond, func(Sample, error) { calls++; cancel() })
	if err != context.Canceled || calls != 1 {
		t.Fatalf("err=%v calls=%d", err, calls)
	}
}
