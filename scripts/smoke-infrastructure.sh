#!/usr/bin/env bash
set -euo pipefail

export RELAY_REGION=local
export OIDC_ISSUER=https://identity.example/realms/coderoam
export OIDC_AUDIENCE=coderoam-mobile
export OIDC_JWKS_URL=https://identity.example/realms/coderoam/protocol/openid-connect/certs
export OIDC_SIGNING_ALGORITHM=RS256

compose=(docker compose --project-name coderoam-m1-smoke -f deployments/compose/docker-compose.yml)
migration_ledger_query="SELECT string_agg(scope || ':' || version, ',' ORDER BY scope, version) FROM coderoam_meta.schema_migrations"
resources_started=false
succeeded=false

cleanup() {
  if [[ "$resources_started" == true && "$succeeded" == false ]]; then
    echo "Smoke test failed. Dumping the last 100 container log lines:" >&2
    "${compose[@]}" logs --no-color --tail=100 >&2 || true
  fi
  "${compose[@]}" down --volumes --remove-orphans >/dev/null 2>&1 || true
}

assert_http_health() {
  local url="$1"
  local service="$2"
  local response

  if ! response="$(curl --fail --silent --show-error --max-time 5 "$url")"; then
    echo "health check request failed for $service" >&2
    return 1
  fi
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

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  cleanup
  trap cleanup EXIT

  "${compose[@]}" config --quiet
  resources_started=true
  WORKER_PROCESSING_ENABLED=false "${compose[@]}" up --build --detach --wait --wait-timeout 120

  POSTGRES_DSN='postgres://postgres:postgres@localhost:5432/coderoam?sslmode=disable' ./scripts/migrate.sh
  POSTGRES_DSN='postgres://postgres:postgres@localhost:5432/coderoam?sslmode=disable' ./scripts/migrate.sh
  applied_migrations="$("${compose[@]}" exec -T postgres psql -U postgres -d coderoam -Atc \
    "$migration_ledger_query")"
  expected_migrations='auth:1,auth:2,device:1,integration:1,outbox:1,preview:1,runbook:1,session:1,workspace:1'
  if [[ "$applied_migrations" != "$expected_migrations" ]]; then
    echo "unexpected migration ledger: $applied_migrations" >&2
    exit 1
  fi
  (cd packages/go/postgresx && \
    POSTGRES_TEST_DSN='postgres://postgres:postgres@localhost:5432/coderoam?sslmode=disable' \
      go test -count=1 -run 'Integration$' ./...)
  (cd services/control-plane && \
    POSTGRES_TEST_DSN='postgres://postgres:postgres@localhost:5432/coderoam?sslmode=disable' \
      go test -count=1 -run '^TestRepositoryIntegration$' ./internal/auth)
  (cd services/control-plane && \
    POSTGRES_TEST_DSN='postgres://postgres:postgres@localhost:5432/coderoam?sslmode=disable' \
      go test -count=1 -run '^TestRuntimeHandlerIntegration$' ./cmd/api)
  (cd services/control-plane && \
    POSTGRES_TEST_DSN='postgres://postgres:postgres@localhost:5432/coderoam?sslmode=disable' \
      go test -count=1 \
        -run '^(TestAuthorizationIntegration|TestAuthorizationLockIntegration|TestAuthorizationTimeoutIntegration|TestRepositoryIntegration)$' \
        ./internal/device)
  (cd services/control-plane && \
    POSTGRES_TEST_DSN='postgres://postgres:postgres@localhost:5432/coderoam?sslmode=disable' \
      go test -count=1 -run '^TestEnqueueIntegration$' ./internal/outbox)
  (cd services/control-plane && \
    POSTGRES_TEST_DSN='postgres://postgres:postgres@localhost:5432/coderoam?sslmode=disable' \
      go test -count=1 \
        -run '^(TestRepositoryCreateIntegration|TestServiceStartIntegration)$' \
        ./internal/session)
  (cd services/control-plane && \
    POSTGRES_TEST_DSN='postgres://postgres:postgres@localhost:5432/coderoam?sslmode=disable' \
      go test -count=1 \
        -run '^(TestRepositoryAgentRevocationIntegration|TestRepositoryAuthorizeAgentIntegration|TestRepositoryAuthorizeAgentLockIntegration|TestRepositoryAuthorizeAgentTimeoutIntegration|TestRepositoryAuthorizeProjectIntegration|TestRepositoryAuthorizeProjectLockIntegration|TestRepositoryListProjectsIntegration)$' \
        ./internal/workspace)
  (cd services/worker && \
    POSTGRES_TEST_DSN='postgres://postgres:postgres@localhost:5432/coderoam?sslmode=disable' \
      go test -count=1 \
        -run '^(TestProcessorIntegration|TestRepositoryClaimFinishIntegration)$' \
        ./internal/outbox)
  (cd services/worker && \
    POSTGRES_TEST_DSN='postgres://postgres:postgres@localhost:5432/coderoam?sslmode=disable' \
      go test -count=1 -run '^TestRunWorkerIntegration$' ./cmd/worker)

  assert_http_health http://localhost:8080/healthz coderoam-control-plane
  assert_http_health http://localhost:8090/healthz coderoam-relay
  assert_service_running worker
  succeeded=true
fi
