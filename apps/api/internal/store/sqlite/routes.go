package sqlite

import (
	"context"
	"database/sql"
	"net/url"
	"strings"

	"github.com/openclaw/clickclack/apps/api/internal/store"
)

func (s *Store) ResolveRouteTarget(ctx context.Context, userID, workspaceRouteID, targetRouteID string) (store.RouteTarget, error) {
	workspaceRouteID = strings.TrimSpace(workspaceRouteID)
	targetRouteID = strings.TrimSpace(targetRouteID)
	if workspaceRouteID == "" || targetRouteID == "" {
		return store.RouteTarget{}, sql.ErrNoRows
	}
	workspace, err := scanWorkspace(s.db.QueryRowContext(ctx, `
		SELECT id, COALESCE(route_id, ''), name, slug, created_at
		FROM workspaces
		WHERE route_id = ?`, workspaceRouteID))
	if err != nil {
		return store.RouteTarget{}, err
	}
	if err := s.requireMembership(ctx, workspace.ID, userID); err != nil {
		return store.RouteTarget{}, sql.ErrNoRows
	}
	return s.resolveTargetInWorkspace(ctx, userID, workspace, targetRouteID, false)
}

func (s *Store) ResolveLegacyRouteTarget(ctx context.Context, userID, workspaceID, targetID string) (store.RouteTarget, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	targetID = strings.TrimSpace(targetID)
	if workspaceID == "" || targetID == "" {
		return store.RouteTarget{}, sql.ErrNoRows
	}
	workspace, err := s.GetWorkspace(ctx, workspaceID, userID)
	if err != nil {
		return store.RouteTarget{}, sql.ErrNoRows
	}
	return s.resolveTargetInWorkspace(ctx, userID, workspace, targetID, true)
}

func (s *Store) resolveTargetInWorkspace(ctx context.Context, userID string, workspace store.Workspace, targetID string, legacy bool) (store.RouteTarget, error) {
	switch {
	case (!legacy && strings.HasPrefix(targetID, "C")) || (legacy && strings.HasPrefix(targetID, "chn_")):
		return s.resolveChannelRouteTarget(ctx, workspace, targetID, legacy)
	case (!legacy && strings.HasPrefix(targetID, "D")) || (legacy && strings.HasPrefix(targetID, "dm_")):
		return s.resolveDirectRouteTarget(ctx, userID, workspace, targetID, legacy)
	case (!legacy && strings.HasPrefix(targetID, "M")) || (legacy && strings.HasPrefix(targetID, "msg_")):
		return s.resolveThreadRouteTarget(ctx, userID, workspace, targetID, legacy)
	default:
		return store.RouteTarget{}, sql.ErrNoRows
	}
}

func (s *Store) resolveChannelRouteTarget(ctx context.Context, workspace store.Workspace, targetID string, legacy bool) (store.RouteTarget, error) {
	var channel store.Channel
	var row *sql.Row
	if legacy {
		row = s.db.QueryRowContext(ctx, `
			SELECT id, COALESCE(route_id, ''), workspace_id, name, kind, created_at, archived_at
			FROM channels
			WHERE workspace_id = ? AND id = ?`, workspace.ID, targetID)
	} else {
		row = s.db.QueryRowContext(ctx, `
			SELECT id, COALESCE(route_id, ''), workspace_id, name, kind, created_at, archived_at
			FROM channels
			WHERE workspace_id = ? AND route_id = ?`, workspace.ID, targetID)
	}
	channel, err := scanChannel(row)
	if err != nil || channel.RouteID == "" {
		return store.RouteTarget{}, sql.ErrNoRows
	}
	return store.RouteTarget{
		WorkspaceID:      workspace.ID,
		WorkspaceRouteID: workspace.RouteID,
		TargetType:       "channel",
		TargetID:         channel.ID,
		TargetRouteID:    channel.RouteID,
		CanonicalPath:    routeCanonicalPath(workspace.RouteID, channel.RouteID),
	}, nil
}

