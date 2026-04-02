package tier1

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/google/uuid"
	"github.com/neureaux/cloudmock/benchmarks/awsclient"
	"github.com/neureaux/cloudmock/benchmarks/harness"
)

type snsSuite struct{}

func NewSNSSuite() harness.Suite { return &snsSuite{} }
func (s *snsSuite) Name() string { return "sns" }
func (s *snsSuite) Tier() int    { return 1 }

func (s *snsSuite) Operations() []harness.Operation {
	topicName := fmt.Sprintf("bench-%s", uuid.New().String()[:8])

	newClient := func(endpoint string) (*sns.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return sns.NewFromConfig(cfg, func(o *sns.Options) {
			o.BaseEndpoint = awsclient.Endpoint(endpoint)
		}), nil
	}

	createTopic := func(ctx context.Context, endpoint string) (string, error) {
		client, err := newClient(endpoint)
		if err != nil {
			return "", err
		}
		out, err := client.CreateTopic(ctx, &sns.CreateTopicInput{
			Name: aws.String(topicName),
		})
		if err != nil {
			return "", err
		}
		return aws.ToString(out.TopicArn), nil
	}

	return []harness.Operation{
		{
			Name: "CreateTopic",
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				out, err := client.CreateTopic(ctx, &sns.CreateTopicInput{
					Name: aws.String(topicName),
				})
				if err != nil {
					return nil, err
				}
				// clean up
				client.DeleteTopic(ctx, &sns.DeleteTopicInput{TopicArn: out.TopicArn})
				return out, nil
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "CreateTopicOutput")}
			},
		},
		{
			Name: "Publish",
			Setup: func(ctx context.Context, endpoint string) error {
				_, err := createTopic(ctx, endpoint)
				return err
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				topicArn, err := createTopic(ctx, endpoint)
				if err != nil {
					return nil, err
				}
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.Publish(ctx, &sns.PublishInput{
					TopicArn: aws.String(topicArn),
					Message:  aws.String("benchmark message"),
				})
			},
			Teardown: func(ctx context.Context, endpoint string) error {
				topicArn, err := createTopic(ctx, endpoint)
				if err != nil {
					return nil
				}
				client, err := newClient(endpoint)
				if err != nil {
					return err
				}
				_, err = client.DeleteTopic(ctx, &sns.DeleteTopicInput{TopicArn: aws.String(topicArn)})
				return err
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "PublishOutput")}
			},
		},
		{
			Name: "Subscribe",
			Setup: func(ctx context.Context, endpoint string) error {
				_, err := createTopic(ctx, endpoint)
				return err
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				topicArn, err := createTopic(ctx, endpoint)
				if err != nil {
					return nil, err
				}
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.Subscribe(ctx, &sns.SubscribeInput{
					TopicArn: aws.String(topicArn),
					Protocol: aws.String("email"),
					Endpoint: aws.String("bench@test.com"),
				})
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "SubscribeOutput")}
			},
		},
		{
			Name: "ListTopics",
			Setup: func(ctx context.Context, endpoint string) error {
				_, err := createTopic(ctx, endpoint)
				return err
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.ListTopics(ctx, &sns.ListTopicsInput{})
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "ListTopicsOutput")}
			},
		},
		{
			Name: "DeleteTopic",
			Setup: func(ctx context.Context, endpoint string) error {
				_, err := createTopic(ctx, endpoint)
				return err
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				topicArn, err := createTopic(ctx, endpoint)
				if err != nil {
					return nil, err
				}
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.DeleteTopic(ctx, &sns.DeleteTopicInput{
					TopicArn: aws.String(topicArn),
				})
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "DeleteTopicOutput")}
			},
		},
	}
}
