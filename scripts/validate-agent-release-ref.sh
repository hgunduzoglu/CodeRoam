#!/usr/bin/env bash
set -euo pipefail

validate_release_tag() {
  local release_tag="$1"
  if [[ ! "$release_tag" =~ ^agent-v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]; then
    echo "agent release tag must match agent-vX.Y.Z" >&2
    return 1
  fi
}

resolve_release_commit() {
  local repository="$1"
  local source_commit="$2"
  if [[ ! "$source_commit" =~ ^[0-9a-f]{40}$ ]]; then
    echo "agent release source must be a canonical Git commit SHA" >&2
    return 1
  fi
  local release_commit
  release_commit="$(git -C "$repository" rev-parse --verify "${source_commit}^{commit}")"
  if [[ ! "$release_commit" =~ ^[0-9a-f]{40}$ ]]; then
    echo "agent release source does not resolve to a canonical Git commit" >&2
    return 1
  fi
  printf '%s\n' "$release_commit"
}

main() {
  local release_tag="${1:-}"
  local source_commit="${2:-}"
  local main_ref="${3:-refs/remotes/origin/main}"
  local repository="${4:-.}"
  validate_release_tag "$release_tag"
  if git -C "$repository" show-ref --verify --quiet "refs/tags/$release_tag"; then
    echo "agent release tag already exists" >&2
    return 1
  fi

  local release_commit
  release_commit="$(resolve_release_commit "$repository" "$source_commit")"
  if ! git -C "$repository" merge-base --is-ancestor "$release_commit" "$main_ref"; then
    echo "agent release source must be a commit already on main" >&2
    return 1
  fi
  printf '%s\n' "$release_commit"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  main "$@"
fi
