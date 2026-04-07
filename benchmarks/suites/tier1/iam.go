package tier1

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/google/uuid"
	"github.com/Viridian-Inc/cloudmock/benchmarks/awsclient"
	"github.com/Viridian-Inc/cloudmock/benchmarks/harness"
)

type iamSuite struct{}

func NewIAMSuite() harness.Suite { return &iamSuite{} }
func (s *iamSuite) Name() string { return "iam" }
func (s *iamSuite) Tier() int    { return 1 }

const assumeRolePolicyDoc = `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"Service":"lambda.amazonaws.com"},"Action":"sts:AssumeRole"}]}`

func (s *iamSuite) Operations() []harness.Operation {
	userName := fmt.Sprintf("bench-%s", uuid.New().String()[:8])
	roleName := fmt.Sprintf("bench-role-%s", uuid.New().String()[:8])

	newClient := func(endpoint string) (*iam.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return iam.NewFromConfig(cfg, func(o *iam.Options) {
			o.BaseEndpoint = awsclient.Endpoint(endpoint)
		}), nil
	}

	createUser := func(ctx context.Context, endpoint string) error {
		client, err := newClient(endpoint)
		if err != nil {
			return err
		}
		_, err = client.CreateUser(ctx, &iam.CreateUserInput{
			UserName: aws.String(userName),
		})
		return err
	}

	deleteUser := func(ctx context.Context, endpoint string) error {
		client, err := newClient(endpoint)
		if err != nil {
			return err
		}
		_, err = client.DeleteUser(ctx, &iam.DeleteUserInput{
			UserName: aws.String(userName),
		})
		return err
	}

	createRole := func(ctx context.Context, endpoint string) error {
		client, err := newClient(endpoint)
		if err != nil {
			return err
		}
		_, err = client.CreateRole(ctx, &iam.CreateRoleInput{
			RoleName:                 aws.String(roleName),
			AssumeRolePolicyDocument: aws.String(assumeRolePolicyDoc),
		})
		return err
	}

	return []harness.Operation{
		{
			Name: "CreateUser",
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				out, err := client.CreateUser(ctx, &iam.CreateUserInput{
					UserName: aws.String(userName),
				})
				if err != nil {
					return nil, err
				}
				client.DeleteUser(ctx, &iam.DeleteUserInput{UserName: aws.String(userName)})
				return out, nil
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "CreateUserOutput")}
			},
		},
		{
			Name: "GetUser",
			Setup: func(ctx context.Context, endpoint string) error {
				return createUser(ctx, endpoint)
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.GetUser(ctx, &iam.GetUserInput{
					UserName: aws.String(userName),
				})
			},
			Teardown: func(ctx context.Context, endpoint string) error {
				return deleteUser(ctx, endpoint)
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "GetUserOutput")}
			},
		},
		{
			Name: "ListUsers",
			Setup: func(ctx context.Context, endpoint string) error {
				return createUser(ctx, endpoint)
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.ListUsers(ctx, &iam.ListUsersInput{})
			},
			Teardown: func(ctx context.Context, endpoint string) error {
				return deleteUser(ctx, endpoint)
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "ListUsersOutput")}
			},
		},
		{
			Name: "CreateRole",
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				out, err := client.CreateRole(ctx, &iam.CreateRoleInput{
					RoleName:                 aws.String(roleName),
					AssumeRolePolicyDocument: aws.String(assumeRolePolicyDoc),
				})
				if err != nil {
					return nil, err
				}
				client.DeleteRole(ctx, &iam.DeleteRoleInput{RoleName: aws.String(roleName)})
				return out, nil
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "CreateRoleOutput")}
			},
		},
		{
			Name: "DeleteUser",
			Setup: func(ctx context.Context, endpoint string) error {
				return createUser(ctx, endpoint)
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.DeleteUser(ctx, &iam.DeleteUserInput{
					UserName: aws.String(userName),
				})
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "DeleteUserOutput")}
			},
		},
		{
			Name: "DeleteRole",
			Setup: func(ctx context.Context, endpoint string) error {
				return createRole(ctx, endpoint)
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.DeleteRole(ctx, &iam.DeleteRoleInput{
					RoleName: aws.String(roleName),
				})
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "DeleteRoleOutput")}
			},
		},
	}
}
