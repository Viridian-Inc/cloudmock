package uptime

// Store defines the interface for persisting uptime checks and results.
type Store interface {
	// CreateCheck persists a new check.
	CreateCheck(check Check) error

	// UpdateCheck updates an existing check.
	UpdateCheck(check Check) error

	// DeleteCheck removes a check and its results.
	DeleteCheck(id string) error

	// GetCheck returns a check by ID.
	GetCheck(id string) (*Check, error)

	// ListChecks returns all checks.
	ListChecks() ([]Check, error)

	// AddResult appends a check result (circular buffer per check).
	AddResult(result CheckResult) error

	// Results returns the result history for a check, newest first.
	Results(checkID string) ([]CheckResult, error)

	// LastResult returns the most recent result for a check.
	LastResult(checkID string) (*CheckResult, error)

	// LastFailure returns the most recent failed result for a check.
	LastFailure(checkID string) (*CheckResult, error)

	// ConsecutiveFailures returns the number of consecutive failures
	// for a check (counting backwards from the most recent result).
	ConsecutiveFailures(checkID string) int
}
