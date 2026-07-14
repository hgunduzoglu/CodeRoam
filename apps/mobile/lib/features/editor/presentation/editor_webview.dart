import 'package:coderoam/shared/webview/embedded_webview.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';

class EditorWebView extends StatelessWidget {
  const EditorWebView({this.onMessage, super.key});

  final ValueChanged<String>? onMessage;

  @override
  Widget build(BuildContext context) {
    return EmbeddedWebView(
      assetPath: 'assets/editor/index.html',
      javascriptChannel: 'CodeRoamEditor',
      backgroundColor: const Color(0xFF111318),
      onMessage:
          onMessage ??
          (message) {
            debugPrint('[CodeRoam Editor] $message');
          },
    );
  }
}
