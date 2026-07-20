import 'package:coderoam/features/auth/application/oidc_session_lifecycle.dart';
import 'package:coderoam/features/auth/presentation/authenticated_control_plane_shell.dart';
import 'package:coderoam/main.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('composes validated build inputs with the injected OIDC session', () {
    final session = _FakeOidcSession();

    final home = createProductionHome(
      controlPlaneOrigin: 'https://control.example',
      oidcIssuer: 'https://identity.example',
      oidcClientId: 'coderoam-mobile',
      deviceId: '4123456789abcdef0123456789abcdef',
      sessionFactory: (_) => session,
    );

    expect(home, isA<AuthenticatedControlPlaneShell>());
    final shell = home as AuthenticatedControlPlaneShell;
    expect(shell.session, same(session));
    expect(shell.configuration.origin.uri.host, 'control.example');
    expect(shell.configuration.oidc.issuer, 'https://identity.example');
    expect(
      shell.configuration.deviceId.value,
      '4123456789abcdef0123456789abcdef',
    );
  });

  testWidgets('renders a fixed message when build inputs are invalid', (
    tester,
  ) async {
    await tester.pumpWidget(
      CodeRoamApp(
        home: createProductionHome(
          controlPlaneOrigin: 'http://untrusted.example',
          oidcIssuer: 'provider-secret-details',
          oidcClientId: '',
          deviceId: '',
        ),
      ),
    );

    expect(
      find.text('This CodeRoam build is not configured for sign-in.'),
      findsOneWidget,
    );
    expect(find.textContaining('provider-secret-details'), findsNothing);
    expect(find.textContaining('untrusted.example'), findsNothing);
  });
}

final class _FakeOidcSession implements OidcSession {
  @override
  Future<bool> restore() async => false;

  @override
  Future<bool> signIn() async => false;

  @override
  Future<String> authenticationEvidence() async => 'opaque-evidence';

  @override
  Future<void> signOut() async {}
}
