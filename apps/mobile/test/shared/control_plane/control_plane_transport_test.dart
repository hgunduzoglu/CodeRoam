import 'package:coderoam/shared/control_plane/control_plane_transport.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('request and response own immutable byte and header snapshots', () {
    final headers = {'Authorization': 'Bearer evidence'};
    final requestBytes = [1, 2, 3];
    final request = ControlPlaneHttpRequest(
      method: 'POST',
      uri: Uri.parse('https://control.example/v1/sessions'),
      headers: headers,
      body: requestBytes,
    );
    final responseBytes = [4, 5, 6];
    final response = ControlPlaneHttpResponse(
      statusCode: 200,
      contentType: 'application/json',
      body: responseBytes,
    );

    headers['Authorization'] = 'Bearer replaced';
    requestBytes[0] = 9;
    responseBytes[0] = 9;

    expect(request.headers['Authorization'], 'Bearer evidence');
    expect(request.body, [1, 2, 3]);
    expect(response.body, [4, 5, 6]);
    expect(() => request.headers['Extra'] = 'value', throwsUnsupportedError);
  });

  test('rejects unsafe requests before opening a connection', () async {
    final requests = <ControlPlaneHttpRequest>[
      ControlPlaneHttpRequest(
        method: 'GET',
        uri: Uri.parse('http://control.example/v1/projects'),
      ),
      ControlPlaneHttpRequest(
        method: 'GET',
        uri: Uri.parse('https://user:secret@control.example/v1/projects'),
      ),
      ControlPlaneHttpRequest(
        method: 'DELETE',
        uri: Uri.parse('https://control.example/v1/projects'),
      ),
      ControlPlaneHttpRequest(
        method: 'GET',
        uri: Uri.parse('https://control.example/v1/projects#fragment'),
      ),
      ControlPlaneHttpRequest(
        method: 'GET',
        uri: Uri.parse('https://control.example/v1/projects'),
        headers: {'Authorization': 'Bearer value\nInjected: yes'},
      ),
      ControlPlaneHttpRequest(
        method: 'GET',
        uri: Uri.parse('https://control.example/v1/projects'),
        headers: {'Host': 'attacker.example'},
      ),
      ControlPlaneHttpRequest(
        method: 'POST',
        uri: Uri.parse('https://control.example/v1/sessions'),
        body: List.filled(16 * 1024 + 1, 0),
      ),
    ];

    for (final request in requests) {
      final transport = IoControlPlaneTransport();
      await expectLater(
        transport.send(request),
        throwsA(isA<ControlPlaneTransportException>()),
      );
      transport.close();
    }
  });

  test('fails closed after close and rejects invalid bounds', () async {
    final transport = IoControlPlaneTransport();
    transport.close();
    transport.close();

    await expectLater(
      transport.send(
        ControlPlaneHttpRequest(
          method: 'GET',
          uri: Uri.parse('https://control.example/v1/projects'),
        ),
      ),
      throwsA(isA<ControlPlaneTransportException>()),
    );
    expect(
      () => IoControlPlaneTransport(timeout: Duration.zero),
      throwsArgumentError,
    );
    expect(
      () => IoControlPlaneTransport(maxResponseBytes: 0),
      throwsArgumentError,
    );
  });
}
