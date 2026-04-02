package tier1

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/google/uuid"
	"github.com/neureaux/cloudmock/benchmarks/awsclient"
	"github.com/neureaux/cloudmock/benchmarks/harness"
)

type cloudFormationSuite struct{}

func NewCloudFormationSuite() harness.Suite { return &cloudFormationSuite{} }
func (s *cloudFormationSuite) Name() string { return "cloudformation" }
func (s *cloudFormationSuite) Tier() int    { return 1 }

const cfnTemplateBody = `{"AWSTemplateFormatVersion":"2010-09-09","Resources":{"Bucket":{"Type":"AWS::S3::Bucket"}}}`

func (s *cloudFormationSuite) Operations() []harness.Operation {
	stackName := fmt.Sprintf("bench-%s", uuid.New().String()[:8])

	newClient := func(endpoint string) (*cloudformation.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return cloudformation.NewFromConfig(cfg, func(o *cloudformation.Options) {
			o.BaseEndpoint = awsclient.Endpoint(endpoint)
		}), nil
	}

	createStack := func(ctx context.Context, endpoint string) error {
		client, err := newClient(endpoint)
		if err != nil {
			return err
		}
		_, err = client.CreateStack(ctx, &cloudformation.CreateStackInput{
			StackName:    aws.String(stackName),
			TemplateBody: aws.String(cfnTemplateBody),
		})
		return err
	}

	deleteStack := func(ctx context.Context, endpoint string) error {
		client, err := newClient(endpoint)
		if err != nil {
			return err
		}
		_, err = client.DeleteStack(ctx, &cloudformation.DeleteStackInput{
			StackName: aws.String(stackName),
		})
		return err
	}

	return []harness.Operation{
		{
			Name: "CreateStack",
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				out, err := client.CreateStack(ctx, &cloudformation.CreateStackInput{
					StackName:    aws.String(stackName),
					TemplateBody: aws.String(cfnTemplateBody),
				})
				if err != nil {
					return nil, err
				}
				client.DeleteStack(ctx, &cloudformation.DeleteStackInput{StackName: aws.String(stackName)})
				return out, nil
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "CreateStackOutput")}
			},
		},
		{
			Name: "DescribeStacks",
			Setup: func(ctx context.Context, endpoint string) error {
				return createStack(ctx, endpoint)
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{
					StackName: aws.String(stackName),
				})
			},
			Teardown: func(ctx context.Context, endpoint string) error {
				return deleteStack(ctx, endpoint)
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "DescribeStacksOutput")}
			},
		},
		{
			Name: "ListStacks",
			Setup: func(ctx context.Context, endpoint string) error {
				return createStack(ctx, endpoint)
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.ListStacks(ctx, &cloudformation.ListStacksInput{})
			},
			Teardown: func(ctx context.Context, endpoint string) error {
				return deleteStack(ctx, endpoint)
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "ListStacksOutput")}
			},
		},
		{
			Name: "DeleteStack",
			Setup: func(ctx context.Context, endpoint string) error {
				return createStack(ctx, endpoint)
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.DeleteStack(ctx, &cloudformation.DeleteStackInput{
					StackName: aws.String(stackName),
				})
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "DeleteStackOutput")}
			},
		},
	}
}
