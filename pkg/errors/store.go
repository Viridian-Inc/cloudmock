package errors

// ErrorStore defines the interface for persisting and querying structured errors.
type ErrorStore interface {
	// IngestError ingests a single error event, creating or updating the group.
	IngestError(event ErrorEvent) error

	// GetGroups returns error groups filtered by status. Pass "" for all statuses.
	GetGroups(status string, limit int) ([]ErrorGroup, error)

	// GetGroup returns a single error group by ID (fingerprint).
	GetGroup(id string) (*ErrorGroup, error)

	// GetEvents returns error events for a group, most recent first.
	GetEvents(groupID string, limit int) ([]ErrorEvent, error)

	// UpdateGroupStatus sets the status of an error group.
	UpdateGroupStatus(id string, status string) error
}
