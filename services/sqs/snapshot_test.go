package sqs_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/config"
	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	sqssvc "github.com/Viridian-Inc/cloudmock/services/sqs"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
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

type snapshotQueue struct {
	Name       string            `json:"name"`
	Attributes map[string]string `json:"attributes"`
}

type snapshotQueues struct {
	Queues []snapshotQueue `json:"queues"`
}

func newSnapshotSQSClient(t *testing.T) (*sqssvc.SQSService, *sqs.Client) {
	t.Helper()
	cfg := config.Default()
	cfg.AccountID, cfg.Region, cfg.IAM.Mode = sqsTestAccount, sqsTestRegion, "none"
	svc := sqssvc.New(cfg.AccountID, cfg.Region)
	registry := routing.NewRegistry()
	registry.Register(svc)
	server := httptest.NewServer(gateway.New(cfg, registry))
	t.Cleanup(server.Close)
	httpClient := server.Client()
	httpClient.Timeout = 5 * time.Second
	client := sqs.NewFromConfig(aws.Config{
		Region: sqsTestRegion, HTTPClient: httpClient, RetryMaxAttempts: 1,
		Credentials: credentials.NewStaticCredentialsProvider("test", "test", ""),
	}, func(options *sqs.Options) {
		options.BaseEndpoint = aws.String(server.URL)
	})
	t.Cleanup(func() {
		for _, queueURL := range svc.GetQueueNames() {
			if _, err := client.DeleteQueue(context.Background(), &sqs.DeleteQueueInput{QueueUrl: aws.String(queueURL)}); err != nil {
				t.Errorf("cleanup queue: %v", err)
			}
		}
	})
	return svc, client
}

func snapshotDLQFixture(t *testing.T, fifo bool) (source, dlq snapshotQueue) {
	t.Helper()
	suffix := ""
	if fifo {
		suffix = ".fifo"
	}
	source = snapshotQueue{Name: "source" + suffix, Attributes: map[string]string{"VisibilityTimeout": "45", "MessageRetentionPeriod": "3600"}}
	dlq = snapshotQueue{Name: "dead" + suffix, Attributes: map[string]string{"VisibilityTimeout": "60", "MessageRetentionPeriod": "7200"}}
	policy, err := json.Marshal(map[string]string{
		"deadLetterTargetArn": fmt.Sprintf("arn:aws:sqs:%s:%s:%s", sqsTestRegion, sqsTestAccount, dlq.Name),
		"maxReceiveCount":     "1",
	})
	if err != nil {
		t.Fatal(err)
	}
	source.Attributes["RedrivePolicy"] = string(policy)
	if fifo {
		for _, queue := range []snapshotQueue{source, dlq} {
			queue.Attributes["FifoQueue"] = "true"
			queue.Attributes["ContentBasedDeduplication"] = "false"
		}
	}
	return source, dlq
}

func snapshotQueueBytes(t *testing.T, queues ...snapshotQueue) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(snapshotQueues{Queues: queues})
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func snapshotQueueURL(t *testing.T, client *sqs.Client, name string) string {
	t.Helper()
	output, err := client.GetQueueUrl(context.Background(), &sqs.GetQueueUrlInput{QueueName: aws.String(name)})
	if err != nil {
		t.Fatalf("GetQueueUrl %s: %v", name, err)
	}
	return aws.ToString(output.QueueUrl)
}

func snapshotSend(t *testing.T, client *sqs.Client, queueURL, body string, fifo bool) {
	t.Helper()
	input := &sqs.SendMessageInput{QueueUrl: aws.String(queueURL), MessageBody: aws.String(body)}
	if fifo {
		input.MessageGroupId, input.MessageDeduplicationId = aws.String("snapshot-group"), aws.String(body)
	}
	if _, err := client.SendMessage(context.Background(), input); err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
}

func snapshotReceive(t *testing.T, client *sqs.Client, queueURL string) []types.Message {
	t.Helper()
	output, err := client.ReceiveMessage(context.Background(), &sqs.ReceiveMessageInput{
		QueueUrl: aws.String(queueURL), MaxNumberOfMessages: 1, VisibilityTimeout: 30,
	})
	if err != nil {
		t.Fatalf("ReceiveMessage: %v", err)
	}
	return output.Messages
}

