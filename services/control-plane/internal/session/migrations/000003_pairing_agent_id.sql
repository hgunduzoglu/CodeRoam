LOCK TABLE session.pairing_attempts IN ACCESS EXCLUSIVE MODE;

-- Pre-v3 attempts do not carry the agent's durable opaque identifier and
-- therefore cannot be completed without inventing identity after bootstrap.
-- Pairing attempts are short-lived, so invalidate them instead of upgrading
-- incomplete trust state.
DELETE FROM session.pairing_attempts;

ALTER TABLE session.pairing_attempts
  ADD COLUMN agent_id text NOT NULL,
  ADD CONSTRAINT pairing_attempts_agent_id_shape
    CHECK (agent_id ~ '^[0-9a-f]{32}$');
