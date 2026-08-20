#!/usr/bin/env bash
set -euo pipefail

require_regular_file() {
  local ruleset_file="$1"
  if [[ ! -f "$ruleset_file" || -L "$ruleset_file" ]]; then
    echo "agent release ruleset must be a regular JSON file" >&2
    return 1
  fi
}

main() {
  local creation_ruleset="${1:-}"
  local immutable_ruleset="${2:-}"
  local creator_actor_id="${3:-}"
  require_regular_file "$creation_ruleset"
  require_regular_file "$immutable_ruleset"
  if [[ ! "$creator_actor_id" =~ ^[1-9][0-9]*$ ]]; then
    echo "agent release creator must be a canonical actor ID" >&2
    return 1
  fi

  if ! jq -e --argjson creator_actor_id "$creator_actor_id" '
    .target == "tag" and
    .enforcement == "active" and
    (.conditions.ref_name.include == ["refs/tags/agent-v*"]) and
    (.conditions.ref_name.exclude == []) and
    ([.rules[].type] == ["creation"]) and
    (.bypass_actors == [{
      "actor_id": $creator_actor_id,
      "actor_type": "Integration",
      "bypass_mode": "always"
    }])
  ' "$creation_ruleset" >/dev/null; then
    echo "agent-v* creation requires exactly one approved GitHub App bypass" >&2
    return 1
  fi

  if ! jq -e '
    .target == "tag" and
    .enforcement == "active" and
    (.conditions.ref_name.include == ["refs/tags/agent-v*"]) and
    (.conditions.ref_name.exclude == []) and
    ([.rules[].type] | sort == ["deletion", "update"]) and
    (.bypass_actors == [])
  ' "$immutable_ruleset" >/dev/null; then
    echo "agent-v* immutability requires update/delete restrictions with no bypass" >&2
    return 1
  fi
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  main "$@"
fi
