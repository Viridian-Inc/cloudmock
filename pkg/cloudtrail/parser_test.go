package cloudtrail

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_SingleEvent(t *testing.T) {
	data := `{"Records":[{"eventSource":"s3.amazonaws.com","eventName":"CreateBucket","eventTime":"2026-04-01T10:00:00Z","awsRegion":"us-east-1","requestParameters":{"bucketName":"test"}}]}`
	events, err := ParseJSON([]byte(data))
	require.NoError(t, err)
	assert.Len(t, events, 1)
	assert.Equal(t, "s3", events[0].ServiceName())
	assert.Equal(t, "CreateBucket", events[0].EventName)
}

func TestParse_MultipleEvents(t *testing.T) {
	data := `{"Records":[
		{"eventSource":"dynamodb.amazonaws.com","eventName":"CreateTable","eventTime":"2026-04-01T10:00:00Z","awsRegion":"us-east-1","requestParameters":{"tableName":"Users"}},
		{"eventSource":"sqs.amazonaws.com","eventName":"CreateQueue","eventTime":"2026-04-01T10:01:00Z","awsRegion":"us-east-1","requestParameters":{"QueueName":"tasks"}}
	]}`
	events, err := ParseJSON([]byte(data))
	require.NoError(t, err)
	assert.Len(t, events, 2)
	assert.Equal(t, "dynamodb", events[0].ServiceName())
	assert.Equal(t, "sqs", events[1].ServiceName())
}

func TestParse_FilterWriteEvents(t *testing.T) {
	events := []CloudTrailEvent{
		{EventName: "CreateTable", ReadOnly: false},
		{EventName: "DescribeTable", ReadOnly: true},
		{EventName: "PutItem", ReadOnly: false},
	}
	writes := FilterWriteEvents(events)
	assert.Len(t, writes, 2)
	assert.Equal(t, "CreateTable", writes[0].EventName)
	assert.Equal(t, "PutItem", writes[1].EventName)
}

func TestParse_SortByTime(t *testing.T) {
	events := []CloudTrailEvent{
		{EventName: "Third", EventTime: "2026-04-01T12:00:00Z"},
		{EventName: "First", EventTime: "2026-04-01T10:00:00Z"},
		{EventName: "Second", EventTime: "2026-04-01T11:00:00Z"},
	}
	SortByTime(events)
	assert.Equal(t, "First", events[0].EventName)
	assert.Equal(t, "Second", events[1].EventName)
	assert.Equal(t, "Third", events[2].EventName)
}

func TestParse_File(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "trail.json")
	data := `{"Records":[{"eventSource":"sns.amazonaws.com","eventName":"CreateTopic","eventTime":"2026-04-01T10:00:00Z","awsRegion":"us-east-1","requestParameters":{"Name":"alerts"}}]}`
	err := os.WriteFile(path, []byte(data), 0644)
	require.NoError(t, err)

	events, err := ParseFile(path)
	require.NoError(t, err)
	assert.Len(t, events, 1)
	assert.Equal(t, "sns", events[0].ServiceName())
}
