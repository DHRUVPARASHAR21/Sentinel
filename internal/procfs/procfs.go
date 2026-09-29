// Package procfs reads a bounded, intentionally small subset of Linux procfs.
//
// Procfs is inherently racy: a process may exit between any two operations.
// Callers should treat ErrProcessGone as normal and retry or omit that process.
package procfs

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

const (
	defaultRoot = "/proc"
	maxFileSize = 1 << 20
)

var (
	// ErrProcessGone reports a normal procfs race: the process no longer exists.
	ErrProcessGone = errors.New("procfs: process disappeared")
	// ErrPermissionDenied reports a procfs permission failure.
	ErrPermissionDenied = errors.New("procfs: permission denied")
	// ErrMalformed reports an invalid or truncated procfs record.
	ErrMalformed = errors.New("procfs: malformed data")
	// ErrInaccessible reports a procfs file that cannot be read for another reason.
	ErrInaccessible = errors.New("procfs: inaccessible file")
	// ErrUnsupported reports a non-Linux host or unavailable procfs root.
	ErrUnsupported = errors.New("procfs: unsupported environment")
)

// Process is a point-in-time observation. UID and GID are the effective IDs
// reported by /proc/<pid>/status. A zero value for an optional memory value
// means the kernel did not expose that value.
type Process struct {
	PID                int
	PPID               int
	UID                uint32
	GID                uint32
	State              rune
	CommandName        string
	CommandLine        []string
	ExecutablePath     string
	RSSBytes           uint64
	VirtualMemoryBytes uint64
	UserCPUTicks       uint64
	SystemCPUTicks     uint64
	ThreadCount        int
	FDCount            int
	StartTimeTicks     uint64
}

// Error adds process and operation context while allowing errors.Is checks
// against the package sentinel errors.
type Error struct {
	Op   string
	PID  int
	Path string
	Kind error
	Err  error
}

func (e *Error) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("procfs %s for pid %d: %v", e.Op, e.PID, e.Kind)
	}
	return fmt.Sprintf("procfs %s for pid %d (%s): %v: %v", e.Op, e.PID, e.Path, e.Kind, e.Err)
}

func (e *Error) Unwrap() error { return e.Err }

func (e *Error) Is(target error) bool { return target == e.Kind }

// Reader reads process records from Root. Root is configurable for tests and
// offline analysis; an empty root uses /proc.
type Reader struct{ Root string }

// New constructs a reader. It performs no I/O.
func New(root string) Reader {
	if root == "" {
		root = defaultRoot
	}
	return Reader{Root: root}
}

// ListPIDs lists numeric process directories in ascending PID order. A process
// disappearing after this call is expected and will be reported by Inspect.
func (r Reader) ListPIDs() ([]int, error) {
	if err := r.supported(); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(r.Root)
	if err != nil {
		return nil, r.wrap("list", 0, r.Root, err)
	}
	pids := make([]int, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err == nil && pid > 0 {
			pids = append(pids, pid)
		}
	}
	sort.Ints(pids)
	return pids, nil
}

