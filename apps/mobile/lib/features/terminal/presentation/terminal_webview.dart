import 'package:coderoam/shared/webview/embedded_webview.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';

class TerminalWebView extends StatelessWidget {
  const TerminalWebView({this.onMessage, super.key});

  final ValueChanged<String>? onMessage;

  @override
  Widget build(BuildContext context) {
    return EmbeddedWebView(
      assetPath: 'assets/terminal/index.html',
      javascriptChannel: 'CodeRoamTerminal',
      backgroundColor: const Color(0xFF0D0F12),
      onMessage:
          onMessage ??
          (message) {
            debugPrint('[CodeRoam Terminal] $message');
          },
    );
  }
}
