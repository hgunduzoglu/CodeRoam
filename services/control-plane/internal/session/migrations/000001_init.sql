CREATE SCHEMA IF NOT EXISTS session;
CREATE TABLE IF NOT EXISTS session.pairing_attempts (
  id text PRIMARY KEY,
  agent_fingerprint text NOT NULL,
  expires_at timestamptz NOT NULL,
  attempt_count integer NOT NULL DEFAULT 0,
  consumed_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS session.sessions (
  id text PRIMARY KEY,
  user_id text NOT NULL,
  device_id text NOT NULL,
  agent_id text NOT NULL,
  project_id text,
  relay_region text NOT NULL,
  started_at timestamptz NOT NULL,
  ended_at timestamptz,
  result text
);
