package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/openclaw/clickclack/apps/api/internal/store"
)

func (s *Store) CreateMagicLink(ctx context.Context, email, displayName string) (store.MagicLink, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return store.MagicLink{}, errors.New("email is required")
	}
	link := store.MagicLink{
		ID:          newID("mln"),
		Token:       newID("mgt"),
		Email:       email,
		DisplayName: strings.TrimSpace(displayName),
		CreatedAt:   now(),
		ExpiresAt:   time.Now().UTC().Add(15 * time.Minute).Format(time.RFC3339Nano),
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO auth_magic_links (id, token, email, display_name, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?)`, link.ID, link.Token, link.Email, link.DisplayName, link.CreatedAt, link.ExpiresAt)
	return link, err
}

func (s *Store) ConsumeMagicLink(ctx context.Context, token string) (store.User, store.Session, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return store.User{}, store.Session{}, err
	}
	defer tx.Rollback()
	link, err := scanMagicLink(tx.QueryRowContext(ctx, `
		SELECT id, token, email, display_name, created_at, expires_at, used_at
		FROM auth_magic_links WHERE token = ?`, strings.TrimSpace(token)))
	if err != nil {
		return store.User{}, store.Session{}, err
	}
	if link.UsedAt != nil {
		return store.User{}, store.Session{}, errors.New("magic link already used")
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, link.ExpiresAt)
	if err != nil || time.Now().UTC().After(expiresAt) {
		return store.User{}, store.Session{}, errors.New("magic link expired")
	}
	user, err := getOrCreateMagicUser(ctx, tx, link.Email, link.DisplayName)
	if err != nil {
		return store.User{}, store.Session{}, err
	}
	usedAt := now()
	if _, err := tx.ExecContext(ctx, `UPDATE auth_magic_links SET used_at = ? WHERE id = ?`, usedAt, link.ID); err != nil {
		return store.User{}, store.Session{}, err
	}
	session, err := createSessionTx(ctx, tx, user.ID)
	if err != nil {
		return store.User{}, store.Session{}, err
	}
	return user, session, tx.Commit()
}

func (s *Store) GetSessionUser(ctx context.Context, token string) (store.User, error) {
	return scanUser(s.db.QueryRowContext(ctx, `
		SELECT u.id, u.display_name, u.handle, u.avatar_url, u.created_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token = ? AND s.revoked_at IS NULL AND s.expires_at > ?`, token, now()))
}

func (s *Store) CreateSession(ctx context.Context, userID string) (store.Session, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return store.Session{}, err
	}
	defer tx.Rollback()
	session, err := createSessionTx(ctx, tx, userID)
	if err != nil {
		return store.Session{}, err
	}
	return session, tx.Commit()
}

func getOrCreateMagicUser(ctx context.Context, tx *sql.Tx, email, displayName string) (store.User, error) {
	user, err := scanUser(tx.QueryRowContext(ctx, `
		SELECT u.id, u.display_name, u.handle, u.avatar_url, u.created_at
		FROM identities i
		JOIN users u ON u.id = i.user_id
		WHERE i.email = ?
		ORDER BY u.created_at LIMIT 1`, email))
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return store.User{}, err
	}
	user = store.User{ID: newID("usr"), DisplayName: strings.TrimSpace(displayName), Handle: "", AvatarURL: "", CreatedAt: now()}
	if user.DisplayName == "" {
		user.DisplayName = email
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO users (id, display_name, avatar_url, created_at) VALUES (?, ?, '', ?)`, user.ID, user.DisplayName, user.CreatedAt); err != nil {
		return store.User{}, err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO identities (id, user_id, provider, provider_subject, email, created_at)
		VALUES (?, ?, 'magic', ?, ?, ?)`, newID("idn"), user.ID, email, email, user.CreatedAt)
	return user, err
}

func createSessionTx(ctx context.Context, tx *sql.Tx, userID string) (store.Session, error) {
	session := store.Session{
		ID:        newID("ses"),
		Token:     newID("sst"),
		UserID:    userID,
		CreatedAt: now(),
		ExpiresAt: time.Now().UTC().Add(30 * 24 * time.Hour).Format(time.RFC3339Nano),
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO sessions (id, token, user_id, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?)`, session.ID, session.Token, session.UserID, session.CreatedAt, session.ExpiresAt)
	return session, err
}

func scanMagicLink(row scanner) (store.MagicLink, error) {
	var link store.MagicLink
	var usedAt sql.NullString
	err := row.Scan(&link.ID, &link.Token, &link.Email, &link.DisplayName, &link.CreatedAt, &link.ExpiresAt, &usedAt)
	if usedAt.Valid {
		link.UsedAt = &usedAt.String
	}
	return link, err
}
