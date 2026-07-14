#!/usr/bin/env bash
set -euo pipefail

if ! command -v flutter >/dev/null 2>&1; then
  echo "Flutter is required: https://docs.flutter.dev/get-started/install" >&2
  exit 1
fi

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
mobile="$root/apps/mobile"
backup="$(mktemp -d)"
trap 'rm -rf "$backup"' EXIT

cp "$mobile/pubspec.yaml" "$backup/pubspec.yaml"
cp -R "$mobile/lib" "$backup/lib"
if [[ -d "$mobile/test" ]]; then cp -R "$mobile/test" "$backup/test"; fi
if [[ -d "$mobile/assets" ]]; then cp -R "$mobile/assets" "$backup/assets"; fi

(
  cd "$mobile"
  flutter create \
    --project-name coderoam \
    --org dev.coderoam \
    --platforms android,ios \
    .
)

cp "$backup/pubspec.yaml" "$mobile/pubspec.yaml"
rm -rf "$mobile/lib"
cp -R "$backup/lib" "$mobile/lib"
if [[ -d "$backup/test" ]]; then
  rm -rf "$mobile/test"
  cp -R "$backup/test" "$mobile/test"
fi
if [[ -d "$backup/assets" ]]; then
  rm -rf "$mobile/assets"
  cp -R "$backup/assets" "$mobile/assets"
fi

(
  cd "$mobile"
  flutter pub get
)

echo "Flutter platform files generated."
