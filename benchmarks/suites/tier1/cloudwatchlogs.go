package tier1

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	cwlogstypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/google/uuid"
	"github.com/neureaux/cloudmock/benchmarks/awsclient"
	"github.com/neureaux/cloudmock/benchmarks/harness"
)

type cloudWatchLogsSuite struct{}

func NewCloudWatchLogsSuite() harness.Suite { return &cloudWatchLogsSuite{} }
func (s *cloudWatchLogsSuite) Name() string { return "cloudwatchlogs" }
func (s *cloudWatchLogsSuite) Tier() int    { return 1 }

func (s *cloudWatchLogsSuite) Operations() []harness.Operation {
	logGroupName := fmt.Sprintf("/bench/%s", uuid.New().String()[:8])
	logStreamName := "bench-stream"

	newClient := func(endpoint string) (*cloudwatchlogs.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return cloudwatchlogs.NewFromConfig(cfg, func(o *cloudwatchlogs.Options) {
			o.BaseEndpoint = awsclient.Endpoint(endpoint)
		}), nil
	}

	createLogGroup := func(ctx context.Context, endpoint string) error {
		client, err := newClient(endpoint)
		if err != nil {
			return err
		}
		_, err = client.CreateLogGroup(ctx, &cloudwatchlogs.CreateLogGroupInput{
			LogGroupName: aws.String(logGroupName),
		})
		return err
	}

	deleteLogGroup := func(ctx context.Context, endpoint string) error {
		client, err := newClient(endpoint)
		if err != nil {
			return err
		}
		_, err = client.DeleteLogGroup(ctx, &cloudwatchlogs.DeleteLogGroupInput{
			LogGroupName: aws.String(logGroupName),
		})
		return err
	}

	return []harness.Operation{
		{
			Name: "CreateLogGroup",
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				out, err := client.CreateLogGroup(ctx, &cloudwatchlogs.CreateLogGroupInput{
					LogGroupName: aws.String(logGroupName),
				})
				if err != nil {
					return nil, err
				}
				client.DeleteLogGroup(ctx, &cloudwatchlogs.DeleteLogGroupInput{LogGroupName: aws.String(logGroupName)})
				return out, nil
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "CreateLogGroupOutput")}
			},
		},
		{
			Name: "DescribeLogGroups",
			Setup: func(ctx context.Context, endpoint string) error {
				return createLogGroup(ctx, endpoint)
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.DescribeLogGroups(ctx, &cloudwatchlogs.DescribeLogGroupsInput{
					LogGroupNamePrefix: aws.String("/bench/"),
				})
			},
			Teardown: func(ctx context.Context, endpoint string) error {
				return deleteLogGroup(ctx, endpoint)
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "DescribeLogGroupsOutput")}
			},
		},
		{
			Name: "PutLogEvents",
			Setup: func(ctx context.Context, endpoint string) error {
				if err := createLogGroup(ctx, endpoint); err != nil {
					return err
				}
				client, err := newClient(endpoint)
				if err != nil {
					return err
				}
				_, err = client.CreateLogStream(ctx, &cloudwatchlogs.CreateLogStreamInput{
					LogGroupName:  aws.String(logGroupName),
					LogStreamName: aws.String(logStreamName),
				})
				return err
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.PutLogEvents(ctx, &cloudwatchlogs.PutLogEventsInput{
					LogGroupName:  aws.String(logGroupName),
					LogStreamName: aws.String(logStreamName),
					LogEvents: []cwlogstypes.InputLogEvent{
						{
							Message:   aws.String("benchmark log event"),
							Timestamp: aws.Int64(time.Now().UnixMilli()),
						},
					},
				})
			},
			Teardown: func(ctx context.Context, endpoint string) error {
				return deleteLogGroup(ctx, endpoint)
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "PutLogEventsOutput")}
			},
		},
		{
			Name: "DeleteLogGroup",
			Setup: func(ctx context.Context, endpoint string) error {
				return createLogGroup(ctx, endpoint)
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.DeleteLogGroup(ctx, &cloudwatchlogs.DeleteLogGroupInput{
					LogGroupName: aws.String(logGroupName),
				})
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "DeleteLogGroupOutput")}
			},
		},
	}
}
