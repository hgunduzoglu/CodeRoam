class TerminalLocalEchoSpikeHarness {
  const TerminalLocalEchoSpikeHarness();

  String outputFor(String input) {
    final output = StringBuffer();

    for (var index = 0; index < input.length; index += 1) {
      final codeUnit = input.codeUnitAt(index);

      if (codeUnit == 0x1b) {
        if (index + 2 < input.length && input.codeUnitAt(index + 1) == 0x5b) {
          final arrow = _arrowLabel(input.codeUnitAt(index + 2));

          if (arrow != null) {
            output.write('[$arrow]');
            index += 2;
            continue;
          }
        }

        output.write('[Esc]');
        continue;
      }

      if (codeUnit == 0x0d || codeUnit == 0x0a) {
        if (codeUnit == 0x0d &&
            index + 1 < input.length &&
            input.codeUnitAt(index + 1) == 0x0a) {
          index += 1;
        }

        output.write('\r\n\$ ');
        continue;
      }

      if (codeUnit == 0x08 || codeUnit == 0x7f) {
        output.write('\b \b');
        continue;
      }

      if (codeUnit == 0x09) {
        output.write('[Tab]');
        continue;
      }

      final controlLabel = _controlLabel(codeUnit);
      if (controlLabel != null) {
        output.write('^$controlLabel');
        continue;
      }

      output.writeCharCode(codeUnit);
    }

    return output.toString();
  }

  String? _arrowLabel(int codeUnit) {
    return switch (codeUnit) {
      0x41 => '↑',
      0x42 => '↓',
      0x43 => '→',
      0x44 => '←',
      _ => null,
    };
  }

  String? _controlLabel(int codeUnit) {
    if (codeUnit >= 1 && codeUnit <= 26) {
      return String.fromCharCode(0x40 + codeUnit);
    }

    return switch (codeUnit) {
      0x00 => '@',
      0x1c => r'\',
      0x1d => ']',
      0x1e => '^',
      0x1f => '_',
      _ => null,
    };
  }
}
