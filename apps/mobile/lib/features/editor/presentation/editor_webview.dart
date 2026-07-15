import 'dart:async';

import 'package:coderoam/features/editor/presentation/editor_bridge_controller.dart';
import 'package:coderoam/features/editor/presentation/editor_spike_fixture.dart';
import 'package:coderoam/shared/webview/embedded_webview.dart';
import 'package:coderoam/shared/webview/webview_bridge.dart';
import 'package:flutter/foundation.dart';
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
  bool _hasStartedPage = false;

  @override
  void initState() {
    super.initState();
    _ownsController = widget.controller == null;
    _controller = widget.controller ?? EditorBridgeController();
    unawaited(_setInitialDocument());
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
        _controller.markReady();
      } else if (message.type == 'editor.focusChanged') {
        final focused = message.payload['focused'];
        if (kDebugMode) {
          debugPrint(
            '[CodeRoam Editor] focus changed: '
            '${focused is bool ? focused : 'invalid'}',
          );
        }
      } else if (message.type == 'editor.selectionChanged') {
        final anchor = message.payload['anchor'];
        final head = message.payload['head'];
        if (kDebugMode) {
          debugPrint(
            '[CodeRoam Editor] selection changed: '
            'anchor=${anchor is int ? anchor : 'invalid'}, '
            'head=${head is int ? head : 'invalid'}',
          );
        }
      } else if (message.type == 'editor.documentChanged') {
        final documentLength = message.payload['documentLength'];
        final insertedLength = message.payload['insertedLength'];
        final deletedLength = message.payload['deletedLength'];
        if (kDebugMode) {
          debugPrint(
            '[CodeRoam Editor] document changed: '
            'length=${documentLength is int ? documentLength : 'invalid'}, '
            'inserted=${insertedLength is int ? insertedLength : 'invalid'}, '
            'deleted=${deletedLength is int ? deletedLength : 'invalid'}',
          );
        }
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
        content: sampleEditorSpikeDocument,
      );
    } catch (_) {
      debugPrint('[CodeRoam Editor] Could not send initial document.');
    }
  }

  void _handlePageStarted() {
    _controller.markNotReady();

    if (_hasStartedPage) {
      unawaited(_restoreDocumentAfterReload());
    } else {
      _hasStartedPage = true;
    }
  }

  Future<void> _restoreDocumentAfterReload() async {
    try {
      await _controller.restoreLastSetDocument();
    } catch (_) {
      debugPrint('[CodeRoam Editor] Could not restore document after reload.');
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
      onPageStarted: _handlePageStarted,
    );
  }
}
