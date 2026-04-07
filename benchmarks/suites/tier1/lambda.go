package tier1

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	lambdatypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/google/uuid"
	"github.com/Viridian-Inc/cloudmock/benchmarks/awsclient"
	"github.com/Viridian-Inc/cloudmock/benchmarks/harness"
)

type lambdaSuite struct{}

func NewLambdaSuite() harness.Suite { return &lambdaSuite{} }
func (s *lambdaSuite) Name() string { return "lambda" }
func (s *lambdaSuite) Tier() int    { return 1 }

var lambdaZipPayload = []byte("fake-zip")

func (s *lambdaSuite) Operations() []harness.Operation {
	funcName := fmt.Sprintf("bench-%s", uuid.New().String()[:8])
	roleArn := "arn:aws:iam::000000000000:role/bench-role"

	newClient := func(endpoint string) (*lambda.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return lambda.NewFromConfig(cfg, func(o *lambda.Options) {
			o.BaseEndpoint = awsclient.Endpoint(endpoint)
		}), nil
	}

	createFunction := func(ctx context.Context, endpoint string) error {
		client, err := newClient(endpoint)
		if err != nil {
			return err
		}
		_, err = client.CreateFunction(ctx, &lambda.CreateFunctionInput{
			FunctionName: aws.String(funcName),
			Runtime:      lambdatypes.RuntimeNodejs20x,
			Handler:      aws.String("index.handler"),
			Role:         aws.String(roleArn),
			Code: &lambdatypes.FunctionCode{
				ZipFile: lambdaZipPayload,
			},
		})
		return err
	}

	deleteFunction := func(ctx context.Context, endpoint string) error {
		client, err := newClient(endpoint)
		if err != nil {
			return err
		}
		_, err = client.DeleteFunction(ctx, &lambda.DeleteFunctionInput{
			FunctionName: aws.String(funcName),
		})
		return err
	}

	return []harness.Operation{
		{
			Name: "CreateFunction",
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				out, err := client.CreateFunction(ctx, &lambda.CreateFunctionInput{
					FunctionName: aws.String(funcName),
					Runtime:      lambdatypes.RuntimeNodejs20x,
					Handler:      aws.String("index.handler"),
					Role:         aws.String(roleArn),
					Code: &lambdatypes.FunctionCode{
						ZipFile: lambdaZipPayload,
					},
				})
				if err != nil {
					return nil, err
				}
				// clean up
				client.DeleteFunction(ctx, &lambda.DeleteFunctionInput{FunctionName: aws.String(funcName)})
				return out, nil
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "CreateFunctionOutput")}
			},
		},
		{
			Name: "GetFunction",
			Setup: func(ctx context.Context, endpoint string) error {
				return createFunction(ctx, endpoint)
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.GetFunction(ctx, &lambda.GetFunctionInput{
					FunctionName: aws.String(funcName),
				})
			},
			Teardown: func(ctx context.Context, endpoint string) error {
				return deleteFunction(ctx, endpoint)
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "GetFunctionOutput")}
			},
		},
		{
			Name: "ListFunctions",
			Setup: func(ctx context.Context, endpoint string) error {
				return createFunction(ctx, endpoint)
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.ListFunctions(ctx, &lambda.ListFunctionsInput{})
			},
			Teardown: func(ctx context.Context, endpoint string) error {
				return deleteFunction(ctx, endpoint)
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "ListFunctionsOutput")}
			},
		},
		{
			Name: "DeleteFunction",
			Setup: func(ctx context.Context, endpoint string) error {
				return createFunction(ctx, endpoint)
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.DeleteFunction(ctx, &lambda.DeleteFunctionInput{
					FunctionName: aws.String(funcName),
				})
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "DeleteFunctionOutput")}
			},
		},
	}
}
