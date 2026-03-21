package eventbus

import (
	"crypto/rand"
	"fmt"
	"strings"
	"sync"
)

// Bus is an in-process event bus for cross-service communication within cloudmock.
type Bus struct {
	mu            sync.RWMutex
	subscriptions []*Subscription
}

// NewBus creates a new event bus.
func NewBus() *Bus {
	return &Bus{
		subscriptions: make([]*Subscription, 0),
	}
}

// Subscribe registers a subscription on the bus. Returns the subscription ID.
// If sub.ID is empty, a random ID is generated.
func (b *Bus) Subscribe(sub *Subscription) string {
	b.mu.Lock()
	defer b.mu.Unlock()

	if sub.ID == "" {
		sub.ID = randomID()
	}
	b.subscriptions = append(b.subscriptions, sub)
	return sub.ID
}

// Unsubscribe removes the subscription with the given ID.
func (b *Bus) Unsubscribe(id string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	updated := make([]*Subscription, 0, len(b.subscriptions))
	for _, s := range b.subscriptions {
		if s.ID != id {
			updated = append(updated, s)
		}
	}
	b.subscriptions = updated
}

// Publish fans out an event to all matching subscriptions asynchronously.
// Each matching handler is invoked in its own goroutine. Errors from handlers
// are silently discarded (fire-and-forget semantics).
func (b *Bus) Publish(event *Event) {
	b.mu.RLock()
	matching := make([]*Subscription, 0)
	for _, sub := range b.subscriptions {
		if matches(sub, event) {
			matching = append(matching, sub)
		}
	}
	b.mu.RUnlock()

	for _, sub := range matching {
		handler := sub.Handler
		go handler(event) //nolint:errcheck
	}
}

// PublishSync fans out an event to all matching subscriptions synchronously.
// This is useful in tests where you need to wait for delivery to complete.
func (b *Bus) PublishSync(event *Event) []error {
	b.mu.RLock()
	matching := make([]*Subscription, 0)
	for _, sub := range b.subscriptions {
		if matches(sub, event) {
			matching = append(matching, sub)
		}
	}
	b.mu.RUnlock()

	var errs []error
	for _, sub := range matching {
		if err := sub.Handler(event); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

// matches checks whether a subscription should receive an event.
// A subscription matches if:
//  1. sub.Source is empty or equals event.Source
//  2. sub.Types is empty, or at least one type pattern matches event.Type
//
// Type patterns support trailing wildcard: "s3:*" matches "s3:ObjectCreated:Put".
func matches(sub *Subscription, event *Event) bool {
	// Check source filter.
	if sub.Source != "" && sub.Source != event.Source {
		return false
	}

	// Check type filter.
	if len(sub.Types) == 0 {
		return true
	}
	for _, pattern := range sub.Types {
		if matchType(pattern, event.Type) {
			return true
		}
	}
	return false
}

// matchType performs simple glob matching: a trailing "*" matches any suffix.
// An exact match is also accepted.
func matchType(pattern, eventType string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(eventType, prefix)
	}
	return pattern == eventType
}

func randomID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
