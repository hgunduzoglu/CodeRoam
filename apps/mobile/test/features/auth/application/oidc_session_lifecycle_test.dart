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

  test('signs in once, persists evidence, and signs out locally', () async {
    var signInCalls = 0;
    final signedInIdToken = _idToken(expiresAtSeconds: 3600);
    final lifecycle = _lifecycle(
      tokenStore: tokenStore,
      now: () => _time(600),
      authorizeAndExchangeCode: (_) async {
        signInCalls++;
        return _response(
          idToken: signedInIdToken,
          refreshToken: 'refresh-token',
        );
      },
      refreshTokens: (_) async => throw StateError('unexpected refresh'),
    );

    expect(await Future.wait([lifecycle.signIn(), lifecycle.signIn()]), [
      true,
      true,
    ]);
    expect(signInCalls, 1);
    expect(await lifecycle.authenticationEvidence(), signedInIdToken);
    expect((await tokenStore.load())?.idToken, signedInIdToken);

    await lifecycle.signOut();
    expect(await tokenStore.load(), isNull);
    expect(
      lifecycle.authenticationEvidence(),
      throwsA(isA<OidcSessionUnavailable>()),
    );
  });

  test('treats user-cancelled sign in as unchanged signed-out state', () async {
    final lifecycle = _lifecycle(
      tokenStore: tokenStore,
      now: () => _time(600),
      authorizeAndExchangeCode: (_) async {
        throw FlutterAppAuthUserCancelledException(
          code: 'cancelled',
          platformErrorDetails: FlutterAppAuthPlatformErrorDetails(),
        );
      },
      refreshTokens: (_) async => throw StateError('unexpected refresh'),
    );

    expect(await lifecycle.signIn(), false);
    expect(await tokenStore.load(), isNull);
  });

  test('sign out cannot be undone by a late refresh', () async {
    await tokenStore.save(
      _tokens(expiresAtSeconds: 1000, refreshToken: 'current-refresh'),
    );
    var clock = _time(100);
    final response = Completer<TokenResponse>();
    var signInCalls = 0;
    final lifecycle = _lifecycle(
      tokenStore: tokenStore,
      now: () => clock,
      authorizeAndExchangeCode: (_) async {
        signInCalls++;
        throw StateError('sign in must be blocked during sign out');
      },
      refreshTokens: (_) => response.future,
    );
    expect(await lifecycle.restore(), true);

    clock = _time(900);
    final evidence = lifecycle.authenticationEvidence();
    final evidenceExpectation = expectLater(
      evidence,
      throwsA(isA<OidcSessionUnavailable>()),
    );
    await Future<void>.delayed(Duration.zero);
    final signOut = lifecycle.signOut();
    await Future<void>.delayed(Duration.zero);
    expect(lifecycle.signIn(), throwsA(isA<OidcSessionUnavailable>()));
    expect(signInCalls, 0);
    response.complete(
      _response(
        idToken: _idToken(expiresAtSeconds: 3600),
        refreshToken: 'rotated-refresh',
      ),
    );

    await signOut;
    await evidenceExpectation;
    expect(await tokenStore.load(), isNull);
  });

  test('delayed restore cannot replace a newly signed-in account', () async {
    final oldTokens = _tokens(
      expiresAtSeconds: 3600,
      refreshToken: 'old-refresh',
    );
    final newIdToken = _idToken(expiresAtSeconds: 7200);
    final delayedStore = _DelayedLoadTokenStore();
    final lifecycle = _lifecycle(
      tokenStore: delayedStore,
      now: () => _time(600),
      authorizeAndExchangeCode:
          (_) async =>
              _response(idToken: newIdToken, refreshToken: 'new-refresh'),
      refreshTokens: (_) async => throw StateError('unexpected refresh'),
    );

    final restoreExpectation = expectLater(
      lifecycle.restore(),
      throwsA(isA<OidcSessionUnavailable>()),
    );
    expect(await lifecycle.signIn(), true);

    delayedStore.loadResult.complete(oldTokens);
    await restoreExpectation;
    expect(await lifecycle.authenticationEvidence(), newIdToken);
    expect(delayedStore.saved?.refreshToken, 'new-refresh');
  });

  test('failed save blocks restore until partial state is cleared', () async {
    final ambiguousStore = _SaveThenThrowTokenStore(clearFailures: 2);
    final lifecycle = _lifecycle(
      tokenStore: ambiguousStore,
      now: () => _time(600),
      authorizeAndExchangeCode:
          (_) async => _response(
            idToken: _idToken(expiresAtSeconds: 3600),
            refreshToken: 'new-refresh',
          ),
      refreshTokens: (_) async => throw StateError('unexpected refresh'),
    );

    await expectLater(
      lifecycle.signIn(),
      throwsA(isA<OidcSessionUnavailable>()),
    );
    expect(ambiguousStore.stored, isNotNull);

    await expectLater(
      lifecycle.restore(),
      throwsA(isA<OidcSessionUnavailable>()),
    );
    expect(ambiguousStore.loadCalls, 0);

    expect(await lifecycle.restore(), false);
    expect(ambiguousStore.stored, isNull);
    expect(ambiguousStore.loadCalls, 1);
  });

  test('late refresh save cannot clear a newer sign in', () async {
    final oldTokens = _tokens(
      expiresAtSeconds: 1000,
      refreshToken: 'old-refresh',
    );
    final delayedStore = _DelayedFirstSaveTokenStore(oldTokens);
    var clock = _time(100);
    var signInCalls = 0;
    final newIdToken = _idToken(expiresAtSeconds: 7200);
    final lifecycle = _lifecycle(
      tokenStore: delayedStore,
      now: () => clock,
      authorizeAndExchangeCode: (_) async {
        signInCalls++;
        return _response(idToken: newIdToken, refreshToken: 'new-refresh');
      },
      refreshTokens:
          (_) async => _response(
            idToken: _idToken(expiresAtSeconds: 3600),
            refreshToken: 'rotated-old-refresh',
          ),
    );
    expect(await lifecycle.restore(), true);

    clock = _time(900);
    final oldEvidenceExpectation = expectLater(
      lifecycle.authenticationEvidence(),
      throwsA(isA<OidcSessionUnavailable>()),
    );
    await delayedStore.firstSaveStarted.future;

    final signIn = lifecycle.signIn();
    await Future<void>.delayed(Duration.zero);
    expect(signInCalls, 0);
    delayedStore.allowFirstSave.complete();

    expect(await signIn, true);
    await oldEvidenceExpectation;
    expect(await lifecycle.authenticationEvidence(), newIdToken);
    expect(delayedStore.stored?.refreshToken, 'new-refresh');
  });

  test('restore waits for a stale sign-in save to be cleared', () async {
    final delayedStore = _DelayedFirstSaveTokenStore(null);
    final lifecycle = _lifecycle(
      tokenStore: delayedStore,
      now: () => _time(600),
      authorizeAndExchangeCode:
          (_) async => _response(
            idToken: _idToken(expiresAtSeconds: 3600),
            refreshToken: 'new-refresh',
          ),
      refreshTokens: (_) async => throw StateError('unexpected refresh'),
    );

    final signIn = lifecycle.signIn();
    await delayedStore.firstSaveStarted.future;
    final restore = lifecycle.restore();
    delayedStore.allowFirstSave.complete();

    expect(await signIn, false);
    expect(await restore, false);
    expect(delayedStore.stored, isNull);
  });
}

