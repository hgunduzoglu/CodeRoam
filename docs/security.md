# CodeRoam Security Model

## Trust boundaries

- The mobile device protects its private identity key in Keychain/Keystore.
- The workspace machine protects the agent private identity under the remote OS user.
- The relay and control plane are not trusted with application plaintext.
- PostgreSQL and Redis are not engineering-payload stores.
- The remote workspace itself is trusted only to the extent granted by its owner.

## Pairing

Initial pairing uses a QR/manual high-entropy secret and
`Noise_XXpsk3_25519_ChaChaPoly_BLAKE2s` over the relay.

Normal sessions use pinned identities and
`Noise_IK_25519_ChaChaPoly_BLAKE2s`.

Pairing codes are:

- high entropy,
- short lived,
- one use,
- attempt limited,
- rate limited,
- bound to a protocol version and agent identity.

## Relay security

The relay validates signed tickets and forwards opaque encrypted frames. It must not log payloads,
persist frame bodies, or expose one endpoint to another without matching authorized tickets.

## Agent security

- Non-root execution.
- Signed release verification.
- Project-root filesystem confinement.
- Structured subprocess execution.
- Sanitized environment.
- Bounded output and lifetime.
- Process-group cancellation.
- No secret or source logging.
- AI permissions narrower than interactive terminal permissions.

## Sensitive actions

Device/agent revocation, preview invitation, and runbook actions are checked by the owning
application module. Production-sensitive runbooks require explicit reason and biometric
confirmation. AI cannot invoke them.
