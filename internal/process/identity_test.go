package process

import (
	"github.com/sentinel/sentinel/internal/procfs"
	"testing"
)

type identityInspector struct{ p procfs.Process }

func (i identityInspector) Inspect(int) (procfs.Process, error) { return i.p, nil }
func TestIdentityRejectsPIDReuse(t *testing.T) {
	id, err := CaptureIdentity(identityInspector{procfs.Process{PID: 42, StartTimeTicks: 10}}, 42)
	if err != nil {
		t.Fatal(err)
	}
	if err := id.Verify(identityInspector{procfs.Process{PID: 42, StartTimeTicks: 11}}); err == nil {
		t.Fatal("expected identity mismatch")
	}
}
