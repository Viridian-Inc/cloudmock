package cloudtrail

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReplay_SendsRequests(t *testing.T) {
	var mu sync.Mutex
	var received []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		received = append(received, r.Header.Get("X-Amz-Target"))
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	events := []CloudTrailEvent{
		{EventSource: "dynamodb.amazonaws.com", EventName: "CreateTable", EventTime: "2026-04-01T10:00:00Z", AWSRegion: "us-east-1", RequestParameters: map[string]any{"tableName": "A"}},
		{EventSource: "dynamodb.amazonaws.com", EventName: "CreateTable", EventTime: "2026-04-01T10:01:00Z", AWSRegion: "us-east-1", RequestParameters: map[string]any{"tableName": "B"}},
		{EventSource: "dynamodb.amazonaws.com", EventName: "CreateTable", EventTime: "2026-04-01T10:02:00Z", AWSRegion: "us-east-1", RequestParameters: map[string]any{"tableName": "C"}},
	}

	result, err := Replay(events, ReplayConfig{Endpoint: srv.URL, Speed: 0})
	require.NoError(t, err)
	assert.Equal(t, 3, result.Replayed)
	assert.Equal(t, 3, result.Succeeded)

	mu.Lock()
	assert.Len(t, received, 3)
	mu.Unlock()
}

func TestReplay_ChronologicalOrder(t *testing.T) {
	var mu sync.Mutex
	var order []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target := r.Header.Get("X-Amz-Target")
		mu.Lock()
		order = append(order, target)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// Events in reverse chronological order.
	events := []CloudTrailEvent{
		{EventSource: "dynamodb.amazonaws.com", EventName: "PutItem", EventTime: "2026-04-01T12:00:00Z", AWSRegion: "us-east-1", RequestParameters: map[string]any{}},
		{EventSource: "dynamodb.amazonaws.com", EventName: "CreateTable", EventTime: "2026-04-01T10:00:00Z", AWSRegion: "us-east-1", RequestParameters: map[string]any{"tableName": "X"}},
	}

	result, err := Replay(events, ReplayConfig{Endpoint: srv.URL, Speed: 0})
	require.NoError(t, err)
	assert.Equal(t, 2, result.Replayed)

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, order, 2)
	// CreateTable should come before PutItem because it has an earlier timestamp.
	assert.Equal(t, "DynamoDB_20120810.CreateTable", order[0])
	assert.Equal(t, "DynamoDB_20120810.PutItem", order[1])
}

func TestReplay_SkipsReadOnly(t *testing.T) {
	var mu sync.Mutex
	var count int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		count++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	events := []CloudTrailEvent{
		{EventSource: "dynamodb.amazonaws.com", EventName: "CreateTable", EventTime: "2026-04-01T10:00:00Z", AWSRegion: "us-east-1", ReadOnly: false, RequestParameters: map[string]any{"tableName": "T"}},
		{EventSource: "dynamodb.amazonaws.com", EventName: "GetItem", EventTime: "2026-04-01T10:01:00Z", AWSRegion: "us-east-1", ReadOnly: true, RequestParameters: map[string]any{}},
		{EventSource: "s3.amazonaws.com", EventName: "CreateBucket", EventTime: "2026-04-01T10:02:00Z", AWSRegion: "us-east-1", ReadOnly: false, RequestParameters: map[string]any{"bucketName": "b"}},
	}

	result, err := Replay(events, ReplayConfig{Endpoint: srv.URL, Speed: 0, FilterWrite: true})
	require.NoError(t, err)
	assert.Equal(t, 3, result.TotalEvents)
	assert.Equal(t, 2, result.Replayed)
	assert.Equal(t, 1, result.Skipped)

	mu.Lock()
	assert.Equal(t, 2, count)
	mu.Unlock()
}

func TestReplay_ServiceFilter(t *testing.T) {
	var mu sync.Mutex
	var count int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		count++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	events := []CloudTrailEvent{
		{EventSource: "dynamodb.amazonaws.com", EventName: "CreateTable", EventTime: "2026-04-01T10:00:00Z", AWSRegion: "us-east-1", RequestParameters: map[string]any{"tableName": "T"}},
		{EventSource: "s3.amazonaws.com", EventName: "CreateBucket", EventTime: "2026-04-01T10:01:00Z", AWSRegion: "us-east-1", RequestParameters: map[string]any{"bucketName": "b"}},
		{EventSource: "sqs.amazonaws.com", EventName: "CreateQueue", EventTime: "2026-04-01T10:02:00Z", AWSRegion: "us-east-1", RequestParameters: map[string]any{"QueueName": "q"}},
	}

	result, err := Replay(events, ReplayConfig{
		Endpoint: srv.URL,
		Speed:    0,
		Services: []string{"s3"},
	})
	require.NoError(t, err)
	assert.Equal(t, 3, result.TotalEvents)
	assert.Equal(t, 1, result.Replayed)
	assert.Equal(t, 2, result.Skipped)

	mu.Lock()
	assert.Equal(t, 1, count)
	mu.Unlock()
}

func TestReplay_ReportCounts(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	events := []CloudTrailEvent{
		{EventSource: "dynamodb.amazonaws.com", EventName: "CreateTable", EventTime: "2026-04-01T10:00:00Z", AWSRegion: "us-east-1", RequestParameters: map[string]any{"tableName": "A"}},
		{EventSource: "dynamodb.amazonaws.com", EventName: "CreateTable", EventTime: "2026-04-01T10:01:00Z", AWSRegion: "us-east-1", RequestParameters: map[string]any{"tableName": "B"}},
		{EventSource: "s3.amazonaws.com", EventName: "CreateBucket", EventTime: "2026-04-01T10:02:00Z", AWSRegion: "us-east-1", RequestParameters: map[string]any{"bucketName": "c"}},
	}

	result, err := Replay(events, ReplayConfig{Endpoint: srv.URL, Speed: 0})
	require.NoError(t, err)
	assert.Equal(t, 3, result.TotalEvents)
	assert.Equal(t, 3, result.Replayed)
	assert.Equal(t, 2, result.Succeeded)
	assert.Equal(t, 1, result.Failed)
	assert.Len(t, result.Errors, 1)
	assert.Equal(t, 500, result.Errors[0].Status)
}
