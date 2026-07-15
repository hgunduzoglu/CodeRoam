import 'package:coderoam/shared/webview/webview_bridge.dart';

const int maximumTerminalClipboardCodeUnits = 262144;

String? boundedTerminalClipboardText(Object? value) {
  if (value is! String ||
      value.isEmpty ||
      value.length > maximumTerminalClipboardCodeUnits) {
    return null;
  }

  return value;
}

String? terminalSelectionText(WebViewBridgeMessage message) {
  if (message.type != 'terminal.copySelection') {
    return null;
  }

  return boundedTerminalClipboardText(message.payload['text']);
}