func snapshotConsume(t *testing.T, client *sqs.Client, queueURL, body string) {
	t.Helper()
	messages := snapshotReceive(t, client, queueURL)
	if len(messages) != 1 || aws.ToString(messages[0].Body) != body {
		t.Fatalf("queue %s: want message %q, got %v", queueURL, body, messages)
	}
	if _, err := client.DeleteMessage(context.Background(), &sqs.DeleteMessageInput{
		QueueUrl: aws.String(queueURL), ReceiptHandle: messages[0].ReceiptHandle,
	}); err != nil {
		t.Fatalf("DeleteMessage: %v", err)
	}
}

func assertSnapshotRedrive(t *testing.T, client *sqs.Client, sourceURL, dlqURL, body string) {
	t.Helper()
	messages := snapshotReceive(t, client, sourceURL)
	if len(messages) != 1 || aws.ToString(messages[0].Body) != body {
		t.Fatalf("initial source receive: want %q, got %v", body, messages)
	}
	// Release the first receipt explicitly so the second receive checks the
	// redrive threshold without sleeps or background timer scheduling.
	if _, err := client.ChangeMessageVisibility(context.Background(), &sqs.ChangeMessageVisibilityInput{
		QueueUrl: aws.String(sourceURL), ReceiptHandle: messages[0].ReceiptHandle, VisibilityTimeout: 0,
	}); err != nil {
		t.Fatalf("ChangeMessageVisibility: %v", err)
	}
	if messages := snapshotReceive(t, client, sourceURL); len(messages) != 0 {
		t.Errorf("source returned %d messages after maxReceiveCount=1; want delivery to DLQ", len(messages))
	}
	snapshotConsume(t, client, dlqURL, body)
	if messages := snapshotReceive(t, client, dlqURL); len(messages) != 0 {
		t.Errorf("DLQ contains duplicate deliveries: %v", messages)
	}
}

func assertSnapshotAttributes(t *testing.T, client *sqs.Client, queueURL string, expected map[string]string) {
	t.Helper()
	output, err := client.GetQueueAttributes(context.Background(), &sqs.GetQueueAttributesInput{
		QueueUrl: aws.String(queueURL), AttributeNames: []types.QueueAttributeName{types.QueueAttributeNameAll},
	})
	if err != nil {
		t.Fatalf("GetQueueAttributes: %v", err)
	}
	for name, value := range expected {
		if output.Attributes[name] != value {
			t.Errorf("queue attribute %s=%q, want %q", name, output.Attributes[name], value)
		}
	}
}

func TestSQS_ImportState_RestoresDLQInEitherOrder(t *testing.T) {
	for _, fifo := range []bool{false, true} {
		for _, sourceFirst := range []bool{false, true} {
			t.Run(fmt.Sprintf("fifo=%t/source-first=%t", fifo, sourceFirst), func(t *testing.T) {
				source, dlq := snapshotDLQFixture(t, fifo)
				queues := []snapshotQueue{dlq, source}
				if sourceFirst {
					queues = []snapshotQueue{source, dlq}
				}
				svc, client := newSnapshotSQSClient(t)
				if err := svc.ImportState(snapshotQueueBytes(t, queues...)); err != nil {
					t.Fatal(err)
				}
				sourceURL, dlqURL := snapshotQueueURL(t, client, source.Name), snapshotQueueURL(t, client, dlq.Name)
				assertSnapshotAttributes(t, client, sourceURL, source.Attributes)
				assertSnapshotAttributes(t, client, dlqURL, dlq.Attributes)
				snapshotSend(t, client, sourceURL, "restored payload", fifo)
				assertSnapshotRedrive(t, client, sourceURL, dlqURL, "restored payload")
			})
		}
	}
}

