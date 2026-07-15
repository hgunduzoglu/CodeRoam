import 'package:coderoam/shared/webview/webview_bridge.dart';
import 'package:webview_flutter/webview_flutter.dart';

class EditorBridgeController {
  EditorBridgeController()
    : _bridge = WebViewBridgeController(
        javascriptReceiver: 'window.CodeRoamEditorReceive',
      );

  final WebViewBridgeController _bridge;
  WebViewBridgeMessage? _lastSetDocumentMessage;

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

  Future<void> setDocument({
    required String content,
    required String language,
  }) {
    final message = WebViewBridgeMessage(
      type: 'editor.setDocument',
      payload: {'content': content, 'language': language},
    );
    _lastSetDocumentMessage = message;
    return _bridge.send(message);
  }

  Future<void> restoreLastSetDocument() {
    final message = _lastSetDocumentMessage;
    return message == null ? Future<void>.value() : _bridge.send(message);
  }

  Future<void> focus() {
    return _bridge.send(const WebViewBridgeMessage(type: 'editor.focus'));
  }

  Future<void> undo() {
    return _bridge.send(const WebViewBridgeMessage(type: 'editor.undo'));
  }

  Future<void> redo() {
    return _bridge.send(const WebViewBridgeMessage(type: 'editor.redo'));
  }

  Future<void> openSearch() {
    return _bridge.send(const WebViewBridgeMessage(type: 'editor.openSearch'));
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
