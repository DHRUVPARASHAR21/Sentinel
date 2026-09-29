# Sentinel execution plan

## Project goal

Deliver `sentineld`: a production-quality, Linux-first Go daemon that supervises explicitly configured workloads, safely observes processes through `/proc`, exposes a local peer-authorized Unix-domain control API, emits Prometheus-compatible metrics and structured logs, integrates with systemd, and ships for Debian/Ubuntu.

## Non-goals

- Recreating systemd, including service dependency management, cgroups, journaling, or an init system.
- Container orchestration, distributed/fleet coordination, remote control APIs, plugins, or arbitrary command execution over IPC.
- A public Go SDK, cross-platform support beyond Linux, or collection of environments/full command lines as metrics.

## Architecture

`cmd/sentineld` wires dependencies, validates config, establishes trusted runtime paths, and coordinates shutdown. `config` owns schema, defaults, and validation. `supervisor` owns desired state, lifecycle transitions, and restart policy; it consumes typed facts from `procfs` and delegates spawning, identity validation, waiting, and signalling to `process`. `control` authenticates Unix-socket peers and forwards bounded requests to the supervisor. `metrics` and `logging` observe state and events without directing lifecycle. `systemd` isolates notifications/watchdog behavior, while `runtime` owns safe paths and privilege checks.

Proposed layout: `cmd/sentineld`, `cmd/sentinelctl`, `internal/{config,procfs,process,supervisor,control,metrics,logging,systemd,runtime}`, `test/integration`, `deploy/systemd`, and `packaging/debian`.

Use concrete composition. Add interfaces only at actual process/kernel/time seams; do not introduce speculative abstractions.

## Execution rule

**A milestone cannot be marked complete until all relevant verification commands pass.** If verification fails:

1. Diagnose the failure.
2. Repair the implementation, test, or documented environment issue.
3. Rerun the milestone’s complete verification set.
4. Proceed only after it passes.

Commands that require Ubuntu, root, systemd, or packaging tools are mandatory in the supported environment, not optional substitutes for unit checks.

## Milestones

### 1. Repository foundation and contracts

- **Goal:** Create a buildable module and side-effect-free configuration contract.
- **Affected:** `go.mod`, `cmd/sentineld/`, `internal/config/`, `internal/runtime/`, `testdata/`, `Makefile`, docs.
- **Scope:** Strict versioned config; `--config`/`--check-config`; safe default paths; Make targets `fmt`, `vet`, `test`, `race`, and `verify`.
- **Non-goals:** No daemon, child process, reload, IPC, metrics, or systemd behavior.
- **Acceptance:** Valid fixtures pass; unknown fields, duplicates, unsafe paths, and invalid executable argv fail before filesystem/process side effects; `make verify` needs neither root nor network.
- **Tests required:** Config table/error tests, runtime-path tests, CLI exit-code tests.
- **Validation:** `go fmt -d .`; `go vet ./...`; `go test ./... -count=1`; `go test -race ./...`; `go run ./cmd/sentineld --check-config --config testdata/valid.yaml`; `go run ./cmd/sentineld --check-config --config testdata/invalid.yaml; test $? -ne 0`; `make verify`.
- **Security:** Model commands as executable path plus argv, never shell strings; validate trusted runtime-directory ancestry.
- **Demo:** Validate a sample config with clear success/failure output and no daemon startup.

### 2. Procfs process inspection

- **Goal:** Produce defensive typed process observations from the required `/proc` files.
- **Affected:** `internal/procfs/`, `testdata/procfs/`, integration tests.
- **Scope:** Bounded readers/parsers for PID identity/start time, state, and PPID; typed absence, permission, and malformed-data errors; configurable procfs root for fixtures.
- **Non-goals:** No spawning, signalling, monitor loop, or command/environment collection.
- **Acceptance:** Parsers handle representative Linux stat edge cases; disappearance, access denial, short data, and malformed records return errors without panic.
- **Tests required:** Fixture tables, fuzz parser tests, bounded-read/error tests, live-process Linux smoke test.
- **Validation:** `go test ./internal/procfs -count=1`; `go test -race ./internal/procfs`; `go test ./internal/procfs -fuzz=FuzzParse -fuzztime=10s`; `go test ./test/integration -run TestProcfsCurrentProcess -count=1`.
- **Security:** Treat procfs as concurrent untrusted operational input; bounded reads prevent resource abuse and observations avoid sensitive data.
- **Demo:** Inspect a live PID and receive a typed error for a nonexistent PID.

### 3. Process monitoring and identity primitives

