import 'dart:async';
import 'dart:io';

import 'package:coderoam/shared/control_plane/control_plane_transport.dart';
import 'package:flutter_test/flutter_test.dart';

final _origin = Uri.parse('https://control.example');

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
    expect(() => request.body[0] = 9, throwsUnsupportedError);
    expect(() => response.body[0] = 9, throwsUnsupportedError);
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
        method: 'GET',
        uri: Uri.parse('https://attacker.example/v1/projects'),
      ),
      ControlPlaneHttpRequest(
        method: 'GET',
        uri: Uri.parse('https://sub.control.example/v1/projects'),
      ),
      ControlPlaneHttpRequest(
        method: 'GET',
        uri: Uri.parse('https://control.example:444/v1/projects'),
      ),
      ControlPlaneHttpRequest(
        method: 'GET',
        uri: Uri.parse('https://127.0.0.1/v1/projects'),
      ),
      ControlPlaneHttpRequest(
        method: 'GET',
        uri: Uri.parse('https://169.254.169.254/v1/projects'),
      ),
      ControlPlaneHttpRequest(
        method: 'GET',
        uri: Uri.parse('https://control.example./v1/projects'),
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
      final transport = IoControlPlaneTransport(origin: _origin);
      await expectLater(
        transport.send(request),
        throwsA(isA<ControlPlaneTransportException>()),
      );
      transport.close();
    }
  });

  test('fails closed after close and rejects invalid bounds', () async {
    final transport = IoControlPlaneTransport(origin: _origin);
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
      () => IoControlPlaneTransport(origin: _origin, timeout: Duration.zero),
      throwsArgumentError,
    );
    expect(
      () => IoControlPlaneTransport(origin: _origin, maxResponseBytes: 0),
      throwsArgumentError,
    );
    for (final origin in [
      Uri.parse('http://control.example'),
      Uri.parse('https://user:secret@control.example'),
      Uri.parse('https://control.example./'),
      Uri.parse('https://control.example/base'),
      Uri.parse('https://control.example?query=yes'),
    ]) {
      expect(
        () => IoControlPlaneTransport(origin: origin),
        throwsArgumentError,
      );
    }
  });

  test('aborts a request that opens after the deadline', () async {
    final opened = Completer<HttpClientRequest>();
    final transport = IoControlPlaneTransport(
      origin: _origin,
      timeout: const Duration(milliseconds: 1),
      openRequest: (_, _) => opened.future,
    );
    addTearDown(transport.close);
    final send = transport.send(
      ControlPlaneHttpRequest(
        method: 'GET',
        uri: Uri.parse('https://control.example/v1/projects'),
      ),
    );

    await expectLater(send, throwsA(isA<ControlPlaneTransportException>()));
    final lateRequest = _LateHttpClientRequest();
    opened.complete(lateRequest);
    await Future<void>.delayed(Duration.zero);

    expect(lateRequest.aborted, isTrue);
  });
}

final class _LateHttpClientRequest implements HttpClientRequest {
  bool aborted = false;

  @override
  void abort([Object? exception, StackTrace? stackTrace]) {
    aborted = true;
  }

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}
