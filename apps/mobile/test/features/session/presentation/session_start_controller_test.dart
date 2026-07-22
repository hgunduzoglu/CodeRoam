import 'dart:async';

import 'package:coderoam/features/session/application/session_repository.dart';
import 'package:coderoam/features/session/domain/session_metadata.dart';
import 'package:coderoam/features/session/domain/session_start_request.dart';
import 'package:coderoam/features/session/presentation/session_start_controller.dart';
import 'package:coderoam/shared/domain/opaque_id.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('accepts only metadata exactly bound to the start request', () async {
    final repository = _SessionRepositoryStub(metadata: _metadata());
    final controller = SessionStartController(repository);
    addTearDown(controller.dispose);
    final statuses = <SessionStartStatus>[];
    controller.addListener(() => statuses.add(controller.status));

    expect(await controller.start(_request()), isTrue);

    expect(repository.calls, 1);
    expect(controller.status, SessionStartStatus.metadataReady);
    expect(controller.metadata, same(repository.metadata));
    expect(statuses, [
      SessionStartStatus.starting,
      SessionStartStatus.metadataReady,
    ]);

    repository.metadata = _metadata(
      projectId: '5123456789abcdef0123456789abcdef',
    );
    expect(await controller.start(_request()), isFalse);
    expect(controller.status, SessionStartStatus.failed);
    expect(controller.metadata, isNull);
  });

  test(
    'coalesces concurrent starts and ignores completion after dispose',
    () async {
      final completer = Completer<SessionMetadata>();
      final repository = _SessionRepositoryStub(completer: completer);
      final controller = SessionStartController(repository);

      final first = controller.start(_request());
      expect(await controller.start(_request()), isFalse);
      controller.dispose();
      completer.complete(_metadata());

      expect(await first, isFalse);
      expect(repository.calls, 1);
    },
  );

  test('maps repository details to a fixed failed state', () async {
    final controller = SessionStartController(
      _SessionRepositoryStub(
        error: const FormatException('provider leaked secret'),
      ),
    );
    addTearDown(controller.dispose);

    expect(await controller.start(_request()), isFalse);
    expect(controller.status, SessionStartStatus.failed);
    expect(controller.metadata, isNull);
  });

  test(
    'retries an unknown commit outcome with the exact same request',
    () async {
      final request = _request();
      final repository = _SessionRepositoryStub(
        error: SessionStartOutcomeUnknown(request),
      );
      final controller = SessionStartController(repository);
      addTearDown(controller.dispose);

      expect(await controller.start(request), isFalse);
      expect(controller.status, SessionStartStatus.outcomeUnknown);
      expect(await controller.start(_request()), isFalse);
      expect(repository.calls, 1);

      repository
        ..error = null
        ..metadata = _metadata();
      expect(await controller.retryOutcomeUnknown(), isTrue);
      expect(repository.calls, 2);
      expect(repository.requests[0], same(request));
      expect(repository.requests[1], same(request));
      expect(controller.status, SessionStartStatus.metadataReady);
    },
  );
}

final class _SessionRepositoryStub implements SessionRepository {
  _SessionRepositoryStub({this.metadata, this.error, this.completer});

  SessionMetadata? metadata;
  Exception? error;
  final Completer<SessionMetadata>? completer;
  int calls = 0;
  final List<SessionStartRequest> requests = [];

  @override
  Future<SessionMetadata> startSession(SessionStartRequest request) async {
    calls++;
    requests.add(request);
    if (error case final error?) {
      throw error;
    }
    if (completer case final completer?) {
      return completer.future;
    }
    return metadata!;
  }
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
