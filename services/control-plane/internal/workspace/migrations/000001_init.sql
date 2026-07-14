CREATE SCHEMA IF NOT EXISTS workspace;
CREATE TABLE IF NOT EXISTS workspace.agents (
  id text PRIMARY KEY,
  user_id text NOT NULL,
  name text NOT NULL,
  static_public_key bytea NOT NULL,
  public_key_fingerprint text NOT NULL UNIQUE,
  version text NOT NULL,
  last_seen_at timestamptz,
  revoked_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS workspace.environments (
  id text PRIMARY KEY,
  user_id text NOT NULL,
  agent_id text NOT NULL,
  name text NOT NULL,
  provider text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS workspace.projects (
  id text PRIMARY KEY,
  user_id text NOT NULL,
  environment_id text NOT NULL,
  name text NOT NULL,
  root_path text NOT NULL,
  repository_url text,
  last_opened_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS projects_user_idx ON workspace.projects(user_id);
