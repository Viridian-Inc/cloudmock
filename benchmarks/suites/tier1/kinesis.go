package tier1

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/google/uuid"
	"github.com/Viridian-Inc/cloudmock/benchmarks/awsclient"
	"github.com/Viridian-Inc/cloudmock/benchmarks/harness"
)

type kinesisSuite struct{}

func NewKinesisSuite() harness.Suite { return &kinesisSuite{} }
func (s *kinesisSuite) Name() string { return "kinesis" }
func (s *kinesisSuite) Tier() int    { return 1 }

func (s *kinesisSuite) Operations() []harness.Operation {
	streamName := fmt.Sprintf("bench-%s", uuid.New().String()[:8])

	newClient := func(endpoint string) (*kinesis.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return kinesis.NewFromConfig(cfg, func(o *kinesis.Options) {
			o.BaseEndpoint = awsclient.Endpoint(endpoint)
		}), nil
	}

	createStream := func(ctx context.Context, endpoint string) error {
		client, err := newClient(endpoint)
		if err != nil {
			return err
		}
		_, err = client.CreateStream(ctx, &kinesis.CreateStreamInput{
			StreamName: aws.String(streamName),
			ShardCount: aws.Int32(1),
		})
		return err
	}

	deleteStream := func(ctx context.Context, endpoint string) error {
		client, err := newClient(endpoint)
		if err != nil {
			return err
		}
		_, err = client.DeleteStream(ctx, &kinesis.DeleteStreamInput{
			StreamName: aws.String(streamName),
		})
		return err
	}

	return []harness.Operation{
		{
			Name: "CreateStream",
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				out, err := client.CreateStream(ctx, &kinesis.CreateStreamInput{
					StreamName: aws.String(streamName),
					ShardCount: aws.Int32(1),
				})
				if err != nil {
					return nil, err
				}
				client.DeleteStream(ctx, &kinesis.DeleteStreamInput{StreamName: aws.String(streamName)})
				return out, nil
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "CreateStreamOutput")}
			},
		},
		{
			Name: "DescribeStream",
			Setup: func(ctx context.Context, endpoint string) error {
				return createStream(ctx, endpoint)
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.DescribeStream(ctx, &kinesis.DescribeStreamInput{
					StreamName: aws.String(streamName),
				})
			},
			Teardown: func(ctx context.Context, endpoint string) error {
				return deleteStream(ctx, endpoint)
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "DescribeStreamOutput")}
			},
		},
		{
			Name: "PutRecord",
			Setup: func(ctx context.Context, endpoint string) error {
				return createStream(ctx, endpoint)
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.PutRecord(ctx, &kinesis.PutRecordInput{
					StreamName:   aws.String(streamName),
					Data:         []byte("benchmark record data"),
					PartitionKey: aws.String("bench-partition"),
				})
			},
			Teardown: func(ctx context.Context, endpoint string) error {
				return deleteStream(ctx, endpoint)
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "PutRecordOutput")}
			},
		},
		{
			Name: "DeleteStream",
			Setup: func(ctx context.Context, endpoint string) error {
				return createStream(ctx, endpoint)
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.DeleteStream(ctx, &kinesis.DeleteStreamInput{
					StreamName: aws.String(streamName),
				})
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "DeleteStreamOutput")}
			},
		},
	}
}
