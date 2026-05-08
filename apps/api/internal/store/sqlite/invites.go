package sqlite

import (
	"context"

	"github.com/openclaw/clickclack/apps/api/internal/store"
)

func (s *Store) CreateInvite(ctx context.Context, workspaceID, createdBy string) (store.Invite, error) {
	if err := s.requireMembership(ctx, workspaceID, createdBy); err != nil {
		return store.Invite{}, err
	}
	invite := store.Invite{
		ID:          newID("inv"),
		WorkspaceID: workspaceID,
		Token:       newID("tok"),
		CreatedBy:   createdBy,
		CreatedAt:   now(),
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO invites (id, workspace_id, token, created_by, created_at)
		VALUES (?, ?, ?, ?, ?)`, invite.ID, invite.WorkspaceID, invite.Token, invite.CreatedBy, invite.CreatedAt)
	return invite, err
}
