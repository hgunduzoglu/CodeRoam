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

assert_runtime_database_privileges() {
  local runtime_role=coderoam_runtime_smoke
  local least_privilege_result

  "${compose[@]}" exec -T postgres psql -U postgres -d coderoam -v ON_ERROR_STOP=1 \
    -c "CREATE ROLE $runtime_role NOLOGIN"
  "${compose[@]}" exec -T postgres psql -U postgres -d coderoam -v ON_ERROR_STOP=1 \
    -v runtime_role="$runtime_role" <scripts/grant-runtime-database-privileges.sql
  "${compose[@]}" exec -T postgres psql -U postgres -d coderoam -v ON_ERROR_STOP=1 \
    -v runtime_role="$runtime_role" <scripts/grant-runtime-database-privileges.sql

  "${compose[@]}" exec -T postgres psql -U postgres -d coderoam -v ON_ERROR_STOP=1 <<SQL
BEGIN;
SET LOCAL ROLE $runtime_role;
SELECT id FROM device.devices WHERE false FOR SHARE;
SELECT id FROM workspace.agents WHERE false FOR SHARE;
SELECT p.id
FROM workspace.projects AS p
JOIN workspace.environments AS e ON e.id = p.environment_id
WHERE false
FOR SHARE OF p, e;
INSERT INTO session.sessions (
  id, user_id, device_id, agent_id, project_id, relay_region, started_at
) VALUES (
  '00000000000000000000000000000001',
  '00000000000000000000000000000002',
  '00000000000000000000000000000003',
  '00000000000000000000000000000004',
  '00000000000000000000000000000005',
  'local',
  now()
);
SELECT id FROM session.sessions
WHERE id = '00000000000000000000000000000001'
FOR SHARE;
ROLLBACK;
SQL

  least_privilege_result="$("${compose[@]}" exec -T postgres psql -U postgres -d coderoam -Atc "
    SELECT
      NOT has_column_privilege('$runtime_role', 'device.devices', 'user_id', 'UPDATE')
      AND NOT has_column_privilege('$runtime_role', 'device.devices', 'revoked_at', 'UPDATE')
      AND NOT has_column_privilege('$runtime_role', 'workspace.agents', 'static_public_key', 'UPDATE')
      AND NOT has_column_privilege('$runtime_role', 'workspace.agents', 'revoked_at', 'UPDATE')
      AND NOT has_column_privilege('$runtime_role', 'workspace.environments', 'user_id', 'UPDATE')
      AND NOT has_column_privilege('$runtime_role', 'workspace.projects', 'root_path', 'UPDATE')
      AND NOT has_column_privilege('$runtime_role', 'session.sessions', 'user_id', 'UPDATE')
      AND NOT has_table_privilege('$runtime_role', 'device.devices', 'INSERT')
      AND NOT has_table_privilege('$runtime_role', 'workspace.projects', 'INSERT')
      AND NOT has_table_privilege('$runtime_role', 'session.sessions', 'DELETE')
  ")"
  if [[ "$least_privilege_result" != t ]]; then
    echo "runtime database role has privileges outside its bounded write set" >&2
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
  assert_runtime_database_privileges
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

  assert_http_health http://localhost:8080/health coderoam-control-plane
  assert_http_health http://localhost:8090/health coderoam-relay
  assert_service_running worker
  succeeded=true
fi
