import 'dart:async';
import 'dart:convert';

import 'package:coderoam/shared/webview/webview_bridge.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('WebViewBridgeMessage.decode', () {
    test('decodes a valid bridge message', () {
      final message = WebViewBridgeMessage.decode(
        jsonEncode({
          'version': 1,
          'id': 'state-1',
          'type': 'editor.state',
          'payload': {
            'selection': {'anchor': 2, 'head': 5},
          },
        }),
      );

      expect(message.id, 'state-1');
      expect(message.type, 'editor.state');
      expect(message.payload['selection'], {'anchor': 2, 'head': 5});
    });

    test('rejects unsupported protocol versions', () {
      expect(
        () => WebViewBridgeMessage.decode(
          '{"version":2,"type":"editor.ready","payload":{}}',
        ),
        throwsA(isA<FormatException>()),
      );
    });

    test('rejects malformed messages', () {
      for (final rawMessage in <String>[
        'not json',
        '[]',
        '{"version":1,"type":"","payload":{}}',
        '{"version":1,"id":7,"type":"editor.ready","payload":{}}',
        '{"version":1,"type":"editor.ready","payload":[]}',
        '{"version":1,"type":"editor.ready","payload":null}',
      ]) {
        expect(
          () => WebViewBridgeMessage.decode(rawMessage),
          throwsA(isA<FormatException>()),
          reason: rawMessage,
        );
      }
    });
  });

  group('WebViewBridgeController', () {
    test('queues before ready and flushes exactly once in order', () async {
      final scripts = <String>[];
      final controller = WebViewBridgeController(
        javascriptReceiver: 'window.CodeRoamReceive',
      );
      controller.attach((source) async => scripts.add(source));

      final first = controller.send(
        const WebViewBridgeMessage(type: 'message.first'),
      );
      final second = controller.send(
        const WebViewBridgeMessage(type: 'message.second'),
      );

      expect(scripts, isEmpty);
      expect(controller.markReady(), isTrue);
      await Future.wait([first, second]);

      expect(_messageTypes(scripts), ['message.first', 'message.second']);
      expect(controller.markReady(), isFalse);
      await Future<void>.delayed(Duration.zero);
      expect(_messageTypes(scripts), ['message.first', 'message.second']);
    });

    test('preserves order for messages added during a flush', () async {
      final scripts = <String>[];
      final firstWrite = Completer<void>();
      final controller = WebViewBridgeController(
        javascriptReceiver: 'window.CodeRoamReceive',
      );
      controller.attach((source) async {
        scripts.add(source);
        if (scripts.length == 1) {
          await firstWrite.future;
        }
      });
      controller.markReady();

      final first = controller.send(
        const WebViewBridgeMessage(type: 'message.first'),
      );
      final second = controller.send(
        const WebViewBridgeMessage(type: 'message.second'),
      );
      await Future<void>.delayed(Duration.zero);
      final third = controller.send(
        const WebViewBridgeMessage(type: 'message.third'),
      );

      expect(_messageTypes(scripts), ['message.first']);
      firstWrite.complete();
      await Future.wait([first, second, third]);

      expect(_messageTypes(scripts), [
        'message.first',
        'message.second',
        'message.third',
      ]);
    });

    test('queues again after page readiness is reset', () async {
      final scripts = <String>[];
      final controller = WebViewBridgeController(
        javascriptReceiver: 'window.CodeRoamReceive',
      );
      controller.attach((source) async => scripts.add(source));
      controller.markReady();

      await controller.send(
        const WebViewBridgeMessage(type: 'message.beforeReload'),
      );
      controller.markNotReady();
      final afterReload = controller.send(
        const WebViewBridgeMessage(type: 'message.afterReload'),
      );
      await Future<void>.delayed(Duration.zero);

      expect(_messageTypes(scripts), ['message.beforeReload']);
      controller.markReady();
      await afterReload;
      expect(_messageTypes(scripts), [
        'message.beforeReload',
        'message.afterReload',
      ]);
    });

    test('flushes when the controller attaches after ready', () async {
      final scripts = <String>[];
      final controller = WebViewBridgeController(
        javascriptReceiver: 'window.CodeRoamReceive',
      );
      controller.markReady();
      final pending = controller.send(
        const WebViewBridgeMessage(type: 'message.pending'),
      );

      controller.attach((source) async => scripts.add(source));
      await pending;

      expect(_messageTypes(scripts), ['message.pending']);
    });

    test('bounds pending messages', () async {
      final controller = WebViewBridgeController(
        javascriptReceiver: 'window.CodeRoamReceive',
        maxPendingMessages: 2,
      );

      final first = controller.send(
        const WebViewBridgeMessage(type: 'message.first'),
      );
      final second = controller.send(
        const WebViewBridgeMessage(type: 'message.second'),
      );

      await expectLater(
        controller.send(const WebViewBridgeMessage(type: 'message.third')),
        throwsStateError,
      );

      controller.attach((_) async {});
      controller.markReady();
      await Future.wait([first, second]);
    });

    test('serializes payloads as data rather than executable source', () async {
      String? script;
      final controller = WebViewBridgeController(
        javascriptReceiver: 'window.CodeRoamReceive',
      );
      controller.attach((source) async => script = source);
      controller.markReady();

      await controller.send(
        const WebViewBridgeMessage(
          type: 'terminal.write',
          payload: {'data': '"); globalThis.injected = true; //'},
        ),
      );

      final decoded = _messageFromScript(script!);
      expect(decoded['payload'], {
        'data': '"); globalThis.injected = true; //',
      });
    });

    test('fails queued messages when disposed', () async {
      final controller = WebViewBridgeController(
        javascriptReceiver: 'window.CodeRoamReceive',
      );
      final pending = controller.send(
        const WebViewBridgeMessage(type: 'message.pending'),
      );
      final pendingExpectation = expectLater(pending, throwsStateError);

      controller.dispose();

      await pendingExpectation;
      await expectLater(
        controller.send(const WebViewBridgeMessage(type: 'message.late')),
        throwsStateError,
      );
    });
  });
}

List<String> _messageTypes(List<String> scripts) {
  return scripts
      .map((script) => _messageFromScript(script)['type']! as String)
      .toList(growable: false);
}

Map<String, Object?> _messageFromScript(String script) {
  const prefix = 'window.CodeRoamReceive(';
  expect(script, startsWith(prefix));
  expect(script, endsWith(');'));

  return Map<String, Object?>.from(
    jsonDecode(script.substring(prefix.length, script.length - 2))
        as Map<String, dynamic>,
  );
}
