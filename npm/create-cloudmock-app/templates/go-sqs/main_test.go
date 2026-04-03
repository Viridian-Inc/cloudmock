package main_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sdk "github.com/neureaux/cloudmock/sdk"
)

func setupTest(t *testing.T) (*httptest.Server, string, func()) {
	t.Helper()
	cm := sdk.New()
	cfg := cm.Config()

	client := sqs.NewFromConfig(cfg)
	ctx := context.Background()

	q, err := client.CreateQueue(ctx, &sqs.CreateQueueInput{
		QueueName: aws.String("messages"),
	})
	if err != nil {
		t.Fatalf("create queue: %v", err)
	}

	qURL := aws.ToString(q.QueueUrl)
	t.Setenv("SQS_QUEUE_URL", qURL)

	mux := NewServer(client)
	srv := httptest.NewServer(mux)
	return srv, qURL, func() {
		srv.Close()
		cm.Stop()
	}
}

func postJSON(t *testing.T, srv *httptest.Server, path string, body any) *http.Response {
	t.Helper()
	b, _ := json.Marshal(body)
	res, err := http.Post(srv.URL+path, "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestSendMessage(t *testing.T) {
	srv, _, cleanup := setupTest(t)
	defer cleanup()

	res := postJSON(t, srv, "/messages", map[string]any{"message": "hello"})
	if res.StatusCode != 202 {
		t.Fatalf("expected 202, got %d", res.StatusCode)
	}

	var body map[string]string
	json.NewDecoder(res.Body).Decode(&body)
	if body["messageId"] == "" {
		t.Error("expected non-empty messageId")
	}
}

func TestReceiveMessages(t *testing.T) {
	srv, _, cleanup := setupTest(t)
	defer cleanup()

	postJSON(t, srv, "/messages", map[string]any{"message": "first"})
	postJSON(t, srv, "/messages", map[string]any{"message": "second"})

	res, err := http.Get(srv.URL + "/messages")
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var messages []map[string]any
	json.NewDecoder(res.Body).Decode(&messages)
	if len(messages) < 1 {
		t.Errorf("expected at least 1 message, got %d", len(messages))
	}
}

func TestSendMessageMissingField(t *testing.T) {
	srv, _, cleanup := setupTest(t)
	defer cleanup()

	res := postJSON(t, srv, "/messages", map[string]any{"data": "no message key"})
	if res.StatusCode != 400 {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
}
