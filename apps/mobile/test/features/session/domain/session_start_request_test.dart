import 'package:coderoam/features/session/domain/session_metadata.dart';
import 'package:coderoam/features/session/domain/session_start_request.dart';
import 'package:coderoam/shared/domain/opaque_id.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('serializes only the four metadata request ids', () {
    final request = _request();

    expect(request.toJson(), {
      'sessionId': '1123456789abcdef0123456789abcdef',
      'deviceId': '2123456789abcdef0123456789abcdef',
      'agentId': '3123456789abcdef0123456789abcdef',
      'projectId': '4123456789abcdef0123456789abcdef',
    });
    expect(request.toJson(), isNot(contains('ticket')));
  });

  test('matches only metadata bound to the complete request', () {
    final request = _request();
    expect(request.matches(_metadata()), isTrue);
    expect(
      request.matches(_metadata(projectId: '5123456789abcdef0123456789abcdef')),
      isFalse,
    );
  });
}

SessionStartRequest _request() => SessionStartRequest(
  sessionId: OpaqueId.parse('1123456789abcdef0123456789abcdef'),
  deviceId: OpaqueId.parse('2123456789abcdef0123456789abcdef'),
  agentId: OpaqueId.parse('3123456789abcdef0123456789abcdef'),
  projectId: OpaqueId.parse('4123456789abcdef0123456789abcdef'),
);

SessionMetadata _metadata({
  String projectId = '4123456789abcdef0123456789abcdef',
}) => SessionMetadata.fromJson({
  'id': '1123456789abcdef0123456789abcdef',
  'deviceId': '2123456789abcdef0123456789abcdef',
  'agentId': '3123456789abcdef0123456789abcdef',
  'projectId': projectId,
  'relayRegion': 'local',
  'startedAt': '2026-07-20T00:00:00Z',
  'capability': 'metadata-only',
});
