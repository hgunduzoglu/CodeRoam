import 'package:coderoam/features/workspace/domain/project_summary.dart';

abstract interface class ProjectCatalogRepository {
  Future<List<ProjectSummary>> listProjects({int limit = 50});
}
