package tier1

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/google/uuid"
	"github.com/neureaux/cloudmock/benchmarks/awsclient"
	"github.com/neureaux/cloudmock/benchmarks/harness"
)

type rdsSuite struct{}

func NewRDSSuite() harness.Suite { return &rdsSuite{} }
func (s *rdsSuite) Name() string { return "rds" }
func (s *rdsSuite) Tier() int    { return 1 }

func (s *rdsSuite) Operations() []harness.Operation {
	dbIdentifier := fmt.Sprintf("bench-%s", uuid.New().String()[:8])

	newClient := func(endpoint string) (*rds.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return rds.NewFromConfig(cfg, func(o *rds.Options) {
			o.BaseEndpoint = awsclient.Endpoint(endpoint)
		}), nil
	}

	createDBInstance := func(ctx context.Context, endpoint string) error {
		client, err := newClient(endpoint)
		if err != nil {
			return err
		}
		_, err = client.CreateDBInstance(ctx, &rds.CreateDBInstanceInput{
			DBInstanceIdentifier: aws.String(dbIdentifier),
			Engine:               aws.String("mysql"),
			DBInstanceClass:      aws.String("db.t3.micro"),
			MasterUsername:       aws.String("benchuser"),
			MasterUserPassword:   aws.String("benchpass123"),
			AllocatedStorage:     aws.Int32(20),
		})
		return err
	}

	deleteDBInstance := func(ctx context.Context, endpoint string) error {
		client, err := newClient(endpoint)
		if err != nil {
			return err
		}
		_, err = client.DeleteDBInstance(ctx, &rds.DeleteDBInstanceInput{
			DBInstanceIdentifier: aws.String(dbIdentifier),
			SkipFinalSnapshot:    aws.Bool(true),
		})
		return err
	}

	return []harness.Operation{
		{
			Name: "CreateDBInstance",
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				out, err := client.CreateDBInstance(ctx, &rds.CreateDBInstanceInput{
					DBInstanceIdentifier: aws.String(dbIdentifier),
					Engine:               aws.String("mysql"),
					DBInstanceClass:      aws.String("db.t3.micro"),
					MasterUsername:       aws.String("benchuser"),
					MasterUserPassword:   aws.String("benchpass123"),
					AllocatedStorage:     aws.Int32(20),
				})
				if err != nil {
					return nil, err
				}
				// clean up
				client.DeleteDBInstance(ctx, &rds.DeleteDBInstanceInput{
					DBInstanceIdentifier: aws.String(dbIdentifier),
					SkipFinalSnapshot:    aws.Bool(true),
				})
				return out, nil
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "CreateDBInstanceOutput")}
			},
		},
		{
			Name: "DescribeDBInstances",
			Setup: func(ctx context.Context, endpoint string) error {
				return createDBInstance(ctx, endpoint)
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{
					DBInstanceIdentifier: aws.String(dbIdentifier),
				})
			},
			Teardown: func(ctx context.Context, endpoint string) error {
				return deleteDBInstance(ctx, endpoint)
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "DescribeDBInstancesOutput")}
			},
		},
		{
			Name: "DeleteDBInstance",
			Setup: func(ctx context.Context, endpoint string) error {
				return createDBInstance(ctx, endpoint)
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.DeleteDBInstance(ctx, &rds.DeleteDBInstanceInput{
					DBInstanceIdentifier: aws.String(dbIdentifier),
					SkipFinalSnapshot:    aws.Bool(true),
				})
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "DeleteDBInstanceOutput")}
			},
		},
	}
}
