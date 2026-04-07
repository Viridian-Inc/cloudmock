package tier1

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
	"github.com/google/uuid"
	"github.com/Viridian-Inc/cloudmock/benchmarks/awsclient"
	"github.com/Viridian-Inc/cloudmock/benchmarks/harness"
)

type cloudTrailSuite struct{}

func NewCloudTrailSuite() harness.Suite { return &cloudTrailSuite{} }
func (s *cloudTrailSuite) Name() string { return "cloudtrail" }
func (s *cloudTrailSuite) Tier() int    { return 1 }

func (s *cloudTrailSuite) Operations() []harness.Operation {
	trailName := fmt.Sprintf("bench-%s", uuid.New().String()[:8])

	newClient := func(endpoint string) (*cloudtrail.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return cloudtrail.NewFromConfig(cfg, func(o *cloudtrail.Options) {
			o.BaseEndpoint = awsclient.Endpoint(endpoint)
		}), nil
	}

	createTrail := func(ctx context.Context, endpoint string) error {
		client, err := newClient(endpoint)
		if err != nil {
			return err
		}
		_, err = client.CreateTrail(ctx, &cloudtrail.CreateTrailInput{
			Name:         aws.String(trailName),
			S3BucketName: aws.String("bench-trail-bucket"),
		})
		return err
	}

	deleteTrail := func(ctx context.Context, endpoint string) error {
		client, err := newClient(endpoint)
		if err != nil {
			return err
		}
		_, err = client.DeleteTrail(ctx, &cloudtrail.DeleteTrailInput{
			Name: aws.String(trailName),
		})
		return err
	}

	return []harness.Operation{
		{
			Name: "CreateTrail",
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				out, err := client.CreateTrail(ctx, &cloudtrail.CreateTrailInput{
					Name:         aws.String(trailName),
					S3BucketName: aws.String("bench-trail-bucket"),
				})
				if err != nil {
					return nil, err
				}
				client.DeleteTrail(ctx, &cloudtrail.DeleteTrailInput{Name: aws.String(trailName)})
				return out, nil
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "CreateTrailOutput")}
			},
		},
		{
			Name: "DescribeTrails",
			Setup: func(ctx context.Context, endpoint string) error {
				return createTrail(ctx, endpoint)
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.DescribeTrails(ctx, &cloudtrail.DescribeTrailsInput{})
			},
			Teardown: func(ctx context.Context, endpoint string) error {
				return deleteTrail(ctx, endpoint)
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "DescribeTrailsOutput")}
			},
		},
		{
			Name: "DeleteTrail",
			Setup: func(ctx context.Context, endpoint string) error {
				return createTrail(ctx, endpoint)
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.DeleteTrail(ctx, &cloudtrail.DeleteTrailInput{
					Name: aws.String(trailName),
				})
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "DeleteTrailOutput")}
			},
		},
	}
}
