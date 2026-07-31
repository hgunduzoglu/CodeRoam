LOCK TABLE session.pairing_attempts IN ACCESS EXCLUSIVE MODE;
DELETE FROM session.pairing_attempts;

ALTER TABLE session.pairing_attempts
  RENAME COLUMN agent_fingerprint TO agent_key_fingerprint;
ALTER TABLE session.pairing_attempts
  RENAME COLUMN attempt_count TO failed_attempt_count;

ALTER TABLE session.pairing_attempts
  ALTER COLUMN failed_attempt_count DROP DEFAULT,
  ALTER COLUMN created_at DROP DEFAULT,
  ADD COLUMN agent_static_public_key bytea NOT NULL,
  ADD COLUMN agent_display_name text NOT NULL,
  ADD COLUMN agent_version text NOT NULL,
  ADD COLUMN protocol_version integer NOT NULL,
  ADD COLUMN relay_region text NOT NULL,
  ADD COLUMN bootstrap_credential_hash bytea NOT NULL,
  ADD COLUMN state text NOT NULL,
  ADD COLUMN claimed_user_id text,
  ADD COLUMN device_id text,
  ADD COLUMN device_display_name text,
  ADD COLUMN device_platform text,
  ADD COLUMN device_static_public_key bytea,
  ADD COLUMN device_key_fingerprint text,
  ADD COLUMN claimed_at timestamptz,
  ADD COLUMN mobile_channel_binding bytea,
  ADD COLUMN mobile_confirmed_at timestamptz,
  ADD COLUMN agent_channel_binding bytea,
  ADD COLUMN agent_confirmed_at timestamptz,
  ADD COLUMN updated_at timestamptz NOT NULL;

