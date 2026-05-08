package sqlite

import "context"

func (s *Store) AddWorkspaceMember(ctx context.Context, workspaceID, userID, role string) error {
	if role == "" {
		role = "member"
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO workspace_members (workspace_id, user_id, role, created_at)
		VALUES (?, ?, ?, ?)`, workspaceID, userID, role, now())
	return err
}
