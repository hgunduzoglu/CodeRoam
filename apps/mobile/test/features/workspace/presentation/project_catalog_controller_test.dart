import 'dart:async';

import 'package:coderoam/features/workspace/application/project_catalog_repository.dart';
import 'package:coderoam/features/workspace/domain/project_summary.dart';
import 'package:coderoam/features/workspace/presentation/project_catalog_controller.dart';
import 'package:coderoam/shared/domain/opaque_id.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('loads a bounded immutable project catalog', () async {
    final project = _project(idPrefix: '1');
    final repository = _ProjectCatalogRepositoryStub(result: [project]);
    final controller = ProjectCatalogController(repository);
    addTearDown(controller.dispose);
    final statuses = <ProjectCatalogStatus>[];
    controller.addListener(() => statuses.add(controller.status));

    await controller.load();

    expect(repository.calls, 1);
    expect(controller.status, ProjectCatalogStatus.ready);
    expect(controller.projects, [project]);
    expect(() => controller.projects.add(project), throwsUnsupportedError);
    expect(statuses, [
      ProjectCatalogStatus.loading,
      ProjectCatalogStatus.ready,
    ]);
  });

  test(
    'coalesces concurrent loads and ignores completion after dispose',
    () async {
      final completer = Completer<List<ProjectSummary>>();
      final repository = _ProjectCatalogRepositoryStub(completer: completer);
      final controller = ProjectCatalogController(repository);

      final first = controller.load();
      await controller.load();
      controller.dispose();
      completer.complete([_project(idPrefix: '1')]);
      await first;

      expect(repository.calls, 1);
    },
  );

  test(
    'maps repository failures and oversized results to fixed failed state',
    () async {
      final failing = ProjectCatalogController(
        _ProjectCatalogRepositoryStub(error: const FormatException('secret')),
      );
      addTearDown(failing.dispose);
      await failing.load();
      expect(failing.status, ProjectCatalogStatus.failed);
      expect(failing.projects, isEmpty);

      final oversized = ProjectCatalogController(
        _ProjectCatalogRepositoryStub(
          result: List.generate(101, (index) => _project(idPrefix: '1')),
        ),
      );
      addTearDown(oversized.dispose);
      await oversized.load();
      expect(oversized.status, ProjectCatalogStatus.failed);
      expect(oversized.projects, isEmpty);
    },
  );

  test(
    'selects only listed projects and preserves a valid selection on reload',
    () async {
      final first = _project(idPrefix: '1');
      final second = _project(idPrefix: '2');
      final repository = _ProjectCatalogRepositoryStub(result: [first, second]);
      final controller = ProjectCatalogController(repository);
      addTearDown(controller.dispose);
      await controller.load();

      expect(controller.select(second.id), isTrue);
      expect(controller.selectedProject, same(second));
      expect(
        controller.select(OpaqueId.parse('9123456789abcdef0123456789abcdef')),
        isFalse,
      );

      final refreshedSecond = _project(
        idPrefix: '2',
        name: 'CodeRoam refreshed',
      );
      repository.result = [refreshedSecond];
      await controller.load();
      expect(controller.selectedProject, same(refreshedSecond));

      repository.result = [_project(idPrefix: '3')];
      await controller.load();
      expect(controller.selectedProject, isNull);
    },
  );
}

final class _ProjectCatalogRepositoryStub implements ProjectCatalogRepository {
  _ProjectCatalogRepositoryStub({
    this.result = const [],
    this.error,
    this.completer,
  });

  List<ProjectSummary> result;
  final Exception? error;
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

ProjectSummary _project({required String idPrefix, String name = 'CodeRoam'}) {
  return ProjectSummary.fromJson({
    'id': '${idPrefix}123456789abcdef0123456789abcdef',
    'environmentId': '4123456789abcdef0123456789abcdef',
    'agentId': '5123456789abcdef0123456789abcdef',
    'name': name,
    'environmentName': 'Development',
    'createdAt': '2026-07-20T00:00:00Z',
  });
}
