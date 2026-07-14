#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

cd "$ROOT"

echo "Building CodeMirror and xterm.js surfaces..."
npm run build:web

echo "Copying editor bundle..."
rm -rf apps/mobile/assets/editor
mkdir -p apps/mobile/assets/editor
cp -R webview/editor/dist/. apps/mobile/assets/editor/

echo "Copying terminal bundle..."
rm -rf apps/mobile/assets/terminal
mkdir -p apps/mobile/assets/terminal
cp -R webview/terminal/dist/. apps/mobile/assets/terminal/

echo "Embedded WebView bundles are ready."
