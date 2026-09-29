# Testing strategy

## Test layers

- Unit tests: config validation, `/proc` parsers using fixtures, lifecycle transitions, backoff, request framing, and metric rendering.
- Race tests: lifecycle state, fan-out metrics reads, control requests, shutdown, and restart activity.
- Linux integration tests: actual child processes, signal escalation, process exit races, Unix peer credentials, procfs disappearance, and systemd notification adapter behavior.
- Packaging smoke tests: build the `.deb`, install in a disposable Ubuntu environment, start/stop the unit, and query a local status endpoint.

Fixtures must be minimal and may not contain real process command lines, environments, or host data. Tests should inject time and process seams only where determinism requires it.

## Baseline commands

```sh
go test ./...
go test -race ./...
go test -count=1 ./...
go vet ./...
go fmt -d .
```

Linux-only checks and package commands are staged in `docs/development.md` because the module and package metadata are not yet present.
