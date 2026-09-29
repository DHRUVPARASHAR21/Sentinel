# Sentinel status

## Current milestone

Milestone 1 — Repository foundation and contracts — **not started**. Milestones 2–4 and 6–7 were completed out of sequence at the user's request and are fully verified.

## Completed milestones

- Milestone 2 — Procfs process inspection — **complete**. This includes bounded parsers, typed race/error handling, fixture/unit/fuzz tests, portable race/static analysis, and a live Ubuntu `/proc` smoke test.
- Milestone 3 — Process monitoring and identity primitives — **complete**. Sentinel now samples procfs continuously at a configurable interval, derives CPU and uptime, captures PID/start-time identity, launches explicit argv children, and terminates Linux children with TERM then KILL after a timeout.
- Milestone 4 — Supervisor lifecycle — **complete**. The single-workload supervisor owns cancellation, child waiting, restart state, capped exponential backoff, retry/reset policy, graceful stop, and concurrent status access.
- Milestone 6 — Daemon IPC — **complete**. `sentineld` exposes a bounded, peer-authorized, JSON-over-Unix-socket control protocol for daemon, process, service, health, and metrics status requests.
- Milestone 7 — CLI and operator workflow — **complete**. `sentinel` provides actionable local commands for status, `ps`, inspect, services, lifecycle actions, doctor, and version. Its real-binary integration test exercises the daemon boundary.

## Known issues

- The workspace is not a Git repository, so local repository metadata, issues, and CI history are unavailable.
- Milestone 1's configuration CLI and Makefile contract have not been implemented; do not mark it complete based on the module alone.
- Ubuntu WSL provides Go 1.18.1 for the procfs integration test, while the Windows development toolchain is Go 1.20.4. Align the Ubuntu toolchain with the project baseline before using it as the primary CI/package environment.

## Decisions made

- Linux is the primary target and Ubuntu LTS is the development, integration-test, and packaging target.
- Sentinel supervises explicitly configured workloads; it does not recreate systemd.
- The initial implementation is internal-package-first and standard-library-first.
- PID-only identity is insufficient: delayed process actions require start-time validation, with pidfd fallback decided by ADR.
- Control is local Unix-domain IPC authorized by peer credentials; remote control and arbitrary execution are out of scope.
- `/run` is volatile runtime storage and `/var/lib/sentinel` is the planned persistent-state location.
- Procfs inspection is a non-atomic observation: core `stat`/`status` data is returned as a partial observation when later optional `cmdline`, `exe`, or `fd` reads fail. A vanished process is a typed normal race, not a daemon-fatal condition.
- CPU percentage uses process tick delta divided by aggregate kernel tick delta, multiplied by online CPU count; process uptime derives from `/proc/uptime` and process start ticks. `USER_HZ` defaults to 100 with an explicit override.
- Sentinel supervises only children it launches; restart policies are `never`, `always`, and `on-failure`, with an explicit stop preventing restart.
- The local API is versioned JSON-over-UDS. The socket is restricted by private directory and `0600` permissions, with Linux peer-credential validation; HTTP-over-UDS and gRPC are intentionally not used.

## Upcoming work

1. Complete milestone 1's strict configuration validation and executable Makefile contract before beginning any later milestone.
2. Align the Ubuntu Go toolchain with the selected project baseline before establishing Linux CI/package validation.
3. Complete milestone 5 health checking and reload semantics; the current `health` and `metrics` API responses intentionally report `not-configured` until those contracts exist.
