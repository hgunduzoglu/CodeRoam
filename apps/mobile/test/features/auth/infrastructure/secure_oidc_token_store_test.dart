import 'dart:convert';

import 'package:coderoam/features/auth/domain/oidc_token_set.dart';
import 'package:coderoam/features/auth/infrastructure/secure_oidc_token_store.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  const storage = FlutterSecureStorage();
  late SecureOidcTokenStore store;

  setUp(() {
    FlutterSecureStorage.setMockInitialValues({});
    store = SecureOidcTokenStore(storage: storage);
  });

  test('atomically saves, loads, and clears one token record', () async {
    final tokenSet = OidcTokenSet.fromTokens(
      idToken: _idToken(expiresAtSeconds: 1800),
      refreshToken: 'opaque-refresh-token',
    );

    expect(await store.load(), isNull);
    await store.save(tokenSet);

    final loaded = await store.load();
    expect(loaded?.idToken, tokenSet.idToken);
    expect(loaded?.refreshToken, tokenSet.refreshToken);
    expect(loaded?.idTokenExpiresAt, tokenSet.idTokenExpiresAt);

    final escapingTokenSet = OidcTokenSet.fromTokens(
      idToken: _idToken(expiresAtSeconds: 1800),
      refreshToken: '"' * (16 * 1024),
    );
    await store.save(escapingTokenSet);
    expect((await store.load())?.refreshToken.length, 16 * 1024);

    await store.clear();
    expect(await store.load(), isNull);
  });

  test('clears malformed, unexpected, or oversized records', () async {
    for (final record in <String>[
      '{',
      '{"idToken":"one","idToken":"two","refreshToken":"three"}',
      '{"idToken":"one","refreshToken":"two","extra":true}',
      'a' * (64 * 1024 + 1),
    ]) {
      FlutterSecureStorage.setMockInitialValues({
        'coderoam.auth.oidc_tokens.v1': record,
      });

      expect(await store.load(), isNull);
      expect(await storage.read(key: 'coderoam.auth.oidc_tokens.v1'), isNull);
    }
  });
}

String _idToken({required int expiresAtSeconds}) {
  final header = base64Url
      .encode(utf8.encode('{"typ":"JWT"}'))
      .replaceAll('=', '');
  final payload = base64Url
      .encode(utf8.encode(jsonEncode({'exp': expiresAtSeconds})))
      .replaceAll('=', '');
  return '$header.$payload.signature';
}
