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
  session.sessions
TO :"runtime_role";

GRANT INSERT ON TABLE session.sessions TO :"runtime_role";

-- PostgreSQL row-locking clauses require UPDATE on at least one column of
-- every locked table. Keep that permission away from ownership, trust, key,
-- revocation, and registered-root columns.
GRANT UPDATE (last_seen_at) ON TABLE device.devices TO :"runtime_role";
GRANT UPDATE (last_seen_at) ON TABLE workspace.agents TO :"runtime_role";
GRANT UPDATE (name) ON TABLE workspace.environments TO :"runtime_role";
GRANT UPDATE (last_opened_at) ON TABLE workspace.projects TO :"runtime_role";
GRANT UPDATE (result) ON TABLE session.sessions TO :"runtime_role";

COMMIT;
