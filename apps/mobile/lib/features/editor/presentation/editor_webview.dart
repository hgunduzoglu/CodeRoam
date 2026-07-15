import 'dart:async';

import 'package:coderoam/features/editor/presentation/editor_bridge_controller.dart';
import 'package:coderoam/shared/webview/embedded_webview.dart';
import 'package:coderoam/shared/webview/webview_bridge.dart';
import 'package:flutter/material.dart';

class EditorWebView extends StatefulWidget {
  const EditorWebView({this.controller, this.onMessage, super.key});

  final EditorBridgeController? controller;
  final ValueChanged<WebViewBridgeMessage>? onMessage;

  @override
  State<EditorWebView> createState() => _EditorWebViewState();
}

class _EditorWebViewState extends State<EditorWebView> {
  late final EditorBridgeController _controller;

  @override
  void initState() {
    super.initState();
    _controller = widget.controller ?? EditorBridgeController();
  }

  void _handleMessage(String rawMessage) {
    try {
      final message = WebViewBridgeMessage.decode(rawMessage);

      debugPrint('[CodeRoam Editor] ${message.type}: ${message.payload}');

      if (message.type == 'editor.ready') {
        _controller.markReady();

        unawaited(
          _controller.setDocument(
            language: 'typescript',
            content: '''
// Document supplied by Flutter through the typed bridge.

function greet(name: string): string {
  return `Welcome, \${name}`;
}

console.log(greet("CodeRoam"));
''',
          ),
        );
      }

      widget.onMessage?.call(message);
    } on FormatException catch (error) {
      debugPrint('[CodeRoam Editor] Invalid bridge message: $error');
    }
  }

  @override
  Widget build(BuildContext context) {
    return EmbeddedWebView(
      assetPath: 'assets/editor/index.html',
      javascriptChannel: 'CodeRoamEditor',
      backgroundColor: const Color(0xFF111318),
      onControllerCreated: _controller.attach,
      onMessage: _handleMessage,
    );
  }
}
