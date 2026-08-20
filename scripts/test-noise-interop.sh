#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
interop_root="$repo_root/tests/noise-interop"
rust_manifest="$interop_root/rust-peer/Cargo.toml"
rust_peer="$interop_root/rust-peer/target/debug/coderoam-noise-rust-peer"

command -v cargo >/dev/null 2>&1 || {
  echo "cargo is required for the Noise interoperability test" >&2
  exit 1
}
command -v go >/dev/null 2>&1 || {
  echo "go is required for the Noise interoperability test" >&2
  exit 1
}

cargo build --locked --manifest-path "$rust_manifest"
(
  cd "$interop_root"
  GOWORK=off CODEROAM_NOISE_RUST_PEER="$rust_peer" go test ./...
)
