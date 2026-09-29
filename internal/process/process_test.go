package process

import (
	"context"
	"os"
	"runtime"
	"testing"
	"time"
)

func TestStartRejectsEmptyPath(t *testing.T) {
	if _, err := Start(Spec{}); err == nil {
		t.Fatal("expected error")
	}
}
func TestChildWaitCanBeCalledMoreThanOnce(t *testing.T) {
	child, err := Start(Spec{Path: os.Args[0], Args: []string{"-test.run=TestProcessHelper", "--"}, Env: append(os.Environ(), "SENTINEL_PROCESS_HELPER=1")})
	if err != nil {
		t.Fatal(err)
	}
	if err := child.Wait(); err != nil {
		t.Fatal(err)
	}
	if err := child.Wait(); err != nil {
		t.Fatal(err)
	}
}
func TestProcessHelper(t *testing.T) {
	if os.Getenv("SENTINEL_PROCESS_HELPER") == "1" {
		os.Exit(0)
	}
}
func TestTerminateBounded(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("graceful Unix signal semantics are verified on Linux")
	}
	child, err := Start(Spec{Path: os.Args[0], Args: []string{"-test.run=TestProcessSleepHelper", "--"}, Env: append(os.Environ(), "SENTINEL_PROCESS_SLEEP_HELPER=1")})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := child.Terminate(ctx, time.Millisecond); err != nil {
		t.Fatal(err)
	}
}
func TestProcessSleepHelper(t *testing.T) {
	if os.Getenv("SENTINEL_PROCESS_SLEEP_HELPER") == "1" {
		time.Sleep(time.Second)
	}
}