func (s *Store) resolveDirectRouteTarget(ctx context.Context, userID string, workspace store.Workspace, targetID string, legacy bool) (store.RouteTarget, error) {
	var dm store.DirectConversation
	var row *sql.Row
	if legacy {
		row = s.db.QueryRowContext(ctx, `
			SELECT dc.id, COALESCE(dc.route_id, ''), dc.workspace_id, dc.created_at
			FROM direct_conversations dc
			JOIN direct_conversation_members dcm ON dcm.conversation_id = dc.id
			WHERE dc.workspace_id = ? AND dc.id = ? AND dcm.user_id = ?`, workspace.ID, targetID, userID)
	} else {
		row = s.db.QueryRowContext(ctx, `
			SELECT dc.id, COALESCE(dc.route_id, ''), dc.workspace_id, dc.created_at
			FROM direct_conversations dc
			JOIN direct_conversation_members dcm ON dcm.conversation_id = dc.id
			WHERE dc.workspace_id = ? AND dc.route_id = ? AND dcm.user_id = ?`, workspace.ID, targetID, userID)
	}
	if err := row.Scan(&dm.ID, &dm.RouteID, &dm.WorkspaceID, &dm.CreatedAt); err != nil || dm.RouteID == "" {
		return store.RouteTarget{}, sql.ErrNoRows
	}
	return store.RouteTarget{
		WorkspaceID:      workspace.ID,
		WorkspaceRouteID: workspace.RouteID,
		TargetType:       "direct",
		TargetID:         dm.ID,
		TargetRouteID:    dm.RouteID,
		CanonicalPath:    routeCanonicalPath(workspace.RouteID, dm.RouteID),
	}, nil
}

func (s *Store) resolveThreadRouteTarget(ctx context.Context, userID string, workspace store.Workspace, targetID string, legacy bool) (store.RouteTarget, error) {
	var root store.Message
	var err error
	if legacy {
		root, err = getMessage(ctx, s.db, targetID)
		if err == nil && root.WorkspaceID != workspace.ID {
			return store.RouteTarget{}, sql.ErrNoRows
		}
		if err == nil {
			err = s.requireMessageAccess(ctx, root, userID)
		}
		if err == nil {
			root, err = s.EnsureThreadRouteID(ctx, userID, root.ID)
		}
	} else {
		root, err = scanMessage(s.db.QueryRowContext(ctx, messageSelect()+`
			WHERE m.workspace_id = ? AND m.route_id = ? AND m.parent_message_id IS NULL`, workspace.ID, targetID))
		if err == nil {
			err = s.requireMessageAccess(ctx, root, userID)
		}
	}
	if err != nil || root.WorkspaceID != workspace.ID || root.RouteID == "" || root.ParentMessageID != nil {
		return store.RouteTarget{}, sql.ErrNoRows
	}
	target := store.RouteTarget{
		WorkspaceID:      workspace.ID,
		WorkspaceRouteID: workspace.RouteID,
		TargetType:       "thread",
		TargetID:         root.ID,
		TargetRouteID:    root.RouteID,
		CanonicalPath:    routeCanonicalPath(workspace.RouteID, root.RouteID),
	}
	if root.ChannelID != "" {
		parentRouteID, err := s.channelRouteID(ctx, workspace.ID, root.ChannelID)
		if err != nil || parentRouteID == "" {
			return store.RouteTarget{}, sql.ErrNoRows
		}
		target.ParentType = "channel"
		target.ParentID = root.ChannelID
		target.ParentRouteID = parentRouteID
		return target, nil
	}
	if root.DirectConversationID != "" {
		parentRouteID, err := s.directRouteID(ctx, userID, workspace.ID, root.DirectConversationID)
		if err != nil || parentRouteID == "" {
			return store.RouteTarget{}, sql.ErrNoRows
		}
		target.ParentType = "direct"
		target.ParentID = root.DirectConversationID
		target.ParentRouteID = parentRouteID
		return target, nil
	}
	return store.RouteTarget{}, sql.ErrNoRows
}

func (s *Store) channelRouteID(ctx context.Context, workspaceID, channelID string) (string, error) {
	var routeID string
	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(route_id, '')
		FROM channels
		WHERE workspace_id = ? AND id = ?`, workspaceID, channelID).Scan(&routeID)
	return routeID, err
}

func (s *Store) directRouteID(ctx context.Context, userID, workspaceID, conversationID string) (string, error) {
	var routeID string
	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(dc.route_id, '')
		FROM direct_conversations dc
		JOIN direct_conversation_members dcm ON dcm.conversation_id = dc.id
		WHERE dc.workspace_id = ? AND dc.id = ? AND dcm.user_id = ?`, workspaceID, conversationID, userID).Scan(&routeID)
	return routeID, err
}

func routeCanonicalPath(workspaceRouteID, targetRouteID string) string {
	return "/app/" + url.PathEscape(workspaceRouteID) + "/" + url.PathEscape(targetRouteID)
}
