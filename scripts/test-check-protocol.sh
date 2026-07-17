#!/usr/bin/env bash
set -euo pipefail

source ./scripts/check-protocol.sh

calls_file="$(mktemp)"
generated_changes=""
trap 'rm -f "$calls_file"' EXIT

buf() {
  echo "buf $*" >>"$calls_file"
}

make() {
  echo "make $*" >>"$calls_file"
}

git() {
  echo "git $*" >>"$calls_file"
  if [[ -n "$generated_changes" ]]; then
    echo "$generated_changes"
  fi
}

BUF_BREAKING_AGAINST='.git#branch=test-main' main

expected_calls=$'buf lint\nbuf breaking --against .git#branch=test-main\nmake proto\ngit status --short --untracked-files=all -- protocol/gen'
if [[ "$(<"$calls_file")" != "$expected_calls" ]]; then
  echo "protocol check used unexpected commands" >&2
  exit 1
fi

: >"$calls_file"
generated_changes=' M protocol/gen/go/coderoam/common/v1/envelope.pb.go'
if output="$(main 2>&1)"; then
  echo "protocol check accepted stale generated code" >&2
  exit 1
fi
if [[ "$output" != *"generated protocol code is not current:"* ||
  "$output" != *"protocol/gen/go/coderoam/common/v1/envelope.pb.go"* ]]; then
  echo "protocol check did not report generated drift" >&2
  exit 1
fi
