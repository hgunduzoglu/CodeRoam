# CodeRoam Mobile Guidance

- Flutter owns touch-first navigation, responsive modes, device keys, encrypted Offline Drafts,
  approvals, and editor/terminal orchestration.
- Milestone 0 real-device behavior is a product gate.
- CodeMirror and xterm remain isolated WebView surfaces with typed, versioned bridges.
- High-frequency phone actions stay thumb-reachable.
- Support software keyboard, physical keyboard, pointer, orientation, and split screen.
- Offline support is selected text-file drafts with base-hash conflicts only.
- Do not implement backend authorization in Flutter.
- Do not log paths, file contents, terminal streams, prompts, or secrets.

# Flutter Mobile Rules

These instructions apply to files under `apps/mobile`.

## Architecture

- Keep Flutter UI, WebView transport, input policy, and temporary spike harnesses separate.
- Keep native WebView creation behind replaceable widget boundaries so unit and widget tests do not require native platform views.
- Preserve widget and controller lifecycle ownership explicitly.
- Dispose controllers, subscriptions, observers, focus nodes, and other owned resources where required.
- Do not move business or protocol logic into widget build methods.
- Avoid unnecessary rebuilds, but do not introduce premature state-management abstractions.
- Preserve Dart null safety.
- Do not use unchecked dynamic payloads across the Flutter-WebView bridge.
- Validate bridge protocol versions, message types, and payloads before use.

## Formatting and Validation

After relevant mobile changes, run:

```sh
cd apps/mobile
dart format lib test
flutter analyze
flutter test
```
