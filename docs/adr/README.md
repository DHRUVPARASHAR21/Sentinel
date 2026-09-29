# Architecture decision records

Use a short ADR for a decision that is hard to reverse, materially affects security/operations, or changes an external contract. Name records as `NNNN-short-title.md` and include status, context, decision, consequences, and alternatives.

Create ADRs before implementation for:

1. Supported supervision model and ownership of child processes (spawned only, adopt existing, or both).
2. Process identity and signalling strategy (start-time validation, pidfd support/fallback).
3. Configuration format, reload semantics, and secret-reference policy.
4. Unix control protocol, peer authorization, and socket ownership.
5. Metrics transport/exposure default and label-cardinality policy.
6. Privilege model and systemd sandboxing/capability policy.
7. Restart policy and behavior across daemon restart.
8. Persistent state model and compatibility/migration rules.
9. Debian package layout and upgrade/rollback behavior.
