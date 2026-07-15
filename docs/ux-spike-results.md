# Touch UX Spike Results

Status: **physical-device validation in progress**

Fill this document during Milestone 0.

## Automated preparation

As of 2026-07-15, the typed Flutter-WebView bridge stabilization and touch-test harnesses are
implemented. Automated coverage includes bridge decoding and rejection, bounded pre-ready
queueing, ordered single flushing, reload readiness reset, controller disposal, bounded editor
indentation, a 10,000-line fixture, mock diagnostics and code actions, lazy terminal loading,
full-screen terminal mode, terminal input routing and recent-event deduplication, the Ctrl
modifier, the developer key row, bounded fast output, the isolated local echo harness, terminal
touch coordinate and selection-range mapping, bounded terminal copy payloads, and short-landscape
tablet navigation. Input deduplication rejects malformed event IDs and delayed events or readiness
from retired page streams while keeping its remembered state bounded.

The editor restores the last document supplied by Flutter after a WebView reload. This recovery is
a spike fallback, not persistence: edits that existed only inside a crashed WebView are not
recoverable by this Milestone 0 harness.

The 2026-07-15 validation run recorded for this spike passed `make fmt`, `make lint`, and
`make test`. Rerun those commands after subsequent changes. This is implementation evidence only
and does not replace the physical-device checks below.

| Device | OS | Editor | Terminal | Keyboard | Pointer | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| iPhone | iOS 26.5.2 | Pass | Partial; selection retest pending | software | n/a | The editor retest passed. Terminal long-press selection did not work, and the navigation rail overflowed in short landscape mode; both fixes await physical retest. |
| iPad | TBD | TBD | TBD | software + hardware | yes | |
| Android phone | TBD | TBD | TBD | software | n/a | |
| Android tablet | TBD | TBD | TBD | software + hardware | yes | |

## Findings awaiting physical retest

- **iPhone terminal selection:** xterm.js renders terminal text to a canvas, so native WebView text
  selection was unavailable. A bounded long-press word selection, drag-to-extend gesture, and
  Copy/Clear touch toolbar now use xterm's selection APIs. Physical-device retest is pending.
- **Short landscape navigation:** Opening the software keyboard left too little vertical room for
  the tablet navigation rail and produced a bottom-overflow warning. The rail is now scrollable.
  Physical-device retest is pending.

## Resolved findings

- **iPhone completion contrast:** The explicit dark completion theme passed physical retest on
  2026-07-15.
- **iPhone indentation:** The bounded indentation behavior passed physical retest on 2026-07-15.

## Decision

Pending.
