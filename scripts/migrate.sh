#!/usr/bin/env bash
set -euo pipefail

: "${POSTGRES_DSN:=postgres://postgres:postgres@localhost:5432/coderoam?sslmode=disable}"
export POSTGRES_DSN

# Schema scopes are explicit so a new module cannot join the writable migration set accidentally.
migration_scopes=(
  'auth=services/control-plane/internal/auth/migrations'
  'device=services/control-plane/internal/device/migrations'
  'integration=services/control-plane/internal/integration/migrations'
  'outbox=services/control-plane/internal/outbox/migrations'
  'preview=services/control-plane/internal/preview/migrations'
  'runbook=services/control-plane/internal/runbook/migrations'
  'session=services/control-plane/internal/session/migrations'
  'workspace=services/control-plane/internal/workspace/migrations'
)

go run ./packages/go/postgresx/cmd/migrate "${migration_scopes[@]}"