- **Goal:** Safely launch, wait for, identify, and signal managed processes.
- **Affected:** `internal/process/`, `internal/procfs/`, `test/integration/`, `docs/adr/`.
- **Scope:** PID-plus-start-time identity; explicit-argv launch; wait and exit classification; identity validation immediately before signalling; ADR for pidfd/fallback strategy.
- **Non-goals:** No restart policy, control server, or adoption of arbitrary existing processes.
- **Acceptance:** A fixture is launched, observed, terminated, and classified; stale identity refuses a signal; contexts do not leak wait/signal goroutines.
- **Tests required:** Process seam tests; Linux tests for TERM, KILL escalation, already-exited child, and stale identity.
- **Validation:** `go test ./internal/process -count=1`; `go test -race ./internal/process`; `go test ./test/integration -run 'Test(SignalLifecycle|StaleProcessIdentity|ChildExitClassification)' -count=1`.
- **Security:** PID reuse must never target an unintended process; never use `sh -c`; control inherited descriptors/environment deliberately.
- **Demo:** Sentinel handles a fixture’s requested termination and reports the exit class.

### 4. Supervisor lifecycle

- **Goal:** Implement deterministic single-workload desired/observed lifecycle control.
- **Affected:** `internal/supervisor/`, `internal/process/`, `internal/config/`, integration tests.
- **Scope:** Start/stop/reconcile, immutable status snapshots, bounded restart backoff/rate limits, idempotent shutdown.
- **Non-goals:** Reload, health checks, IPC, persistence, and metrics listeners.
- **Acceptance:** State transitions and restart policy are documented; failure cannot spin; concurrent status/reconcile/stop has no races.
- **Tests required:** Transition tables, fake-clock backoff, concurrent lifecycle, child-process integration.
- **Validation:** `go test ./internal/supervisor -count=1`; `go test -race ./internal/supervisor`; `go test ./test/integration -run 'Test(RestartPolicy|SupervisorStopIsIdempotent)' -count=1`.
- **Security:** Restart limits bound local resource exhaustion; only declared children are managed.
- **Demo:** A failing fixture restarts with backoff and exits cleanly when Sentinel stops.

### 5. Health checking and configuration reload

- **Goal:** Add bounded health evaluation and atomic last-known-good config reload.
- **Affected:** `internal/config/`, `internal/supervisor/`, `internal/process/`, integration tests, ADRs.
- **Scope:** Configured timeout/interval/threshold checks; reload trigger; parse and validate full snapshots, then atomically apply or retain old config; ADR reload semantics.
- **Non-goals:** Network config, arbitrary scripts, partial application, or remote checks.
- **Acceptance:** Health failures follow policy; checks honor cancellation and cannot overlap unboundedly; invalid reload preserves current behavior; valid reload is atomic to observers.
- **Tests required:** Timeout/threshold/cancellation tests, reload races, valid/invalid integration reloads.
- **Validation:** `go test ./internal/config ./internal/supervisor -count=1`; `go test -race ./internal/config ./internal/supervisor`; `go test ./test/integration -run 'Test(ConfigReload|HealthCheckTimeout|HealthFailureThreshold)' -count=1`.
- **Security:** Reload repeats strict ownership/schema checks; any command-style check uses explicit argv and least privilege.
- **Demo:** A valid config update changes policy; an invalid update is rejected while existing workload operation continues.

### 6. Daemon IPC

- **Goal:** Provide a bounded local Unix-domain API for status and explicit lifecycle requests.
- **Affected:** `internal/control/`, `internal/runtime/`, `internal/supervisor/`, `cmd/sentineld/`, integration tests, ADRs.
- **Scope:** Versioned request/response protocol; trusted socket creation/cleanup; peer-credential authorization; frame, connection, and deadline limits; protocol/authz ADR.
- **Non-goals:** TCP, unauthenticated users, arbitrary exec, shell passthrough, or log streaming.
- **Acceptance:** Authorized peers can only perform documented actions; unauthorized/malformed/oversized/slow clients are safely denied; unsafe socket paths fail closed.
- **Tests required:** Framing/limit and peer-auth unit tests; path attacks; distinct-identity integration tests when supported.
- **Validation:** `go test ./internal/control ./internal/runtime -count=1`; `go test -race ./internal/control ./internal/runtime`; `go test ./test/integration -run 'Test(AuthorizedSocketControl|UnauthorizedSocketPeer|OversizedControlRequest|UnsafeRuntimePath)' -count=1`.
- **Security:** Peer credentials, parent ownership/mode, bounded input, and safe stale-socket cleanup are hard authorization boundaries.
- **Demo:** An authorized local user retrieves status/restarts a workload; an unauthorized user is rejected.

### 7. CLI and operator workflow