ALTER TABLE session.pairing_attempts
  ADD CONSTRAINT pairing_attempts_id_shape
    CHECK (id ~ '^[0-9a-f]{32}$'),
  ADD CONSTRAINT pairing_attempts_agent_key_length
    CHECK (octet_length(agent_static_public_key) = 32),
  ADD CONSTRAINT pairing_attempts_agent_fingerprint_shape
    CHECK (agent_key_fingerprint ~ '^x25519-sha256:[0-9a-f]{64}$'),
  ADD CONSTRAINT pairing_attempts_agent_name_shape
    CHECK (
      agent_display_name = btrim(agent_display_name)
      AND char_length(agent_display_name) BETWEEN 1 AND 128
      AND octet_length(agent_display_name) <= 512
    ),
  ADD CONSTRAINT pairing_attempts_agent_version_shape
    CHECK (
      agent_version = btrim(agent_version)
      AND octet_length(agent_version) BETWEEN 1 AND 64
    ),
  ADD CONSTRAINT pairing_attempts_protocol_version
    CHECK (protocol_version = 1),
  ADD CONSTRAINT pairing_attempts_relay_region_shape
    CHECK (
      octet_length(relay_region) BETWEEN 1 AND 64
      AND relay_region ~ '^[a-z0-9]+(-[a-z0-9]+)*$'
    ),
  ADD CONSTRAINT pairing_attempts_bootstrap_hash_length
    CHECK (octet_length(bootstrap_credential_hash) = 32),
  ADD CONSTRAINT pairing_attempts_failure_count_range
    CHECK (failed_attempt_count BETWEEN 0 AND 8),
  ADD CONSTRAINT pairing_attempts_lifetime
    CHECK (
      expires_at > created_at
      AND expires_at <= created_at + interval '5 minutes'
      AND updated_at >= created_at
      AND updated_at <= expires_at
    ),
  ADD CONSTRAINT pairing_attempts_state_value
    CHECK (state IN ('open', 'claimed', 'confirming', 'consumed')),
  ADD CONSTRAINT pairing_attempts_claim_shape
    CHECK (
      (
        claimed_user_id IS NULL
        AND device_id IS NULL
        AND device_display_name IS NULL
        AND device_platform IS NULL
        AND device_static_public_key IS NULL
        AND device_key_fingerprint IS NULL
        AND claimed_at IS NULL
      )
      OR
      (
        claimed_user_id IS NOT NULL
        AND claimed_user_id ~ '^[0-9a-f]{32}$'
        AND device_id IS NOT NULL
        AND device_id ~ '^[0-9a-f]{32}$'
        AND device_display_name IS NOT NULL
        AND device_display_name = btrim(device_display_name)
        AND char_length(device_display_name) BETWEEN 1 AND 128
        AND octet_length(device_display_name) <= 512
        AND device_platform IS NOT NULL
        AND device_platform IN ('ios', 'ipados', 'android')
        AND device_static_public_key IS NOT NULL
        AND octet_length(device_static_public_key) = 32
        AND device_key_fingerprint IS NOT NULL
        AND device_key_fingerprint ~ '^x25519-sha256:[0-9a-f]{64}$'
        AND claimed_at IS NOT NULL
      )
    ),
  ADD CONSTRAINT pairing_attempts_mobile_confirmation_shape
    CHECK (
      (mobile_channel_binding IS NULL AND mobile_confirmed_at IS NULL)
      OR
      (
        mobile_channel_binding IS NOT NULL
        AND octet_length(mobile_channel_binding) = 32
        AND mobile_confirmed_at IS NOT NULL
      )
    ),
  ADD CONSTRAINT pairing_attempts_agent_confirmation_shape
    CHECK (
      (agent_channel_binding IS NULL AND agent_confirmed_at IS NULL)
      OR
      (
        agent_channel_binding IS NOT NULL
        AND octet_length(agent_channel_binding) = 32
        AND agent_confirmed_at IS NOT NULL
      )
    ),
  ADD CONSTRAINT pairing_attempts_channel_binding_match
    CHECK (
      mobile_channel_binding IS NULL
      OR agent_channel_binding IS NULL
      OR mobile_channel_binding = agent_channel_binding
    ),
  ADD CONSTRAINT pairing_attempts_transition_timestamps
    CHECK (
      (claimed_at IS NULL OR (claimed_at >= created_at AND claimed_at <= expires_at))
      AND (claimed_at IS NULL OR updated_at >= claimed_at)
      AND (
        mobile_confirmed_at IS NULL
        OR (mobile_confirmed_at >= claimed_at AND mobile_confirmed_at <= expires_at)
      )
      AND (mobile_confirmed_at IS NULL OR updated_at >= mobile_confirmed_at)
      AND (
        agent_confirmed_at IS NULL
        OR (agent_confirmed_at >= claimed_at AND agent_confirmed_at <= expires_at)
      )
      AND (agent_confirmed_at IS NULL OR updated_at >= agent_confirmed_at)
      AND (
        consumed_at IS NULL
        OR (
          consumed_at >= mobile_confirmed_at
          AND consumed_at >= agent_confirmed_at
          AND consumed_at <= expires_at
          AND updated_at >= consumed_at
        )
      )
    ),
  ADD CONSTRAINT pairing_attempts_state_shape
    CHECK (
      state NOT IN ('open', 'claimed', 'confirming', 'consumed')
      OR (
        (
          state = 'open'
          AND claimed_user_id IS NULL
          AND mobile_channel_binding IS NULL
          AND agent_channel_binding IS NULL
          AND consumed_at IS NULL
        )
        OR
        (
          state = 'claimed'
          AND claimed_user_id IS NOT NULL
          AND mobile_channel_binding IS NULL
          AND agent_channel_binding IS NULL
          AND consumed_at IS NULL
        )
        OR
        (
          state = 'confirming'
          AND claimed_user_id IS NOT NULL
          AND (mobile_channel_binding IS NOT NULL OR agent_channel_binding IS NOT NULL)
          AND consumed_at IS NULL
        )
        OR
        (
          state = 'consumed'
          AND claimed_user_id IS NOT NULL
          AND mobile_channel_binding IS NOT NULL
          AND agent_channel_binding IS NOT NULL
          AND consumed_at IS NOT NULL
        )
      )
    );

CREATE INDEX pairing_attempts_expiry_idx
  ON session.pairing_attempts(expires_at)
  WHERE consumed_at IS NULL;
