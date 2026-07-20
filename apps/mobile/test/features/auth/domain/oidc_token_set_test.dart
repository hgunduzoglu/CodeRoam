import 'dart:convert';

import 'package:coderoam/features/auth/domain/oidc_token_set.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('retains bounded tokens and refreshes before ID-token expiry', () {
    final tokenSet = OidcTokenSet.fromTokens(
      idToken: _idToken(expiresAtSeconds: 1800),
      refreshToken: 'opaque-refresh-token',
    );

    expect(tokenSet.refreshToken, 'opaque-refresh-token');
    expect(
      tokenSet.idTokenExpiresAt,
      DateTime.fromMillisecondsSinceEpoch(1800000, isUtc: true),
    );
    expect(
      tokenSet.refreshRequired(
        DateTime.fromMillisecondsSinceEpoch(1679000, isUtc: true),
      ),
      false,
    );
    expect(
      tokenSet.refreshRequired(
        DateTime.fromMillisecondsSinceEpoch(1680000, isUtc: true),
      ),
      true,
    );
  });

  test('rejects malformed or unbounded token state', () {
    for (final idToken in <String>[
      '',
      'two.segments',
      'empty..payload',
      '${base64Url.encode(utf8.encode('{}'))}.***.signature',
      _idToken(expiresAtSeconds: 0),
      _idToken(expiresAtSeconds: 253402300800),
      _idToken(expiresAtSeconds: null),
      '${'a' * (8 * 1024)}.payload.signature',
    ]) {
      expect(
        () => OidcTokenSet.fromTokens(
          idToken: idToken,
          refreshToken: 'opaque-refresh-token',
        ),
        throwsFormatException,
      );
    }
    for (final refreshToken in <String>[
      '',
      'refresh token',
      'refresh\ntoken',
      'a' * (16 * 1024 + 1),
    ]) {
      expect(
        () => OidcTokenSet.fromTokens(
          idToken: _idToken(expiresAtSeconds: 1800),
          refreshToken: refreshToken,
        ),
        throwsFormatException,
      );
    }
  });
}

String _idToken({required int? expiresAtSeconds}) {
  final header = base64Url
      .encode(utf8.encode('{"typ":"JWT"}'))
      .replaceAll('=', '');
  final payload = base64Url
      .encode(utf8.encode(jsonEncode({'exp': expiresAtSeconds})))
      .replaceAll('=', '');
  return '$header.$payload.signature';
}
