import 'dart:convert';

import 'package:coderoam/features/pairing/domain/mobile_device_identity.dart';
import 'package:coderoam/features/pairing/infrastructure/secure_mobile_identity_provider.dart';
import 'package:cryptography/cryptography.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('creates once and restores the exact same identity', () async {
    final storage = _MemoryIdentityStorage();
    final provider = SecureMobileIdentityProvider(storage: storage);

    final created = await provider.create();
    final loaded = await provider.load();

    expect(loaded.publicKey, created.publicKey);
    expect(loaded.fingerprint, created.fingerprint);
    expect(storage.writes, 1);
    await expectLater(
      provider.create(),
      _failsWith(MobileIdentityFailure.alreadyExists),
    );
  });

  test(
    'atomically arbitrates concurrent creation across provider instances',
    () async {
      final storage = _MemoryIdentityStorage();
      final firstProvider = SecureMobileIdentityProvider(storage: storage);
      final secondProvider = SecureMobileIdentityProvider(storage: storage);

      final results = await Future.wait<Object>([
        firstProvider.create().then<Object>(
          (identity) => identity,
          onError: (Object error) => error,
        ),
        secondProvider.create().then<Object>(
          (identity) => identity,
          onError: (Object error) => error,
        ),
      ]);

      final identities = results.whereType<MobileDeviceIdentity>().toList();
      final failures = results.whereType<MobileIdentityException>().toList();
      expect(identities, hasLength(1));
      expect(failures, hasLength(1));
      expect(failures.single.failure, MobileIdentityFailure.alreadyExists);
      expect(storage.writes, 1);
      final restored = await firstProvider.load();
      expect(restored.publicKey, identities.single.publicKey);
    },
  );

  test('missing identity fails closed without creating a record', () async {
    final storage = _MemoryIdentityStorage();
    final provider = SecureMobileIdentityProvider(storage: storage);

    await expectLater(
      provider.load(),
      _failsWith(MobileIdentityFailure.notFound),
    );
    expect(storage.value, isNull);
    expect(storage.writes, 0);
  });

  test('rejects and preserves malformed or mismatched records', () async {
    final validRecord = await _identityRecord(seedOffset: 1);
    final mismatchedRecord = await _identityRecord(
      seedOffset: 1,
      publicSeedOffset: 2,
    );
    final decoded = jsonDecode(validRecord) as Map<String, Object?>;
    final paddedPrivateKey = '${decoded['privateKey']}=';
    final records = <String>[
      '{',
      '{"version":1,"version":1,"privateKey":"a","publicKey":"b"}',
      ' $validRecord',
      jsonEncode({
        'version': 2,
        'privateKey': decoded['privateKey'],
        'publicKey': decoded['publicKey'],
      }),
      jsonEncode({
        'version': 1,
        'privateKey': decoded['privateKey'],
        'publicKey': decoded['publicKey'],
        'extra': true,
      }),
      jsonEncode({
        'version': 1,
        'privateKey': paddedPrivateKey,
        'publicKey': decoded['publicKey'],
      }),
      mismatchedRecord,
      'a' * 1025,
      '⛔' * 1025,
    ];

    for (final record in records) {
      final storage = _MemoryIdentityStorage(value: record);
      final provider = SecureMobileIdentityProvider(storage: storage);

      await expectLater(
        provider.load(),
        _failsWith(MobileIdentityFailure.invalid),
      );
      expect(storage.value, record);
      expect(storage.writes, 0);
      await expectLater(
        provider.create(),
        _failsWith(MobileIdentityFailure.alreadyExists),
      );
    }
  });

  test(
    'preserves an ambiguously committed identity for reconciliation',
    () async {
      final storage = _MemoryIdentityStorage(throwAfterWrite: true);
      final provider = SecureMobileIdentityProvider(storage: storage);

      await expectLater(
        provider.create(),
        _failsWith(MobileIdentityFailure.unavailable),
      );
      expect(storage.value, isNotNull);

      final restored = await provider.load();
      await expectLater(
        provider.create(),
        _failsWith(MobileIdentityFailure.alreadyExists),
      );
      expect(restored.publicKey, hasLength(32));
    },
  );

  test('maps storage read failures without modifying an identity', () async {
    final storage = _MemoryIdentityStorage(throwOnRead: true);
    final provider = SecureMobileIdentityProvider(storage: storage);

    await expectLater(
      provider.load(),
      _failsWith(MobileIdentityFailure.unavailable),
    );
    expect(storage.writes, 0);
  });

  test('preserves a committed identity when verification read fails', () async {
    final storage = _MemoryIdentityStorage(throwOnRead: true);
    final provider = SecureMobileIdentityProvider(storage: storage);

    await expectLater(
      provider.create(),
      _failsWith(MobileIdentityFailure.unavailable),
    );
    expect(storage.value, isNotNull);
    expect(storage.writes, 1);

    storage.throwOnRead = false;
    final restored = await provider.load();
    expect(restored.publicKey, hasLength(32));
  });

  test(
    'maps pre-commit storage failures without publishing an identity',
    () async {
      final storage = _MemoryIdentityStorage(throwBeforeCreate: true);
      final provider = SecureMobileIdentityProvider(storage: storage);

      await expectLater(
        provider.create(),
        _failsWith(MobileIdentityFailure.unavailable),
      );
      expect(storage.value, isNull);
      expect(storage.writes, 0);
    },
  );
}

Matcher _failsWith(MobileIdentityFailure failure) => throwsA(
  isA<MobileIdentityException>().having(
    (error) => error.failure,
    'failure',
    failure,
  ),
);

Future<String> _identityRecord({
  required int seedOffset,
  int? publicSeedOffset,
}) async {
  final algorithm = X25519();
  final privatePair = await algorithm.newKeyPairFromSeed(
    List<int>.generate(32, (index) => index + seedOffset),
  );
  final publicPair =
      publicSeedOffset == null
          ? privatePair
          : await algorithm.newKeyPairFromSeed(
            List<int>.generate(32, (index) => index + publicSeedOffset),
          );
  final privateKey = await privatePair.extractPrivateKeyBytes();
  final publicKey = await publicPair.extractPublicKey();
  final record = jsonEncode({
    'version': 1,
    'privateKey': _rawBase64(privateKey),
    'publicKey': _rawBase64(publicKey.bytes),
  });
  privatePair.destroy();
  if (!identical(publicPair, privatePair)) {
    publicPair.destroy();
  }
  return record;
}

String _rawBase64(List<int> value) => base64.encode(value).replaceAll('=', '');

final class _MemoryIdentityStorage implements MobileIdentityStorage {
  _MemoryIdentityStorage({
    this.value,
    this.throwOnRead = false,
    this.throwBeforeCreate = false,
    this.throwAfterWrite = false,
  });

  String? value;
  bool throwOnRead;
  bool throwBeforeCreate;
  bool throwAfterWrite;
  int writes = 0;

  @override
  Future<String?> read() async {
    if (throwOnRead) {
      throw StateError('read unavailable');
    }
    return value;
  }

  @override
  Future<bool> createIfAbsent(String value) async {
    if (throwBeforeCreate) {
      throw StateError('create unavailable');
    }
    if (this.value != null) {
      return false;
    }
    writes++;
    this.value = value;
    if (throwAfterWrite) {
      throw StateError('write result unavailable');
    }
    return true;
  }
}
