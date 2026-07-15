import 'package:coderoam/features/terminal/presentation/terminal_selection_bridge.dart';
import 'package:coderoam/shared/webview/webview_bridge.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('accepts a bounded terminal selection', () {
    const message = WebViewBridgeMessage(
      type: 'terminal.copySelection',
      payload: {'text': 'selected terminal text'},
    );

    expect(terminalSelectionText(message), 'selected terminal text');
  });

  test('rejects empty, oversized, and unrelated payloads', () {
    for (final message in [
      const WebViewBridgeMessage(
        type: 'terminal.copySelection',
        payload: {'text': ''},
      ),
      WebViewBridgeMessage(
        type: 'terminal.copySelection',
        payload: {
          'text':
              List.filled(maximumTerminalClipboardCodeUnits + 1, 'x').join(),
        },
      ),
      const WebViewBridgeMessage(
        type: 'terminal.input',
        payload: {'text': 'not a selection'},
      ),
    ]) {
      expect(terminalSelectionText(message), isNull);
    }
  });

  test('bounds clipboard text before paste', () {
    expect(boundedTerminalClipboardText('paste me'), 'paste me');
    expect(boundedTerminalClipboardText(null), isNull);
    expect(boundedTerminalClipboardText(''), isNull);
    expect(boundedTerminalClipboardText(7), isNull);
    expect(
      boundedTerminalClipboardText(
        List.filled(maximumTerminalClipboardCodeUnits + 1, 'x').join(),
      ),
      isNull,
    );
  });
}
