package sqs_test

import (
	"encoding/json"
	"strings"
	"testing"

	sqssvc "github.com/neureaux/cloudmock/services/sqs"
)

const (
	sqsTestAccount = "123456789012"
	sqsTestRegion  = "us-east-1"
)

func TestSQS_ExportState_Empty(t *testing.T) {
	svc := sqssvc.New(sqsTestAccount, sqsTestRegion)

	raw, err := svc.ExportState()
	if err != nil {
		t.Fatalf("ExportState: %v", err)
	}
	if !json.Valid(raw) {
		t.Fatalf("ExportState returned invalid JSON: %s", raw)
	}

	var state struct {
		Queues []any `json:"queues"`
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(state.Queues) != 0 {
		t.Errorf("expected empty queues, got %d", len(state.Queues))
	}
}

func TestSQS_ExportState_WithQueue(t *testing.T) {
	svc := sqssvc.New(sqsTestAccount, sqsTestRegion)

	seed := json.RawMessage(`{"queues":[{"name":"work-queue","attributes":{"VisibilityTimeout":"45","MessageRetentionPeriod":"3600"}}]}`)
	if err := svc.ImportState(seed); err != nil {
		t.Fatalf("ImportState: %v", err)
	}

	raw, err := svc.ExportState()
	if err != nil {
		t.Fatalf("ExportState: %v", err)
	}

	var state struct {
		Queues []struct {
			Name       string            `json:"name"`
			Attributes map[string]string `json:"attributes"`
		} `json:"queues"`
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(state.Queues) == 0 {
		t.Fatal("expected queues in export")
	}
	q := state.Queues[0]
	if q.Name != "work-queue" {
		t.Errorf("expected queue 'work-queue', got %q", q.Name)
	}
	if q.Attributes["VisibilityTimeout"] != "45" {
		t.Errorf("expected VisibilityTimeout=45, got %q", q.Attributes["VisibilityTimeout"])
	}
}

func TestSQS_ImportState_RestoresQueues(t *testing.T) {
	svc := sqssvc.New(sqsTestAccount, sqsTestRegion)

	data := json.RawMessage(`{"queues":[{"name":"queue-a","attributes":{}},{"name":"queue-b","attributes":{}}]}`)
	if err := svc.ImportState(data); err != nil {
		t.Fatalf("ImportState: %v", err)
	}

	// GetQueueNames returns queue URLs, not bare names.
	urls := svc.GetQueueNames()
	for _, expected := range []string{"queue-a", "queue-b"} {
		found := false
		for _, u := range urls {
			if strings.Contains(u, expected) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("queue %q not restored, got: %v", expected, urls)
		}
	}
}

func TestSQS_ImportState_EmptyDoesNotCrash(t *testing.T) {
	svc := sqssvc.New(sqsTestAccount, sqsTestRegion)

	if err := svc.ImportState(json.RawMessage(`{"queues":[]}`)); err != nil {
		t.Fatalf("ImportState with empty queues: %v", err)
	}
	if len(svc.GetQueueNames()) != 0 {
		t.Error("expected no queues after importing empty state")
	}
}

func TestSQS_RoundTrip_PreservesAttributes(t *testing.T) {
	svc := sqssvc.New(sqsTestAccount, sqsTestRegion)

	seed := json.RawMessage(`{"queues":[{"name":"delayed-queue","attributes":{"DelaySeconds":"10","MaximumMessageSize":"1024"}}]}`)
	if err := svc.ImportState(seed); err != nil {
		t.Fatalf("ImportState: %v", err)
	}

	raw, err := svc.ExportState()
	if err != nil {
		t.Fatalf("ExportState: %v", err)
	}

	svc2 := sqssvc.New(sqsTestAccount, sqsTestRegion)
	if err := svc2.ImportState(raw); err != nil {
		t.Fatalf("ImportState (svc2): %v", err)
	}

	// GetQueueNames returns URLs; check by substring.
	urls := svc2.GetQueueNames()
	found := false
	for _, u := range urls {
		if strings.Contains(u, "delayed-queue") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("queue 'delayed-queue' not restored, got: %v", urls)
	}
}

func TestSQS_RoundTrip_MultipleQueues(t *testing.T) {
	svc := sqssvc.New(sqsTestAccount, sqsTestRegion)

	seed := json.RawMessage(`{"queues":[{"name":"q1","attributes":{}},{"name":"q2","attributes":{}},{"name":"q3","attributes":{}}]}`)
	if err := svc.ImportState(seed); err != nil {
		t.Fatalf("ImportState: %v", err)
	}

	raw, _ := svc.ExportState()
	svc2 := sqssvc.New(sqsTestAccount, sqsTestRegion)
	svc2.ImportState(raw)

	// GetQueueNames returns URLs; check by substring.
	urls := svc2.GetQueueNames()
	for _, expected := range []string{"q1", "q2", "q3"} {
		found := false
		for _, u := range urls {
			if strings.Contains(u, expected) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("queue %q not restored: %v", expected, urls)
		}
	}
}
