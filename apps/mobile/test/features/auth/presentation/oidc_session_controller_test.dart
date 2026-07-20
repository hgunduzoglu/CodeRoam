import 'dart:async';

import 'package:coderoam/features/auth/application/oidc_session_lifecycle.dart';
import 'package:coderoam/features/auth/presentation/oidc_session_controller.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('restores signed-in and signed-out state', () async {
    for (final restored in [true, false]) {
      final session = _FakeOidcSession()..restoreResult = restored;
      final controller = OidcSessionController(session);
      final states = <OidcSessionStatus>[];
      controller.addListener(() => states.add(controller.status));

      await controller.restore();

      expect(states, [
        OidcSessionStatus.restoring,
        restored ? OidcSessionStatus.signedIn : OidcSessionStatus.signedOut,
      ]);
      expect(session.restoreCalls, 1);
      controller.dispose();
    }
  });

  test(
    'maps sign-in cancellation, success, and failure to fixed states',
    () async {
      final session = _FakeOidcSession()..restoreResult = false;
      final controller = OidcSessionController(session);
      await controller.restore();

      session.signInResult = false;
      await controller.signIn();
      expect(controller.status, OidcSessionStatus.signedOut);

      session.signInResult = true;
      await controller.signIn();
      expect(controller.status, OidcSessionStatus.signedIn);

      final failedSession = _FakeOidcSession()..restoreResult = false;
      final failedController = OidcSessionController(failedSession);
      await failedController.restore();
      failedSession.signInError = StateError('provider details');
      await failedController.signIn();
      expect(failedController.status, OidcSessionStatus.failed);
    },
  );

  test('ignores completion and further work after dispose', () async {
    final pendingRestore = Completer<bool>();
    final session = _FakeOidcSession()..pendingRestore = pendingRestore;
    final controller = OidcSessionController(session);
    final restore = controller.restore();

    controller.dispose();
    pendingRestore.complete(true);
    await restore;
    await controller.signIn();

    expect(session.signInCalls, 0);
  });
}

final class _FakeOidcSession implements OidcSession {
  bool restoreResult = false;
  bool signInResult = false;
  Object? signInError;
  Completer<bool>? pendingRestore;
  int restoreCalls = 0;
  int signInCalls = 0;

  @override
  Future<bool> restore() async {
    restoreCalls++;
    return pendingRestore?.future ?? restoreResult;
  }

  @override
  Future<bool> signIn() async {
    signInCalls++;
    final error = signInError;
    if (error != null) {
      throw error;
    }
    return signInResult;
  }

  @override
  Future<String> authenticationEvidence() async => 'opaque-evidence';

  @override
  Future<void> signOut() async {}
}
