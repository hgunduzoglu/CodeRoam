LOCK TABLE device.devices IN ACCESS EXCLUSIVE MODE;

DO $migration$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM device.devices
    WHERE octet_length(static_public_key) <> 32
  ) THEN
    RAISE EXCEPTION 'device identity contains an invalid X25519 public key'
      USING ERRCODE = '23514', CONSTRAINT = 'devices_static_public_key_valid';
  END IF;

  IF EXISTS (
    SELECT 1
    FROM device.devices
    WHERE NOT (
      get_byte(static_public_key, 31) < 127
      OR (
        get_byte(static_public_key, 31) = 127
        AND (
          substring(static_public_key FROM 2 FOR 30) <> decode(repeat('ff', 30), 'hex')
          OR get_byte(static_public_key, 0) < 237
        )
      )
    )
  ) THEN
    RAISE EXCEPTION 'device identity contains a noncanonical X25519 public key'
      USING ERRCODE = '23514', CONSTRAINT = 'devices_static_public_key_valid';
  END IF;

  IF EXISTS (
    SELECT 1
    FROM device.devices
    WHERE static_public_key IN (
      decode(repeat('00', 32), 'hex'),
      decode('01' || repeat('00', 31), 'hex'),
      decode('e0eb7a7c3b41b8ae1656e3faf19fc46ada098deb9c32b1fd866205165f49b800', 'hex'),
      decode('5f9c95bca3508c24b1d0b1559c83ef5b04445cc4581c8e86d8224eddd09f1157', 'hex'),
      decode('ec' || repeat('ff', 30) || '7f', 'hex')
    )
  ) THEN
    RAISE EXCEPTION 'device identity contains an unusable X25519 public key'
      USING ERRCODE = '23514', CONSTRAINT = 'devices_static_public_key_valid';
  END IF;

  IF EXISTS (
    SELECT static_public_key
    FROM device.devices
    GROUP BY static_public_key
    HAVING count(*) > 1
  ) THEN
    RAISE EXCEPTION 'device identities contain a duplicate X25519 public key'
      USING ERRCODE = '23505', CONSTRAINT = 'devices_static_public_key_unique';
  END IF;
END
$migration$;

UPDATE device.devices
SET public_key_fingerprint =
  'x25519-sha256:' || encode(sha256(static_public_key), 'hex');

ALTER TABLE device.devices
  ADD CONSTRAINT devices_static_public_key_length
    CHECK (octet_length(static_public_key) = 32),
  ADD CONSTRAINT devices_static_public_key_canonical
    CHECK (
      CASE
        WHEN octet_length(static_public_key) = 32 THEN
          get_byte(static_public_key, 31) < 127
          OR (
            get_byte(static_public_key, 31) = 127
            AND (
              substring(static_public_key FROM 2 FOR 30) <> decode(repeat('ff', 30), 'hex')
              OR get_byte(static_public_key, 0) < 237
            )
          )
        ELSE false
      END
    ),
  ADD CONSTRAINT devices_static_public_key_usable
    CHECK (
      static_public_key NOT IN (
        decode(repeat('00', 32), 'hex'),
        decode('01' || repeat('00', 31), 'hex'),
        decode('e0eb7a7c3b41b8ae1656e3faf19fc46ada098deb9c32b1fd866205165f49b800', 'hex'),
        decode('5f9c95bca3508c24b1d0b1559c83ef5b04445cc4581c8e86d8224eddd09f1157', 'hex'),
        decode('ec' || repeat('ff', 30) || '7f', 'hex')
      )
    ),
  ADD CONSTRAINT devices_public_key_fingerprint_shape
    CHECK (public_key_fingerprint ~ '^x25519-sha256:[0-9a-f]{64}$'),
  ADD CONSTRAINT devices_public_key_fingerprint_matches_key
    CHECK (
      public_key_fingerprint =
        'x25519-sha256:' || encode(sha256(static_public_key), 'hex')
    );
