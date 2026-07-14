import 'package:coderoam/features/workspace/presentation/touch_spike_shell.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('switches between editor and terminal modes', (tester) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: TouchSpikeShell(
          editorSurface: Center(child: Text('editor mock')),
          terminalSurface: Center(child: Text('terminal mock')),
        ),
      ),
    );

    expect(find.text('editor mock'), findsOneWidget);

    await tester.tap(find.text('Terminal'));
    await tester.pumpAndSettle();

    expect(find.text('terminal mock'), findsOneWidget);
  });
}
