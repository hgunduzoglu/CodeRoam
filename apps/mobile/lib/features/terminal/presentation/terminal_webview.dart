import 'dart:async';

import 'package:coderoam/features/terminal/presentation/terminal_bridge_controller.dart';
import 'package:coderoam/shared/webview/embedded_webview.dart';
import 'package:coderoam/shared/webview/webview_bridge.dart';
import 'package:flutter/material.dart';

class TerminalWebView extends StatefulWidget {
  const TerminalWebView({this.controller, this.onMessage, super.key});

  final TerminalBridgeController? controller;
  final ValueChanged<WebViewBridgeMessage>? onMessage;

  @override
  State<TerminalWebView> createState() => _TerminalWebViewState();
}

class _TerminalWebViewState extends State<TerminalWebView> {
  late final TerminalBridgeController _controller;

  @override
  void initState() {
    super.initState();
    _controller = widget.controller ?? TerminalBridgeController();
  }

  void _handleMessage(String rawMessage) {
    try {
      final message = WebViewBridgeMessage.decode(rawMessage);

      debugPrint('[CodeRoam Terminal] ${message.type}: ${message.payload}');

      if (message.type == 'terminal.ready') {
        _controller.markReady();

        unawaited(
          _controller.write(
            '\r\nFlutter ↔ xterm.js bridge connected.\r\n\r\n\$ ',
          ),
        );
      }

      widget.onMessage?.call(message);
    } on FormatException catch (error) {
      debugPrint('[CodeRoam Terminal] Invalid bridge message: $error');
    }
  }

  @override
  Widget build(BuildContext context) {
    return EmbeddedWebView(
      assetPath: 'assets/terminal/index.html',
      javascriptChannel: 'CodeRoamTerminal',
      backgroundColor: const Color(0xFF0D0F12),
      onControllerCreated: _controller.attach,
      onMessage: _handleMessage,
    );
  }
}
