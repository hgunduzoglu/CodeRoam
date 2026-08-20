# M3 Noise XXpsk3 Interoperability Spike

Date: 2026-07-24
Updated: 2026-07-25

## Purpose

Evaluate a production candidate for the M3 mobile/agent pairing handshake before adding any runtime
dependency. The candidate must implement the exact approved suite:

```text
Noise_XXpsk3_25519_ChaChaPoly_BLAKE2s
```

This spike report supplied the evidence for the approval recorded below. It adds no dependency or
code to the mobile or agent runtime.

## Acceptance criteria

The disposable experiment must prove that:

1. independent Go and native-core implementations accept the exact protocol name;
2. the Rust initiator and Go responder exchange all three XX messages;
3. both sides recover the expected authenticated payloads;
4. both sides recover a 32-byte peer static public key;
5. both sides produce the same 32-byte handshake hash for channel binding;
6. a wrong PSK or different prologue fails authentication; and
7. no production dependency or generated artifact enters the repository.

## Environment and candidates

- macOS arm64
- Go 1.26.0
- Rust 1.96.0
- `github.com/flynn/noise` v1.1.0
- `snow` v0.10.0
- `noise_protocol_framework` v1.2.0, evaluated but not used in the successful transcript

Primary references:

- [Noise Protocol Framework revision 34](https://noiseprotocol.org/noise.pdf)
- [`flynn/noise` package documentation](https://pkg.go.dev/github.com/flynn/noise)
- [`snow` documentation](https://docs.rs/snow/0.10.0/snow/)
- [Flutter FFI package guidance](https://docs.flutter.dev/platform-integration/bind-native-code)
- [`noise_protocol_framework` package](https://pub.dev/packages/noise_protocol_framework)

## Disposable experiment

The initiator used `snow` and the responder used `flynn/noise`. Both received:

- the exact XXpsk3 protocol name;
- independent static X25519 private keys;
- one identical 32-byte PSK;
- the prologue `coderoam-pairing-v1`; and
- bounded test payloads in each of the three messages.

The experiment used line-delimited hexadecimal only as a local process harness. It did not model
the future relay framing.

Observed result:

```text
status: ok
Rust and Go channel binding: identical 32-byte value
Rust recovered Go static key: 32 bytes
Go recovered Rust static key: 32 bytes
```

Negative runs produced authenticated-decryption failure:

```text
wrong PSK: rejected while processing message 3
different prologue: rejected while processing message 1
```

The successful handshake was repeated after a clean Rust build. The channel-binding bytes changed
with the fresh ephemeral key, as expected, while both parties still agreed exactly within each run.

Validation commands:

```text
go mod tidy
go build
cargo build --locked
node run.mjs
WRONG_PSK=1 node run.mjs
WRONG_PROLOGUE=1 node run.mjs
govulncheck ./...
cargo test --locked
cargo clippy --locked --manifest-path rust-peer/Cargo.toml -- -D warnings
```

`govulncheck` found no reachable vulnerability in the disposable Go program. It also reported 23
known vulnerabilities in required module versions that were not reached by the program. The
`flynn/noise` module currently selects old `golang.org/x/*` versions, so a production adoption must
pin supported transitive versions explicitly and pass the repository-wide vulnerability gate.

The Rust build and Clippy passed. No iOS or Android Rust target was installed in this environment,
so the native-core result is not yet a Flutter/iOS/Android build result.

The committed harness was subsequently validated with `make test-noise-interop` from the repository
root and the remaining commands from `tests/noise-interop`:

```text
cargo clippy --locked --manifest-path rust-peer/Cargo.toml -- -D warnings
GOWORK=off go vet ./...
GOWORK=off CODEROAM_NOISE_RUST_PEER="$PWD/rust-peer/target/debug/coderoam-noise-rust-peer" \
  go test -race ./...
GOWORK=off govulncheck ./...
```

The pinned Go module reported no known reachable vulnerabilities.

## Pure-Dart candidate result

`noise_protocol_framework` v1.2.0 does not satisfy M3:

- its public constructors expose only `KNpsk0` and `NKpsk0`;
- its implemented handshakes are two-message patterns rather than XX;
- no X25519 implementation is present in the published source;
- no ready `XXpsk3` implementation exists in the published source; and
- implementing the missing cryptographic state machine locally would violate CodeRoam's rule
  against inventing cryptography.

The package also has a much smaller adoption and review surface than the native candidate. It is
rejected for M3 rather than extended inside CodeRoam.

## Protocol mismatch and resolution

The pre-approval CodeRoam development specification and initial M3 ExecPlan required a random
128-bit pairing secret. Noise revision 34 defines PSK mode for a 32-byte shared secret and says PSKs
must carry 256 bits of entropy.

Passing the 16-byte product secret directly is rejected by the evaluated implementations. Hashing
or expanding it to 32 bytes would satisfy the API length but would retain only 128 bits of entropy
and therefore would not meet the Noise PSK guidance.

Options:

1. Generate a random 32-byte secret. QR pairing remains straightforward. The equivalent unpadded
   Base32 manual value becomes 52 characters and must be grouped for entry.
2. Expand the existing 16-byte secret with a domain-separated KDF. This preserves the current
   26-character manual value but provides only 128 bits of PSK entropy.
3. Add a password-authenticated key exchange before Noise. This adds a second cryptographic
   protocol and is outside the approved M3 scope.

Option 1 was approved on 2026-07-25. It follows the Noise specification without a CodeRoam-specific
key-stretching construction. The development specification and M3 ExecPlan now require the 256-bit
secret; Protobuf bounds and UI copy will change with their owning implementation slices.

## Dependency direction

Approved for prototyping and test-harness use:

- Go agent: evaluate `github.com/flynn/noise` v1.1.0 with current, explicitly pinned
  `golang.org/x/crypto` and `golang.org/x/sys`; retain only the handshake state and discard
  transport cipher states in M3.
- Mobile: wrap `snow` v0.10.0 behind a narrow Rust C ABI using Flutter's `package_ffi` build hooks.
  The wrapper must own and erase handshake state, expose no private-key bytes to Dart, validate
  exact buffer bounds, and return typed failures.
- Shared tests: keep cross-language golden behavior, wrong-PSK, wrong-prologue, wrong-static-key,
  truncated-message, order, replay, and oversize cases in the repository after approval.

`snow` documents that it has not received a formal audit. Passing interoperability is not a
security audit. Before adoption, the exact feature set, transitive dependency graph, mobile target
builds, license notices, vulnerability results, and C ABI must receive adversarial review.

## Approved direction

On 2026-07-25 the user approved all three evaluated items:

1. use a random 256-bit pairing secret with a grouped 52-character Base32 manual fallback;
2. use `flynn/noise` v1.1.0 for the Go agent, with supported transitive crypto versions pinned; and
3. prototype `snow` v0.10.0 through a narrow Flutter FFI package for iOS and Android.

This slice adds the reproducible cross-language handshake harness at `tests/noise-interop`, exposed
through `make test-noise-interop`, with success and wrong-PSK coverage. The dependency remains
isolated to the test module: production pairing, Flutter FFI, relay, persistence, and UI wiring
remain separate later slices.

## Host FFI probe

The next bounded slice adds `packages/dart/coderoam_noise_ffi`, a Dart code-assets package whose
build hook compiles the same minimal `snow` feature set into a Rust `cdylib`. A non-secret-bearing
ABI probe constructs the approved XXpsk3 initiator and writes its first bounded handshake message;
Dart resolves that symbol through `@Native` and verifies ABI version 1. Cargo runs in an isolated
POSIX process group, and a timeout regression verifies that both a fake Cargo process and its
persistent child are terminated before the hook reports failure.

`make check-noise-ffi` passes on the macOS arm64 host. The mobile app now consumes the internal
package, while `make check-noise-ffi-mobile` compiles and bundles the same Rust core for an unsigned
iOS device build and an Android APK. The mobile package test invokes the bundled ABI on the host.
Opaque state handles, secret/key buffer ownership, zeroization, and runtime invocation on physical
devices remain later slices.
