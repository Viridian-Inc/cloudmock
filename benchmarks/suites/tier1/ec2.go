package tier1

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/Viridian-Inc/cloudmock/benchmarks/awsclient"
	"github.com/Viridian-Inc/cloudmock/benchmarks/harness"
)

type ec2Suite struct{}

func NewEC2Suite() harness.Suite { return &ec2Suite{} }
func (s *ec2Suite) Name() string { return "ec2" }
func (s *ec2Suite) Tier() int    { return 1 }

func (s *ec2Suite) Operations() []harness.Operation {
	newClient := func(endpoint string) (*ec2.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return ec2.NewFromConfig(cfg, func(o *ec2.Options) {
			o.BaseEndpoint = awsclient.Endpoint(endpoint)
		}), nil
	}

	return []harness.Operation{
		{
			Name: "DescribeInstances",
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{})
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "DescribeInstancesOutput")}
			},
		},
		{
			Name: "DescribeVpcs",
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{})
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "DescribeVpcsOutput")}
			},
		},
		{
			Name: "DescribeSubnets",
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{})
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "DescribeSubnetsOutput")}
			},
		},
		{
			Name: "DescribeSecurityGroups",
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{})
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "DescribeSecurityGroupsOutput")}
			},
		},
	}
}
