import 'package:coderoam/features/terminal/presentation/terminal_bridge_controller.dart';
import 'package:coderoam/shared/webview/embedded_webview.dart';
import 'package:coderoam/shared/webview/webview_bridge.dart';
import 'package:flutter/material.dart';

class TerminalWebView extends StatefulWidget {
  const TerminalWebView({
    this.controller,
    this.onMessage,
    this.onReady,
    super.key,
  });

  final TerminalBridgeController? controller;
  final ValueChanged<WebViewBridgeMessage>? onMessage;
  final VoidCallback? onReady;

  @override
  State<TerminalWebView> createState() => _TerminalWebViewState();
}

class _TerminalWebViewState extends State<TerminalWebView> {
  late final TerminalBridgeController _controller;
  late final bool _ownsController;

  @override
  void initState() {
    super.initState();
    _ownsController = widget.controller == null;
    _controller = widget.controller ?? TerminalBridgeController();
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

      if (message.type == 'terminal.ready') {
        if (_controller.markReady()) {
          widget.onReady?.call();
        }
      } else if (message.type == 'terminal.resized') {
        final columns = message.payload['columns'];
        final rows = message.payload['rows'];
        debugPrint(
          '[CodeRoam Terminal] resized: '
          'columns=${columns is int ? columns : 'invalid'}, '
          'rows=${rows is int ? rows : 'invalid'}',
        );
      } else if (message.type == 'terminal.focusChanged') {
        final focused = message.payload['focused'];
        debugPrint(
          '[CodeRoam Terminal] focus changed: '
          '${focused is bool ? focused : 'invalid'}',
        );
      } else if (message.type == 'terminal.error') {
        debugPrint('[CodeRoam Terminal] bridge error event received.');
      }

      widget.onMessage?.call(message);
    } on FormatException {
      debugPrint('[CodeRoam Terminal] Invalid bridge message received.');
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
      onPageStarted: _controller.markNotReady,
    );
  }
}
