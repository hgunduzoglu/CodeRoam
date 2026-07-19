import 'dart:convert';

import 'package:coderoam/shared/domain/control_plane_timestamp.dart';
import 'package:coderoam/shared/domain/opaque_id.dart';

final class ProjectSummary {
  const ProjectSummary._({
    required this.id,
    required this.environmentId,
    required this.agentId,
    required this.name,
    required this.environmentName,
    required this.createdAt,
  });

  factory ProjectSummary.fromJson(Map<String, Object?> json) {
    const fields = {
      'id',
      'environmentId',
      'agentId',
      'name',
      'environmentName',
      'createdAt',
    };
    if (json.length != fields.length || !fields.every(json.containsKey)) {
      throw const FormatException('Project summary fields are invalid.');
    }
    final name = json['name'];
    final environmentName = json['environmentName'];
    if (name is! String || !_validDisplayName(name)) {
      throw const FormatException('Project name is invalid.');
    }
    if (environmentName is! String || !_validDisplayName(environmentName)) {
      throw const FormatException('Environment name is invalid.');
    }
    return ProjectSummary._(
      id: OpaqueId.parse(json['id']),
      environmentId: OpaqueId.parse(json['environmentId']),
      agentId: OpaqueId.parse(json['agentId']),
      name: name,
      environmentName: environmentName,
      createdAt: parseControlPlaneUtcTimestamp(json['createdAt']),
    );
  }

  final OpaqueId id;
  final OpaqueId environmentId;
  final OpaqueId agentId;
  final String name;
  final String environmentName;
  final DateTime createdAt;
}

bool _validDisplayName(String value) {
  if (value.isEmpty || value.length > 512 || value.trim() != value) {
    return false;
  }
  final runes = value.runes;
  return runes.length <= 128 &&
      utf8.encode(value).length <= 512 &&
      !runes.any((rune) => rune < 0x20 || (rune >= 0x7f && rune <= 0x9f));
}
