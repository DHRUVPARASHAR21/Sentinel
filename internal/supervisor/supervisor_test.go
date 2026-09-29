package supervisor

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sentinel/sentinel/internal/process"
)

func TestHelperProcess(t *testing.T) {
	if os.Getenv("SENTINEL_HELPER") != "1" {
		return
	}
	mode := os.Args[len(os.Args)-1]
	switch mode {
	case "success":
		os.Exit(0)
	case "crash":
		os.Exit(3)
	case "sleep":
		time.Sleep(10 * time.Second)
		os.Exit(0)
	case "ignoreterm":
		signal.Ignore(terminationSignal())
		time.Sleep(10 * time.Second)
		os.Exit(0)
	}
}

func helper(mode string) process.Spec {
	return process.Spec{Path: os.Args[0], Args: []string{"-test.run=TestHelperProcess", "--", mode}, Env: append(os.Environ(), "SENTINEL_HELPER=1")}
}
func testConfig(mode string) Config {
	return Config{Process: helper(mode), Restart: RestartOnFailure, MaxRetries: 2, InitialBackoff: time.Millisecond, MaxBackoff: 5 * time.Millisecond, TerminationTimeout: 50 * time.Millisecond}
}

func waitState(t *testing.T, s *Supervisor, want State) Status {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		status := s.Status()
		if status.State == want {
			return status
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("state=%+v want %s", s.Status(), want)
	return Status{}
}

func TestSuccessfulProcessLaunchAndNormalExit(t *testing.T) {
	cfg := testConfig("success")
	cfg.Restart = RestartNever
	s, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	s.Wait()
	if status := s.Status(); status.State != StateStopped || status.LastExit != nil {
		t.Fatalf("status=%+v", status)
	}
}
func TestCrashRestartsUntilLimit(t *testing.T) {
	s, err := New(testConfig("crash"))
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	s.Wait()
	status := s.Status()
	if status.Retries != 2 || status.Restarts != 2 || status.LastExit == nil {
		t.Fatalf("status=%+v", status)
	}
}
func TestLaunchFailureRespectsRetryLimit(t *testing.T) {
	cfg := testConfig("success")
	cfg.Process.Path = "definitely-not-a-sentinel-executable"
	s, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	s.Wait()
	status := s.Status()
	if status.Retries != 2 || status.Restarts != 2 || status.LastExit == nil {
		t.Fatalf("status=%+v", status)
	}
}
func TestShutdownDuringRestartBackoff(t *testing.T) {
	cfg := testConfig("crash")
	cfg.InitialBackoff = time.Second
	cfg.MaxBackoff = time.Second
	s, _ := New(cfg)
	s.Start(context.Background())
	waitState(t, s, StateBackoff)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := s.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	if s.Status().State != StateStopped {
		t.Fatal(s.Status())
	}
}
func TestHungProcessForcedTermination(t *testing.T) {
	cfg := testConfig("ignoreterm")
	cfg.Restart = RestartNever
	s, _ := New(cfg)
	s.Start(context.Background())
	waitState(t, s, StateRunning)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := s.Stop(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestExponentialBackoffIsCapped(t *testing.T) {
	s, err := New(Config{InitialBackoff: 10 * time.Millisecond, MaxBackoff: 25 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	if s.backoff(1) != 10*time.Millisecond || s.backoff(2) != 20*time.Millisecond || s.backoff(3) != 25*time.Millisecond {
		t.Fatal("unexpected backoff")
	}
}
func TestConcurrentOperations(t *testing.T) {
	cfg := testConfig("sleep")
	cfg.Restart = RestartNever
	s, _ := New(cfg)
	s.Start(context.Background())
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _ = s.Status(); _ = s.HandleSignal(context.Background(), os.Interrupt) }()
	}
	wg.Wait()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := s.Stop(ctx); err != nil && !strings.Contains(err.Error(), "context") {
		t.Fatal(err)
	}
}
