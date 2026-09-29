// Package control implements Sentinel's local JSON-over-Unix-socket protocol.
package control

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	Version         = 1
	MaxRequestBytes = 64 << 10
)

type Request struct {
	Version   int    `json:"version"`
	Operation string `json:"operation"`
	PID       int    `json:"pid,omitempty"`
	Service   string `json:"service,omitempty"`
}
type Response struct {
	OK    bool            `json:"ok"`
	Error string          `json:"error,omitempty"`
	Data  json.RawMessage `json:"data,omitempty"`
}
type Handler interface {
	Handle(context.Context, Request) (any, error)
}

type Server struct {
	Path     string
	Handler  Handler
	listener net.Listener
	wg       sync.WaitGroup
	sem      chan struct{}
}

func (s *Server) Listen() error {
	if s.Path == "" {
		return errors.New("control: socket path is required")
	}
	dir := filepath.Dir(s.Path)
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
		if err := os.Chmod(dir, 0o700); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	if err := validateSocketDir(dir); err != nil {
		return err
	}
	if info, err := os.Lstat(s.Path); err == nil {
		if info.Mode()&os.ModeSocket == 0 {
			return fmt.Errorf("control: refusing to remove non-socket %s", s.Path)
		}
		if err := os.Remove(s.Path); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	l, err := net.Listen("unix", s.Path)
	if err != nil {
		return err
	}
	if err := os.Chmod(s.Path, 0o600); err != nil {
		l.Close()
		return err
	}
	s.listener = l
	s.sem = make(chan struct{}, 32)
	return nil
}
func (s *Server) Serve(ctx context.Context) error {
	if s.listener == nil {
		return errors.New("control: Listen must be called first")
	}
	go func() { <-ctx.Done(); _ = s.listener.Close() }()
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		select {
		case s.sem <- struct{}{}:
			s.wg.Add(1)
			go s.handle(conn)
		default:
			_ = conn.Close()
		}
	}
}
func (s *Server) Close() error {
	if s.listener != nil {
		_ = s.listener.Close()
	}
	s.wg.Wait()
	if s.Path != "" {
		if info, err := os.Lstat(s.Path); err == nil && info.Mode()&os.ModeSocket != 0 {
			return os.Remove(s.Path)
		}
	}
	return nil
}
func (s *Server) handle(conn net.Conn) {
	defer s.wg.Done()
	defer func() { <-s.sem; _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	if err := authorize(conn); err != nil {
		write(conn, Response{Error: err.Error()})
		return
	}
	limited := &io.LimitedReader{R: conn, N: MaxRequestBytes + 1}
	decoder := json.NewDecoder(limited)
	var req Request
	if err := decoder.Decode(&req); err != nil || limited.N == 0 {
		write(conn, Response{Error: "invalid or oversized request"})
		return
	}
	if req.Version != Version || req.Operation == "" {
		write(conn, Response{Error: "invalid request"})
		return
	}
	value, err := s.Handler.Handle(context.Background(), req)
	if err != nil {
		write(conn, Response{Error: err.Error()})
		return
	}
	encoded, _ := json.Marshal(value)
	write(conn, Response{OK: true, Data: encoded})
}
func write(c net.Conn, r Response) { _ = json.NewEncoder(c).Encode(r) }
func Call(ctx context.Context, path string, req Request, out any) error {
	c, err := (&net.Dialer{}).DialContext(ctx, "unix", path)
	if err != nil {
		return fmt.Errorf("cannot connect to Sentinel daemon at %s: %w", path, err)
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(5 * time.Second))
	if err := json.NewEncoder(c).Encode(req); err != nil {
		return err
	}
	var response Response
	if err := json.NewDecoder(io.LimitReader(c, MaxRequestBytes+1)).Decode(&response); err != nil {
		return err
	}
	if !response.OK {
		return errors.New(response.Error)
	}
	if out != nil {
		return json.Unmarshal(response.Data, out)
	}
	return nil
}
