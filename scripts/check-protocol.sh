#!/usr/bin/env bash
set -euo pipefail

check_generated_protocol() {
  local changes

  changes="$(git status --short --untracked-files=all -- protocol/gen)"
  if [[ -n "$changes" ]]; then
    echo "generated protocol code is not current:" >&2
    echo "$changes" >&2
    return 1
  fi
}

main() {
  local baseline="${BUF_BREAKING_AGAINST:-.git#branch=main}"

  buf lint
  buf breaking --against "$baseline"
  make proto
  check_generated_protocol
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  main "$@"
fi
