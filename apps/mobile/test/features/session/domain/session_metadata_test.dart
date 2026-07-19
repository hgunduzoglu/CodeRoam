import 'package:coderoam/features/session/domain/session_metadata.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('parses metadata-only session response', () {
    final session = SessionMetadata.fromJson(_validSessionJson());

    expect(session.id.value, '1123456789abcdef0123456789abcdef');
    expect(session.deviceId.value, '2123456789abcdef0123456789abcdef');
    expect(session.agentId.value, '3123456789abcdef0123456789abcdef');
    expect(session.projectId.value, '4123456789abcdef0123456789abcdef');
    expect(session.relayRegion, 'eu-central-1');
    expect(session.startedAt, DateTime.utc(2026, 7, 20));
  });

  test('rejects capabilities and fields outside M2 metadata', () {
    final invalid = <Map<String, Object?>>[
      {..._validSessionJson(), 'ticket': 'forged-ticket'},
      {..._validSessionJson(), 'capability': 'relay-connect'},
      {..._validSessionJson(), 'relayRegion': '-eu-central-1'},
      {..._validSessionJson(), 'relayRegion': 'EU-CENTRAL-1'},
      {..._validSessionJson(), 'relayRegion': 'a' * 10000},
      {..._validSessionJson(), 'projectId': 'not-an-id'},
      {..._validSessionJson(), 'startedAt': '2026-07-20T00:00:00'},
      {..._validSessionJson(), 'startedAt': '2' * 10000},
      {..._validSessionJson()}..remove('deviceId'),
    ];

    for (final json in invalid) {
      expect(() => SessionMetadata.fromJson(json), throwsFormatException);
    }
  });
}

Map<String, Object?> _validSessionJson() => {
  'id': '1123456789abcdef0123456789abcdef',
  'deviceId': '2123456789abcdef0123456789abcdef',
  'agentId': '3123456789abcdef0123456789abcdef',
  'projectId': '4123456789abcdef0123456789abcdef',
  'relayRegion': 'eu-central-1',
  'startedAt': '2026-07-20T00:00:00Z',
  'capability': 'metadata-only',
};
