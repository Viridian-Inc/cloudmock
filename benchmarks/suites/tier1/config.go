package tier1

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/configservice"
	configtypes "github.com/aws/aws-sdk-go-v2/service/configservice/types"
	"github.com/google/uuid"
	"github.com/neureaux/cloudmock/benchmarks/awsclient"
	"github.com/neureaux/cloudmock/benchmarks/harness"
)

type configSuite struct{}

func NewConfigSuite() harness.Suite { return &configSuite{} }
func (s *configSuite) Name() string { return "config" }
func (s *configSuite) Tier() int    { return 1 }

func (s *configSuite) Operations() []harness.Operation {
	ruleName := fmt.Sprintf("bench-%s", uuid.New().String()[:8])

	newClient := func(endpoint string) (*configservice.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return configservice.NewFromConfig(cfg, func(o *configservice.Options) {
			o.BaseEndpoint = awsclient.Endpoint(endpoint)
		}), nil
	}

	putConfigRule := func(ctx context.Context, endpoint string) error {
		client, err := newClient(endpoint)
		if err != nil {
			return err
		}
		_, err = client.PutConfigRule(ctx, &configservice.PutConfigRuleInput{
			ConfigRule: &configtypes.ConfigRule{
				ConfigRuleName: aws.String(ruleName),
				Source: &configtypes.Source{
					Owner:            configtypes.OwnerAws,
					SourceIdentifier: aws.String("S3_BUCKET_VERSIONING_ENABLED"),
				},
			},
		})
		return err
	}

	deleteConfigRule := func(ctx context.Context, endpoint string) error {
		client, err := newClient(endpoint)
		if err != nil {
			return err
		}
		_, err = client.DeleteConfigRule(ctx, &configservice.DeleteConfigRuleInput{
			ConfigRuleName: aws.String(ruleName),
		})
		return err
	}

	return []harness.Operation{
		{
			Name: "PutConfigRule",
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				out, err := client.PutConfigRule(ctx, &configservice.PutConfigRuleInput{
					ConfigRule: &configtypes.ConfigRule{
						ConfigRuleName: aws.String(ruleName),
						Source: &configtypes.Source{
							Owner:            configtypes.OwnerAws,
							SourceIdentifier: aws.String("S3_BUCKET_VERSIONING_ENABLED"),
						},
					},
				})
				if err != nil {
					return nil, err
				}
				client.DeleteConfigRule(ctx, &configservice.DeleteConfigRuleInput{ConfigRuleName: aws.String(ruleName)})
				return out, nil
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "PutConfigRuleOutput")}
			},
		},
		{
			Name: "DescribeConfigRules",
			Setup: func(ctx context.Context, endpoint string) error {
				return putConfigRule(ctx, endpoint)
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.DescribeConfigRules(ctx, &configservice.DescribeConfigRulesInput{
					ConfigRuleNames: []string{ruleName},
				})
			},
			Teardown: func(ctx context.Context, endpoint string) error {
				return deleteConfigRule(ctx, endpoint)
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "DescribeConfigRulesOutput")}
			},
		},
		{
			Name: "DeleteConfigRule",
			Setup: func(ctx context.Context, endpoint string) error {
				return putConfigRule(ctx, endpoint)
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.DeleteConfigRule(ctx, &configservice.DeleteConfigRuleInput{
					ConfigRuleName: aws.String(ruleName),
				})
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "DeleteConfigRuleOutput")}
			},
		},
	}
}
