#!/usr/bin/env bash
set -euo pipefail

: "${POSTGRES_DSN:=postgres://postgres:postgres@localhost:5432/coderoam?sslmode=disable}"

if ! command -v psql >/dev/null 2>&1; then
  echo "psql is required to apply the starter migrations." >&2
  exit 1
fi

while IFS= read -r migration; do
  echo "Applying $migration"
  psql "$POSTGRES_DSN" -v ON_ERROR_STOP=1 -f "$migration"
done < <(find services/control-plane/internal -path '*/migrations/*.sql' -type f | sort)
