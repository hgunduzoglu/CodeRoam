# CodeRoam mobile

Generate iOS and Android platform files once:

```bash
make bootstrap-mobile
```

Milestone 0 is implemented before backend-heavy product work. See
`../../docs/touch-ux-spike.md`.

## Milestone 2 production sign-in

The mobile app is a public OIDC Authorization Code client. Register this exact redirect URI with
the provider:

```text
dev.coderoam.coderoam:/oauthredirect
```

Enable Authorization Code with PKCE, the `openid` and `offline_access` scopes, and ID tokens whose
audience is the same public client ID configured as the control plane's `OIDC_AUDIENCE`. Do not
create or embed a mobile client secret.

Build or run the app with the HTTPS control-plane origin, exact OIDC issuer, public client ID, and
an already-registered active device ID:

```bash
flutter run \
  --dart-define=CODEROAM_CONTROL_PLANE_ORIGIN=https://control.example \
  --dart-define=CODEROAM_OIDC_ISSUER=https://identity.example/realms/coderoam \
  --dart-define=CODEROAM_OIDC_CLIENT_ID=coderoam-mobile \
  --dart-define=CODEROAM_DEVICE_ID=0123456789abcdef0123456789abcdef
```

These values are compiled into the app and are not secrets. The M2 device ID is a temporary
bootstrap selector, not proof of device identity; M3 pairing replaces it with registration and
pinned device identity.

Before real-device acceptance, the database must contain:

- an active user and an `auth.oidc_identities` row with the provider's exact case-sensitive issuer
  and subject;
- an active `device.devices` row owned by that user and matching `CODEROAM_DEVICE_ID`;
- an active owner-scoped agent, environment, and project to exercise project and session metadata.

The control plane must be reachable from the physical device over HTTPS with the matching OIDC
trust anchors described in `../../services/control-plane/README.md`.

### M2 physical-device acceptance

1. Launch a build containing the four values above and confirm the fixed sign-in screen appears.
2. Sign in through the registered system-browser OIDC flow and return through the CodeRoam redirect.
3. Confirm the authenticated project catalog loads without exposing the ID or refresh token.
4. Open the owned project and create metadata for the registered device. Confirm the result says no
   relay capability was issued and the local touch workspace still opens.
5. Sign out from the project catalog. Confirm the catalog disappears, the sign-in screen returns,
   and relaunching the app does not restore the signed-out session.
6. Revoke the device or use a foreign device ID and confirm session creation fails without exposing
   provider, credential, or ownership details.
