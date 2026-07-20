import 'package:coderoam/features/auth/domain/oidc_client_configuration.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('retains exact valid public-client configuration', () {
    final configuration = OidcClientConfiguration.fromValues(
      issuer: 'https://identity.example.test/tenant/',
      clientId: 'coderoam-mobile',
    );

    expect(configuration.issuer, 'https://identity.example.test/tenant/');
    expect(configuration.clientId, 'coderoam-mobile');
    expect(
      OidcClientConfiguration.redirectUrl,
      'dev.coderoam.coderoam:/oauthredirect',
    );
    expect(OidcClientConfiguration.scopes, ['openid', 'offline_access']);
  });

  test('rejects malformed or unsafe public-client configuration', () {
    for (final issuer in <String>[
      '',
      'http://identity.example.test',
      'https://',
      'https://user@identity.example.test',
      'https://identity.example.test?tenant=one',
      'https://identity.example.test#fragment',
      'https://identity.example.test:0',
      'https://${'a' * 2048}.example.test',
    ]) {
      expect(
        () => OidcClientConfiguration.fromValues(
          issuer: issuer,
          clientId: 'coderoam-mobile',
        ),
        throwsFormatException,
        reason: issuer,
      );
    }
    for (final clientId in <String>[
      '',
      ' coderoam-mobile',
      'coderoam\nmobile',
      'a' * 257,
    ]) {
      expect(
        () => OidcClientConfiguration.fromValues(
          issuer: 'https://identity.example.test',
          clientId: clientId,
        ),
        throwsFormatException,
        reason: clientId,
      );
    }
  });
}
