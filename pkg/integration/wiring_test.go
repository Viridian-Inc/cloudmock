package integration_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/neureaux/cloudmock/pkg/eventbus"
	"github.com/neureaux/cloudmock/pkg/integration"
	"github.com/neureaux/cloudmock/pkg/routing"
	"github.com/neureaux/cloudmock/pkg/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSQS implements both service.Service and integration.SQSEnqueuer.
type mockSQS struct {
	messages map[string][]string // queueName → messages
}

func newMockSQS() *mockSQS {
	return &mockSQS{messages: make(map[string][]string)}
}

func (m *mockSQS) Name() string                                                    { return "sqs" }
func (m *mockSQS) Actions() []service.Action                                       { return nil }
func (m *mockSQS) HandleRequest(ctx *service.RequestContext) (*service.Response, error) { return nil, nil }
func (m *mockSQS) HealthCheck() error                                              { return nil }
func (m *mockSQS) EnqueueDirect(queueName, messageBody string) bool {
	m.messages[queueName] = append(m.messages[queueName], messageBody)
	return true
}

func TestWireIntegrations_S3EventToSQS(t *testing.T) {
	bus := eventbus.NewBus()
	reg := routing.NewRegistry()
	sqs := newMockSQS()
	reg.Register(sqs)

	integration.WireIntegrations(bus, reg, "000000000000", "us-east-1")

	// Publish an S3 ObjectCreated event
	bus.PublishSync(&eventbus.Event{
		Source:    "s3",
		Type:      "s3:ObjectCreated:Put",
		Detail:    map[string]interface{}{"bucket": "test-bucket", "key": "file.txt", "size": int64(1024), "etag": "abc123"},
		Time:      time.Now(),
		Region:    "us-east-1",
		AccountID: "000000000000",
	})

	// Verify SQS received the message in the convention-named queue
	msgs := sqs.messages["s3-events-test-bucket"]
	require.Len(t, msgs, 1)

	// Verify the message is valid AWS S3 event notification JSON
	var envelope map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(msgs[0]), &envelope))

	records, ok := envelope["Records"].([]interface{})
	require.True(t, ok)
	require.Len(t, records, 1)

	record := records[0].(map[string]interface{})
	assert.Equal(t, "aws:s3", record["eventSource"])
	assert.Equal(t, "us-east-1", record["awsRegion"])
	assert.Equal(t, "s3:ObjectCreated:Put", record["eventName"])

	s3Detail := record["s3"].(map[string]interface{})
	bucket := s3Detail["bucket"].(map[string]interface{})
	assert.Equal(t, "test-bucket", bucket["name"])

	obj := s3Detail["object"].(map[string]interface{})
	assert.Equal(t, "file.txt", obj["key"])
	assert.Equal(t, float64(1024), obj["size"])
}

func TestWireIntegrations_S3DeleteEvent(t *testing.T) {
	bus := eventbus.NewBus()
	reg := routing.NewRegistry()
	sqs := newMockSQS()
	reg.Register(sqs)

	integration.WireIntegrations(bus, reg, "000000000000", "us-east-1")

	bus.PublishSync(&eventbus.Event{
		Source: "s3",
		Type:   "s3:ObjectRemoved:Delete",
		Detail: map[string]interface{}{"bucket": "my-bucket", "key": "deleted.txt", "size": int64(0), "etag": ""},
		Time:   time.Now(),
	})

	msgs := sqs.messages["s3-events-my-bucket"]
	require.Len(t, msgs, 1)

	var envelope map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(msgs[0]), &envelope))
	records := envelope["Records"].([]interface{})
	record := records[0].(map[string]interface{})
	assert.Equal(t, "s3:ObjectRemoved:Delete", record["eventName"])
}

func TestWireIntegrations_NoSQSRegistered(t *testing.T) {
	bus := eventbus.NewBus()
	reg := routing.NewRegistry() // no SQS registered

	integration.WireIntegrations(bus, reg, "000000000000", "us-east-1")

	// Should not panic when SQS is not available
	bus.PublishSync(&eventbus.Event{
		Source: "s3",
		Type:   "s3:ObjectCreated:Put",
		Detail: map[string]interface{}{"bucket": "test", "key": "k"},
		Time:   time.Now(),
	})
}

func TestWireIntegrations_NonS3EventIgnored(t *testing.T) {
	bus := eventbus.NewBus()
	reg := routing.NewRegistry()
	sqs := newMockSQS()
	reg.Register(sqs)

	integration.WireIntegrations(bus, reg, "000000000000", "us-east-1")

	// Publish a non-S3 event
	bus.PublishSync(&eventbus.Event{
		Source: "dynamodb",
		Type:   "dynamodb:StreamRecord",
		Detail: map[string]interface{}{},
		Time:   time.Now(),
	})

	// SQS should have received nothing
	assert.Empty(t, sqs.messages)
}
