ALTER TABLE workspaces ADD COLUMN route_id TEXT;
ALTER TABLE channels ADD COLUMN route_id TEXT;
ALTER TABLE direct_conversations ADD COLUMN route_id TEXT;
ALTER TABLE messages ADD COLUMN route_id TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS idx_workspaces_route_id
  ON workspaces(route_id)
  WHERE route_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_channels_workspace_route_id
  ON channels(workspace_id, route_id)
  WHERE route_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_direct_conversations_workspace_route_id
  ON direct_conversations(workspace_id, route_id)
  WHERE route_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_messages_workspace_route_id
  ON messages(workspace_id, route_id)
  WHERE route_id IS NOT NULL;

CREATE TRIGGER IF NOT EXISTS workspaces_route_id_immutable
BEFORE UPDATE OF route_id ON workspaces
WHEN OLD.route_id IS NOT NULL AND NEW.route_id IS NOT OLD.route_id
BEGIN
  SELECT RAISE(ABORT, 'workspace route_id is immutable');
END;

CREATE TRIGGER IF NOT EXISTS channels_route_id_immutable
BEFORE UPDATE OF route_id ON channels
WHEN OLD.route_id IS NOT NULL AND NEW.route_id IS NOT OLD.route_id
BEGIN
  SELECT RAISE(ABORT, 'channel route_id is immutable');
END;

CREATE TRIGGER IF NOT EXISTS direct_conversations_route_id_immutable
BEFORE UPDATE OF route_id ON direct_conversations
WHEN OLD.route_id IS NOT NULL AND NEW.route_id IS NOT OLD.route_id
BEGIN
  SELECT RAISE(ABORT, 'direct conversation route_id is immutable');
END;

CREATE TRIGGER IF NOT EXISTS messages_route_id_immutable
BEFORE UPDATE OF route_id ON messages
WHEN OLD.route_id IS NOT NULL AND NEW.route_id IS NOT OLD.route_id
BEGIN
  SELECT RAISE(ABORT, 'message route_id is immutable');
END;
