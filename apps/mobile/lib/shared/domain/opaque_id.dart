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

  final String value;

  @override
  bool operator ==(Object other) => other is OpaqueId && other.value == value;

  @override
  int get hashCode => value.hashCode;

  @override
  String toString() => value;
}
