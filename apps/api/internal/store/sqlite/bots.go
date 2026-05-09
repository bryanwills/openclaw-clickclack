package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"
	"strings"

	"github.com/openclaw/clickclack/apps/api/internal/store"
)

var botScopeBundles = map[string][]string{
	"bot:read": {
		"workspaces:read",
		"channels:read",
		"messages:read",
		"threads:read",
		"dms:read",
		"realtime:read",
		"profile:read",
	},
	"bot:write": {
		"workspaces:read",
		"channels:read",
		"messages:read",
		"messages:write",
		"threads:read",
		"threads:write",
		"dms:read",
		"dms:write",
		"realtime:read",
		"uploads:write",
		"profile:read",
	},
	"bot:admin": {
		"workspaces:read",
		"channels:read",
		"channels:write",
		"messages:read",
		"messages:write",
		"threads:read",
		"threads:write",
		"dms:read",
		"dms:write",
		"realtime:read",
		"uploads:write",
		"profile:read",
	},
}

var botAllowedScopes = []string{
	"workspaces:read",
	"channels:read",
	"channels:write",
	"messages:read",
	"messages:write",
	"threads:read",
	"threads:write",
	"dms:read",
	"dms:write",
	"realtime:read",
	"uploads:write",
	"profile:read",
}

