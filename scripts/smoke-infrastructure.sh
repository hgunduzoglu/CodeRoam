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
SELECT id FROM session.pairing_attempts WHERE false FOR UPDATE;
INSERT INTO device.devices (
  id, user_id, name, platform, static_public_key, public_key_fingerprint, paired_at
) VALUES (
  '10000000000000000000000000000001',
  '10000000000000000000000000000002',
  'Runtime privilege smoke device',
  'ios',
  decode(repeat('11', 32), 'hex'),
  'x25519-sha256:' || encode(sha256(decode(repeat('11', 32), 'hex')), 'hex'),
  now()
);
INSERT INTO workspace.agents (
  id, user_id, name, static_public_key, public_key_fingerprint, version, created_at
) VALUES (
  '10000000000000000000000000000003',
  '10000000000000000000000000000002',
  'Runtime privilege smoke agent',
  decode(repeat('22', 32), 'hex'),
  'x25519-sha256:' || encode(sha256(decode(repeat('22', 32), 'hex')), 'hex'),
  'smoke',
  now()
);
INSERT INTO device.devices (
  id, user_id, name, platform, static_public_key, public_key_fingerprint, paired_at
) VALUES (
  '10000000000000000000000000000001',
  '10000000000000000000000000000002',
  'Runtime privilege smoke device',
  'ios',
  decode(repeat('11', 32), 'hex'),
  'x25519-sha256:' || encode(sha256(decode(repeat('11', 32), 'hex')), 'hex'),
  now()
) ON CONFLICT DO NOTHING;
SELECT id FROM device.devices
WHERE id = '10000000000000000000000000000001'
   OR public_key_fingerprint =
     'x25519-sha256:' || encode(sha256(decode(repeat('11', 32), 'hex')), 'hex')
FOR UPDATE;
INSERT INTO workspace.agents (
  id, user_id, name, static_public_key, public_key_fingerprint, version, created_at
) VALUES (
  '10000000000000000000000000000003',
  '10000000000000000000000000000002',
  'Runtime privilege smoke agent',
  decode(repeat('22', 32), 'hex'),
  'x25519-sha256:' || encode(sha256(decode(repeat('22', 32), 'hex')), 'hex'),
  'smoke',
  now()
) ON CONFLICT DO NOTHING;
SELECT id FROM workspace.agents
WHERE id = '10000000000000000000000000000003'
   OR public_key_fingerprint =
     'x25519-sha256:' || encode(sha256(decode(repeat('22', 32), 'hex')), 'hex')
FOR UPDATE;
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
INSERT INTO session.pairing_attempts (
  id, agent_id, agent_static_public_key, agent_key_fingerprint,
  agent_display_name, agent_version, protocol_version, relay_region,
  bootstrap_credential_hash, expires_at, failed_attempt_count, state,
  created_at, updated_at
) VALUES (
  '20000000000000000000000000000001',
  '20000000000000000000000000000002',
  decode(repeat('33', 32), 'hex'),
  'x25519-sha256:' || encode(sha256(decode(repeat('33', 32), 'hex')), 'hex'),
  'Runtime privilege smoke pairing agent',
  'smoke',
  1,
  'local',
  decode(repeat('44', 32), 'hex'),
  now() + interval '5 minutes',
  0,
  'open',
  now(),
  now()
);
UPDATE session.pairing_attempts
SET state = 'claimed',
    claimed_user_id = '20000000000000000000000000000003',
    device_id = '20000000000000000000000000000004',
    device_display_name = 'Runtime privilege smoke phone',
    device_platform = 'ios',
    device_static_public_key = decode(repeat('55', 32), 'hex'),
    device_key_fingerprint =
      'x25519-sha256:' || encode(sha256(decode(repeat('55', 32), 'hex')), 'hex'),
    claimed_at = now(),
    updated_at = now()
WHERE id = '20000000000000000000000000000001';
UPDATE session.pairing_attempts
SET state = 'confirming',
    mobile_channel_binding = decode(repeat('66', 32), 'hex'),
    mobile_confirmed_at = now(),
    updated_at = now()
WHERE id = '20000000000000000000000000000001';
UPDATE session.pairing_attempts
SET agent_channel_binding = decode(repeat('66', 32), 'hex'),
    agent_confirmed_at = now(),
    updated_at = now()
