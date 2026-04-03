package tier2

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/account"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/acmpca"
	"github.com/aws/aws-sdk-go-v2/service/amplify"
	"github.com/aws/aws-sdk-go-v2/service/appconfig"
	"github.com/aws/aws-sdk-go-v2/service/applicationautoscaling"
	"github.com/aws/aws-sdk-go-v2/service/apprunner"
	"github.com/aws/aws-sdk-go-v2/service/appsync"
	"github.com/aws/aws-sdk-go-v2/service/athena"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/backup"
	"github.com/aws/aws-sdk-go-v2/service/batch"
	"github.com/aws/aws-sdk-go-v2/service/bedrock"
	"github.com/aws/aws-sdk-go-v2/service/cloudcontrol"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/codeartifact"
	"github.com/aws/aws-sdk-go-v2/service/codecommit"
	"github.com/aws/aws-sdk-go-v2/service/codeconnections"
	"github.com/aws/aws-sdk-go-v2/service/codedeploy"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	cetypes "github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
	"github.com/aws/aws-sdk-go-v2/service/databasemigrationservice"
	"github.com/aws/aws-sdk-go-v2/service/datasync"
	"github.com/aws/aws-sdk-go-v2/service/dax"
	"github.com/aws/aws-sdk-go-v2/service/docdb"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	"github.com/aws/aws-sdk-go-v2/service/elasticbeanstalk"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	"github.com/aws/aws-sdk-go-v2/service/elasticsearchservice"
	"github.com/aws/aws-sdk-go-v2/service/emr"
	"github.com/aws/aws-sdk-go-v2/service/fis"
	"github.com/aws/aws-sdk-go-v2/service/glacier"
	"github.com/aws/aws-sdk-go-v2/service/glue"
	"github.com/aws/aws-sdk-go-v2/service/identitystore"
	"github.com/aws/aws-sdk-go-v2/service/iot"
	"github.com/aws/aws-sdk-go-v2/service/iotdataplane"
	"github.com/aws/aws-sdk-go-v2/service/iotwireless"
	"github.com/aws/aws-sdk-go-v2/service/kafka"
	"github.com/aws/aws-sdk-go-v2/service/kinesisanalytics"
	"github.com/aws/aws-sdk-go-v2/service/lakeformation"
	"github.com/aws/aws-sdk-go-v2/service/managedblockchain"
	"github.com/aws/aws-sdk-go-v2/service/mediaconvert"
	"github.com/aws/aws-sdk-go-v2/service/memorydb"
	"github.com/aws/aws-sdk-go-v2/service/mq"
	"github.com/aws/aws-sdk-go-v2/service/neptune"
	"github.com/aws/aws-sdk-go-v2/service/opensearch"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/pinpoint"
	"github.com/aws/aws-sdk-go-v2/service/pipes"
	"github.com/aws/aws-sdk-go-v2/service/ram"
	"github.com/aws/aws-sdk-go-v2/service/redshift"
	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroups"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go-v2/service/route53resolver"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/serverlessapplicationrepository"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/aws/aws-sdk-go-v2/service/shield"
	"github.com/aws/aws-sdk-go-v2/service/ssoadmin"
	"github.com/aws/aws-sdk-go-v2/service/support"
	"github.com/aws/aws-sdk-go-v2/service/swf"
	"github.com/aws/aws-sdk-go-v2/service/textract"
	"github.com/aws/aws-sdk-go-v2/service/timestreamwrite"
	"github.com/aws/aws-sdk-go-v2/service/transcribe"
	"github.com/aws/aws-sdk-go-v2/service/transfer"
	"github.com/aws/aws-sdk-go-v2/service/verifiedpermissions"
	"github.com/aws/aws-sdk-go-v2/service/wafregional"
	"github.com/aws/aws-sdk-go-v2/service/wafv2"
	wafv2types "github.com/aws/aws-sdk-go-v2/service/wafv2/types"
	"github.com/aws/aws-sdk-go-v2/service/xray"
	"github.com/neureaux/cloudmock/benchmarks/awsclient"
	"github.com/neureaux/cloudmock/benchmarks/harness"
)

// sdkSuite implements harness.Suite for tier-2 services using real AWS SDK clients.
type sdkSuite struct {
	name string
	ops  []harness.Operation
}

func (s *sdkSuite) Name() string              { return s.name }
func (s *sdkSuite) Tier() int                 { return 2 }
func (s *sdkSuite) Operations() []harness.Operation { return s.ops }

// op builds a harness.Operation with a simple NotNil validator.
func op(name string, run func(ctx context.Context, endpoint string) (any, error)) harness.Operation {
	return harness.Operation{
		Name: name,
		Run:  run,
		Validate: func(resp any) []harness.Finding {
			return []harness.Finding{harness.CheckNotNil(resp, name+"Output")}
		},
	}
}

// clientCache is a generic helper that caches one SDK client per endpoint.
type clientCache[T any] struct {
	mu      sync.Mutex
	clients map[string]*T
	factory func(endpoint string) (*T, error)
}

func newCache[T any](factory func(endpoint string) (*T, error)) *clientCache[T] {
	return &clientCache[T]{clients: make(map[string]*T), factory: factory}
}

func (c *clientCache[T]) get(endpoint string) (*T, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if cl, ok := c.clients[endpoint]; ok {
		return cl, nil
	}
	cl, err := c.factory(endpoint)
	if err != nil {
		return nil, err
	}
	c.clients[endpoint] = cl
	return cl, nil
}

