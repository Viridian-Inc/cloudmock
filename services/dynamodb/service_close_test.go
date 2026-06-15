package dynamodb

import "testing"

func TestDynamoDBService_CloseStopsReaper(t *testing.T) {
	svc := New("000000000000", "us-east-1")

	svc.Close()
	// ttlDone must be closed so the reaper goroutine's <-done case fires and it
	// returns (a receive on a closed channel proceeds immediately).
	select {
	case <-svc.ttlDone:
	default:
		t.Fatal("ttlDone not closed after Close()")
	}

	// Idempotent: a second Close must not panic (close() of a closed channel does).
	svc.Close()
}
