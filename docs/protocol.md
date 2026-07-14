# CodeRoam Data-Plane Protocol

## Transport

Both client and agent create TLS WebSocket connections to the relay. The application payload is
encrypted again end-to-end through Noise.

## Handshakes

- Initial pairing: `Noise_XXpsk3_25519_ChaChaPoly_BLAKE2s`.
- Normal paired session: `Noise_IK_25519_ChaChaPoly_BLAKE2s`.

## Envelope

The Protobuf envelope identifies protocol version, session, channel, stream, sequence,
acknowledgement watermark, flags, and payload. The payload is encrypted before the relay sees it.

## Required semantics

Every channel defines:

- ordering,
- acknowledgements,
- cancellation,
- deadlines,
- maximum sizes,
- flow control,
- priority,
- reconnect and replay,
- malformed-input behavior.

Generated Go and Dart code under `protocol/gen` is never edited manually.
