package tier1

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/google/uuid"
	"github.com/Viridian-Inc/cloudmock/benchmarks/awsclient"
	"github.com/Viridian-Inc/cloudmock/benchmarks/harness"
)

type sqsSuite struct{}

func NewSQSSuite() harness.Suite { return &sqsSuite{} }
func (s *sqsSuite) Name() string { return "sqs" }
func (s *sqsSuite) Tier() int    { return 1 }

func newSQSClient(endpoint string) (*sqs.Client, error) {
	cfg, err := awsclient.NewConfig(endpoint)
	if err != nil {
		return nil, err
	}
	client := sqs.NewFromConfig(cfg, func(o *sqs.Options) {
		o.BaseEndpoint = awsclient.Endpoint(endpoint)
	})
	return client, nil
}

func createQueue(ctx context.Context, endpoint string) (string, error) {
	client, err := newSQSClient(endpoint)
	if err != nil {
		return "", err
	}
	name := fmt.Sprintf("bench-%s", uuid.New().String()[:8])
	out, err := client.CreateQueue(ctx, &sqs.CreateQueueInput{
		QueueName: aws.String(name),
	})
	if err != nil {
		return "", err
	}
	return aws.ToString(out.QueueUrl), nil
}

func deleteQueue(ctx context.Context, endpoint, queueURL string) error {
	if queueURL == "" {
		return nil
	}
	client, err := newSQSClient(endpoint)
	if err != nil {
		return err
	}
	_, err = client.DeleteQueue(ctx, &sqs.DeleteQueueInput{
		QueueUrl: aws.String(queueURL),
	})
	return err
}

func (s *sqsSuite) Operations() []harness.Operation {
	return []harness.Operation{
		createQueueOp(),
		sendMessageOp(),
		receiveMessageOp(),
		deleteMessageOp(),
		deleteQueueOp(),
	}
}

func createQueueOp() harness.Operation {
	var queueURL string
	return harness.Operation{
		Name: "CreateQueue",
		Run: func(ctx context.Context, endpoint string) (any, error) {
			client, err := newSQSClient(endpoint)
			if err != nil {
				return nil, err
			}
			name := fmt.Sprintf("bench-%s", uuid.New().String()[:8])
			out, err := client.CreateQueue(ctx, &sqs.CreateQueueInput{
				QueueName: aws.String(name),
			})
			if err != nil {
				return nil, err
			}
			queueURL = aws.ToString(out.QueueUrl)
			return out, nil
		},
		Validate: func(resp any) []harness.Finding {
			out, ok := resp.(*sqs.CreateQueueOutput)
			if !ok || out == nil {
				return []harness.Finding{harness.CheckNotNil(nil, "CreateQueueOutput")}
			}
			findings := []harness.Finding{harness.CheckNotNil(out.QueueUrl, "QueueUrl")}
			return findings
		},
		Teardown: func(ctx context.Context, endpoint string) error {
			return deleteQueue(ctx, endpoint, queueURL)
		},
	}
}

func sendMessageOp() harness.Operation {
	var queueURL string
	return harness.Operation{
		Name: "SendMessage",
		Setup: func(ctx context.Context, endpoint string) error {
			url, err := createQueue(ctx, endpoint)
			if err != nil {
				return err
			}
			queueURL = url
			return nil
		},
		Run: func(ctx context.Context, endpoint string) (any, error) {
			client, err := newSQSClient(endpoint)
			if err != nil {
				return nil, err
			}
			return client.SendMessage(ctx, &sqs.SendMessageInput{
				QueueUrl:    aws.String(queueURL),
				MessageBody: aws.String("benchmark-message"),
			})
		},
		Validate: func(resp any) []harness.Finding {
			out, ok := resp.(*sqs.SendMessageOutput)
			if !ok || out == nil {
				return []harness.Finding{harness.CheckNotNil(nil, "SendMessageOutput")}
			}
			return []harness.Finding{harness.CheckNotNil(out.MessageId, "MessageId")}
		},
		Teardown: func(ctx context.Context, endpoint string) error {
			return deleteQueue(ctx, endpoint, queueURL)
		},
	}
}

