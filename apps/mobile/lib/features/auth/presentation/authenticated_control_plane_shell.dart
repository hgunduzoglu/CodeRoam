import 'dart:async';

import 'package:coderoam/features/auth/application/oidc_session_lifecycle.dart';
import 'package:coderoam/features/auth/presentation/oidc_session_controller.dart';
import 'package:coderoam/features/control_plane/application/mobile_control_plane_configuration.dart';
import 'package:coderoam/features/control_plane/presentation/control_plane_shell.dart';
import 'package:flutter/material.dart';

final class AuthenticatedControlPlaneShell extends StatefulWidget {
  const AuthenticatedControlPlaneShell({
    required this.configuration,
    required this.session,
    this.transportFactory,
    this.touchWorkspaceBuilder,
    super.key,
  });

  final MobileControlPlaneConfiguration configuration;
  final OidcSession session;
  final ControlPlaneTransportFactory? transportFactory;
  final WidgetBuilder? touchWorkspaceBuilder;

  @override
  State<AuthenticatedControlPlaneShell> createState() =>
      _AuthenticatedControlPlaneShellState();
}

final class _AuthenticatedControlPlaneShellState
    extends State<AuthenticatedControlPlaneShell> {
  late final OidcSessionController _controller;

  @override
  void initState() {
    super.initState();
    _controller = OidcSessionController(widget.session);
    unawaited(_controller.restore());
  }

  @override
  Widget build(BuildContext context) {
    return ListenableBuilder(
      listenable: _controller,
      builder: (context, _) {
        return switch (_controller.status) {
          OidcSessionStatus.initial ||
          OidcSessionStatus.restoring ||
          OidcSessionStatus.signingIn ||
          OidcSessionStatus.signingOut => const _AuthenticationProgress(),
          OidcSessionStatus.signedOut => _SignInPrompt(
            onSignIn: () => unawaited(_controller.signIn()),
          ),
          OidcSessionStatus.failed => _SignInPrompt(
            failed: true,
            onSignIn: () => unawaited(_controller.signIn()),
          ),
          OidcSessionStatus.signedIn => ControlPlaneShell(
            origin: widget.configuration.origin,
            deviceId: widget.configuration.deviceId,
            authenticationEvidence: widget.session.authenticationEvidence,
            onSignOut: () => unawaited(_controller.signOut()),
            transportFactory: widget.transportFactory,
            touchWorkspaceBuilder: widget.touchWorkspaceBuilder,
          ),
        };
      },
    );
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }
}

final class _AuthenticationProgress extends StatelessWidget {
  const _AuthenticationProgress();

  @override
  Widget build(BuildContext context) {
    return const Scaffold(body: Center(child: CircularProgressIndicator()));
  }
}

final class _SignInPrompt extends StatelessWidget {
  const _SignInPrompt({required this.onSignIn, this.failed = false});

  final VoidCallback onSignIn;
  final bool failed;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('CodeRoam')),
      body: Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(
                failed
                    ? 'Sign-in is unavailable right now.'
                    : 'Sign in to reach your registered workspaces.',
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: 16),
              FilledButton(
                onPressed: onSignIn,
                child: Text(failed ? 'Try again' : 'Sign in'),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
