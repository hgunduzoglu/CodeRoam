import 'package:coderoam/shared/domain/control_plane_timestamp.dart';
import 'package:coderoam/shared/domain/opaque_id.dart';

final class SessionMetadata {
  const SessionMetadata._({
    required this.id,
    required this.deviceId,
    required this.agentId,
    required this.projectId,
    required this.relayRegion,
    required this.startedAt,
  });

  factory SessionMetadata.fromJson(Map<String, Object?> json) {
    const fields = {
      'id',
      'deviceId',
      'agentId',
      'projectId',
      'relayRegion',
      'startedAt',
      'capability',
    };
    if (json.length != fields.length || !fields.every(json.containsKey)) {
      throw const FormatException('Session metadata fields are invalid.');
    }
    final relayRegion = json['relayRegion'];
    if (relayRegion is! String ||
        relayRegion.length > 64 ||
        !RegExp(
          r'^[a-z0-9](?:[a-z0-9-]{0,62}[a-z0-9])?$',
        ).hasMatch(relayRegion)) {
      throw const FormatException('Relay region is invalid.');
    }
    if (json['capability'] != 'metadata-only') {
      throw const FormatException('Session capability is invalid.');
    }
    return SessionMetadata._(
      id: OpaqueId.parse(json['id']),
      deviceId: OpaqueId.parse(json['deviceId']),
      agentId: OpaqueId.parse(json['agentId']),
      projectId: OpaqueId.parse(json['projectId']),
      relayRegion: relayRegion,
      startedAt: parseControlPlaneUtcTimestamp(json['startedAt']),
    );
  }

  final OpaqueId id;
  final OpaqueId deviceId;
  final OpaqueId agentId;
  final OpaqueId projectId;
  final String relayRegion;
  final DateTime startedAt;
}
