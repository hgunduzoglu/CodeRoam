#!/usr/bin/env bash
set -euo pipefail

required=(git go docker python3 node npm)
optional=(flutter dart buf protoc)

echo "CodeRoam tool check"
for command_name in "${required[@]}"; do
  if ! command -v "$command_name" >/dev/null 2>&1; then
    echo "missing required command: $command_name" >&2
    exit 1
  fi
  echo "ok: $command_name"
done

for command_name in "${optional[@]}"; do
  if command -v "$command_name" >/dev/null 2>&1; then
    echo "ok: $command_name"
  else
    echo "optional tool not installed yet: $command_name"
  fi
done

python3 scripts/sync-agent-skills.py --check
echo "Bootstrap verification complete."
