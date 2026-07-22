import 'dart:convert';

import 'package:coderoam/features/auth/domain/oidc_token_set.dart';
import 'package:coderoam/shared/control_plane/strict_json.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

final class OidcTokenStoreException implements Exception {
  const OidcTokenStoreException();
}

abstract interface class OidcTokenStore {
  Future<OidcTokenSet?> load();
  Future<void> save(OidcTokenSet tokenSet);
  Future<void> clear();
}

final class SecureOidcTokenStore implements OidcTokenStore {
  SecureOidcTokenStore({FlutterSecureStorage? storage})
    : _storage =
          storage ??
          const FlutterSecureStorage(
            iOptions: IOSOptions(
              accessibility: KeychainAccessibility.unlocked_this_device,
            ),
            aOptions: AndroidOptions(storageNamespace: 'coderoam_auth'),
          );

  static const _storageKey = 'coderoam.auth.oidc_tokens.v1';
  static const _maximumRecordBytes = 64 * 1024;

  final FlutterSecureStorage _storage;

  @override
  Future<OidcTokenSet?> load() async {
    final String? encoded;
    try {
      encoded = await _storage.read(key: _storageKey);
    } catch (_) {
      throw const OidcTokenStoreException();
    }
    if (encoded == null) {
      return null;
    }
    try {
      if (utf8.encode(encoded).length > _maximumRecordBytes) {
        throw const FormatException('OIDC token state is invalid.');
      }
      final decoded = decodeStrictJson(encoded);
      if (decoded is! Map<String, Object?> ||
          decoded.length != 2 ||
          decoded['idToken'] is! String ||
          decoded['refreshToken'] is! String) {
        throw const FormatException('OIDC token state is invalid.');
      }
      return OidcTokenSet.fromTokens(
        idToken: decoded['idToken']! as String,
        refreshToken: decoded['refreshToken']! as String,
      );
    } on FormatException {
      try {
        await _storage.delete(key: _storageKey);
      } catch (_) {
        throw const OidcTokenStoreException();
      }
      return null;
    }
  }

  @override
  Future<void> save(OidcTokenSet tokenSet) async {
    try {
      await _storage.write(
        key: _storageKey,
        value: jsonEncode({
          'idToken': tokenSet.idToken,
          'refreshToken': tokenSet.refreshToken,
        }),
      );
    } catch (_) {
      throw const OidcTokenStoreException();
    }
  }

  @override
  Future<void> clear() async {
    try {
      await _storage.delete(key: _storageKey);
    } catch (_) {
      throw const OidcTokenStoreException();
    }
  }
}
