import 'dart:async';

import 'package:coderoam/features/terminal/presentation/terminal_bridge_controller.dart';
import 'package:coderoam/features/terminal/presentation/terminal_input_event_deduplicator.dart';
import 'package:coderoam/features/terminal/presentation/terminal_selection_bridge.dart';
import 'package:coderoam/shared/webview/embedded_webview.dart';
import 'package:coderoam/shared/webview/webview_bridge.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

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
  final TerminalInputEventDeduplicator _inputEventDeduplicator =
      TerminalInputEventDeduplicator();

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
        final streamId = message.payload['streamId'];
        if (streamId is! String ||
            !_inputEventDeduplicator.beginStream(streamId)) {
          debugPrint('[CodeRoam Terminal] Invalid ready event ignored.');
          return;
        }

        if (_controller.markReady()) {
          widget.onReady?.call();
        }
      } else if (message.type == 'terminal.input' ||
          message.type == 'terminal.copySelection') {
        final eventId = message.id;
        final streamId = message.payload['streamId'];
        if (eventId == null ||
            streamId is! String ||
            !_inputEventDeduplicator.accept(
              streamId: streamId,
              eventId: eventId,
            )) {
          if (kDebugMode) {
            debugPrint(
              '[CodeRoam Terminal] Invalid, duplicate, or stale stream event '
              'ignored.',
            );
          }
          return;
        }

        if (message.type == 'terminal.copySelection') {
          final selection = terminalSelectionText(message);
          if (selection == null) {
            if (kDebugMode) {
              debugPrint(
                '[CodeRoam Terminal] Invalid terminal selection ignored.',
              );
            }
            return;
          }

          unawaited(_copySelection(selection));
          return;
        }
      } else if (message.type == 'terminal.resized') {
        final columns = message.payload['columns'];
        final rows = message.payload['rows'];
        if (kDebugMode) {
          debugPrint(
            '[CodeRoam Terminal] resized: '
            'columns=${columns is int ? columns : 'invalid'}, '
            'rows=${rows is int ? rows : 'invalid'}',
          );
        }
      } else if (message.type == 'terminal.focusChanged') {
        final focused = message.payload['focused'];
        if (kDebugMode) {
          debugPrint(
            '[CodeRoam Terminal] focus changed: '
            '${focused is bool ? focused : 'invalid'}',
          );
        }
      } else if (message.type == 'terminal.error') {
        debugPrint('[CodeRoam Terminal] bridge error event received.');
      }

      widget.onMessage?.call(message);
    } on FormatException {
      debugPrint('[CodeRoam Terminal] Invalid bridge message received.');
    }
  }

  Future<void> _copySelection(String selection) async {
    var copied = false;
    try {
      await Clipboard.setData(ClipboardData(text: selection));
      copied = true;
    } catch (_) {
      // Clipboard access can fail when the host platform denies the request.
    }

    if (!mounted) {
      return;
    }

    final messenger = ScaffoldMessenger.maybeOf(context);
    messenger?.hideCurrentSnackBar();
    messenger?.showSnackBar(
      SnackBar(
        content: Text(
          copied
              ? 'Terminal selection copied.'
              : 'Could not copy terminal selection.',
        ),
        duration: const Duration(seconds: 2),
      ),
    );
  }

  void _handlePageStarted() {
    _inputEventDeduplicator.reset();
    _controller.markNotReady();
  }

  @override
  Widget build(BuildContext context) {
    return EmbeddedWebView(
      assetPath: 'assets/terminal/index.html',
      javascriptChannel: 'CodeRoamTerminal',
      backgroundColor: const Color(0xFF0D0F12),
      onControllerCreated: _controller.attach,
      onMessage: _handleMessage,
      onPageStarted: _handlePageStarted,
    );
  }
}
