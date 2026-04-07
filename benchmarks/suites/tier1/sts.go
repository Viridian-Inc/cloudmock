package tier1

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/Viridian-Inc/cloudmock/benchmarks/awsclient"
	"github.com/Viridian-Inc/cloudmock/benchmarks/harness"
)

type stsSuite struct{}

func NewSTSSuite() harness.Suite { return &stsSuite{} }
func (s *stsSuite) Name() string { return "sts" }
func (s *stsSuite) Tier() int    { return 1 }

func (s *stsSuite) Operations() []harness.Operation {
	newClient := func(endpoint string) (*sts.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return sts.NewFromConfig(cfg, func(o *sts.Options) {
			o.BaseEndpoint = awsclient.Endpoint(endpoint)
		}), nil
	}

	return []harness.Operation{
		{
			Name: "GetCallerIdentity",
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "GetCallerIdentityOutput")}
			},
		},
		{
			Name: "AssumeRole",
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.AssumeRole(ctx, &sts.AssumeRoleInput{
					RoleArn:         aws.String("arn:aws:iam::000000000000:role/bench-role"),
					RoleSessionName: aws.String("bench-session"),
				})
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "AssumeRoleOutput")}
			},
		},
	}
}
