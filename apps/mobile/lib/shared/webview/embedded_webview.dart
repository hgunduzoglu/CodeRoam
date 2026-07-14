import 'dart:async';

import 'package:flutter/material.dart';
import 'package:webview_flutter/webview_flutter.dart';

class EmbeddedWebView extends StatefulWidget {
  const EmbeddedWebView({
    required this.assetPath,
    required this.javascriptChannel,
    required this.backgroundColor,
    this.onMessage,
    super.key,
  });

  final String assetPath;
  final String javascriptChannel;
  final Color backgroundColor;
  final ValueChanged<String>? onMessage;

  @override
  State<EmbeddedWebView> createState() => _EmbeddedWebViewState();
}

class _EmbeddedWebViewState extends State<EmbeddedWebView> {
  late final WebViewController _controller;

  int _progress = 0;
  String? _error;

  @override
  void initState() {
    super.initState();

    _controller = WebViewController()
      ..setJavaScriptMode(JavaScriptMode.unrestricted)
      ..setBackgroundColor(widget.backgroundColor)
      ..enableZoom(false)
      ..addJavaScriptChannel(
        widget.javascriptChannel,
        onMessageReceived: (message) {
          widget.onMessage?.call(message.message);
        },
      )
      ..setNavigationDelegate(
        NavigationDelegate(
          onProgress: (progress) {
            if (!mounted) return;

            setState(() {
              _progress = progress;
            });
          },
          onPageStarted: (_) {
            if (!mounted) return;

            setState(() {
              _error = null;
            });
          },
          onWebResourceError: (error) {
            if (error.isForMainFrame == false || !mounted) return;

            setState(() {
              _error = error.description;
            });
          },
          onNavigationRequest: (request) {
            return _isAllowedAssetUrl(request.url)
                ? NavigationDecision.navigate
                : NavigationDecision.prevent;
          },
        ),
      );

    unawaited(_loadAsset());
  }

  Future<void> _loadAsset() async {
    try {
      await _controller.loadFlutterAsset(widget.assetPath);
    } catch (error) {
      if (!mounted) return;

      setState(() {
        _error = error.toString();
      });
    }
  }

  bool _isAllowedAssetUrl(String value) {
    final uri = Uri.tryParse(value);

    if (uri == null) {
      return false;
    }

    return uri.scheme == 'file' ||
        uri.scheme == 'about' ||
        uri.host == 'appassets.androidplatform.net';
  }

  @override
  Widget build(BuildContext context) {
    return Stack(
      children: [
        Positioned.fill(
          child: WebViewWidget(controller: _controller),
        ),
        if (_progress < 100 && _error == null)
          const Align(
            alignment: Alignment.topCenter,
            child: LinearProgressIndicator(),
          ),
        if (_error case final error?)
          Positioned.fill(
            child: ColoredBox(
              color: widget.backgroundColor,
              child: Center(
                child: Padding(
                  padding: const EdgeInsets.all(24),
                  child: Text(
                    'WebView could not be loaded.\n\n$error',
                    textAlign: TextAlign.center,
                  ),
                ),
              ),
            ),
          ),
      ],
    );
  }
}
