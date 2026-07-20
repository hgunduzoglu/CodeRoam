import 'dart:convert';

import 'package:coderoam/shared/control_plane/strict_json.dart';

final class OidcTokenSet {
  const OidcTokenSet._({
    required this.idToken,
    required this.refreshToken,
    required this.idTokenExpiresAt,
  });

  static const _refreshSkew = Duration(minutes: 2);

  final String idToken;
  final String refreshToken;
  final DateTime idTokenExpiresAt;

  factory OidcTokenSet.fromTokens({
    required String idToken,
    required String refreshToken,
  }) {
    final idTokenBytes = utf8.encode(idToken);
    final refreshTokenBytes = utf8.encode(refreshToken);
    final segments = idToken.split('.');
    if (idTokenBytes.isEmpty ||
        idTokenBytes.length > 8 * 1024 ||
        segments.length != 3 ||
        segments.any((segment) => segment.isEmpty) ||
        !idToken.runes.every((rune) => rune >= 0x21 && rune <= 0x7e) ||
        refreshTokenBytes.isEmpty ||
        refreshTokenBytes.length > 16 * 1024 ||
        !refreshToken.runes.every((rune) => rune >= 0x21 && rune <= 0x7e)) {
      throw const FormatException('OIDC token state is invalid.');
    }

    try {
      final payload = decodeStrictJson(
        utf8.decode(
          base64Url.decode(base64Url.normalize(segments[1])),
          allowMalformed: false,
        ),
      );
      if (payload is! Map<String, Object?> || payload['exp'] is! int) {
        throw const FormatException('OIDC token state is invalid.');
      }
      final expiresAtSeconds = payload['exp']! as int;
      if (expiresAtSeconds < 1 || expiresAtSeconds > 253402300799) {
        throw const FormatException('OIDC token state is invalid.');
      }
      return OidcTokenSet._(
        idToken: idToken,
        refreshToken: refreshToken,
        idTokenExpiresAt: DateTime.fromMillisecondsSinceEpoch(
          expiresAtSeconds * 1000,
          isUtc: true,
        ),
      );
    } catch (_) {
      throw const FormatException('OIDC token state is invalid.');
    }
  }

  bool refreshRequired(DateTime now) {
    return !idTokenExpiresAt.isAfter(now.toUtc().add(_refreshSkew));
  }
}
