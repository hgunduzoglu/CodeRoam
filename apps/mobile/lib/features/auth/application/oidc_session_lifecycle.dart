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
    required OidcTokenStore tokenStore,
    DateTime Function() now = DateTime.now,
  }) : _client = client,
       _tokenStore = tokenStore,
       _now = now;

  final AppAuthOidcClient _client;
  final OidcTokenStore _tokenStore;
  final DateTime Function() _now;
  OidcTokenSet? _tokens;
  Future<bool>? _signInInFlight;
  Future<void>? _signOutInFlight;
  Future<void>? _refreshInFlight;
  var _generation = 0;
  var _cleanupRequired = false;

  Future<bool> restore() async {
    final generation = ++_generation;
    _tokens = null;
    try {
      try {
        await _signInInFlight;
      } catch (_) {
        // Restore reads storage only after stale sign-in cleanup finishes.
      }
      try {
        await _refreshInFlight;
      } catch (_) {
        // Restore reads storage only after stale refresh cleanup finishes.
      }
      if (generation != _generation || _signOutInFlight != null) {
        throw const OidcSessionUnavailable();
      }
      if (_cleanupRequired) {
        await _clearContaminatedState();
      }
      final restored = await _tokenStore.load();
      if (generation != _generation || _signOutInFlight != null) {
        throw const OidcSessionUnavailable();
      }
      _tokens = restored;
      if (restored == null) {
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

  Future<bool> signIn() async {
    if (_signOutInFlight != null) {
      throw const OidcSessionUnavailable();
    }
    final inFlight = _signInInFlight;
    if (inFlight != null) {
      return inFlight;
    }
    final generation = ++_generation;
    _tokens = null;
    final operation = () async {
      try {
        await _refreshInFlight;
      } catch (_) {
        // The new sign-in starts only after stale refresh cleanup finishes.
      }
      if (generation != _generation || _signOutInFlight != null) {
        return false;
      }
      if (_cleanupRequired) {
        await _clearContaminatedState();
      }
      final OidcTokenSet tokens;
      try {
        tokens = await _client.signIn();
      } on OidcAuthorizationCancelled {
        return false;
      } catch (_) {
        throw const OidcSessionUnavailable();
      }
      if (generation != _generation) {
        return false;
      }
      try {
        await _tokenStore.save(tokens);
        if (generation != _generation) {
          await _tokenStore.clear();
          return false;
        }
        _tokens = tokens;
        _cleanupRequired = false;
        return true;
      } catch (_) {
        _tokens = null;
        _cleanupRequired = true;
        try {
          await _clearContaminatedState();
        } catch (_) {
          // A later restore retries cleanup before reading token state.
        }
        throw const OidcSessionUnavailable();
      }
    }();
    _signInInFlight = operation;
    try {
      return await operation;
    } finally {
      if (identical(_signInInFlight, operation)) {
        _signInInFlight = null;
      }
    }
  }

  Future<void> signOut() async {
    final inFlight = _signOutInFlight;
    if (inFlight != null) {
      return inFlight;
    }
    final operation = () async {
      _generation++;
      _tokens = null;
      _cleanupRequired = true;
      try {
        await _tokenStore.clear();
      } catch (_) {
        // A second clear runs after late authentication operations finish.
      }
      try {
        await _signInInFlight;
      } catch (_) {
        // Only the final secure-storage clear decides logout success.
      }
      try {
        await _refreshInFlight;
      } catch (_) {
        // Only the final secure-storage clear decides logout success.
      }
      _generation++;
      _tokens = null;
      await _clearContaminatedState();
    }();
    _signOutInFlight = operation;
    try {
      await operation;
    } catch (_) {
      throw const OidcSessionUnavailable();
    } finally {
      if (identical(_signOutInFlight, operation)) {
        _signOutInFlight = null;
      }
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
    final generation = _generation;
    final operation = () async {
      final refreshed = await _client.refresh(current);
      if (generation != _generation || !identical(_tokens, current)) {
        throw const OidcSessionUnavailable();
      }
      try {
        await _tokenStore.save(refreshed);
        if (generation != _generation || !identical(_tokens, current)) {
          await _tokenStore.clear();
          throw const OidcSessionUnavailable();
        }
      } catch (_) {
        _tokens = null;
        _cleanupRequired = true;
        try {
          await _clearContaminatedState();
        } catch (_) {
          // A later restore retries cleanup before reading token state.
        }
        throw const OidcSessionUnavailable();
      }
      _tokens = refreshed;
      _cleanupRequired = false;
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

  Future<void> _clearContaminatedState() async {
    try {
      await _tokenStore.clear();
      _cleanupRequired = false;
    } catch (_) {
      _cleanupRequired = true;
      throw const OidcSessionUnavailable();
    }
  }
}
