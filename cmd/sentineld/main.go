package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/sentinel/sentinel/internal/control"
	"github.com/sentinel/sentinel/internal/daemon"
	"github.com/sentinel/sentinel/internal/logging"
	"github.com/sentinel/sentinel/internal/process"
	"github.com/sentinel/sentinel/internal/supervisor"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"
)

func main() {
	var socket string
	var metricsListen string
	var definitions values
	flag.StringVar(&socket, "socket", "/run/sentinel/sentinel.sock", "Unix control socket")
	flag.StringVar(&metricsListen, "metrics-listen", "127.0.0.1:9464", "Loopback Prometheus listener")
	flag.Var(&definitions, "service", "NAME=/absolute/executable (repeatable)")
	flag.Parse()
	services := make([]daemon.Service, 0, len(definitions))
	for _, d := range definitions {
		parts := strings.SplitN(d, "=", 2)
		if len(parts) != 2 || parts[0] == "" || !strings.HasPrefix(parts[1], "/") {
			fmt.Fprintln(os.Stderr, "invalid --service; expected NAME=/absolute/executable")
			os.Exit(2)
		}
		services = append(services, daemon.Service{Name: parts[0], Config: supervisor.Config{Process: process.Spec{Path: parts[1]}, Restart: supervisor.RestartOnFailure, MaxRetries: 3, InitialBackoff: time.Second, MaxBackoff: time.Minute, TerminationTimeout: 10 * time.Second}})
	}
	d, err := daemon.New(services)
	if err != nil {
		fatal(err)
	}
	server := &control.Server{Path: socket, Handler: d}
	if err := server.Listen(); err != nil {
		fatal(err)
	}
	logger := logging.New()
	logger.Event("daemon_started", map[string]any{"socket": socket})
	metricsServer := &http.Server{Addr: metricsListen, Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = w.Write([]byte(d.Metrics()))
	})}
	go func() { _ = metricsServer.ListenAndServe() }()
	ctx, cancel := signal.NotifyContext(context.Background(), daemonSignals()...)
	defer cancel()
	defer func() {
		logger.Event("daemon_stopping", nil)
		_ = metricsServer.Shutdown(context.Background())
		_ = server.Close()
	}()
	defer d.Shutdown(context.Background())
	if err := server.Serve(ctx); err != nil {
		fatal(err)
	}
}

type values []string

func (v *values) String() string     { return strings.Join(*v, ",") }
func (v *values) Set(s string) error { *v = append(*v, s); return nil }
func fatal(e error)                  { fmt.Fprintln(os.Stderr, "sentineld:", e); os.Exit(1) }