- **Goal:** Offer a safe operator CLI layered strictly on the control protocol.
- **Affected:** `cmd/sentinelctl/`, `cmd/sentineld/`, `internal/control/`, README and development docs.
- **Scope:** `status`, `start`, `stop`, and `restart`; human and documented JSON output; useful exit codes.
- **Non-goals:** TUI, remote hosts, embedded shell, or duplicated lifecycle policy.
- **Acceptance:** Commands map one-to-one to IPC operations, retain daemon authorization, distinguish usage/connection/authz/operation errors, and have stable JSON fields.
- **Tests required:** Argument/exit tests, client protocol tests, end-to-end socket workflow.
- **Validation:** `go test ./cmd/sentinelctl ./internal/control -count=1`; `go test -race ./cmd/sentinelctl ./internal/control`; `go test ./test/integration -run TestSentinelctlWorkflow -count=1`; `go run ./cmd/sentinelctl --help`.
- **Security:** The CLI may not broaden daemon authority or bypass socket peer authorization.
- **Demo:** An operator issues `sentinelctl status` and `sentinelctl restart <workload>` to a foreground daemon.

### 8. Metrics and structured logging

- **Goal:** Expose safe lifecycle/health observability.
- **Affected:** `internal/metrics/`, `internal/logging/`, `internal/supervisor/`, daemon entry point, integration tests.
- **Scope:** Prometheus text output from synchronized snapshots; stable low-cardinality labels; `slog` event schema and redaction; local-by-default listener policy.
- **Non-goals:** Remote telemetry, log indexing, secret collection, or metrics that modify lifecycle state.
- **Acceptance:** Exposition parses; labels are documented and bounded; logs redact secrets/raw untrusted data; concurrent scrape/lifecycle is race-free.
- **Tests required:** Golden exposition and cardinality tests, redaction tests, concurrent scrape tests, HTTP integration.
- **Validation:** `go test ./internal/metrics ./internal/logging -count=1`; `go test -race ./internal/metrics ./internal/logging ./internal/supervisor`; `go test ./test/integration -run 'Test(MetricsExposition|StructuredLogRedaction)' -count=1`.
- **Security:** Metrics default to loopback/Unix socket and never label by full argv, environment, or unbounded PID values.
- **Demo:** A local scrape reports state/restarts/health while logs show redacted lifecycle events.

### 9. systemd integration and security hardening

- **Goal:** Operate reliably as a hardened Ubuntu service while retaining foreground behavior.
- **Affected:** `internal/systemd/`, `internal/runtime/`, `deploy/systemd/`, daemon entry point, security docs, ADRs, integration tests.
- **Scope:** Readiness/status/watchdog adapter; unit/tmpfiles definitions; safe `/run` and `/var/lib` handling; resource limits and graceful shutdown; privilege/sandboxing ADR.
- **Non-goals:** Managing unrelated units or implementing an init system.
- **Acceptance:** Foreground/systemd supervision semantics match; unit static verification passes; hardening rationale is documented; unsafe paths fail closed; orderly shutdown is bounded.
- **Tests required:** Adapter/path tests, unit verification, graceful-shutdown and Ubuntu systemd integration tests.
- **Validation:** `go test ./internal/systemd ./internal/runtime -count=1`; `go test -race ./internal/systemd ./internal/runtime ./internal/supervisor`; `systemd-analyze verify deploy/systemd/sentineld.service`; `go test ./test/integration -run 'Test(SystemdNotify|GracefulShutdown|UnsafeRuntimePath)' -count=1`.
- **Security:** Least privilege, capability rationale, systemd sandboxing, and trusted directory ancestry are mandatory.
- **Demo:** systemd reports Sentinel ready; stopping the service produces bounded orderly workload shutdown.

### 10. Packaging, CI, and release documentation

- **Goal:** Make Sentinel installable, continuously verified, and operable by someone new to the repository.
- **Affected:** `packaging/debian/`, `.github/workflows/`, `test/integration/`, docs, README, Makefile.
- **Scope:** Debian metadata/scripts/unit/config; pinned least-privilege CI for format/vet/unit/race/Linux integration/package; checksummed release artifacts; install/upgrade/rollback/troubleshooting docs.
- **Non-goals:** Apt repository publication, fleet rollout, or secret-dependent release automation.
- **Acceptance:** Clean Ubuntu builds, installs, starts, queries, stops, upgrades, and removes the package via documented steps; CI gates regressions; operational/developer/security docs are complete.
- **Tests required:** Package lifecycle smoke test, workflow static review, full race/integration suite.
- **Validation:** `make verify`; `go test ./test/integration -count=1`; `dpkg-buildpackage -us -uc -b`; `lintian ../sentinel_*.changes`; `sudo ./test/integration/package-smoke.sh ../sentinel_*.deb`; `act -W .github/workflows/ci.yml`.
- **Security:** Maintainer scripts are minimal/idempotent; installed files use correct ownership; workflows pin actions and do not expose secrets to pull requests.
- **Demo:** A fresh Ubuntu VM installs the built `.deb`, starts the systemd unit, and controls a sample workload locally.

## Implementation order

Foundation → procfs → process identity → supervisor → health/reload → IPC → CLI → observability → systemd hardening → packaging/CI. This resolves unsafe process and configuration contracts before public surfaces, minimizing later architectural rework. Any necessary deviation requires an ADR and an update to `docs/status.md`.
