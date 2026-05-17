package postgres

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strings"

	"github.com/openclaw/clickclack/apps/api/internal/store"
	"github.com/openclaw/clickclack/apps/api/internal/store/postgres/storedb"
)

var pushoverUserKeyRE = regexp.MustCompile(`^[A-Za-z0-9]{30}$`)

func (s *Store) UpdateNotificationSettings(ctx context.Context, input store.UpdateNotificationSettingsInput) (store.NotificationSettings, error) {
	settings, enabled, err := normalizeNotificationSettings(input)
	if err != nil {
		return store.NotificationSettings{}, err
	}
	if err := s.q.UpsertNotificationSettings(ctx, storedb.UpsertNotificationSettingsParams{
		UserID:          input.UserID,
		PushoverEnabled: enabled,
		PushoverUserKey: settings.PushoverUserKey,
	}); err != nil {
		return store.NotificationSettings{}, err
	}
	return settings, nil
}

func normalizeNotificationSettings(input store.UpdateNotificationSettingsInput) (store.NotificationSettings, int64, error) {
	userKey := strings.TrimSpace(input.PushoverUserKey)
	if input.PushoverEnabled && userKey == "" {
		return store.NotificationSettings{}, 0, errors.New("pushover_user_key is required when pushover notifications are enabled")
	}
	if userKey != "" && !pushoverUserKeyRE.MatchString(userKey) {
		return store.NotificationSettings{}, 0, errors.New("pushover_user_key must be 30 alphanumeric characters")
	}
	var enabled int64
	if input.PushoverEnabled {
		enabled = 1
	}
	return store.NotificationSettings{PushoverEnabled: input.PushoverEnabled, PushoverUserKey: userKey}, enabled, nil
}

func (s *Store) hydrateUserNotificationSettings(ctx context.Context, user store.User) (store.User, error) {
	settings, err := s.getNotificationSettings(ctx, user.ID)
	if err != nil {
		return store.User{}, err
	}
	user.NotificationSettings = &settings
	return user, nil
}

func (s *Store) getNotificationSettings(ctx context.Context, userID string) (store.NotificationSettings, error) {
	row, err := s.q.GetNotificationSettings(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return store.NotificationSettings{}, nil
	}
	if err != nil {
		return store.NotificationSettings{}, err
	}
	return storeNotificationSettingsFromDB(row), nil
}

func (s *Store) ListPushNotificationRecipients(ctx context.Context, messageID string) ([]store.PushNotificationRecipient, error) {
	message, err := getMessage(ctx, s.db, messageID)
	if err != nil {
		return nil, err
	}
	if message.DirectConversationID != "" {
		return s.listDirectPushNotificationRecipients(ctx, message)
	}
	return s.listWorkspacePushNotificationRecipients(ctx, message)
}

func (s *Store) listWorkspacePushNotificationRecipients(ctx context.Context, message store.Message) ([]store.PushNotificationRecipient, error) {
	rows, err := s.q.ListWorkspacePushNotificationRecipients(ctx, storedb.ListWorkspacePushNotificationRecipientsParams{
		WorkspaceID: message.WorkspaceID,
		AuthorID:    message.AuthorID,
	})
	if err != nil {
		return nil, err
	}
	out := make([]store.PushNotificationRecipient, 0, len(rows))
	for _, row := range rows {
		out = append(out, storePushRecipient(row.UserID, row.DisplayName, row.PushoverUserKey))
	}
	return out, nil
}

func (s *Store) listDirectPushNotificationRecipients(ctx context.Context, message store.Message) ([]store.PushNotificationRecipient, error) {
	rows, err := s.q.ListDirectPushNotificationRecipients(ctx, storedb.ListDirectPushNotificationRecipientsParams{
		ConversationID: message.DirectConversationID,
		AuthorID:       message.AuthorID,
	})
	if err != nil {
		return nil, err
	}
	out := make([]store.PushNotificationRecipient, 0, len(rows))
	for _, row := range rows {
		out = append(out, storePushRecipient(row.UserID, row.DisplayName, row.PushoverUserKey))
	}
	return out, nil
}
