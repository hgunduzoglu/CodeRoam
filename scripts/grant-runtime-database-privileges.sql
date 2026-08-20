\if :{?runtime_role}
\else
  \echo 'runtime_role psql variable is required' >&2
  \quit 3
\endif

BEGIN;

GRANT USAGE ON SCHEMA auth, device, workspace, session TO :"runtime_role";

GRANT SELECT ON TABLE
  auth.users,
  auth.oidc_identities,
  device.devices,
  workspace.agents,
  workspace.environments,
  workspace.projects,
  session.sessions,
  session.pairing_attempts
TO :"runtime_role";

GRANT INSERT ON TABLE session.sessions TO :"runtime_role";

GRANT INSERT (
  id,
  agent_id,
  agent_static_public_key,
  agent_key_fingerprint,
  agent_display_name,
  agent_version,
  protocol_version,
  relay_region,
  bootstrap_credential_hash,
  expires_at,
  failed_attempt_count,
  state,
  created_at,
  updated_at
) ON TABLE session.pairing_attempts TO :"runtime_role";

GRANT INSERT (
  id,
  user_id,
  name,
  platform,
  static_public_key,
  public_key_fingerprint,
  paired_at
) ON TABLE device.devices TO :"runtime_role";

GRANT INSERT (
  id,
  user_id,
  name,
  static_public_key,
  public_key_fingerprint,
  version,
  created_at
) ON TABLE workspace.agents TO :"runtime_role";

-- PostgreSQL row-locking clauses require UPDATE on at least one column of
-- every locked table. Keep that permission away from ownership, trust, key,
-- revocation, and registered-root columns.
GRANT UPDATE (last_seen_at) ON TABLE device.devices TO :"runtime_role";
GRANT UPDATE (last_seen_at) ON TABLE workspace.agents TO :"runtime_role";
GRANT UPDATE (name) ON TABLE workspace.environments TO :"runtime_role";
GRANT UPDATE (last_opened_at) ON TABLE workspace.projects TO :"runtime_role";
GRANT UPDATE (result) ON TABLE session.sessions TO :"runtime_role";
GRANT UPDATE (
  failed_attempt_count,
  state,
  claimed_user_id,
  device_id,
  device_display_name,
  device_platform,
  device_static_public_key,
  device_key_fingerprint,
  claimed_at,
  mobile_channel_binding,
  mobile_confirmed_at,
  agent_channel_binding,
  agent_confirmed_at,
  consumed_at,
  updated_at
) ON TABLE session.pairing_attempts TO :"runtime_role";

COMMIT;
