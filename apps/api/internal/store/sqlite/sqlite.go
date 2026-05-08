package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/openclaw/clickclack/apps/api/internal/store"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type Store struct {
	db *sql.DB
}

func Open(dbURL string) (*Store, error) {
	path := strings.TrimPrefix(dbURL, "sqlite://")
	if path == "" || path == dbURL {
		path = dbURL
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA busy_timeout=5000",
	} {
		if _, err := db.ExecContext(ctx, pragma); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) Migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (name TEXT PRIMARY KEY, applied_at TEXT NOT NULL)`); err != nil {
		return err
	}
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		name := entry.Name()
		var applied string
		err := s.db.QueryRowContext(ctx, `SELECT name FROM schema_migrations WHERE name = ?`, name).Scan(&applied)
		if err == nil {
			continue
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		body, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, string(body)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("%s: %w", name, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (name, applied_at) VALUES (?, ?)`, name, now()); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) EnsureBootstrap(ctx context.Context, name, email string) (store.User, error) {
	user, err := s.FirstUser(ctx)
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return store.User{}, err
	}
	user, err = s.CreateUser(ctx, store.CreateUserInput{DisplayName: name, Email: email})
	if err != nil {
		return store.User{}, err
	}
	ws, err := s.CreateWorkspace(ctx, store.CreateWorkspaceInput{Name: "ClickClack", Slug: "clickclack"}, user.ID)
	if err != nil {
		return store.User{}, err
	}
	_, _, err = s.CreateChannel(ctx, store.CreateChannelInput{WorkspaceID: ws.ID, Name: "general", Kind: "public", UserID: user.ID})
	return user, err
}

func (s *Store) CreateUser(ctx context.Context, input store.CreateUserInput) (store.User, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return store.User{}, err
	}
	defer tx.Rollback()
	user := store.User{
		ID:          newID("usr"),
		DisplayName: strings.TrimSpace(input.DisplayName),
		AvatarURL:   "",
		CreatedAt:   now(),
	}
	if user.DisplayName == "" {
		user.DisplayName = "Local User"
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO users (id, display_name, avatar_url, created_at) VALUES (?, ?, ?, ?)`, user.ID, user.DisplayName, user.AvatarURL, user.CreatedAt); err != nil {
		return store.User{}, err
	}
	if input.Email != "" {
		_, err = tx.ExecContext(ctx, `INSERT INTO identities (id, user_id, provider, provider_subject, email, created_at) VALUES (?, ?, 'local', ?, ?, ?)`, newID("idn"), user.ID, input.Email, input.Email, user.CreatedAt)
		if err != nil {
			return store.User{}, err
		}
	}
	return user, tx.Commit()
}

func (s *Store) FirstUser(ctx context.Context) (store.User, error) {
	return scanUser(s.db.QueryRowContext(ctx, `SELECT id, display_name, avatar_url, created_at FROM users ORDER BY created_at LIMIT 1`))
}

func (s *Store) GetUser(ctx context.Context, id string) (store.User, error) {
	return scanUser(s.db.QueryRowContext(ctx, `SELECT id, display_name, avatar_url, created_at FROM users WHERE id = ?`, id))
}

func (s *Store) ListWorkspaces(ctx context.Context, userID string) ([]store.Workspace, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT w.id, w.name, w.slug, w.created_at
		FROM workspaces w
		JOIN workspace_members wm ON wm.workspace_id = w.id
		WHERE wm.user_id = ?
		ORDER BY w.created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []store.Workspace{}
	for rows.Next() {
		var w store.Workspace
		if err := rows.Scan(&w.ID, &w.Name, &w.Slug, &w.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (s *Store) CreateWorkspace(ctx context.Context, input store.CreateWorkspaceInput, ownerID string) (store.Workspace, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return store.Workspace{}, err
	}
	defer tx.Rollback()
	w := store.Workspace{ID: newID("wsp"), Name: strings.TrimSpace(input.Name), Slug: slug(input.Slug), CreatedAt: now()}
	if w.Name == "" {
		w.Name = "Untitled"
	}
	if w.Slug == "" {
		w.Slug = slug(w.Name)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO workspaces (id, name, slug, created_at) VALUES (?, ?, ?, ?)`, w.ID, w.Name, w.Slug, w.CreatedAt); err != nil {
		return store.Workspace{}, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO workspace_members (workspace_id, user_id, role, created_at) VALUES (?, ?, 'owner', ?)`, w.ID, ownerID, w.CreatedAt); err != nil {
		return store.Workspace{}, err
	}
	return w, tx.Commit()
}

func (s *Store) GetWorkspace(ctx context.Context, workspaceID, userID string) (store.Workspace, error) {
	return scanWorkspace(s.db.QueryRowContext(ctx, `
		SELECT w.id, w.name, w.slug, w.created_at
		FROM workspaces w
		JOIN workspace_members wm ON wm.workspace_id = w.id
		WHERE w.id = ? AND wm.user_id = ?`, workspaceID, userID))
}

func (s *Store) ListChannels(ctx context.Context, workspaceID, userID string) ([]store.Channel, error) {
	if err := s.requireMembership(ctx, workspaceID, userID); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, workspace_id, name, kind, created_at, archived_at FROM channels WHERE workspace_id = ? ORDER BY name`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []store.Channel{}
	for rows.Next() {
		ch, err := scanChannel(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, ch)
	}
	return out, rows.Err()
}

