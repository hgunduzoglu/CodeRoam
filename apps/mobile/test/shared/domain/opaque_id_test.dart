import 'package:coderoam/shared/domain/opaque_id.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('parses only canonical opaque ids', () {
    const encoded = '0123456789abcdef0123456789abcdef';
    final id = OpaqueId.parse(encoded);

    expect(id.value, encoded);
    expect(id, OpaqueId.parse(encoded));
    expect(id.toString(), encoded);
  });

  test('rejects malformed opaque ids', () {
    for (final value in <Object?>[
      null,
      1,
      '',
      '0123456789abcdef0123456789abcde',
      '0123456789ABCDEF0123456789ABCDEF',
      'g123456789abcdef0123456789abcdef',
      'a' * 10000,
    ]) {
      expect(() => OpaqueId.parse(value), throwsFormatException);
    }
  });
}
