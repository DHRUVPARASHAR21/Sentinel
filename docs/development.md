# Development and delivery plan

## Initial toolchain

Use the current supported Go release selected when implementation begins, `go mod` for dependency management, and Ubuntu LTS for integration/package CI. Standard-library candidates include `log/slog`, `net`, `net/http`, `os/exec`, `syscall` (only where unavoidable), `runtime`, and `testing`. Add `golang.org/x/sys/unix` only if pidfd, peer credentials, or other Linux APIs cannot be safely reached through the standard library; record that choice in an ADR.

## Proposed automation

Create these workflow files when the Go module exists:

- `.github/workflows/ci.yml`: Ubuntu matrix; format check, vet, unit tests, race test, and Linux integration tests.
- `.github/workflows/package.yml`: build Debian package and run disposable Ubuntu install/start/stop smoke test.
- `.github/workflows/release.yml`: tag-gated provenance-aware artifact build, checksum, and release attachment.

CI should pin action versions by commit digest, use least-privilege permissions, and avoid secrets in pull-request workflows.

## Milestones

| # | Milestone | Acceptance criteria | Verification commands |
| --- | --- | --- | --- |
| 1 | Bootstrap and contracts | Go module, daemon CLI, config schema, directory policy, and documented supported Ubuntu versions exist; invalid config fails before side effects. | `go test ./...`; `go vet ./...`; `go run ./cmd/sentineld --check-config --config testdata/valid.yaml`; `! go run ./cmd/sentineld --check-config --config testdata/invalid.yaml` |
| 2 | Procfs reader | Typed, bounded parsers cover required stat/status/cmdline data; exit/permission/malformed cases are non-panicking. | `go test ./internal/procfs -count=1`; `go test -race ./internal/procfs` |
| 3 | Process identity and lifecycle primitives | Spawn/wait/signal APIs validate identity; no shell execution; TERM-to-KILL policy is testable. | `go test ./internal/process -count=1`; `go test -race ./internal/process`; `sudo go test ./test/integration -run TestSignalLifecycle -count=1` |
| 4 | Supervisor state machine | Desired/observed states, bounded restart backoff, and idempotent stop are defined and tested. | `go test ./internal/supervisor -count=1`; `go test -race ./internal/supervisor`; `go test ./test/integration -run TestRestartPolicy -count=1` |
| 5 | Configuration reload | Atomic validated snapshot replacement has clear reject/retain semantics and no partial apply. | `go test ./internal/config ./internal/supervisor -count=1`; `go test -race ./internal/...`; `go test ./test/integration -run TestConfigReload -count=1` |
| 6 | Local control API | Unix protocol is bounded, peer-authorized, and exposes only explicit lifecycle/status operations. | `go test ./internal/control -count=1`; `go test -race ./internal/control`; `sudo go test ./test/integration -run TestUnauthorizedSocketPeer -count=1` |
| 7 | Metrics and logging | Prometheus text output has stable names/low-cardinality labels; structured events are redactable and race-safe. | `go test ./internal/metrics ./internal/logging -count=1`; `go test -race ./internal/metrics ./internal/logging`; `go test ./test/integration -run TestMetricsExposition -count=1` |
| 8 | systemd integration | Notify/watchdog behavior and a hardened unit work on supported Ubuntu; foreground mode remains usable. | `go test ./internal/systemd -count=1`; `systemd-analyze verify deploy/systemd/sentineld.service`; `sudo go test ./test/integration -run TestSystemdNotify -count=1` |
| 9 | Operational hardening | Path/socket checks fail closed; limits, graceful shutdown, and security review cases pass. | `go test ./... -count=1`; `go test -race ./...`; `sudo go test ./test/integration -run 'Test(UnsafeRuntimePath|GracefulShutdown|SignalIdentity)' -count=1` |
| 10 | Debian package | Reproducible package metadata, maintainer scripts, unit/tmpfiles install, upgrade behavior, and clean uninstall are tested on Ubuntu. | `dpkg-buildpackage -us -uc -b`; `lintian ../sentinel_*.changes`; `sudo ./test/integration/package-smoke.sh ../sentinel_*.deb` |
| 11 | CI/CD and release readiness | CI gates every change; package workflow works; release workflow produces checksummed artifacts; docs cover operations and upgrades. | `go fmt -d .`; `go vet ./...`; `go test -race ./...`; `act -W .github/workflows/ci.yml` |
| 12 | Reliability release gate | Failure-mode tests, upgrade/downgrade exercise, security review, and an operator quickstart meet release checklist. | `make verify`; `sudo make integration`; `sudo make package-smoke` |

Commands prefixed with `sudo`, `dpkg-buildpackage`, `lintian`, `act`, `make`, or paths not yet present are planned verification commands; they become executable as their corresponding milestone introduces the required files and tooling.

## Makefile target contract

At milestone 1, add a small Makefile with `fmt`, `vet`, `test`, `race`, `integration`, `package`, `package-smoke`, and `verify`. `verify` should run formatting check, vet, unit tests, and race tests; it must not silently require root or network access.
