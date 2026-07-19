import 'package:coderoam/features/session/application/session_repository.dart';
import 'package:coderoam/features/session/domain/session_metadata.dart';
import 'package:coderoam/features/session/domain/session_start_request.dart';
import 'package:coderoam/features/session/presentation/project_session_screen.dart';
import 'package:coderoam/features/session/presentation/session_start_controller.dart';
import 'package:coderoam/features/workspace/domain/project_summary.dart';
import 'package:coderoam/shared/domain/opaque_id.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('creates metadata without presenting a relay capability', (
    tester,
  ) async {
    final repository = _SessionRepositoryStub(
      (request, _) async => _metadata(request),
    );
    final controller = SessionStartController(repository);
    addTearDown(controller.dispose);
    var opened = false;

    await tester.pumpWidget(
      MaterialApp(
        home: ProjectSessionScreen(
          project: _project(),
          deviceId: _deviceId(),
          controller: controller,
          onOpenTouchWorkspace: () => opened = true,
        ),
      ),
    );

    expect(find.textContaining('tickets arrive in M4'), findsOneWidget);
    await tester.tap(find.text('Create session metadata'));
    await tester.pump();

    expect(
      find.textContaining('No relay capability was issued'),
      findsOneWidget,
    );
    expect(find.textContaining('ticket', findRichText: true), findsOneWidget);
    expect(find.text(repository.requests.single.sessionId.value), findsNothing);

    await tester.tap(find.text('Open local touch workspace'));
    expect(opened, isTrue);
  });

  testWidgets('retries an unknown outcome with the same request', (
    tester,
  ) async {
    final repository = _SessionRepositoryStub((request, call) async {
      if (call == 1) {
        throw SessionStartOutcomeUnknown(request);
      }
      return _metadata(request);
    });
    final controller = SessionStartController(repository);
    addTearDown(controller.dispose);

    await tester.pumpWidget(
      MaterialApp(
        home: ProjectSessionScreen(
          project: _project(),
          deviceId: _deviceId(),
          controller: controller,
          onOpenTouchWorkspace: () {},
        ),
      ),
    );
    await tester.tap(find.text('Create session metadata'));
    await tester.pump();

    expect(find.text('Retry same request'), findsOneWidget);
    await tester.tap(find.text('Retry same request'));
    await tester.pump();

    expect(repository.requests, hasLength(2));
    expect(repository.requests[1], same(repository.requests[0]));
    expect(find.text('Open local touch workspace'), findsOneWidget);
  });

  testWidgets('uses a new id only after a known failure', (tester) async {
    final repository = _SessionRepositoryStub((request, call) async {
      if (call == 1) {
        throw const FormatException('secret detail');
      }
      return _metadata(request);
    });
    final controller = SessionStartController(repository);
    addTearDown(controller.dispose);

    await tester.pumpWidget(
      MaterialApp(
        home: ProjectSessionScreen(
          project: _project(),
          deviceId: _deviceId(),
          controller: controller,
          onOpenTouchWorkspace: () {},
        ),
      ),
    );
    await tester.tap(find.text('Create session metadata'));
    await tester.pump();

    expect(find.textContaining('secret'), findsNothing);
    await tester.tap(find.text('Try again with a new session'));
    await tester.pump();

    expect(repository.requests, hasLength(2));
    expect(
      repository.requests[1].sessionId,
      isNot(repository.requests[0].sessionId),
    );
  });
}

final class _SessionRepositoryStub implements SessionRepository {
  _SessionRepositoryStub(this._start);

  final Future<SessionMetadata> Function(SessionStartRequest, int) _start;
  final List<SessionStartRequest> requests = [];

  @override
  Future<SessionMetadata> startSession(SessionStartRequest request) {
    requests.add(request);
    return _start(request, requests.length);
  }
}

ProjectSummary _project() => ProjectSummary.fromJson({
  'id': '1123456789abcdef0123456789abcdef',
  'environmentId': '2123456789abcdef0123456789abcdef',
  'agentId': '3123456789abcdef0123456789abcdef',
  'name': 'CodeRoam',
  'environmentName': 'Development',
  'createdAt': '2026-07-20T00:00:00Z',
});

OpaqueId _deviceId() => OpaqueId.parse('4123456789abcdef0123456789abcdef');

SessionMetadata _metadata(SessionStartRequest request) =>
    SessionMetadata.fromJson({
      'id': request.sessionId.value,
      'deviceId': request.deviceId.value,
      'agentId': request.agentId.value,
      'projectId': request.projectId.value,
      'relayRegion': 'local',
      'startedAt': '2026-07-20T00:00:00Z',
      'capability': 'metadata-only',
    });
