import 'package:coderoam/features/workspace/presentation/touch_spike_shell.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('switches between editor and terminal modes', (tester) async {
    await tester.binding.setSurfaceSize(const Size(390, 844));
    addTearDown(() => tester.binding.setSurfaceSize(null));

    await tester.pumpWidget(
      const MaterialApp(
        home: TouchSpikeShell(
          editorSurface: Center(child: Text('editor mock')),
          terminalSurface: Center(child: Text('terminal mock')),
        ),
      ),
    );

    expect(find.text('editor mock'), findsOneWidget);

    expect(find.text('Esc'), findsNothing);

    await tester.tap(find.byIcon(Icons.terminal));
    await tester.pumpAndSettle();

    expect(find.text('terminal mock'), findsOneWidget);
    expect(find.text('Esc'), findsOneWidget);
    expect(find.text('Tab'), findsOneWidget);
    expect(find.text('Ctrl'), findsOneWidget);
    expect(find.byTooltip('Left arrow'), findsOneWidget);
    expect(find.byTooltip('Up arrow'), findsOneWidget);
    expect(find.byTooltip('Down arrow'), findsOneWidget);
    expect(find.byTooltip('Right arrow'), findsOneWidget);

    var ctrlChip = tester.widget<FilterChip>(
      find.widgetWithText(FilterChip, 'Ctrl'),
    );
    expect(ctrlChip.selected, isFalse);

    await tester.tap(find.text('Ctrl'));
    await tester.pump();

    ctrlChip = tester.widget<FilterChip>(
      find.widgetWithText(FilterChip, 'Ctrl'),
    );
    expect(ctrlChip.selected, isTrue);

    await tester.tap(find.byIcon(Icons.code));
    await tester.pumpAndSettle();

    expect(find.text('Esc'), findsNothing);
  });
}
