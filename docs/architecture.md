# Architecture

## Concise design

`sentineld` is a single long-running Linux daemon. At startup it validates configuration, initializes trusted runtime paths, starts the control and metrics listeners, and reconciles configured workloads. A supervisor owns desired state and lifecycle decisions. It obtains process facts through a narrow procfs adapter and delegates signalling/spawning to a process adapter. Control requests are authorized at the Unix-socket boundary before calling the supervisor. Metrics and logs observe component outcomes; neither controls lifecycle behavior.

```text
config file -> config -> runtime/composition -> supervisor -> process
                                      |              -> procfs
Unix socket -> control/authz ---------+              -> metrics
systemd ----> systemd adapter --------+              -> logging
```

## Major components and boundaries

| Component | Owns | Must not own |
| --- | --- | --- |
| `cmd/sentineld` | dependency wiring, CLI, exit code | domain policy |
| `config` | schema, defaults, validation | filesystem mutation or signals |
| `supervisor` | desired state, restart/backoff, lifecycle state machine | raw `/proc` parsing or peer credentials |
| `process` | child launch, signal delivery, process identity checks | restart policy |
| `monitor` | periodic procfs sampling and derived resource usage | process lifecycle decisions |
| `procfs` | read/parse selected procfs records; typed results/errors and partial observations | lifecycle changes |
| `control` | socket framing, peer authorization, request validation | direct process manipulation |
| `metrics` | metric registration/rendering | reading mutable supervisor internals unsafely |
| `systemd` | notify/watchdog integration and unit-facing behavior | config parsing |
| `logging` | structured event schema, safe fields/redaction | business decisions |

Initial implementation should use concrete types at the composition root. Introduce interfaces only at an actual test or operating-system boundary (`procfs`, clock, child/process operations), not merely for layering.

`procfs` observations are intentionally point-in-time rather than atomic snapshots. A process can exit while an observation is assembled: callers treat the typed `process disappeared` result as a normal race. When an optional record such as `cmdline`, `exe`, or `fd` becomes inaccessible after core identity/status data is read, the adapter returns the usable partial observation alongside the typed error.

The control plane uses one versioned JSON request and one JSON response per Unix-domain socket connection. The daemon bounds each request at 64 KiB, caps concurrent clients, applies deadlines, removes only a verified stale socket, and creates sockets with `0600` mode beneath a private `0700` daemon-owned directory. Linux verifies `SO_PEERCRED` against the daemon effective UID. HTTP-over-UDS and gRPC were rejected for this local, non-streaming command surface; see ADR 0003.

The monitor samples procfs at a configured interval. CPU percentage is calculated without assuming a tick frequency: `((process user ticks + system ticks) delta / aggregate /proc/stat cpu ticks delta) × online CPUs × 100`. This reports the share of one CPU a process used during the sample interval; aggregate kernel ticks include all online CPUs. Process uptime is `system uptime - (process start-time ticks / USER_HZ)`; Sentinel defaults `USER_HZ` to 100 and exposes an override for Linux environments with a different ABI value.

Metrics use Prometheus text exposition with only configured service names as labels. PIDs, command lines, executable paths, and arbitrary process names are deliberately excluded from labels because they are high-cardinality or untrusted. Health checks are a single cancellable loop per configured check, with per-probe contexts and success/failure thresholds; supported probes are process-state, TCP connect, and HTTP GET.

## Linux assumptions

- Linux procfs is mounted at `/proc`; file reads can race process exit and return `ENOENT`, partial content, or permission errors.
- Sentinel runs on Ubuntu LTS initially, with systemd and Unix-domain sockets available.
- `/run` is volatile; persistent state belongs under `/var/lib/sentinel` and logs are emitted to journald/stdout rather than managed log files by default.
- PID values are reused. PID-only identity is never sufficient for a delayed signal or status assertion.
- systemd is optional for foreground development but required for packaged service integration.
- Prometheus scrapes a local HTTP listener only after listener exposure/authentication policy is explicitly configured.

## Data and control flow

1. Decode and validate immutable configuration into a typed snapshot.
2. Establish state/runtime directories with ownership and mode checks.
3. Reconcile workload definitions into supervisor state machines.
4. Process/procfs adapters report observations; the supervisor decides transitions.
5. Authorized control requests query or request transitions through the supervisor.
6. Metrics take immutable snapshots or synchronized reads; logs record events without sensitive payloads.
7. Shutdown stops intake, requests workload termination according to policy, drains goroutines, and reports final status to systemd.

## Non-goals for the first release

- Container orchestration, distributed coordination, remote control APIs, plugin systems, and arbitrary command execution over the socket.
- A public Go SDK or broad cross-platform support.
