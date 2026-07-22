import 'package:coderoam/features/session/domain/session_start_request.dart';
import 'package:coderoam/features/session/presentation/session_start_controller.dart';
import 'package:coderoam/features/workspace/domain/project_summary.dart';
import 'package:coderoam/shared/domain/opaque_id.dart';
import 'package:flutter/material.dart';

final class ProjectSessionScreen extends StatelessWidget {
  const ProjectSessionScreen({
    required this.project,
    required this.deviceId,
    required this.controller,
    required this.onOpenTouchWorkspace,
    super.key,
  });

  final ProjectSummary project;
  final OpaqueId deviceId;
  final SessionStartController controller;
  final VoidCallback onOpenTouchWorkspace;

  @override
  Widget build(BuildContext context) {
    return ListenableBuilder(
      listenable: controller,
      builder:
          (context, _) => Scaffold(
            appBar: AppBar(title: Text(project.name)),
            body: SafeArea(
              child: Center(
                child: ConstrainedBox(
                  constraints: const BoxConstraints(maxWidth: 560),
                  child: SingleChildScrollView(
                    padding: const EdgeInsets.all(24),
                    child: _SessionBody(
                      project: project,
                      deviceId: deviceId,
                      controller: controller,
                      onOpenTouchWorkspace: onOpenTouchWorkspace,
                    ),
                  ),
                ),
              ),
            ),
          ),
    );
  }
}

final class _SessionBody extends StatelessWidget {
  const _SessionBody({
    required this.project,
    required this.deviceId,
    required this.controller,
    required this.onOpenTouchWorkspace,
  });

  final ProjectSummary project;
  final OpaqueId deviceId;
  final SessionStartController controller;
  final VoidCallback onOpenTouchWorkspace;

  @override
  Widget build(BuildContext context) {
    final status = controller.status;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text(
          project.environmentName,
          style: Theme.of(context).textTheme.titleMedium,
        ),
        const SizedBox(height: 16),
        const Text(
          'M2 creates authorized session metadata only. Relay connection and tickets arrive in M4.',
        ),
        const SizedBox(height: 24),
        if (status == SessionStartStatus.starting)
          const Center(child: CircularProgressIndicator())
        else if (status == SessionStartStatus.outcomeUnknown) ...[
          const Text(
            'The session result is unknown. Retry the same request to reconcile it safely.',
          ),
          const SizedBox(height: 16),
          FilledButton.icon(
            onPressed: controller.retryOutcomeUnknown,
            icon: const Icon(Icons.refresh),
            label: const Text('Retry same request'),
          ),
        ] else if (status == SessionStartStatus.metadataReady) ...[
          const Text(
            'Session metadata is ready. No relay capability was issued.',
          ),
          const SizedBox(height: 16),
          FilledButton(
            onPressed: onOpenTouchWorkspace,
            child: const Text('Open local touch workspace'),
          ),
        ] else ...[
          if (status == SessionStartStatus.failed) ...[
            const Text('Session metadata is unavailable right now.'),
            const SizedBox(height: 16),
          ],
          FilledButton.icon(
            onPressed:
                () => controller.start(
                  SessionStartRequest(
                    sessionId: OpaqueId.generate(),
                    deviceId: deviceId,
                    agentId: project.agentId,
                    projectId: project.id,
                  ),
                ),
            icon: const Icon(Icons.play_arrow),
            label: Text(
              status == SessionStartStatus.failed
                  ? 'Try again with a new session'
                  : 'Create session metadata',
            ),
          ),
        ],
      ],
    );
  }
}
