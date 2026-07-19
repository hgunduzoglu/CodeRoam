import 'dart:convert';
import 'dart:async';

import 'package:coderoam/features/control_plane/infrastructure/control_plane_api_repository.dart';
import 'package:coderoam/features/session/application/session_repository.dart';
import 'package:coderoam/features/session/domain/session_start_request.dart';
import 'package:coderoam/shared/control_plane/control_plane_transport.dart';
import 'package:coderoam/shared/domain/opaque_id.dart';
import 'package:flutter_test/flutter_test.dart';

final _origin = ControlPlaneOrigin.parse(Uri.parse('https://control.example'));

void main() {
  test(
    'lists bounded projects with provider-neutral bearer evidence',
    () async {
      final transport = _TransportStub(response: _projectsResponse());
      final repository = _repository(transport);
      addTearDown(repository.close);

      final projects = await repository.listProjects(limit: 25);

      expect(projects, hasLength(1));
      expect(projects.single.name, 'CodeRoam');
      expect(transport.requests, hasLength(1));
      final request = transport.requests.single;
      expect(request.method, 'GET');
      expect(
        request.uri,
        Uri.parse('https://control.example/v1/projects?limit=25'),
      );
      expect(request.headers['Authorization'], 'Bearer opaque-evidence');
      expect(request.headers['Accept'], 'application/json');
      expect(request.body, isEmpty);
    },
  );

  test('rejects malformed and oversized project responses', () async {
    final invalidResponses = <ControlPlaneHttpResponse>[
      _response(200, 'text/plain', utf8.encode('{}')),
      _jsonResponse(200, {'projects': [], 'rootPath': '/secret'}),
      _jsonResponse(200, {
        'projects': [
          {..._projectJson(), 'rootPath': '/secret'},
        ],
      }),
      _jsonResponse(200, {'projects': List.generate(2, (_) => _projectJson())}),
      _jsonResponse(503, {
        'error': {'code': 'projects_unavailable'},
      }),
      _response(200, 'application/json', List.filled(64 * 1024 + 1, 0)),
    ];

    for (final response in invalidResponses) {
      final transport = _TransportStub(response: response);
      final repository = _repository(transport);
      await expectLater(
        repository.listProjects(limit: 1),
        throwsA(isA<ControlPlaneApiException>()),
      );
      repository.close();
    }
  });

  test('starts and reconciles metadata-only sessions', () async {
    final transport = _TransportStub(response: _sessionResponse());
    final repository = _repository(transport);
    addTearDown(repository.close);
    final request = _sessionRequest();

    final metadata = await repository.startSession(request);

    expect(request.matches(metadata), isTrue);
    final sent = transport.requests.single;
    expect(sent.method, 'POST');
    expect(sent.uri, Uri.parse('https://control.example/v1/sessions'));
    expect(sent.headers['Content-Type'], 'application/json');
    expect(jsonDecode(utf8.decode(sent.body)), request.toJson());
    expect(utf8.decode(sent.body), isNot(contains('ticket')));
  });

  test(
    'preserves only an exact unknown commit outcome for safe retry',
    () async {
      final exact = _repository(
        _TransportStub(
          response: _jsonResponse(503, {
            'error': {'code': 'session_outcome_unknown'},
          }),
        ),
      );
      await expectLater(
        exact.startSession(_sessionRequest()),
        throwsA(isA<SessionStartOutcomeUnknown>()),
      );
      exact.close();

      final ambiguous = _repository(
        _TransportStub(
          response: _jsonResponse(503, {
            'error': {'code': 'session_outcome_unknown', 'detail': 'secret'},
          }),
        ),
      );
      await expectLater(
        ambiguous.startSession(_sessionRequest()),
        throwsA(isA<ControlPlaneApiException>()),
      );
      ambiguous.close();
    },
  );

  test('marks unusable successful session metadata outcome unknown', () async {
    final responses = [
      _sessionResponse(projectId: '5123456789abcdef0123456789abcdef'),
      _jsonResponse(200, {..._sessionJson(), 'ticket': 'forged'}),
      _jsonResponse(200, {..._sessionJson(), 'capability': 'relay-connect'}),
    ];
    for (final response in responses) {
      final request = _sessionRequest();
      final repository = _repository(_TransportStub(response: response));
      await expectLater(
        repository.startSession(request),
        throwsA(
          isA<SessionStartOutcomeUnknown>().having(
            (error) => error.request.sameAs(request),
            'same request',
            isTrue,
          ),
        ),
      );
      repository.close();
    }
  });

  test(
    'marks post-dispatch transport and close races outcome unknown',
    () async {
      final request = _sessionRequest();
      final failingTransport = _TransportStub(
        error: const ControlPlaneTransportException(),
      );
      final failingRepository = _repository(failingTransport);
      await expectLater(
        failingRepository.startSession(request),
        throwsA(
          isA<SessionStartOutcomeUnknown>().having(
            (error) => error.request.sameAs(request),
            'same request',
            isTrue,
          ),
        ),
      );
      expect(failingTransport.requests, hasLength(1));
      failingRepository.close();

      final response = Completer<ControlPlaneHttpResponse>();
      final delayedTransport = _TransportStub(completer: response);
      final delayedRepository = _repository(delayedTransport);
      final delayedStart = delayedRepository.startSession(request);
      await delayedTransport.dispatched.future;
      delayedRepository.close();
      response.complete(_sessionResponse());
      await expectLater(
        delayedStart,
        throwsA(
          isA<SessionStartOutcomeUnknown>().having(
            (error) => error.request.sameAs(request),
            'same request',
            isTrue,
          ),
        ),
      );
      expect(delayedTransport.requests, hasLength(1));
    },
  );

  test('rejects duplicate JSON keys including escaped aliases', () async {
    for (final response in [
      _rawJsonResponse(200, '{"projects":[],"projects":[]}'),
      _rawJsonResponse(
        200,
        '{"projects":[{'
        '"id":"1123456789abcdef0123456789abcdef",'
        '"id":"2123456789abcdef0123456789abcdef",'
        '"environmentId":"2123456789abcdef0123456789abcdef",'
        '"agentId":"3123456789abcdef0123456789abcdef",'
        '"name":"CodeRoam","environmentName":"Development",'
        '"createdAt":"2026-07-20T00:00:00Z"}]}',
      ),
    ]) {
      final repository = _repository(_TransportStub(response: response));
      await expectLater(
        repository.listProjects(),
        throwsA(isA<ControlPlaneApiException>()),
      );
      repository.close();
    }

    final request = _sessionRequest();
    for (final response in [
      _rawJsonResponse(
        200,
        jsonEncode(_sessionJson()).replaceFirst(
          '"capability":"metadata-only"',
          r'"capability":"relay-connect","cap\u0061bility":"metadata-only"',
        ),
      ),
      _rawJsonResponse(
        200,
        jsonEncode(_sessionJson()).replaceFirst(
          '"projectId":"1123456789abcdef0123456789abcdef"',
          '"projectId":"5123456789abcdef0123456789abcdef",'
              '"projectId":"1123456789abcdef0123456789abcdef"',
        ),
      ),
    ]) {
      final repository = _repository(_TransportStub(response: response));
      await expectLater(
        repository.startSession(request),
        throwsA(isA<SessionStartOutcomeUnknown>()),
      );
      repository.close();
    }

    for (final response in [
      _rawJsonResponse(
        503,
        '{"error":{"code":"sessions_unavailable"},'
        '"error":{"code":"session_outcome_unknown"}}',
      ),
      _rawJsonResponse(
        503,
        '{"error":{"code":"sessions_unavailable",'
        '"code":"session_outcome_unknown"}}',
      ),
    ]) {
      final repository = _repository(_TransportStub(response: response));
      await expectLater(
        repository.startSession(request),
        throwsA(isA<ControlPlaneApiException>()),
      );
      repository.close();
    }
  });

  test('rejects malformed evidence before transport and closes once', () async {
    for (final evidence in [
      '',
      '=',
      'Bearer nested',
      'opaque\nevidence',
      'opaque=after',
      'a' * (8 * 1024 + 1),
    ]) {
      final transport = _TransportStub(response: _projectsResponse());
      final repository = ControlPlaneApiRepository(
        origin: _origin,
        transport: transport,
        authenticationEvidence: () async => evidence,
      );
      await expectLater(
        repository.listProjects(),
        throwsA(isA<ControlPlaneApiException>()),
      );
      expect(transport.requests, isEmpty);
      repository
        ..close()
        ..close();
      expect(transport.closeCalls, 1);
    }
  });

  test(
    'accepts the bounded authentication evidence contract maximum',
    () async {
      final evidence = 'a' * (8 * 1024);
      final transport = _TransportStub(response: _projectsResponse());
      final repository = ControlPlaneApiRepository(
        origin: _origin,
        transport: transport,
        authenticationEvidence: () async => evidence,
      );
      addTearDown(repository.close);

      await repository.listProjects();

      expect(
        transport.requests.single.headers['Authorization'],
        'Bearer $evidence',
      );
    },
  );

  test('fails closed after close without requesting authentication', () async {
    var evidenceCalls = 0;
    final transport = _TransportStub(response: _projectsResponse());
    final repository = ControlPlaneApiRepository(
      origin: _origin,
      transport: transport,
      authenticationEvidence: () async {
        evidenceCalls++;
        return 'opaque-evidence';
      },
    );
    repository.close();

    await expectLater(
      repository.listProjects(),
      throwsA(isA<ControlPlaneApiException>()),
    );
    expect(evidenceCalls, 0);
    expect(transport.requests, isEmpty);
  });

  test('bounds authentication evidence lookup', () async {
    final evidence = Completer<String>();
    final repository = ControlPlaneApiRepository(
      origin: _origin,
      transport: _TransportStub(response: _projectsResponse()),
      authenticationEvidence: () => evidence.future,
      authenticationTimeout: const Duration(milliseconds: 1),
    );
    addTearDown(repository.close);

    await expectLater(
      repository.listProjects(),
      throwsA(isA<ControlPlaneApiException>()),
    );
    expect(
      () => ControlPlaneApiRepository(
        origin: _origin,
        transport: _TransportStub(response: _projectsResponse()),
        authenticationEvidence: () async => 'evidence',
        authenticationTimeout: Duration.zero,
      ),
      throwsArgumentError,
    );
  });
}

