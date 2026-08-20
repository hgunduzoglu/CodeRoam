import 'package:coderoam/features/pairing/domain/mobile_device_identity.dart';
import 'package:cryptography/cryptography.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('matches the shared canonical X25519 fingerprint vector', () async {
    final publicKey = List<int>.generate(32, (index) => index);

    expect(
      await fingerprintX25519PublicKey(publicKey),
      'x25519-sha256:'
      '630dcd2966c4336691125448bbb25b4ff412a49c732db2c8abc1b8581bd710dd',
    );
  });

  test('rejects malformed public keys and protects returned bytes', () async {
    await expectLater(
      fingerprintX25519PublicKey(List<int>.filled(31, 0)),
      throwsFormatException,
    );
    await expectLater(
      fingerprintX25519PublicKey(<int>[...List<int>.filled(31, 0), 256]),
      throwsFormatException,
    );

    final keyPair = await X25519().newKeyPairFromSeed(
      List<int>.generate(32, (index) => index + 1),
    );
    final identity = await MobileDeviceIdentity.fromKeyPair(keyPair);
    final firstRead = identity.publicKey;
    firstRead[0] ^= 0xff;

    expect(identity.publicKey, isNot(firstRead));
    keyPair.destroy();
  });
}
