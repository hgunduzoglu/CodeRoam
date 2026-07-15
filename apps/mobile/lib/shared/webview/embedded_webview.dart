import 'dart:async';

import 'package:flutter/material.dart';
import 'package:webview_flutter/webview_flutter.dart';

class EmbeddedWebView extends StatefulWidget {
  const EmbeddedWebView({
    required this.assetPath,
    required this.javascriptChannel,
    required this.backgroundColor,
    this.onMessage,
    this.onControllerCreated,
    this.onPageStarted,
    super.key,
  });
  final ValueChanged<WebViewController>? onControllerCreated;
  final String assetPath;
  final String javascriptChannel;
  final Color backgroundColor;
  final ValueChanged<String>? onMessage;
  final VoidCallback? onPageStarted;

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

    _controller =
        WebViewController()
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
                widget.onPageStarted?.call();

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
                return isAllowedEmbeddedAssetUrl(
                      value: request.url,
                      assetPath: widget.assetPath,
                    )
                    ? NavigationDecision.navigate
                    : NavigationDecision.prevent;
              },
            ),
          );
    widget.onControllerCreated?.call(_controller);

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

  @override
  Widget build(BuildContext context) {
    return Stack(
      children: [
        Positioned.fill(child: WebViewWidget(controller: _controller)),
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

bool isAllowedEmbeddedAssetUrl({
  required String value,
  required String assetPath,
}) {
  final uri = Uri.tryParse(value);
  final assetUri = Uri.tryParse(assetPath);

  if (uri == null ||
      assetUri == null ||
      assetUri.hasScheme ||
      assetUri.hasAuthority ||
      assetUri.hasAbsolutePath ||
      assetUri.hasQuery ||
      assetUri.hasFragment) {
    return false;
  }

  late final List<String> uriSegments;
  late final List<String> assetSegments;
  try {
    uriSegments = uri.pathSegments;
    assetSegments = assetUri.pathSegments;
  } on FormatException {
    return false;
  }

  if (assetSegments.isEmpty || assetSegments.any(_isUnsafePathSegment)) {
    return false;
  }

  if (uri == Uri.parse('about:blank')) {
    return true;
  }

  if (uri.hasQuery ||
      uri.hasFragment ||
      uri.userInfo.isNotEmpty ||
      uriSegments.any(_isUnsafePathSegment)) {
    return false;
  }

  if (uri.scheme != 'file' || uri.host.isNotEmpty || !uri.hasAbsolutePath) {
    return false;
  }

  return _pathEquals(uriSegments, [
        'android_asset',
        'flutter_assets',
        ...assetSegments,
      ]) ||
      _isAllowedIosAssetPath(uriSegments, assetSegments);
}

bool _isUnsafePathSegment(String segment) {
  return segment.isEmpty || segment == '.' || segment == '..';
}

bool _pathEquals(List<String> actual, List<String> expected) {
  if (actual.length != expected.length) {
    return false;
  }

  for (var index = 0; index < expected.length; index += 1) {
    if (actual[index] != expected[index]) {
      return false;
    }
  }

  return true;
}

bool _isAllowedIosAssetPath(List<String> actual, List<String> assetSegments) {
  final bundleTail = <String>[
    'Runner.app',
    'Frameworks',
    'App.framework',
    'flutter_assets',
    ...assetSegments,
  ];

  if (!_pathEndsWith(actual, bundleTail)) {
    return false;
  }

  final runnerIndex = actual.length - bundleTail.length;
  if (runnerIndex < 3 ||
      actual[runnerIndex - 3] != 'Bundle' ||
      actual[runnerIndex - 2] != 'Application' ||
      !_isUuid(actual[runnerIndex - 1])) {
    return false;
  }

  final containerPrefix = actual.sublist(0, runnerIndex - 3);
  return _pathEquals(containerPrefix, const ['private', 'var', 'containers']) ||
      _pathEquals(containerPrefix, const ['var', 'containers']) ||
      _isIosSimulatorContainerPrefix(containerPrefix);
}

bool _isIosSimulatorContainerPrefix(List<String> segments) {
  return segments.length == 9 &&
      segments[0] == 'Users' &&
      segments[1].isNotEmpty &&
      segments[2] == 'Library' &&
      segments[3] == 'Developer' &&
      segments[4] == 'CoreSimulator' &&
      segments[5] == 'Devices' &&
      _isUuid(segments[6]) &&
      segments[7] == 'data' &&
      segments[8] == 'Containers';
}

bool _isUuid(String value) {
  return RegExp(
    r'^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-'
    r'[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$',
  ).hasMatch(value);
}

bool _pathEndsWith(List<String> actual, List<String> expected) {
  if (actual.length < expected.length) {
    return false;
  }

  final offset = actual.length - expected.length;
  for (var index = 0; index < expected.length; index += 1) {
    if (actual[offset + index] != expected[index]) {
      return false;
    }
  }

  return true;
}
