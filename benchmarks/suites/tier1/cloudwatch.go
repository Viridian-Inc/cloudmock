package tier1

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/Viridian-Inc/cloudmock/benchmarks/awsclient"
	"github.com/Viridian-Inc/cloudmock/benchmarks/harness"
)

type cloudWatchSuite struct{}

func NewCloudWatchSuite() harness.Suite { return &cloudWatchSuite{} }
func (s *cloudWatchSuite) Name() string { return "cloudwatch" }
func (s *cloudWatchSuite) Tier() int    { return 1 }

func (s *cloudWatchSuite) Operations() []harness.Operation {
	newClient := func(endpoint string) (*cloudwatch.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return cloudwatch.NewFromConfig(cfg, func(o *cloudwatch.Options) {
			o.BaseEndpoint = awsclient.Endpoint(endpoint)
		}), nil
	}

	return []harness.Operation{
		{
			Name: "PutMetricData",
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.PutMetricData(ctx, &cloudwatch.PutMetricDataInput{
					Namespace: aws.String("Bench"),
					MetricData: []cwtypes.MetricDatum{
						{
							MetricName: aws.String("TestMetric"),
							Value:      aws.Float64(42.0),
							Timestamp:  aws.Time(time.Now()),
							Unit:       cwtypes.StandardUnitCount,
						},
					},
				})
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "PutMetricDataOutput")}
			},
		},
		{
			Name: "GetMetricData",
			Setup: func(ctx context.Context, endpoint string) error {
				client, err := newClient(endpoint)
				if err != nil {
					return err
				}
				_, err = client.PutMetricData(ctx, &cloudwatch.PutMetricDataInput{
					Namespace: aws.String("Bench"),
					MetricData: []cwtypes.MetricDatum{
						{
							MetricName: aws.String("TestMetric"),
							Value:      aws.Float64(42.0),
							Timestamp:  aws.Time(time.Now()),
							Unit:       cwtypes.StandardUnitCount,
						},
					},
				})
				return err
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				now := time.Now()
				return client.GetMetricData(ctx, &cloudwatch.GetMetricDataInput{
					StartTime: aws.Time(now.Add(-5 * time.Minute)),
					EndTime:   aws.Time(now),
					MetricDataQueries: []cwtypes.MetricDataQuery{
						{
							Id: aws.String("m1"),
							MetricStat: &cwtypes.MetricStat{
								Metric: &cwtypes.Metric{
									Namespace:  aws.String("Bench"),
									MetricName: aws.String("TestMetric"),
								},
								Period: aws.Int32(60),
								Stat:   aws.String("Sum"),
							},
						},
					},
				})
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "GetMetricDataOutput")}
			},
		},
		{
			Name: "ListMetrics",
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.ListMetrics(ctx, &cloudwatch.ListMetricsInput{
					Namespace: aws.String("Bench"),
				})
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "ListMetricsOutput")}
			},
		},
	}
}