// Inspect returns an observation assembled from /proc/<pid>. It does not
// promise all fields came from the same kernel instant. ErrProcessGone is a
// normal outcome if the process exits during inspection.
func (r Reader) Inspect(pid int) (Process, error) {
	if pid <= 0 {
		return Process{}, r.error("inspect", pid, "", ErrMalformed, fmt.Errorf("invalid pid"))
	}
	if err := r.supported(); err != nil {
		return Process{}, err
	}
	dir := filepath.Join(r.Root, strconv.Itoa(pid))
	statData, err := r.readFile(pid, filepath.Join(dir, "stat"))
	if err != nil {
		return Process{}, err
	}
	stat, err := ParseStat(statData)
	if err != nil {
		return Process{}, r.error("parse stat", pid, filepath.Join(dir, "stat"), ErrMalformed, err)
	}
	if stat.PID != pid {
		return Process{}, r.error("parse stat", pid, filepath.Join(dir, "stat"), ErrMalformed, fmt.Errorf("stat pid %d", stat.PID))
	}
	statusData, err := r.readFile(pid, filepath.Join(dir, "status"))
	if err != nil {
		return Process{}, err
	}
	status, err := ParseStatus(statusData)
	if err != nil {
		return Process{}, r.error("parse status", pid, filepath.Join(dir, "status"), ErrMalformed, err)
	}
	process := Process{
		PID:                stat.PID,
		PPID:               stat.PPID,
		UID:                status.EffectiveUID,
		GID:                status.EffectiveGID,
		State:              stat.State,
		CommandName:        stat.CommandName,
		RSSBytes:           status.RSSBytes,
		VirtualMemoryBytes: status.VirtualMemoryBytes,
		UserCPUTicks:       stat.UserCPUTicks,
		SystemCPUTicks:     stat.SystemCPUTicks,
		ThreadCount:        stat.ThreadCount,
		StartTimeTicks:     stat.StartTimeTicks,
	}
	cmdlineData, err := r.readFile(pid, filepath.Join(dir, "cmdline"))
	if err != nil {
		return process, err
	}
	process.CommandLine = ParseCmdline(cmdlineData)
	exe, err := os.Readlink(filepath.Join(dir, "exe"))
	if err != nil {
		return process, r.wrap("read executable", pid, filepath.Join(dir, "exe"), err)
	}
	process.ExecutablePath = exe
	fds, err := os.ReadDir(filepath.Join(dir, "fd"))
	if err != nil {
		return process, r.wrap("count file descriptors", pid, filepath.Join(dir, "fd"), err)
	}
	process.FDCount = len(fds)
	return process, nil
}

func (r Reader) supported() error {
	if runtime.GOOS != "linux" && r.Root == defaultRoot {
		return r.error("open procfs", 0, r.Root, ErrUnsupported, fmt.Errorf("requires linux"))
	}
	if r.Root == defaultRoot {
		if _, err := os.Stat(r.Root); err != nil {
			return r.error("open procfs", 0, r.Root, ErrUnsupported, err)
		}
	}
	return nil
}

func (r Reader) readFile(pid int, path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, r.wrap("open", pid, path, err)
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxFileSize+1))
	if err != nil {
		return nil, r.wrap("read", pid, path, err)
	}
	if len(data) > maxFileSize {
		return nil, r.error("read", pid, path, ErrMalformed, fmt.Errorf("record exceeds %d bytes", maxFileSize))
	}
	return data, nil
}

func (r Reader) wrap(op string, pid int, path string, err error) error {
	kind := ErrInaccessible
	switch {
	case errors.Is(err, fs.ErrNotExist):
		kind = ErrProcessGone
	case errors.Is(err, fs.ErrPermission):
		kind = ErrPermissionDenied
	}
	return r.error(op, pid, path, kind, err)
}

func (r Reader) error(op string, pid int, path string, kind, err error) error {
	return &Error{Op: op, PID: pid, Path: path, Kind: kind, Err: err}
}

// Stat is the parsed subset of /proc/<pid>/stat used by Sentinel.
type Stat struct {
	PID            int
	CommandName    string
	State          rune
	PPID           int
	UserCPUTicks   uint64
	SystemCPUTicks uint64
	ThreadCount    int
	StartTimeTicks uint64
}

