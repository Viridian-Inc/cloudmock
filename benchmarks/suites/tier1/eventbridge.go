package tier1

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	eventbridgetypes "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/google/uuid"
	"github.com/Viridian-Inc/cloudmock/benchmarks/awsclient"
	"github.com/Viridian-Inc/cloudmock/benchmarks/harness"
)

type eventBridgeSuite struct{}

func NewEventBridgeSuite() harness.Suite { return &eventBridgeSuite{} }
func (s *eventBridgeSuite) Name() string { return "eventbridge" }
func (s *eventBridgeSuite) Tier() int    { return 1 }

func (s *eventBridgeSuite) Operations() []harness.Operation {
	busName := fmt.Sprintf("bench-%s", uuid.New().String()[:8])
	ruleName := fmt.Sprintf("bench-rule-%s", uuid.New().String()[:8])

	newClient := func(endpoint string) (*eventbridge.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return eventbridge.NewFromConfig(cfg, func(o *eventbridge.Options) {
			o.BaseEndpoint = awsclient.Endpoint(endpoint)
		}), nil
	}

	createBus := func(ctx context.Context, endpoint string) error {
		client, err := newClient(endpoint)
		if err != nil {
			return err
		}
		_, err = client.CreateEventBus(ctx, &eventbridge.CreateEventBusInput{
			Name: aws.String(busName),
		})
		return err
	}

	deleteBus := func(ctx context.Context, endpoint string) error {
		client, err := newClient(endpoint)
		if err != nil {
			return err
		}
		_, err = client.DeleteEventBus(ctx, &eventbridge.DeleteEventBusInput{
			Name: aws.String(busName),
		})
		return err
	}

	return []harness.Operation{
		{
			Name: "CreateEventBus",
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				out, err := client.CreateEventBus(ctx, &eventbridge.CreateEventBusInput{
					Name: aws.String(busName),
				})
				if err != nil {
					return nil, err
				}
				client.DeleteEventBus(ctx, &eventbridge.DeleteEventBusInput{Name: aws.String(busName)})
				return out, nil
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "CreateEventBusOutput")}
			},
		},
		{
			Name: "PutRule",
			Setup: func(ctx context.Context, endpoint string) error {
				return createBus(ctx, endpoint)
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.PutRule(ctx, &eventbridge.PutRuleInput{
					Name:         aws.String(ruleName),
					EventBusName: aws.String(busName),
					ScheduleExpression: aws.String("rate(5 minutes)"),
					State:        eventbridgetypes.RuleStateEnabled,
				})
			},
			Teardown: func(ctx context.Context, endpoint string) error {
				return deleteBus(ctx, endpoint)
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "PutRuleOutput")}
			},
		},
		{
			Name: "PutEvents",
			Setup: func(ctx context.Context, endpoint string) error {
				return createBus(ctx, endpoint)
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.PutEvents(ctx, &eventbridge.PutEventsInput{
					Entries: []eventbridgetypes.PutEventsRequestEntry{
						{
							EventBusName: aws.String(busName),
							Source:       aws.String("bench.test"),
							DetailType:   aws.String("BenchmarkEvent"),
							Detail:       aws.String(`{"key":"value"}`),
							Time:         aws.Time(time.Now()),
						},
					},
				})
			},
			Teardown: func(ctx context.Context, endpoint string) error {
				return deleteBus(ctx, endpoint)
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "PutEventsOutput")}
			},
		},
		{
			Name: "DeleteEventBus",
			Setup: func(ctx context.Context, endpoint string) error {
				return createBus(ctx, endpoint)
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.DeleteEventBus(ctx, &eventbridge.DeleteEventBusInput{
					Name: aws.String(busName),
				})
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "DeleteEventBusOutput")}
			},
		},
	}
}
