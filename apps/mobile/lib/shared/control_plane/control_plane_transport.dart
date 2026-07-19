import 'dart:async';
import 'dart:io';
import 'dart:typed_data';

final class ControlPlaneTransportException implements Exception {
  const ControlPlaneTransportException();
}

final class ControlPlaneHttpRequest {
  ControlPlaneHttpRequest({
    required this.method,
    required this.uri,
    Map<String, String> headers = const {},
    List<int> body = const [],
  }) : headers = Map.unmodifiable(headers),
       body = Uint8List.fromList(body);

  final String method;
  final Uri uri;
  final Map<String, String> headers;
  final Uint8List body;
}

final class ControlPlaneHttpResponse {
  ControlPlaneHttpResponse({
    required this.statusCode,
    required this.contentType,
    required List<int> body,
  }) : body = Uint8List.fromList(body);

  final int statusCode;
  final String? contentType;
  final Uint8List body;
}

abstract interface class ControlPlaneTransport {
  Future<ControlPlaneHttpResponse> send(ControlPlaneHttpRequest request);
  void close();
}

final class IoControlPlaneTransport implements ControlPlaneTransport {
  IoControlPlaneTransport({
    Duration timeout = const Duration(seconds: 10),
    int maxResponseBytes = 64 * 1024,
  }) : _timeout = timeout,
       _maxResponseBytes = maxResponseBytes,
       _client = HttpClient() {
    if (timeout <= Duration.zero || maxResponseBytes < 1) {
      _client.close(force: true);
      throw ArgumentError('Control-plane transport bounds are invalid.');
    }
    _client.connectionTimeout = timeout;
  }

  final Duration _timeout;
  final int _maxResponseBytes;
  final HttpClient _client;
  bool _closed = false;

  @override
  Future<ControlPlaneHttpResponse> send(ControlPlaneHttpRequest request) async {
    if (_closed || !_validRequest(request)) {
      throw const ControlPlaneTransportException();
    }
    final stopwatch = Stopwatch()..start();
    Duration remaining() {
      final duration = _timeout - stopwatch.elapsed;
      if (duration <= Duration.zero) {
        throw const ControlPlaneTransportException();
      }
      return duration;
    }

    HttpClientRequest? pendingRequest;
    try {
      pendingRequest = await _client
          .openUrl(request.method, request.uri)
          .timeout(remaining());
      pendingRequest
        ..followRedirects = false
        ..maxRedirects = 0;
      request.headers.forEach(pendingRequest.headers.set);
      if (request.body.isNotEmpty) {
        pendingRequest.add(request.body);
      }
      final response = await pendingRequest.close().timeout(remaining());
      if (response.contentLength > _maxResponseBytes) {
        pendingRequest.abort();
        throw const ControlPlaneTransportException();
      }
      final bytes = await _readBoundedResponse(
        response,
        remaining(),
        _maxResponseBytes,
      );
      return ControlPlaneHttpResponse(
        statusCode: response.statusCode,
        contentType: response.headers.contentType?.mimeType,
        body: bytes,
      );
    } on ControlPlaneTransportException {
      pendingRequest?.abort();
      rethrow;
    } on Exception {
      pendingRequest?.abort();
      throw const ControlPlaneTransportException();
    }
  }

  @override
  void close() {
    if (_closed) {
      return;
    }
    _closed = true;
    _client.close(force: true);
  }
}

Future<Uint8List> _readBoundedResponse(
  HttpClientResponse response,
  Duration timeout,
  int maxBytes,
) {
  final completer = Completer<Uint8List>();
  final bytes = BytesBuilder(copy: false);
  StreamSubscription<List<int>>? subscription;
  Timer? timer;

  void fail() {
    if (completer.isCompleted) {
      return;
    }
    timer?.cancel();
    subscription?.cancel();
    completer.completeError(const ControlPlaneTransportException());
  }

  timer = Timer(timeout, fail);
  subscription = response.listen(
    (chunk) {
      if (bytes.length + chunk.length > maxBytes) {
        fail();
        return;
      }
      bytes.add(chunk);
    },
    onError: (Object _, StackTrace _) => fail(),
    onDone: () {
      if (completer.isCompleted) {
        return;
      }
      timer?.cancel();
      completer.complete(bytes.takeBytes());
    },
    cancelOnError: true,
  );
  return completer.future;
}

bool _validRequest(ControlPlaneHttpRequest request) {
  if (request.method != 'GET' && request.method != 'POST') {
    return false;
  }
  final uri = request.uri;
  if (uri.toString().length > 2048 ||
      uri.scheme != 'https' ||
      uri.host.isEmpty ||
      uri.userInfo.isNotEmpty ||
      uri.fragment.isNotEmpty ||
      request.headers.length > 16 ||
      request.body.length > 16 * 1024) {
    return false;
  }
  final headerName = RegExp(r'^[A-Za-z0-9-]{1,64}$');
  const forbiddenHeaders = {
    'connection',
    'content-length',
    'host',
    'transfer-encoding',
  };
  for (final header in request.headers.entries) {
    if (!headerName.hasMatch(header.key) ||
        forbiddenHeaders.contains(header.key.toLowerCase()) ||
        header.value.length > 8192 ||
        header.value.runes.any((rune) => rune < 0x20 || rune == 0x7f)) {
      return false;
    }
  }
  return true;
}
