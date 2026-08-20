#!/usr/bin/env bash
set -euo pipefail

agent_release_test_root=""

cleanup_test_root() {
  if [[ -n "$agent_release_test_root" ]]; then
    rm -rf -- "$agent_release_test_root"
  fi
}

verify_checksums() {
  local directory="$1"
  if command -v sha256sum >/dev/null 2>&1; then
    (cd "$directory" && sha256sum --check SHA256SUMS)
    return
  fi
  (cd "$directory" && shasum -a 256 --check SHA256SUMS)
}

assert_build_metadata() {
  local artifact="$1"
  local architecture="$2"
  local commit="$3"
  local metadata
  metadata="$(go version -m "$artifact")"
  for expected in \
    $'path\tgithub.com/hgunduzoglu/coderoam/services/agent/cmd/coderoam-agent' \
    $'build\tCGO_ENABLED=0' \
    $'build\t'"GOARCH=$architecture" \
    $'build\tGOOS=linux' \
    $'build\t'"vcs.revision=$commit"; do
    if [[ "$metadata" != *"$expected"* ]]; then
      echo "missing build metadata $expected in $artifact" >&2
      return 1
    fi
  done
  if [[ "${AGENT_RELEASE_REQUIRE_CLEAN:-false}" == "true" ]] &&
    [[ "$metadata" != *$'build\tvcs.modified=false'* ]]; then
    echo "release artifact is not marked as an unmodified build: $artifact" >&2
    return 1
  fi
}

assert_release_ref_validation() {
  local repository_root="$1"
  local temporary_root="$2"
  local test_repository="$temporary_root/ref-validation"
  git init --quiet --initial-branch=main "$test_repository"
  git -C "$test_repository" config user.email release-test@coderoam.invalid
  git -C "$test_repository" config user.name "CodeRoam release test"

  printf 'main\n' >"$test_repository/input.go"
  git -C "$test_repository" add input.go
  git -C "$test_repository" commit --quiet -m main
  local main_commit
  main_commit="$(git -C "$test_repository" rev-parse HEAD)"
  git -C "$test_repository" update-ref refs/remotes/origin/main "$main_commit"
  local resolved_commit
  resolved_commit="$("$repository_root/scripts/validate-agent-release-ref.sh" \
    agent-v0.0.0 "$main_commit" refs/remotes/origin/main "$test_repository")"
  if [[ "$resolved_commit" != "$main_commit" ]]; then
    echo "main release source resolved to $resolved_commit, want $main_commit" >&2
    return 1
  fi

  git -C "$test_repository" switch --quiet --create feature
  printf 'branch only\n' >"$test_repository/input.go"
  git -C "$test_repository" commit --quiet --all -m feature
  if "$repository_root/scripts/validate-agent-release-ref.sh" \
    agent-v9.9.9 "$(git -C "$test_repository" rev-parse HEAD)" \
    refs/remotes/origin/main "$test_repository" >/dev/null 2>&1; then
    echo "release validator accepted a branch-only source commit" >&2
    return 1
  fi

  git -C "$test_repository" tag agent-v0.0.0 "$main_commit"
  if "$repository_root/scripts/validate-agent-release-ref.sh" \
    agent-v0.0.0 "$main_commit" refs/remotes/origin/main \
    "$test_repository" >/dev/null 2>&1; then
    echo "release validator accepted an existing tag" >&2
    return 1
  fi

  printf 'package injected\n' >"$test_repository/untracked.go"
  if AGENT_RELEASE_REQUIRE_CLEAN=true bash -c \
    'source "$1"; require_clean_repository "$2"' \
    _ "$repository_root/scripts/build-agent-release.sh" "$test_repository" \
    >/dev/null 2>&1; then
    echo "release builder accepted an untracked Go input" >&2
    return 1
  fi
}

