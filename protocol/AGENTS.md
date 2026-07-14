# CodeRoam Protocol Guidance

- `protocol/proto` is authoritative; `protocol/gen` is generated.
- Never reuse field numbers.
- Pairing uses Noise XXpsk3; normal sessions use Noise IK.
- Relay sees only outer routing metadata and ciphertext.
- Define ordering, ACK, cancellation, deadline, max size, flow control, priority, replay, and resume.
- Protocol changes require compatibility tests and `$change-data-plane-protocol`.
