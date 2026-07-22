import 'dart:collection';

import 'package:coderoam/features/workspace/application/project_catalog_repository.dart';
import 'package:coderoam/features/workspace/domain/project_summary.dart';
import 'package:coderoam/shared/domain/opaque_id.dart';
import 'package:flutter/foundation.dart';

enum ProjectCatalogStatus { idle, loading, ready, failed }

final class ProjectCatalogController extends ChangeNotifier {
  ProjectCatalogController(this._repository);

  final ProjectCatalogRepository _repository;
  List<ProjectSummary> _projects = const [];
  ProjectSummary? _selectedProject;
  ProjectCatalogStatus _status = ProjectCatalogStatus.idle;
  bool _disposed = false;

  ProjectCatalogStatus get status => _status;
  UnmodifiableListView<ProjectSummary> get projects =>
      UnmodifiableListView(_projects);
  ProjectSummary? get selectedProject => _selectedProject;

  Future<void> load() async {
    if (_disposed || _status == ProjectCatalogStatus.loading) {
      return;
    }
    _status = ProjectCatalogStatus.loading;
    notifyListeners();
    try {
      final projects = await _repository.listProjects();
      if (_disposed) {
        return;
      }
      if (projects.length > 100) {
        throw const FormatException('Project catalog is too large.');
      }
      _projects = List.unmodifiable(projects);
      if (_selectedProject case final selected?) {
        _selectedProject = _projectWithId(projects, selected.id);
      }
      _status = ProjectCatalogStatus.ready;
    } on Exception {
      if (_disposed) {
        return;
      }
      _projects = const [];
      _selectedProject = null;
      _status = ProjectCatalogStatus.failed;
    }
    notifyListeners();
  }

  bool select(OpaqueId projectId) {
    if (_disposed || _status != ProjectCatalogStatus.ready) {
      return false;
    }
    final project = _projectWithId(_projects, projectId);
    if (project == null || identical(project, _selectedProject)) {
      return project != null;
    }
    _selectedProject = project;
    notifyListeners();
    return true;
  }

  @override
  void dispose() {
    _disposed = true;
    super.dispose();
  }
}

ProjectSummary? _projectWithId(
  List<ProjectSummary> projects,
  OpaqueId projectId,
) {
  for (final project in projects) {
    if (project.id == projectId) {
      return project;
    }
  }
  return null;
}
