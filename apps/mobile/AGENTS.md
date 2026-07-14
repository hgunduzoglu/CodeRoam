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
