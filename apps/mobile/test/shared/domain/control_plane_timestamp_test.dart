import 'package:coderoam/shared/domain/control_plane_timestamp.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('parses canonical RFC3339 UTC timestamps', () {
    expect(
      parseControlPlaneUtcTimestamp('2026-07-20T01:02:03Z'),
      DateTime.utc(2026, 7, 20, 1, 2, 3),
    );
    expect(
      parseControlPlaneUtcTimestamp('2026-07-20T01:02:03.123456789Z'),
      DateTime.utc(2026, 7, 20, 1, 2, 3, 123, 456),
    );
  });

  test('rejects normalized, alternate, and oversized timestamps', () {
    for (final value in <Object?>[
      null,
      '2026-02-30T00:00:00Z',
      '2026-13-01T00:00:00Z',
      '2026-07-20T24:00:00Z',
      '2026-07-20 00:00:00Z',
      '20260720T000000Z',
      '2026-07-20T00:00:00.1234567890Z',
      '2026-07-20T00:00:00+00:00',
      '2' * 10000,
    ]) {
      expect(() => parseControlPlaneUtcTimestamp(value), throwsFormatException);
    }
  });
}
