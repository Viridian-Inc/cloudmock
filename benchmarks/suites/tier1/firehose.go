package tier1

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/firehose"
	firehosetypes "github.com/aws/aws-sdk-go-v2/service/firehose/types"
	"github.com/google/uuid"
	"github.com/neureaux/cloudmock/benchmarks/awsclient"
	"github.com/neureaux/cloudmock/benchmarks/harness"
)

type firehoseSuite struct{}

func NewFirehoseSuite() harness.Suite { return &firehoseSuite{} }
func (s *firehoseSuite) Name() string { return "firehose" }
func (s *firehoseSuite) Tier() int    { return 1 }

func (s *firehoseSuite) Operations() []harness.Operation {
	streamName := fmt.Sprintf("bench-%s", uuid.New().String()[:8])
	roleArn := "arn:aws:iam::000000000000:role/bench-firehose-role"
	bucketArn := "arn:aws:s3:::bench-firehose-bucket"

	newClient := func(endpoint string) (*firehose.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return firehose.NewFromConfig(cfg, func(o *firehose.Options) {
			o.BaseEndpoint = awsclient.Endpoint(endpoint)
		}), nil
	}

	createStream := func(ctx context.Context, endpoint string) error {
		client, err := newClient(endpoint)
		if err != nil {
			return err
		}
		_, err = client.CreateDeliveryStream(ctx, &firehose.CreateDeliveryStreamInput{
			DeliveryStreamName: aws.String(streamName),
			S3DestinationConfiguration: &firehosetypes.S3DestinationConfiguration{
				RoleARN:   aws.String(roleArn),
				BucketARN: aws.String(bucketArn),
			},
		})
		return err
	}

	deleteStream := func(ctx context.Context, endpoint string) error {
		client, err := newClient(endpoint)
		if err != nil {
			return err
		}
		_, err = client.DeleteDeliveryStream(ctx, &firehose.DeleteDeliveryStreamInput{
			DeliveryStreamName: aws.String(streamName),
		})
		return err
	}

	return []harness.Operation{
		{
			Name: "CreateDeliveryStream",
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				out, err := client.CreateDeliveryStream(ctx, &firehose.CreateDeliveryStreamInput{
					DeliveryStreamName: aws.String(streamName),
					S3DestinationConfiguration: &firehosetypes.S3DestinationConfiguration{
						RoleARN:   aws.String(roleArn),
						BucketARN: aws.String(bucketArn),
					},
				})
				if err != nil {
					return nil, err
				}
				client.DeleteDeliveryStream(ctx, &firehose.DeleteDeliveryStreamInput{DeliveryStreamName: aws.String(streamName)})
				return out, nil
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "CreateDeliveryStreamOutput")}
			},
		},
		{
			Name: "DescribeDeliveryStream",
			Setup: func(ctx context.Context, endpoint string) error {
				return createStream(ctx, endpoint)
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.DescribeDeliveryStream(ctx, &firehose.DescribeDeliveryStreamInput{
					DeliveryStreamName: aws.String(streamName),
				})
			},
			Teardown: func(ctx context.Context, endpoint string) error {
				return deleteStream(ctx, endpoint)
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "DescribeDeliveryStreamOutput")}
			},
		},
		{
			Name: "DeleteDeliveryStream",
			Setup: func(ctx context.Context, endpoint string) error {
				return createStream(ctx, endpoint)
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.DeleteDeliveryStream(ctx, &firehose.DeleteDeliveryStreamInput{
					DeliveryStreamName: aws.String(streamName),
				})
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "DeleteDeliveryStreamOutput")}
			},
		},
	}
}
