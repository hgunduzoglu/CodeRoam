import 'dart:async';
import 'dart:convert';

import 'package:coderoam/features/control_plane/presentation/control_plane_shell.dart';
import 'package:coderoam/shared/control_plane/control_plane_transport.dart';
import 'package:coderoam/shared/domain/opaque_id.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('owns the catalog to metadata-only local workspace flow', (
    tester,
  ) async {
    final transport = _ControlPlaneTransportStub();
    var evidenceCalls = 0;

    await tester.pumpWidget(
      MaterialApp(
        home: ControlPlaneShell(
          origin: _origin('control.example'),
          deviceId: _deviceId(),
          authenticationEvidence: () async {
            evidenceCalls++;
            return 'opaque-evidence';
          },
          transportFactory: (_) => transport,
          touchWorkspaceBuilder:
              (_) => const Scaffold(body: Text('local touch workspace')),
        ),
      ),
    );
    await tester.pump();

    expect(find.text('CodeRoam'), findsOneWidget);
    await tester.tap(find.text('CodeRoam'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Create session metadata'));
    await tester.pump();

    expect(
      find.textContaining('No relay capability was issued'),
      findsOneWidget,
    );
    expect(transport.requests, hasLength(2));
    expect(evidenceCalls, 2);
    expect(
      transport.requests.every(
        (request) =>
            request.headers['Authorization'] == 'Bearer opaque-evidence',
      ),
      isTrue,
    );

    await tester.tap(find.text('Open local touch workspace'));
    await tester.pumpAndSettle();
    expect(find.text('local touch workspace'), findsOneWidget);

    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump();
    expect(transport.closeCalls, 1);
  });

  testWidgets('rebuilds and closes runtime when trust inputs change', (
    tester,
  ) async {
    final first = _ControlPlaneTransportStub();
    final second = _ControlPlaneTransportStub();
    final firstOrigin = _origin('first.example');
    final secondOrigin = _origin('second.example');

    await tester.pumpWidget(
      MaterialApp(
        home: ControlPlaneShell(
          origin: firstOrigin,
          deviceId: _deviceId(),
          authenticationEvidence: _firstEvidence,
          transportFactory: (_) => first,
          touchWorkspaceBuilder: (_) => const SizedBox.shrink(),
        ),
      ),
    );
    await tester.pump();

    await tester.pumpWidget(
      MaterialApp(
        home: ControlPlaneShell(
          origin: secondOrigin,
          deviceId: _deviceId(),
          authenticationEvidence: _secondEvidence,
          transportFactory: (_) => second,
          touchWorkspaceBuilder: (_) => const SizedBox.shrink(),
        ),
      ),
    );
    await tester.pump();

    expect(first.closeCalls, 1);
    expect(first.requests.single.uri.host, 'first.example');
    expect(second.requests.single.uri.host, 'second.example');
    expect(
      second.requests.single.headers['Authorization'],
      'Bearer second-evidence',
    );
  });

  for (final mode in _TrustRotationMode.values) {
    testWidgets('removes $mode project state when trust inputs change', (
      tester,
    ) async {
      final first = _ControlPlaneTransportStub(mode: mode);
      final second = _ControlPlaneTransportStub();

      await tester.pumpWidget(
        MaterialApp(
          home: ControlPlaneShell(
            origin: _origin('first.example'),
            deviceId: _deviceId(),
            authenticationEvidence: _firstEvidence,
            transportFactory: (_) => first,
            touchWorkspaceBuilder:
                (_) => const Scaffold(body: Text('stale local workspace')),
          ),
        ),
      );
      await tester.pump();
      await tester.tap(find.text('CodeRoam'));
      await tester.pumpAndSettle();

      if (mode != _TrustRotationMode.idle) {
        await tester.tap(find.text('Create session metadata'));
        await tester.pump();
      }

      await tester.pumpWidget(
        MaterialApp(
          home: ControlPlaneShell(
            origin: _origin('second.example'),
            deviceId: _secondDeviceId(),
            authenticationEvidence: _secondEvidence,
            transportFactory: (_) => second,
            touchWorkspaceBuilder:
                (_) => const Scaffold(body: Text('current local workspace')),
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(first.closeCalls, 1);
      expect(find.text('Projects'), findsOneWidget);
      expect(find.text('Create session metadata'), findsNothing);
      expect(find.text('Retry same request'), findsNothing);
      expect(find.text('Open local touch workspace'), findsNothing);
      expect(find.text('stale local workspace'), findsNothing);

      if (mode == _TrustRotationMode.inFlight) {
        first.completeInFlightSession();
        await tester.pump();
        expect(find.text('stale local workspace'), findsNothing);
        expect(tester.takeException(), isNull);
      }

      await tester.tap(find.text('CodeRoam'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Create session metadata'));
      await tester.pump();

      final sessionRequest = second.requests.singleWhere(
        (request) => request.uri.path == '/v1/sessions',
      );
      final body =
          jsonDecode(utf8.decode(sessionRequest.body)) as Map<String, dynamic>;
      expect(sessionRequest.uri.host, 'second.example');
      expect(sessionRequest.headers['Authorization'], 'Bearer second-evidence');
      expect(body['deviceId'], _secondDeviceId().value);

      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pump();
      expect(second.closeCalls, 1);
    });
  }

  for (final state in _ShellDisposalState.values) {
    testWidgets('removes $state routes when the shell is replaced', (
      tester,
    ) async {
      final transport = _ControlPlaneTransportStub(
        mode:
            state == _ShellDisposalState.inFlight
                ? _TrustRotationMode.inFlight
                : _TrustRotationMode.ready,
      );

      await tester.pumpWidget(
        MaterialApp(
          home: ControlPlaneShell(
            origin: _origin('control.example'),
            deviceId: _deviceId(),
            authenticationEvidence: _firstEvidence,
            transportFactory: (_) => transport,
            touchWorkspaceBuilder:
                (_) => const Scaffold(body: Text('stale local workspace')),
          ),
        ),
      );
      await tester.pump();
      await tester.tap(find.text('CodeRoam'));
      await tester.pumpAndSettle();

      if (state != _ShellDisposalState.project) {
        await tester.tap(find.text('Create session metadata'));
        await tester.pump();
      }
      if (state == _ShellDisposalState.workspace) {
        await tester.tap(find.text('Open local touch workspace'));
        await tester.pumpAndSettle();
        expect(find.text('stale local workspace'), findsOneWidget);
      }

      await tester.pumpWidget(
        const MaterialApp(home: Scaffold(body: Text('signed out'))),
      );
      await tester.pumpAndSettle();

      expect(transport.closeCalls, 1);
      expect(find.text('signed out'), findsOneWidget);
      expect(find.text('Create session metadata'), findsNothing);
      expect(find.text('Open local touch workspace'), findsNothing);
      expect(find.text('stale local workspace'), findsNothing);

      if (state == _ShellDisposalState.inFlight) {
        transport.completeInFlightSession();
        await tester.pump();
        expect(find.text('stale local workspace'), findsNothing);
        expect(tester.takeException(), isNull);
      }
    });
  }
}

Future<String> _firstEvidence() async => 'first-evidence';
Future<String> _secondEvidence() async => 'second-evidence';

ControlPlaneOrigin _origin(String host) =>
    ControlPlaneOrigin.parse(Uri.parse('https://$host'));

OpaqueId _deviceId() => OpaqueId.parse('4123456789abcdef0123456789abcdef');

OpaqueId _secondDeviceId() =>
    OpaqueId.parse('5123456789abcdef0123456789abcdef');

enum _TrustRotationMode { idle, inFlight, outcomeUnknown, ready }

enum _ShellDisposalState { project, inFlight, metadataReady, workspace }

final class _ControlPlaneTransportStub implements ControlPlaneTransport {
  _ControlPlaneTransportStub({this.mode = _TrustRotationMode.ready});

  final _TrustRotationMode mode;
  final List<ControlPlaneHttpRequest> requests = [];
  final Completer<ControlPlaneHttpResponse> _inFlightSession = Completer();
  int closeCalls = 0;

  @override
  Future<ControlPlaneHttpResponse> send(ControlPlaneHttpRequest request) async {
    requests.add(request);
    if (request.uri.path == '/v1/projects') {
      return _jsonResponse({
        'projects': [
          {
            'id': '1123456789abcdef0123456789abcdef',
            'environmentId': '2123456789abcdef0123456789abcdef',
            'agentId': '3123456789abcdef0123456789abcdef',
            'name': 'CodeRoam',
            'environmentName': 'Development',
            'createdAt': '2026-07-20T00:00:00Z',
          },
        ],
      });
    }
    if (mode == _TrustRotationMode.inFlight) {
      return _inFlightSession.future;
    }
    if (mode == _TrustRotationMode.outcomeUnknown) {
      return _jsonResponse({
        'error': {'code': 'session_outcome_unknown'},
      }, statusCode: 503);
    }
    return _sessionSuccess(request);
  }

  void completeInFlightSession() {
    if (!_inFlightSession.isCompleted) {
      _inFlightSession.complete(_sessionSuccess(requests.last));
    }
  }

  ControlPlaneHttpResponse _sessionSuccess(ControlPlaneHttpRequest request) {
    final body = jsonDecode(utf8.decode(request.body)) as Map<String, dynamic>;
    return _jsonResponse({
      'id': body['sessionId'],
      'deviceId': body['deviceId'],
      'agentId': body['agentId'],
      'projectId': body['projectId'],
      'relayRegion': 'local',
      'startedAt': '2026-07-20T00:00:00Z',
      'capability': 'metadata-only',
    });
  }

  @override
  void close() {
    closeCalls++;
  }
}

ControlPlaneHttpResponse _jsonResponse(Object body, {int statusCode = 200}) =>
    ControlPlaneHttpResponse(
      statusCode: statusCode,
      contentType: 'application/json',
      body: utf8.encode(jsonEncode(body)),
    );
