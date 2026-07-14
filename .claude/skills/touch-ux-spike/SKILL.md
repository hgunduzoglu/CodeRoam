<!-- GENERATED MIRROR. Edit .agents/skills and run scripts/sync-agent-skills.py. -->
---
name: touch-ux-spike
description: Build or validate CodeRoam's Milestone 0 Flutter + CodeMirror + xterm.js real-device touch spike. Use for selection, IME, keyboard, focus, gesture, scroll, touch-target, and tablet/phone acceptance work.
---

# Touch UX Spike

1. Read `docs/touch-ux-spike.md` and mobile/editor/terminal instructions.
2. Keep the spike independent of control-plane, relay, and agent behavior.
3. Test physical iPhone, iPad, Android phone, and Android tablet.
4. Cover cursor placement, selection handles, copy/paste, IME, keyboard viewport, large-file scroll,
   terminal key row, fast output, focus transitions, orientation, split screen, pointer, and
   hardware keyboard.
5. Record the device matrix and blocking findings in `docs/ux-spike-results.md`.
6. Do not declare success based only on simulator/emulator behavior.
7. Add bridge-level tests for ordering, duplicate messages, and stale responses.

Done means the real-device result is documented and the go/no-go decision is explicit.
