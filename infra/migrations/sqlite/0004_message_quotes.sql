ALTER TABLE messages ADD COLUMN quoted_message_id TEXT REFERENCES messages(id) ON DELETE SET NULL;
ALTER TABLE messages ADD COLUMN quoted_body_snapshot TEXT NOT NULL DEFAULT '';
ALTER TABLE messages ADD COLUMN quoted_author_id TEXT REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_messages_quoted_message_id
  ON messages(quoted_message_id) WHERE quoted_message_id IS NOT NULL;
