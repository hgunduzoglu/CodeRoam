final RegExp _utcTimestampPattern = RegExp(
  r'^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2})(?:\.\d{1,9})?Z$',
);

DateTime parseControlPlaneUtcTimestamp(Object? value) {
  if (value is! String || value.length < 20 || value.length > 30) {
    throw const FormatException('Control-plane timestamp is invalid.');
  }
  final match = _utcTimestampPattern.firstMatch(value);
  final parsed = DateTime.tryParse(value);
  if (match == null ||
      parsed == null ||
      !parsed.isUtc ||
      parsed.year != int.parse(match.group(1)!) ||
      parsed.month != int.parse(match.group(2)!) ||
      parsed.day != int.parse(match.group(3)!) ||
      parsed.hour != int.parse(match.group(4)!) ||
      parsed.minute != int.parse(match.group(5)!) ||
      parsed.second != int.parse(match.group(6)!)) {
    throw const FormatException('Control-plane timestamp is invalid.');
  }
  return parsed;
}
