import 'package:flutter/material.dart';

void main() {
  runApp(const CodeRoamApp());
}

class CodeRoamApp extends StatelessWidget {
  const CodeRoamApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'CodeRoam',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        colorSchemeSeed: const Color(0xFF635BFF),
        brightness: Brightness.dark,
        useMaterial3: true,
      ),
      home: const TouchSpikeShell(),
    );
  }
}

enum WorkspaceMode { editor, terminal, review, operations }

class TouchSpikeShell extends StatefulWidget {
  const TouchSpikeShell({super.key});

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
              onDestinationSelected: (index) {
                setState(() => mode = WorkspaceMode.values[index]);
              },
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
          Expanded(child: _ModePlaceholder(mode: mode)),
        ],
      ),
      bottomNavigationBar: isTablet
          ? null
          : NavigationBar(
              selectedIndex: mode.index,
              onDestinationSelected: (index) {
                setState(() => mode = WorkspaceMode.values[index]);
              },
              destinations: const [
                NavigationDestination(icon: Icon(Icons.code), label: 'Editor'),
                NavigationDestination(icon: Icon(Icons.terminal), label: 'Terminal'),
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
}

class _ModePlaceholder extends StatelessWidget {
  const _ModePlaceholder({required this.mode});

  final WorkspaceMode mode;

  @override
  Widget build(BuildContext context) {
    final title = switch (mode) {
      WorkspaceMode.editor => 'Editor touch spike',
      WorkspaceMode.terminal => 'Terminal touch spike',
      WorkspaceMode.review => 'Review mode',
      WorkspaceMode.operations => 'Operations mode',
    };
    final detail = switch (mode) {
      WorkspaceMode.editor =>
        'Embed the CodeMirror bundle and validate cursor, selection, IME, keyboard, and focus.',
      WorkspaceMode.terminal =>
        'Embed the xterm.js bundle and validate input, selection, output, resize, and key row.',
      WorkspaceMode.review => 'Structured diffs and review comments arrive in a later milestone.',
      WorkspaceMode.operations =>
        'Logs, deployment status, and controlled runbooks arrive in a later milestone.',
    };

    return SafeArea(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 640),
            child: Card(
              child: Padding(
                padding: const EdgeInsets.all(24),
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Text(title, style: Theme.of(context).textTheme.headlineSmall),
                    const SizedBox(height: 12),
                    Text(detail, textAlign: TextAlign.center),
                    const SizedBox(height: 24),
                    FilledButton.icon(
                      onPressed: () {},
                      icon: const Icon(Icons.science_outlined),
                      label: const Text('Run interaction checklist'),
                    ),
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
