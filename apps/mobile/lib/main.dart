import 'package:coderoam/features/auth/application/oidc_session_lifecycle.dart';
import 'package:coderoam/features/auth/domain/oidc_client_configuration.dart';
import 'package:coderoam/features/auth/infrastructure/appauth_oidc_client.dart';
import 'package:coderoam/features/auth/infrastructure/secure_oidc_token_store.dart';
import 'package:coderoam/features/auth/presentation/authenticated_control_plane_shell.dart';
import 'package:coderoam/features/control_plane/application/mobile_control_plane_configuration.dart';
import 'package:flutter/material.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();
  runApp(CodeRoamApp(home: createProductionHome()));
}

typedef OidcSessionFactory = OidcSession Function(OidcClientConfiguration);

Widget createProductionHome({
  String controlPlaneOrigin = const String.fromEnvironment(
    'CODEROAM_CONTROL_PLANE_ORIGIN',
  ),
  String oidcIssuer = const String.fromEnvironment('CODEROAM_OIDC_ISSUER'),
  String oidcClientId = const String.fromEnvironment('CODEROAM_OIDC_CLIENT_ID'),
  String deviceId = const String.fromEnvironment('CODEROAM_DEVICE_ID'),
  OidcSessionFactory? sessionFactory,
}) {
  try {
    final configuration = MobileControlPlaneConfiguration.fromValues(
      controlPlaneOrigin: controlPlaneOrigin,
      oidcIssuer: oidcIssuer,
      oidcClientId: oidcClientId,
      deviceId: deviceId,
    );
    final session =
        sessionFactory?.call(configuration.oidc) ??
        OidcSessionLifecycle(
          client: AppAuthOidcClient(configuration: configuration.oidc),
          tokenStore: SecureOidcTokenStore(),
        );
    return AuthenticatedControlPlaneShell(
      configuration: configuration,
      session: session,
    );
  } on FormatException {
    return const _ConfigurationUnavailable();
  }
}

class CodeRoamApp extends StatelessWidget {
  const CodeRoamApp({required this.home, super.key});

  final Widget home;

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'CodeRoam',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        colorSchemeSeed: const Color(0xFF635BFF),
        brightness: Brightness.dark,
        useMaterial3: true,
      ),
      home: home,
    );
  }
}

final class _ConfigurationUnavailable extends StatelessWidget {
  const _ConfigurationUnavailable();

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('CodeRoam')),
      body: const Center(
        child: Padding(
          padding: EdgeInsets.all(24),
          child: Text(
            'This CodeRoam build is not configured for sign-in.',
            textAlign: TextAlign.center,
          ),
        ),
      ),
    );
  }
}
