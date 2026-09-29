# Sentinel contributor guide

## Scope and principles

Sentinel is a Linux-first Go daemon for supervising opted-in processes and exposing local observability. Prefer the Go standard library; add a dependency only when it removes meaningful security or maintenance risk.

## Repository conventions

- Keep production code under `cmd/` and `internal/`; public packages need an explicit compatibility commitment.
- Target Ubuntu LTS and Linux kernels with procfs available. Keep platform-specific code behind build tags when needed.
- Pass `context.Context` through blocking operations and make shutdown idempotent.
- Parse `/proc` defensively: files may disappear or change between reads. Never panic on malformed or partial kernel data.
- Use structured logs with stable event names and avoid logging secrets, full command-line arguments, or environment values by default.
- Treat PIDs as reusable identifiers. Validate process identity with start time (and, where required, pidfd) before signalling.
- Default Unix socket and state paths must be root-owned, non-world-writable, and created with restrictive permissions.

## Required checks before a change is proposed

```sh
go fmt ./...
go vet ./...
go test ./...
go test -race ./...
go test -count=1 ./...
```

Run integration tests only on Linux/Ubuntu and document required privileges. Package changes must also run the Debian package build and installation smoke test described in `docs/development.md`.

## Documentation and decisions

- Update `README.md` for user-visible behavior.
- Update `docs/architecture.md` when a component boundary changes.
- Update `docs/security.md` for changed trust boundaries, permissions, or signal behavior.
- Add an ADR in `docs/adr/` for decisions listed there or any hard-to-reverse operational choice.
- Do not commit generated binaries, coverage profiles, or Debian build output.