func (s *Store) CreateChannel(ctx context.Context, input store.CreateChannelInput) (store.Channel, store.Event, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return store.Channel{}, store.Event{}, err
	}
	defer tx.Rollback()
	if err := requireMembershipTx(ctx, tx, input.WorkspaceID, input.UserID); err != nil {
		return store.Channel{}, store.Event{}, err
	}
	ch := store.Channel{ID: newID("chn"), WorkspaceID: input.WorkspaceID, Name: slug(input.Name), Kind: input.Kind, CreatedAt: now()}
	if ch.Name == "" {
		ch.Name = "general"
	}
	if ch.Kind == "" {
		ch.Kind = "public"
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO channels (id, workspace_id, name, kind, created_at) VALUES (?, ?, ?, ?, ?)`, ch.ID, ch.WorkspaceID, ch.Name, ch.Kind, ch.CreatedAt); err != nil {
		return store.Channel{}, store.Event{}, err
	}
	event, err := insertEvent(ctx, tx, ch.WorkspaceID, ch.ID, "channel.created", nil, map[string]string{"channel_id": ch.ID})
	if err != nil {
		return store.Channel{}, store.Event{}, err
	}
	return ch, event, tx.Commit()
}

func (s *Store) ListMessages(ctx context.Context, channelID, userID string, afterSeq int64, limit int) ([]store.Message, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	var workspaceID string
	if err := s.db.QueryRowContext(ctx, `SELECT workspace_id FROM channels WHERE id = ?`, channelID).Scan(&workspaceID); err != nil {
		return nil, err
	}
	if err := s.requireMembership(ctx, workspaceID, userID); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT m.id, m.workspace_id, COALESCE(m.channel_id, ''), COALESCE(m.direct_conversation_id, ''), m.author_id, m.parent_message_id, m.thread_root_id, m.channel_seq, m.thread_seq,
		       m.body, m.body_format, m.created_at, m.edited_at, m.deleted_at,
		       u.id, u.display_name, u.avatar_url, u.created_at
		FROM messages m
		JOIN users u ON u.id = m.author_id
		WHERE m.channel_id = ? AND m.parent_message_id IS NULL AND m.channel_seq > ?
		ORDER BY m.channel_seq
		LIMIT ?`, channelID, afterSeq, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	messages, err := scanMessages(rows)
	if err != nil {
		return nil, err
	}
	return s.hydrateAttachments(ctx, messages)
}

