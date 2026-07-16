#!/usr/bin/env bash
set -euo pipefail

source ./scripts/smoke-infrastructure.sh

expected_compose=(docker compose --project-name coderoam-m1-smoke -f deployments/compose/docker-compose.yml)
if [[ "${compose[*]}" != "${expected_compose[*]}" ]]; then
  echo "smoke test does not use the isolated Compose project" >&2
  exit 1
fi

curl() {
  return 7
}

if error="$(assert_http_health http://localhost:1/healthz unavailable-service 2>&1)"; then
  echo "assert_http_health unexpectedly succeeded" >&2
  exit 1
fi
if [[ "$error" != *"health check request failed for unavailable-service"* ]]; then
  echo "assert_http_health did not report the failing service" >&2
  exit 1
fi

compose_calls=()
fake_compose() {
  compose_calls+=("$*")
  if [[ "$1" == logs ]]; then
    echo "bounded container failure"
    return 71
  fi
  return 72
}

compose=(fake_compose)
stderr_file="$(mktemp)"
trap 'rm -f "$stderr_file"' EXIT

resources_started=false
succeeded=false
compose_calls=()
cleanup 2>"$stderr_file"
if [[ -s "$stderr_file" || "${compose_calls[*]}" != "down --volumes --remove-orphans" ]]; then
  echo "initial cleanup emitted logs or used unexpected arguments" >&2
  exit 1
fi

resources_started=true
succeeded=false
compose_calls=()
cleanup 2>"$stderr_file"
failure_output="$(<"$stderr_file")"
if [[ "$failure_output" != *"bounded container failure"* ]]; then
  echo "cleanup did not capture failure logs" >&2
  exit 1
fi
if [[ "${compose_calls[0]}" != "logs --no-color --tail=100" ||
  "${compose_calls[1]}" != "down --volumes --remove-orphans" ]]; then
  echo "failure cleanup used unexpected Compose arguments" >&2
  exit 1
fi

succeeded=true
compose_calls=()
cleanup 2>"$stderr_file"
if [[ -s "$stderr_file" || "${compose_calls[*]}" != "down --volumes --remove-orphans" ]]; then
  echo "successful cleanup emitted logs or used unexpected arguments" >&2
  exit 1
fi

if bash -c '
  source ./scripts/smoke-infrastructure.sh
  failing_compose() { return 91; }
  compose=(failing_compose)
  resources_started=true
  succeeded=false
  trap cleanup EXIT
  exit 42
' >/dev/null 2>&1; then
  exit_status=0
else
  exit_status=$?
fi
if [[ "$exit_status" -ne 42 ]]; then
  echo "cleanup masked the original failure status" >&2
  exit 1
fi
