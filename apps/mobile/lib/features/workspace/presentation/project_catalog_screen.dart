import 'package:coderoam/features/workspace/domain/project_summary.dart';
import 'package:coderoam/features/workspace/presentation/project_catalog_controller.dart';
import 'package:flutter/material.dart';

final class ProjectCatalogScreen extends StatefulWidget {
  const ProjectCatalogScreen({
    required this.controller,
    required this.onProjectSelected,
    super.key,
  });

  final ProjectCatalogController controller;
  final ValueChanged<ProjectSummary> onProjectSelected;

  @override
  State<ProjectCatalogScreen> createState() => _ProjectCatalogScreenState();
}

final class _ProjectCatalogScreenState extends State<ProjectCatalogScreen> {
  @override
  void initState() {
    super.initState();
    widget.controller.load();
  }

  @override
  void didUpdateWidget(ProjectCatalogScreen oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.controller != widget.controller) {
      widget.controller.load();
    }
  }

  @override
  Widget build(BuildContext context) {
    return ListenableBuilder(
      listenable: widget.controller,
      builder:
          (context, _) => Scaffold(
            appBar: AppBar(
              title: const Text('Projects'),
              actions: [
                IconButton(
                  tooltip: 'Refresh projects',
                  onPressed:
                      widget.controller.status == ProjectCatalogStatus.loading
                          ? null
                          : widget.controller.load,
                  icon: const Icon(Icons.refresh),
                ),
              ],
            ),
            body: SafeArea(
              child: _ProjectCatalogBody(
                controller: widget.controller,
                onProjectSelected: widget.onProjectSelected,
              ),
            ),
          ),
    );
  }
}

final class _ProjectCatalogBody extends StatelessWidget {
  const _ProjectCatalogBody({
    required this.controller,
    required this.onProjectSelected,
  });

  final ProjectCatalogController controller;
  final ValueChanged<ProjectSummary> onProjectSelected;

  @override
  Widget build(BuildContext context) {
    return switch (controller.status) {
      ProjectCatalogStatus.idle || ProjectCatalogStatus.loading => const Center(
        child: CircularProgressIndicator(),
      ),
      ProjectCatalogStatus.failed => Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              const Text(
                'Projects are unavailable right now.',
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: 16),
              FilledButton.icon(
                onPressed: controller.load,
                icon: const Icon(Icons.refresh),
                label: const Text('Try again'),
              ),
            ],
          ),
        ),
      ),
      ProjectCatalogStatus.ready when controller.projects.isEmpty =>
        const Center(
          child: Padding(
            padding: EdgeInsets.all(24),
            child: Text(
              'No projects are available yet.',
              textAlign: TextAlign.center,
            ),
          ),
        ),
      ProjectCatalogStatus.ready => Align(
        alignment: Alignment.topCenter,
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 720),
          child: ListView.separated(
            padding: const EdgeInsets.symmetric(vertical: 8),
            itemCount: controller.projects.length,
            separatorBuilder: (_, _) => const Divider(height: 1),
            itemBuilder: (context, index) {
              final project = controller.projects[index];
              return ListTile(
                minTileHeight: 56,
                selected: controller.selectedProject?.id == project.id,
                title: Text(project.name),
                subtitle: Text(project.environmentName),
                trailing: const Icon(Icons.chevron_right),
                onTap: () {
                  if (controller.select(project.id)) {
                    onProjectSelected(project);
                  }
                },
              );
            },
          ),
        ),
      ),
    };
  }
}
