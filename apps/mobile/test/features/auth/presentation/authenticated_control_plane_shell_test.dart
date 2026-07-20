import 'dart:convert';

import 'package:coderoam/features/auth/application/oidc_session_lifecycle.dart';
import 'package:coderoam/features/auth/presentation/authenticated_control_plane_shell.dart';
import 'package:coderoam/features/control_plane/application/mobile_control_plane_configuration.dart';
import 'package:coderoam/shared/control_plane/control_plane_transport.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('restores, signs in, and removes authenticated state on logout', (
    tester,
  ) async {
    final session = _FakeOidcSession();
    final transport = _ProjectTransportStub();

    await tester.pumpWidget(
      MaterialApp(
        home: AuthenticatedControlPlaneShell(
          configuration: _configuration(),
          session: session,
          transportFactory: (_) => transport,
        ),
      ),
    );
    await tester.pump();

    expect(find.text('Sign in'), findsOneWidget);
    expect(session.restoreCalls, 1);

    session.signInResult = true;
    await tester.tap(find.text('Sign in'));
    await tester.pump();
    await tester.pump();

    expect(find.text('Projects'), findsOneWidget);
    expect(find.text('No projects are available yet.'), findsOneWidget);
    expect(session.signInCalls, 1);

    await tester.tap(find.byTooltip('Sign out'));
    await tester.pump();
    await tester.pump();

    expect(find.text('Sign in'), findsOneWidget);
    expect(find.text('Projects'), findsNothing);
    expect(session.signOutCalls, 1);
    expect(transport.closeCalls, 1);
  });

  testWidgets('shows only a fixed retry message after provider failure', (
    tester,
  ) async {
    final session =
        _FakeOidcSession()..signInError = StateError('provider-secret-details');

    await tester.pumpWidget(
      MaterialApp(
        home: AuthenticatedControlPlaneShell(
          configuration: _configuration(),
          session: session,
          transportFactory: (_) => _ProjectTransportStub(),
        ),
      ),
    );
    await tester.pump();
    await tester.tap(find.text('Sign in'));
    await tester.pump();
    await tester.pump();

    expect(find.text('Sign-in is unavailable right now.'), findsOneWidget);
    expect(find.text('Try again'), findsOneWidget);
    expect(find.textContaining('provider-secret-details'), findsNothing);
  });
}

MobileControlPlaneConfiguration _configuration() =>
    MobileControlPlaneConfiguration.fromValues(
      controlPlaneOrigin: 'https://control.example',
      oidcIssuer: 'https://identity.example',
      oidcClientId: 'coderoam-mobile',
      deviceId: '4123456789abcdef0123456789abcdef',
    );

final class _FakeOidcSession implements OidcSession {
  bool signInResult = false;
  Object? signInError;
  int restoreCalls = 0;
  int signInCalls = 0;
  int signOutCalls = 0;

  @override
  Future<bool> restore() async {
    restoreCalls++;
    return false;
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
  Future<void> signOut() async {
    signOutCalls++;
  }
}

final class _ProjectTransportStub implements ControlPlaneTransport {
  int closeCalls = 0;

  @override
  Future<ControlPlaneHttpResponse> send(ControlPlaneHttpRequest request) async {
    return ControlPlaneHttpResponse(
      statusCode: 200,
      contentType: 'application/json',
      body: utf8.encode(jsonEncode({'projects': <Object>[]})),
    );
  }

  @override
  void close() {
    closeCalls++;
  }
}
