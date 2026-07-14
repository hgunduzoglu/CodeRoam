# CodeRoam CodeMirror Guidance

- Own only the CodeMirror surface and typed Flutter bridge.
- Touch behavior is a first-class acceptance requirement.
- Preserve IME, native selection handles, cursor visibility, undo/redo, large-file performance,
  completion positioning, diagnostics, and keyboard/pointer behavior.
- Render input locally and avoid whole-document transfer per keystroke.
- Test duplicate, delayed, stale, and out-of-order bridge messages.
