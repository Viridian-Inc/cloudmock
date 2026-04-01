package replay

import "time"

// ReplayEvent represents a single DOM mutation or user interaction captured
// during a session recording.
type ReplayEvent struct {
	Type      string `json:"type"`      // "mutation", "scroll", "click", "input", "resize"
	Timestamp int64  `json:"timestamp"` // ms since session start
	Data      any    `json:"data"`      // event-specific payload
}

// Session is a recorded browser session containing DOM replay events.
type Session struct {
	ID        string        `json:"id"`
	URL       string        `json:"url"`
	UserAgent string        `json:"user_agent"`
	StartedAt time.Time     `json:"started_at"`
	Duration  int64         `json:"duration_ms"`
	Events    []ReplayEvent `json:"events"`
	ErrorIDs  []string      `json:"error_ids"` // linked error groups
	Width     int           `json:"width"`
	Height    int           `json:"height"`
}

// Store defines the interface for persisting and querying replay sessions.
type Store interface {
	// SaveSession persists a recorded session.
	SaveSession(session Session) error

	// GetSession returns a single session by ID, or nil if not found.
	GetSession(id string) (*Session, error)

	// ListSessions returns the most recent sessions, newest first.
	ListSessions(limit int) ([]Session, error)

	// LinkError associates an error ID with a session.
	LinkError(sessionID, errorID string) error
}
