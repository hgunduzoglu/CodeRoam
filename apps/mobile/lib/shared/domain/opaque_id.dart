import 'dart:math';

final class OpaqueId {
  const OpaqueId._(this.value);

  factory OpaqueId.parse(Object? value) {
    if (value is! String ||
        value.length != 32 ||
        !RegExp(r'^[0-9a-f]{32}$').hasMatch(value)) {
      throw const FormatException('Control-plane id is invalid.');
    }
    return OpaqueId._(value);
  }

  factory OpaqueId.generate() {
    final random = Random.secure();
    const hex = '0123456789abcdef';
    final encoded = StringBuffer();
    for (var index = 0; index < 16; index++) {
      final byte = random.nextInt(256);
      encoded
        ..write(hex[byte >> 4])
        ..write(hex[byte & 0x0f]);
    }
    return OpaqueId._(encoded.toString());
  }

  final String value;

  @override
  bool operator ==(Object other) => other is OpaqueId && other.value == value;

  @override
  int get hashCode => value.hashCode;

  @override
  String toString() => value;
}
