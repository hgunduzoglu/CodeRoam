#!/usr/bin/env bash
set -euo pipefail

compose=(docker compose --project-name coderoam-m1-smoke -f deployments/compose/docker-compose.yml)

cleanup() {
  "${compose[@]}" down --volumes --remove-orphans >/dev/null 2>&1 || true
}

assert_http_health() {
  local url="$1"
  local service="$2"
  local response

  response="$(curl --fail --silent --show-error --max-time 5 "$url")"
  if [[ "$response" != *"\"service\":\"$service\""* || "$response" != *'"status":"ok"'* ]]; then
    echo "unexpected health response from $service" >&2
    return 1
  fi
}

assert_service_running() {
  local service="$1"
  local running

  running="$("${compose[@]}" ps --services --status running "$service")"
  if [[ "$running" != "$service" ]]; then
    echo "$service is not running" >&2
    return 1
  fi
}

cleanup
trap cleanup EXIT

"${compose[@]}" config --quiet
"${compose[@]}" up --build --detach --wait --wait-timeout 120

POSTGRES_DSN='postgres://postgres:postgres@localhost:5432/coderoam?sslmode=disable' ./scripts/migrate.sh
POSTGRES_DSN='postgres://postgres:postgres@localhost:5432/coderoam?sslmode=disable' ./scripts/migrate.sh

assert_http_health http://localhost:8080/healthz coderoam-control-plane
assert_http_health http://localhost:8090/healthz coderoam-relay
assert_service_running worker
