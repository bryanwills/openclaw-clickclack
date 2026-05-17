CREATE INDEX IF NOT EXISTS idx_message_attachments_upload_message
  ON message_attachments(upload_id, message_id);
