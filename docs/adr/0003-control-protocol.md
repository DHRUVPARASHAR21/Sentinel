# ADR 0003: JSON messages over a Unix-domain socket

## Status

Accepted.

## Context

Sentinel needs a local, low-volume control API. Newline-delimited JSON is easy to inspect and strictly bound. HTTP over a Unix socket would provide familiar routing but adds headers, parsing surface, and no needed capability. gRPC requires a dependency and schema/tooling that is disproportionate to this local API.

## Decision

Use one versioned JSON request and one JSON response per Unix-domain connection. The socket directory is mode `0700` and the socket is mode `0600`; Linux additionally verifies the connecting peer UID matches the daemon UID.

## Consequences

The protocol is simple, bounded, and dependency-free. Streaming and remote clients are deliberately out of scope.