func (s *Store) CreateBot(ctx context.Context, input store.CreateBotInput) (store.User, store.BotToken, error) {
	workspaceID := strings.TrimSpace(input.WorkspaceID)
	if workspaceID == "" {
		return store.User{}, store.BotToken{}, errors.New("workspace is required")
	}
	displayName := strings.TrimSpace(input.DisplayName)
	if displayName == "" {
		return store.User{}, store.BotToken{}, errors.New("display_name is required")
	}
	if len(displayName) > 80 {
		return store.User{}, store.BotToken{}, errors.New("display_name is too long")
	}
	handle, err := normalizeHandle(input.Handle)
	if err != nil {
		return store.User{}, store.BotToken{}, err
	}
	avatarURL, err := normalizeAvatarURL(input.AvatarURL)
	if err != nil {
		return store.User{}, store.BotToken{}, err
	}
	scopes, err := normalizeBotScopes(input.Scopes)
	if err != nil {
		return store.User{}, store.BotToken{}, err
	}
	tokenName := strings.TrimSpace(input.TokenName)
	if tokenName == "" {
		tokenName = "default"
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return store.User{}, store.BotToken{}, err
	}
	defer tx.Rollback()
	if input.OwnerUserID != "" {
		owner, err := scanUser(tx.QueryRowContext(ctx, `SELECT id, kind, owner_user_id, display_name, handle, avatar_url, created_at FROM users WHERE id = ?`, input.OwnerUserID))
		if err != nil {
			return store.User{}, store.BotToken{}, err
		}
		if owner.Kind == "bot" {
			return store.User{}, store.BotToken{}, errors.New("bot owner must be a human")
		}
		if err := requireMembershipTx(ctx, tx, workspaceID, owner.ID); err != nil {
			return store.User{}, store.BotToken{}, errors.New("bot owner is not a workspace member")
		}
	}
	bot := store.User{
		ID:          newID("usr"),
		Kind:        "bot",
		OwnerUserID: strings.TrimSpace(input.OwnerUserID),
		DisplayName: displayName,
		Handle:      handle,
		AvatarURL:   avatarURL,
		CreatedAt:   now(),
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO users (id, kind, owner_user_id, display_name, handle, avatar_url, created_at)
		VALUES (?, 'bot', ?, ?, ?, ?, ?)`, bot.ID, nullableString(bot.OwnerUserID), bot.DisplayName, bot.Handle, bot.AvatarURL, bot.CreatedAt); err != nil {
		if strings.Contains(err.Error(), "idx_users_handle") || strings.Contains(err.Error(), "users.handle") {
			return store.User{}, store.BotToken{}, errors.New("handle is already taken")
		}
		return store.User{}, store.BotToken{}, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role, created_at)
		VALUES (?, ?, 'bot', ?)`, workspaceID, bot.ID, bot.CreatedAt); err != nil {
		return store.User{}, store.BotToken{}, err
	}
	token := newID("ccb")
	scopesJSON, err := json.Marshal(scopes)
	if err != nil {
		return store.User{}, store.BotToken{}, err
	}
	botToken := store.BotToken{
		ID:          newID("btok"),
		Token:       token,
		BotUserID:   bot.ID,
		WorkspaceID: workspaceID,
		OwnerUserID: bot.OwnerUserID,
		Name:        tokenName,
		Scopes:      scopes,
		CreatedBy:   strings.TrimSpace(input.CreatedBy),
		CreatedAt:   bot.CreatedAt,
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO bot_tokens (id, token_hash, bot_user_id, workspace_id, owner_user_id, name, scopes_json, created_by, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		botToken.ID,
		hashBotToken(token),
		botToken.BotUserID,
		botToken.WorkspaceID,
		nullableString(botToken.OwnerUserID),
		botToken.Name,
		string(scopesJSON),
		nullableString(botToken.CreatedBy),
		botToken.CreatedAt,
	)
	if err != nil {
		return store.User{}, store.BotToken{}, err
	}
	return bot, botToken, tx.Commit()
}

func (s *Store) GetBotTokenAuth(ctx context.Context, token string) (store.BotTokenAuth, error) {
	token = strings.TrimSpace(token)
	if !strings.HasPrefix(token, "ccb_") {
		return store.BotTokenAuth{}, sql.ErrNoRows
	}
	var auth store.BotTokenAuth
	var scopesJSON string
	var owner sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT u.id, u.kind, u.owner_user_id, u.display_name, u.handle, u.avatar_url, u.created_at,
		       bt.id, bt.workspace_id, bt.scopes_json
		FROM bot_tokens bt
		JOIN users u ON u.id = bt.bot_user_id
		WHERE bt.token_hash = ? AND bt.revoked_at IS NULL`, hashBotToken(token)).Scan(
		&auth.User.ID,
		&auth.User.Kind,
		&owner,
		&auth.User.DisplayName,
		&auth.User.Handle,
		&auth.User.AvatarURL,
		&auth.User.CreatedAt,
		&auth.TokenID,
		&auth.WorkspaceID,
		&scopesJSON,
	)
	if err != nil {
		return store.BotTokenAuth{}, err
	}
	if owner.Valid {
		auth.User.OwnerUserID = owner.String
	}
	if err := json.Unmarshal([]byte(scopesJSON), &auth.Scopes); err != nil {
		return store.BotTokenAuth{}, err
	}
	if auth.User.OwnerUserID != "" {
		if err := s.requireMembership(ctx, auth.WorkspaceID, auth.User.OwnerUserID); err != nil {
			return store.BotTokenAuth{}, errors.New("bot owner is not a workspace member")
		}
	}
	if err := s.requireMembership(ctx, auth.WorkspaceID, auth.User.ID); err != nil {
		return store.BotTokenAuth{}, err
	}
	_, _ = s.db.ExecContext(ctx, `UPDATE bot_tokens SET last_used_at = ? WHERE id = ?`, now(), auth.TokenID)
	return auth, nil
}

func normalizeBotScopes(values []string) ([]string, error) {
	seen := map[string]bool{}
	var scopes []string
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			scope := strings.TrimSpace(part)
			if scope == "" {
				continue
			}
			if bundle, ok := botScopeBundles[scope]; ok {
				for _, bundled := range bundle {
					if !seen[bundled] {
						seen[bundled] = true
						scopes = append(scopes, bundled)
					}
				}
				continue
			}
			if !slices.Contains(botAllowedScopes, scope) {
				return nil, errors.New("unknown bot scope: " + scope)
			}
			if !seen[scope] {
				seen[scope] = true
				scopes = append(scopes, scope)
			}
		}
	}
	if len(scopes) == 0 {
		return normalizeBotScopes([]string{"bot:write"})
	}
	slices.Sort(scopes)
	return scopes, nil
}

func hashBotToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
