package realtime

import (
	"testing"
	"time"

	"github.com/openclaw/clickclack/apps/api/internal/store"
)

func TestHubSubscribePublishAndUnsubscribe(t *testing.T) {
	t.Parallel()
	hub := NewHub()
	events, unsubscribe := hub.Subscribe("wsp_1")
	event := store.Event{ID: "evt_1", WorkspaceID: "wsp_1", Type: "message.created"}
	hub.Publish(event)

	select {
	case got := <-events:
		if got.ID != event.ID {
			t.Fatalf("expected %s, got %s", event.ID, got.ID)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}

	hub.PublishMany([]store.Event{{ID: "evt_2", WorkspaceID: "wsp_other"}})
	select {
	case got := <-events:
		t.Fatalf("unexpected event for other workspace: %#v", got)
	case <-time.After(10 * time.Millisecond):
	}

	unsubscribe()
	_, ok := <-events
	if ok {
		t.Fatal("expected subscription channel to close")
	}
}
