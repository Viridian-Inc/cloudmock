package sdk

import "testing"

// Close must tear down background work (the DynamoDB TTL reaper) and be safe to
// call more than once.
func TestCloudMock_Close_Idempotent(t *testing.T) {
	cm := New()
	cm.Close()
	cm.Close()
}
