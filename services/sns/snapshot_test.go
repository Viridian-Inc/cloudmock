package sns_test

import (
	"encoding/json"
	"strings"
	"testing"

	snssvc "github.com/neureaux/cloudmock/services/sns"
)

const (
	snsTestAccount = "123456789012"
	snsTestRegion  = "us-east-1"
)

func TestSNS_ExportState_Empty(t *testing.T) {
	svc := snssvc.New(snsTestAccount, snsTestRegion)

	raw, err := svc.ExportState()
	if err != nil {
		t.Fatalf("ExportState: %v", err)
	}
	if !json.Valid(raw) {
		t.Fatalf("ExportState returned invalid JSON: %s", raw)
	}

	var state struct {
		Topics []any `json:"topics"`
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(state.Topics) != 0 {
		t.Errorf("expected empty topics, got %d", len(state.Topics))
	}
}

func TestSNS_ExportState_WithTopicAndSubscription(t *testing.T) {
	svc := snssvc.New(snsTestAccount, snsTestRegion)

	seed := json.RawMessage(`{"topics":[{"name":"notifications","subscriptions":[{"protocol":"email","endpoint":"user@example.com"},{"protocol":"sqs","endpoint":"arn:aws:sqs:us-east-1:123456789012:my-queue"}]}]}`)
	if err := svc.ImportState(seed); err != nil {
		t.Fatalf("ImportState: %v", err)
	}

	raw, err := svc.ExportState()
	if err != nil {
		t.Fatalf("ExportState: %v", err)
	}

	var state struct {
		Topics []struct {
			Name          string `json:"name"`
			Subscriptions []struct {
				Protocol string `json:"protocol"`
				Endpoint string `json:"endpoint"`
			} `json:"subscriptions"`
		} `json:"topics"`
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(state.Topics) == 0 {
		t.Fatal("expected topics in export")
	}

	var found *struct {
		Name          string
		Subscriptions []struct {
			Protocol string `json:"protocol"`
			Endpoint string `json:"endpoint"`
		}
	}
	for i := range state.Topics {
		if state.Topics[i].Name == "notifications" {
			found = &struct {
				Name          string
				Subscriptions []struct {
					Protocol string `json:"protocol"`
					Endpoint string `json:"endpoint"`
				}
			}{Name: state.Topics[i].Name, Subscriptions: state.Topics[i].Subscriptions}
			break
		}
	}
	if found == nil {
		t.Fatal("topic 'notifications' not found in export")
	}
	if len(found.Subscriptions) != 2 {
		t.Errorf("expected 2 subscriptions, got %d", len(found.Subscriptions))
	}
}

func TestSNS_ImportState_RestoresTopics(t *testing.T) {
	svc := snssvc.New(snsTestAccount, snsTestRegion)

	data := json.RawMessage(`{"topics":[{"name":"topic-a","subscriptions":[]},{"name":"topic-b","subscriptions":[]}]}`)
	if err := svc.ImportState(data); err != nil {
		t.Fatalf("ImportState: %v", err)
	}

	topics := svc.GetAllTopics()
	for _, expected := range []string{"topic-a", "topic-b"} {
		found := false
		for _, arn := range topics {
			if strings.Contains(arn, expected) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("topic %q not restored, got: %v", expected, topics)
		}
	}
}

func TestSNS_ImportState_EmptyDoesNotCrash(t *testing.T) {
	svc := snssvc.New(snsTestAccount, snsTestRegion)

	if err := svc.ImportState(json.RawMessage(`{"topics":[]}`)); err != nil {
		t.Fatalf("ImportState with empty topics: %v", err)
	}
	if len(svc.GetAllTopics()) != 0 {
		t.Error("expected no topics after importing empty state")
	}
}

func TestSNS_RoundTrip_PreservesSubscriptions(t *testing.T) {
	svc := snssvc.New(snsTestAccount, snsTestRegion)

	seed := json.RawMessage(`{"topics":[{"name":"events","subscriptions":[{"protocol":"sqs","endpoint":"arn:aws:sqs:us-east-1:123456789012:events-queue"}]}]}`)
	if err := svc.ImportState(seed); err != nil {
		t.Fatalf("ImportState: %v", err)
	}

	raw, err := svc.ExportState()
	if err != nil {
		t.Fatalf("ExportState: %v", err)
	}

	svc2 := snssvc.New(snsTestAccount, snsTestRegion)
	if err := svc2.ImportState(raw); err != nil {
		t.Fatalf("ImportState (svc2): %v", err)
	}

	topics := svc2.GetAllTopics()
	found := false
	for _, arn := range topics {
		if strings.Contains(arn, "events") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("topic 'events' not restored, got: %v", topics)
	}

	subs := svc2.GetAllSubscriptions()
	if len(subs) == 0 {
		t.Error("expected subscriptions after import")
	}
}

func TestSNS_RoundTrip_MultipleTopics(t *testing.T) {
	svc := snssvc.New(snsTestAccount, snsTestRegion)

	seed := json.RawMessage(`{"topics":[{"name":"topic-x","subscriptions":[]},{"name":"topic-y","subscriptions":[]},{"name":"topic-z","subscriptions":[]}]}`)
	if err := svc.ImportState(seed); err != nil {
		t.Fatalf("ImportState: %v", err)
	}

	raw, _ := svc.ExportState()
	svc2 := snssvc.New(snsTestAccount, snsTestRegion)
	svc2.ImportState(raw)

	topics := svc2.GetAllTopics()
	for _, expected := range []string{"topic-x", "topic-y", "topic-z"} {
		found := false
		for _, arn := range topics {
			if strings.Contains(arn, expected) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("topic %q not restored: %v", expected, topics)
		}
	}
}
