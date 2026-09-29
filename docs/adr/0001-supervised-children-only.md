# ADR 0001: Supervise children launched by Sentinel

## Status

Accepted.

## Context

Adopting arbitrary existing PIDs creates ambiguous ownership and PID-reuse risk.

## Decision

The initial supervisor manages only children it launches. It retains the Go child handle for waiting and uses the procfs start-time tuple for monitoring identity. Adopting processes requires a future ADR.

## Consequences

Lifecycle behavior is deterministic and does not signal unrelated existing processes. Existing services must be launched by Sentinel to be supervised.
