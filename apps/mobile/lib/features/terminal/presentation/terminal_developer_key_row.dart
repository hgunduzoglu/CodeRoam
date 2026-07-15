import 'dart:async';

import 'package:coderoam/features/terminal/presentation/terminal_input_spike_controller.dart';
import 'package:flutter/material.dart';

class TerminalDeveloperKeyRow extends StatelessWidget {
  const TerminalDeveloperKeyRow({
    required this.controller,
    required this.isFullScreen,
    required this.onToggleFullScreen,
    super.key,
  });

  final TerminalInputSpikeController controller;
  final bool isFullScreen;
  final VoidCallback onToggleFullScreen;

  @override
  Widget build(BuildContext context) {
    return Material(
      color: Theme.of(context).colorScheme.surfaceContainer,
      child: SafeArea(
        top: false,
        child: SizedBox(
          height: 64,
          child: ListenableBuilder(
            listenable: controller,
            builder: (context, _) {
              return SingleChildScrollView(
                scrollDirection: Axis.horizontal,
                child: Padding(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 8,
                    vertical: 8,
                  ),
                  child: Row(
                    children: [
                      _IconKey(
                        icon:
                            isFullScreen
                                ? Icons.fullscreen_exit
                                : Icons.fullscreen,
                        tooltip:
                            isFullScreen
                                ? 'Exit full-screen terminal'
                                : 'Enter full-screen terminal',
                        onPressed: onToggleFullScreen,
                      ),
                      _TextKey(
                        label: 'Esc',
                        onPressed:
                            () => unawaited(
                              controller.pressDeveloperKey(
                                TerminalDeveloperKey.escape,
                              ),
                            ),
                      ),
                      _TextKey(
                        label: 'Tab',
                        onPressed:
                            () => unawaited(
                              controller.pressDeveloperKey(
                                TerminalDeveloperKey.tab,
                              ),
                            ),
                      ),
                      Padding(
                        padding: const EdgeInsets.symmetric(horizontal: 4),
                        child: Semantics(
                          label: 'Control modifier',
                          toggled: controller.ctrlActive,
                          button: true,
                          child: FilterChip(
                            label: const Text('Ctrl'),
                            selected: controller.ctrlActive,
                            showCheckmark: true,
                            onSelected: (_) => controller.toggleCtrl(),
                          ),
                        ),
                      ),
                      _TextKey(
                        label: 'Burst',
                        onPressed:
                            () => unawaited(controller.runFastOutputHarness()),
                      ),
                      _IconKey(
                        icon: Icons.arrow_left,
                        tooltip: 'Left arrow',
                        onPressed:
                            () => unawaited(
                              controller.pressDeveloperKey(
                                TerminalDeveloperKey.left,
                              ),
                            ),
                      ),
                      _IconKey(
                        icon: Icons.arrow_upward,
                        tooltip: 'Up arrow',
                        onPressed:
                            () => unawaited(
                              controller.pressDeveloperKey(
                                TerminalDeveloperKey.up,
                              ),
                            ),
                      ),
                      _IconKey(
                        icon: Icons.arrow_downward,
                        tooltip: 'Down arrow',
                        onPressed:
                            () => unawaited(
                              controller.pressDeveloperKey(
                                TerminalDeveloperKey.down,
                              ),
                            ),
                      ),
                      _IconKey(
                        icon: Icons.arrow_right,
                        tooltip: 'Right arrow',
                        onPressed:
                            () => unawaited(
                              controller.pressDeveloperKey(
                                TerminalDeveloperKey.right,
                              ),
                            ),
                      ),
                    ],
                  ),
                ),
              );
            },
          ),
        ),
      ),
    );
  }
}

class _TextKey extends StatelessWidget {
  const _TextKey({required this.label, required this.onPressed});

  final String label;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 4),
      child: OutlinedButton(
        style: OutlinedButton.styleFrom(
          minimumSize: const Size(52, 48),
          padding: const EdgeInsets.symmetric(horizontal: 12),
        ),
        onPressed: onPressed,
        child: Text(label),
      ),
    );
  }
}

class _IconKey extends StatelessWidget {
  const _IconKey({
    required this.icon,
    required this.tooltip,
    required this.onPressed,
  });

  final IconData icon;
  final String tooltip;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 4),
      child: IconButton.outlined(
        constraints: const BoxConstraints.tightFor(width: 48, height: 48),
        tooltip: tooltip,
        onPressed: onPressed,
        icon: Icon(icon),
      ),
    );
  }
}
