package tier2

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/amplify"
	"github.com/aws/aws-sdk-go-v2/service/applicationautoscaling"
	"github.com/aws/aws-sdk-go-v2/service/appsync"
	"github.com/aws/aws-sdk-go-v2/service/athena"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/backup"
	"github.com/aws/aws-sdk-go-v2/service/batch"
	"github.com/aws/aws-sdk-go-v2/service/cloudcontrol"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/codecommit"
	"github.com/aws/aws-sdk-go-v2/service/codedeploy"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	cetypes "github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
	"github.com/aws/aws-sdk-go-v2/service/databasemigrationservice"
	"github.com/aws/aws-sdk-go-v2/service/datasync"
	"github.com/aws/aws-sdk-go-v2/service/docdb"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	"github.com/aws/aws-sdk-go-v2/service/elasticbeanstalk"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	"github.com/aws/aws-sdk-go-v2/service/emr"
	"github.com/aws/aws-sdk-go-v2/service/glacier"
	"github.com/aws/aws-sdk-go-v2/service/glue"
	"github.com/aws/aws-sdk-go-v2/service/iot"
	"github.com/aws/aws-sdk-go-v2/service/kafka"
	"github.com/aws/aws-sdk-go-v2/service/memorydb"
	"github.com/aws/aws-sdk-go-v2/service/mq"
	"github.com/aws/aws-sdk-go-v2/service/neptune"
	"github.com/aws/aws-sdk-go-v2/service/opensearch"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/pipes"
	"github.com/aws/aws-sdk-go-v2/service/ram"
	"github.com/aws/aws-sdk-go-v2/service/redshift"
	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroups"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/aws/aws-sdk-go-v2/service/shield"
	"github.com/aws/aws-sdk-go-v2/service/ssoadmin"
	"github.com/aws/aws-sdk-go-v2/service/support"
	"github.com/aws/aws-sdk-go-v2/service/swf"
	"github.com/aws/aws-sdk-go-v2/service/timestreamwrite"
	"github.com/aws/aws-sdk-go-v2/service/transcribe"
	"github.com/aws/aws-sdk-go-v2/service/transfer"
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
		acmSuite(),
		amplifySuite(),
		applicationAutoscalingSuite(),
		appsyncSuite(),
		athenaSuite(),
		autoscalingSuite(),
		backupSuite(),
		batchSuite(),
		cloudcontrolSuite(),
		cloudfrontSuite(),
		codecommitSuite(),
		codedeploySuite(),
		costexplorerSuite(),
		databasemigrationSuite(),
		datasyncSuite(),
		docdbSuite(),
		elasticacheSuite(),
		elasticbeanstalkSuite(),
		elasticloadbalancingSuite(),
		emrSuite(),
		glacierSuite(),
		glueSuite(),
		iotSuite(),
		kafkaSuite(),
		memorydbSuite(),
		mqSuite(),
		neptuneSuite(),
		opensearchSuite(),
		organizationsSuite(),
		pipesSuite(),
		ramSuite(),
		redshiftSuite(),
		rekognitionSuite(),
		resourcegroupsSuite(),
		resourcegroupstaggingapiSuite(),
		sagemakerSuite(),
		schedulerSuite(),
		secretsmanagerSuite(),
		servicediscoverySuite(),
		sfnSuite(),
		shieldSuite(),
		ssoadminSuite(),
		supportSuite(),
		swfSuite(),
		timestreamwriteSuite(),
		transcribeSuite(),
		transferSuite(),
		wafv2Suite(),
		xraySuite(),
	}
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
