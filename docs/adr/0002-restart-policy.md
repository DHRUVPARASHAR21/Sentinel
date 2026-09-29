# ADR 0002: Restart policy and reset window

## Status

Accepted.

## Decision

Restart policies are `never`, `always`, and `on-failure`. A failed run increments the retry count; successful runs reset it. Failed retry history resets only after a continuous healthy run lasting the configured reset window. Restart delay is `min(initial * 2^(retry-1), maximum)`.

## Consequences

Crash loops are bounded and visible. An explicit stop never restarts a workload regardless of policy.
