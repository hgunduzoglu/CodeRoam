import 'dart:convert';

import 'package:webview_flutter/webview_flutter.dart';

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
    final type = decoded['type'];
    final payload = decoded['payload'];

    if (version != protocolVersion) {
      throw FormatException('Unsupported bridge version: $version');
    }

    if (type is! String || type.isEmpty) {
      throw const FormatException('Bridge message type is required.');
    }

    return WebViewBridgeMessage(
      id: decoded['id'] as String?,
      type: type,
      payload:
          payload is Map<String, dynamic>
              ? Map<String, Object?>.from(payload)
              : const {},
    );
  }
}

class WebViewBridgeController {
  WebViewBridgeController({required this.javascriptReceiver});

  final String javascriptReceiver;

  WebViewController? _webViewController;
  bool _ready = false;

  final List<WebViewBridgeMessage> _pendingMessages = [];

  void attach(WebViewController controller) {
    _webViewController = controller;
    _flush();
  }

  void markReady() {
    _ready = true;
    _flush();
  }

  Future<void> send(WebViewBridgeMessage message) async {
    if (!_ready || _webViewController == null) {
      _pendingMessages.add(message);
      return;
    }

    await _sendNow(message);
  }

  Future<void> _flush() async {
    if (!_ready || _webViewController == null) {
      return;
    }

    final messages = List<WebViewBridgeMessage>.from(_pendingMessages);
    _pendingMessages.clear();

    for (final message in messages) {
      await _sendNow(message);
    }
  }

  Future<void> _sendNow(WebViewBridgeMessage message) async {
    final encodedMessage = jsonEncode(message.toJson());

    await _webViewController!.runJavaScript(
      '$javascriptReceiver($encodedMessage);',
    );
  }
}
