import 'package:coderoam/shared/webview/embedded_webview.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  const assetPath = 'assets/editor/index.html';

  group('isAllowedEmbeddedAssetUrl', () {
    test('allows only the requested Flutter asset on supported platforms', () {
      expect(
        isAllowedEmbeddedAssetUrl(
          value:
              'file:///Users/developer/Library/Developer/CoreSimulator/'
              'Devices/CCEF807C-C916-46FF-97E2-66ED1B14E771/data/'
              'Containers/Bundle/Application/'
              '2A6B0718-456C-47A3-B477-8185489DBC0B/Runner.app/'
              'Frameworks/App.framework/'
              'flutter_assets/assets/editor/index.html',
          assetPath: assetPath,
        ),
        isTrue,
      );
      expect(
        isAllowedEmbeddedAssetUrl(
          value:
              'file:///private/var/containers/Bundle/Application/'
              '2A6B0718-456C-47A3-B477-8185489DBC0B/Runner.app/'
              'Frameworks/App.framework/'
              'flutter_assets/assets/editor/index.html',
          assetPath: assetPath,
        ),
        isTrue,
      );
      expect(
        isAllowedEmbeddedAssetUrl(
          value:
              'file:///android_asset/flutter_assets/'
              'assets/editor/index.html',
          assetPath: assetPath,
        ),
        isTrue,
      );
      expect(
        isAllowedEmbeddedAssetUrl(value: 'about:blank', assetPath: assetPath),
        isTrue,
      );
    });

    test('rejects local files outside the requested asset', () {
      for (final value in <String>[
        'file:///private/var/mobile/Library/preferences.plist',
        'file:///tmp/flutter_assets/assets/editor/index.html',
        'file:///android_asset/flutter_assets/assets/terminal/index.html',
        'file:///sdcard/android_asset/flutter_assets/'
            'assets/editor/index.html',
        'file:relative/android_asset/flutter_assets/'
            'assets/editor/index.html',
        'file:///Documents/Frameworks/App.framework/flutter_assets/'
            'assets/editor/index.html',
        'file:///Documents/Bundle/Application/'
            '2A6B0718-456C-47A3-B477-8185489DBC0B/Runner.app/'
            'Frameworks/App.framework/flutter_assets/'
            'assets/editor/index.html',
        'file:///private/app/Runner.app/Frameworks/App.framework/'
            'flutter_assets/assets/editor/../terminal/index.html',
        'file://localhost/android_asset/flutter_assets/'
            'assets/editor/index.html',
      ]) {
        expect(
          isAllowedEmbeddedAssetUrl(value: value, assetPath: assetPath),
          isFalse,
          reason: value,
        );
      }
    });

    test('rejects non-asset origins and malformed navigation targets', () {
      for (final value in <String>[
        'https://example.com/assets/assets/editor/index.html',
        'http://appassets.androidplatform.net/'
            'assets/assets/editor/index.html',
        'https://appassets.androidplatform.net:444/'
            'assets/assets/editor/index.html',
        'https://appassets.androidplatform.net/'
            'assets/assets/editor/index.html',
        'https://appassets.androidplatform.net/'
            'assets/assets/editor/index.html?redirect=1',
        'https://appassets.androidplatform.net/'
            'assets/assets/terminal/index.html',
        'file:///%FF',
        'https://appassets.androidplatform.net/%FF',
        'about:srcdoc',
        'javascript:alert(1)',
        'data:text/html,unsafe',
        'not a url',
      ]) {
        expect(
          isAllowedEmbeddedAssetUrl(value: value, assetPath: assetPath),
          isFalse,
          reason: value,
        );
      }
    });

    test('fails closed for invalid configured asset paths', () {
      for (final invalidAssetPath in <String>[
        '',
        '/assets/editor/index.html',
        '../assets/editor/index.html',
        'https://example.com/editor.html',
      ]) {
        expect(
          isAllowedEmbeddedAssetUrl(
            value: 'about:blank',
            assetPath: invalidAssetPath,
          ),
          isFalse,
          reason: invalidAssetPath,
        );
      }
    });
  });
}
