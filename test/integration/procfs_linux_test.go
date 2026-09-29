//go:build linux

package integration

import (
	"os"
	"testing"

	"github.com/sentinel/sentinel/internal/procfs"
)

func TestProcfsCurrentProcess(t *testing.T) {
	process, err := procfs.New("").Inspect(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	if process.PID != os.Getpid() || process.PPID <= 0 || process.CommandName == "" || process.ThreadCount < 1 {
		t.Fatalf("unexpected observation: %+v", process)
	}
}
