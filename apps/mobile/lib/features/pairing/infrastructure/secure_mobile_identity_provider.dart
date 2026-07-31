import 'dart:convert';

import 'package:coderoam/features/pairing/domain/mobile_device_identity.dart';
import 'package:coderoam/shared/control_plane/strict_json.dart';
import 'package:cryptography/cryptography.dart';
import 'package:flutter/services.dart';

enum MobileIdentityFailure { notFound, alreadyExists, invalid, unavailable }

final class MobileIdentityException implements Exception {
  const MobileIdentityException(this.failure);

  final MobileIdentityFailure failure;
}

abstract interface class MobileIdentityProvider {
  Future<MobileDeviceIdentity> load();
  Future<MobileDeviceIdentity> create();
}

abstract interface class MobileIdentityStorage {
  Future<String?> read();
  Future<bool> createIfAbsent(String value);
}

final class NativeMobileIdentityStorage implements MobileIdentityStorage {
  const NativeMobileIdentityStorage({MethodChannel? channel})
    : _channel =
          channel ??
          const MethodChannel('dev.coderoam/mobile_identity_storage_v1');

  final MethodChannel _channel;

  @override
  Future<String?> read() => _channel.invokeMethod<String>('read');

  @override
  Future<bool> createIfAbsent(String value) async {
    final created = await _channel.invokeMethod<bool>('createIfAbsent', {
      'value': value,
    });
    if (created == null) {
      throw PlatformException(
        code: 'identity_storage_unavailable',
        message: 'Identity storage is unavailable.',
      );
    }
    return created;
  }
}

final class SecureMobileIdentityProvider implements MobileIdentityProvider {
  SecureMobileIdentityProvider({
    MobileIdentityStorage? storage,
    X25519? algorithm,
  }) : _storage = storage ?? const NativeMobileIdentityStorage(),
       _algorithm = algorithm ?? X25519();

  static const _storageVersion = 1;
  static const _maximumRecordBytes = 1024;
  static const _keyLength = 32;

  final MobileIdentityStorage _storage;
  final X25519 _algorithm;

  @override
  Future<MobileDeviceIdentity> load() async {
    final encoded = await _readRecord();
    if (encoded == null) {
      throw const MobileIdentityException(MobileIdentityFailure.notFound);
    }
    try {
      return await _decodeIdentity(encoded);
    } on FormatException {
      throw const MobileIdentityException(MobileIdentityFailure.invalid);
    } on ArgumentError {
      throw const MobileIdentityException(MobileIdentityFailure.invalid);
    } on MobileIdentityException {
      rethrow;
    } catch (_) {
      throw const MobileIdentityException(MobileIdentityFailure.unavailable);
    }
  }

  @override
  Future<MobileDeviceIdentity> create() async {
    SimpleKeyPair? keyPair;
    Uint8List? privateKey;
    try {
      keyPair = await _algorithm.newKeyPair();
      final identity = await MobileDeviceIdentity.fromKeyPair(keyPair);
      privateKey = Uint8List.fromList(await keyPair.extractPrivateKeyBytes());
      final encoded = jsonEncode({
        'version': _storageVersion,
        'privateKey': _encodeRawBase64(privateKey),
        'publicKey': _encodeRawBase64(identity.publicKey),
      });
      if (!await _createRecord(encoded)) {
        throw const MobileIdentityException(
          MobileIdentityFailure.alreadyExists,
        );
      }
      if (await _readRecord() != encoded) {
        throw const MobileIdentityException(MobileIdentityFailure.unavailable);
      }
      return identity;
    } on MobileIdentityException {
      rethrow;
    } catch (_) {
      throw const MobileIdentityException(MobileIdentityFailure.unavailable);
    } finally {
      privateKey?.fillRange(0, privateKey.length, 0);
      keyPair?.destroy();
    }
  }

  Future<MobileDeviceIdentity> _decodeIdentity(String encoded) async {
    if (encoded.length > _maximumRecordBytes ||
        utf8.encode(encoded).length > _maximumRecordBytes) {
      throw const FormatException('Mobile identity is invalid.');
    }
    final decoded = decodeStrictJson(encoded);
    if (decoded is! Map<String, Object?> ||
        decoded.length != 3 ||
        decoded['version'] != _storageVersion ||
        decoded['privateKey'] is! String ||
        decoded['publicKey'] is! String) {
      throw const FormatException('Mobile identity is invalid.');
    }
    final canonical = jsonEncode({
      'version': _storageVersion,
      'privateKey': decoded['privateKey'],
      'publicKey': decoded['publicKey'],
    });
    if (canonical != encoded) {
      throw const FormatException('Mobile identity is invalid.');
    }

    Uint8List? privateKey;
    SimpleKeyPair? keyPair;
    try {
      privateKey = _decodeRawBase64(decoded['privateKey']! as String);
      final publicKey = _decodeRawBase64(decoded['publicKey']! as String);
      if (privateKey.length != _keyLength || publicKey.length != _keyLength) {
        throw const FormatException('Mobile identity is invalid.');
      }
      keyPair = await _algorithm.newKeyPairFromSeed(privateKey);
      final identity = await MobileDeviceIdentity.fromKeyPair(keyPair);
      if (!_bytesEqual(identity.publicKey, publicKey)) {
        throw const FormatException('Mobile identity is invalid.');
      }
      return identity;
    } finally {
      privateKey?.fillRange(0, privateKey.length, 0);
      keyPair?.destroy();
    }
  }

  Future<String?> _readRecord() async {
    try {
      return await _storage.read();
    } catch (_) {
      throw const MobileIdentityException(MobileIdentityFailure.unavailable);
    }
  }

  Future<bool> _createRecord(String encoded) async {
    try {
      return await _storage.createIfAbsent(encoded);
    } catch (_) {
      throw const MobileIdentityException(MobileIdentityFailure.unavailable);
    }
  }
}

String _encodeRawBase64(List<int> value) =>
    base64.encode(value).replaceAll('=', '');

Uint8List _decodeRawBase64(String value) {
  Uint8List? decoded;
  try {
    final paddingLength = (4 - value.length % 4) % 4;
    decoded = Uint8List.fromList(
      base64.decode(value.padRight(value.length + paddingLength, '=')),
    );
    if (_encodeRawBase64(decoded) != value) {
      throw const FormatException('Base64 value is noncanonical.');
    }
    return decoded;
  } on FormatException {
    decoded?.fillRange(0, decoded.length, 0);
    throw const FormatException('Mobile identity is invalid.');
  }
}

bool _bytesEqual(List<int> first, List<int> second) {
  if (first.length != second.length) {
    return false;
  }
  var difference = 0;
  for (var index = 0; index < first.length; index++) {
    difference |= first[index] ^ second[index];
  }
  return difference == 0;
}