func receiveMessageOp() harness.Operation {
	var queueURL string
	return harness.Operation{
		Name: "ReceiveMessage",
		Setup: func(ctx context.Context, endpoint string) error {
			url, err := createQueue(ctx, endpoint)
			if err != nil {
				return err
			}
			queueURL = url
			client, err := newSQSClient(endpoint)
			if err != nil {
				return err
			}
			_, err = client.SendMessage(ctx, &sqs.SendMessageInput{
				QueueUrl:    aws.String(queueURL),
				MessageBody: aws.String("benchmark-message"),
			})
			return err
		},
		Run: func(ctx context.Context, endpoint string) (any, error) {
			client, err := newSQSClient(endpoint)
			if err != nil {
				return nil, err
			}
			return client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
				QueueUrl:            aws.String(queueURL),
				MaxNumberOfMessages: 1,
				WaitTimeSeconds:     1,
			})
		},
		Validate: func(resp any) []harness.Finding {
			out, ok := resp.(*sqs.ReceiveMessageOutput)
			if !ok || out == nil {
				return []harness.Finding{harness.CheckNotNil(nil, "ReceiveMessageOutput")}
			}
			f := harness.Finding{Field: "Messages", Expected: ">=1"}
			if len(out.Messages) >= 1 {
				f.Actual = fmt.Sprintf("%d", len(out.Messages))
				f.Grade = harness.GradePass
			} else {
				f.Actual = "0"
				f.Grade = harness.GradeFail
			}
			return []harness.Finding{f}
		},
		Teardown: func(ctx context.Context, endpoint string) error {
			return deleteQueue(ctx, endpoint, queueURL)
		},
	}
}

func deleteMessageOp() harness.Operation {
	var queueURL string
	var receiptHandle string
	return harness.Operation{
		Name: "DeleteMessage",
		Setup: func(ctx context.Context, endpoint string) error {
			url, err := createQueue(ctx, endpoint)
			if err != nil {
				return err
			}
			queueURL = url
			client, err := newSQSClient(endpoint)
			if err != nil {
				return err
			}
			_, err = client.SendMessage(ctx, &sqs.SendMessageInput{
				QueueUrl:    aws.String(queueURL),
				MessageBody: aws.String("benchmark-message"),
			})
			if err != nil {
				return err
			}
			recv, err := client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
				QueueUrl:            aws.String(queueURL),
				MaxNumberOfMessages: 1,
				WaitTimeSeconds:     1,
			})
			if err != nil {
				return err
			}
			if len(recv.Messages) > 0 {
				receiptHandle = aws.ToString(recv.Messages[0].ReceiptHandle)
			}
			return nil
		},
		Run: func(ctx context.Context, endpoint string) (any, error) {
			client, err := newSQSClient(endpoint)
			if err != nil {
				return nil, err
			}
			return client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
				QueueUrl:      aws.String(queueURL),
				ReceiptHandle: aws.String(receiptHandle),
			})
		},
		Validate: func(resp any) []harness.Finding {
			return []harness.Finding{harness.CheckNotNil(resp, "DeleteMessageOutput")}
		},
		Teardown: func(ctx context.Context, endpoint string) error {
			return deleteQueue(ctx, endpoint, queueURL)
		},
	}
}

func deleteQueueOp() harness.Operation {
	var queueURL string
	return harness.Operation{
		Name: "DeleteQueue",
		Setup: func(ctx context.Context, endpoint string) error {
			url, err := createQueue(ctx, endpoint)
			if err != nil {
				return err
			}
			queueURL = url
			return nil
		},
		Run: func(ctx context.Context, endpoint string) (any, error) {
			client, err := newSQSClient(endpoint)
			if err != nil {
				return nil, err
			}
			out, err := client.DeleteQueue(ctx, &sqs.DeleteQueueInput{
				QueueUrl: aws.String(queueURL),
			})
			if err == nil {
				queueURL = "" // already deleted, skip teardown
			}
			return out, err
		},
		Validate: func(resp any) []harness.Finding {
			return []harness.Finding{harness.CheckNotNil(resp, "DeleteQueueOutput")}
		},
		Teardown: func(ctx context.Context, endpoint string) error {
			return deleteQueue(ctx, endpoint, queueURL)
		},
	}
}
