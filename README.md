# Sentinel

Sentinel is a planned Linux-first process supervisor and observability daemon written in Go. It will supervise explicitly configured workloads, gather selected process data from `/proc`, provide a permission-controlled Unix-domain control API, and export Prometheus-compatible metrics.

This repository currently contains the architecture and delivery plan only; the daemon has deliberately not been implemented yet.

## Target capabilities

- Lifecycle-aware supervision with careful signal delivery and exit classification.
- Defensive Linux `/proc` collection and process identity handling.
- systemd service integration, structured logging, local control over Unix sockets, and Prometheus metrics.
- Ubuntu-focused tests, Debian packaging, and CI checks including the Go race detector.

## Proposed layout

```text
cmd/sentineld/                 daemon entry point and composition root
internal/config/               config decoding, validation, defaults
internal/supervisor/           desired state, lifecycle, restart policy
internal/procfs/               defensive readers for selected /proc files
internal/process/              process identity, spawning, signalling, pidfds
internal/control/              Unix-domain request/response server and authz
internal/metrics/              Prometheus text exposition and metric registry
internal/systemd/              systemd notification and service helpers
internal/logging/              structured logging setup and redaction policy
internal/runtime/              paths, privilege checks, shutdown wiring
packaging/debian/              Debian source package metadata
deploy/systemd/                unit and tmpfiles definitions
test/integration/              Linux-only black-box tests and fixtures
docs/                          design, security, development, testing, ADRs
.github/workflows/             proposed CI and release workflows
```

`internal/` is intentional: Sentinel does not initially promise a reusable Go API.

## Planning documents

- [Architecture](docs/architecture.md)
- [Security model](docs/security.md)
- [Development and delivery plan](docs/development.md)
- [Testing strategy](docs/testing.md)
- [ADR index](docs/adr/README.md)
