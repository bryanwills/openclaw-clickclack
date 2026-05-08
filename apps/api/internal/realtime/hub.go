package realtime

import (
	"sync"

	"github.com/openclaw/clickclack/apps/api/internal/store"
)

type Hub struct {
	mu   sync.RWMutex
	subs map[string]map[chan store.Event]struct{}
}

func NewHub() *Hub {
	return &Hub{subs: map[string]map[chan store.Event]struct{}{}}
}

func (h *Hub) Subscribe(workspaceID string) (<-chan store.Event, func()) {
	ch := make(chan store.Event, 32)
	h.mu.Lock()
	if h.subs[workspaceID] == nil {
		h.subs[workspaceID] = map[chan store.Event]struct{}{}
	}
	h.subs[workspaceID][ch] = struct{}{}
	h.mu.Unlock()
	return ch, func() {
		h.mu.Lock()
		delete(h.subs[workspaceID], ch)
		close(ch)
		h.mu.Unlock()
	}
}

func (h *Hub) Publish(event store.Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.subs[event.WorkspaceID] {
		select {
		case ch <- event:
		default:
		}
	}
}

func (h *Hub) PublishMany(events []store.Event) {
	for _, event := range events {
		h.Publish(event)
	}
}
