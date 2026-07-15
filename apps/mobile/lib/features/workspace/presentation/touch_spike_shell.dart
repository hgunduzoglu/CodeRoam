import 'dart:async';

import 'package:coderoam/features/editor/presentation/editor_webview.dart';
import 'package:coderoam/features/terminal/presentation/terminal_bridge_controller.dart';
import 'package:coderoam/features/terminal/presentation/terminal_developer_key_row.dart';
import 'package:coderoam/features/terminal/presentation/terminal_input_spike_controller.dart';
import 'package:coderoam/features/terminal/presentation/terminal_webview.dart';
import 'package:coderoam/shared/webview/webview_bridge.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

enum WorkspaceMode { editor, terminal, review, operations }

class TouchSpikeShell extends StatefulWidget {
  const TouchSpikeShell({this.editorSurface, this.terminalSurface, super.key});

  final Widget? editorSurface;
  final Widget? terminalSurface;

  @override
  State<TouchSpikeShell> createState() => _TouchSpikeShellState();
}

class _TouchSpikeShellState extends State<TouchSpikeShell>
    with WidgetsBindingObserver {
  WorkspaceMode mode = WorkspaceMode.editor;

  late final TerminalBridgeController _terminalBridgeController;
  late final TerminalInputSpikeController _terminalInputController;
  late final FocusNode _terminalHardwareKeyboardFocusNode;

  double? _lastKeyboardInset;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _terminalBridgeController = TerminalBridgeController();
    _terminalInputController = TerminalInputSpikeController(
      writeOutput: _terminalBridgeController.write,
    );
    _terminalHardwareKeyboardFocusNode = FocusNode(
      debugLabel: 'terminal spike hardware keyboard',
    );
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    _terminalHardwareKeyboardFocusNode.dispose();
    _terminalInputController.dispose();
    _terminalBridgeController.dispose();
    super.dispose();
  }

  @override
  void didChangeMetrics() {
    if (!mounted) {
      return;
    }

    final view = View.of(context);
    final keyboardInset = view.viewInsets.bottom / view.devicePixelRatio;

    if (_lastKeyboardInset == keyboardInset) {
      return;
    }

    _lastKeyboardInset = keyboardInset;
    debugPrint(
      '[CodeRoam Keyboard] visible=${keyboardInset > 0}, '
      'bottomInset=${keyboardInset.round()}',
    );
  }

  @override
  Widget build(BuildContext context) {
    final width = MediaQuery.sizeOf(context).width;
    final isTablet = width >= 760;

    return Scaffold(
      appBar: AppBar(
        title: const Text('CodeRoam'),
        actions: [
          IconButton(
            tooltip: 'Connection status',
            onPressed: () {},
            icon: const Badge(
              backgroundColor: Colors.orange,
              child: Icon(Icons.cloud_off_outlined),
            ),
          ),
        ],
      ),
      body: Row(
        children: [
          if (isTablet)
            NavigationRail(
              selectedIndex: mode.index,
              onDestinationSelected: _selectMode,
              destinations: const [
                NavigationRailDestination(
                  icon: Icon(Icons.code),
                  label: Text('Editor'),
                ),
                NavigationRailDestination(
                  icon: Icon(Icons.terminal),
                  label: Text('Terminal'),
                ),
                NavigationRailDestination(
                  icon: Icon(Icons.rate_review_outlined),
                  label: Text('Review'),
                ),
                NavigationRailDestination(
                  icon: Icon(Icons.monitor_heart_outlined),
                  label: Text('Operate'),
                ),
              ],
            ),
          Expanded(
            child: IndexedStack(
              index: mode.index,
              children: [
                widget.editorSurface ?? const EditorWebView(),
                _buildTerminalWorkspace(),
                const _FutureModePlaceholder(
                  title: 'Review mode',
                  detail: 'Structured diffs and review comments arrive later.',
                ),
                const _FutureModePlaceholder(
                  title: 'Operations mode',
                  detail:
                      'Logs, deployment status, and controlled runbooks arrive later.',
                ),
              ],
            ),
          ),
        ],
      ),
      bottomNavigationBar:
          isTablet
              ? null
              : NavigationBar(
                selectedIndex: mode.index,
                onDestinationSelected: _selectMode,
                destinations: const [
                  NavigationDestination(
                    icon: Icon(Icons.code),
                    label: 'Editor',
                  ),
                  NavigationDestination(
                    icon: Icon(Icons.terminal),
                    label: 'Terminal',
                  ),
                  NavigationDestination(
                    icon: Icon(Icons.rate_review_outlined),
                    label: 'Review',
                  ),
                  NavigationDestination(
                    icon: Icon(Icons.monitor_heart_outlined),
                    label: 'Operate',
                  ),
                ],
              ),
    );
  }

  void _selectMode(int index) {
    if (index < 0 || index >= WorkspaceMode.values.length) {
      return;
    }

    setState(() {
      mode = WorkspaceMode.values[index];
    });

    if (mode == WorkspaceMode.terminal) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted) {
          _terminalHardwareKeyboardFocusNode.requestFocus();
        }
      });
    } else {
      _terminalHardwareKeyboardFocusNode.unfocus();
    }
  }

  Widget _buildTerminalWorkspace() {
    final terminalSurface =
        widget.terminalSurface ??
        TerminalWebView(
          controller: _terminalBridgeController,
          onReady:
              () => unawaited(_terminalInputController.startLocalEchoHarness()),
          onMessage: _handleTerminalMessage,
        );

    return Column(
      children: [
        Expanded(
          child: Focus(
            focusNode: _terminalHardwareKeyboardFocusNode,
            onKeyEvent: _handleTerminalHardwareKey,
            child: terminalSurface,
          ),
        ),
        TerminalDeveloperKeyRow(controller: _terminalInputController),
      ],
    );
  }

  void _handleTerminalMessage(WebViewBridgeMessage message) {
    if (message.type != 'terminal.input') {
      return;
    }

    final data = message.payload['data'];
    if (data is! String) {
      debugPrint('[CodeRoam Terminal] Invalid terminal.input payload.');
      return;
    }

    unawaited(_terminalInputController.handleXtermInput(data));
  }

  KeyEventResult _handleTerminalHardwareKey(FocusNode _, KeyEvent event) {
    if (event is! KeyDownEvent && event is! KeyRepeatEvent) {
      return KeyEventResult.ignored;
    }

    final input = _terminalInputFor(event);
    if (input == null) {
      return KeyEventResult.ignored;
    }

    unawaited(
      _terminalInputController.handlePhysicalKeyboardInput(
        input,
        controlPressed: HardwareKeyboard.instance.isControlPressed,
      ),
    );
    return KeyEventResult.handled;
  }

  String? _terminalInputFor(KeyEvent event) {
    final logicalKey = event.logicalKey;

    if (logicalKey == LogicalKeyboardKey.enter ||
        logicalKey == LogicalKeyboardKey.numpadEnter) {
      return '\r';
    }

    if (logicalKey == LogicalKeyboardKey.backspace) {
      return '\x7f';
    }

    if (logicalKey == LogicalKeyboardKey.tab) {
      return '\t';
    }

    if (logicalKey == LogicalKeyboardKey.escape) {
      return '\x1b';
    }

    if (logicalKey == LogicalKeyboardKey.arrowLeft) {
      return '\x1b[D';
    }

    if (logicalKey == LogicalKeyboardKey.arrowUp) {
      return '\x1b[A';
    }

    if (logicalKey == LogicalKeyboardKey.arrowDown) {
      return '\x1b[B';
    }

    if (logicalKey == LogicalKeyboardKey.arrowRight) {
      return '\x1b[C';
    }

    final character = event.character;
    return character == null || character.isEmpty ? null : character;
  }
}

class _FutureModePlaceholder extends StatelessWidget {
  const _FutureModePlaceholder({required this.title, required this.detail});

  final String title;
  final String detail;

  @override
  Widget build(BuildContext context) {
    return SafeArea(
      child: Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 640),
            child: Card(
              child: Padding(
                padding: const EdgeInsets.all(24),
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Text(
                      title,
                      style: Theme.of(context).textTheme.headlineSmall,
                    ),
                    const SizedBox(height: 12),
                    Text(detail, textAlign: TextAlign.center),
                  ],
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}
