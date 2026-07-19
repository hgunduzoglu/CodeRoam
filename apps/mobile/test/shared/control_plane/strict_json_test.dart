import 'package:coderoam/shared/control_plane/strict_json.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('decodes nested JSON with unique object keys', () {
    expect(
      decodeStrictJson(
        '{"projects":[{"id":"one","metadata":{"name":"CodeRoam"}}]}',
      ),
      {
        'projects': [
          {
            'id': 'one',
            'metadata': {'name': 'CodeRoam'},
          },
        ],
      },
    );
  });

  test('rejects duplicate keys at every nesting level and escaped aliases', () {
    for (final source in [
      '{"projects":[],"projects":[]}',
      '{"error":{"code":"one","code":"two"}}',
      '{"sessions":[{"capability":"relay","capability":"metadata-only"}]}',
      r'{"capability":"relay","cap\u0061bility":"metadata-only"}',
      '{"outer":{"id":"one"},"outer":{"id":"two"}}',
    ]) {
      expect(() => decodeStrictJson(source), throwsFormatException);
    }
  });

  test('rejects malformed and excessively nested JSON', () {
    final nested = '${'[' * 65}null${']' * 65}';
    for (final source in [
      '',
      '{"id":}',
      '{"id":1,}',
      '[1,]',
      '{"id":"unterminated}',
      '{"id":1}{}',
      nested,
    ]) {
      expect(() => decodeStrictJson(source), throwsFormatException);
    }
  });
}
