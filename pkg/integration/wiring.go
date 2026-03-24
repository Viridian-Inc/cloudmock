// Package integration wires cross-service integrations within cloudmock.
package integration

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/neureaux/cloudmock/pkg/eventbus"
	"github.com/neureaux/cloudmock/pkg/routing"
)

// SQSEnqueuer is implemented by the SQS service for direct message delivery.
type SQSEnqueuer interface {
	EnqueueDirect(queueName, messageBody string) bool
}

// WireIntegrations subscribes to event bus events and routes them to target
// services via the registry. This wires up:
//   - S3 events → SQS (based on event bus subscriptions)
func WireIntegrations(bus *eventbus.Bus, registry *routing.Registry, accountID, region string) {
	// Subscribe to all S3 events and route to SQS queues.
	// In a real implementation, this would use S3 bucket notification
	// configuration to determine which queues to deliver to. For now,
	// we provide a simple mechanism: any SQS queue can receive S3 events
	// by being configured through the event bus subscription API.
	bus.Subscribe(&eventbus.Subscription{
		Source: "s3",
		Types:  []string{"s3:*"},
		Handler: func(event *eventbus.Event) error {
			return handleS3Event(registry, event, accountID, region)
		},
	})
}

// handleS3Event formats an S3 event notification in the AWS format and
// delivers it to any configured SQS queue.
func handleS3Event(registry *routing.Registry, event *eventbus.Event, accountID, region string) error {
	svc, err := registry.Lookup("sqs")
	if err != nil {
		return nil // SQS not registered, skip
	}

	enqueuer, ok := svc.(SQSEnqueuer)
	if !ok {
		return nil
	}

	bucket, _ := event.Detail["bucket"].(string)
	key, _ := event.Detail["key"].(string)
	size, _ := event.Detail["size"].(int64)
	etag, _ := event.Detail["etag"].(string)

	// Build AWS S3 event notification JSON.
	s3Event := buildS3EventNotification(event.Type, bucket, key, size, etag, accountID, region, event.Time)

	// Deliver to SQS queues that are configured for this bucket.
	// Convention: queue name matching "s3-events-{bucket}" or configured
	// via bucket notification config. For now, we use a naming convention.
	queueName := fmt.Sprintf("s3-events-%s", bucket)
	enqueuer.EnqueueDirect(queueName, s3Event)

	return nil
}

// buildS3EventNotification creates an AWS-compatible S3 event notification JSON.
func buildS3EventNotification(eventName, bucket, key string, size int64, etag, accountID, region string, eventTime time.Time) string {
	// Map internal event type to AWS event name.
	awsEventName := strings.Replace(eventName, ":", ":", -1) // already correct format

	record := map[string]any{
		"eventVersion": "2.1",
		"eventSource":  "aws:s3",
		"awsRegion":    region,
		"eventTime":    eventTime.Format(time.RFC3339),
		"eventName":    awsEventName,
		"s3": map[string]any{
			"s3SchemaVersion": "1.0",
			"bucket": map[string]any{
				"name":          bucket,
				"ownerIdentity": map[string]string{"principalId": accountID},
				"arn":           fmt.Sprintf("arn:aws:s3:::%s", bucket),
			},
			"object": map[string]any{
				"key":  key,
				"size": size,
				"eTag": etag,
			},
		},
	}

	envelope := map[string]any{
		"Records": []any{record},
	}

	data, _ := json.Marshal(envelope)
	return string(data)
}