assert_release_ruleset_validation() {
  local repository_root="$1"
  local temporary_root="$2"
  local creation_ruleset="$temporary_root/creation-ruleset.json"
  local immutable_ruleset="$temporary_root/immutable-ruleset.json"
  local invalid_ruleset="$temporary_root/invalid-ruleset.json"
  printf '%s\n' \
    '{"target":"tag","enforcement":"active","conditions":{"ref_name":{"include":["refs/tags/agent-v*"],"exclude":[]}},"rules":[{"type":"creation"}],"bypass_actors":[{"actor_id":15368,"actor_type":"Integration","bypass_mode":"always"}]}' \
    >"$creation_ruleset"
  printf '%s\n' \
    '{"target":"tag","enforcement":"active","conditions":{"ref_name":{"include":["refs/tags/agent-v*"],"exclude":[]}},"rules":[{"type":"update"},{"type":"deletion"}],"bypass_actors":[]}' \
    >"$immutable_ruleset"
  "$repository_root/scripts/validate-agent-release-ruleset.sh" \
    "$creation_ruleset" "$immutable_ruleset" 15368

  printf '%s\n' \
    '{"target":"tag","enforcement":"active","conditions":{"ref_name":{"include":["refs/tags/agent-v*"],"exclude":[]}},"rules":[{"type":"update"},{"type":"deletion"}],"bypass_actors":[{"actor_id":5,"actor_type":"RepositoryRole","bypass_mode":"always"}]}' \
    >"$invalid_ruleset"
  if "$repository_root/scripts/validate-agent-release-ruleset.sh" \
    "$creation_ruleset" "$invalid_ruleset" 15368 >/dev/null 2>&1; then
    echo "release ruleset validator accepted an immutability bypass" >&2
    return 1
  fi

  printf '%s\n' \
    '{"target":"tag","enforcement":"active","conditions":{"ref_name":{"include":["refs/tags/agent-v*"],"exclude":[]}},"rules":[{"type":"creation"}],"bypass_actors":[{"actor_id":15368,"actor_type":"Integration","bypass_mode":"always"},{"actor_id":5,"actor_type":"RepositoryRole","bypass_mode":"always"}]}' \
    >"$invalid_ruleset"
  if "$repository_root/scripts/validate-agent-release-ruleset.sh" \
    "$invalid_ruleset" "$immutable_ruleset" 15368 >/dev/null 2>&1; then
    echo "release ruleset validator accepted an additional creation bypass" >&2
    return 1
  fi

  printf '%s\n' \
    '{"target":"tag","enforcement":"active","conditions":{"ref_name":{"include":["refs/tags/agent-v*"],"exclude":[]}},"rules":[{"type":"creation"}],"bypass_actors":[{"actor_id":5,"actor_type":"RepositoryRole","bypass_mode":"always"}]}' \
    >"$invalid_ruleset"
  if "$repository_root/scripts/validate-agent-release-ruleset.sh" \
    "$invalid_ruleset" "$immutable_ruleset" 5 >/dev/null 2>&1; then
    echo "release ruleset validator accepted a broad repository-role creator" >&2
    return 1
  fi
}

