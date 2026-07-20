# CodeRoam control plane

Go modular-monolith REST API.

## Runtime configuration

The API requires `POSTGRES_DSN`, `RELAY_REGION`, `OIDC_ISSUER`, `OIDC_AUDIENCE`, `OIDC_JWKS_URL`,
and `OIDC_SIGNING_ALGORITHM`. OIDC values are exact trust anchors; the process rejects invalid or
missing values before opening its HTTP listener. The mobile app is a public PKCE client, so no OIDC
client secret belongs in this service. Compose requires these values explicitly; copy `.env.example`
for the nonfunctional local placeholder, or provide the real registered provider values.

`GET /healthz` is public. `GET /v1/projects` and metadata-only `POST /v1/sessions` require a signed
OIDC ID token whose exact issuer/subject pair is already linked to a local user in
`auth.oidc_identities`. Invalid credentials return fixed errors and are never logged.
