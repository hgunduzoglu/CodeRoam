CREATE SCHEMA integration;
CREATE TABLE integration.github_installations (
  id text PRIMARY KEY,
  user_id text NOT NULL,
  installation_id bigint NOT NULL UNIQUE,
  account_login text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
