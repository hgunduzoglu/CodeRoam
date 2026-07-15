import 'dart:convert';

import 'package:coderoam/features/terminal/presentation/terminal_local_echo_spike_harness.dart';
import 'package:flutter/foundation.dart';

typedef TerminalOutputWriter = Future<void> Function(String data);

enum TerminalDeveloperKey { escape, tab, left, up, down, right }

enum TerminalInputSource { xterm, physicalKeyboard, developerKey }

class TerminalInputSpikeController extends ChangeNotifier {
  TerminalInputSpikeController({
    required TerminalOutputWriter writeOutput,
    TerminalLocalEchoSpikeHarness harness =
        const TerminalLocalEchoSpikeHarness(),
  }) : _writeOutput = writeOutput,
       _harness = harness;

  final TerminalOutputWriter _writeOutput;
  final TerminalLocalEchoSpikeHarness _harness;

  bool _ctrlActive = false;

  bool get ctrlActive => _ctrlActive;

  void toggleCtrl() {
    _ctrlActive = !_ctrlActive;
    notifyListeners();
  }

  Future<void> startLocalEchoHarness() {
    return _writeSafely(
      '\r\nCodeRoam local terminal-input spike (no PTY).\r\n\r\n\$ ',
      operation: 'start local echo harness',
    );
  }

  Future<void> handleXtermInput(String data) {
    return _handleInput(data, source: TerminalInputSource.xterm);
  }

  Future<void> handlePhysicalKeyboardInput(
    String data, {
    required bool controlPressed,
  }) {
    return _handleInput(
      data,
      source: TerminalInputSource.physicalKeyboard,
      controlPressed: controlPressed,
    );
  }

  Future<void> pressDeveloperKey(TerminalDeveloperKey key) {
    return _handleInput(switch (key) {
      TerminalDeveloperKey.escape => '\x1b',
      TerminalDeveloperKey.tab => '\t',
      TerminalDeveloperKey.left => '\x1b[D',
      TerminalDeveloperKey.up => '\x1b[A',
      TerminalDeveloperKey.down => '\x1b[B',
      TerminalDeveloperKey.right => '\x1b[C',
    }, source: TerminalInputSource.developerKey);
  }

  Future<void> _handleInput(
    String data, {
    required TerminalInputSource source,
    bool controlPressed = false,
  }) async {
    if (data.isEmpty) {
      return;
    }

    var terminalInput = data;

    if (controlPressed) {
      terminalInput = _ctrlSequenceFor(data) ?? data;
    } else if (_ctrlActive) {
      terminalInput = _ctrlSequenceFor(data) ?? data;
      _ctrlActive = false;
      notifyListeners();
    }

    debugPrint(
      '[CodeRoam Terminal] input source=${source.name}, '
      'bytes=${utf8.encode(terminalInput).length}',
    );

    final output = _harness.outputFor(terminalInput);
    if (output.isEmpty) {
      return;
    }

    await _writeSafely(output, operation: 'echo terminal input');
  }

  String? _ctrlSequenceFor(String data) {
    final runes = data.runes.toList(growable: false);
    if (runes.length != 1) {
      return null;
    }

    var value = runes.single;
    if (value >= 0x61 && value <= 0x7a) {
      value -= 0x20;
    }

    if (value >= 0x40 && value <= 0x5f) {
      return String.fromCharCode(value & 0x1f);
    }

    if (value == 0x3f) {
      return String.fromCharCode(0x7f);
    }

    return null;
  }

  Future<void> _writeSafely(String output, {required String operation}) async {
    try {
      await _writeOutput(output);
    } catch (_) {
      debugPrint('[CodeRoam Terminal] Could not $operation.');
    }
  }
}
