// Package daemon composes Sentinel's control surface with procfs and services.
package daemon

import (
	"context"
	"errors"
	"fmt"
	"github.com/sentinel/sentinel/internal/control"
	"github.com/sentinel/sentinel/internal/procfs"
	"github.com/sentinel/sentinel/internal/supervisor"
	"sort"
	"sync"
)

type Service struct {
	Name   string
	Config supervisor.Config
}
type Daemon struct {
	proc     procfs.Reader
	mu       sync.RWMutex
	services map[string]Service
	running  map[string]*supervisor.Supervisor
}

func New(services []Service) (*Daemon, error) {
	d := &Daemon{proc: procfs.New(""), services: map[string]Service{}, running: map[string]*supervisor.Supervisor{}}
	for _, s := range services {
		if s.Name == "" {
			return nil, errors.New("daemon: service name required")
		}
		if _, ok := d.services[s.Name]; ok {
			return nil, fmt.Errorf("daemon: duplicate service %q", s.Name)
		}
		d.services[s.Name] = s
	}
	return d, nil
}
func (d *Daemon) Handle(ctx context.Context, r control.Request) (any, error) {
	switch r.Operation {
	case "status":
		return map[string]any{"daemon": "running", "services": len(d.services)}, nil
	case "ps":
		return d.proc.ListPIDs()
	case "inspect":
		return d.proc.Inspect(r.PID)
	case "services":
		return d.list(), nil
	case "start":
		return d.start(ctx, r.Service)
	case "stop":
		return d.stop(ctx, r.Service)
	case "restart":
		if _, err := d.stop(ctx, r.Service); err != nil {
			return nil, err
		}
		return d.start(ctx, r.Service)
	case "health":
		return map[string]string{"status": "not-configured"}, nil
	case "metrics":
		return map[string]string{"status": "not-configured"}, nil
	case "doctor":
		return map[string]string{"status": "ok", "ipc": "unix-json"}, nil
	default:
		return nil, fmt.Errorf("unsupported operation %q", r.Operation)
	}
}
func (d *Daemon) list() []map[string]any {
	d.mu.RLock()
	defer d.mu.RUnlock()
	names := make([]string, 0, len(d.services))
	for n := range d.services {
		names = append(names, n)
	}
	sort.Strings(names)
	out := make([]map[string]any, 0, len(names))
	for _, n := range names {
		state := "stopped"
		if s := d.running[n]; s != nil {
			state = string(s.Status().State)
		}
		out = append(out, map[string]any{"name": n, "state": state})
	}
	return out
}
func (d *Daemon) start(ctx context.Context, name string) (any, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	spec, ok := d.services[name]
	if !ok {
		return nil, fmt.Errorf("service %q is not configured", name)
	}
	if d.running[name] != nil {
		return nil, fmt.Errorf("service %q is already running", name)
	}
	s, err := supervisor.New(spec.Config)
	if err != nil {
		return nil, err
	}
	if err = s.Start(ctx); err != nil {
		return nil, err
	}
	d.running[name] = s
	return map[string]string{"service": name, "state": "starting"}, nil
}
func (d *Daemon) stop(ctx context.Context, name string) (any, error) {
	d.mu.Lock()
	s := d.running[name]
	if s == nil {
		d.mu.Unlock()
		return nil, fmt.Errorf("service %q is not running", name)
	}
	delete(d.running, name)
	d.mu.Unlock()
	if err := s.Stop(ctx); err != nil {
		return nil, err
	}
	return map[string]string{"service": name, "state": "stopped"}, nil
}
func (d *Daemon) Shutdown(ctx context.Context) error {
	d.mu.Lock()
	all := make([]*supervisor.Supervisor, 0, len(d.running))
	for _, s := range d.running {
		all = append(all, s)
	}
	d.running = map[string]*supervisor.Supervisor{}
	d.mu.Unlock()
	for _, s := range all {
		if err := s.Stop(ctx); err != nil {
			return err
		}
	}
	return nil
}
