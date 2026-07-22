import 'package:coderoam/features/auth/domain/oidc_client_configuration.dart';
import 'package:coderoam/features/auth/domain/oidc_token_set.dart';
import 'package:flutter_appauth/flutter_appauth.dart';

final class OidcAuthorizationCancelled implements Exception {
  const OidcAuthorizationCancelled();
}

final class OidcAuthorizationException implements Exception {
  const OidcAuthorizationException();
}

typedef AuthorizeAndExchangeCode =
    Future<AuthorizationTokenResponse> Function(
      AuthorizationTokenRequest request,
    );
typedef RefreshTokens = Future<TokenResponse> Function(TokenRequest request);

final class AppAuthOidcClient {
  AppAuthOidcClient({
    required OidcClientConfiguration configuration,
    FlutterAppAuth appAuth = const FlutterAppAuth(),
  }) : _configuration = configuration,
       _authorizeAndExchangeCode = appAuth.authorizeAndExchangeCode,
       _refreshTokens = appAuth.token;

  AppAuthOidcClient.withAuthorization({
    required OidcClientConfiguration configuration,
    required AuthorizeAndExchangeCode authorizeAndExchangeCode,
    required RefreshTokens refreshTokens,
  }) : _configuration = configuration,
       _authorizeAndExchangeCode = authorizeAndExchangeCode,
       _refreshTokens = refreshTokens;

  final OidcClientConfiguration _configuration;
  final AuthorizeAndExchangeCode _authorizeAndExchangeCode;
  final RefreshTokens _refreshTokens;

  Future<OidcTokenSet> signIn() async {
    try {
      final response = await _authorizeAndExchangeCode(
        AuthorizationTokenRequest(
          _configuration.clientId,
          OidcClientConfiguration.redirectUrl,
          issuer: _configuration.issuer,
          scopes: OidcClientConfiguration.scopes,
          promptValues: const [Prompt.consent],
        ),
      );
      return OidcTokenSet.fromTokens(
        idToken: response.idToken ?? '',
        refreshToken: response.refreshToken ?? '',
      );
    } on FlutterAppAuthUserCancelledException {
      throw const OidcAuthorizationCancelled();
    } catch (_) {
      throw const OidcAuthorizationException();
    }
  }

  Future<OidcTokenSet> refresh(OidcTokenSet current) async {
    try {
      final response = await _refreshTokens(
        TokenRequest(
          _configuration.clientId,
          OidcClientConfiguration.redirectUrl,
          issuer: _configuration.issuer,
          scopes: OidcClientConfiguration.scopes,
          refreshToken: current.refreshToken,
        ),
      );
      return OidcTokenSet.fromTokens(
        idToken: response.idToken ?? '',
        refreshToken: response.refreshToken ?? current.refreshToken,
      );
    } catch (_) {
      throw const OidcAuthorizationException();
    }
  }
}
