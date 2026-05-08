package sqlite

import (
	"context"
	"database/sql"
	"strings"

	"github.com/openclaw/clickclack/apps/api/internal/store"
)

func (s *Store) SearchMessages(ctx context.Context, workspaceID, userID, query string, limit int) ([]store.SearchResult, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if err := s.requireMembership(ctx, workspaceID, userID); err != nil {
		return nil, err
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return []store.SearchResult{}, nil
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT m.id, m.workspace_id, COALESCE(m.channel_id, ''), COALESCE(m.direct_conversation_id, ''), m.author_id, m.parent_message_id, m.thread_root_id, m.channel_seq, m.thread_seq,
		       m.body, m.body_format, m.created_at, m.edited_at, m.deleted_at,
		       u.id, u.display_name, u.avatar_url, u.created_at,
		       bm25(messages_fts) AS rank
		FROM messages_fts
		JOIN messages m ON m.id = messages_fts.message_id
		JOIN users u ON u.id = m.author_id
		WHERE messages_fts.workspace_id = ? AND messages_fts MATCH ?
		ORDER BY rank
		LIMIT ?`, workspaceID, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []store.SearchResult{}
	for rows.Next() {
		msg, rank, err := scanSearchMessage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, store.SearchResult{Message: msg, Rank: rank})
	}
	return out, rows.Err()
}

func scanSearchMessage(row scanner) (store.Message, float64, error) {
	var msg store.Message
	var parent, edited, deleted sql.NullString
	var channelSeq, threadSeq sql.NullInt64
	var author store.User
	var rank float64
	err := row.Scan(&msg.ID, &msg.WorkspaceID, &msg.ChannelID, &msg.DirectConversationID, &msg.AuthorID, &parent, &msg.ThreadRootID, &channelSeq, &threadSeq, &msg.Body, &msg.BodyFormat, &msg.CreatedAt, &edited, &deleted, &author.ID, &author.DisplayName, &author.AvatarURL, &author.CreatedAt, &rank)
	if err != nil {
		return store.Message{}, 0, err
	}
	if parent.Valid {
		msg.ParentMessageID = &parent.String
	}
	if channelSeq.Valid {
		msg.ChannelSeq = &channelSeq.Int64
	}
	if threadSeq.Valid {
		msg.ThreadSeq = &threadSeq.Int64
	}
	if edited.Valid {
		msg.EditedAt = &edited.String
	}
	if deleted.Valid {
		msg.DeletedAt = &deleted.String
	}
	msg.Author = &author
	return msg, rank, nil
}
