import 'package:coderoam/features/editor/presentation/editor_webview.dart';
import 'package:coderoam/features/terminal/presentation/terminal_webview.dart';
import 'package:flutter/material.dart';

enum WorkspaceMode { editor, terminal, review, operations }

class TouchSpikeShell extends StatefulWidget {
  const TouchSpikeShell({this.editorSurface, this.terminalSurface, super.key});

  final Widget? editorSurface;
  final Widget? terminalSurface;

  @override
  State<TouchSpikeShell> createState() => _TouchSpikeShellState();
}

class _TouchSpikeShellState extends State<TouchSpikeShell> {
  WorkspaceMode mode = WorkspaceMode.editor;

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
                widget.terminalSurface ?? const TerminalWebView(),
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
    setState(() {
      mode = WorkspaceMode.values[index];
    });
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
