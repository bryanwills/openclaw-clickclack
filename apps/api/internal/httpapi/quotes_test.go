package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openclaw/clickclack/apps/api/internal/realtime"
	"github.com/openclaw/clickclack/apps/api/internal/store"
	sqlitestore "github.com/openclaw/clickclack/apps/api/internal/store/sqlite"
)

// postJSONStatus posts a body and returns the response status and body bytes.
// Use it when the test expects a non-2xx response.
func postJSONStatus(t *testing.T, endpoint string, body any) (int, []byte) {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Post(endpoint, "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, out
}

func TestQuoteHTTPRoundTrip(t *testing.T) {
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
	otherChannel, _, err := st.CreateChannel(ctx, store.CreateChannelInput{WorkspaceID: workspaces[0].ID, Name: "second", Kind: "public", UserID: owner.ID})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(New(st, realtime.NewHub(), Options{UploadDir: filepath.Join(dataDir, "uploads")}).Handler())
	t.Cleanup(server.Close)

	root := postJSON[struct {
		Message store.Message `json:"message"`
	}](t, server.URL+"/api/channels/"+channels[0].ID+"/messages", map[string]string{"body": "first"}).Message
	other := postJSON[struct {
		Message store.Message `json:"message"`
	}](t, server.URL+"/api/channels/"+otherChannel.ID+"/messages", map[string]string{"body": "in another channel"}).Message

	reply := postJSON[struct {
		Message store.Message `json:"message"`
	}](t, server.URL+"/api/channels/"+channels[0].ID+"/messages", map[string]any{
		"body":              "responding",
		"quoted_message_id": root.ID,
	}).Message

	if reply.QuotedMessageID == nil || *reply.QuotedMessageID != root.ID {
		t.Fatalf("expected quoted_message_id %q, got %#v", root.ID, reply.QuotedMessageID)
	}
	if reply.QuotedBodySnapshot != "first" {
		t.Fatalf("expected snapshot %q, got %q", "first", reply.QuotedBodySnapshot)
	}
	if reply.QuotedAuthor == nil || reply.QuotedAuthor.ID != owner.ID {
		t.Fatalf("expected hydrated quoted_author, got %#v", reply.QuotedAuthor)
	}

	// Cross-channel quote → 400 with scope error message.
	status, body := postJSONStatus(t, server.URL+"/api/channels/"+channels[0].ID+"/messages", map[string]any{
		"body":              "leak",
		"quoted_message_id": other.ID,
	})
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400 cross-channel, got %d (%s)", status, string(body))
	}
	if !strings.Contains(string(body), "channel") && !strings.Contains(string(body), "conversation") {
		t.Fatalf("expected scope error in body, got %s", string(body))
	}

	// Empty quoted_message_id is treated as absent (no error).
	noQuote := postJSON[struct {
		Message store.Message `json:"message"`
	}](t, server.URL+"/api/channels/"+channels[0].ID+"/messages", map[string]any{
		"body":              "plain",
		"quoted_message_id": "",
	}).Message
	if noQuote.QuotedMessageID != nil {
		t.Fatalf("expected no quote, got %v", *noQuote.QuotedMessageID)
	}

	// Round-trip via list endpoint.
	list := getJSON[struct {
		Messages []store.Message `json:"messages"`
	}](t, server.URL+"/api/channels/"+channels[0].ID+"/messages")
	var seen bool
	for _, m := range list.Messages {
		if m.ID == reply.ID {
			seen = true
			if m.QuotedAuthor == nil || m.QuotedAuthor.DisplayName == "" {
				t.Fatalf("expected hydrated quoted author in list, got %#v", m.QuotedAuthor)
			}
		}
	}
	if !seen {
		t.Fatalf("reply not found in list response")
	}
}

func TestQuoteHTTPThreadAndDM(t *testing.T) {
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
	other, err := st.CreateUser(ctx, store.CreateUserInput{DisplayName: "Other", Email: "other@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.AddWorkspaceMember(ctx, workspaces[0].ID, other.ID, "member"); err != nil {
		t.Fatal(err)
	}
	dm, err := st.CreateDirectConversation(ctx, store.CreateDirectConversationInput{WorkspaceID: workspaces[0].ID, UserID: owner.ID, MemberIDs: []string{other.ID}})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(New(st, realtime.NewHub(), Options{UploadDir: filepath.Join(dataDir, "uploads")}).Handler())
	t.Cleanup(server.Close)

	root := postJSON[struct {
		Message store.Message `json:"message"`
	}](t, server.URL+"/api/channels/"+channels[0].ID+"/messages", map[string]string{"body": "root"}).Message

	reply := postJSON[struct {
		Message store.Message `json:"message"`
	}](t, server.URL+"/api/messages/"+root.ID+"/thread/replies", map[string]any{
		"body":              "in thread",
		"quoted_message_id": root.ID,
	}).Message
	if reply.QuotedMessageID == nil || *reply.QuotedMessageID != root.ID {
		t.Fatalf("expected thread quote, got %#v", reply.QuotedMessageID)
	}

	// thread reply quoting a non-thread channel message → 400
	status, _ := postJSONStatus(t, server.URL+"/api/messages/"+root.ID+"/thread/replies", map[string]any{
		"body":              "broken",
		"quoted_message_id": "msg_does_not_exist",
	})
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown quote, got %d", status)
	}

	dmRoot := postJSON[struct {
		Message store.Message `json:"message"`
	}](t, server.URL+"/api/dms/"+dm.ID+"/messages", map[string]string{"body": "dm hi"}).Message
	dmReply := postJSON[struct {
		Message store.Message `json:"message"`
	}](t, server.URL+"/api/dms/"+dm.ID+"/messages", map[string]any{
		"body":              "responding",
		"quoted_message_id": dmRoot.ID,
	}).Message
	if dmReply.QuotedMessageID == nil || *dmReply.QuotedMessageID != dmRoot.ID {
		t.Fatalf("expected dm quote, got %#v", dmReply.QuotedMessageID)
	}

	// quoting a channel message from a DM → 400
	status, _ = postJSONStatus(t, server.URL+"/api/dms/"+dm.ID+"/messages", map[string]any{
		"body":              "leak",
		"quoted_message_id": root.ID,
	})
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400 for cross-context dm quote, got %d", status)
	}
}
