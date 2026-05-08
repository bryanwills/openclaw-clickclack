ALTER TABLE users ADD COLUMN handle TEXT NOT NULL DEFAULT '';

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_handle ON users(handle) WHERE handle <> '';
