import 'dart:async';

import 'package:coderoam/features/auth/domain/oidc_token_set.dart';
import 'package:coderoam/features/auth/infrastructure/appauth_oidc_client.dart';
import 'package:coderoam/features/auth/infrastructure/secure_oidc_token_store.dart';

final class OidcSessionUnavailable implements Exception {
  const OidcSessionUnavailable();
}

final class OidcSessionLifecycle {
  OidcSessionLifecycle({
    required AppAuthOidcClient client,
    required SecureOidcTokenStore tokenStore,
    DateTime Function() now = DateTime.now,
  }) : _client = client,
       _tokenStore = tokenStore,
       _now = now;

  final AppAuthOidcClient _client;
  final SecureOidcTokenStore _tokenStore;
  final DateTime Function() _now;
  OidcTokenSet? _tokens;
  Future<void>? _refreshInFlight;

  Future<bool> restore() async {
    try {
      _tokens = await _tokenStore.load();
      if (_tokens == null) {
        return false;
      }
      if (_tokens!.refreshRequired(_now())) {
        await _refresh();
      }
      return true;
    } catch (_) {
      throw const OidcSessionUnavailable();
    }
  }

  Future<String> authenticationEvidence() async {
    final tokens = _tokens;
    if (tokens == null) {
      throw const OidcSessionUnavailable();
    }
    try {
      if (tokens.refreshRequired(_now())) {
        await _refresh();
      }
      return _tokens!.idToken;
    } catch (_) {
      throw const OidcSessionUnavailable();
    }
  }

  Future<void> _refresh() async {
    final inFlight = _refreshInFlight;
    if (inFlight != null) {
      return inFlight;
    }
    final current = _tokens;
    if (current == null) {
      throw const OidcSessionUnavailable();
    }
    final operation = () async {
      final refreshed = await _client.refresh(current);
      await _tokenStore.save(refreshed);
      _tokens = refreshed;
    }();
    _refreshInFlight = operation;
    try {
      await operation;
    } finally {
      if (identical(_refreshInFlight, operation)) {
        _refreshInFlight = null;
      }
    }
  }
}