func (s *Store) CreateMessage(ctx context.Context, input store.CreateMessageInput) (store.Message, store.Event, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return store.Message{}, store.Event{}, err
	}
	defer tx.Rollback()
	var workspaceID string
	if err := tx.QueryRowContext(ctx, `SELECT workspace_id FROM channels WHERE id = ?`, input.ChannelID).Scan(&workspaceID); err != nil {
		return store.Message{}, store.Event{}, err
	}
	if err := requireMembershipTx(ctx, tx, workspaceID, input.AuthorID); err != nil {
		return store.Message{}, store.Event{}, err
	}
	var seq int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(channel_seq), 0) + 1 FROM messages WHERE channel_id = ? AND parent_message_id IS NULL`, input.ChannelID).Scan(&seq); err != nil {
		return store.Message{}, store.Event{}, err
	}
	id := newID("msg")
	createdAt := now()
	body := strings.TrimSpace(input.Body)
	if body == "" {
		return store.Message{}, store.Event{}, errors.New("message body is required")
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO messages (id, workspace_id, channel_id, direct_conversation_id, author_id, parent_message_id, thread_root_id, channel_seq, thread_seq, body, body_format, created_at)
		VALUES (?, ?, ?, NULL, ?, NULL, ?, ?, NULL, ?, 'markdown', ?)`, id, workspaceID, input.ChannelID, input.AuthorID, id, seq, body, createdAt); err != nil {
		return store.Message{}, store.Event{}, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO thread_state (root_message_id) VALUES (?)`, id); err != nil {
		return store.Message{}, store.Event{}, err
	}
	event, err := insertEvent(ctx, tx, workspaceID, input.ChannelID, "message.created", &seq, map[string]string{"message_id": id})
	if err != nil {
		return store.Message{}, store.Event{}, err
	}
	msg, err := getMessageTx(ctx, tx, id)
	if err != nil {
		return store.Message{}, store.Event{}, err
	}
	return msg, event, tx.Commit()
}

func (s *Store) GetThread(ctx context.Context, rootMessageID, userID string, limit int) (store.Message, []store.Message, store.ThreadState, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	root, err := getMessage(ctx, s.db, rootMessageID)
	if err != nil {
		return store.Message{}, nil, store.ThreadState{}, err
	}
	if root.ParentMessageID != nil {
		return store.Message{}, nil, store.ThreadState{}, errors.New("thread root must be a root message")
	}
	if err := s.requireMembership(ctx, root.WorkspaceID, userID); err != nil {
		return store.Message{}, nil, store.ThreadState{}, err
	}
	roots, err := s.hydrateAttachments(ctx, []store.Message{root})
	if err != nil {
		return store.Message{}, nil, store.ThreadState{}, err
	}
	root = roots[0]
	rows, err := s.db.QueryContext(ctx, `
		SELECT m.id, m.workspace_id, COALESCE(m.channel_id, ''), COALESCE(m.direct_conversation_id, ''), m.author_id, m.parent_message_id, m.thread_root_id, m.channel_seq, m.thread_seq,
		       m.body, m.body_format, m.created_at, m.edited_at, m.deleted_at,
		       u.id, u.display_name, u.avatar_url, u.created_at
		FROM messages m
		JOIN users u ON u.id = m.author_id
		WHERE m.thread_root_id = ? AND m.parent_message_id = ?
		ORDER BY m.thread_seq
		LIMIT ?`, rootMessageID, rootMessageID, limit)
	if err != nil {
		return store.Message{}, nil, store.ThreadState{}, err
	}
	defer rows.Close()
	replies, err := scanMessages(rows)
	if err != nil {
		return store.Message{}, nil, store.ThreadState{}, err
	}
	replies, err = s.hydrateAttachments(ctx, replies)
	if err != nil {
		return store.Message{}, nil, store.ThreadState{}, err
	}
	state, err := getThreadState(ctx, s.db, rootMessageID)
	return root, replies, state, err
}

func (s *Store) CreateThreadReply(ctx context.Context, input store.CreateThreadReplyInput) (store.Message, store.ThreadState, []store.Event, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return store.Message{}, store.ThreadState{}, nil, err
	}
	defer tx.Rollback()
	root, err := getMessageTx(ctx, tx, input.RootMessageID)
	if err != nil {
		return store.Message{}, store.ThreadState{}, nil, err
	}
	if root.ParentMessageID != nil {
		return store.Message{}, store.ThreadState{}, nil, errors.New("nested thread replies are not supported")
	}
	if err := requireMembershipTx(ctx, tx, root.WorkspaceID, input.AuthorID); err != nil {
		return store.Message{}, store.ThreadState{}, nil, err
	}
	var seq int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(thread_seq), 0) + 1 FROM messages WHERE thread_root_id = ? AND parent_message_id = ?`, root.ID, root.ID).Scan(&seq); err != nil {
		return store.Message{}, store.ThreadState{}, nil, err
	}
	id := newID("msg")
	createdAt := now()
	body := strings.TrimSpace(input.Body)
	if body == "" {
		return store.Message{}, store.ThreadState{}, nil, errors.New("reply body is required")
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO messages (id, workspace_id, channel_id, direct_conversation_id, author_id, parent_message_id, thread_root_id, channel_seq, thread_seq, body, body_format, created_at)
		VALUES (?, ?, ?, NULL, ?, ?, ?, NULL, ?, ?, 'markdown', ?)`, id, root.WorkspaceID, root.ChannelID, input.AuthorID, root.ID, root.ID, seq, body, createdAt); err != nil {
		return store.Message{}, store.ThreadState{}, nil, err
	}
	state, err := updateThreadState(ctx, tx, root.ID, input.AuthorID, createdAt)
	if err != nil {
		return store.Message{}, store.ThreadState{}, nil, err
	}
	replyEvent, err := insertEvent(ctx, tx, root.WorkspaceID, root.ChannelID, "thread.reply_created", nil, map[string]string{"message_id": id, "root_message_id": root.ID})
	if err != nil {
		return store.Message{}, store.ThreadState{}, nil, err
	}
	stateEvent, err := insertEvent(ctx, tx, root.WorkspaceID, root.ChannelID, "thread.state_updated", nil, map[string]string{"root_message_id": root.ID})
	if err != nil {
		return store.Message{}, store.ThreadState{}, nil, err
	}
	msg, err := getMessageTx(ctx, tx, id)
	if err != nil {
		return store.Message{}, store.ThreadState{}, nil, err
	}
	return msg, state, []store.Event{replyEvent, stateEvent}, tx.Commit()
}

func (s *Store) AddReaction(ctx context.Context, input store.CreateReactionInput) (store.Event, error) {
	return s.reaction(ctx, input, true)
}

func (s *Store) RemoveReaction(ctx context.Context, input store.CreateReactionInput) (store.Event, error) {
	return s.reaction(ctx, input, false)
}

func (s *Store) ListEventsAfter(ctx context.Context, workspaceID, userID, cursor string, limit int) ([]store.Event, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	if err := s.requireMembership(ctx, workspaceID, userID); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, cursor, workspace_id, COALESCE(channel_id, ''), type, seq, payload_json, created_at
		FROM events
		WHERE workspace_id = ? AND cursor > ?
		ORDER BY cursor
		LIMIT ?`, workspaceID, cursor, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEvents(rows)
}

