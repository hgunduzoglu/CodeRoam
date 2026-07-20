import 'dart:async';
import 'dart:convert';

import 'package:coderoam/features/auth/application/oidc_session_lifecycle.dart';
import 'package:coderoam/features/auth/domain/oidc_client_configuration.dart';
import 'package:coderoam/features/auth/domain/oidc_token_set.dart';
import 'package:coderoam/features/auth/infrastructure/appauth_oidc_client.dart';
import 'package:coderoam/features/auth/infrastructure/secure_oidc_token_store.dart';
import 'package:flutter_appauth/flutter_appauth.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  const storage = FlutterSecureStorage();
  late SecureOidcTokenStore tokenStore;

  setUp(() {
    FlutterSecureStorage.setMockInitialValues({});
    tokenStore = SecureOidcTokenStore(storage: storage);
  });

  test('restores fresh evidence without contacting the provider', () async {
    final stored = _tokens(expiresAtSeconds: 3600, refreshToken: 'refresh');
    await tokenStore.save(stored);
    var refreshCalls = 0;
    final lifecycle = _lifecycle(
      tokenStore: tokenStore,
      now: () => _time(600),
      refreshTokens: (_) async {
        refreshCalls++;
        throw StateError('unexpected refresh');
      },
    );

    expect(await lifecycle.restore(), true);
    expect(await lifecycle.authenticationEvidence(), stored.idToken);
    expect(refreshCalls, 0);
  });

  test('refreshes and persists evidence during restore', () async {
    await tokenStore.save(
      _tokens(expiresAtSeconds: 700, refreshToken: 'current-refresh'),
    );
    final lifecycle = _lifecycle(
      tokenStore: tokenStore,
      now: () => _time(600),
      refreshTokens:
          (_) async => _response(
            idToken: _idToken(expiresAtSeconds: 3600),
            refreshToken: 'rotated-refresh',
          ),
    );

    expect(await lifecycle.restore(), true);
    expect(
      await lifecycle.authenticationEvidence(),
      _idToken(expiresAtSeconds: 3600),
    );
    expect((await tokenStore.load())?.refreshToken, 'rotated-refresh');
  });

  test('coalesces concurrent evidence refreshes', () async {
    await tokenStore.save(
      _tokens(expiresAtSeconds: 1000, refreshToken: 'current-refresh'),
    );
    final response = Completer<TokenResponse>();
    var refreshCalls = 0;
    var clock = _time(100);
    final lifecycle = _lifecycle(
      tokenStore: tokenStore,
      now: () => clock,
      refreshTokens: (_) {
        refreshCalls++;
        return response.future;
      },
    );
    expect(await lifecycle.restore(), true);

    clock = _time(900);
    final first = lifecycle.authenticationEvidence();
    final second = lifecycle.authenticationEvidence();
    await Future<void>.delayed(Duration.zero);
    expect(refreshCalls, 1);

    final refreshedIdToken = _idToken(expiresAtSeconds: 3600);
    response.complete(
      _response(idToken: refreshedIdToken, refreshToken: 'rotated-refresh'),
    );
    expect(await Future.wait([first, second]), [
      refreshedIdToken,
      refreshedIdToken,
    ]);
  });

  test('fails closed before a token set is restored', () async {
    final lifecycle = _lifecycle(
      tokenStore: tokenStore,
      now: () => _time(600),
      refreshTokens: (_) async => throw StateError('unexpected refresh'),
    );

    expect(await lifecycle.restore(), false);
    expect(
      lifecycle.authenticationEvidence(),
      throwsA(isA<OidcSessionUnavailable>()),
    );
  });
}

OidcSessionLifecycle _lifecycle({
  required SecureOidcTokenStore tokenStore,
  required DateTime Function() now,
  required RefreshTokens refreshTokens,
}) {
  final configuration = OidcClientConfiguration.fromValues(
    issuer: 'https://identity.example.test',
    clientId: 'coderoam-mobile',
  );
  return OidcSessionLifecycle(
    client: AppAuthOidcClient.withAuthorization(
      configuration: configuration,
      authorizeAndExchangeCode:
          (_) async => throw StateError('unexpected sign in'),
      refreshTokens: refreshTokens,
    ),
    tokenStore: tokenStore,
    now: now,
  );
}

OidcTokenSet _tokens({
  required int expiresAtSeconds,
  required String refreshToken,
}) {
  return OidcTokenSet.fromTokens(
    idToken: _idToken(expiresAtSeconds: expiresAtSeconds),
    refreshToken: refreshToken,
  );
}

AuthorizationTokenResponse _response({
  required String idToken,
  required String refreshToken,
}) {
  return AuthorizationTokenResponse(
    null,
    refreshToken,
    null,
    idToken,
    null,
    OidcClientConfiguration.scopes,
    null,
    null,
  );
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

DateTime _time(int seconds) =>
    DateTime.fromMillisecondsSinceEpoch(seconds * 1000, isUtc: true);
