package tier1

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/google/uuid"
	"github.com/Viridian-Inc/cloudmock/benchmarks/awsclient"
	"github.com/Viridian-Inc/cloudmock/benchmarks/harness"
)

type route53Suite struct{}

func NewRoute53Suite() harness.Suite { return &route53Suite{} }
func (s *route53Suite) Name() string { return "route53" }
func (s *route53Suite) Tier() int    { return 1 }

func (s *route53Suite) Operations() []harness.Operation {
	callerRef := uuid.New().String()

	newClient := func(endpoint string) (*route53.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return route53.NewFromConfig(cfg, func(o *route53.Options) {
			o.BaseEndpoint = awsclient.Endpoint(endpoint)
		}), nil
	}

	createHostedZone := func(ctx context.Context, endpoint string) (string, error) {
		client, err := newClient(endpoint)
		if err != nil {
			return "", err
		}
		out, err := client.CreateHostedZone(ctx, &route53.CreateHostedZoneInput{
			Name:            aws.String("bench.example.com"),
			CallerReference: aws.String(callerRef),
		})
		if err != nil {
			return "", err
		}
		return aws.ToString(out.HostedZone.Id), nil
	}

	return []harness.Operation{
		{
			Name: "CreateHostedZone",
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				out, err := client.CreateHostedZone(ctx, &route53.CreateHostedZoneInput{
					Name:            aws.String("bench.example.com"),
					CallerReference: aws.String(uuid.New().String()),
				})
				if err != nil {
					return nil, err
				}
				// clean up
				client.DeleteHostedZone(ctx, &route53.DeleteHostedZoneInput{Id: out.HostedZone.Id})
				return out, nil
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "CreateHostedZoneOutput")}
			},
		},
		{
			Name: "ListHostedZones",
			Setup: func(ctx context.Context, endpoint string) error {
				_, err := createHostedZone(ctx, endpoint)
				return err
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.ListHostedZones(ctx, &route53.ListHostedZonesInput{})
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "ListHostedZonesOutput")}
			},
		},
		{
			Name: "DeleteHostedZone",
			Setup: func(ctx context.Context, endpoint string) error {
				_, err := createHostedZone(ctx, endpoint)
				return err
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				zoneID, err := createHostedZone(ctx, endpoint)
				if err != nil {
					return nil, err
				}
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.DeleteHostedZone(ctx, &route53.DeleteHostedZoneInput{
					Id: aws.String(zoneID),
				})
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "DeleteHostedZoneOutput")}
			},
		},
	}
}
