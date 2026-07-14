CREATE SCHEMA IF NOT EXISTS device;
CREATE TABLE IF NOT EXISTS device.devices (
  id text PRIMARY KEY,
  user_id text NOT NULL,
  name text NOT NULL,
  platform text NOT NULL,
  static_public_key bytea NOT NULL,
  public_key_fingerprint text NOT NULL UNIQUE,
  paired_at timestamptz NOT NULL,
  last_seen_at timestamptz,
  revoked_at timestamptz
);
CREATE INDEX IF NOT EXISTS devices_user_idx ON device.devices(user_id);
