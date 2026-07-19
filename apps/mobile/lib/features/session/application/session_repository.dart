import 'package:coderoam/features/session/domain/session_metadata.dart';
import 'package:coderoam/features/session/domain/session_start_request.dart';

final class SessionStartOutcomeUnknown implements Exception {
  const SessionStartOutcomeUnknown(this.request);

  final SessionStartRequest request;
}

abstract interface class SessionRepository {
  Future<SessionMetadata> startSession(SessionStartRequest request);
}
