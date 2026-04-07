package tier1

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/google/uuid"
	"github.com/Viridian-Inc/cloudmock/benchmarks/awsclient"
	"github.com/Viridian-Inc/cloudmock/benchmarks/harness"
)

type apiGatewaySuite struct{}

func NewAPIGatewaySuite() harness.Suite { return &apiGatewaySuite{} }
func (s *apiGatewaySuite) Name() string { return "apigateway" }
func (s *apiGatewaySuite) Tier() int    { return 1 }

func (s *apiGatewaySuite) Operations() []harness.Operation {
	apiName := fmt.Sprintf("bench-%s", uuid.New().String()[:8])

	newClient := func(endpoint string) (*apigateway.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return apigateway.NewFromConfig(cfg, func(o *apigateway.Options) {
			o.BaseEndpoint = awsclient.Endpoint(endpoint)
		}), nil
	}

	createRestAPI := func(ctx context.Context, endpoint string) (string, error) {
		client, err := newClient(endpoint)
		if err != nil {
			return "", err
		}
		out, err := client.CreateRestApi(ctx, &apigateway.CreateRestApiInput{
			Name: aws.String(apiName),
		})
		if err != nil {
			return "", err
		}
		return aws.ToString(out.Id), nil
	}

	return []harness.Operation{
		{
			Name: "CreateRestApi",
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				out, err := client.CreateRestApi(ctx, &apigateway.CreateRestApiInput{
					Name: aws.String(apiName),
				})
				if err != nil {
					return nil, err
				}
				// clean up
				client.DeleteRestApi(ctx, &apigateway.DeleteRestApiInput{RestApiId: out.Id})
				return out, nil
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "CreateRestApiOutput")}
			},
		},
		{
			Name: "GetRestApi",
			Setup: func(ctx context.Context, endpoint string) error {
				_, err := createRestAPI(ctx, endpoint)
				return err
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				apiID, err := createRestAPI(ctx, endpoint)
				if err != nil {
					return nil, err
				}
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.GetRestApi(ctx, &apigateway.GetRestApiInput{
					RestApiId: aws.String(apiID),
				})
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "GetRestApiOutput")}
			},
		},
		{
			Name: "GetRestApis",
			Setup: func(ctx context.Context, endpoint string) error {
				_, err := createRestAPI(ctx, endpoint)
				return err
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.GetRestApis(ctx, &apigateway.GetRestApisInput{})
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "GetRestApisOutput")}
			},
		},
		{
			Name: "DeleteRestApi",
			Setup: func(ctx context.Context, endpoint string) error {
				_, err := createRestAPI(ctx, endpoint)
				return err
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				apiID, err := createRestAPI(ctx, endpoint)
				if err != nil {
					return nil, err
				}
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.DeleteRestApi(ctx, &apigateway.DeleteRestApiInput{
					RestApiId: aws.String(apiID),
				})
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "DeleteRestApiOutput")}
			},
		},
	}
}
