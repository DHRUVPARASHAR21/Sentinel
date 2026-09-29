# Sentinel release audit

## Initial findings

### Critical

1. **Packaged control socket rejected root** — `internal/control/auth_linux.go`. The systemd unit runs as `sentinel`, but peer authorization accepted only that UID. Root can traverse a `0600` Unix socket yet was denied by `SO_PEERCRED`, leaving the host administrator unable to operate the packaged daemon. Allow UID 0 in addition to the daemon UID; retain all other peer rejection.
2. **Metrics bind failures were discarded** — `cmd/sentineld/main.go`. The loopback metrics goroutine ignored `ListenAndServe` errors, so a port conflict silently removed observability from an apparently healthy daemon. Bind synchronously and fail startup with an actionable error.

### High

1. **Child descendants could escape shutdown** — `internal/process`. Signalling only the direct child leaves descendants running. Launch each managed child in a distinct process group and signal the group on Unix.
2. **Health errors stayed stale after recovery** — `internal/health/health.go`. A healthy result retained the previous error, confusing status consumers. Clear `LastError` on successful probes.
3. **Lifecycle logging is incomplete** — `internal/supervisor` and `internal/control`. Only daemon start/stop events are emitted; process/restart and client events require wiring before a production claim of complete lifecycle audit logging.

### Medium

1. **Metrics currently expose service-state counters but not live supervised process samples** — `internal/daemon` and `internal/metrics`. CPU/RSS/FD metrics render when samples are supplied, but daemon sampling is not wired. Document this limitation and add sampling with configuration work.
2. **Service flags are a bootstrap interface, not validated configuration** — `cmd/sentineld/main.go`. They accept only absolute executable paths but do not reject symlinked/untrusted paths or support argv. The README correctly calls this out; a configuration loader remains required.
3. **Tests do not cover cross-UID peer rejection or process-group descendant cleanup** — `internal/control` and `internal/process`. Add privileged Linux integration coverage when CI can create a second user.

### Low

1. **`PLAN.md` still references `log/slog`** although the Go 1.20 baseline uses an internal JSON logger. Align the planning documentation when revising the plan.
2. **The staticcheck Make target is optional**, while CI performs the available dependency scan. Pin a development-tool policy before calling it a mandatory local check.

### Polish

1. Add a changelog and release version injection before tagged releases.
2. Add GitHub issue templates (the PR template and contribution guide already exist).

## Architectural review

The internal package boundaries are appropriate and dependencies remain standard-library-only. Procfs parsing is bounded and typed; lifecycle state is mutex-protected; the control socket has size, deadline, and concurrency limits. The principal residual architectural gap is the missing strict configuration/reload layer, already documented as milestone 1/5 work.

## Security review

Socket path validation, stale-socket handling, private modes, explicit argv, PID start-time inspection, bounded request/procfs reads, and low-cardinality metric labels are sound foundations. The Critical/High fixes above are required before release readiness is assessed.

## Remediation completed

- Root and the daemon UID are accepted as Linux control peers; all other peer UIDs remain denied.
- The metrics listener is synchronously bound before startup; bind failures now stop the daemon with the endpoint in the error.
- Unix children are isolated in process groups and TERM/KILL targets that group, preventing ordinary descendants from escaping teardown.
- Health success clears a stale `LastError`.

## Tests and validation results

All completed successfully in the Ubuntu WSL environment after `make clean`:

```sh
make clean
make verify
go run golang.org/x/vuln/cmd/govulncheck@v1.0.4 ./...
make package
dpkg-deb --info packaging/debian/out/*.deb
systemd-analyze verify packaging/systemd/sentinel.service
bash scripts/demo.sh
```

`make verify` runs formatting validation, vet, all tests, race tests, integration tests, and builds. The package build produced `packaging/debian/out/sentinel_0.1.0_amd64.deb`; inspection confirmed root-owned package contents and maintainer `postinst`. The demo confirmed daemon startup, intentional `/bin/false` crash/restart, CLI service state, Prometheus scrape, and SIGTERM shutdown.

Local Windows validation also passed:

```sh
gofmt -w cmd internal
go test ./... -count=1
go test -race ./...
go vet ./...
staticcheck ./...
```

## Remaining known limitations

- There is no strict configuration-file loader, config reload, or package configuration activation; `--service NAME=/absolute/executable` is a deliberately narrow bootstrap interface.
- Health checks and live process samples are not yet attached to declared services, so the Prometheus renderer currently reports service state/restarts rather than live CPU/RSS per service.
- Lifecycle JSON logging currently covers daemon boundaries; process/restart/health/client events need full logger injection.
- Package install/remove/upgrade was inspected but not run in a disposable Ubuntu VM. The project does not yet have a selected license.

## Release readiness

The repository is runnable from a clean Ubuntu checkout, produces a Debian artifact, and has CI/package/demo coverage. It is suitable for a **technical preview**, not a stable public release, until the configuration, observability wiring, full lifecycle logging, disposable-VM package smoke test, and license decision are complete.
