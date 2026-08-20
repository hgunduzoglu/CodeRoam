import 'package:coderoam/features/pairing/infrastructure/secure_mobile_identity_provider.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  const channel = MethodChannel('test/mobile_identity_storage');
  const storage = NativeMobileIdentityStorage(channel: channel);

  tearDown(() async {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, null);
  });

  test('maps read and atomic creation to the native channel', () async {
    final calls = <MethodCall>[];
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (call) async {
          calls.add(call);
          return switch (call.method) {
            'read' => 'persisted-record',
            'createIfAbsent' => true,
            _ => throw MissingPluginException(),
          };
        });

    expect(await storage.read(), 'persisted-record');
    expect(await storage.createIfAbsent('new-record'), isTrue);
    expect(calls, hasLength(2));
    expect(calls[0].method, 'read');
    expect(calls[1].method, 'createIfAbsent');
    expect(calls[1].arguments, <String, Object?>{'value': 'new-record'});
  });

  test('fails closed when native creation returns no result', () async {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (_) async => null);

    await expectLater(
      storage.createIfAbsent('new-record'),
      throwsA(
        isA<PlatformException>().having(
          (error) => error.code,
          'code',
          'identity_storage_unavailable',
        ),
      ),
    );
  });
}
