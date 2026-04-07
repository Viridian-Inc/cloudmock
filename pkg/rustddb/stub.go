//go:build !rustddb

package rustddb

// Store is a no-op stub when the Rust DynamoDB library is not available.
type Store struct{}

// Result of a DynamoDB operation.
type Result struct {
	Status int
	Body   []byte
}

// New returns nil when Rust DDB is not available.
func New(accountID, region string) *Store {
	return nil
}

// Close is a no-op.
func (s *Store) Close() {}

// Handle always returns status=0 (not handled) so the caller falls back to Go.
func (s *Store) Handle(action string, body []byte) Result {
	return Result{Status: 0}
}
