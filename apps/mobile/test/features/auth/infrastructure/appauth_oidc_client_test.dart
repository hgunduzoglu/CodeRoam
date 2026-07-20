import 'dart:convert';

import 'package:coderoam/features/auth/domain/oidc_client_configuration.dart';
import 'package:coderoam/features/auth/infrastructure/appauth_oidc_client.dart';
import 'package:flutter_appauth/flutter_appauth.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  final configuration = OidcClientConfiguration.fromValues(
    issuer: 'https://identity.example.test/tenant',
    clientId: 'coderoam-mobile',
  );

  test(
    'starts a public-client authorization code flow with PKCE inputs',
    () async {
      AuthorizationTokenRequest? capturedRequest;
      final client = AppAuthOidcClient.withAuthorization(
        configuration: configuration,
        authorizeAndExchangeCode: (request) async {
          capturedRequest = request;
          return _response(
            idToken: _idToken(expiresAtSeconds: 1800),
            refreshToken: 'opaque-refresh-token',
          );
        },
      );

      final tokenSet = await client.signIn();

      expect(capturedRequest?.clientId, 'coderoam-mobile');
      expect(
        capturedRequest?.redirectUrl,
        'dev.coderoam.coderoam:/oauthredirect',
      );
      expect(capturedRequest?.issuer, 'https://identity.example.test/tenant');
      expect(capturedRequest?.clientSecret, isNull);
      expect(capturedRequest?.grantType, GrantType.authorizationCode);
      expect(capturedRequest?.scopes, ['openid', 'offline_access']);
      expect(capturedRequest?.promptValues, [Prompt.consent]);
      expect(tokenSet.refreshToken, 'opaque-refresh-token');
    },
  );

  test('fails closed when the provider omits required token state', () async {
    for (final response in <AuthorizationTokenResponse>[
      _response(idToken: null, refreshToken: 'opaque-refresh-token'),
      _response(idToken: _idToken(expiresAtSeconds: 1800), refreshToken: null),
    ]) {
      final client = AppAuthOidcClient.withAuthorization(
        configuration: configuration,
        authorizeAndExchangeCode: (_) async => response,
      );

      expect(client.signIn(), throwsA(isA<OidcAuthorizationException>()));
    }
  });

  test('distinguishes user cancellation from provider failure', () async {
    final cancelled = AppAuthOidcClient.withAuthorization(
      configuration: configuration,
      authorizeAndExchangeCode: (_) async {
        throw FlutterAppAuthUserCancelledException(
          code: 'cancelled',
          platformErrorDetails: FlutterAppAuthPlatformErrorDetails(),
        );
      },
    );
    final failed = AppAuthOidcClient.withAuthorization(
      configuration: configuration,
      authorizeAndExchangeCode: (_) async => throw StateError('provider'),
    );

    expect(cancelled.signIn(), throwsA(isA<OidcAuthorizationCancelled>()));
    expect(failed.signIn(), throwsA(isA<OidcAuthorizationException>()));
  });
}

AuthorizationTokenResponse _response({
  required String? idToken,
  required String? refreshToken,
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
