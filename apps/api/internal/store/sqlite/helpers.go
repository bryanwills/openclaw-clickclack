package sqlite

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"regexp"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/openclaw/clickclack/apps/api/internal/store"
)

type scanner interface {
	Scan(dest ...any) error
}

func scanUser(row scanner) (store.User, error) {
	var u store.User
	err := row.Scan(&u.ID, &u.DisplayName, &u.AvatarURL, &u.CreatedAt)
	return u, err
}

func scanWorkspace(row scanner) (store.Workspace, error) {
	var w store.Workspace
	err := row.Scan(&w.ID, &w.Name, &w.Slug, &w.CreatedAt)
	return w, err
}

func scanChannel(row scanner) (store.Channel, error) {
	var ch store.Channel
	err := row.Scan(&ch.ID, &ch.WorkspaceID, &ch.Name, &ch.Kind, &ch.CreatedAt, &ch.ArchivedAt)
	return ch, err
}

func getMessage(ctx context.Context, db *sql.DB, id string) (store.Message, error) {
	return scanMessage(db.QueryRowContext(ctx, messageSelect()+` WHERE m.id = ?`, id))
}

func getMessageTx(ctx context.Context, tx *sql.Tx, id string) (store.Message, error) {
	return scanMessage(tx.QueryRowContext(ctx, messageSelect()+` WHERE m.id = ?`, id))
}

func messageSelect() string {
	return `SELECT m.id, m.workspace_id, COALESCE(m.channel_id, ''), COALESCE(m.direct_conversation_id, ''), m.author_id, m.parent_message_id, m.thread_root_id, m.channel_seq, m.thread_seq,
		       m.body, m.body_format, m.created_at, m.edited_at, m.deleted_at,
		       u.id, u.display_name, u.avatar_url, u.created_at
		FROM messages m
		JOIN users u ON u.id = m.author_id`
}

func scanMessage(row scanner) (store.Message, error) {
	var m store.Message
	var parent, edited, deleted sql.NullString
	var channelSeq, threadSeq sql.NullInt64
	var author store.User
	err := row.Scan(&m.ID, &m.WorkspaceID, &m.ChannelID, &m.DirectConversationID, &m.AuthorID, &parent, &m.ThreadRootID, &channelSeq, &threadSeq, &m.Body, &m.BodyFormat, &m.CreatedAt, &edited, &deleted, &author.ID, &author.DisplayName, &author.AvatarURL, &author.CreatedAt)
	if err != nil {
		return store.Message{}, err
	}
	if parent.Valid {
		m.ParentMessageID = &parent.String
	}
	if channelSeq.Valid {
		m.ChannelSeq = &channelSeq.Int64
	}
	if threadSeq.Valid {
		m.ThreadSeq = &threadSeq.Int64
	}
	if edited.Valid {
		m.EditedAt = &edited.String
	}
	if deleted.Valid {
		m.DeletedAt = &deleted.String
	}
	m.Author = &author
	return m, nil
}

func scanMessages(rows *sql.Rows) ([]store.Message, error) {
	out := []store.Message{}
	for rows.Next() {
		msg, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, msg)
	}
	return out, rows.Err()
}

func getThreadState(ctx context.Context, db *sql.DB, rootID string) (store.ThreadState, error) {
	return scanThreadState(db.QueryRowContext(ctx, `SELECT root_message_id, reply_count, last_reply_at, last_reply_author_ids_json FROM thread_state WHERE root_message_id = ?`, rootID))
}

func scanThreadState(row scanner) (store.ThreadState, error) {
	var state store.ThreadState
	var lastReply sql.NullString
	if err := row.Scan(&state.RootMessageID, &state.ReplyCount, &lastReply, &state.LastReplyAuthorIDsJSON); err != nil {
		return store.ThreadState{}, err
	}
	if lastReply.Valid {
		state.LastReplyAt = &lastReply.String
	}
	_ = json.Unmarshal([]byte(state.LastReplyAuthorIDsJSON), &state.LastReplyAuthorIDs)
	return state, nil
}

func updateThreadState(ctx context.Context, tx *sql.Tx, rootID, authorID, createdAt string) (store.ThreadState, error) {
	state, err := scanThreadState(tx.QueryRowContext(ctx, `SELECT root_message_id, reply_count, last_reply_at, last_reply_author_ids_json FROM thread_state WHERE root_message_id = ?`, rootID))
	if err != nil {
		return store.ThreadState{}, err
	}
	ids := append([]string{authorID}, state.LastReplyAuthorIDs...)
	seen := map[string]bool{}
	compact := make([]string, 0, 3)
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		compact = append(compact, id)
		if len(compact) == 3 {
			break
		}
	}
	body, _ := json.Marshal(compact)
	if _, err := tx.ExecContext(ctx, `UPDATE thread_state SET reply_count = reply_count + 1, last_reply_at = ?, last_reply_author_ids_json = ? WHERE root_message_id = ?`, createdAt, string(body), rootID); err != nil {
		return store.ThreadState{}, err
	}
	return scanThreadState(tx.QueryRowContext(ctx, `SELECT root_message_id, reply_count, last_reply_at, last_reply_author_ids_json FROM thread_state WHERE root_message_id = ?`, rootID))
}

func insertEvent(ctx context.Context, tx *sql.Tx, workspaceID, channelID, eventType string, seq *int64, payload any) (store.Event, error) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return store.Event{}, err
	}
	event := store.Event{
		ID:          newID("evt"),
		Cursor:      newID("cur"),
		Type:        eventType,
		WorkspaceID: workspaceID,
		ChannelID:   channelID,
		Seq:         seq,
		CreatedAt:   now(),
		PayloadJSON: string(payloadJSON),
		Payload:     payload,
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO events (id, cursor, workspace_id, channel_id, type, seq, payload_json, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, event.ID, event.Cursor, event.WorkspaceID, nullableString(event.ChannelID), event.Type, event.Seq, event.PayloadJSON, event.CreatedAt); err != nil {
		return store.Event{}, err
	}
	return event, nil
}

func scanEvents(rows *sql.Rows) ([]store.Event, error) {
	out := []store.Event{}
	for rows.Next() {
		var event store.Event
		var seq sql.NullInt64
		if err := rows.Scan(&event.ID, &event.Cursor, &event.WorkspaceID, &event.ChannelID, &event.Type, &seq, &event.PayloadJSON, &event.CreatedAt); err != nil {
			return nil, err
		}
		if seq.Valid {
			event.Seq = &seq.Int64
		}
		var payload any
		_ = json.Unmarshal([]byte(event.PayloadJSON), &payload)
		event.Payload = payload
		out = append(out, event)
	}
	return out, rows.Err()
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func newID(prefix string) string {
	return prefix + "_" + strings.ToLower(ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String())
}

func now() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

var slugRE = regexp.MustCompile(`[^a-z0-9]+`)

func slug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = slugRE.ReplaceAllString(value, "-")
	return strings.Trim(value, "-")
}
