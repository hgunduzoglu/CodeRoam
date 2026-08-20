#!/usr/bin/env bash
set -euo pipefail

agent_release_staging_dir=""

cleanup_staging_directory() {
  if [[ -n "$agent_release_staging_dir" ]]; then
    rm -rf -- "$agent_release_staging_dir"
  fi
}

validate_release_tag() {
  local release_tag="$1"
  if [[ ! "$release_tag" =~ ^agent-v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]; then
    echo "agent release tag must match agent-vX.Y.Z" >&2
    return 1
  fi
}

require_clean_repository() {
  local repository_root="$1"
  if [[ "${AGENT_RELEASE_REQUIRE_CLEAN:-false}" != "true" ]]; then
    return
  fi

  local status
  status="$(git -C "$repository_root" status --porcelain=v1 --untracked-files=all)"
  if [[ -n "$status" ]]; then
    echo "agent release requires a clean repository, including untracked files" >&2
    return 1
  fi
}

write_checksums() {
  local directory="$1"
  shift
  if command -v sha256sum >/dev/null 2>&1; then
    (cd "$directory" && sha256sum -- "$@") >"$directory/SHA256SUMS"
    return
  fi
  if command -v shasum >/dev/null 2>&1; then
    (cd "$directory" && shasum -a 256 -- "$@") >"$directory/SHA256SUMS"
    return
  fi
  echo "sha256sum or shasum is required" >&2
  return 1
}

main() {
  local release_tag="${1:-}"
  local output_dir="${2:-dist/agent-release}"
  validate_release_tag "$release_tag"

  local repository_root
  repository_root="$(git rev-parse --show-toplevel)"
  if [[ "$output_dir" != /* ]]; then
    output_dir="$repository_root/$output_dir"
  fi
  if [[ -e "$output_dir" || -L "$output_dir" ]]; then
    echo "agent release output already exists: $output_dir" >&2
    return 1
  fi
  require_clean_repository "$repository_root"

  local output_parent release_version commit staging_dir
  output_parent="$(dirname "$output_dir")"
  release_version="${release_tag#agent-v}"
  commit="$(git -C "$repository_root" rev-parse --verify HEAD)"
  if [[ ! "$commit" =~ ^[0-9a-f]{40}$ ]]; then
    echo "agent release requires a canonical Git commit SHA" >&2
    return 1
  fi
  mkdir -p "$output_parent"
  staging_dir="$(mktemp -d "$output_parent/.agent-release.XXXXXX")"
  agent_release_staging_dir="$staging_dir"
  trap cleanup_staging_directory EXIT

  local architecture artifact_name
  local -a artifact_names=()
  for architecture in amd64 arm64; do
    artifact_name="coderoam-agent_${release_version}_linux_${architecture}"
    artifact_names+=("$artifact_name")
    (
      cd "$repository_root/services/agent"
      CGO_ENABLED=0 GOOS=linux GOARCH="$architecture" go build \
        -trimpath \
        -buildvcs=true \
        -ldflags "-s -w -X main.version=$release_version -X main.commit=$commit" \
        -o "$staging_dir/$artifact_name" \
        ./cmd/coderoam-agent
    )
    chmod 0755 "$staging_dir/$artifact_name"
  done
  write_checksums "$staging_dir" "${artifact_names[@]}"

  mv "$staging_dir" "$output_dir"
  agent_release_staging_dir=""
  trap - EXIT
  printf 'agent release artifacts: %s\n' "$output_dir"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  main "$@"
fi
