package integration

import (
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestDaemonCLIWorkflow(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	binDir := t.TempDir()
	daemonBin := filepath.Join(binDir, "sentineld")
	cliBin := filepath.Join(binDir, "sentinel")
	if runtime.GOOS == "windows" {
		daemonBin += ".exe"
		cliBin += ".exe"
	}
	build(t, root, daemonBin, "./cmd/sentineld")
	build(t, root, cliBin, "./cmd/sentinel")
	socket := integrationSocket(t)
	daemon := exec.Command(daemonBin, "--socket", socket)
	if err := daemon.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = daemon.Process.Kill(); _ = daemon.Wait(); _ = os.Remove(socket) })
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if c, err := net.Dial("unix", socket); err == nil {
			_ = c.Close()
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	commands := [][]string{{"status"}, {"services"}, {"doctor"}}
	if runtime.GOOS == "linux" {
		commands = append(commands, []string{"ps"}, []string{"inspect", strconv.Itoa(os.Getpid())})
	}
	for _, args := range commands {
		cmd := exec.Command(cliBin, append([]string{"--socket", socket}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("sentinel %s: %v: %s", strings.Join(args, " "), err, out)
		}
	}
	cmd := exec.Command(cliBin, "--socket", socket+"-missing", "status")
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "cannot connect to Sentinel daemon") {
		t.Fatalf("unavailable daemon error: %v %s", err, out)
	}
}
func build(t *testing.T, root, output, pkg string) {
	t.Helper()
	c := exec.Command("go", "build", "-o", output, pkg)
	c.Dir = root
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("build %s: %v: %s", pkg, err, out)
	}
}
func integrationSocket(t *testing.T) string {
	if runtime.GOOS != "windows" {
		dir := t.TempDir()
		if err := os.Chmod(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		return filepath.Join(dir, "sentinel.sock")
	}
	dir := filepath.Join(`C:\`, "sentinel-test")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(dir, "integration-"+strconv.Itoa(os.Getpid())+".sock")
}
