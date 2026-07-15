import 'package:coderoam/shared/webview/webview_bridge.dart';

const int maximumCopiedTerminalSelectionCodeUnits = 262144;

String? terminalSelectionText(WebViewBridgeMessage message) {
  if (message.type != 'terminal.copySelection') {
    return null;
  }

  final text = message.payload['text'];
  if (text is! String ||
      text.isEmpty ||
      text.length > maximumCopiedTerminalSelectionCodeUnits) {
    return null;
  }

  return text;
}
