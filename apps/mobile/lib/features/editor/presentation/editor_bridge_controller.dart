import 'package:coderoam/shared/webview/webview_bridge.dart';
import 'package:webview_flutter/webview_flutter.dart';

class EditorBridgeController {
  EditorBridgeController()
    : _bridge = WebViewBridgeController(
        javascriptReceiver: 'window.CodeRoamEditorReceive',
      );

  final WebViewBridgeController _bridge;

  void attach(WebViewController controller) {
    _bridge.attach(controller);
  }

  void markReady() {
    _bridge.markReady();
  }

  Future<void> setDocument({
    required String content,
    required String language,
  }) {
    return _bridge.send(
      WebViewBridgeMessage(
        type: 'editor.setDocument',
        payload: {'content': content, 'language': language},
      ),
    );
  }

  Future<void> focus() {
    return _bridge.send(const WebViewBridgeMessage(type: 'editor.focus'));
  }

  Future<void> requestState() {
    return _bridge.send(const WebViewBridgeMessage(type: 'editor.getState'));
  }

  Future<void> setSelection({required int anchor, required int head}) {
    return _bridge.send(
      WebViewBridgeMessage(
        type: 'editor.setSelection',
        payload: {'anchor': anchor, 'head': head},
      ),
    );
  }
}
