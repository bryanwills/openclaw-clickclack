package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/openclaw/clickclack/apps/api/internal/realtime"
	"github.com/openclaw/clickclack/apps/api/internal/store"
	sqlitestore "github.com/openclaw/clickclack/apps/api/internal/store/sqlite"
)

func TestMutationAndEphemeralEndpoints(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dataDir := t.TempDir()
	st, err := sqlitestore.Open("sqlite://" + filepath.Join(dataDir, "clickclack.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	owner, err := st.EnsureBootstrap(ctx, "Owner", "owner@example.com")
	if err != nil {
		t.Fatal(err)
	}
	workspaces, err := st.ListWorkspaces(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	channels, err := st.ListChannels(ctx, workspaces[0].ID, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(New(st, realtime.NewHub(), Options{UploadDir: filepath.Join(dataDir, "uploads")}).Handler())
	t.Cleanup(server.Close)

	updatedChannel := patchJSON[struct {
		Channel store.Channel `json:"channel"`
		Event   store.Event   `json:"event"`
	}](t, server.URL+"/api/channels/"+channels[0].ID, map[string]any{"name": "dock"})
	if updatedChannel.Channel.Name != "dock" || updatedChannel.Event.Type != "channel.updated" {
		t.Fatalf("unexpected channel update: %#v", updatedChannel)
	}
	message := postJSON[struct {
		Message store.Message `json:"message"`
	}](t, server.URL+"/api/channels/"+channels[0].ID+"/messages", map[string]string{"body": "original"}).Message
	updatedMessage := patchJSON[struct {
		Message store.Message `json:"message"`
		Event   store.Event   `json:"event"`
	}](t, server.URL+"/api/messages/"+message.ID, map[string]string{"body": "edited"})
	if updatedMessage.Message.Body != "edited" || updatedMessage.Event.Type != "message.updated" {
		t.Fatalf("unexpected message update: %#v", updatedMessage)
	}
	deletedMessage := deleteJSONBody[struct {
		Message store.Message `json:"message"`
		Event   store.Event   `json:"event"`
	}](t, server.URL+"/api/messages/"+message.ID)
	if deletedMessage.Message.DeletedAt == nil || deletedMessage.Event.Type != "message.deleted" {
		t.Fatalf("unexpected message delete: %#v", deletedMessage)
	}
	ephemeral := postJSON[struct {
		Event store.Event `json:"event"`
	}](t, server.URL+"/api/realtime/ephemeral", map[string]any{"workspace_id": workspaces[0].ID, "channel_id": channels[0].ID, "type": "typing.started"})
	if ephemeral.Event.Type != "typing.started" || ephemeral.Event.Cursor != "" {
		t.Fatalf("unexpected ephemeral event: %#v", ephemeral.Event)
	}
	presence := postJSON[struct {
		Event store.Event `json:"event"`
	}](t, server.URL+"/api/realtime/ephemeral", map[string]any{"workspace_id": workspaces[0].ID, "type": "presence.changed", "payload": map[string]any{"status": "afk"}})
	if presence.Event.Type != "presence.changed" {
		t.Fatalf("unexpected presence event: %#v", presence.Event)
	}
	expectStatus(t, http.MethodPatch, server.URL+"/api/channels/"+channels[0].ID, bytes.NewReader([]byte(`{`)), http.StatusBadRequest)
	expectStatus(t, http.MethodPatch, server.URL+"/api/channels/missing", bytes.NewReader([]byte(`{"name":"missing"}`)), http.StatusBadRequest)
	expectStatus(t, http.MethodPatch, server.URL+"/api/messages/"+message.ID, bytes.NewReader([]byte(`{`)), http.StatusBadRequest)
	expectStatus(t, http.MethodPatch, server.URL+"/api/messages/"+message.ID, bytes.NewReader([]byte(`{"body":" "}`)), http.StatusBadRequest)
	expectStatus(t, http.MethodDelete, server.URL+"/api/messages/missing", nil, http.StatusBadRequest)
	expectStatus(t, http.MethodPost, server.URL+"/api/realtime/ephemeral", bytes.NewReader([]byte(`{`)), http.StatusBadRequest)
	expectStatus(t, http.MethodPost, server.URL+"/api/realtime/ephemeral", bytes.NewReader([]byte(`{"workspace_id":"`+workspaces[0].ID+`","type":"bad"}`)), http.StatusBadRequest)
	expectStatus(t, http.MethodPost, server.URL+"/api/realtime/ephemeral", bytes.NewReader([]byte(`{"workspace_id":"missing","type":"typing.started"}`)), http.StatusForbidden)
}

func patchJSON[T any](t *testing.T, endpoint string, body any) T {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPatch, endpoint, bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	return doJSON[T](t, req)
}

func deleteJSONBody[T any](t *testing.T, endpoint string) T {
	t.Helper()
	req, err := http.NewRequest(http.MethodDelete, endpoint, nil)
	if err != nil {
		t.Fatal(err)
	}
	return doJSON[T](t, req)
}

func doJSON[T any](t *testing.T, req *http.Request) T {
	t.Helper()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		t.Fatalf("%s %s: %s", req.Method, req.URL, resp.Status)
	}
	var out T
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out
}
