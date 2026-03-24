package admin

import (
	"encoding/json"
	"sync"
)

// EventBroadcaster fans out SSE events to all connected clients.
type EventBroadcaster struct {
	mu      sync.RWMutex
	clients map[chan string]struct{}
}

// NewEventBroadcaster creates a new EventBroadcaster.
func NewEventBroadcaster() *EventBroadcaster {
	return &EventBroadcaster{
		clients: make(map[chan string]struct{}),
	}
}

// Subscribe registers a new client channel (buffered, size 100).
func (b *EventBroadcaster) Subscribe() chan string {
	ch := make(chan string, 100)
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

// Unsubscribe removes a client channel and closes it.
func (b *EventBroadcaster) Unsubscribe(ch chan string) {
	b.mu.Lock()
	delete(b.clients, ch)
	b.mu.Unlock()
	close(ch)
}

// Broadcast sends a JSON-encoded event to all connected clients.
// If a client's channel is full, the event is dropped for that client.
func (b *EventBroadcaster) Broadcast(eventType string, data any) {
	payload, err := json.Marshal(map[string]any{
		"type": eventType,
		"data": data,
	})
	if err != nil {
		return
	}
	msg := string(payload)

	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.clients {
		select {
		case ch <- msg:
		default:
			// Drop event for slow client.
		}
	}
}
