package tier1

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/google/uuid"
	"github.com/neureaux/cloudmock/benchmarks/awsclient"
	"github.com/neureaux/cloudmock/benchmarks/harness"
)

type ecsSuite struct{}

func NewECSSuite() harness.Suite { return &ecsSuite{} }
func (s *ecsSuite) Name() string { return "ecs" }
func (s *ecsSuite) Tier() int    { return 1 }

func (s *ecsSuite) Operations() []harness.Operation {
	clusterName := fmt.Sprintf("bench-%s", uuid.New().String()[:8])

	newClient := func(endpoint string) (*ecs.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return ecs.NewFromConfig(cfg, func(o *ecs.Options) {
			o.BaseEndpoint = awsclient.Endpoint(endpoint)
		}), nil
	}

	createCluster := func(ctx context.Context, endpoint string) error {
		client, err := newClient(endpoint)
		if err != nil {
			return err
		}
		_, err = client.CreateCluster(ctx, &ecs.CreateClusterInput{
			ClusterName: aws.String(clusterName),
		})
		return err
	}

	deleteCluster := func(ctx context.Context, endpoint string) error {
		client, err := newClient(endpoint)
		if err != nil {
			return err
		}
		_, err = client.DeleteCluster(ctx, &ecs.DeleteClusterInput{
			Cluster: aws.String(clusterName),
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
				out, err := client.CreateCluster(ctx, &ecs.CreateClusterInput{
					ClusterName: aws.String(clusterName),
				})
				if err != nil {
					return nil, err
				}
				client.DeleteCluster(ctx, &ecs.DeleteClusterInput{Cluster: aws.String(clusterName)})
				return out, nil
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "CreateClusterOutput")}
			},
		},
		{
			Name: "DescribeClusters",
			Setup: func(ctx context.Context, endpoint string) error {
				return createCluster(ctx, endpoint)
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.DescribeClusters(ctx, &ecs.DescribeClustersInput{
					Clusters: []string{clusterName},
				})
			},
			Teardown: func(ctx context.Context, endpoint string) error {
				return deleteCluster(ctx, endpoint)
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "DescribeClustersOutput")}
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
				return client.ListClusters(ctx, &ecs.ListClustersInput{})
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
				return client.DeleteCluster(ctx, &ecs.DeleteClusterInput{
					Cluster: aws.String(clusterName),
				})
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "DeleteClusterOutput")}
			},
		},
	}
}
