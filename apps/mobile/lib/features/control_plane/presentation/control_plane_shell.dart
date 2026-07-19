import 'dart:async';

import 'package:coderoam/features/control_plane/infrastructure/control_plane_api_repository.dart';
import 'package:coderoam/features/session/presentation/project_session_screen.dart';
import 'package:coderoam/features/session/presentation/session_start_controller.dart';
import 'package:coderoam/features/workspace/domain/project_summary.dart';
import 'package:coderoam/features/workspace/presentation/project_catalog_controller.dart';
import 'package:coderoam/features/workspace/presentation/project_catalog_screen.dart';
import 'package:coderoam/features/workspace/presentation/touch_spike_shell.dart';
import 'package:coderoam/shared/control_plane/control_plane_transport.dart';
import 'package:coderoam/shared/domain/opaque_id.dart';
import 'package:flutter/material.dart';

typedef ControlPlaneTransportFactory =
    ControlPlaneTransport Function(ControlPlaneOrigin origin);

final class ControlPlaneShell extends StatefulWidget {
  const ControlPlaneShell({
    required this.origin,
    required this.deviceId,
    required this.authenticationEvidence,
    this.transportFactory,
    this.touchWorkspaceBuilder,
    super.key,
  });

  final ControlPlaneOrigin origin;
  final OpaqueId deviceId;
  final AuthenticationEvidenceProvider authenticationEvidence;
  final ControlPlaneTransportFactory? transportFactory;
  final WidgetBuilder? touchWorkspaceBuilder;

  @override
  State<ControlPlaneShell> createState() => _ControlPlaneShellState();
}

final class _ControlPlaneShellState extends State<ControlPlaneShell> {
  late ControlPlaneApiRepository _repository;
  late ProjectCatalogController _projects;
  Route<void>? _activeProjectRoute;
  Route<void>? _activeWorkspaceRoute;
  SessionStartController? _activeSessions;
  var _runtimeGeneration = 0;

  @override
  void initState() {
    super.initState();
    _createRuntime();
  }

  @override
  void didUpdateWidget(ControlPlaneShell oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.origin.uri != widget.origin.uri ||
        oldWidget.deviceId != widget.deviceId ||
        !identical(
          oldWidget.authenticationEvidence,
          widget.authenticationEvidence,
        ) ||
        !identical(oldWidget.transportFactory, widget.transportFactory)) {
      _invalidateActiveRoutes();
      _projects.dispose();
      _repository.close();
      _createRuntime();
    }
  }

  void _createRuntime() {
    _runtimeGeneration++;
    final transport =
        widget.transportFactory?.call(widget.origin) ??
        IoControlPlaneTransport(origin: widget.origin);
    _repository = ControlPlaneApiRepository(
      origin: widget.origin,
      transport: transport,
      authenticationEvidence: widget.authenticationEvidence,
    );
    _projects = ProjectCatalogController(_repository);
  }

  @override
  Widget build(BuildContext context) {
    return ProjectCatalogScreen(
      controller: _projects,
      onProjectSelected: _openProject,
    );
  }

  void _openProject(ProjectSummary project) {
    _invalidateActiveRoutes();
    final generation = _runtimeGeneration;
    final sessions = SessionStartController(_repository);
    final route = MaterialPageRoute<void>(
      builder:
          (routeContext) => ProjectSessionScreen(
            project: project,
            deviceId: widget.deviceId,
            controller: sessions,
            onOpenTouchWorkspace: () {
              if (!mounted ||
                  generation != _runtimeGeneration ||
                  !identical(_activeSessions, sessions)) {
                return;
              }
              final workspaceRoute = MaterialPageRoute<void>(
                builder:
                    widget.touchWorkspaceBuilder ??
                    (_) => const TouchSpikeShell(),
              );
              _activeWorkspaceRoute = workspaceRoute;
              unawaited(
                Navigator.of(
                  routeContext,
                ).push<void>(workspaceRoute).whenComplete(() {
                  if (identical(_activeWorkspaceRoute, workspaceRoute)) {
                    _activeWorkspaceRoute = null;
                  }
                }),
              );
            },
          ),
    );
    _activeProjectRoute = route;
    _activeSessions = sessions;
    unawaited(
      Navigator.of(context).push<void>(route).whenComplete(() {
        _releaseProjectRoute(route, sessions);
      }),
    );
  }

  void _invalidateActiveRoutes() {
    final workspaceRoute = _activeWorkspaceRoute;
    _activeWorkspaceRoute = null;
    final projectRoute = _activeProjectRoute;
    final sessions = _activeSessions;
    _activeProjectRoute = null;
    _activeSessions = null;
    sessions?.dispose();

    if (workspaceRoute == null && projectRoute == null) {
      return;
    }
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final workspaceNavigator = workspaceRoute?.navigator;
      if (workspaceNavigator != null) {
        workspaceNavigator.removeRoute(workspaceRoute!);
      }
      final projectNavigator = projectRoute?.navigator;
      if (projectNavigator != null) {
        projectNavigator.removeRoute(projectRoute!);
      }
    });
  }

  void _releaseProjectRoute(
    Route<void> route,
    SessionStartController sessions,
  ) {
    if (!identical(_activeProjectRoute, route) ||
        !identical(_activeSessions, sessions)) {
      return;
    }
    _activeProjectRoute = null;
    _activeSessions = null;
    sessions.dispose();
  }

  @override
  void dispose() {
    _runtimeGeneration++;
    _invalidateActiveRoutes();
    _projects.dispose();
    _repository.close();
    super.dispose();
  }
}
