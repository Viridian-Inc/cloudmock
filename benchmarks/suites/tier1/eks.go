package tier1

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	ekstypes "github.com/aws/aws-sdk-go-v2/service/eks/types"
	"github.com/google/uuid"
	"github.com/Viridian-Inc/cloudmock/benchmarks/awsclient"
	"github.com/Viridian-Inc/cloudmock/benchmarks/harness"
)

type eksSuite struct{}

func NewEKSSuite() harness.Suite { return &eksSuite{} }
func (s *eksSuite) Name() string { return "eks" }
func (s *eksSuite) Tier() int    { return 1 }

func (s *eksSuite) Operations() []harness.Operation {
	clusterName := fmt.Sprintf("bench-%s", uuid.New().String()[:8])
	roleArn := "arn:aws:iam::000000000000:role/bench-eks-role"

	newClient := func(endpoint string) (*eks.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return eks.NewFromConfig(cfg, func(o *eks.Options) {
			o.BaseEndpoint = awsclient.Endpoint(endpoint)
		}), nil
	}

	createCluster := func(ctx context.Context, endpoint string) error {
		client, err := newClient(endpoint)
		if err != nil {
			return err
		}
		_, err = client.CreateCluster(ctx, &eks.CreateClusterInput{
			Name:    aws.String(clusterName),
			RoleArn: aws.String(roleArn),
			ResourcesVpcConfig: &ekstypes.VpcConfigRequest{
				SubnetIds: []string{"subnet-1"},
			},
		})
		return err
	}

	deleteCluster := func(ctx context.Context, endpoint string) error {
		client, err := newClient(endpoint)
		if err != nil {
			return err
		}
		_, err = client.DeleteCluster(ctx, &eks.DeleteClusterInput{
			Name: aws.String(clusterName),
		})
		return err
	}

	return []harness.Operation{
		{
			Name: "CreateCluster",
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				out, err := client.CreateCluster(ctx, &eks.CreateClusterInput{
					Name:    aws.String(clusterName),
					RoleArn: aws.String(roleArn),
					ResourcesVpcConfig: &ekstypes.VpcConfigRequest{
						SubnetIds: []string{"subnet-1"},
					},
				})
				if err != nil {
					return nil, err
				}
				client.DeleteCluster(ctx, &eks.DeleteClusterInput{Name: aws.String(clusterName)})
				return out, nil
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "CreateClusterOutput")}
			},
		},
		{
			Name: "DescribeCluster",
			Setup: func(ctx context.Context, endpoint string) error {
				return createCluster(ctx, endpoint)
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.DescribeCluster(ctx, &eks.DescribeClusterInput{
					Name: aws.String(clusterName),
				})
			},
			Teardown: func(ctx context.Context, endpoint string) error {
				return deleteCluster(ctx, endpoint)
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "DescribeClusterOutput")}
			},
		},
		{
			Name: "ListClusters",
			Setup: func(ctx context.Context, endpoint string) error {
				return createCluster(ctx, endpoint)
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.ListClusters(ctx, &eks.ListClustersInput{})
			},
			Teardown: func(ctx context.Context, endpoint string) error {
				return deleteCluster(ctx, endpoint)
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "ListClustersOutput")}
			},
		},
		{
			Name: "DeleteCluster",
			Setup: func(ctx context.Context, endpoint string) error {
				return createCluster(ctx, endpoint)
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.DeleteCluster(ctx, &eks.DeleteClusterInput{
					Name: aws.String(clusterName),
				})
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "DeleteClusterOutput")}
			},
		},
	}
}
