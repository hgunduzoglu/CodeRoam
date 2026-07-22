# CodeRoam control plane

Go modular-monolith REST API.

## Runtime configuration

The API requires `POSTGRES_DSN`, `RELAY_REGION`, `OIDC_ISSUER`, `OIDC_AUDIENCE`, `OIDC_JWKS_URL`,
and `OIDC_SIGNING_ALGORITHM`. OIDC values are exact trust anchors; the process rejects invalid or
missing values before opening its HTTP listener. The mobile app is a public PKCE client, so no OIDC
client secret belongs in this service. Compose requires these values explicitly; copy `.env.example`
for the nonfunctional local placeholder, or provide the real registered provider values.

`GET /health` is public; its path intentionally avoids Cloud Run's reserved URL paths ending in
`z`. `GET /v1/projects` and metadata-only `POST /v1/sessions` require a signed OIDC ID token whose
exact issuer/subject pair is already linked to a local user in `auth.oidc_identities`. Invalid
credentials return fixed errors and are never logged.

## Runtime database privileges

Database roles belong to each deployment, so module migrations do not create or name them. After
applying migrations, a database administrator grants the control-plane runtime role its bounded
access with:

```bash
psql "$MIGRATOR_DSN" -v ON_ERROR_STOP=1 \
  -v runtime_role=coderoam_app \
  -f scripts/grant-runtime-database-privileges.sql
```

The script is safe to repeat. Its column-level `UPDATE` grants exist only because PostgreSQL
requires update privilege on at least one column for `FOR SHARE`; ownership, keys, revocation,
registered roots, and other trust fields remain read-only to the runtime role. To roll the grant
back, revoke the exact schema, table, and column privileges listed in the script from that role.