func TestSQS_ImportState_RepeatedImportKeepsLiveQueues(t *testing.T) {
	for _, fifo := range []bool{false, true} {
		for _, sourceFirst := range []bool{false, true} {
			t.Run(fmt.Sprintf("fifo=%t/source-first=%t", fifo, sourceFirst), func(t *testing.T) {
				source, dlq := snapshotDLQFixture(t, fifo)
				svc, client := newSnapshotSQSClient(t)
				// An existing source may refer to a DLQ that has not arrived yet.
				if err := svc.ImportState(snapshotQueueBytes(t, source)); err != nil {
					t.Fatal(err)
				}
				sourceURL := snapshotQueueURL(t, client, source.Name)
				snapshotSend(t, client, sourceURL, "pending source payload", fifo)
				liveAttributes := map[string]string{"VisibilityTimeout": "47"}
				if _, err := client.SetQueueAttributes(context.Background(), &sqs.SetQueueAttributesInput{
					QueueUrl: aws.String(sourceURL), Attributes: liveAttributes,
				}); err != nil {
					t.Fatal(err)
				}
				queues := []snapshotQueue{dlq, source}
				if sourceFirst {
					queues = []snapshotQueue{source, dlq}
				}
				data := snapshotQueueBytes(t, queues...)
				if err := svc.ImportState(data); err != nil {
					t.Fatal(err)
				}
				dlqURL := snapshotQueueURL(t, client, dlq.Name)
				snapshotSend(t, client, dlqURL, "pending DLQ payload", fifo)
				for i := 0; i < 2; i++ {
					if err := svc.ImportState(data); err != nil {
						t.Fatalf("repeat import %d: %v", i, err)
					}
				}
				assertSnapshotAttributes(t, client, sourceURL, liveAttributes)
				assertSnapshotAttributes(t, client, sourceURL, map[string]string{"RedrivePolicy": source.Attributes["RedrivePolicy"], "MessageRetentionPeriod": "3600"})
				assertSnapshotAttributes(t, client, dlqURL, dlq.Attributes)
				// Both queue objects must survive: snapshots do not contain messages.
				snapshotConsume(t, client, dlqURL, "pending DLQ payload")
				assertSnapshotRedrive(t, client, sourceURL, dlqURL, "pending source payload")
			})
		}
	}
}

func TestSQS_RoundTrip_DLQDeliveryInEitherOrder(t *testing.T) {
	for _, fifo := range []bool{false, true} {
		t.Run(fmt.Sprintf("fifo=%t", fifo), func(t *testing.T) {
			source, dlq := snapshotDLQFixture(t, fifo)
			original, originalClient := newSnapshotSQSClient(t)
			for _, queue := range []snapshotQueue{dlq, source} {
				if _, err := originalClient.CreateQueue(context.Background(), &sqs.CreateQueueInput{
					QueueName: aws.String(queue.Name), Attributes: queue.Attributes,
				}); err != nil {
					t.Fatalf("CreateQueue %s: %v", queue.Name, err)
				}
			}
			raw, err := original.ExportState()
			if err != nil {
				t.Fatal(err)
			}
			var exported snapshotQueues
			if err := json.Unmarshal(raw, &exported); err != nil {
				t.Fatal(err)
			}
			if len(exported.Queues) != 2 {
				t.Fatalf("exported %d queues, want 2", len(exported.Queues))
			}
			for _, sourceFirst := range []bool{false, true} {
				t.Run(fmt.Sprintf("source-first=%t", sourceFirst), func(t *testing.T) {
					// Force both permutations of the actual export rather than rely
					// on the map iteration order chosen during this test run.
					sort.Slice(exported.Queues, func(i, j int) bool {
						if sourceFirst {
							return exported.Queues[i].Name > exported.Queues[j].Name
						}
						return exported.Queues[i].Name < exported.Queues[j].Name
					})
					restored, client := newSnapshotSQSClient(t)
					if err := restored.ImportState(snapshotQueueBytes(t, exported.Queues...)); err != nil {
						t.Fatal(err)
					}
					sourceURL, dlqURL := snapshotQueueURL(t, client, source.Name), snapshotQueueURL(t, client, dlq.Name)
					assertSnapshotAttributes(t, client, sourceURL, source.Attributes)
					assertSnapshotAttributes(t, client, dlqURL, dlq.Attributes)
					snapshotSend(t, client, sourceURL, "round trip payload", fifo)
					assertSnapshotRedrive(t, client, sourceURL, dlqURL, "round trip payload")
				})
			}
		})
	}
}
