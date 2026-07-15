# Touch UX Spike Results

Status: **physical-device validation not run**

Fill this document during Milestone 0.

## Automated preparation

As of 2026-07-15, the typed Flutter-WebView bridge stabilization and minimal terminal input spike
are implemented. Automated coverage includes bridge decoding and rejection, bounded pre-ready
queueing, ordered single flushing, reload readiness reset, controller disposal, terminal input
routing, the Ctrl modifier, the developer key row, and the isolated local echo harness.

The 2026-07-15 validation run recorded for this spike passed `make fmt`, `make lint`, and
`make test`. Rerun those commands after subsequent changes. This is implementation evidence only
and does not replace the physical-device checks below.

| Device | OS | Editor | Terminal | Keyboard | Pointer | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| iPhone | TBD | TBD | TBD | software | n/a | |
| iPad | TBD | TBD | TBD | software + hardware | yes | |
| Android phone | TBD | TBD | TBD | software | n/a | |
| Android tablet | TBD | TBD | TBD | software + hardware | yes | |

## Blocking findings

None recorded yet.

## Decision

Pending.
