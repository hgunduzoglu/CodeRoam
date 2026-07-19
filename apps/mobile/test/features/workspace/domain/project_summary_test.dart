import 'package:coderoam/features/workspace/domain/project_summary.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('parses a bounded project summary', () {
    final project = ProjectSummary.fromJson(_validProjectJson());

    expect(project.id.value, '1123456789abcdef0123456789abcdef');
    expect(project.environmentId.value, '2123456789abcdef0123456789abcdef');
    expect(project.agentId.value, '3123456789abcdef0123456789abcdef');
    expect(project.name, 'CodeRoam');
    expect(project.environmentName, 'Development');
    expect(project.createdAt, DateTime.utc(2026, 7, 20));
  });

  test('rejects unexpected or invalid project metadata', () {
    final invalid = <Map<String, Object?>>[
      {..._validProjectJson(), 'rootPath': '/secret/project'},
      {..._validProjectJson(), 'id': 'not-an-id'},
      {..._validProjectJson(), 'name': ' CodeRoam'},
      {..._validProjectJson(), 'name': 'Code\nRoam'},
      {..._validProjectJson(), 'name': 'a' * 129},
      {..._validProjectJson(), 'name': 'a' * 513},
      {..._validProjectJson(), 'name': 'a' * (2 * 1024 * 1024)},
      {..._validProjectJson(), 'name': '😀' * (1024 * 1024)},
      {..._validProjectJson(), 'environmentName': ''},
      {..._validProjectJson(), 'createdAt': '2026-07-20T00:00:00'},
      {..._validProjectJson(), 'createdAt': '2' * 10000},
      {..._validProjectJson()}..remove('agentId'),
    ];

    for (final json in invalid) {
      expect(() => ProjectSummary.fromJson(json), throwsFormatException);
    }
  });
}

Map<String, Object?> _validProjectJson() => {
  'id': '1123456789abcdef0123456789abcdef',
  'environmentId': '2123456789abcdef0123456789abcdef',
  'agentId': '3123456789abcdef0123456789abcdef',
  'name': 'CodeRoam',
  'environmentName': 'Development',
  'createdAt': '2026-07-20T00:00:00Z',
};
