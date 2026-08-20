CREATE INDEX agents_owner_created_idx
  ON workspace.agents (user_id, created_at DESC, id);
