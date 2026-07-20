import 'dart:convert';

final class OidcClientConfiguration {
  const OidcClientConfiguration._({
    required this.issuer,
    required this.clientId,
  });

  static const redirectUrl = 'dev.coderoam.coderoam:/oauthredirect';
  static const scopes = <String>['openid', 'offline_access'];

  final String issuer;
  final String clientId;

  factory OidcClientConfiguration.fromValues({
    required String issuer,
    required String clientId,
  }) {
    final boundedIssuer = _boundedValue(issuer, 2048);
    final boundedClientId = _boundedValue(clientId, 256);
    Uri issuerUri;
    try {
      issuerUri = Uri.parse(boundedIssuer);
      issuerUri.port;
    } on FormatException {
      throw const FormatException('OIDC configuration is invalid.');
    }
    if (issuerUri.scheme != 'https' ||
        issuerUri.host.isEmpty ||
        issuerUri.userInfo.isNotEmpty ||
        (issuerUri.hasPort && issuerUri.port == 0) ||
        issuerUri.hasQuery ||
        issuerUri.hasFragment) {
      throw const FormatException('OIDC configuration is invalid.');
    }
    return OidcClientConfiguration._(
      issuer: boundedIssuer,
      clientId: boundedClientId,
    );
  }
}

String _boundedValue(String value, int maximumBytes) {
  if (value.isEmpty ||
      value.trim() != value ||
      utf8.encode(value).length > maximumBytes ||
      value.runes.any((rune) => rune < 0x20 || rune == 0x7f)) {
    throw const FormatException('OIDC configuration is invalid.');
  }
  return value;
}
