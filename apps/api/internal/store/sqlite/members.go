package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"github.com/openclaw/clickclack/apps/api/internal/store"
)

func (s *Store) AddWorkspaceMember(ctx context.Context, workspaceID, userID, role string) error {
	if role == "" {
		role = "member"
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO workspace_members (workspace_id, user_id, role, created_at)
		VALUES (?, ?, ?, ?)`, workspaceID, userID, role, now())
	return err
}

func (s *Store) EnsureDefaultWorkspaceMember(ctx context.Context, userID string) (store.Workspace, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return store.Workspace{}, err
	}
	defer tx.Rollback()

	var workspace store.Workspace
	err = tx.QueryRowContext(ctx, `SELECT id, COALESCE(route_id, ''), name, slug, created_at FROM workspaces ORDER BY created_at LIMIT 1`).Scan(
		&workspace.ID,
		&workspace.RouteID,
		&workspace.Name,
		&workspace.Slug,
		&workspace.CreatedAt,
	)
	if err != nil && err != sql.ErrNoRows {
		return store.Workspace{}, err
	}
	if err == sql.ErrNoRows {
		workspace = store.Workspace{ID: newID("wsp"), Name: "ClickClack", Slug: "clickclack", CreatedAt: now()}
		insertedWorkspace := false
		for attempt := 0; attempt < routeIDInsertAttempts; attempt++ {
			workspaceRouteID, err := newRouteID('T')
			if err != nil {
				return store.Workspace{}, err
			}
			workspace.RouteID = workspaceRouteID
			if _, err := tx.ExecContext(ctx, `INSERT INTO workspaces (id, route_id, name, slug, created_at) VALUES (?, ?, ?, ?, ?)`, workspace.ID, workspace.RouteID, workspace.Name, workspace.Slug, workspace.CreatedAt); err != nil {
				if isRouteIDConflict(err) {
					continue
				}
				return store.Workspace{}, err
			}
			insertedWorkspace = true
			break
		}
		if !insertedWorkspace {
			return store.Workspace{}, errors.New("could not create workspace route_id after collision retries")
		}
		channelID := newID("chn")
		insertedChannel := false
		for attempt := 0; attempt < routeIDInsertAttempts; attempt++ {
			channelRouteID, err := newRouteID('C')
			if err != nil {
				return store.Workspace{}, err
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO channels (id, route_id, workspace_id, name, kind, created_at) VALUES (?, ?, ?, 'general', 'public', ?)`, channelID, channelRouteID, workspace.ID, workspace.CreatedAt); err != nil {
				if isRouteIDConflict(err) {
					continue
				}
				return store.Workspace{}, err
			}
			insertedChannel = true
			break
		}
		if !insertedChannel {
			return store.Workspace{}, errors.New("could not create channel route_id after collision retries")
		}
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO workspace_members (workspace_id, user_id, role, created_at)
		VALUES (?, ?, 'member', ?)`, workspace.ID, userID, now()); err != nil {
		return store.Workspace{}, err
	}
	return workspace, tx.Commit()
}
