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
  late final bool _ownsController;

  @override
  void initState() {
    super.initState();
    _ownsController = widget.controller == null;
    _controller = widget.controller ?? EditorBridgeController();
  }

  @override
  void dispose() {
    if (_ownsController) {
      _controller.dispose();
    }
    super.dispose();
  }

  void _handleMessage(String rawMessage) {
    try {
      final message = WebViewBridgeMessage.decode(rawMessage);

      if (message.type == 'editor.ready') {
        if (_controller.markReady()) {
          unawaited(_setInitialDocument());
        }
      } else if (message.type == 'editor.focusChanged') {
        final focused = message.payload['focused'];
        debugPrint(
          '[CodeRoam Editor] focus changed: '
          '${focused is bool ? focused : 'invalid'}',
        );
      } else if (message.type == 'editor.selectionChanged') {
        final anchor = message.payload['anchor'];
        final head = message.payload['head'];
        debugPrint(
          '[CodeRoam Editor] selection changed: '
          'anchor=${anchor is int ? anchor : 'invalid'}, '
          'head=${head is int ? head : 'invalid'}',
        );
      } else if (message.type == 'editor.documentChanged') {
        final documentLength = message.payload['documentLength'];
        final insertedLength = message.payload['insertedLength'];
        final deletedLength = message.payload['deletedLength'];
        debugPrint(
          '[CodeRoam Editor] document changed: '
          'length=${documentLength is int ? documentLength : 'invalid'}, '
          'inserted=${insertedLength is int ? insertedLength : 'invalid'}, '
          'deleted=${deletedLength is int ? deletedLength : 'invalid'}',
        );
      } else if (message.type == 'editor.error') {
        debugPrint('[CodeRoam Editor] bridge error event received.');
      }

      widget.onMessage?.call(message);
    } on FormatException {
      debugPrint('[CodeRoam Editor] Invalid bridge message received.');
    }
  }

  Future<void> _setInitialDocument() async {
    try {
      await _controller.setDocument(
        language: 'typescript',
        content: '''
// Document supplied by Flutter through the typed bridge.

function greet(name: string): string {
  return `Welcome, \${name}`;
}

console.log(greet("CodeRoam"));
''',
      );
    } catch (_) {
      debugPrint('[CodeRoam Editor] Could not send initial document.');
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
      onPageStarted: _controller.markNotReady,
    );
  }
}
