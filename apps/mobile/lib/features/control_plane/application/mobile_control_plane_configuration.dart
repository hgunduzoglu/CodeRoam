import 'package:coderoam/features/auth/domain/oidc_client_configuration.dart';
import 'package:coderoam/shared/control_plane/control_plane_transport.dart';
import 'package:coderoam/shared/domain/opaque_id.dart';

final class MobileControlPlaneConfiguration {
  const MobileControlPlaneConfiguration._({
    required this.origin,
    required this.oidc,
    required this.deviceId,
  });

  final ControlPlaneOrigin origin;
  final OidcClientConfiguration oidc;
  final OpaqueId deviceId;

  factory MobileControlPlaneConfiguration.fromValues({
    required String controlPlaneOrigin,
    required String oidcIssuer,
    required String oidcClientId,
    required String deviceId,
  }) {
    try {
      return MobileControlPlaneConfiguration._(
        origin: ControlPlaneOrigin.parse(Uri.parse(controlPlaneOrigin)),
        oidc: OidcClientConfiguration.fromValues(
          issuer: oidcIssuer,
          clientId: oidcClientId,
        ),
        deviceId: OpaqueId.parse(deviceId),
      );
    } catch (_) {
      throw const FormatException(
        'Mobile control-plane configuration is invalid.',
      );
    }
  }
}
