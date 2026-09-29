# Sentinel

Sentinel is a Linux-first Go process supervisor and local observability daemon. It is intended for small, explicitly declared workloads where process lifecycle safety and `/proc` visibility matter.

```text
sentinel CLI ── Unix socket ── sentineld ── supervisor ── child process
                                  │              └────── /proc
                                  └── loopback /metrics
```

## Features

- Defensive `/proc` inspection: PID identity, credentials, memory, CPU ticks, threads, FDs, state, argv, and executable path.
- Child-only supervision with `never`, `always`, and `on-failure` restart policies; capped exponential backoff and TERM→KILL shutdown.
- Local, peer-authorized JSON-over-Unix-socket CLI.
- Prometheus-compatible metrics on `127.0.0.1:9464` and structured JSON daemon lifecycle logs.
- Process, TCP, and HTTP health-check primitives.

## Install and quick start

On Ubuntu, build a package with `make package`, then install `sudo dpkg -i packaging/debian/out/*.deb`. Binaries live in `/usr/bin`, packaged defaults in `/etc/sentinel`, runtime sockets in `/run/sentinel`, persistent state in `/var/lib/sentinel`, and reserved log storage in `/var/log/sentinel`.

Until configuration loading lands, declare services explicitly:

```sh
./dist/sentineld --socket /tmp/sentinel.sock --service demo=/bin/sh
./dist/sentinel --socket /tmp/sentinel.sock status
./dist/sentinel --socket /tmp/sentinel.sock ps
curl http://127.0.0.1:9464/metrics
```

`--service` currently accepts an absolute executable only; command arguments and config-file loading are not yet implemented.

## CLI

`sentinel status`, `ps`, `inspect PID`, `services`, `start NAME`, `stop NAME`, `restart NAME`, `doctor`, and `version` are available. Errors include the socket path and underlying connection reason.

## Development and tests

Requires Go 1.20+ and Ubuntu for Linux integration/package validation.

```sh
make build
make test
make test-race
make lint
make integration
make package
```

## Security and systemd

The socket is private (`0700` parent, `0600` socket) and Linux checks peer credentials. Sentinel never invokes a shell for child launch and does not expose command lines or environments as metric labels. The packaged [systemd unit](packaging/systemd/sentinel.service) uses an unprivileged account, `NoNewPrivileges`, `ProtectSystem`, `ProtectHome`, `PrivateTmp`, and an empty capability bounding set. See [docs/security.md](docs/security.md).

## Demo

1. `make build`
2. Start `./dist/sentineld --socket /tmp/sentinel.sock --service demo=/bin/false`.
3. Run `./dist/sentinel --socket /tmp/sentinel.sock start demo`; it exits and is restarted according to policy.
4. Inspect `./dist/sentinel --socket /tmp/sentinel.sock services` and `curl http://127.0.0.1:9464/metrics`.
5. Press `Ctrl-C` in the daemon terminal for graceful shutdown.

## Limitations and roadmap

Configuration-file loading, configured health-check attachment, service arguments, config reload, full systemd notification, and package upgrade smoke tests remain planned. Sentinel does not recreate systemd or manage remote/container workloads.

## Contributing and license

See [CONTRIBUTING.md](CONTRIBUTING.md), [PLAN.md](PLAN.md), and [docs/status.md](docs/status.md). A license has not been selected by the repository owner; no license file is included until that decision is made.
