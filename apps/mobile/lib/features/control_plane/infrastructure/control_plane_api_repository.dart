import 'dart:convert';

import 'package:coderoam/features/session/application/session_repository.dart';
import 'package:coderoam/features/session/domain/session_metadata.dart';
import 'package:coderoam/features/session/domain/session_start_request.dart';
import 'package:coderoam/features/workspace/application/project_catalog_repository.dart';
import 'package:coderoam/features/workspace/domain/project_summary.dart';
import 'package:coderoam/shared/control_plane/control_plane_transport.dart';
import 'package:coderoam/shared/control_plane/strict_json.dart';

typedef AuthenticationEvidenceProvider = Future<String> Function();

final class ControlPlaneApiException implements Exception {
  const ControlPlaneApiException();
}

final class ControlPlaneApiRepository
    implements ProjectCatalogRepository, SessionRepository {
  ControlPlaneApiRepository({
    required ControlPlaneOrigin origin,
    required ControlPlaneTransport transport,
    required AuthenticationEvidenceProvider authenticationEvidence,
    Duration authenticationTimeout = const Duration(seconds: 10),
  }) : _origin = origin,
       _transport = transport,
       _authenticationEvidence = authenticationEvidence,
       _authenticationTimeout = authenticationTimeout {
    if (authenticationTimeout <= Duration.zero) {
      throw ArgumentError('Authentication timeout is invalid.');
    }
  }

  static const _maxResponseBytes = 64 * 1024;

  final ControlPlaneOrigin _origin;
  final ControlPlaneTransport _transport;
  final AuthenticationEvidenceProvider _authenticationEvidence;
  final Duration _authenticationTimeout;
  bool _closed = false;

  @override
  Future<List<ProjectSummary>> listProjects({int limit = 50}) async {
    if (limit < 1 || limit > 100) {
      throw const ControlPlaneApiException();
    }
    try {
      final response = await _sendAuthenticated(
        ControlPlaneHttpRequest(
          method: 'GET',
          uri: _origin.endpoint(
            '/v1/projects',
            queryParameters: {'limit': '$limit'},
          ),
        ),
      );
      if (response.statusCode != 200) {
        throw const ControlPlaneApiException();
      }
      final envelope = _decodeJsonObject(response);
      if (envelope.length != 1 || !envelope.containsKey('projects')) {
        throw const ControlPlaneApiException();
      }
      final encodedProjects = envelope['projects'];
      if (encodedProjects is! List<Object?> ||
          encodedProjects.length > limit ||
          encodedProjects.length > 100) {
        throw const ControlPlaneApiException();
      }
      final projects = <ProjectSummary>[];
      for (final encodedProject in encodedProjects) {
        if (encodedProject is! Map<String, Object?>) {
          throw const ControlPlaneApiException();
        }
        projects.add(ProjectSummary.fromJson(encodedProject));
      }
      return List.unmodifiable(projects);
    } on ControlPlaneApiException {
      rethrow;
    } catch (_) {
      throw const ControlPlaneApiException();
    }
  }

  @override
  Future<SessionMetadata> startSession(SessionStartRequest request) async {
    try {
      final response = await _sendAuthenticated(
        ControlPlaneHttpRequest(
          method: 'POST',
          uri: _origin.endpoint('/v1/sessions'),
          headers: const {'Content-Type': 'application/json'},
          body: utf8.encode(jsonEncode(request.toJson())),
        ),
        outcomeUnknownRequest: request,
      );
      if (response.statusCode == 503 &&
          _errorCode(response) == 'session_outcome_unknown') {
        throw SessionStartOutcomeUnknown(request);
      }
      if (response.statusCode != 200) {
        throw const ControlPlaneApiException();
      }
      try {
        final metadata = SessionMetadata.fromJson(_decodeJsonObject(response));
        if (!request.matches(metadata)) {
          throw const ControlPlaneApiException();
        }
        return metadata;
      } catch (_) {
        throw SessionStartOutcomeUnknown(request);
      }
    } on SessionStartOutcomeUnknown {
      rethrow;
    } on ControlPlaneApiException {
      rethrow;
    } catch (_) {
      throw const ControlPlaneApiException();
    }
  }

  void close() {
    if (_closed) {
      return;
    }
    _closed = true;
    _transport.close();
  }

  Future<ControlPlaneHttpResponse> _sendAuthenticated(
    ControlPlaneHttpRequest request, {
    SessionStartRequest? outcomeUnknownRequest,
  }) async {
    if (_closed) {
      throw const ControlPlaneApiException();
    }
    String evidence;
    try {
      evidence = await _authenticationEvidence().timeout(
        _authenticationTimeout,
      );
    } catch (_) {
      throw const ControlPlaneApiException();
    }
    if (_closed || !_validBearerEvidence(evidence)) {
      throw const ControlPlaneApiException();
    }
    final headers = <String, String>{
      ...request.headers,
      'Accept': 'application/json',
      'Authorization': 'Bearer $evidence',
    };
    ControlPlaneHttpResponse response;
    try {
      response = await _transport.send(
        ControlPlaneHttpRequest(
          method: request.method,
          uri: request.uri,
          headers: headers,
          body: request.body,
        ),
      );
    } catch (_) {
      if (outcomeUnknownRequest != null) {
        throw SessionStartOutcomeUnknown(outcomeUnknownRequest);
      }
      throw const ControlPlaneApiException();
    }
    if (_closed) {
      if (outcomeUnknownRequest != null) {
        throw SessionStartOutcomeUnknown(outcomeUnknownRequest);
      }
      throw const ControlPlaneApiException();
    }
    return response;
  }
}

Map<String, Object?> _decodeJsonObject(ControlPlaneHttpResponse response) {
  if (response.contentType != 'application/json' ||
      response.body.isEmpty ||
      response.body.length > ControlPlaneApiRepository._maxResponseBytes) {
    throw const ControlPlaneApiException();
  }
  final decoded = decodeStrictJson(
    utf8.decode(response.body, allowMalformed: false),
  );
  if (decoded is! Map<String, Object?>) {
    throw const ControlPlaneApiException();
  }
  return decoded;
}

String? _errorCode(ControlPlaneHttpResponse response) {
  try {
    final envelope = _decodeJsonObject(response);
    if (envelope.length != 1 || !envelope.containsKey('error')) {
      return null;
    }
    final error = envelope['error'];
    if (error is! Map<String, Object?> ||
        error.length != 1 ||
        !error.containsKey('code')) {
      return null;
    }
    return error['code'] is String ? error['code'] as String : null;
  } catch (_) {
    return null;
  }
}

bool _validBearerEvidence(String evidence) {
  return evidence.isNotEmpty &&
      evidence.length <= 8 * 1024 &&
      RegExp(r'^[A-Za-z0-9\-._~+/]+=*$').hasMatch(evidence);
}
