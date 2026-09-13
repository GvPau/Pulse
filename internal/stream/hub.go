package stream

import (
	"encoding/json"
	"sync"
	"uuid"
)

type Event struct {
	Type   string
	UserId uuid.UUID
	Data   json.RawMessage
}

type subscriber struct {
	userID uuid.UUID
	ch     chan []byte
}

type Hub struct {
	mu   sync.RWMutex
	subs map[*subscriber]struct{}
}

func NewHub() *Hub {
	return &Hub{subs: make(map[*subscriber]struct{})}
}

// Subscribe register a client from the user; returns its chanel of frames
func (h *Hub) Subscribe(userID uuid.UUID) <-chan []byte {
	s := &subscriber{userID: userID, ch: make(chan []byte, 64)}
	h.mu.Lock()
	h.subs[s] = struct{}{}
	h.mu.Unlock()
	return s.ch
}

func (h *Hub) Unsubscribe(userID uuid.UUID, ch <-chan []byte) {
	h.mu.Lock()
	for s := range h.subs {
		if s.userID == userID && s.ch == ch {
			delete(h.subs, s)
			return
		}
	}
	h.mu.Unlock()
}

func (h *Hub) Publish(ev Event) {
	frame := []byte("event: " + ev.Type + "\ndata: " + string(ev.Data) + "\n\n")
	h.mu.RLock()
	defer h.mu.RUnlock()
	for s := range h.subs {
		if s.userID != ev.UserId {
			continue
		}
		select {
		case s.ch <- frame:
		default:
		}
	}
}
