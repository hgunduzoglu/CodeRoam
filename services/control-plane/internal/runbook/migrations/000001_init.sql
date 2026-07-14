CREATE SCHEMA IF NOT EXISTS runbook;
CREATE TABLE IF NOT EXISTS runbook.definitions (
  id text PRIMARY KEY,
  user_id text NOT NULL,
  environment_id text NOT NULL,
  name text NOT NULL,
  definition jsonb NOT NULL,
  enabled boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS runbook.executions (
  id text PRIMARY KEY,
  user_id text NOT NULL,
  device_id text NOT NULL,
  runbook_id text NOT NULL,
  reason text NOT NULL,
  started_at timestamptz NOT NULL,
  completed_at timestamptz,
  result text
);
