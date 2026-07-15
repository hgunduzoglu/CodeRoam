import 'dart:async';

import 'package:coderoam/features/editor/presentation/editor_bridge_controller.dart';
import 'package:coderoam/features/editor/presentation/editor_spike_fixture.dart';
import 'package:coderoam/features/editor/presentation/editor_webview.dart';
import 'package:coderoam/features/terminal/presentation/terminal_bridge_controller.dart';
import 'package:coderoam/features/terminal/presentation/terminal_developer_key_row.dart';
import 'package:coderoam/features/terminal/presentation/terminal_input_spike_controller.dart';
import 'package:coderoam/features/terminal/presentation/terminal_webview.dart';
import 'package:coderoam/shared/webview/webview_bridge.dart';
import 'package:flutter/foundation.dart';
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
  late final EditorBridgeController _editorBridgeController;

  bool _terminalLoaded = false;
  bool _terminalFullScreen = false;
  double? _lastKeyboardInset;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _editorBridgeController = EditorBridgeController();
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
    _editorBridgeController.dispose();
    super.dispose();
  }

  @override
  void didChangeMetrics() {
    if (!kDebugMode || !mounted) {
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
    final isTerminalFullScreen =
        mode == WorkspaceMode.terminal && _terminalFullScreen;

    return PopScope<Object?>(
      canPop: !isTerminalFullScreen,
      onPopInvokedWithResult: (didPop, _) {
        if (!didPop && isTerminalFullScreen) {
          setState(() {
            _terminalFullScreen = false;
          });
        }
      },
      child: Scaffold(
        appBar:
            isTerminalFullScreen
                ? null
                : AppBar(
                  title: const Text('CodeRoam'),
                  actions: [
                    if (mode == WorkspaceMode.editor)
                      PopupMenuButton<_EditorSpikeAction>(
                        tooltip: 'Editor spike actions',
                        onSelected:
                            (action) =>
                                unawaited(_handleEditorSpikeAction(action)),
                        itemBuilder:
                            (_) => const [
                              PopupMenuItem(
                                value: _EditorSpikeAction.undo,
                                child: Text('Undo'),
                              ),
                              PopupMenuItem(
                                value: _EditorSpikeAction.redo,
                                child: Text('Redo'),
                              ),
                              PopupMenuItem(
                                value: _EditorSpikeAction.search,
                                child: Text('Search and replace'),
                              ),
                              PopupMenuDivider(),
                              PopupMenuItem(
                                value: _EditorSpikeAction.sampleFixture,
                                child: Text('Load sample fixture'),
                              ),
                              PopupMenuItem(
                                value: _EditorSpikeAction.largeFixture,
                                child: Text('Load 10,000-line fixture'),
                              ),
                            ],
                      ),
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
        body: SafeArea(
          top: isTerminalFullScreen,
          bottom: false,
          child: Row(
            children: [
              if (isTablet && !isTerminalFullScreen)
                NavigationRail(
                  scrollable: true,
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
                    widget.editorSurface ??
                        EditorWebView(controller: _editorBridgeController),
                    _terminalLoaded
                        ? _buildTerminalWorkspace()
                        : const SizedBox.shrink(),
                    const _FutureModePlaceholder(
                      title: 'Review mode',
                      detail:
                          'Structured diffs and review comments arrive later.',
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
        ),
        bottomNavigationBar:
            isTablet || isTerminalFullScreen
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
      ),
    );
  }

  void _selectMode(int index) {
    if (index < 0 || index >= WorkspaceMode.values.length) {
      return;
    }

    final selectedMode = WorkspaceMode.values[index];
    setState(() {
      mode = selectedMode;
      _terminalLoaded =
          _terminalLoaded || selectedMode == WorkspaceMode.terminal;
      if (selectedMode != WorkspaceMode.terminal) {
        _terminalFullScreen = false;
      }
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

  void _toggleTerminalFullScreen() {
    setState(() {
      _terminalFullScreen = !_terminalFullScreen;
    });
  }

  Future<void> _handleEditorSpikeAction(_EditorSpikeAction action) async {
    try {
      await switch (action) {
        _EditorSpikeAction.undo => _editorBridgeController.undo(),
        _EditorSpikeAction.redo => _editorBridgeController.redo(),
        _EditorSpikeAction.search => _editorBridgeController.openSearch(),
        _EditorSpikeAction.sampleFixture => _editorBridgeController.setDocument(
          content: editorSpikeDocument(EditorSpikeFixture.sample),
          language: 'typescript',
        ),
        _EditorSpikeAction.largeFixture => _editorBridgeController.setDocument(
          content: editorSpikeDocument(EditorSpikeFixture.largeFile),
          language: 'typescript',
        ),
      };
    } catch (_) {
      debugPrint('[CodeRoam Editor] Could not run spike action.');
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
        TerminalDeveloperKeyRow(
          controller: _terminalInputController,
          isFullScreen: _terminalFullScreen,
          onToggleFullScreen: _toggleTerminalFullScreen,
        ),
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

enum _EditorSpikeAction { undo, redo, search, sampleFixture, largeFixture }

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
