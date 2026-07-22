import 'package:coderoam/features/control_plane/application/mobile_control_plane_configuration.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('retains exact production bootstrap configuration', () {
    final configuration = MobileControlPlaneConfiguration.fromValues(
      controlPlaneOrigin: 'https://control.example.test',
      oidcIssuer: 'https://identity.example.test/tenant',
      oidcClientId: 'coderoam-mobile',
      deviceId: '4123456789abcdef0123456789abcdef',
    );

    expect(configuration.origin.uri, Uri.parse('https://control.example.test'));
    expect(configuration.oidc.issuer, 'https://identity.example.test/tenant');
    expect(configuration.oidc.clientId, 'coderoam-mobile');
    expect(configuration.deviceId.value, '4123456789abcdef0123456789abcdef');
  });

  test('fails closed when any bootstrap value is invalid', () {
    final cases = <Map<String, String>>[
      {
        'origin': '',
        'issuer': 'https://identity.example.test',
        'clientId': 'coderoam-mobile',
        'deviceId': '4123456789abcdef0123456789abcdef',
      },
      {
        'origin': 'http://control.example.test',
        'issuer': 'https://identity.example.test',
        'clientId': 'coderoam-mobile',
        'deviceId': '4123456789abcdef0123456789abcdef',
      },
      {
        'origin': 'https://control.example.test',
        'issuer': 'http://identity.example.test',
        'clientId': 'coderoam-mobile',
        'deviceId': '4123456789abcdef0123456789abcdef',
      },
      {
        'origin': 'https://control.example.test',
        'issuer': 'https://identity.example.test',
        'clientId': '',
        'deviceId': '4123456789abcdef0123456789abcdef',
      },
      {
        'origin': 'https://control.example.test',
        'issuer': 'https://identity.example.test',
        'clientId': 'coderoam-mobile',
        'deviceId': 'not-a-device-id',
      },
    ];

    for (final values in cases) {
      expect(
        () => MobileControlPlaneConfiguration.fromValues(
          controlPlaneOrigin: values['origin']!,
          oidcIssuer: values['issuer']!,
          oidcClientId: values['clientId']!,
          deviceId: values['deviceId']!,
        ),
        throwsFormatException,
        reason: values.toString(),
      );
    }
  });
}
