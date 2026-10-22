package awsclient

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

func NewConfig(endpoint string) (aws.Config, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider("test", "test", ""),
		),
	)
	if err != nil {
		return cfg, err
	}
	// A custom HTTPClient bypasses the SDK's per-service defaults-mode transport
	// adjustments. Preserve those adjustments when a nonlegacy mode is selected.
	if cfg.DefaultsMode != "" && cfg.DefaultsMode != aws.DefaultsModeLegacy {
		return cfg, nil
	}
	client := cfg.HTTPClient
	if client == nil {
		client = awshttp.NewBuildableClient()
	}
	cfg.HTTPClient = &coalescingClient{base: client}
	return cfg, nil
}

func Endpoint(endpoint string) *string {
	return aws.String(endpoint)
}
