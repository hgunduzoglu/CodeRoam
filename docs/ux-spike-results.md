# Touch UX Spike Results

Status: **complete — Go**

## Automated preparation

As of 2026-07-15, the typed Flutter-WebView bridge stabilization and touch-test harnesses are
implemented. Automated coverage includes bridge decoding and rejection, bounded pre-ready
queueing, ordered single flushing, reload readiness reset, controller disposal, bounded editor
indentation, a 10,000-line fixture, mock diagnostics and code actions, lazy terminal loading,
full-screen terminal mode, terminal input routing and recent-event deduplication, the Ctrl
modifier, the developer key row, bounded fast output, the isolated local echo harness, terminal
touch coordinate and selection-range mapping, bounded terminal copy payloads, and short-landscape
tablet navigation. Terminal selection coverage also verifies handle-edge positioning, compact
toolbar placement above or below edge-adjacent selections, and bounded clipboard text. Input
deduplication rejects malformed event IDs and delayed events or readiness from retired page streams
while keeping its remembered state bounded, and clipboard responses are discarded after their
originating WebView stream retires.

The editor restores the last document supplied by Flutter after a WebView reload. This recovery is
a spike fallback, not persistence: edits that existed only inside a crashed WebView are not
recoverable by this Milestone 0 harness.

The 2026-07-15 validation run recorded for this spike passed `make fmt`, `make lint`, and
`make test`. Rerun those commands after subsequent changes. This is implementation evidence only
and does not replace the physical-device checks below.

| Device | OS | Editor | Terminal | Keyboard | Pointer | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| iPhone | iOS 26.5.2 | Pass | Pass | software | n/a | All scripted checks passed, including the final compact selection-menu and short-landscape retests. |
| iPad | iPadOS (version not recorded) | Pass | Pass | software + hardware | yes | All scripted checks passed, including hardware keyboard, pointer, orientation, and split-screen behavior. |
| Android phone | Android (version not recorded) | Pass | Pass | software | n/a | All scripted checks passed. |
| Android tablet | Android (version not recorded) | Pass | Pass | software + hardware | yes | All scripted checks passed, including hardware keyboard, pointer, orientation, and split-screen behavior. |

## Resolved physical-device findings

- **iPhone terminal selection:** xterm.js renders terminal text to a canvas, so native WebView text
  selection was unavailable. Physical retest confirmed word selection but exposed missing
  draggable handles and Paste, plus selection controls that remained after Copy. The terminal now
  provides draggable start/end handles and Copy/Paste/close controls; Copy and Paste dismiss the
  controls immediately. Paste uses bounded native clipboard data and ignores stale WebView
  responses. Copy and Paste passed the next iPhone retest, which exposed an oversized toolbar. The
  toolbar is now a compact contextual pill anchored above or below the selection with a right-side
  close action. The final physical-device retest passed.
- **Short landscape navigation:** Opening the software keyboard left too little vertical room for
  the tablet navigation rail and produced a bottom-overflow warning. The rail is now scrollable.
  The physical-device retest passed.
- **iPhone completion contrast:** The explicit dark completion theme passed physical retest on
  2026-07-15.
- **iPhone indentation:** The bounded indentation behavior passed physical retest on 2026-07-15.

## Decision

**Go.** The CodeMirror and xterm.js surfaces satisfy the Milestone 0 touch acceptance criteria on
the required iPhone, iPad, Android phone, and Android tablet device classes. Hardware keyboard,
pointer, orientation, split-screen, keyboard viewport, selection, copy/paste, fast-output, and
focus-transition checks passed. The exact iPadOS and Android version numbers were not recorded;
that documentation gap does not block the Milestone 0 product decision.