// ParseStat parses Linux procfs stat data. It finds the final ')' because a
// process comm field can itself contain spaces and parentheses.
func ParseStat(data []byte) (Stat, error) {
	s := strings.TrimSpace(string(data))
	open := strings.IndexByte(s, '(')
	close := strings.LastIndex(s, ")")
	if open <= 0 || close <= open || close+1 >= len(s) {
		return Stat{}, fmt.Errorf("invalid stat framing")
	}
	pid, err := parseInt(s[:open])
	if err != nil || pid <= 0 {
		return Stat{}, fmt.Errorf("invalid pid")
	}
	fields := strings.Fields(s[close+1:])
	// field 3 is state. Needed fields reach field 22, so indices reach 19.
	if len(fields) < 20 || len(fields[0]) != 1 {
		return Stat{}, fmt.Errorf("truncated stat")
	}
	ppid, err := parseInt(fields[1])
	if err != nil || ppid < 0 {
		return Stat{}, fmt.Errorf("invalid ppid")
	}
	utime, err := parseUint(fields[11])
	if err != nil {
		return Stat{}, fmt.Errorf("invalid user cpu time")
	}
	stime, err := parseUint(fields[12])
	if err != nil {
		return Stat{}, fmt.Errorf("invalid system cpu time")
	}
	threads, err := parseInt(fields[17])
	if err != nil || threads < 0 {
		return Stat{}, fmt.Errorf("invalid thread count")
	}
	start, err := parseUint(fields[19])
	if err != nil {
		return Stat{}, fmt.Errorf("invalid start time")
	}
	return Stat{PID: pid, CommandName: s[open+1 : close], State: rune(fields[0][0]), PPID: ppid, UserCPUTicks: utime, SystemCPUTicks: stime, ThreadCount: threads, StartTimeTicks: start}, nil
}

// Status is the parsed subset of /proc/<pid>/status used by Sentinel.
type Status struct {
	EffectiveUID       uint32
	EffectiveGID       uint32
	RSSBytes           uint64
	VirtualMemoryBytes uint64
}

// ParseStatus parses effective credentials and memory values. VmRSS is absent
// for some processes; its zero value is therefore meaningful and not an error.
func ParseStatus(data []byte) (Status, error) {
	var result Status
	var uid, gid bool
	for _, line := range strings.Split(string(data), "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		fields := strings.Fields(value)
		switch key {
		case "Uid":
			if len(fields) < 2 {
				return Status{}, fmt.Errorf("invalid Uid")
			}
			v, err := parseUint32(fields[1])
			if err != nil {
				return Status{}, fmt.Errorf("invalid Uid")
			}
			result.EffectiveUID, uid = v, true
		case "Gid":
			if len(fields) < 2 {
				return Status{}, fmt.Errorf("invalid Gid")
			}
			v, err := parseUint32(fields[1])
			if err != nil {
				return Status{}, fmt.Errorf("invalid Gid")
			}
			result.EffectiveGID, gid = v, true
		case "VmRSS":
			v, err := parseKiB(fields)
			if err != nil {
				return Status{}, fmt.Errorf("invalid VmRSS")
			}
			result.RSSBytes = v
		case "VmSize":
			v, err := parseKiB(fields)
			if err != nil {
				return Status{}, fmt.Errorf("invalid VmSize")
			}
			result.VirtualMemoryBytes = v
		}
	}
	if !uid || !gid {
		return Status{}, fmt.Errorf("missing credentials")
	}
	return result, nil
}

// ParseCmdline splits the NUL-delimited argv representation from procfs. Empty
// input is valid for kernel threads; a trailing NUL is not returned as an arg.
func ParseCmdline(data []byte) []string {
	if len(data) == 0 {
		return nil
	}
	parts := strings.Split(string(data), "\x00")
	if parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
}

func parseInt(s string) (int, error) {
	v, err := strconv.ParseInt(strings.TrimSpace(s), 10, 0)
	return int(v), err
}

func parseUint(s string) (uint64, error) {
	return strconv.ParseUint(strings.TrimSpace(s), 10, 64)
}

func parseUint32(s string) (uint32, error) {
	v, err := strconv.ParseUint(strings.TrimSpace(s), 10, 32)
	return uint32(v), err
}

func parseKiB(fields []string) (uint64, error) {
	if len(fields) != 2 || fields[1] != "kB" {
		return 0, fmt.Errorf("expected kB")
	}
	v, err := parseUint(fields[0])
	if err != nil || v > ^uint64(0)/1024 {
		return 0, fmt.Errorf("invalid size")
	}
	return v * 1024, nil
}
