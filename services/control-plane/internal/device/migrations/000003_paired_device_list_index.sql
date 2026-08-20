CREATE INDEX devices_owner_paired_idx
  ON device.devices (user_id, paired_at DESC, id);
