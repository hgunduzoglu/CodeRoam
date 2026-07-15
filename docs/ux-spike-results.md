# Touch UX Spike Results

Status: **physical-device validation in progress**

Fill this document during Milestone 0.

## Automated preparation

As of 2026-07-15, the typed Flutter-WebView bridge stabilization and touch-test harnesses are
implemented. Automated coverage includes bridge decoding and rejection, bounded pre-ready
queueing, ordered single flushing, reload readiness reset, controller disposal, bounded editor
indentation, a 10,000-line fixture, mock diagnostics and code actions, lazy terminal loading,
full-screen terminal mode, terminal input routing and recent-event deduplication, the Ctrl
modifier, the developer key row, bounded fast output, and the isolated local echo harness. Input
deduplication rejects malformed event IDs and delayed events or readiness from retired page
streams while keeping its remembered state bounded.

The editor restores the last document supplied by Flutter after a WebView reload. This recovery is
a spike fallback, not persistence: edits that existed only inside a crashed WebView are not
recoverable by this Milestone 0 harness.

The 2026-07-15 validation run recorded for this spike passed `make fmt`, `make lint`, and
`make test`. Rerun those commands after subsequent changes. This is implementation evidence only
and does not replace the physical-device checks below.

| Device | OS | Editor | Terminal | Keyboard | Pointer | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| iPhone | iOS (version TBD) | Partial; retest pending | Not tested | software | n/a | Completion contrast passed after the theme fix. Excessive indentation fix needs retest. |
| iPad | TBD | TBD | TBD | software + hardware | yes | |
| Android phone | TBD | TBD | TBD | software | n/a | |
| Android tablet | TBD | TBD | TBD | software + hardware | yes | |

## Blocking findings

- **iPhone indentation:** Malformed JavaScript could make CodeMirror align a new line far beyond the
  current block indentation. The editor now keeps the current indentation when the parser suggests
  a jump larger than one indent level. Physical-device retest is pending.

## Resolved findings

- **iPhone completion contrast:** The explicit dark completion theme passed physical retest on
  2026-07-15.

## Decision

Pending.