final class _TransportStub implements ControlPlaneTransport {
  _TransportStub({this.response, this.error, this.completer});

  final ControlPlaneHttpResponse? response;
  final Object? error;
  final Completer<ControlPlaneHttpResponse>? completer;
  final List<ControlPlaneHttpRequest> requests = [];
  final Completer<void> dispatched = Completer<void>();
  int closeCalls = 0;

  @override
  Future<ControlPlaneHttpResponse> send(ControlPlaneHttpRequest request) async {
    requests.add(request);
    if (!dispatched.isCompleted) {
      dispatched.complete();
    }
    if (error case final error?) {
      throw error;
    }
    if (completer case final completer?) {
      return completer.future;
    }
    final value = response;
    if (value == null) {
      throw StateError('Transport stub response is required.');
    }
    return value;
  }

  @override
  void close() {
    closeCalls++;
  }
}

ControlPlaneApiRepository _repository(ControlPlaneTransport transport) =>
    ControlPlaneApiRepository(
      origin: _origin,
      transport: transport,
      authenticationEvidence: () async => 'opaque-evidence',
    );

ControlPlaneHttpResponse _projectsResponse() => _jsonResponse(200, {
  'projects': [_projectJson()],
});

Map<String, Object?> _projectJson() => {
  'id': '1123456789abcdef0123456789abcdef',
  'environmentId': '2123456789abcdef0123456789abcdef',
  'agentId': '3123456789abcdef0123456789abcdef',
  'name': 'CodeRoam',
  'environmentName': 'Development',
  'createdAt': '2026-07-20T00:00:00Z',
};

