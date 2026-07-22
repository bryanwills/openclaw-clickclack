CREATE TABLE IF NOT EXISTS user_appearance_preferences (
  user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  color_mode TEXT NOT NULL DEFAULT '',
  board_theme TEXT NOT NULL DEFAULT '',
  message_layout TEXT NOT NULL DEFAULT '',
  density TEXT NOT NULL DEFAULT ''
);
