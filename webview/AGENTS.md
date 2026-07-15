# Embedded WebView Surface Rules

These instructions apply to files under `webview`.

## Architecture

- CodeMirror remains the editor implementation.
- xterm.js remains the terminal emulator.
- Keep editor and terminal implementations separate when their semantics differ.
- Communicate with Flutter only through the versioned JSON bridge.
- Do not manually interpolate untrusted values into JavaScript source strings.
- Validate all inbound bridge messages before use.
- Do not use `any` unless unavoidable and localized.
- Do not emit full document contents through routine events or logs.
- Do not implement terminal local echo in xterm.js. Terminal input must be emitted to Flutter and output must return through the terminal output path.
- Disconnect `ResizeObserver` and other owned browser resources when their lifecycle ends.
- Avoid resize feedback loops.
- Preserve relative local asset paths.
- Preserve the existing classic IIFE WebView bundle format unless a verified replacement supports Flutter local assets on all target platforms.

## Generated Flutter Assets

The contents of:

- `apps/mobile/assets/editor`
- `apps/mobile/assets/terminal`

are generated from the WebView source packages.

Do not manually edit generated asset bundles.

Use the canonical root command:

```sh
npm run sync:webviews
```
