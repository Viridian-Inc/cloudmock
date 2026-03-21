package eventbus

import "time"

// Event represents an internal cross-service event within cloudmock.
type Event struct {
	Source    string                 // service that produced the event, e.g. "s3"
	Type      string                 // event type, e.g. "s3:ObjectCreated:Put"
	Detail    map[string]interface{} // event payload
	Time      time.Time
	Region    string
	AccountID string
}

// Subscription describes a listener registered on the event bus.
type Subscription struct {
	ID      string
	Source  string   // filter by source service
	Types   []string // filter by event types (supports prefix matching with *)
	Handler EventHandler
}

// EventHandler is a callback invoked when a matching event is published.
type EventHandler func(event *Event) error
