import 'dart:async';
import 'dart:convert';

typedef JavaScriptRunner = Future<void> Function(String source);

class WebViewBridgeMessage {
  const WebViewBridgeMessage({
    required this.type,
    this.id,
    this.payload = const {},
  });

  static const int protocolVersion = 1;

  final String type;
  final String? id;
  final Map<String, Object?> payload;

  Map<String, Object?> toJson() {
    return {
      'version': protocolVersion,
      if (id != null) 'id': id,
      'type': type,
      'payload': payload,
    };
  }

  factory WebViewBridgeMessage.decode(String raw) {
    final decoded = jsonDecode(raw);

    if (decoded is! Map<String, dynamic>) {
      throw const FormatException('Bridge message must be a JSON object.');
    }

    final version = decoded['version'];
    final id = decoded['id'];
    final type = decoded['type'];
    final payload = decoded['payload'];

    if (version != protocolVersion) {
      throw FormatException('Unsupported bridge version: $version');
    }

    if (id != null && id is! String) {
      throw const FormatException('Bridge message id must be a string.');
    }

    if (type is! String || type.trim().isEmpty) {
      throw const FormatException('Bridge message type is required.');
    }

    if (decoded.containsKey('payload') && payload is! Map<String, dynamic>) {
      throw const FormatException('Bridge message payload must be an object.');
    }

    return WebViewBridgeMessage(
      id: id as String?,
      type: type,
      payload:
          payload is Map<String, dynamic>
              ? Map<String, Object?>.from(payload)
              : const {},
    );
  }
}

class WebViewBridgeController {
  WebViewBridgeController({
    required this.javascriptReceiver,
    this.maxPendingMessages = 256,
  }) {
    if (!_javascriptReceiverPattern.hasMatch(javascriptReceiver)) {
      throw ArgumentError.value(
        javascriptReceiver,
        'javascriptReceiver',
        'Must be a window function reference.',
      );
    }

    if (maxPendingMessages < 1) {
      throw ArgumentError.value(
        maxPendingMessages,
        'maxPendingMessages',
        'Must be greater than zero.',
      );
    }
  }

  static final RegExp _javascriptReceiverPattern = RegExp(
    r'^window\.[A-Za-z_$][A-Za-z0-9_$]*$',
  );

  final String javascriptReceiver;
  final int maxPendingMessages;

  JavaScriptRunner? _runJavaScript;
  bool _ready = false;
  bool _draining = false;
  bool _disposed = false;

  final List<_PendingBridgeMessage> _pendingMessages = [];

  void attach(JavaScriptRunner runJavaScript) {
    _checkNotDisposed();
    _runJavaScript = runJavaScript;
    _scheduleDrain();
  }

  bool markReady() {
    _checkNotDisposed();

    if (_ready) {
      return false;
    }

    _ready = true;
    _scheduleDrain();
    return true;
  }

  void markNotReady() {
    if (_disposed) {
      return;
    }

    _ready = false;
  }

  Future<void> send(WebViewBridgeMessage message) {
    if (_disposed) {
      return Future<void>.error(
        StateError('WebView bridge controller is disposed.'),
      );
    }

    if (_pendingMessages.length >= maxPendingMessages) {
      return Future<void>.error(
        StateError(
          'WebView bridge pending-message limit of '
          '$maxPendingMessages reached.',
        ),
      );
    }

    final pendingMessage = _PendingBridgeMessage(message);
    _pendingMessages.add(pendingMessage);
    _scheduleDrain();
    return pendingMessage.completer.future;
  }

  void _scheduleDrain() {
    if (_draining ||
        _disposed ||
        !_ready ||
        _runJavaScript == null ||
        _pendingMessages.isEmpty) {
      return;
    }

    _draining = true;
    unawaited(_drain());
  }

  Future<void> _drain() async {
    try {
      while (!_disposed &&
          _ready &&
          _runJavaScript != null &&
          _pendingMessages.isNotEmpty) {
        final pendingMessage = _pendingMessages.first;

        try {
          await _runJavaScript!(_javascriptFor(pendingMessage.message));
          _pendingMessages.remove(pendingMessage);
          if (!pendingMessage.completer.isCompleted) {
            pendingMessage.completer.complete();
          }
        } catch (error, stackTrace) {
          _pendingMessages.remove(pendingMessage);
          if (!pendingMessage.completer.isCompleted) {
            pendingMessage.completer.completeError(error, stackTrace);
          }
        }
      }
    } finally {
      _draining = false;
      _scheduleDrain();
    }
  }

  String _javascriptFor(WebViewBridgeMessage message) {
    final encodedMessage = jsonEncode(message.toJson());

    return '$javascriptReceiver($encodedMessage);';
  }

  void dispose() {
    if (_disposed) {
      return;
    }

    _disposed = true;
    _ready = false;
    _runJavaScript = null;

    final error = StateError('WebView bridge controller was disposed.');
    for (final pendingMessage in _pendingMessages) {
      if (!pendingMessage.completer.isCompleted) {
        pendingMessage.completer.completeError(error);
      }
    }
    _pendingMessages.clear();
  }

  void _checkNotDisposed() {
    if (_disposed) {
      throw StateError('WebView bridge controller is disposed.');
    }
  }
}

class _PendingBridgeMessage {
  _PendingBridgeMessage(this.message);

  final WebViewBridgeMessage message;
  final Completer<void> completer = Completer<void>();
}
