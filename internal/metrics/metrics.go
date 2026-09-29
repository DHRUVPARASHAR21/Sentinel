// Package metrics renders a deliberately small Prometheus exposition surface.
package metrics

import (
	"fmt"
	"github.com/sentinel/sentinel/internal/monitor"
	"sort"
	"strings"
)

type Service struct {
	Name          string
	Restarts      int
	Healthy       bool
	UptimeSeconds float64
	Sample        *monitor.Sample
}

func Render(services []Service) string {
	sort.Slice(services, func(i, j int) bool { return services[i].Name < services[j].Name })
	var b strings.Builder
	fmt.Fprintf(&b, "# HELP sentinel_supervised_services Number of configured services\n# TYPE sentinel_supervised_services gauge\nsentinel_supervised_services %d\n", len(services))
	for _, s := range services {
		n := escape(s.Name)
		fmt.Fprintf(&b, "sentinel_service_restarts_total{service=\"%s\"} %d\nsentinel_service_health{service=\"%s\"} %d\nsentinel_service_uptime_seconds{service=\"%s\"} %.3f\n", n, s.Restarts, n, boolNumber(s.Healthy), n, s.UptimeSeconds)
		if s.Sample != nil {
			p := s.Sample.Process
			fmt.Fprintf(&b, "sentinel_process_cpu_percent{service=\"%s\"} %.6f\nsentinel_process_resident_memory_bytes{service=\"%s\"} %d\nsentinel_process_virtual_memory_bytes{service=\"%s\"} %d\nsentinel_process_threads{service=\"%s\"} %d\nsentinel_process_open_fds{service=\"%s\"} %d\n", n, s.Sample.CPUPercent, n, p.RSSBytes, n, p.VirtualMemoryBytes, n, p.ThreadCount, n, p.FDCount)
		}
	}
	return b.String()
}
func escape(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\\", "\\\\"), "\"", "\\\"")
}
func boolNumber(v bool) int {
	if v {
		return 1
	}
	return 0
}
