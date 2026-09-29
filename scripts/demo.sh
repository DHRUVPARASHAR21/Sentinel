#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
demo_dir="$(mktemp -d)"
trap 'rm -rf "$demo_dir"' EXIT
chmod 700 "$demo_dir"
"$root/dist/sentineld" --socket "$demo_dir/sentinel.sock" --metrics-listen 127.0.0.1:19464 --service demo=/bin/false >"$demo_dir/daemon.log" 2>&1 &
daemon_pid=$!
trap 'kill -TERM "$daemon_pid" 2>/dev/null || true; wait "$daemon_pid" 2>/dev/null || true; rm -rf "$demo_dir"' EXIT
for _ in $(seq 1 50); do
  test -S "$demo_dir/sentinel.sock" && break
  sleep 0.1
done
"$root/dist/sentinel" --socket "$demo_dir/sentinel.sock" status
"$root/dist/sentinel" --socket "$demo_dir/sentinel.sock" start demo
sleep 1
"$root/dist/sentinel" --socket "$demo_dir/sentinel.sock" services
curl -fsS http://127.0.0.1:19464/metrics
kill -TERM "$daemon_pid"
wait "$daemon_pid"