// GenerateAll returns one harness.Suite per tier-2 service, using real AWS SDK
// clients for correct protocol serialization.
func GenerateAll() []harness.Suite {
	return []harness.Suite{
		accountSuite(),
		acmSuite(),
		acmpcaSuite(),
		amplifySuite(),
		appconfigSuite(),
		applicationAutoscalingSuite(),
		apprunnerSuite(),
		appsyncSuite(),
		athenaSuite(),
		autoscalingSuite(),
		backupSuite(),
		batchSuite(),
		bedrockSuite(),
		cloudcontrolSuite(),
		cloudfrontSuite(),
		codeartifactSuite(),
		codecommitSuite(),
		codeconnectionsSuite(),
		codedeploySuite(),
		costexplorerSuite(),
		databasemigrationSuite(),
		datasyncSuite(),
		daxSuite(),
		docdbSuite(),
		elasticacheSuite(),
		elasticbeanstalkSuite(),
		elasticloadbalancingSuite(),
		elasticsearchSuite(),
		emrSuite(),
		fisSuite(),
		glacierSuite(),
		glueSuite(),
		identitystoreSuite(),
		iotSuite(),
		iotdataSuite(),
		iotwirelessSuite(),
		kafkaSuite(),
		kinesisanalyticsSuite(),
		lakeformationSuite(),
		managedblockchainSuite(),
		mediaconvertSuite(),
		memorydbSuite(),
		mqSuite(),
		neptuneSuite(),
		opensearchSuite(),
		organizationsSuite(),
		pinpointSuite(),
		pipesSuite(),
		ramSuite(),
		redshiftSuite(),
		rekognitionSuite(),
		resourcegroupsSuite(),
		resourcegroupstaggingapiSuite(),
		route53resolverSuite(),
		sagemakerSuite(),
		schedulerSuite(),
		secretsmanagerSuite(),
		serverlessrepoSuite(),
		servicediscoverySuite(),
		sesSuite(),
		sfnSuite(),
		shieldSuite(),
		ssoadminSuite(),
		supportSuite(),
		swfSuite(),
		textractSuite(),
		timestreamwriteSuite(),
		transcribeSuite(),
		transferSuite(),
		verifiedpermissionsSuite(),
		wafregionalSuite(),
		wafv2Suite(),
		xraySuite(),
	}
}

// ─── Account ───────────────────────────────────────────────────────────────────

