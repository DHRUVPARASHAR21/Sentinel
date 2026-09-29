package procfs

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseStatFixture(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "stat"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := ParseStat(data)
	if err != nil {
		t.Fatal(err)
	}
	if got.PID != 4242 || got.PPID != 7 || got.CommandName != "worker (blue)" || got.State != 'S' || got.UserCPUTicks != 14 || got.SystemCPUTicks != 15 || got.ThreadCount != 20 || got.StartTimeTicks != 22 {
		t.Fatalf("unexpected stat: %+v", got)
	}
}

func TestParseStatRejectsMalformedAndTruncatedInput(t *testing.T) {
	for _, data := range [][]byte{nil, []byte("42 (bad) S 1"), []byte("42 bad) S 1"), []byte("42 (x) SS 1 2 3")} {
		if _, err := ParseStat(data); err == nil {
			t.Fatalf("ParseStat(%q) succeeded", data)
		}
	}
}

func TestParseStatusFixture(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "status"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := ParseStatus(data)
	if err != nil {
		t.Fatal(err)
	}
	if got.EffectiveUID != 1001 || got.EffectiveGID != 1002 || got.RSSBytes != 12*1024 || got.VirtualMemoryBytes != 34*1024 {
		t.Fatalf("unexpected status: %+v", got)
	}
}

func TestParseStatusRejectsMalformedInput(t *testing.T) {
	for _, data := range [][]byte{[]byte("Uid:\t1\nGid:\t1\t1\n"), []byte("Uid:\t1\t1\nGid:\t1\t1\nVmRSS:\tnope kB\n")} {
		if _, err := ParseStatus(data); err == nil {
			t.Fatalf("ParseStatus(%q) succeeded", data)
		}
	}
}

func TestParseCmdlineUnusualContents(t *testing.T) {
	got := ParseCmdline([]byte("worker\x00--label=a b\x00\x00emoji-✓\x00"))
	want := []string{"worker", "--label=a b", "", "emoji-✓"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseCmdline() = %#v, want %#v", got, want)
	}
}

func TestListPIDsUsesNumericDirectoriesOnly(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"10", "2", "self", "-1", "1x"} {
		if err := os.Mkdir(filepath.Join(root, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	got, err := New(root).ListPIDs()
	if err != nil {
		t.Fatal(err)
	}
	want := []int{2, 10}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ListPIDs() = %v, want %v", got, want)
	}
}

func TestInspectDisappearedProcess(t *testing.T) {
	_, err := New(t.TempDir()).Inspect(4242)
	if !errors.Is(err, ErrProcessGone) {
		t.Fatalf("Inspect() error = %v, want ErrProcessGone", err)
	}
}

func TestInspectPermissionStyleError(t *testing.T) {
	root := t.TempDir()
	pidDir := filepath.Join(root, "4242")
	if err := os.Mkdir(pidDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pidDir, "stat"), []byte(""), 0o000); err != nil {
		t.Fatal(err)
	}
	_, err := New(root).Inspect(4242)
	if err == nil {
		t.Fatal("Inspect() unexpectedly succeeded")
	}
	// Windows does not enforce POSIX file modes; malformed or inaccessible is
	// acceptable there. Linux non-root runs should classify it as permission.
	if !errors.Is(err, ErrPermissionDenied) && !errors.Is(err, ErrMalformed) && !errors.Is(err, ErrInaccessible) {
		t.Fatalf("Inspect() error = %v", err)
	}
}

func TestInspectReturnsPartialObservationForOptionalFileFailure(t *testing.T) {
	root := t.TempDir()
	pidDir := filepath.Join(root, "4242")
	if err := os.Mkdir(pidDir, 0o755); err != nil {
		t.Fatal(err)
	}
	stat, err := os.ReadFile(filepath.Join("testdata", "stat"))
	if err != nil {
		t.Fatal(err)
	}
	status, err := os.ReadFile(filepath.Join("testdata", "status"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pidDir, "stat"), stat, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pidDir, "status"), status, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pidDir, "cmdline"), []byte("worker\x00--safe\x00"), 0o644); err != nil {
		t.Fatal(err)
	}
	process, err := New(root).Inspect(4242)
	if err == nil || !errors.Is(err, ErrProcessGone) {
		t.Fatalf("Inspect() error = %v, want missing optional exe", err)
	}
	if process.PID != 4242 || !reflect.DeepEqual(process.CommandLine, []string{"worker", "--safe"}) {
		t.Fatalf("partial Process = %+v", process)
	}
}

func FuzzParseStat(f *testing.F) {
	f.Add([]byte("4242 (worker (blue)) S 7 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22"))
	f.Fuzz(func(t *testing.T, data []byte) { _, _ = ParseStat(data) })
}

func FuzzParseStatus(f *testing.F) {
	f.Add([]byte("Uid:\t1001\t1001\t1001\t1001\nGid:\t1002\t1002\t1002\t1002\n"))
	f.Fuzz(func(t *testing.T, data []byte) { _, _ = ParseStatus(data) })
}
