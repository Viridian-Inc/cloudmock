package eventbus

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBus(t *testing.T) {
	bus := NewBus()
	require.NotNil(t, bus)
}

func TestSubscribeAndPublishSync(t *testing.T) {
	bus := NewBus()
	var received *Event
	var mu sync.Mutex

	bus.Subscribe(&Subscription{
		Source: "s3",
		Types:  []string{"s3:ObjectCreated:Put"},
		Handler: func(e *Event) error {
			mu.Lock()
			received = e
			mu.Unlock()
			return nil
		},
	})

	event := &Event{
		Source:    "s3",
		Type:      "s3:ObjectCreated:Put",
		Detail:    map[string]any{"bucket": "my-bucket", "key": "my-key"},
		Time:      time.Now(),
		Region:    "us-east-1",
		AccountID: "123456789012",
	}

	errs := bus.PublishSync(event)
	assert.Empty(t, errs)

	mu.Lock()
	assert.Equal(t, event, received)
	mu.Unlock()
}

func TestSubscriptionFiltering_SourceMismatch(t *testing.T) {
	bus := NewBus()
	called := false

	bus.Subscribe(&Subscription{
		Source: "sns",
		Types:  []string{"*"},
		Handler: func(e *Event) error {
			called = true
			return nil
		},
	})

	bus.PublishSync(&Event{Source: "s3", Type: "s3:ObjectCreated:Put"})
	assert.False(t, called)
}

func TestSubscriptionFiltering_TypeMismatch(t *testing.T) {
	bus := NewBus()
	called := false

	bus.Subscribe(&Subscription{
		Source: "s3",
		Types:  []string{"s3:ObjectRemoved:Delete"},
		Handler: func(e *Event) error {
			called = true
			return nil
		},
	})

	bus.PublishSync(&Event{Source: "s3", Type: "s3:ObjectCreated:Put"})
	assert.False(t, called)
}

func TestSubscriptionFiltering_WildcardType(t *testing.T) {
	bus := NewBus()
	var received []*Event
	var mu sync.Mutex

	bus.Subscribe(&Subscription{
		Source: "s3",
		Types:  []string{"s3:ObjectCreated:*"},
		Handler: func(e *Event) error {
			mu.Lock()
			received = append(received, e)
			mu.Unlock()
			return nil
		},
	})

	bus.PublishSync(&Event{Source: "s3", Type: "s3:ObjectCreated:Put"})
	bus.PublishSync(&Event{Source: "s3", Type: "s3:ObjectCreated:Copy"})
	bus.PublishSync(&Event{Source: "s3", Type: "s3:ObjectRemoved:Delete"})

	mu.Lock()
	assert.Len(t, received, 2)
	mu.Unlock()
}

func TestSubscriptionFiltering_EmptyTypesMatchAll(t *testing.T) {
	bus := NewBus()
	count := 0

	bus.Subscribe(&Subscription{
		Source: "s3",
		Handler: func(e *Event) error {
			count++
			return nil
		},
	})

	bus.PublishSync(&Event{Source: "s3", Type: "s3:ObjectCreated:Put"})
	bus.PublishSync(&Event{Source: "s3", Type: "s3:ObjectRemoved:Delete"})
	assert.Equal(t, 2, count)
}

func TestUnsubscribe(t *testing.T) {
	bus := NewBus()
	called := false

	id := bus.Subscribe(&Subscription{
		Source: "s3",
		Handler: func(e *Event) error {
			called = true
			return nil
		},
	})

	bus.Unsubscribe(id)
	bus.PublishSync(&Event{Source: "s3", Type: "s3:ObjectCreated:Put"})
	assert.False(t, called)
}

func TestPublishAsync(t *testing.T) {
	bus := NewBus()
	var mu sync.Mutex
	var received *Event
	done := make(chan struct{})

	bus.Subscribe(&Subscription{
		Source: "s3",
		Handler: func(e *Event) error {
			mu.Lock()
			received = e
			mu.Unlock()
			close(done)
			return nil
		},
	})

	event := &Event{Source: "s3", Type: "s3:ObjectCreated:Put"}
	bus.Publish(event)

	select {
	case <-done:
		mu.Lock()
		assert.Equal(t, event, received)
		mu.Unlock()
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for async publish")
	}
}

func TestMultipleSubscribers(t *testing.T) {
	bus := NewBus()
	var mu sync.Mutex
	count := 0

	for i := 0; i < 3; i++ {
		bus.Subscribe(&Subscription{
			Source: "s3",
			Types:  []string{"s3:ObjectCreated:Put"},
			Handler: func(e *Event) error {
				mu.Lock()
				count++
				mu.Unlock()
				return nil
			},
		})
	}

	bus.PublishSync(&Event{Source: "s3", Type: "s3:ObjectCreated:Put"})
	mu.Lock()
	assert.Equal(t, 3, count)
	mu.Unlock()
}

func TestMatchType(t *testing.T) {
	tests := []struct {
		pattern   string
		eventType string
		want      bool
	}{
		{"*", "anything", true},
		{"s3:ObjectCreated:Put", "s3:ObjectCreated:Put", true},
		{"s3:ObjectCreated:Put", "s3:ObjectRemoved:Delete", false},
		{"s3:ObjectCreated:*", "s3:ObjectCreated:Put", true},
		{"s3:ObjectCreated:*", "s3:ObjectCreated:Copy", true},
		{"s3:ObjectCreated:*", "s3:ObjectRemoved:Delete", false},
		{"s3:*", "s3:ObjectCreated:Put", true},
		{"s3:*", "s3:ObjectRemoved:Delete", true},
	}

	for _, tt := range tests {
		got := matchType(tt.pattern, tt.eventType)
		assert.Equal(t, tt.want, got, "matchType(%q, %q)", tt.pattern, tt.eventType)
	}
}
