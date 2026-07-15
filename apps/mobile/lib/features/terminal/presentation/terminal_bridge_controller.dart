import 'package:coderoam/shared/webview/webview_bridge.dart';
import 'package:webview_flutter/webview_flutter.dart';

class TerminalBridgeController {
  TerminalBridgeController()
    : _bridge = WebViewBridgeController(
        javascriptReceiver: 'window.CodeRoamTerminalReceive',
      );

  final WebViewBridgeController _bridge;

  void attach(WebViewController controller) {
    _bridge.attach(controller.runJavaScript);
  }

  bool markReady() {
    return _bridge.markReady();
  }

  void markNotReady() {
    _bridge.markNotReady();
  }

  void dispose() {
    _bridge.dispose();
  }

  Future<void> write(String data) {
    return _bridge.send(
      WebViewBridgeMessage(type: 'terminal.write', payload: {'data': data}),
    );
  }

  Future<void> focus() {
    return _bridge.send(const WebViewBridgeMessage(type: 'terminal.focus'));
  }

  Future<void> clear() {
    return _bridge.send(const WebViewBridgeMessage(type: 'terminal.clear'));
  }

  Future<void> resize({required int columns, required int rows}) {
    return _bridge.send(
      WebViewBridgeMessage(
        type: 'terminal.resize',
        payload: {'columns': columns, 'rows': rows},
      ),
    );
  }
}