func (s *Store) reaction(ctx context.Context, input store.CreateReactionInput, add bool) (store.Event, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return store.Event{}, err
	}
	defer tx.Rollback()
	msg, err := getMessageTx(ctx, tx, input.MessageID)
	if err != nil {
		return store.Event{}, err
	}
	if err := requireMembershipTx(ctx, tx, msg.WorkspaceID, input.UserID); err != nil {
		return store.Event{}, err
	}
	if add {
		_, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO reactions (message_id, user_id, emoji, created_at) VALUES (?, ?, ?, ?)`, input.MessageID, input.UserID, input.Emoji, now())
	} else {
		_, err = tx.ExecContext(ctx, `DELETE FROM reactions WHERE message_id = ? AND user_id = ? AND emoji = ?`, input.MessageID, input.UserID, input.Emoji)
	}
	if err != nil {
		return store.Event{}, err
	}
	eventType := "reaction.added"
	if !add {
		eventType = "reaction.removed"
	}
	event, err := insertEvent(ctx, tx, msg.WorkspaceID, msg.ChannelID, eventType, msg.ChannelSeq, map[string]string{"message_id": input.MessageID, "emoji": input.Emoji})
	if err != nil {
		return store.Event{}, err
	}
	return event, tx.Commit()
}

func (s *Store) requireMembership(ctx context.Context, workspaceID, userID string) error {
	var one int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM workspace_members WHERE workspace_id = ? AND user_id = ?`, workspaceID, userID).Scan(&one)
	return err
}

func requireMembershipTx(ctx context.Context, tx *sql.Tx, workspaceID, userID string) error {
	var one int
	err := tx.QueryRowContext(ctx, `SELECT 1 FROM workspace_members WHERE workspace_id = ? AND user_id = ?`, workspaceID, userID).Scan(&one)
	return err
}
