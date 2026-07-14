CREATE SCHEMA IF NOT EXISTS integration;
CREATE TABLE IF NOT EXISTS integration.github_installations (
  id text PRIMARY KEY,
  user_id text NOT NULL,
  installation_id bigint NOT NULL UNIQUE,
  account_login text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
