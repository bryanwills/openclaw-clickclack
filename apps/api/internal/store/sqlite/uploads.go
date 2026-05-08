package sqlite

import (
	"context"
	"errors"

	"github.com/openclaw/clickclack/apps/api/internal/store"
)

func (s *Store) CreateUpload(ctx context.Context, input store.CreateUploadInput) (store.Upload, error) {
	if err := s.requireMembership(ctx, input.WorkspaceID, input.OwnerID); err != nil {
		return store.Upload{}, err
	}
	upload := store.Upload{
		ID:          newID("upl"),
		WorkspaceID: input.WorkspaceID,
		OwnerID:     input.OwnerID,
		Filename:    input.Filename,
		ContentType: input.ContentType,
		ByteSize:    input.ByteSize,
		StoragePath: input.StoragePath,
		CreatedAt:   now(),
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO uploads (id, workspace_id, owner_id, filename, content_type, byte_size, storage_path, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		upload.ID, upload.WorkspaceID, upload.OwnerID, upload.Filename, upload.ContentType, upload.ByteSize, upload.StoragePath, upload.CreatedAt)
	return upload, err
}

func (s *Store) GetUpload(ctx context.Context, uploadID, userID string) (store.Upload, error) {
	upload, err := scanUpload(s.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, owner_id, filename, content_type, byte_size, storage_path, created_at
		FROM uploads
		WHERE id = ?`, uploadID))
	if err != nil {
		return store.Upload{}, err
	}
	if err := s.requireMembership(ctx, upload.WorkspaceID, userID); err != nil {
		return store.Upload{}, err
	}
	return upload, nil
}

func (s *Store) AttachUpload(ctx context.Context, input store.AttachUploadInput) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	msg, err := getMessageTx(ctx, tx, input.MessageID)
	if err != nil {
		return err
	}
	if err := requireMembershipTx(ctx, tx, msg.WorkspaceID, input.UserID); err != nil {
		return err
	}
	var uploadWorkspace string
	if err := tx.QueryRowContext(ctx, `SELECT workspace_id FROM uploads WHERE id = ?`, input.UploadID).Scan(&uploadWorkspace); err != nil {
		return err
	}
	if uploadWorkspace != msg.WorkspaceID {
		return errors.New("upload and message workspaces differ")
	}
	_, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO message_attachments (message_id, upload_id, created_at) VALUES (?, ?, ?)`, input.MessageID, input.UploadID, now())
	if err != nil {
		return err
	}
	return tx.Commit()
}

func scanUpload(row scanner) (store.Upload, error) {
	var upload store.Upload
	err := row.Scan(&upload.ID, &upload.WorkspaceID, &upload.OwnerID, &upload.Filename, &upload.ContentType, &upload.ByteSize, &upload.StoragePath, &upload.CreatedAt)
	return upload, err
}

func (s *Store) hydrateAttachments(ctx context.Context, messages []store.Message) ([]store.Message, error) {
	for i := range messages {
		rows, err := s.db.QueryContext(ctx, `
			SELECT u.id, u.workspace_id, u.owner_id, u.filename, u.content_type, u.byte_size, u.storage_path, u.created_at
			FROM uploads u
			JOIN message_attachments ma ON ma.upload_id = u.id
			WHERE ma.message_id = ?
			ORDER BY ma.created_at`, messages[i].ID)
		if err != nil {
			return nil, err
		}
		uploads := []store.Upload{}
		for rows.Next() {
			upload, err := scanUpload(rows)
			if err != nil {
				_ = rows.Close()
				return nil, err
			}
			uploads = append(uploads, upload)
		}
		if err := rows.Close(); err != nil {
			return nil, err
		}
		if len(uploads) > 0 {
			messages[i].Attachments = uploads
		}
	}
	return messages, nil
}
