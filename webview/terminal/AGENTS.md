# CodeRoam xterm.js Guidance

- Own terminal rendering and typed Flutter bridge, not PTY lifecycle.
- Optimize for touch selection, copy/paste, developer key row, keyboard viewport, pointer, and
  bounded scrollback.
- Input has higher priority than output.
- Test rapid output, resize races, duplicate bridge events, focus, and slow devices.
