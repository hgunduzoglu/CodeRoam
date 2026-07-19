import 'dart:async';

import 'package:coderoam/features/workspace/application/project_catalog_repository.dart';
import 'package:coderoam/features/workspace/domain/project_summary.dart';
import 'package:coderoam/features/workspace/presentation/project_catalog_controller.dart';
import 'package:coderoam/features/workspace/presentation/project_catalog_screen.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('loads and selects a project without exposing metadata ids', (
    tester,
  ) async {
    await tester.binding.setSurfaceSize(const Size(390, 844));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final project = _project();
    final controller = ProjectCatalogController(
      _RepositoryStub(result: [project]),
    );
    addTearDown(controller.dispose);
    ProjectSummary? selected;

    await tester.pumpWidget(
      MaterialApp(
        home: ProjectCatalogScreen(
          controller: controller,
          onProjectSelected: (project) => selected = project,
        ),
      ),
    );
    await tester.pump();

    expect(find.text('CodeRoam'), findsOneWidget);
    expect(find.text('Development'), findsOneWidget);
    expect(find.text(project.id.value), findsNothing);
    expect(find.text(project.agentId.value), findsNothing);

    await tester.tap(find.text('CodeRoam'));
    await tester.pump();

    expect(selected, same(project));
    expect(controller.selectedProject, same(project));
  });

  testWidgets('shows a fixed failure and retries explicitly', (tester) async {
    final repository = _RepositoryStub(
      error: const FormatException('backend secret detail'),
    );
    final controller = ProjectCatalogController(repository);
    addTearDown(controller.dispose);

    await tester.pumpWidget(
      MaterialApp(
        home: ProjectCatalogScreen(
          controller: controller,
          onProjectSelected: (_) {},
        ),
      ),
    );
    await tester.pump();

    expect(find.text('Projects are unavailable right now.'), findsOneWidget);
    expect(find.textContaining('secret'), findsNothing);

    repository
      ..error = null
      ..result = [_project()];
    await tester.tap(find.text('Try again'));
    await tester.pump();

    expect(repository.calls, 2);
    expect(find.text('CodeRoam'), findsOneWidget);
  });

  testWidgets('keeps loading and empty states usable on a short tablet', (
    tester,
  ) async {
    await tester.binding.setSurfaceSize(const Size(900, 260));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final completer = Completer<List<ProjectSummary>>();
    final controller = ProjectCatalogController(
      _RepositoryStub(completer: completer),
    );
    addTearDown(controller.dispose);

    await tester.pumpWidget(
      MaterialApp(
        home: ProjectCatalogScreen(
          controller: controller,
          onProjectSelected: (_) {},
        ),
      ),
    );

    expect(find.byType(CircularProgressIndicator), findsOneWidget);
    final refresh = tester.widget<IconButton>(
      find.ancestor(
        of: find.byIcon(Icons.refresh),
        matching: find.byType(IconButton),
      ),
    );
    expect(refresh.onPressed, isNull);

    completer.complete([]);
    await tester.pump();
    expect(find.text('No projects are available yet.'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });
}

final class _RepositoryStub implements ProjectCatalogRepository {
  _RepositoryStub({this.result = const [], this.error, this.completer});

  List<ProjectSummary> result;
  Exception? error;
  final Completer<List<ProjectSummary>>? completer;
  int calls = 0;

  @override
  Future<List<ProjectSummary>> listProjects({int limit = 50}) async {
    calls++;
    if (error case final error?) {
      throw error;
    }
    if (completer case final completer?) {
      return completer.future;
    }
    return result;
  }
}

ProjectSummary _project() => ProjectSummary.fromJson({
  'id': '1123456789abcdef0123456789abcdef',
  'environmentId': '2123456789abcdef0123456789abcdef',
  'agentId': '3123456789abcdef0123456789abcdef',
  'name': 'CodeRoam',
  'environmentName': 'Development',
  'createdAt': '2026-07-20T00:00:00Z',
});
