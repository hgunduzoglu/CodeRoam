import 'package:coderoam/features/session/application/session_repository.dart';
import 'package:coderoam/features/session/domain/session_metadata.dart';
import 'package:coderoam/features/session/domain/session_start_request.dart';
import 'package:flutter/foundation.dart';

enum SessionStartStatus {
  idle,
  starting,
  metadataReady,
  outcomeUnknown,
  failed,
}

final class SessionStartController extends ChangeNotifier {
  SessionStartController(this._repository);

  final SessionRepository _repository;
  SessionStartStatus _status = SessionStartStatus.idle;
  SessionMetadata? _metadata;
  SessionStartRequest? _outcomeUnknownRequest;
  bool _disposed = false;

  SessionStartStatus get status => _status;
  SessionMetadata? get metadata => _metadata;

  Future<bool> start(SessionStartRequest request) async {
    if (_disposed ||
        _status == SessionStartStatus.starting ||
        _status == SessionStartStatus.outcomeUnknown) {
      return false;
    }
    return _start(request);
  }

  Future<bool> retryOutcomeUnknown() async {
    final request = _outcomeUnknownRequest;
    if (_disposed ||
        _status != SessionStartStatus.outcomeUnknown ||
        request == null) {
      return false;
    }
    return _start(request);
  }

  Future<bool> _start(SessionStartRequest request) async {
    _metadata = null;
    _status = SessionStartStatus.starting;
    notifyListeners();
    try {
      final metadata = await _repository.startSession(request);
      if (_disposed) {
        return false;
      }
      if (!request.matches(metadata)) {
        throw const FormatException('Session metadata does not match request.');
      }
      _metadata = metadata;
      _outcomeUnknownRequest = null;
      _status = SessionStartStatus.metadataReady;
      notifyListeners();
      return true;
    } on SessionStartOutcomeUnknown catch (failure) {
      if (_disposed) {
        return false;
      }
      if (!request.sameAs(failure.request)) {
        _metadata = null;
        _outcomeUnknownRequest = null;
        _status = SessionStartStatus.failed;
        notifyListeners();
        return false;
      }
      _metadata = null;
      _outcomeUnknownRequest = request;
      _status = SessionStartStatus.outcomeUnknown;
      notifyListeners();
      return false;
    } on Exception {
      if (_disposed) {
        return false;
      }
      _metadata = null;
      _outcomeUnknownRequest = null;
      _status = SessionStartStatus.failed;
      notifyListeners();
      return false;
    }
  }

  @override
  void dispose() {
    _disposed = true;
    super.dispose();
  }
}
