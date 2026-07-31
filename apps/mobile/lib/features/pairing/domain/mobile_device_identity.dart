import 'dart:typed_data';

import 'package:cryptography/cryptography.dart';

const _x25519PublicKeyLength = 32;
const _x25519FingerprintPrefix = 'x25519-sha256:';

final class MobileDeviceIdentity {
  MobileDeviceIdentity._({
    required Uint8List publicKey,
    required this.fingerprint,
  }) : _publicKey = Uint8List.fromList(publicKey);

  final Uint8List _publicKey;
  final String fingerprint;

  Uint8List get publicKey => Uint8List.fromList(_publicKey);

  static Future<MobileDeviceIdentity> fromKeyPair(SimpleKeyPair keyPair) async {
    final publicKey = await keyPair.extractPublicKey();
    if (publicKey.type != KeyPairType.x25519 ||
        publicKey.bytes.length != _x25519PublicKeyLength) {
      throw const FormatException('Mobile identity is invalid.');
    }
    final publicKeyBytes = Uint8List.fromList(publicKey.bytes);
    return MobileDeviceIdentity._(
      publicKey: publicKeyBytes,
      fingerprint: await fingerprintX25519PublicKey(publicKeyBytes),
    );
  }
}

Future<String> fingerprintX25519PublicKey(List<int> publicKey) async {
  if (publicKey.length != _x25519PublicKeyLength ||
      publicKey.any((value) => value < 0 || value > 255)) {
    throw const FormatException('X25519 public key is invalid.');
  }
  final digest = await Sha256().hash(publicKey);
  final encoded = StringBuffer(_x25519FingerprintPrefix);
  for (final value in digest.bytes) {
    encoded.write(value.toRadixString(16).padLeft(2, '0'));
  }
  return encoded.toString();
}