func accountSuite() harness.Suite {
	c := newCache(func(endpoint string) (*account.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return account.NewFromConfig(cfg, func(o *account.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "account", ops: []harness.Operation{
		op("ListRegions", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListRegions(ctx, &account.ListRegionsInput{})
		}),
	}}
}

// ─── ACM ────────────────────────────────────────────────────────────────────────

func acmSuite() harness.Suite {
	c := newCache(func(endpoint string) (*acm.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return acm.NewFromConfig(cfg, func(o *acm.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "acm", ops: []harness.Operation{
		op("RequestCertificate", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.RequestCertificate(ctx, &acm.RequestCertificateInput{
				DomainName: aws.String("bench.example.com"),
			})
		}),
		op("ListCertificates", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListCertificates(ctx, &acm.ListCertificatesInput{})
		}),
		op("DescribeCertificate", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.DescribeCertificate(ctx, &acm.DescribeCertificateInput{
				CertificateArn: aws.String("arn:aws:acm:us-east-1:000000000000:certificate/bench-cert"),
			})
		}),
	}}
}

// ─── ACM PCA ───────────────────────────────────────────────────────────────────

func acmpcaSuite() harness.Suite {
	c := newCache(func(endpoint string) (*acmpca.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return acmpca.NewFromConfig(cfg, func(o *acmpca.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "acm-pca", ops: []harness.Operation{
		op("ListCertificateAuthorities", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListCertificateAuthorities(ctx, &acmpca.ListCertificateAuthoritiesInput{})
		}),
	}}
}

// ─── Amplify ────────────────────────────────────────────────────────────────────

func amplifySuite() harness.Suite {
	c := newCache(func(endpoint string) (*amplify.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return amplify.NewFromConfig(cfg, func(o *amplify.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "amplify", ops: []harness.Operation{
		op("CreateApp", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.CreateApp(ctx, &amplify.CreateAppInput{
				Name: aws.String("bench-app"),
			})
		}),
		op("ListApps", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListApps(ctx, &amplify.ListAppsInput{})
		}),
	}}
}

// ─── AppConfig ─────────────────────────────────────────────────────────────────

func appconfigSuite() harness.Suite {
	c := newCache(func(endpoint string) (*appconfig.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return appconfig.NewFromConfig(cfg, func(o *appconfig.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "appconfig", ops: []harness.Operation{
		op("ListApplications", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListApplications(ctx, &appconfig.ListApplicationsInput{})
		}),
		op("ListEnvironments", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListEnvironments(ctx, &appconfig.ListEnvironmentsInput{
				ApplicationId: aws.String("bench-app-id"),
			})
		}),
	}}
}

// ─── Application Auto Scaling ───────────────────────────────────────────────────

func applicationAutoscalingSuite() harness.Suite {
	c := newCache(func(endpoint string) (*applicationautoscaling.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return applicationautoscaling.NewFromConfig(cfg, func(o *applicationautoscaling.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "application-autoscaling", ops: []harness.Operation{
		op("DescribeScalableTargets", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.DescribeScalableTargets(ctx, &applicationautoscaling.DescribeScalableTargetsInput{
				ServiceNamespace: "ecs",
			})
		}),
	}}
}

// ─── App Runner ────────────────────────────────────────────────────────────────

func apprunnerSuite() harness.Suite {
	c := newCache(func(endpoint string) (*apprunner.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return apprunner.NewFromConfig(cfg, func(o *apprunner.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "apprunner", ops: []harness.Operation{
		op("ListServices", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListServices(ctx, &apprunner.ListServicesInput{})
		}),
		op("ListAutoScalingConfigurations", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListAutoScalingConfigurations(ctx, &apprunner.ListAutoScalingConfigurationsInput{})
		}),
	}}
}

// ─── AppSync ────────────────────────────────────────────────────────────────────

func appsyncSuite() harness.Suite {
	c := newCache(func(endpoint string) (*appsync.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return appsync.NewFromConfig(cfg, func(o *appsync.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "appsync", ops: []harness.Operation{
		op("CreateGraphqlApi", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.CreateGraphqlApi(ctx, &appsync.CreateGraphqlApiInput{
				Name:               aws.String("bench-api"),
				AuthenticationType: "API_KEY",
			})
		}),
		op("ListGraphqlApis", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListGraphqlApis(ctx, &appsync.ListGraphqlApisInput{})
		}),
	}}
}

// ─── Athena ─────────────────────────────────────────────────────────────────────

func athenaSuite() harness.Suite {
	c := newCache(func(endpoint string) (*athena.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return athena.NewFromConfig(cfg, func(o *athena.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "athena", ops: []harness.Operation{
		op("ListQueryExecutions", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListQueryExecutions(ctx, &athena.ListQueryExecutionsInput{})
		}),
		op("StartQueryExecution", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.StartQueryExecution(ctx, &athena.StartQueryExecutionInput{
				QueryString: aws.String("SELECT 1"),
			})
		}),
	}}
}

// ─── Auto Scaling ───────────────────────────────────────────────────────────────

func autoscalingSuite() harness.Suite {
	c := newCache(func(endpoint string) (*autoscaling.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return autoscaling.NewFromConfig(cfg, func(o *autoscaling.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "autoscaling", ops: []harness.Operation{
		op("DescribeAutoScalingGroups", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.DescribeAutoScalingGroups(ctx, &autoscaling.DescribeAutoScalingGroupsInput{})
		}),
		op("DescribeLaunchConfigurations", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.DescribeLaunchConfigurations(ctx, &autoscaling.DescribeLaunchConfigurationsInput{})
		}),
		op("DescribeAutoScalingInstances", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.DescribeAutoScalingInstances(ctx, &autoscaling.DescribeAutoScalingInstancesInput{})
		}),
	}}
}

// ─── Backup ─────────────────────────────────────────────────────────────────────

func backupSuite() harness.Suite {
	c := newCache(func(endpoint string) (*backup.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return backup.NewFromConfig(cfg, func(o *backup.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "backup", ops: []harness.Operation{
		op("ListBackupPlans", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListBackupPlans(ctx, &backup.ListBackupPlansInput{})
		}),
		op("ListBackupVaults", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListBackupVaults(ctx, &backup.ListBackupVaultsInput{})
		}),
	}}
}

// ─── Batch ──────────────────────────────────────────────────────────────────────

func batchSuite() harness.Suite {
	c := newCache(func(endpoint string) (*batch.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return batch.NewFromConfig(cfg, func(o *batch.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "batch", ops: []harness.Operation{
		op("DescribeComputeEnvironments", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.DescribeComputeEnvironments(ctx, &batch.DescribeComputeEnvironmentsInput{})
		}),
		op("DescribeJobQueues", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.DescribeJobQueues(ctx, &batch.DescribeJobQueuesInput{})
		}),
	}}
}

// ─── Bedrock ───────────────────────────────────────────────────────────────────

func bedrockSuite() harness.Suite {
	c := newCache(func(endpoint string) (*bedrock.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return bedrock.NewFromConfig(cfg, func(o *bedrock.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "bedrock", ops: []harness.Operation{
		op("ListFoundationModels", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListFoundationModels(ctx, &bedrock.ListFoundationModelsInput{})
		}),
		op("ListCustomModels", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListCustomModels(ctx, &bedrock.ListCustomModelsInput{})
		}),
	}}
}

// ─── Cloud Control ──────────────────────────────────────────────────────────────

func cloudcontrolSuite() harness.Suite {
	c := newCache(func(endpoint string) (*cloudcontrol.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return cloudcontrol.NewFromConfig(cfg, func(o *cloudcontrol.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "cloudcontrol", ops: []harness.Operation{
		op("ListResources", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListResources(ctx, &cloudcontrol.ListResourcesInput{
				TypeName: aws.String("AWS::S3::Bucket"),
			})
		}),
	}}
}

// ─── CloudFront ─────────────────────────────────────────────────────────────────

func cloudfrontSuite() harness.Suite {
	c := newCache(func(endpoint string) (*cloudfront.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return cloudfront.NewFromConfig(cfg, func(o *cloudfront.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "cloudfront", ops: []harness.Operation{
		op("ListDistributions", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListDistributions(ctx, &cloudfront.ListDistributionsInput{})
		}),
	}}
}

// ─── CodeArtifact ──────────────────────────────────────────────────────────────

func codeartifactSuite() harness.Suite {
	c := newCache(func(endpoint string) (*codeartifact.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return codeartifact.NewFromConfig(cfg, func(o *codeartifact.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "codeartifact", ops: []harness.Operation{
		op("ListDomains", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListDomains(ctx, &codeartifact.ListDomainsInput{})
		}),
		op("ListRepositories", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListRepositories(ctx, &codeartifact.ListRepositoriesInput{})
		}),
	}}
}

// ─── CodeCommit ─────────────────────────────────────────────────────────────────

func codecommitSuite() harness.Suite {
	c := newCache(func(endpoint string) (*codecommit.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return codecommit.NewFromConfig(cfg, func(o *codecommit.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "codecommit", ops: []harness.Operation{
		op("CreateRepository", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.CreateRepository(ctx, &codecommit.CreateRepositoryInput{
				RepositoryName: aws.String("bench-repo"),
			})
		}),
		op("ListRepositories", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListRepositories(ctx, &codecommit.ListRepositoriesInput{})
		}),
	}}
}

// ─── CodeConnections ───────────────────────────────────────────────────────────

func codeconnectionsSuite() harness.Suite {
	c := newCache(func(endpoint string) (*codeconnections.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return codeconnections.NewFromConfig(cfg, func(o *codeconnections.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "codeconnections", ops: []harness.Operation{
		op("ListConnections", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListConnections(ctx, &codeconnections.ListConnectionsInput{})
		}),
		op("ListHosts", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListHosts(ctx, &codeconnections.ListHostsInput{})
		}),
	}}
}

// ─── CodeDeploy ─────────────────────────────────────────────────────────────────

func codedeploySuite() harness.Suite {
	c := newCache(func(endpoint string) (*codedeploy.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return codedeploy.NewFromConfig(cfg, func(o *codedeploy.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "codedeploy", ops: []harness.Operation{
		op("CreateApplication", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.CreateApplication(ctx, &codedeploy.CreateApplicationInput{
				ApplicationName: aws.String("bench-app"),
			})
		}),
		op("ListApplications", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListApplications(ctx, &codedeploy.ListApplicationsInput{})
		}),
	}}
}

// ─── Cost Explorer ──────────────────────────────────────────────────────────────

func costexplorerSuite() harness.Suite {
	c := newCache(func(endpoint string) (*costexplorer.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return costexplorer.NewFromConfig(cfg, func(o *costexplorer.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "ce", ops: []harness.Operation{
		op("GetCostAndUsage", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
				TimePeriod: &cetypes.DateInterval{
					Start: aws.String("2026-01-01"),
					End:   aws.String("2026-01-31"),
				},
				Granularity: cetypes.GranularityMonthly,
				Metrics:     []string{"BlendedCost"},
			})
		}),
	}}
}

// ─── Database Migration Service ─────────────────────────────────────────────────

func databasemigrationSuite() harness.Suite {
	c := newCache(func(endpoint string) (*databasemigrationservice.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return databasemigrationservice.NewFromConfig(cfg, func(o *databasemigrationservice.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "dms", ops: []harness.Operation{
		op("DescribeReplicationInstances", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.DescribeReplicationInstances(ctx, &databasemigrationservice.DescribeReplicationInstancesInput{})
		}),
		op("DescribeEndpoints", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.DescribeEndpoints(ctx, &databasemigrationservice.DescribeEndpointsInput{})
		}),
	}}
}

// ─── DataSync ───────────────────────────────────────────────────────────────────

func datasyncSuite() harness.Suite {
	c := newCache(func(endpoint string) (*datasync.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return datasync.NewFromConfig(cfg, func(o *datasync.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "datasync", ops: []harness.Operation{
		op("ListTasks", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListTasks(ctx, &datasync.ListTasksInput{})
		}),
		op("ListLocations", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListLocations(ctx, &datasync.ListLocationsInput{})
		}),
	}}
}

// ─── DAX ───────────────────────────────────────────────────────────────────────

func daxSuite() harness.Suite {
	c := newCache(func(endpoint string) (*dax.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return dax.NewFromConfig(cfg, func(o *dax.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "dax", ops: []harness.Operation{
		op("DescribeClusters", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.DescribeClusters(ctx, &dax.DescribeClustersInput{})
		}),
		op("DescribeSubnetGroups", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.DescribeSubnetGroups(ctx, &dax.DescribeSubnetGroupsInput{})
		}),
	}}
}

// ─── DocumentDB ─────────────────────────────────────────────────────────────────

func docdbSuite() harness.Suite {
	c := newCache(func(endpoint string) (*docdb.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return docdb.NewFromConfig(cfg, func(o *docdb.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "docdb", ops: []harness.Operation{
		op("DescribeDBClusters", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.DescribeDBClusters(ctx, &docdb.DescribeDBClustersInput{})
		}),
		op("DescribeDBInstances", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.DescribeDBInstances(ctx, &docdb.DescribeDBInstancesInput{})
		}),
	}}
}

// ─── ElastiCache ────────────────────────────────────────────────────────────────

func elasticacheSuite() harness.Suite {
	c := newCache(func(endpoint string) (*elasticache.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return elasticache.NewFromConfig(cfg, func(o *elasticache.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "elasticache", ops: []harness.Operation{
		op("DescribeCacheClusters", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.DescribeCacheClusters(ctx, &elasticache.DescribeCacheClustersInput{})
		}),
		op("DescribeReplicationGroups", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.DescribeReplicationGroups(ctx, &elasticache.DescribeReplicationGroupsInput{})
		}),
	}}
}

// ─── Elastic Beanstalk ──────────────────────────────────────────────────────────

func elasticbeanstalkSuite() harness.Suite {
	c := newCache(func(endpoint string) (*elasticbeanstalk.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return elasticbeanstalk.NewFromConfig(cfg, func(o *elasticbeanstalk.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "elasticbeanstalk", ops: []harness.Operation{
		op("CreateApplication", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.CreateApplication(ctx, &elasticbeanstalk.CreateApplicationInput{
				ApplicationName: aws.String("bench-app"),
			})
		}),
		op("DescribeApplications", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.DescribeApplications(ctx, &elasticbeanstalk.DescribeApplicationsInput{})
		}),
		op("DescribeEnvironments", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.DescribeEnvironments(ctx, &elasticbeanstalk.DescribeEnvironmentsInput{})
		}),
	}}
}

// ─── Elastic Load Balancing (Classic) ───────────────────────────────────────────

func elasticloadbalancingSuite() harness.Suite {
	c := newCache(func(endpoint string) (*elasticloadbalancing.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return elasticloadbalancing.NewFromConfig(cfg, func(o *elasticloadbalancing.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "elasticloadbalancing", ops: []harness.Operation{
		op("DescribeLoadBalancers", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.DescribeLoadBalancers(ctx, &elasticloadbalancing.DescribeLoadBalancersInput{})
		}),
	}}
}

// ─── Elasticsearch Service ─────────────────────────────────────────────────────

func elasticsearchSuite() harness.Suite {
	c := newCache(func(endpoint string) (*elasticsearchservice.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return elasticsearchservice.NewFromConfig(cfg, func(o *elasticsearchservice.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "es", ops: []harness.Operation{
		op("ListDomainNames", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListDomainNames(ctx, &elasticsearchservice.ListDomainNamesInput{})
		}),
	}}
}

// ─── EMR ────────────────────────────────────────────────────────────────────────

func emrSuite() harness.Suite {
	c := newCache(func(endpoint string) (*emr.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return emr.NewFromConfig(cfg, func(o *emr.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "elasticmapreduce", ops: []harness.Operation{
		op("ListClusters", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListClusters(ctx, &emr.ListClustersInput{})
		}),
		op("ListSteps", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListSteps(ctx, &emr.ListStepsInput{
				ClusterId: aws.String("j-BENCHCLUSTER"),
			})
		}),
	}}
}

// ─── FIS ───────────────────────────────────────────────────────────────────────

func fisSuite() harness.Suite {
	c := newCache(func(endpoint string) (*fis.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return fis.NewFromConfig(cfg, func(o *fis.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "fis", ops: []harness.Operation{
		op("ListExperimentTemplates", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListExperimentTemplates(ctx, &fis.ListExperimentTemplatesInput{})
		}),
		op("ListExperiments", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListExperiments(ctx, &fis.ListExperimentsInput{})
		}),
	}}
}

// ─── Glacier ────────────────────────────────────────────────────────────────────

func glacierSuite() harness.Suite {
	c := newCache(func(endpoint string) (*glacier.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return glacier.NewFromConfig(cfg, func(o *glacier.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "glacier", ops: []harness.Operation{
		op("ListVaults", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListVaults(ctx, &glacier.ListVaultsInput{
				AccountId: aws.String("-"),
			})
		}),
		op("CreateVault", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.CreateVault(ctx, &glacier.CreateVaultInput{
				AccountId: aws.String("-"),
				VaultName: aws.String("bench-vault"),
			})
		}),
	}}
}

// ─── Glue ───────────────────────────────────────────────────────────────────────

func glueSuite() harness.Suite {
	c := newCache(func(endpoint string) (*glue.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return glue.NewFromConfig(cfg, func(o *glue.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "glue", ops: []harness.Operation{
		op("GetDatabases", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.GetDatabases(ctx, &glue.GetDatabasesInput{})
		}),
		op("GetCrawlers", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.GetCrawlers(ctx, &glue.GetCrawlersInput{})
		}),
		op("GetJobs", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.GetJobs(ctx, &glue.GetJobsInput{})
		}),
	}}
}

// ─── Identity Store ────────────────────────────────────────────────────────────

func identitystoreSuite() harness.Suite {
	c := newCache(func(endpoint string) (*identitystore.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return identitystore.NewFromConfig(cfg, func(o *identitystore.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "identitystore", ops: []harness.Operation{
		op("ListUsers", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListUsers(ctx, &identitystore.ListUsersInput{
				IdentityStoreId: aws.String("d-0000000000"),
			})
		}),
		op("ListGroups", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListGroups(ctx, &identitystore.ListGroupsInput{
				IdentityStoreId: aws.String("d-0000000000"),
			})
		}),
	}}
}

// ─── IoT ────────────────────────────────────────────────────────────────────────

func iotSuite() harness.Suite {
	c := newCache(func(endpoint string) (*iot.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return iot.NewFromConfig(cfg, func(o *iot.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "iot", ops: []harness.Operation{
		op("ListThings", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListThings(ctx, &iot.ListThingsInput{})
		}),
		op("CreateThing", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.CreateThing(ctx, &iot.CreateThingInput{
				ThingName: aws.String("bench-thing"),
			})
		}),
	}}
}

// ─── IoT Data Plane ────────────────────────────────────────────────────────────

func iotdataSuite() harness.Suite {
	c := newCache(func(endpoint string) (*iotdataplane.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return iotdataplane.NewFromConfig(cfg, func(o *iotdataplane.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "iot-data", ops: []harness.Operation{
		op("ListRetainedMessages", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListRetainedMessages(ctx, &iotdataplane.ListRetainedMessagesInput{})
		}),
	}}
}

// ─── IoT Wireless ──────────────────────────────────────────────────────────────

func iotwirelessSuite() harness.Suite {
	c := newCache(func(endpoint string) (*iotwireless.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return iotwireless.NewFromConfig(cfg, func(o *iotwireless.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "iotwireless", ops: []harness.Operation{
		op("ListWirelessDevices", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListWirelessDevices(ctx, &iotwireless.ListWirelessDevicesInput{})
		}),
		op("ListWirelessGateways", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListWirelessGateways(ctx, &iotwireless.ListWirelessGatewaysInput{})
		}),
	}}
}

// ─── MSK (Kafka) ────────────────────────────────────────────────────────────────

func kafkaSuite() harness.Suite {
	c := newCache(func(endpoint string) (*kafka.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return kafka.NewFromConfig(cfg, func(o *kafka.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "kafka", ops: []harness.Operation{
		op("ListClustersV2", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListClustersV2(ctx, &kafka.ListClustersV2Input{})
		}),
	}}
}

// ─── Kinesis Analytics ─────────────────────────────────────────────────────────

func kinesisanalyticsSuite() harness.Suite {
	c := newCache(func(endpoint string) (*kinesisanalytics.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return kinesisanalytics.NewFromConfig(cfg, func(o *kinesisanalytics.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "kinesisanalytics", ops: []harness.Operation{
		op("ListApplications", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListApplications(ctx, &kinesisanalytics.ListApplicationsInput{})
		}),
	}}
}

// ─── Lake Formation ────────────────────────────────────────────────────────────

func lakeformationSuite() harness.Suite {
	c := newCache(func(endpoint string) (*lakeformation.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return lakeformation.NewFromConfig(cfg, func(o *lakeformation.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "lakeformation", ops: []harness.Operation{
		op("ListResources", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListResources(ctx, &lakeformation.ListResourcesInput{})
		}),
		op("ListPermissions", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListPermissions(ctx, &lakeformation.ListPermissionsInput{})
		}),
	}}
}

// ─── Managed Blockchain ────────────────────────────────────────────────────────

func managedblockchainSuite() harness.Suite {
	c := newCache(func(endpoint string) (*managedblockchain.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return managedblockchain.NewFromConfig(cfg, func(o *managedblockchain.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "managedblockchain", ops: []harness.Operation{
		op("ListNetworks", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListNetworks(ctx, &managedblockchain.ListNetworksInput{})
		}),
	}}
}

// ─── MediaConvert ──────────────────────────────────────────────────────────────

func mediaconvertSuite() harness.Suite {
	c := newCache(func(endpoint string) (*mediaconvert.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return mediaconvert.NewFromConfig(cfg, func(o *mediaconvert.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "mediaconvert", ops: []harness.Operation{
		op("ListJobs", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListJobs(ctx, &mediaconvert.ListJobsInput{})
		}),
		op("ListQueues", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListQueues(ctx, &mediaconvert.ListQueuesInput{})
		}),
	}}
}

// ─── MemoryDB ───────────────────────────────────────────────────────────────────

func memorydbSuite() harness.Suite {
	c := newCache(func(endpoint string) (*memorydb.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return memorydb.NewFromConfig(cfg, func(o *memorydb.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "memorydb", ops: []harness.Operation{
		op("DescribeClusters", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.DescribeClusters(ctx, &memorydb.DescribeClustersInput{})
		}),
	}}
}

// ─── Amazon MQ ──────────────────────────────────────────────────────────────────

func mqSuite() harness.Suite {
	c := newCache(func(endpoint string) (*mq.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return mq.NewFromConfig(cfg, func(o *mq.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "mq", ops: []harness.Operation{
		op("ListBrokers", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListBrokers(ctx, &mq.ListBrokersInput{})
		}),
	}}
}

// ─── Neptune ────────────────────────────────────────────────────────────────────

func neptuneSuite() harness.Suite {
	c := newCache(func(endpoint string) (*neptune.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return neptune.NewFromConfig(cfg, func(o *neptune.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "neptune", ops: []harness.Operation{
		op("DescribeDBInstances", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.DescribeDBInstances(ctx, &neptune.DescribeDBInstancesInput{})
		}),
		op("DescribeDBClusters", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.DescribeDBClusters(ctx, &neptune.DescribeDBClustersInput{})
		}),
	}}
}

// ─── OpenSearch ─────────────────────────────────────────────────────────────────

func opensearchSuite() harness.Suite {
	c := newCache(func(endpoint string) (*opensearch.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return opensearch.NewFromConfig(cfg, func(o *opensearch.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "opensearch", ops: []harness.Operation{
		op("ListDomainNames", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListDomainNames(ctx, &opensearch.ListDomainNamesInput{})
		}),
	}}
}

// ─── Organizations ──────────────────────────────────────────────────────────────

func organizationsSuite() harness.Suite {
	c := newCache(func(endpoint string) (*organizations.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return organizations.NewFromConfig(cfg, func(o *organizations.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "organizations", ops: []harness.Operation{
		op("ListAccounts", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListAccounts(ctx, &organizations.ListAccountsInput{})
		}),
		op("DescribeOrganization", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.DescribeOrganization(ctx, &organizations.DescribeOrganizationInput{})
		}),
	}}
}

// ─── Pinpoint ──────────────────────────────────────────────────────────────────

func pinpointSuite() harness.Suite {
	c := newCache(func(endpoint string) (*pinpoint.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return pinpoint.NewFromConfig(cfg, func(o *pinpoint.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "pinpoint", ops: []harness.Operation{
		op("GetApps", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.GetApps(ctx, &pinpoint.GetAppsInput{})
		}),
	}}
}

// ─── Pipes ──────────────────────────────────────────────────────────────────────

func pipesSuite() harness.Suite {
	c := newCache(func(endpoint string) (*pipes.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return pipes.NewFromConfig(cfg, func(o *pipes.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "pipes", ops: []harness.Operation{
		op("ListPipes", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListPipes(ctx, &pipes.ListPipesInput{})
		}),
	}}
}

// ─── RAM ────────────────────────────────────────────────────────────────────────

func ramSuite() harness.Suite {
	c := newCache(func(endpoint string) (*ram.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return ram.NewFromConfig(cfg, func(o *ram.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "ram", ops: []harness.Operation{
		op("GetResourceShares", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.GetResourceShares(ctx, &ram.GetResourceSharesInput{
				ResourceOwner: "SELF",
			})
		}),
	}}
}

// ─── Redshift ───────────────────────────────────────────────────────────────────

func redshiftSuite() harness.Suite {
	c := newCache(func(endpoint string) (*redshift.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return redshift.NewFromConfig(cfg, func(o *redshift.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "redshift", ops: []harness.Operation{
		op("DescribeClusters", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.DescribeClusters(ctx, &redshift.DescribeClustersInput{})
		}),
	}}
}

// ─── Rekognition ────────────────────────────────────────────────────────────────

func rekognitionSuite() harness.Suite {
	c := newCache(func(endpoint string) (*rekognition.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return rekognition.NewFromConfig(cfg, func(o *rekognition.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "rekognition", ops: []harness.Operation{
		op("ListCollections", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListCollections(ctx, &rekognition.ListCollectionsInput{})
		}),
		op("CreateCollection", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.CreateCollection(ctx, &rekognition.CreateCollectionInput{
				CollectionId: aws.String("bench-collection"),
			})
		}),
	}}
}

// ─── Resource Groups ────────────────────────────────────────────────────────────

func resourcegroupsSuite() harness.Suite {
	c := newCache(func(endpoint string) (*resourcegroups.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return resourcegroups.NewFromConfig(cfg, func(o *resourcegroups.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "resource-groups", ops: []harness.Operation{
		op("ListGroups", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListGroups(ctx, &resourcegroups.ListGroupsInput{})
		}),
	}}
}

// ─── Resource Groups Tagging API ────────────────────────────────────────────────

func resourcegroupstaggingapiSuite() harness.Suite {
	c := newCache(func(endpoint string) (*resourcegroupstaggingapi.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return resourcegroupstaggingapi.NewFromConfig(cfg, func(o *resourcegroupstaggingapi.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "tagging", ops: []harness.Operation{
		op("GetResources", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.GetResources(ctx, &resourcegroupstaggingapi.GetResourcesInput{})
		}),
	}}
}

// ─── Route 53 Resolver ─────────────────────────────────────────────────────────

func route53resolverSuite() harness.Suite {
	c := newCache(func(endpoint string) (*route53resolver.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return route53resolver.NewFromConfig(cfg, func(o *route53resolver.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "route53resolver", ops: []harness.Operation{
		op("ListResolverEndpoints", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListResolverEndpoints(ctx, &route53resolver.ListResolverEndpointsInput{})
		}),
		op("ListResolverRules", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListResolverRules(ctx, &route53resolver.ListResolverRulesInput{})
		}),
	}}
}

// ─── SageMaker ──────────────────────────────────────────────────────────────────

func sagemakerSuite() harness.Suite {
	c := newCache(func(endpoint string) (*sagemaker.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return sagemaker.NewFromConfig(cfg, func(o *sagemaker.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "sagemaker", ops: []harness.Operation{
		op("ListNotebookInstances", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListNotebookInstances(ctx, &sagemaker.ListNotebookInstancesInput{})
		}),
		op("ListEndpoints", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListEndpoints(ctx, &sagemaker.ListEndpointsInput{})
		}),
	}}
}

// ─── Scheduler ──────────────────────────────────────────────────────────────────

func schedulerSuite() harness.Suite {
	c := newCache(func(endpoint string) (*scheduler.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return scheduler.NewFromConfig(cfg, func(o *scheduler.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "scheduler", ops: []harness.Operation{
		op("ListSchedules", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListSchedules(ctx, &scheduler.ListSchedulesInput{})
		}),
		op("ListScheduleGroups", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListScheduleGroups(ctx, &scheduler.ListScheduleGroupsInput{})
		}),
	}}
}

// ─── Secrets Manager ────────────────────────────────────────────────────────────

func secretsmanagerSuite() harness.Suite {
	c := newCache(func(endpoint string) (*secretsmanager.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return secretsmanager.NewFromConfig(cfg, func(o *secretsmanager.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "secretsmanager", ops: []harness.Operation{
		op("ListSecrets", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListSecrets(ctx, &secretsmanager.ListSecretsInput{})
		}),
		op("CreateSecret", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
				Name:         aws.String(fmt.Sprintf("bench-secret-%s", "tier2")),
				SecretString: aws.String("bench-value"),
			})
		}),
	}}
}

// ─── Serverless Application Repository ─────────────────────────────────────────

func serverlessrepoSuite() harness.Suite {
	c := newCache(func(endpoint string) (*serverlessapplicationrepository.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return serverlessapplicationrepository.NewFromConfig(cfg, func(o *serverlessapplicationrepository.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "serverlessrepo", ops: []harness.Operation{
		op("ListApplications", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListApplications(ctx, &serverlessapplicationrepository.ListApplicationsInput{})
		}),
	}}
}

// ─── Service Discovery (Cloud Map) ──────────────────────────────────────────────

func servicediscoverySuite() harness.Suite {
	c := newCache(func(endpoint string) (*servicediscovery.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return servicediscovery.NewFromConfig(cfg, func(o *servicediscovery.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "servicediscovery", ops: []harness.Operation{
		op("ListNamespaces", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListNamespaces(ctx, &servicediscovery.ListNamespacesInput{})
		}),
		op("ListServices", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListServices(ctx, &servicediscovery.ListServicesInput{})
		}),
	}}
}

// ─── SES ───────────────────────────────────────────────────────────────────────

func sesSuite() harness.Suite {
	c := newCache(func(endpoint string) (*ses.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return ses.NewFromConfig(cfg, func(o *ses.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "ses", ops: []harness.Operation{
		op("ListIdentities", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListIdentities(ctx, &ses.ListIdentitiesInput{})
		}),
		op("ListVerifiedEmailAddresses", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListVerifiedEmailAddresses(ctx, &ses.ListVerifiedEmailAddressesInput{})
		}),
	}}
}

// ─── Step Functions ─────────────────────────────────────────────────────────────

func sfnSuite() harness.Suite {
	c := newCache(func(endpoint string) (*sfn.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return sfn.NewFromConfig(cfg, func(o *sfn.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "stepfunctions", ops: []harness.Operation{
		op("ListStateMachines", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListStateMachines(ctx, &sfn.ListStateMachinesInput{})
		}),
		op("ListActivities", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListActivities(ctx, &sfn.ListActivitiesInput{})
		}),
	}}
}

// ─── Shield ─────────────────────────────────────────────────────────────────────

func shieldSuite() harness.Suite {
	c := newCache(func(endpoint string) (*shield.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return shield.NewFromConfig(cfg, func(o *shield.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "shield", ops: []harness.Operation{
		op("ListProtections", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListProtections(ctx, &shield.ListProtectionsInput{})
		}),
	}}
}

// ─── SSO Admin ──────────────────────────────────────────────────────────────────

func ssoadminSuite() harness.Suite {
	c := newCache(func(endpoint string) (*ssoadmin.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return ssoadmin.NewFromConfig(cfg, func(o *ssoadmin.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "sso-admin", ops: []harness.Operation{
		op("ListInstances", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListInstances(ctx, &ssoadmin.ListInstancesInput{})
		}),
	}}
}

// ─── Support ────────────────────────────────────────────────────────────────────

func supportSuite() harness.Suite {
	c := newCache(func(endpoint string) (*support.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return support.NewFromConfig(cfg, func(o *support.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "support", ops: []harness.Operation{
		op("DescribeTrustedAdvisorChecks", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.DescribeTrustedAdvisorChecks(ctx, &support.DescribeTrustedAdvisorChecksInput{
				Language: aws.String("en"),
			})
		}),
	}}
}

// ─── SWF ────────────────────────────────────────────────────────────────────────

func swfSuite() harness.Suite {
	c := newCache(func(endpoint string) (*swf.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return swf.NewFromConfig(cfg, func(o *swf.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "swf", ops: []harness.Operation{
		op("ListDomains", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListDomains(ctx, &swf.ListDomainsInput{
				RegistrationStatus: "REGISTERED",
			})
		}),
	}}
}

// ─── Textract ──────────────────────────────────────────────────────────────────

func textractSuite() harness.Suite {
	c := newCache(func(endpoint string) (*textract.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return textract.NewFromConfig(cfg, func(o *textract.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "textract", ops: []harness.Operation{
		op("ListAdapterVersions", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListAdapterVersions(ctx, &textract.ListAdapterVersionsInput{})
		}),
		op("ListAdapters", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListAdapters(ctx, &textract.ListAdaptersInput{})
		}),
	}}
}

// ─── Timestream Write ───────────────────────────────────────────────────────────

func timestreamwriteSuite() harness.Suite {
	c := newCache(func(endpoint string) (*timestreamwrite.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return timestreamwrite.NewFromConfig(cfg, func(o *timestreamwrite.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "timestream-write", ops: []harness.Operation{
		op("ListDatabases", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListDatabases(ctx, &timestreamwrite.ListDatabasesInput{})
		}),
	}}
}

// ─── Transcribe ─────────────────────────────────────────────────────────────────

func transcribeSuite() harness.Suite {
	c := newCache(func(endpoint string) (*transcribe.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return transcribe.NewFromConfig(cfg, func(o *transcribe.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "transcribe", ops: []harness.Operation{
		op("ListTranscriptionJobs", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListTranscriptionJobs(ctx, &transcribe.ListTranscriptionJobsInput{})
		}),
	}}
}

// ─── Transfer ───────────────────────────────────────────────────────────────────

func transferSuite() harness.Suite {
	c := newCache(func(endpoint string) (*transfer.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return transfer.NewFromConfig(cfg, func(o *transfer.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "transfer", ops: []harness.Operation{
		op("ListServers", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListServers(ctx, &transfer.ListServersInput{})
		}),
	}}
}

// ─── Verified Permissions ──────────────────────────────────────────────────────

func verifiedpermissionsSuite() harness.Suite {
	c := newCache(func(endpoint string) (*verifiedpermissions.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return verifiedpermissions.NewFromConfig(cfg, func(o *verifiedpermissions.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "verifiedpermissions", ops: []harness.Operation{
		op("ListPolicyStores", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListPolicyStores(ctx, &verifiedpermissions.ListPolicyStoresInput{})
		}),
	}}
}

// ─── WAF Regional ──────────────────────────────────────────────────────────────

func wafregionalSuite() harness.Suite {
	c := newCache(func(endpoint string) (*wafregional.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return wafregional.NewFromConfig(cfg, func(o *wafregional.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "waf-regional", ops: []harness.Operation{
		op("ListWebACLs", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListWebACLs(ctx, &wafregional.ListWebACLsInput{})
		}),
		op("ListRules", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListRules(ctx, &wafregional.ListRulesInput{})
		}),
	}}
}

// ─── WAFv2 ──────────────────────────────────────────────────────────────────────

func wafv2Suite() harness.Suite {
	c := newCache(func(endpoint string) (*wafv2.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return wafv2.NewFromConfig(cfg, func(o *wafv2.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "wafv2", ops: []harness.Operation{
		op("ListWebACLs", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			return cl.ListWebACLs(ctx, &wafv2.ListWebACLsInput{
				Scope: wafv2types.ScopeRegional,
			})
		}),
	}}
}

// ─── X-Ray ──────────────────────────────────────────────────────────────────────

func xraySuite() harness.Suite {
	c := newCache(func(endpoint string) (*xray.Client, error) {
		cfg, err := awsclient.NewConfig(endpoint)
		if err != nil {
			return nil, err
		}
		return xray.NewFromConfig(cfg, func(o *xray.Options) { o.BaseEndpoint = awsclient.Endpoint(endpoint) }), nil
	})
	return &sdkSuite{name: "xray", ops: []harness.Operation{
		op("GetTraceSummaries", func(ctx context.Context, endpoint string) (any, error) {
			cl, err := c.get(endpoint)
			if err != nil {
				return nil, err
			}
			now := time.Now()
			return cl.GetTraceSummaries(ctx, &xray.GetTraceSummariesInput{
				StartTime: &now,
				EndTime:   &now,
			})
		}),
	}}
}
