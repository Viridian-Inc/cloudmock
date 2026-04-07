package tier1

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/codepipeline"
	cpltypes "github.com/aws/aws-sdk-go-v2/service/codepipeline/types"
	"github.com/google/uuid"
	"github.com/Viridian-Inc/cloudmock/benchmarks/awsclient"
	"github.com/Viridian-Inc/cloudmock/benchmarks/harness"
)

type codePipelineSuite struct{}

func NewCodePipelineSuite() harness.Suite { return &codePipelineSuite{} }
func (s *codePipelineSuite) Name() string { return "codepipeline" }
func (s *codePipelineSuite) Tier() int    { return 1 }

func (s *codePipelineSuite) Operations() []harness.Operation {
	pipelineName := fmt.Sprintf("bench-%s", uuid.New().String()[:8])
	roleArn := "arn:aws:iam::000000000000:role/bench-pipeline-role"

	newClient := func(endpoint string) (*codepipeline.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return codepipeline.NewFromConfig(cfg, func(o *codepipeline.Options) {
			o.BaseEndpoint = awsclient.Endpoint(endpoint)
		}), nil
	}

	pipelineDeclaration := func() *cpltypes.PipelineDeclaration {
		return &cpltypes.PipelineDeclaration{
			Name:    aws.String(pipelineName),
			RoleArn: aws.String(roleArn),
			ArtifactStore: &cpltypes.ArtifactStore{
				Type:     cpltypes.ArtifactStoreTypeS3,
				Location: aws.String("bench-pipeline-bucket"),
			},
			Stages: []cpltypes.StageDeclaration{
				{
					Name: aws.String("Source"),
					Actions: []cpltypes.ActionDeclaration{
						{
							Name: aws.String("Source"),
							ActionTypeId: &cpltypes.ActionTypeId{
								Category: cpltypes.ActionCategorySource,
								Owner:    cpltypes.ActionOwnerThirdParty,
								Provider: aws.String("GitHub"),
								Version:  aws.String("1"),
							},
							Configuration: map[string]string{
								"Owner":  "bench",
								"Repo":   "bench-repo",
								"Branch": "main",
								"OAuthToken": "fake-token",
							},
							OutputArtifacts: []cpltypes.OutputArtifact{
								{Name: aws.String("SourceOutput")},
							},
						},
					},
				},
				{
					Name: aws.String("Build"),
					Actions: []cpltypes.ActionDeclaration{
						{
							Name: aws.String("Build"),
							ActionTypeId: &cpltypes.ActionTypeId{
								Category: cpltypes.ActionCategoryBuild,
								Owner:    cpltypes.ActionOwnerAws,
								Provider: aws.String("CodeBuild"),
								Version:  aws.String("1"),
							},
							Configuration: map[string]string{
								"ProjectName": "bench-project",
							},
							InputArtifacts: []cpltypes.InputArtifact{
								{Name: aws.String("SourceOutput")},
							},
						},
					},
				},
			},
		}
	}

	createPipeline := func(ctx context.Context, endpoint string) error {
		client, err := newClient(endpoint)
		if err != nil {
			return err
		}
		_, err = client.CreatePipeline(ctx, &codepipeline.CreatePipelineInput{
			Pipeline: pipelineDeclaration(),
		})
		return err
	}

	deletePipeline := func(ctx context.Context, endpoint string) error {
		client, err := newClient(endpoint)
		if err != nil {
			return err
		}
		_, err = client.DeletePipeline(ctx, &codepipeline.DeletePipelineInput{
			Name: aws.String(pipelineName),
		})
		return err
	}

	return []harness.Operation{
		{
			Name: "CreatePipeline",
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				out, err := client.CreatePipeline(ctx, &codepipeline.CreatePipelineInput{
					Pipeline: pipelineDeclaration(),
				})
				if err != nil {
					return nil, err
				}
				client.DeletePipeline(ctx, &codepipeline.DeletePipelineInput{Name: aws.String(pipelineName)})
				return out, nil
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "CreatePipelineOutput")}
			},
		},
		{
			Name: "GetPipeline",
			Setup: func(ctx context.Context, endpoint string) error {
				return createPipeline(ctx, endpoint)
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.GetPipeline(ctx, &codepipeline.GetPipelineInput{
					Name: aws.String(pipelineName),
				})
			},
			Teardown: func(ctx context.Context, endpoint string) error {
				return deletePipeline(ctx, endpoint)
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "GetPipelineOutput")}
			},
		},
		{
			Name: "ListPipelines",
			Setup: func(ctx context.Context, endpoint string) error {
				return createPipeline(ctx, endpoint)
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.ListPipelines(ctx, &codepipeline.ListPipelinesInput{})
			},
			Teardown: func(ctx context.Context, endpoint string) error {
				return deletePipeline(ctx, endpoint)
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "ListPipelinesOutput")}
			},
		},
		{
			Name: "DeletePipeline",
			Setup: func(ctx context.Context, endpoint string) error {
				return createPipeline(ctx, endpoint)
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.DeletePipeline(ctx, &codepipeline.DeletePipelineInput{
					Name: aws.String(pipelineName),
				})
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "DeletePipelineOutput")}
			},
		},
	}
}