OidcSessionLifecycle _lifecycle({
  required OidcTokenStore tokenStore,
  required DateTime Function() now,
  required RefreshTokens refreshTokens,
  AuthorizeAndExchangeCode? authorizeAndExchangeCode,
}) {
  final configuration = OidcClientConfiguration.fromValues(
    issuer: 'https://identity.example.test',
    clientId: 'coderoam-mobile',
  );
  return OidcSessionLifecycle(
    client: AppAuthOidcClient.withAuthorization(
      configuration: configuration,
      authorizeAndExchangeCode:
          authorizeAndExchangeCode ??
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

final class _DelayedLoadTokenStore implements OidcTokenStore {
  final loadResult = Completer<OidcTokenSet?>();
  OidcTokenSet? saved;

  @override
  Future<OidcTokenSet?> load() => loadResult.future;

  @override
  Future<void> save(OidcTokenSet tokenSet) async {
    saved = tokenSet;
  }

  @override
  Future<void> clear() async {
    saved = null;
  }
}

final class _SaveThenThrowTokenStore implements OidcTokenStore {
  _SaveThenThrowTokenStore({required this.clearFailures});

  int clearFailures;
  int loadCalls = 0;
  OidcTokenSet? stored;

  @override
  Future<OidcTokenSet?> load() async {
    loadCalls++;
    return stored;
  }

  @override
  Future<void> save(OidcTokenSet tokenSet) async {
    stored = tokenSet;
    throw const OidcTokenStoreException();
  }

  @override
  Future<void> clear() async {
    if (clearFailures > 0) {
      clearFailures--;
      throw const OidcTokenStoreException();
    }
    stored = null;
  }
}

final class _DelayedFirstSaveTokenStore implements OidcTokenStore {
  _DelayedFirstSaveTokenStore(this.stored);

  final firstSaveStarted = Completer<void>();
  final allowFirstSave = Completer<void>();
  OidcTokenSet? stored;
  var saveCalls = 0;

  @override
  Future<OidcTokenSet?> load() async => stored;

  @override
  Future<void> save(OidcTokenSet tokenSet) async {
    saveCalls++;
    stored = tokenSet;
    if (saveCalls == 1) {
      firstSaveStarted.complete();
      await allowFirstSave.future;
    }
  }

  @override
  Future<void> clear() async {
    stored = null;
  }
}
