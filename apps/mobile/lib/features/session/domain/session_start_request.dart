import 'package:coderoam/features/session/domain/session_metadata.dart';
import 'package:coderoam/shared/domain/opaque_id.dart';

final class SessionStartRequest {
  const SessionStartRequest({
    required this.sessionId,
    required this.deviceId,
    required this.agentId,
    required this.projectId,
  });

  final OpaqueId sessionId;
  final OpaqueId deviceId;
  final OpaqueId agentId;
  final OpaqueId projectId;

  Map<String, String> toJson() => {
    'sessionId': sessionId.value,
    'deviceId': deviceId.value,
    'agentId': agentId.value,
    'projectId': projectId.value,
  };

  bool matches(SessionMetadata metadata) =>
      metadata.id == sessionId &&
      metadata.deviceId == deviceId &&
      metadata.agentId == agentId &&
      metadata.projectId == projectId;
}
