package incident

import "time"

// Comment represents a user comment on an incident.
type Comment struct {
	ID         string    `json:"id"`
	IncidentID string    `json:"incident_id"`
	Author     string    `json:"author"`
	Body       string    `json:"body"`
	Mentions   []string  `json:"mentions"` // @username references
	CreatedAt  time.Time `json:"created_at"`
}