assert_immutable_action_pins() {
  local workflow="$1"
  local uses_lines action
  uses_lines="$(grep -E '^[[:space:]]+uses:' "$workflow")"
  if [[ -z "$uses_lines" ]]; then
    echo "agent release workflow declares no external actions" >&2
    return 1
  fi
  while IFS= read -r action; do
    if [[ ! "$action" =~ ^[[:space:]]+uses:[[:space:]]+[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+@[0-9a-f]{40}([[:space:]]+#[[:space:]].*)?$ ]]; then
      echo "agent release workflow action is not immutably pinned: $action" >&2
      return 1
    fi
  done <<<"$uses_lines"
}

assert_workflow_trust_boundary() {
  local workflow="$1"
  if ! grep -Fqx \
    "  group: agent-release-\${{ github.event.client_payload.release_tag }}" \
    "$workflow"; then
    echo "agent release workflow does not serialize one release version" >&2
    return 1
  fi
  if ! grep -Eq '^[[:space:]]+repository_dispatch:' "$workflow"; then
    echo "agent release workflow is not sourced through repository_dispatch" >&2
    return 1
  fi
  if grep -Eq '^[[:space:]]+(push|workflow_dispatch|pull_request_target):' "$workflow"; then
    echo "agent release workflow permits caller-selected or untrusted workflow content" >&2
    return 1
  fi
  for permission in 'contents: write' 'id-token: write' 'attestations: write'; do
    if [[ "$(grep -Fc "$permission" "$workflow")" != "1" ]]; then
      echo "agent release permission must occur only in the publish job: $permission" >&2
      return 1
    fi
  done
  for digest_flag in --source-digest --signer-digest; do
    if ! grep -Fq -- "$digest_flag" "$workflow"; then
      echo "agent release provenance omits $digest_flag" >&2
      return 1
    fi
  done
}

main() {
  local repository_root temporary_root output_dir commit
  repository_root="$(git rev-parse --show-toplevel)"
  temporary_root="$(mktemp -d "${TMPDIR:-/tmp}/coderoam-agent-release.XXXXXX")"
  agent_release_test_root="$temporary_root"
  trap cleanup_test_root EXIT
  output_dir="$temporary_root/release"
  commit="$(git -C "$repository_root" rev-parse --verify HEAD)"

  assert_release_ref_validation "$repository_root" "$temporary_root"
  assert_release_ruleset_validation "$repository_root" "$temporary_root"

  if "$repository_root/scripts/build-agent-release.sh" \
    v0.0.0 "$temporary_root/invalid" >/dev/null 2>&1; then
    echo "agent release builder accepted an invalid tag" >&2
    return 1
  fi
  ln -s "$temporary_root/missing" "$temporary_root/dangling-output"
  if "$repository_root/scripts/build-agent-release.sh" \
    agent-v0.0.0 "$temporary_root/dangling-output" >/dev/null 2>&1; then
    echo "agent release builder replaced a dangling output symlink" >&2
    return 1
  fi
  "$repository_root/scripts/build-agent-release.sh" agent-v0.0.0 "$output_dir"
  if "$repository_root/scripts/build-agent-release.sh" \
    agent-v0.0.0 "$output_dir" >/dev/null 2>&1; then
    echo "agent release builder overwrote an existing output" >&2
    return 1
  fi

  local architecture artifact
  for architecture in amd64 arm64; do
    artifact="$output_dir/coderoam-agent_0.0.0_linux_${architecture}"
    if [[ ! -x "$artifact" ]]; then
      echo "missing executable release artifact: $artifact" >&2
      return 1
    fi
    assert_build_metadata "$artifact" "$architecture" "$commit"
  done
  if [[ "$(find "$output_dir" -maxdepth 1 -type f | wc -l | tr -d ' ')" != "3" ]]; then
    echo "agent release output contains unexpected files" >&2
    return 1
  fi
  verify_checksums "$output_dir"
  assert_immutable_action_pins "$repository_root/.github/workflows/release-agent.yml"
  assert_workflow_trust_boundary "$repository_root/.github/workflows/release-agent.yml"

  if [[ "$(uname -s)" == "Linux" && "$(uname -m)" == "x86_64" ]]; then
    if [[ "$(id -u)" == "0" ]]; then
      echo "Linux agent release smoke install must run as non-root" >&2
      return 1
    fi
    local install_dir version_output
    install_dir="$temporary_root/install/bin"
    install -d -m 0700 "$install_dir"
    install -m 0755 "$output_dir/coderoam-agent_0.0.0_linux_amd64" \
      "$install_dir/coderoam-agent"
    version_output="$("$install_dir/coderoam-agent" version)"
    if [[ "$version_output" != "coderoam-agent 0.0.0 ($commit)" ]]; then
      echo "installed agent version = $version_output" >&2
      return 1
    fi
  fi
}

main "$@"