WHERE id = '20000000000000000000000000000001';
UPDATE session.pairing_attempts
SET state = 'consumed', consumed_at = now(), updated_at = now()
WHERE id = '20000000000000000000000000000001';
ROLLBACK;
SQL

  least_privilege_result="$("${compose[@]}" exec -T postgres psql -U postgres -d coderoam -Atc "
    SELECT
      NOT has_column_privilege('$runtime_role', 'device.devices', 'user_id', 'UPDATE')
      AND NOT has_column_privilege('$runtime_role', 'device.devices', 'static_public_key', 'UPDATE')
      AND NOT has_column_privilege('$runtime_role', 'device.devices', 'public_key_fingerprint', 'UPDATE')
      AND NOT has_column_privilege('$runtime_role', 'device.devices', 'paired_at', 'UPDATE')
      AND NOT has_column_privilege('$runtime_role', 'device.devices', 'revoked_at', 'UPDATE')
      AND has_column_privilege('$runtime_role', 'device.devices', 'id', 'INSERT')
      AND has_column_privilege('$runtime_role', 'device.devices', 'static_public_key', 'INSERT')
      AND NOT has_column_privilege('$runtime_role', 'device.devices', 'revoked_at', 'INSERT')
      AND NOT has_column_privilege('$runtime_role', 'device.devices', 'last_seen_at', 'INSERT')
      AND NOT has_column_privilege('$runtime_role', 'workspace.agents', 'user_id', 'UPDATE')
      AND NOT has_column_privilege('$runtime_role', 'workspace.agents', 'static_public_key', 'UPDATE')
      AND NOT has_column_privilege('$runtime_role', 'workspace.agents', 'public_key_fingerprint', 'UPDATE')
      AND NOT has_column_privilege('$runtime_role', 'workspace.agents', 'created_at', 'UPDATE')
      AND NOT has_column_privilege('$runtime_role', 'workspace.agents', 'revoked_at', 'UPDATE')
      AND has_column_privilege('$runtime_role', 'workspace.agents', 'id', 'INSERT')
      AND has_column_privilege('$runtime_role', 'workspace.agents', 'static_public_key', 'INSERT')
      AND NOT has_column_privilege('$runtime_role', 'workspace.agents', 'revoked_at', 'INSERT')
      AND NOT has_column_privilege('$runtime_role', 'workspace.agents', 'last_seen_at', 'INSERT')
      AND NOT has_column_privilege('$runtime_role', 'workspace.environments', 'user_id', 'UPDATE')
      AND NOT has_column_privilege('$runtime_role', 'workspace.projects', 'root_path', 'UPDATE')
      AND NOT has_column_privilege('$runtime_role', 'session.sessions', 'user_id', 'UPDATE')
      AND has_column_privilege('$runtime_role', 'session.pairing_attempts', 'id', 'INSERT')
      AND has_column_privilege('$runtime_role', 'session.pairing_attempts', 'agent_id', 'INSERT')
      AND has_column_privilege('$runtime_role', 'session.pairing_attempts', 'bootstrap_credential_hash', 'INSERT')
      AND NOT has_column_privilege('$runtime_role', 'session.pairing_attempts', 'consumed_at', 'INSERT')
      AND has_column_privilege('$runtime_role', 'session.pairing_attempts', 'consumed_at', 'UPDATE')
      AND NOT has_column_privilege('$runtime_role', 'session.pairing_attempts', 'agent_id', 'UPDATE')
      AND NOT has_column_privilege('$runtime_role', 'session.pairing_attempts', 'agent_static_public_key', 'UPDATE')
      AND NOT has_column_privilege('$runtime_role', 'session.pairing_attempts', 'bootstrap_credential_hash', 'UPDATE')
      AND NOT has_column_privilege('$runtime_role', 'session.pairing_attempts', 'expires_at', 'UPDATE')
      AND NOT has_table_privilege('$runtime_role', 'session.pairing_attempts', 'INSERT')
      AND NOT has_table_privilege('$runtime_role', 'session.pairing_attempts', 'UPDATE')
      AND NOT has_table_privilege('$runtime_role', 'session.pairing_attempts', 'DELETE')
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
  expected_migrations='auth:1,auth:2,device:1,device:2,integration:1,outbox:1,preview:1,runbook:1,session:1,session:2,session:3,workspace:1,workspace:2'
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
        -run '^(TestAuthorizationIntegration|TestAuthorizationLockIntegration|TestAuthorizationTimeoutIntegration|TestDeviceFingerprintMigrationIntegration|TestRepositoryIntegration|TestRepositoryRegisterPairedIntegration|TestRepositoryRegisterPairedTimeoutIntegration|TestRepositoryRegisterPairedConcurrentRetryIntegration)$' \
        ./internal/device)
  (cd services/control-plane && \
    POSTGRES_TEST_DSN='postgres://postgres:postgres@localhost:5432/coderoam?sslmode=disable' \
      go test -count=1 -run '^TestEnqueueIntegration$' ./internal/outbox)
  (cd services/control-plane && \
    POSTGRES_TEST_DSN='postgres://postgres:postgres@localhost:5432/coderoam?sslmode=disable' \
      go test -count=1 \
        -run '^(TestPairingAttemptMigrationIntegration|TestPairingAttemptRepositoryIntegration|TestPairingCompletionIntegration|TestRepositoryCreateIntegration|TestServiceStartIntegration)$' \
        ./internal/session)
  (cd services/control-plane && \
    POSTGRES_TEST_DSN='postgres://postgres:postgres@localhost:5432/coderoam?sslmode=disable' \
      go test -count=1 \
        -run '^(TestAgentFingerprintMigrationIntegration|TestRepositoryAgentRevocationIntegration|TestRepositoryAuthorizeAgentIntegration|TestRepositoryAuthorizeAgentLockIntegration|TestRepositoryAuthorizeAgentTimeoutIntegration|TestRepositoryAuthorizeProjectIntegration|TestRepositoryAuthorizeProjectLockIntegration|TestRepositoryListProjectsIntegration|TestRepositoryRegisterPairedAgentIntegration|TestRepositoryRegisterPairedAgentTimeoutIntegration|TestRepositoryRegisterPairedAgentConcurrentRetryIntegration)$' \
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
