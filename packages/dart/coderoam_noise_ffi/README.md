# CodeRoam Noise FFI

Internal `package_ffi` prototype for the approved `snow` native core.

The current ABI is deliberately non-secret-bearing: it only proves that the exact XXpsk3 suite can
be compiled, bundled as a code asset, resolved through Dart `@Native`, and invoked successfully.
It does not accept pairing secrets or identity keys and is not connected to the mobile app.

The build hook currently supports macOS and Linux hosts. Cargo runs in an isolated POSIX process
group so timeout cleanup terminates its compiler and linker descendants before the hook fails.
iOS/Android targets, opaque handshake handles, buffer ownership, zeroization, cancellation, and
app integration belong to later reviewed M3 slices.
