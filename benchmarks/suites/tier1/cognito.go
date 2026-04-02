package tier1

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/google/uuid"
	"github.com/neureaux/cloudmock/benchmarks/awsclient"
	"github.com/neureaux/cloudmock/benchmarks/harness"
)

type cognitoSuite struct{}

func NewCognitoSuite() harness.Suite { return &cognitoSuite{} }
func (s *cognitoSuite) Name() string { return "cognito" }
func (s *cognitoSuite) Tier() int    { return 1 }

func (s *cognitoSuite) Operations() []harness.Operation {
	poolName := fmt.Sprintf("bench-%s", uuid.New().String()[:8])

	newClient := func(endpoint string) (*cognitoidentityprovider.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return cognitoidentityprovider.NewFromConfig(cfg, func(o *cognitoidentityprovider.Options) {
			o.BaseEndpoint = awsclient.Endpoint(endpoint)
		}), nil
	}

	createUserPool := func(ctx context.Context, endpoint string) (string, error) {
		client, err := newClient(endpoint)
		if err != nil {
			return "", err
		}
		out, err := client.CreateUserPool(ctx, &cognitoidentityprovider.CreateUserPoolInput{
			PoolName: aws.String(poolName),
		})
		if err != nil {
			return "", err
		}
		return aws.ToString(out.UserPool.Id), nil
	}

	return []harness.Operation{
		{
			Name: "CreateUserPool",
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				out, err := client.CreateUserPool(ctx, &cognitoidentityprovider.CreateUserPoolInput{
					PoolName: aws.String(poolName),
				})
				if err != nil {
					return nil, err
				}
				// clean up
				client.DeleteUserPool(ctx, &cognitoidentityprovider.DeleteUserPoolInput{
					UserPoolId: out.UserPool.Id,
				})
				return out, nil
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "CreateUserPoolOutput")}
			},
		},
		{
			Name: "DescribeUserPool",
			Setup: func(ctx context.Context, endpoint string) error {
				_, err := createUserPool(ctx, endpoint)
				return err
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				poolID, err := createUserPool(ctx, endpoint)
				if err != nil {
					return nil, err
				}
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.DescribeUserPool(ctx, &cognitoidentityprovider.DescribeUserPoolInput{
					UserPoolId: aws.String(poolID),
				})
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "DescribeUserPoolOutput")}
			},
		},
		{
			Name: "ListUserPools",
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				maxResults := int32(10)
				return client.ListUserPools(ctx, &cognitoidentityprovider.ListUserPoolsInput{
					MaxResults: &maxResults,
				})
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "ListUserPoolsOutput")}
			},
		},
		{
			Name: "DeleteUserPool",
			Setup: func(ctx context.Context, endpoint string) error {
				_, err := createUserPool(ctx, endpoint)
				return err
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				poolID, err := createUserPool(ctx, endpoint)
				if err != nil {
					return nil, err
				}
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.DeleteUserPool(ctx, &cognitoidentityprovider.DeleteUserPoolInput{
					UserPoolId: aws.String(poolID),
				})
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "DeleteUserPoolOutput")}
			},
		},
	}
}
