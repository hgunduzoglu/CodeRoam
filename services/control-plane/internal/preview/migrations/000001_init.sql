CREATE SCHEMA preview;
CREATE TABLE preview.sessions (
  id text PRIMARY KEY,
  owner_user_id text NOT NULL,
  project_id text NOT NULL,
  remote_port integer NOT NULL CHECK (remote_port BETWEEN 1 AND 65535),
  expires_at timestamptz NOT NULL,
  revoked_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE preview.invitations (
  id text PRIMARY KEY,
  preview_session_id text NOT NULL,
  recipient_user_id text,
  recipient_email text,
  capability_hash bytea NOT NULL,
  expires_at timestamptz NOT NULL,
  accepted_at timestamptz,
  revoked_at timestamptz
);
