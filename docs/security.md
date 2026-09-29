# Security model

## Trust boundaries and sensitive operations

| Boundary/operation | Risk | Required control |
| --- | --- | --- |
| Configuration input | command/path injection, unsafe ownership | strict schema; reject unknown/invalid values; root-owned config; no shell invocation |
| Workload launch | privilege escalation, inherited secrets | use `exec.Command` with explicit argv/environment; no `sh -c`; define UID/GID/capability policy before launch |
| PID signalling | PID reuse or signalling an unintended process | verify start-time identity immediately before action; prefer pidfds when supported; classify `ESRCH` as a race, not success by default |
| `/proc` reads | TOCTOU, malformed data, information disclosure | bounded reads, typed parse errors, no assumptions of file permanence, least data collection |
| Unix control socket | unauthorized supervision or request abuse | socket directory owned by trusted user; `0600`/group-restricted mode; peer credential authorization; bounded framing/timeouts; no arbitrary exec endpoint |
| Metrics endpoint | disclosure or unauthenticated access | default loopback/Unix socket; explicit opt-in for non-local bind; expose aggregate labels only; no command/env labels |
| Runtime/state paths | symlink/path traversal, stale socket | trusted parent directory, `lstat`/ownership/mode validation, atomic create practices, safe stale-socket handling |
| Logs | secret leakage, log injection | structured fields; redact command arguments/environment and credentials; normalize untrusted values |
| Debian/systemd install | privilege and unit hardening | root-owned files; systemd hardening reviewed; package scripts idempotent and minimal |

## Security baseline

- Run with the least privileges that meet the configured supervision model. Privilege dropping and the supported workload ownership model require an explicit ADR.
- Never construct shell command strings. Configuration represents executable path plus argument array.
- Apply limits to config size, socket request size, line length, `/proc` reads, concurrent requests, and restart rate.
- Treat all kernel and client input as untrusted operational input: return errors, not panics.
- Secrets are configuration references where possible, not values logged or surfaced by status/metrics APIs.
- Security-sensitive defaults must fail closed: unsafe directory ownership, mode, or socket peer identity rejects startup/request.

## Hardening review

- The socket parent is Linux-validated as owned by the daemon user and not accessible to group/other users; Sentinel removes only an existing socket, never an arbitrary path.
- No shell is used for supervised commands. CLI service definitions require an absolute executable path; a future config loader must reject symlinks and untrusted writable ancestors before enabling arbitrary path input.
- Procfs data, process names, and client JSON are untrusted. Request and procfs reads are bounded; logs use JSON fields and never include argv, environment, or raw client bodies.
- Restart caps, health-check intervals/timeouts, client limits, and socket deadlines bound common resource-exhaustion paths. A compromised supervised child remains able to consume its own granted resources; cgroup/resource controls are deferred to systemd policy.
- `packaging/systemd/sentinel.service` runs an unprivileged `sentinel` account with `NoNewPrivileges`, private temporary storage, read-only system paths, no capabilities, and explicit writable runtime/state paths. `ProtectSystem=full` is selected rather than `strict` because package/config paths still require ordinary system visibility; protected-home is compatible with the supported `/run` and `/var/lib` paths.

## Review triggers

Security review is required for changes to process credentials/capabilities, socket authorization/protocol, listener exposure, config loading paths, signal rules, systemd hardening, package maintainer scripts, or logged/metric fields.
