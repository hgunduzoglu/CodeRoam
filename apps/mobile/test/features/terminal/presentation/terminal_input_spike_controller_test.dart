import 'package:coderoam/features/terminal/presentation/terminal_input_spike_controller.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('local harness renders supported input without a PTY', () async {
    final output = <String>[];
    final controller = TerminalInputSpikeController(
      writeOutput: (data) async => output.add(data),
    );
    addTearDown(controller.dispose);

    await controller.handleXtermInput(
      'hello\r\x7f\t\x1b\x1b[D\x1b[A\x1b[B\x1b[C\x03',
    );

    expect(output.single, 'hello\r\n\$ \b \b[Tab][Esc][←][↑][↓][→]^C');
  });

  test('Ctrl is observable and modifies the next supported key', () async {
    final output = <String>[];
    final controller = TerminalInputSpikeController(
      writeOutput: (data) async => output.add(data),
    );
    addTearDown(controller.dispose);

    controller.toggleCtrl();
    expect(controller.ctrlActive, isTrue);
    expect(output, isEmpty);

    await controller.handleXtermInput('c');

    expect(controller.ctrlActive, isFalse);
    expect(output, ['^C']);
  });

  test('developer and physical keys use the same input path', () async {
    final output = <String>[];
    final controller = TerminalInputSpikeController(
      writeOutput: (data) async => output.add(data),
    );
    addTearDown(controller.dispose);

    await controller.pressDeveloperKey(TerminalDeveloperKey.tab);
    await controller.pressDeveloperKey(TerminalDeveloperKey.up);
    await controller.handlePhysicalKeyboardInput('d', controlPressed: true);

    expect(output, ['[Tab]', '[↑]', '^D']);
  });

  test('fast output harness streams a bounded number of batches', () async {
    final output = <String>[];
    final controller = TerminalInputSpikeController(
      writeOutput: (data) async => output.add(data),
    );
    addTearDown(controller.dispose);

    await controller.runFastOutputHarness();

    expect(
      output,
      hasLength(terminalFastOutputLineCount ~/ terminalFastOutputBatchSize),
    );
    expect(output.first, startsWith('burst 0001'));
    expect(output.last, contains('burst 0600'));
    expect(output.join(), isNot(contains('burst 0601')));
  });
}
