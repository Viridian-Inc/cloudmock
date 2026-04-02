package tier1

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/codebuild"
	codebuildtypes "github.com/aws/aws-sdk-go-v2/service/codebuild/types"
	"github.com/google/uuid"
	"github.com/neureaux/cloudmock/benchmarks/awsclient"
	"github.com/neureaux/cloudmock/benchmarks/harness"
)

type codeBuildSuite struct{}

func NewCodeBuildSuite() harness.Suite { return &codeBuildSuite{} }
func (s *codeBuildSuite) Name() string { return "codebuild" }
func (s *codeBuildSuite) Tier() int    { return 1 }

func (s *codeBuildSuite) Operations() []harness.Operation {
	projectName := fmt.Sprintf("bench-%s", uuid.New().String()[:8])

	newClient := func(endpoint string) (*codebuild.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return codebuild.NewFromConfig(cfg, func(o *codebuild.Options) {
			o.BaseEndpoint = awsclient.Endpoint(endpoint)
		}), nil
	}

	createProject := func(ctx context.Context, endpoint string) error {
		client, err := newClient(endpoint)
		if err != nil {
			return err
		}
		_, err = client.CreateProject(ctx, &codebuild.CreateProjectInput{
			Name: aws.String(projectName),
			Source: &codebuildtypes.ProjectSource{
				Type:      codebuildtypes.SourceTypeNoSource,
				Buildspec: aws.String("version: 0.2\nphases:\n  build:\n    commands:\n      - echo hi"),
			},
			Artifacts: &codebuildtypes.ProjectArtifacts{
				Type: codebuildtypes.ArtifactsTypeNoArtifacts,
			},
			Environment: &codebuildtypes.ProjectEnvironment{
				Type:           codebuildtypes.EnvironmentTypeLinuxContainer,
				Image:          aws.String("aws/codebuild/standard:7.0"),
				ComputeType:    codebuildtypes.ComputeTypeBuildGeneral1Small,
			},
		})
		return err
	}

	deleteProject := func(ctx context.Context, endpoint string) error {
		client, err := newClient(endpoint)
		if err != nil {
			return err
		}
		_, err = client.DeleteProject(ctx, &codebuild.DeleteProjectInput{
			Name: aws.String(projectName),
		})
		return err
	}

	return []harness.Operation{
		{
			Name: "CreateProject",
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				out, err := client.CreateProject(ctx, &codebuild.CreateProjectInput{
					Name: aws.String(projectName),
					Source: &codebuildtypes.ProjectSource{
						Type:      codebuildtypes.SourceTypeNoSource,
						Buildspec: aws.String("version: 0.2\nphases:\n  build:\n    commands:\n      - echo hi"),
					},
					Artifacts: &codebuildtypes.ProjectArtifacts{
						Type: codebuildtypes.ArtifactsTypeNoArtifacts,
					},
					Environment: &codebuildtypes.ProjectEnvironment{
						Type:        codebuildtypes.EnvironmentTypeLinuxContainer,
						Image:       aws.String("aws/codebuild/standard:7.0"),
						ComputeType: codebuildtypes.ComputeTypeBuildGeneral1Small,
					},
				})
				if err != nil {
					return nil, err
				}
				client.DeleteProject(ctx, &codebuild.DeleteProjectInput{Name: aws.String(projectName)})
				return out, nil
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "CreateProjectOutput")}
			},
		},
		{
			Name: "ListProjects",
			Setup: func(ctx context.Context, endpoint string) error {
				return createProject(ctx, endpoint)
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.ListProjects(ctx, &codebuild.ListProjectsInput{})
			},
			Teardown: func(ctx context.Context, endpoint string) error {
				return deleteProject(ctx, endpoint)
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "ListProjectsOutput")}
			},
		},
		{
			Name: "DeleteProject",
			Setup: func(ctx context.Context, endpoint string) error {
				return createProject(ctx, endpoint)
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.DeleteProject(ctx, &codebuild.DeleteProjectInput{
					Name: aws.String(projectName),
				})
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "DeleteProjectOutput")}
			},
		},
	}
}