SessionStartRequest _sessionRequest() => SessionStartRequest(
  sessionId: OpaqueId.parse('6123456789abcdef0123456789abcdef'),
  deviceId: OpaqueId.parse('7123456789abcdef0123456789abcdef'),
  agentId: OpaqueId.parse('3123456789abcdef0123456789abcdef'),
  projectId: OpaqueId.parse('1123456789abcdef0123456789abcdef'),
);

ControlPlaneHttpResponse _sessionResponse({
  String projectId = '1123456789abcdef0123456789abcdef',
}) => _jsonResponse(200, _sessionJson(projectId: projectId));

Map<String, Object?> _sessionJson({
  String projectId = '1123456789abcdef0123456789abcdef',
}) => {
  'id': '6123456789abcdef0123456789abcdef',
  'deviceId': '7123456789abcdef0123456789abcdef',
  'agentId': '3123456789abcdef0123456789abcdef',
  'projectId': projectId,
  'relayRegion': 'local',
  'startedAt': '2026-07-20T00:00:00Z',
  'capability': 'metadata-only',
};

ControlPlaneHttpResponse _jsonResponse(int status, Object body) =>
    _response(status, 'application/json', utf8.encode(jsonEncode(body)));

ControlPlaneHttpResponse _rawJsonResponse(int status, String body) =>
    _response(status, 'application/json', utf8.encode(body));

ControlPlaneHttpResponse _response(
  int status,
  String? contentType,
  List<int> body,
) => ControlPlaneHttpResponse(
  statusCode: status,
  contentType: contentType,
  body: body,
);
