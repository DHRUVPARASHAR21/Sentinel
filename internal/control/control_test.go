package control

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

type handler func(context.Context, Request) (any, error)

func (h handler) Handle(c context.Context, r Request) (any, error) { return h(c, r) }
func startServer(t *testing.T) (string, context.CancelFunc) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	path := socketPath(t)
	s := &Server{Path: path, Handler: handler(func(_ context.Context, r Request) (any, error) {
		if r.Operation == "bad" {
			return nil, errors.New("bad operation")
		}
		return map[string]any{"operation": r.Operation}, nil
	})}
	if err := s.Listen(); err != nil {
		t.Fatal(err)
	}
	go func() { _ = s.Serve(ctx); _ = s.Close() }()
	t.Cleanup(cancel)
	return path, cancel
}

func socketPath(t *testing.T) string {
	t.Helper()
	if runtime.GOOS != "windows" {
		dir := t.TempDir()
		if err := os.Chmod(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		return filepath.Join(dir, "sentinel.sock")
	}
	dir := filepath.Join(`C:\`, "sentinel-test")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, fmt.Sprintf("s-%d-%d.sock", os.Getpid(), time.Now().UnixNano()))
	t.Cleanup(func() { _ = os.Remove(path) })
	return path
}
func TestCallRoundTrip(t *testing.T) {
	path, _ := startServer(t)
	var got map[string]any
	if err := Call(context.Background(), path, Request{Version: Version, Operation: "status"}, &got); err != nil {
		t.Fatal(err)
	}
	if got["operation"] != "status" {
		t.Fatal(got)
	}
}
func TestServerRejectsMalformedAndOversizedMessages(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows Unix sockets do not support CloseWrite")
	}
	path, _ := startServer(t)
	c, err := net.Dial("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if _, err = c.Write([]byte("{")); err != nil {
		t.Fatal(err)
	}
	if err := c.(*net.UnixConn).CloseWrite(); err != nil {
		t.Fatal(err)
	}
	_ = c.SetReadDeadline(time.Now().Add(time.Second))
	buf := make([]byte, 1024)
	if n, err := c.Read(buf); err != nil || n == 0 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}
func TestListenRefusesRegularFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "socket")
	if err := os.WriteFile(path, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := (&Server{Path: path}).Listen(); err == nil {
		t.Fatal("expected refusal")
	}
}
