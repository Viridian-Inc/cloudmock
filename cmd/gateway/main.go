package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/neureaux/cloudmock/pkg/admin"
	"github.com/neureaux/cloudmock/pkg/auth"
	authmemory "github.com/neureaux/cloudmock/pkg/auth/memory"
	authpg "github.com/neureaux/cloudmock/pkg/auth/postgres"
	"github.com/neureaux/cloudmock/pkg/ratelimit"
	saasclerk "github.com/neureaux/cloudmock/pkg/saas/clerk"
	"github.com/neureaux/cloudmock/pkg/saas/provisioning"
	"github.com/neureaux/cloudmock/pkg/saas/quota"
	saasstripe "github.com/neureaux/cloudmock/pkg/saas/stripe"
	"github.com/neureaux/cloudmock/pkg/saas/tenant"
	"github.com/neureaux/cloudmock/pkg/audit"
	auditmemory "github.com/neureaux/cloudmock/pkg/audit/memory"
	auditpg "github.com/neureaux/cloudmock/pkg/audit/postgres"
	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/cost"
	"github.com/neureaux/cloudmock/pkg/dashboard"
	"github.com/neureaux/cloudmock/pkg/dataplane"
	"github.com/neureaux/cloudmock/pkg/tenantscope"
	"github.com/neureaux/cloudmock/pkg/tracecompare"
	duckImpl "github.com/neureaux/cloudmock/pkg/dataplane/duckdb"
	"github.com/neureaux/cloudmock/pkg/dataplane/memory"
	pgImpl "github.com/neureaux/cloudmock/pkg/dataplane/postgres"
	promImpl "github.com/neureaux/cloudmock/pkg/dataplane/prometheus"
	"github.com/neureaux/cloudmock/pkg/eventbus"
	"github.com/neureaux/cloudmock/pkg/profiling"
	"github.com/neureaux/cloudmock/pkg/gateway"
	"github.com/neureaux/cloudmock/pkg/incident"
	incfilestore "github.com/neureaux/cloudmock/pkg/incident/filestore"
	incmemory "github.com/neureaux/cloudmock/pkg/incident/memory"
	"github.com/neureaux/cloudmock/pkg/otlp"
	incpg "github.com/neureaux/cloudmock/pkg/incident/postgres"
	"github.com/neureaux/cloudmock/pkg/monitor"
	notifypkg "github.com/neureaux/cloudmock/pkg/notify"
	monfilestore "github.com/neureaux/cloudmock/pkg/monitor/filestore"
	"github.com/neureaux/cloudmock/pkg/regression"
	"github.com/neureaux/cloudmock/pkg/report"
	"github.com/neureaux/cloudmock/pkg/webhook"
	whmemory "github.com/neureaux/cloudmock/pkg/webhook/memory"
	whpg "github.com/neureaux/cloudmock/pkg/webhook/postgres"
	regmemory "github.com/neureaux/cloudmock/pkg/regression/memory"
	regpg "github.com/neureaux/cloudmock/pkg/regression/postgres"
	errsfilestore "github.com/neureaux/cloudmock/pkg/errors/filestore"
	logsfilestore "github.com/neureaux/cloudmock/pkg/logstore/filestore"
	annotationspkg "github.com/neureaux/cloudmock/pkg/annotations"
	anomalypkg "github.com/neureaux/cloudmock/pkg/anomaly"
	cicdfilestore "github.com/neureaux/cloudmock/pkg/cicd/filestore"
	replayfilestore "github.com/neureaux/cloudmock/pkg/replay/filestore"
	"github.com/neureaux/cloudmock/pkg/filestore"
	rumpkg "github.com/neureaux/cloudmock/pkg/rum"
	rumfilestore "github.com/neureaux/cloudmock/pkg/rum/filestore"
	uptimepkg "github.com/neureaux/cloudmock/pkg/uptime"
	uptimefilestore "github.com/neureaux/cloudmock/pkg/uptime/filestore"
	"github.com/neureaux/cloudmock/pkg/marketplace"
	"github.com/neureaux/cloudmock/pkg/security"
	"github.com/neureaux/cloudmock/pkg/synthetics"
	"github.com/neureaux/cloudmock/pkg/worker"
	"github.com/neureaux/cloudmock/pkg/iac"
	iampkg "github.com/neureaux/cloudmock/pkg/iam"
	trafficpkg "github.com/neureaux/cloudmock/pkg/traffic"
	trafficfilestore "github.com/neureaux/cloudmock/pkg/traffic/filestore"
	"github.com/neureaux/cloudmock/pkg/integration"
	"github.com/neureaux/cloudmock/pkg/plugin"
	argoplugin "github.com/neureaux/cloudmock/plugins/argocd"
	k8splugin "github.com/neureaux/cloudmock/plugins/kubernetes"
	"github.com/neureaux/cloudmock/pkg/routing"
	"github.com/neureaux/cloudmock/pkg/service"
	apigwsvc "github.com/neureaux/cloudmock/services/apigateway"
	ec2svc "github.com/neureaux/cloudmock/services/ec2"
	cfnsvc "github.com/neureaux/cloudmock/services/cloudformation"
	cwsvc "github.com/neureaux/cloudmock/services/cloudwatch"
	logssvc "github.com/neureaux/cloudmock/services/cloudwatchlogs"
	cognitosvc "github.com/neureaux/cloudmock/services/cognito"
	dynamodbsvc "github.com/neureaux/cloudmock/services/dynamodb"
	ebsvc "github.com/neureaux/cloudmock/services/eventbridge"
	ecrsvc "github.com/neureaux/cloudmock/services/ecr"
	ecssvc "github.com/neureaux/cloudmock/services/ecs"
	firehosesvc "github.com/neureaux/cloudmock/services/firehose"
	kinesissvc "github.com/neureaux/cloudmock/services/kinesis"
	kmssvc "github.com/neureaux/cloudmock/services/kms"
	rdssvc "github.com/neureaux/cloudmock/services/rds"
	r53svc "github.com/neureaux/cloudmock/services/route53"
	s3svc "github.com/neureaux/cloudmock/services/s3"
	secretssvc "github.com/neureaux/cloudmock/services/secretsmanager"
	sessvc "github.com/neureaux/cloudmock/services/ses"
	snssvc "github.com/neureaux/cloudmock/services/sns"
	sqssvc "github.com/neureaux/cloudmock/services/sqs"
	ssmsvc "github.com/neureaux/cloudmock/services/ssm"
	stssvc "github.com/neureaux/cloudmock/services/sts"
	iamsvc "github.com/neureaux/cloudmock/services/iam"
	lambdasvc "github.com/neureaux/cloudmock/services/lambda"
	sfnsvc "github.com/neureaux/cloudmock/services/stepfunctions"

	// Promoted Tier 1 services (formerly Tier 2 stubs)
	accountsvc "github.com/neureaux/cloudmock/services/account"
	acmsvc "github.com/neureaux/cloudmock/services/acm"
	acmpcasvc "github.com/neureaux/cloudmock/services/acmpca"
	airflowsvc "github.com/neureaux/cloudmock/services/airflow"
	amplifysvc "github.com/neureaux/cloudmock/services/amplify"
	appconfigsvc "github.com/neureaux/cloudmock/services/appconfig"
	appautoscalingsvc "github.com/neureaux/cloudmock/services/applicationautoscaling"
	appsyncsvc "github.com/neureaux/cloudmock/services/appsync"
	athenasvc "github.com/neureaux/cloudmock/services/athena"
	autoscalingsvc "github.com/neureaux/cloudmock/services/autoscaling"
	backupsvc "github.com/neureaux/cloudmock/services/backup"
	batchsvc "github.com/neureaux/cloudmock/services/batch"
	bedrocksvc "github.com/neureaux/cloudmock/services/bedrock"
	cesvc "github.com/neureaux/cloudmock/services/ce"
	cloudcontrolsvc "github.com/neureaux/cloudmock/services/cloudcontrol"
	cloudfrontsvc "github.com/neureaux/cloudmock/services/cloudfront"
	cloudtrailsvc "github.com/neureaux/cloudmock/services/cloudtrail"
	codeartifactsvc "github.com/neureaux/cloudmock/services/codeartifact"
	codebuildsvc "github.com/neureaux/cloudmock/services/codebuild"
	codecommitsvc "github.com/neureaux/cloudmock/services/codecommit"
	codeconnectionssvc "github.com/neureaux/cloudmock/services/codeconnections"
	codedeploysvc "github.com/neureaux/cloudmock/services/codedeploy"
	codepipelinesvc "github.com/neureaux/cloudmock/services/codepipeline"
	configsvc "github.com/neureaux/cloudmock/services/config"
	dmssvc "github.com/neureaux/cloudmock/services/dms"
	docdbsvc "github.com/neureaux/cloudmock/services/docdb"
	ekssvcsvc "github.com/neureaux/cloudmock/services/eks"
	elasticachesvc "github.com/neureaux/cloudmock/services/elasticache"
	ebsvc2 "github.com/neureaux/cloudmock/services/elasticbeanstalk"
	elbsvc "github.com/neureaux/cloudmock/services/elasticloadbalancing"
	emrsvc "github.com/neureaux/cloudmock/services/elasticmapreduce"
	essvc "github.com/neureaux/cloudmock/services/es"
	fissvc "github.com/neureaux/cloudmock/services/fis"
	glaciersvc "github.com/neureaux/cloudmock/services/glacier"
	gluesvc "github.com/neureaux/cloudmock/services/glue"
	identitystoresvc "github.com/neureaux/cloudmock/services/identitystore"
	iotsvc "github.com/neureaux/cloudmock/services/iot"
	iotdatasvc "github.com/neureaux/cloudmock/services/iotdata"
	iotwirelesssvc "github.com/neureaux/cloudmock/services/iotwireless"
	kafkasvc "github.com/neureaux/cloudmock/services/kafka"
	kinesisanalyticssvc "github.com/neureaux/cloudmock/services/kinesisanalytics"
	lakeformationsvc "github.com/neureaux/cloudmock/services/lakeformation"
	managedblockchain "github.com/neureaux/cloudmock/services/managedblockchain"
	mediaconvertsvc "github.com/neureaux/cloudmock/services/mediaconvert"
	memorydbsvc "github.com/neureaux/cloudmock/services/memorydb"
	mqsvc "github.com/neureaux/cloudmock/services/mq"
	neptunesvc "github.com/neureaux/cloudmock/services/neptune"
	opensearchsvc "github.com/neureaux/cloudmock/services/opensearch"
	organizationssvc "github.com/neureaux/cloudmock/services/organizations"
	pinpointsvc "github.com/neureaux/cloudmock/services/pinpoint"
	pipessvc "github.com/neureaux/cloudmock/services/pipes"
	ramsvc "github.com/neureaux/cloudmock/services/ram"
	redshiftsvc "github.com/neureaux/cloudmock/services/redshift"
	resourcegroupssvc "github.com/neureaux/cloudmock/services/resourcegroups"
	route53resolversvc "github.com/neureaux/cloudmock/services/route53resolver"
	s3tablessvc "github.com/neureaux/cloudmock/services/s3tables"
	sagemakersvc "github.com/neureaux/cloudmock/services/sagemaker"
	schedulersvc "github.com/neureaux/cloudmock/services/scheduler"
	serverlessreposvc "github.com/neureaux/cloudmock/services/serverlessrepo"
	servicediscoverysvc "github.com/neureaux/cloudmock/services/servicediscovery"
	shieldsvc "github.com/neureaux/cloudmock/services/shield"
	ssoadminsvc "github.com/neureaux/cloudmock/services/ssoadmin"
	supportsvc "github.com/neureaux/cloudmock/services/support"
	swfsvc "github.com/neureaux/cloudmock/services/swf"
	taggingsvc "github.com/neureaux/cloudmock/services/tagging"
	textractsvc "github.com/neureaux/cloudmock/services/textract"
	timestreamwritesvc "github.com/neureaux/cloudmock/services/timestreamwrite"
	transcribesvc "github.com/neureaux/cloudmock/services/transcribe"
	transfersvc "github.com/neureaux/cloudmock/services/transfer"
	verifiedpermissionssvc "github.com/neureaux/cloudmock/services/verifiedpermissions"
	wafregionalsvc "github.com/neureaux/cloudmock/services/wafregional"
	wafv2svc "github.com/neureaux/cloudmock/services/wafv2"
)

func main() {
	// Initialize structured logging. JSON in production, text for local dev.
	logFormat := os.Getenv("CLOUDMOCK_LOG_FORMAT")
	var logHandler slog.Handler
	if logFormat == "json" {
		logHandler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	} else {
		logHandler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	}
	slog.SetDefault(slog.New(logHandler))

	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()

	configPath := flag.String("config", "", "path to cloudmock YAML config file")
	pluginDir := flag.String("plugin-dir", "", "directory containing external plugin binaries (default: ~/.cloudmock/plugins/)")
	iacDir := flag.String("iac", "", "path to Pulumi/Terraform project directory — auto-provisions DynamoDB tables, API routes from IaC source")
	iacEnv := flag.String("iac-env", "dev", "environment name for IaC resource name resolution (e.g., dev, stage, prod)")
	flag.Parse()

	var cfg *config.Config
	var err error

	if *configPath != "" {
		cfg, err = config.LoadFromFile(*configPath)
		if err != nil {
			log.Fatalf("failed to load config from %q: %v", *configPath, err)
		}
	} else {
		cfg = config.Default()
	}

	cfg.ApplyEnv()

	// IAM engine and credential store
	store := iampkg.NewStore(cfg.AccountID)
	store.InitRoot(cfg.IAM.RootAccessKey, cfg.IAM.RootSecretKey)
	engine := iampkg.NewEngine()

	// Event bus for cross-service communication
	bus := eventbus.NewBus()

	// Service registry
	registry := routing.NewRegistry()

	// Determine which Tier 1 services to eagerly initialize based on profile.
	// "minimal"  — only the 8 core services used by almost every app.
	// "standard" — all 97 Tier 1 services loaded eagerly.
	// "full"     — all 97 Tier 1 services loaded eagerly.
	profile := cfg.Profile

	// minimalSet is the set of Tier 1 service names always loaded eagerly.
	minimalSet := map[string]bool{
		"s3": true, "sts": true, "iam": true, "dynamodb": true,
		"sqs": true, "sns": true, "lambda": true, "logs": true,
	}

	eagerAll := profile == "standard" || profile == "full"

	// --- Always-eager: S3 with event bus support ---
	registry.Register(s3svc.NewWithBus(bus))

	registry.Register(stssvc.New(cfg.AccountID))

	// IAM service — always eager, shares engine and store with the gateway auth layer.
	registry.Register(iamsvc.New(cfg.AccountID, engine, store))

	// Tier 1 services that may be eager or lazy depending on profile.
	registerOrDefer := func(name string, factory func() service.Service) service.Service {
		if eagerAll || minimalSet[name] {
			svc := factory()
			registry.Register(svc)
			return svc
		}
		registry.RegisterLazy(name, factory)
		return nil
	}

	_ = registerOrDefer("kms", func() service.Service { return kmssvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("secretsmanager", func() service.Service { return secretssvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("ssm", func() service.Service { return ssmsvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("sqs", func() service.Service { return sqssvc.New(cfg.AccountID, cfg.Region) })

	// SES — keep a typed reference only when eager (needed by admin API).
	var sesService *sessvc.SESService
	if eagerAll || minimalSet["ses"] {
		sesService = sessvc.New(cfg.AccountID, cfg.Region)
		registry.Register(sesService)
	} else {
		registry.RegisterLazy("ses", func() service.Service { return sessvc.New(cfg.AccountID, cfg.Region) })
	}

	// SNS — needs a locator; wire it after all services are registered.
	var snsService *snssvc.SNSService
	if eagerAll || minimalSet["sns"] {
		snsService = snssvc.New(cfg.AccountID, cfg.Region)
		registry.Register(snsService)
	} else {
		registry.RegisterLazy("sns", func() service.Service {
			svc := snssvc.New(cfg.AccountID, cfg.Region)
			svc.SetLocator(registry)
			return svc
		})
	}

	_ = registerOrDefer("dynamodb", func() service.Service { return dynamodbsvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("logs", func() service.Service { return logssvc.New(cfg.AccountID, cfg.Region) })
	// CloudWatch — needs a locator for alarm → SNS delivery; wire after registration.
	var cwService *cwsvc.CloudWatchService
	if eagerAll || minimalSet["cloudwatch"] {
		cwService = cwsvc.New(cfg.AccountID, cfg.Region)
		registry.Register(cwService)
	} else {
		registry.RegisterLazy("monitoring", func() service.Service {
			svc := cwsvc.New(cfg.AccountID, cfg.Region)
			svc.SetLocator(registry)
			return svc
		})
	}
	_ = registerOrDefer("firehose", func() service.Service { return firehosesvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("kinesis", func() service.Service { return kinesissvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("route53", func() service.Service { return r53svc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("ecr", func() service.Service { return ecrsvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("ecs", func() service.Service { return ecssvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("cognito-idp", func() service.Service { return cognitosvc.New(cfg.AccountID, cfg.Region) })

	// EventBridge — needs a locator; wire it after all services are registered.
	var ebService *ebsvc.EventBridgeService
	if eagerAll || minimalSet["events"] {
		ebService = ebsvc.New(cfg.AccountID, cfg.Region)
		registry.Register(ebService)
	} else {
		registry.RegisterLazy("events", func() service.Service {
			svc := ebsvc.New(cfg.AccountID, cfg.Region)
			svc.SetLocator(registry)
			return svc
		})
	}

	_ = registerOrDefer("states", func() service.Service { return sfnsvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("rds", func() service.Service { return rdssvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("apigateway", func() service.Service { return apigwsvc.New(cfg.AccountID, cfg.Region) })
	// CloudFormation — needs a locator for resource provisioning; wire after registration.
	var cfnService *cfnsvc.CloudFormationService
	if eagerAll || minimalSet["cloudformation"] {
		cfnService = cfnsvc.New(cfg.AccountID, cfg.Region)
		registry.Register(cfnService)
	} else {
		registry.RegisterLazy("cloudformation", func() service.Service {
			svc := cfnsvc.New(cfg.AccountID, cfg.Region)
			svc.SetLocator(registry)
			return svc
		})
	}

	// Lambda — needs a locator; wire it after all services are registered.
	var lambdaService *lambdasvc.LambdaService
	if eagerAll || minimalSet["lambda"] {
		lambdaService = lambdasvc.New(cfg.AccountID, cfg.Region)
		registry.Register(lambdaService)
	} else {
		registry.RegisterLazy("lambda", func() service.Service {
			svc := lambdasvc.New(cfg.AccountID, cfg.Region)
			svc.SetLocator(registry)
			return svc
		})
	}

	_ = registerOrDefer("ec2", func() service.Service { return ec2svc.New(cfg.AccountID, cfg.Region) })

	// Set service locators now that all eager services are registered.
	// (This breaks the circular dependency: services need the registry,
	// but the registry needs the services.)
	if snsService != nil {
		snsService.SetLocator(registry)
	}
	if ebService != nil {
		ebService.SetLocator(registry)
	}
	if lambdaService != nil {
		lambdaService.SetLocator(registry)
	}
	if cwService != nil {
		cwService.SetLocator(registry)
	}
	if cfnService != nil {
		cfnService.SetLocator(registry)
	}

	// Wire cross-service integrations via event bus
	integration.WireIntegrations(bus, registry, cfg.AccountID, cfg.Region)

	// ── Promoted Tier 1 services (formerly Tier 2 stubs) ──────────────────────

	// Services with ServiceLocator (need SetLocator wired after registration)
	var elbService *elbsvc.ELBService
	if eagerAll {
		elbService = elbsvc.New(cfg.AccountID, cfg.Region)
		registry.Register(elbService)
	} else {
		registry.RegisterLazy("elasticloadbalancing", func() service.Service {
			svc := elbsvc.New(cfg.AccountID, cfg.Region)
			svc.SetLocator(registry)
			return svc
		})
	}

	var autoscalingService *autoscalingsvc.AutoScalingService
	if eagerAll {
		autoscalingService = autoscalingsvc.New(cfg.AccountID, cfg.Region)
		registry.Register(autoscalingService)
	} else {
		registry.RegisterLazy("autoscaling", func() service.Service {
			svc := autoscalingsvc.New(cfg.AccountID, cfg.Region)
			svc.SetLocator(registry)
			return svc
		})
	}

	var cloudfrontService *cloudfrontsvc.CloudFrontService
	if eagerAll {
		cloudfrontService = cloudfrontsvc.New(cfg.AccountID, cfg.Region)
		registry.Register(cloudfrontService)
	} else {
		registry.RegisterLazy("cloudfront", func() service.Service {
			svc := cloudfrontsvc.New(cfg.AccountID, cfg.Region)
			svc.SetLocator(registry)
			return svc
		})
	}

	var elasticacheService *elasticachesvc.ElastiCacheService
	if eagerAll {
		elasticacheService = elasticachesvc.New(cfg.AccountID, cfg.Region)
		registry.Register(elasticacheService)
	} else {
		registry.RegisterLazy("elasticache", func() service.Service {
			svc := elasticachesvc.New(cfg.AccountID, cfg.Region)
			svc.SetLocator(registry)
			return svc
		})
	}

	var eksService *ekssvcsvc.EKSService
	if eagerAll {
		eksService = ekssvcsvc.New(cfg.AccountID, cfg.Region)
		registry.Register(eksService)
	} else {
		registry.RegisterLazy("eks", func() service.Service {
			svc := ekssvcsvc.New(cfg.AccountID, cfg.Region)
			svc.SetLocator(registry)
			return svc
		})
	}

	var codebuildService *codebuildsvc.CodeBuildService
	if eagerAll {
		codebuildService = codebuildsvc.New(cfg.AccountID, cfg.Region)
		registry.Register(codebuildService)
	} else {
		registry.RegisterLazy("codebuild", func() service.Service {
			svc := codebuildsvc.New(cfg.AccountID, cfg.Region)
			svc.SetLocator(registry)
			return svc
		})
	}

	var codepipelineService *codepipelinesvc.CodePipelineService
	if eagerAll {
		codepipelineService = codepipelinesvc.New(cfg.AccountID, cfg.Region)
		registry.Register(codepipelineService)
	} else {
		registry.RegisterLazy("codepipeline", func() service.Service {
			svc := codepipelinesvc.New(cfg.AccountID, cfg.Region)
			svc.SetLocator(registry)
			return svc
		})
	}

	var codedeployService *codedeploysvc.CodeDeployService
	if eagerAll {
		codedeployService = codedeploysvc.New(cfg.AccountID, cfg.Region)
		registry.Register(codedeployService)
	} else {
		registry.RegisterLazy("codedeploy", func() service.Service {
			svc := codedeploysvc.New(cfg.AccountID, cfg.Region)
			svc.SetLocator(registry)
			return svc
		})
	}

	var codecommitService *codecommitsvc.CodeCommitService
	if eagerAll {
		codecommitService = codecommitsvc.New(cfg.AccountID, cfg.Region)
		registry.Register(codecommitService)
	} else {
		registry.RegisterLazy("codecommit", func() service.Service {
			svc := codecommitsvc.New(cfg.AccountID, cfg.Region)
			svc.SetLocator(registry)
			return svc
		})
	}

	var codeconnectionsService *codeconnectionssvc.CodeConnectionsService
	if eagerAll {
		codeconnectionsService = codeconnectionssvc.New(cfg.AccountID, cfg.Region)
		registry.Register(codeconnectionsService)
	} else {
		registry.RegisterLazy("codeconnections", func() service.Service {
			svc := codeconnectionssvc.New(cfg.AccountID, cfg.Region)
			svc.SetLocator(registry)
			return svc
		})
	}

	var codeartifactService *codeartifactsvc.CodeArtifactService
	if eagerAll {
		codeartifactService = codeartifactsvc.New(cfg.AccountID, cfg.Region)
		registry.Register(codeartifactService)
	} else {
		registry.RegisterLazy("codeartifact", func() service.Service {
			svc := codeartifactsvc.New(cfg.AccountID, cfg.Region)
			svc.SetLocator(registry)
			return svc
		})
	}

	// Wire locators for promoted services with cross-service integration
	if elbService != nil {
		elbService.SetLocator(registry)
	}
	if autoscalingService != nil {
		autoscalingService.SetLocator(registry)
		autoscalingService.SetEventBus(bus)
	}
	if cloudfrontService != nil {
		cloudfrontService.SetLocator(registry)
	}
	if elasticacheService != nil {
		elasticacheService.SetLocator(registry)
	}
	if eksService != nil {
		eksService.SetLocator(registry)
	}
	if codebuildService != nil {
		codebuildService.SetLocator(registry)
	}
	if codepipelineService != nil {
		codepipelineService.SetLocator(registry)
	}
	if codedeployService != nil {
		codedeployService.SetLocator(registry)
	}
	if codecommitService != nil {
		codecommitService.SetLocator(registry)
	}
	if codeconnectionsService != nil {
		codeconnectionsService.SetLocator(registry)
	}
	if codeartifactService != nil {
		codeartifactService.SetLocator(registry)
	}

	// Simple services (no cross-service locator needed) — registerOrDefer pattern
	_ = registerOrDefer("acm", func() service.Service { return acmsvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("acm-pca", func() service.Service { return acmpcasvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("cloudtrail", func() service.Service { return cloudtrailsvc.NewWithBus(cfg.AccountID, cfg.Region, bus) })
	_ = registerOrDefer("config", func() service.Service { return configsvc.NewWithBus(cfg.AccountID, cfg.Region, bus) })
	_ = registerOrDefer("organizations", func() service.Service { return organizationssvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("wafv2", func() service.Service { return wafv2svc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("waf-regional", func() service.Service { return wafregionalsvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("shield", func() service.Service { return shieldsvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("sso-admin", func() service.Service { return ssoadminsvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("verifiedpermissions", func() service.Service { return verifiedpermissionssvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("ram", func() service.Service { return ramsvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("athena", func() service.Service { return athenasvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("glue", func() service.Service { return gluesvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("redshift", func() service.Service { return redshiftsvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("elasticmapreduce", func() service.Service { return emrsvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("kinesisanalytics", func() service.Service { return kinesisanalyticssvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("lakeformation", func() service.Service { return lakeformationsvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("opensearch", func() service.Service { return opensearchsvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("es", func() service.Service { return essvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("timestream-write", func() service.Service { return timestreamwritesvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("neptune", func() service.Service { return neptunesvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("docdb", func() service.Service { return docdbsvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("memorydb", func() service.Service { return memorydbsvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("appconfig", func() service.Service { return appconfigsvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("application-autoscaling", func() service.Service { return appautoscalingsvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("servicediscovery", func() service.Service { return servicediscoverysvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("scheduler", func() service.Service { return schedulersvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("pipes", func() service.Service { return pipessvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("swf", func() service.Service { return swfsvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("appsync", func() service.Service { return appsyncsvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("amplify", func() service.Service { return amplifysvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("sagemaker", func() service.Service { return sagemakersvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("textract", func() service.Service { return textractsvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("transcribe", func() service.Service { return transcribesvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("bedrock", func() service.Service { return bedrocksvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("kafka", func() service.Service { return kafkasvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("mq", func() service.Service { return mqsvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("airflow", func() service.Service { return airflowsvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("iot", func() service.Service { return iotsvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("iot-data", func() service.Service { return iotdatasvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("iot-wireless", func() service.Service { return iotwirelesssvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("dms", func() service.Service { return dmssvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("transfer", func() service.Service { return transfersvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("glacier", func() service.Service { return glaciersvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("s3tables", func() service.Service { return s3tablessvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("cloudcontrol", func() service.Service { return cloudcontrolsvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("ce", func() service.Service { return cesvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("tagging", func() service.Service { return taggingsvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("support", func() service.Service { return supportsvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("backup", func() service.Service { return backupsvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("account", func() service.Service { return accountsvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("resource-groups", func() service.Service { return resourcegroupssvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("fis", func() service.Service { return fissvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("pinpoint", func() service.Service { return pinpointsvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("mediaconvert", func() service.Service { return mediaconvertsvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("managedblockchain", func() service.Service { return managedblockchain.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("serverlessrepo", func() service.Service { return serverlessreposvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("route53resolver", func() service.Service { return route53resolversvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("identitystore", func() service.Service { return identitystoresvc.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("elasticbeanstalk", func() service.Service { return ebsvc2.New(cfg.AccountID, cfg.Region) })
	_ = registerOrDefer("batch", func() service.Service { return batchsvc.New(cfg.AccountID, cfg.Region) })

	// Auto-provision resources from IaC source (Pulumi/Terraform).
	// This reads DynamoDB table definitions, API Gateway routes, etc. from the
	// IaC project directory and creates them in CloudMock — no seed scripts needed.
	var iacMicroservices []iac.MicroserviceDef
	if *iacDir != "" {
		iacResult, err := iac.ImportPulumiDir(*iacDir, *iacEnv, slog.Default())
		if err != nil {
			slog.Error("failed to import IaC", "dir", *iacDir, "error", err)
		} else {
			// Provision DynamoDB tables
			if dynamoSvc, lookupErr := registry.Lookup("dynamodb"); lookupErr == nil {
				iac.ProvisionDynamoTables(iacResult.Tables, dynamoSvc, slog.Default())
			}
			// Provision Lambda functions
			if lambdaSvc, lookupErr := registry.Lookup("lambda"); lookupErr == nil {
				iac.ProvisionLambdas(iacResult.Lambdas, lambdaSvc, cfg.AccountID, cfg.Region, slog.Default())
			}
			// Provision Cognito User Pools
			if cognitoSvc, lookupErr := registry.Lookup("cognito-idp"); lookupErr == nil {
				iac.ProvisionCognitoPools(iacResult.CognitoPools, cognitoSvc, slog.Default())
			}
			// Provision SQS Queues
			if sqsSvc, lookupErr := registry.Lookup("sqs"); lookupErr == nil {
				iac.ProvisionSQSQueues(iacResult.SQSQueues, sqsSvc, slog.Default())
			}
			// Provision SNS Topics
			if snsSvc, lookupErr := registry.Lookup("sns"); lookupErr == nil {
				iac.ProvisionSNSTopics(iacResult.SNSTopics, snsSvc, slog.Default())
			}
			// Provision S3 Buckets
			if s3Svc, lookupErr := registry.Lookup("s3"); lookupErr == nil {
				iac.ProvisionS3Buckets(iacResult.S3Buckets, s3Svc, slog.Default())
			}
			// Provision API Gateways
			if apigwSvc, lookupErr := registry.Lookup("apigateway"); lookupErr == nil {
				iac.ProvisionAPIGateways(iacResult.APIGateways, apigwSvc, slog.Default())
			}

			total := len(iacResult.Tables) + len(iacResult.Lambdas) + len(iacResult.CognitoPools) +
				len(iacResult.SQSQueues) + len(iacResult.SNSTopics) + len(iacResult.S3Buckets) + len(iacResult.APIGateways) +
				len(iacResult.Microservices)
			slog.Info("auto-provisioned resources from IaC", "dir", *iacDir, "env", *iacEnv,
				"tables", len(iacResult.Tables), "lambdas", len(iacResult.Lambdas),
				"cognito_pools", len(iacResult.CognitoPools), "sqs_queues", len(iacResult.SQSQueues),
				"sns_topics", len(iacResult.SNSTopics), "s3_buckets", len(iacResult.S3Buckets),
				"api_gateways", len(iacResult.APIGateways), "microservices", len(iacResult.Microservices),
				"total", total)

			// Store microservices for topology — will be set on adminAPI after it's created
			iacMicroservices = iacResult.Microservices
		}
	}

	// Determine DataPlane mode
	ctx := context.Background()

	mode := cfg.DataPlane.Mode
	if mode == "" {
		mode = "local"
	}

	var dp *dataplane.DataPlane
	var requestLog *gateway.RequestLog
	var requestStats *gateway.RequestStats
	var traceStore *gateway.TraceStore
	var sloEngine *gateway.SLOEngine

	// Production-mode clients, hoisted for use by the regression engine.
	var duckClient *duckImpl.Client
	var pgPool *pgxpool.Pool
	var promClient *promImpl.Client
	var otelShutdown func(context.Context) error

	switch mode {
	case "local":
		requestLog = gateway.NewRequestLog(1000)
		requestStats = gateway.NewRequestStats()
		traceStore = gateway.NewTraceStore(500)
		sloEngine = gateway.NewSLOEngine(cfg.SLO.Rules)

		dp = &dataplane.DataPlane{
			Traces:   memory.NewTraceStore(traceStore),
			TraceW:   memory.NewTraceStore(traceStore),
			Requests: memory.NewRequestStore(requestLog),
			RequestW: memory.NewRequestStore(requestLog),
			Metrics:  memory.NewMetricStore(requestStats, requestLog),
			MetricW:  memory.NewMetricStore(requestStats, requestLog),
			SLO:      memory.NewSLOStore(sloEngine),
			Config:      memory.NewConfigStore(cfg),
			Topology:    memory.NewTopologyStore(),
			Preferences: memory.NewPreferenceStore(),
			Mode:        "local",
		}
	case "production":
		var err error
		duckPath := cfg.DataPlane.DuckDB.Path
		if duckPath == "" {
			duckPath = "cloudmock.duckdb"
		}
		duckClient, err = duckImpl.NewClient(duckPath)
		if err != nil {
			log.Fatalf("duckdb: %v", err)
		}
		if err := duckClient.InitSchema(); err != nil {
			log.Fatalf("duckdb schema: %v", err)
		}

		pgPool, err = pgImpl.NewPool(ctx, cfg.DataPlane.PostgreSQL)
		if err != nil {
			log.Fatalf("postgres: %v", err)
		}

		promClient, err = promImpl.NewClient(cfg.DataPlane.Prometheus)
		if err != nil {
			log.Fatalf("prometheus: %v", err)
		}

		otelShutdown, err = dataplane.InitTracer(ctx, cfg.DataPlane.OTel)
		if err != nil {
			log.Fatalf("otel: %v", err)
		}

		duckTraces := duckImpl.NewTraceStore(duckClient)
		duckRequests := duckImpl.NewRequestStore(duckClient)

		dp = &dataplane.DataPlane{
			Traces:   duckTraces,
			TraceW:   duckTraces,
			Requests: duckRequests,
			RequestW: duckRequests,
			Metrics:  promImpl.NewMetricReader(promClient),
			MetricW:  promImpl.NewMetricWriter(),
			SLO:      pgImpl.NewSLOStore(pgPool),
			Config:      pgImpl.NewConfigStore(pgPool),
			Topology:    pgImpl.NewTopologyStore(pgPool),
			Preferences: pgImpl.NewPreferenceStore(pgPool),
			Mode:        "production",
		}
	default:
		log.Fatalf("unknown dataplane mode: %q", mode)
	}

	// Tenant isolation: when auth is enabled, wrap DataPlane readers so that
	// non-admin users only see data belonging to their tenant.
	if cfg.Auth.Enabled {
		dp.Traces = tenantscope.NewTraceReader(dp.Traces)
		dp.Requests = tenantscope.NewRequestReader(dp.Requests)
	}

	// Base directory for file-backed stores.
	baseDir := filepath.Join(os.Getenv("HOME"), ".cloudmock")

	// Chaos engine for fault injection — file-backed
	chaosStore, err := filestore.New[gateway.ChaosRule](filepath.Join(baseDir, "chaos"))
	if err != nil {
		slog.Error("failed to create chaos file store", "error", err)
	}
	var chaosRules []gateway.ChaosRule
	if chaosStore != nil {
		chaosRules, _ = chaosStore.List()
	}
	chaosEngine := gateway.NewChaosEngineWithRules(chaosRules)
	if chaosStore != nil {
		chaosEngine.PersistFunc = func(rules []gateway.ChaosRule) {
			// Delete all existing files, then save current rules.
			if existing, err := chaosStore.List(); err == nil {
				for _, r := range existing {
					_ = chaosStore.Delete(r.ID)
				}
			}
			for _, r := range rules {
				_ = chaosStore.Save(r.ID, r)
			}
		}
	}

	// Admin API (with CORS for dashboard cross-origin access)
	adminAPI := admin.NewWithDataPlane(cfg, registry, dp)
	// File-backed persistence for dashboards, saved views, and deploy events
	adminAPI.SetPersistDir(baseDir)
	// Also set the direct request log/stats for topology edge enrichment
	adminAPI.SetRequestLog(requestLog, requestStats)
	// Set IaC microservices for topology rendering
	if len(iacMicroservices) > 0 {
		adminAPI.SetMicroservices(iacMicroservices)
	}
	// Audit logger
	var auditLog audit.Logger
	switch mode {
	case "local":
		auditLog = auditmemory.NewLogger()
	case "production":
		auditLog = auditpg.NewLogger(pgPool)
	}
	adminAPI.SetAuditLogger(auditLog)

	// JWT-based RBAC auth (opt-in)
	if cfg.Auth.Enabled {
		var userStore auth.UserStore
		switch mode {
		case "local":
			userStore = authmemory.NewStore()
		case "production":
			userStore = authpg.NewStore(pgPool)
		}
		adminAPI.SetUserStore(userStore)
		adminAPI.SetAuthSecret([]byte(cfg.Auth.Secret))
	}

	adminAPI.SetChaosEngine(chaosEngine)

	profileDir := filepath.Join(os.TempDir(), "cloudmock-profiles")
	os.MkdirAll(profileDir, 0755)
	profEngine := profiling.New(profileDir, 100)
	symbolizer := profiling.NewSymbolizer()
	adminAPI.SetProfilingEngine(profEngine)
	adminAPI.SetSymbolizer(symbolizer)

	costEngine := cost.New(dp.Requests, cfg.Cost.Pricing)
	adminAPI.SetCostEngine(costEngine)

	tc := tracecompare.New(dp.Traces)
	adminAPI.SetTraceComparer(tc)

	// RUM (Real User Monitoring) engine — file-backed
	if cfg.RUM.Enabled {
		rumStore, rumErr := rumfilestore.New(filepath.Join(baseDir, "rum"), cfg.RUM.MaxEvents)
		if rumErr != nil {
			slog.Error("failed to create RUM file store, falling back to memory", "error", rumErr)
		}
		if rumStore != nil {
			rumEngine := rumpkg.New(rumStore, rumpkg.EngineConfig{
				SampleRate: cfg.RUM.SampleRate,
				MaxEvents:  cfg.RUM.MaxEvents,
			})
			adminAPI.SetRUMEngine(rumEngine)
		}
		slog.Info("RUM engine enabled", "sample_rate", cfg.RUM.SampleRate, "max_events", cfg.RUM.MaxEvents)
	}

	// Session Replay — file-backed
	replayStore, replayErr := replayfilestore.New(filepath.Join(baseDir, "replay"))
	if replayErr != nil {
		slog.Error("failed to create replay file store", "error", replayErr)
	}
	if replayStore != nil {
		adminAPI.SetReplayStore(replayStore)
	}
	slog.Info("session replay store initialized", "storage", filepath.Join(baseDir, "replay"))

	// ML-Powered Anomaly Detection — learns baselines and detects deviations.
	anomalyDetector := anomalypkg.NewDetector(7*24*time.Hour, 2.0)
	adminAPI.SetAnomalyDetector(anomalyDetector)
	slog.Info("anomaly detector initialized", "window", "7d", "threshold", 2.0)

	// Uptime / endpoint monitoring — file-backed
	{
		uptimeStore, uptimeErr := uptimefilestore.New(filepath.Join(baseDir, "uptime"), 1000)
		if uptimeErr != nil {
			slog.Error("failed to create uptime file store", "error", uptimeErr)
		}
		if uptimeStore != nil {
			workerPool := worker.NewPool(rootCtx, nil)
			uptimeEngine := uptimepkg.NewEngine(uptimeStore, workerPool, uptimepkg.DefaultEngineConfig())
			uptimeEngine.StartAll()
			adminAPI.SetUptimeEngine(uptimeEngine)
		}
		slog.Info("uptime monitoring engine initialized", "storage", filepath.Join(baseDir, "uptime"))
	}

	// Structured error tracking — file-backed
	errStore, errStoreErr := errsfilestore.New(filepath.Join(baseDir, "errors"), 10000)
	if errStoreErr != nil {
		slog.Error("failed to create error file store", "error", errStoreErr)
	}
	if errStore != nil {
		adminAPI.SetErrorStore(errStore)
	}
	slog.Info("error tracking store initialized", "storage", filepath.Join(baseDir, "errors"))

	// Log management — file-backed
	logStore, logStoreErr := logsfilestore.New(filepath.Join(baseDir, "logs"), 50000)
	if logStoreErr != nil {
		slog.Error("failed to create log file store", "error", logStoreErr)
	}
	if logStore != nil {
		adminAPI.SetLogStore(logStore)
	}
	slog.Info("log store initialized", "storage", filepath.Join(baseDir, "logs"))

	// Wire Lambda logs, IAM engine, and SES store to admin API.
	// lambdaService and sesService may be nil when running in minimal profile
	// (they are registered lazily). In that case, skip optional admin wiring.
	if lambdaService != nil {
		adminAPI.SetLambdaLogs(lambdaService.Logs())
	}
	adminAPI.SetIAMEngine(engine)
	if sesService != nil {
		adminAPI.SetSESStore(sesService.GetStore())
	}

	// Regression detection engine
	var regEngine *regression.Engine
	if cfg.Regression.Enabled {
		var regStore regression.RegressionStore
		var regSource regression.MetricSource

		switch mode {
		case "local":
			regStore = regmemory.NewStore()
			regSource = regmemory.NewMetricSource(requestLog, traceStore)
		case "production":
			regStore = regpg.NewStore(pgPool)
			regSource = regression.NewMetricSource(promClient.API(), duckClient.DB())
		}

		scanInterval, _ := time.ParseDuration(cfg.Regression.ScanInterval)
		if scanInterval == 0 {
			scanInterval = 5 * time.Minute
		}
		window, _ := time.ParseDuration(cfg.Regression.Window)
		if window == 0 {
			window = 15 * time.Minute
		}

		regEngine = regression.New(regSource, regStore, dp.Config, regression.DefaultAlgorithmConfig(), scanInterval, window)
		regEngine.Start(ctx)

		adminAPI.SetRegressionEngine(regEngine)
	}

	// Incident management service
	if cfg.Incidents.Enabled {
		var incStore incident.IncidentStore
		switch mode {
		case "local":
			if fs, err := incfilestore.New(filepath.Join(baseDir, "incidents")); err == nil {
				incStore = fs
			} else {
				slog.Error("failed to create incident file store, falling back to memory", "error", err)
				incStore = incmemory.NewStore()
			}
		case "production":
			incStore = incpg.NewStore(pgPool)
		}

		groupWindow, _ := time.ParseDuration(cfg.Incidents.GroupWindow)
		if groupWindow == 0 {
			groupWindow = 5 * time.Minute
		}

		var regStore regression.RegressionStore
		if regEngine != nil {
			regStore = regEngine.Store()
		}
		incService := incident.NewService(incStore, regStore, groupWindow)

		// Wire callbacks
		if regEngine != nil {
			regEngine.SetAlertCallback(func(ctx context.Context, r regression.Regression) {
				incService.OnRegression(ctx, r)
			})
		}
		if sloEngine != nil {
			sloEngine.SetAlertFunc(func(service, action string, burnRate, budgetUsed float64) {
				incService.OnSLOBreach(rootCtx, service, action, burnRate, budgetUsed)
			})
		}

		adminAPI.SetIncidentService(incService)

		var reportRegStore regression.RegressionStore
		if regEngine != nil {
			reportRegStore = regEngine.Store()
		}
		reportGen := report.New(incService.Store(), reportRegStore, dp.Traces)
		adminAPI.SetReportGenerator(reportGen)

		// Webhook dispatcher — wired inside the incident block so it is always
		// co-located with the incident service.
		var whStore webhook.Store
		switch mode {
		case "local":
			whStore = whmemory.NewStore()
		case "production":
			whStore = whpg.NewStore(pgPool)
		}
		whDispatcher := webhook.NewDispatcher(whStore)
		adminAPI.SetWebhookDispatcher(whDispatcher)
		incService.SetWebhookDispatcher(whDispatcher)

		// Notification router — smart alert routing to Slack, PagerDuty, email
		notifyRouter := notifypkg.NewRouter()

		// Load channels and routes from config
		if len(cfg.Notifications.Channels) > 0 {
			var channelRefs []notifypkg.ChannelRef
			channelConfigMap := make(map[string]config.NotifyChannelConfig)
			for _, ch := range cfg.Notifications.Channels {
				ref := notifypkg.ChannelRef{Type: ch.Type, Name: ch.Name, Config: make(map[string]string)}
				switch ch.Type {
				case "slack":
					ref.Config["webhook_url"] = ch.WebhookURL
				case "pagerduty":
					ref.Config["routing_key"] = ch.RoutingKey
				case "email":
					ref.Config["smtp_host"] = ch.SMTPHost
					ref.Config["smtp_port"] = strconv.Itoa(ch.SMTPPort)
					ref.Config["username"] = ch.Username
					ref.Config["password"] = ch.Password
					ref.Config["from"] = ch.From
					ref.Config["to"] = ch.To
				}
				channelRefs = append(channelRefs, ref)
				channelConfigMap[ch.Name] = ch
			}
			notifyRouter.LoadChannels(channelRefs)

			// Load routes from config, resolving channel names to refs
			var routes []notifypkg.Route
			for _, rc := range cfg.Notifications.Routes {
				route := notifypkg.Route{
					Name:    rc.Name,
					Enabled: true,
					Match: notifypkg.RouteMatch{
						Services:   rc.Match.Services,
						Severities: rc.Match.Severities,
						Types:      rc.Match.Types,
					},
				}
				for _, chName := range rc.Channels {
					if chCfg, ok := channelConfigMap[chName]; ok {
						ref := notifypkg.ChannelRef{Type: chCfg.Type, Name: chCfg.Name}
						route.Channels = append(route.Channels, ref)
					}
				}
				routes = append(routes, route)
			}
			notifyRouter.LoadRoutes(routes)
			slog.Info("notification routing configured", "channels", len(cfg.Notifications.Channels), "routes", len(cfg.Notifications.Routes))
		}

		// Wire notification router into existing alert callbacks.
		// The callbacks above are overwritten to also route through the
		// notification system in addition to incident correlation.
		if regEngine != nil {
			regEngine.SetAlertCallback(func(ctx context.Context, r regression.Regression) {
				incService.OnRegression(ctx, r)
				notifyRouter.Notify(ctx, notifypkg.Notification{
					Title:     r.Title,
					Severity:  string(r.Severity),
					Service:   r.Service,
					Type:      "regression",
					Timestamp: r.DetectedAt,
					Fields: map[string]string{
						"Algorithm":  string(r.Algorithm),
						"Confidence": strconv.Itoa(r.Confidence),
						"Deploy":     r.DeployID,
					},
				})
			})
		}
		if sloEngine != nil {
			sloEngine.SetAlertFunc(func(service, action string, burnRate, budgetUsed float64) {
				incService.OnSLOBreach(rootCtx, service, action, burnRate, budgetUsed)
				severity := "warning"
				if budgetUsed > 0.9 {
					severity = "critical"
				}
				notifyRouter.Notify(rootCtx, notifypkg.Notification{
					Title:     fmt.Sprintf("SLO burn rate alert: %s/%s", service, action),
					Message:   fmt.Sprintf("%.0f%% of error budget consumed, burn rate %.1fx", budgetUsed*100, burnRate),
					Severity:  severity,
					Service:   service,
					Type:      "slo_breach",
					Timestamp: time.Now(),
					Fields: map[string]string{
						"Action":      action,
						"Burn Rate":   fmt.Sprintf("%.1f", burnRate),
						"Budget Used": fmt.Sprintf("%.0f%%", budgetUsed*100),
					},
				})
			})
		}

		adminAPI.SetNotificationRouter(notifyRouter)
	}

	// Monitoring and alerting service — file-backed
	if cfg.Monitor.Enabled {
		monStore, monErr := monfilestore.New(
			filepath.Join(baseDir, "monitors"),
			filepath.Join(baseDir, "alerts"),
		)
		if monErr != nil {
			slog.Error("failed to create monitor file store", "error", monErr)
		}

		var provider monitor.MetricsProvider
		if sloEngine != nil {
			provider = monitor.NewGatewayProvider(sloEngine, requestStats)
		}

		evalInterval, _ := time.ParseDuration(cfg.Monitor.EvalInterval)
		if evalInterval == 0 {
			evalInterval = 30 * time.Second
		}

		if monStore != nil {
			monService := monitor.NewService(monStore, monStore, provider, evalInterval)
			monService.Start(rootCtx)
			adminAPI.SetMonitorService(monService)
		}
		slog.Info("monitor service started", "eval_interval", evalInterval)
	}

	// Traffic simulator / replay engine — file-backed so recordings survive restarts
	trafficDir := filepath.Join(baseDir, "traffic")
	trafficStore, err := trafficfilestore.New(trafficDir)
	if err != nil {
		slog.Error("failed to create traffic store", "error", err, "dir", trafficDir)
		trafficStore, _ = trafficfilestore.New(os.TempDir())
	}
	trafficEng := trafficpkg.New(trafficStore, requestLog, cfg.Gateway.Port)
	adminAPI.SetTrafficEngine(trafficEng)
	slog.Info("traffic replay engine initialized", "storage", trafficDir)

	// Annotations store — file-backed
	{
		annotationsDir := filepath.Join(baseDir, "annotations")
		annStore, _ := filestore.New[annotationspkg.Annotation](annotationsDir)
		var existing []annotationspkg.Annotation
		if annStore != nil {
			existing, _ = annStore.List()
		}
		annotationStore := annotationspkg.NewStoreWithData(existing)
		if annStore != nil {
			annotationStore.PersistFunc = func(annotations []annotationspkg.Annotation) {
				// Delete all, then save current.
				if old, err := annStore.List(); err == nil {
					for _, a := range old {
						_ = annStore.Delete(a.ID)
					}
				}
				for _, a := range annotations {
					_ = annStore.Save(a.ID, a)
				}
			}
		}
		adminAPI.SetAnnotationStore(annotationStore)
		slog.Info("annotation store initialized", "storage", annotationsDir)
	}

	// CI/CD visibility store — file-backed
	{
		cs, csErr := cicdfilestore.New(filepath.Join(baseDir, "cicd"))
		if csErr != nil {
			slog.Error("failed to create CI/CD file store", "error", csErr)
		}
		if cs != nil {
			adminAPI.SetCICDStore(cs)
		}
		slog.Info("CI/CD store initialized", "storage", filepath.Join(baseDir, "cicd"))
	}

	// Synthetic browser/HTTP tests — file-backed
	{
		synthDir := filepath.Join(baseDir, "synthetics")
		synthTestStore, _ := filestore.New[synthetics.SyntheticTest](filepath.Join(synthDir, "tests"))
		synthResultStore, _ := filestore.New[map[string][]synthetics.TestResult](filepath.Join(synthDir, "results"))

		var existingTests []synthetics.SyntheticTest
		existingResults := make(map[string][]synthetics.TestResult)
		if synthTestStore != nil {
			existingTests, _ = synthTestStore.List()
		}
		if synthResultStore != nil {
			if r, err := synthResultStore.Get("all"); err == nil {
				existingResults = r
			}
		}

		synthStore := synthetics.NewStoreWithData(500, existingTests, existingResults)
		if synthTestStore != nil && synthResultStore != nil {
			synthStore.PersistFunc = func(tests []synthetics.SyntheticTest, results map[string][]synthetics.TestResult) {
				// Save tests individually.
				if old, err := synthTestStore.List(); err == nil {
					for _, t := range old {
						_ = synthTestStore.Delete(t.ID)
					}
				}
				for _, t := range tests {
					_ = synthTestStore.Save(t.ID, t)
				}
				// Save all results as a single blob.
				_ = synthResultStore.Save("all", results)
			}
		}

		synthWorkerPool := worker.NewPool(rootCtx, nil)
		synthEngine := synthetics.NewEngine(synthStore, synthWorkerPool)
		synthEngine.StartAll()
		adminAPI.SetSyntheticsEngine(synthEngine)
		slog.Info("synthetics engine initialized", "storage", synthDir)
	}

	// Security posture scanner — checks mock resources for misconfigurations.
	{
		secScanner := security.NewScanner(registry)
		adminAPI.SetSecurityScanner(secScanner)
		slog.Info("security scanner initialized")
	}

	// Plugin marketplace — search and (placeholder) install community plugins.
	{
		mpRegistry := marketplace.NewRegistry()
		adminAPI.SetMarketplace(mpRegistry)
		slog.Info("marketplace initialized", "listings", len(mpRegistry.List()))
	}

	// Plugin manager — enables hybrid in-process / external plugin routing.
	pluginMgr := plugin.NewManager(slog.Default())
	adminAPI.SetPluginManager(pluginMgr)

	// Bridge all registered services (eager + lazy) from the legacy registry
	// into the plugin system via ServiceAdapter. This makes every AWS service
	// available through the unified plugin interface while keeping the legacy
	// registry as a fallback.
	for _, svc := range registry.List() {
		adapter := plugin.NewServiceAdapter(svc, cfg.Region, cfg.AccountID)
		if err := pluginMgr.RegisterServiceAdapter(rootCtx, adapter); err != nil {
			slog.Warn("failed to register service as plugin", "service", svc.Name(), "error", err)
		}
	}
	slog.Info("bridged legacy services to plugin system", "count", len(pluginMgr.Names()))

	// Register Kubernetes API emulation plugin.
	k8sPlugin := k8splugin.New()
	if err := pluginMgr.RegisterInProcess(rootCtx, k8sPlugin); err != nil {
		slog.Error("failed to register kubernetes plugin", "error", err)
	}

	// Register ArgoCD API emulation plugin, wired to k8s for sync operations.
	argoPlugin := argoplugin.New(k8sPlugin)
	if err := pluginMgr.RegisterInProcess(rootCtx, argoPlugin); err != nil {
		slog.Error("failed to register argocd plugin", "error", err)
	}

	// Load external plugins from filesystem.
	extPluginDir := *pluginDir
	if extPluginDir == "" {
		home, _ := os.UserHomeDir()
		extPluginDir = filepath.Join(home, ".cloudmock", "plugins")
	}
	if err := plugin.LoadExternalPlugins(rootCtx, pluginMgr, extPluginDir, slog.Default()); err != nil {
		slog.Warn("failed to load external plugins", "dir", extPluginDir, "error", err)
	}

	// --- SaaS hosted-tier wiring ---
	var quotaMiddleware *quota.Middleware
	if cfg.SaaS.Enabled {
		slog.Info("SaaS mode enabled, initializing hosted-tier components")

		// 1. Tenant store (memory or postgres based on dataplane mode).
		var tenantStore tenant.Store
		switch mode {
		case "production":
			tenantStore = tenant.NewPostgresStore(pgPool)
		default:
			tenantStore = tenant.NewMemoryStore()
		}
		adminAPI.SetTenantStore(tenantStore)

		// 2. Clerk webhook handler and JWT verifier.
		var userStore auth.UserStore
		if cfg.Auth.Enabled {
			// Reuse the user store already created for auth.
			// The adminAPI already has it set; we just need a reference
			// for the Clerk webhook handler.
			switch mode {
			case "production":
				userStore = authpg.NewStore(pgPool)
			default:
				userStore = authmemory.NewStore()
			}
		}
		clerkWH := saasclerk.NewWebhookHandler(
			tenantStore, userStore,
			cfg.SaaS.Clerk.WebhookSecret, slog.Default(),
		)
		adminAPI.SetClerkWebhook(clerkWH)

		// Clerk JWT verifier (for authenticating SaaS requests).
		if cfg.SaaS.Clerk.Domain != "" {
			clerkVerifier := saasclerk.NewJWTVerifier(cfg.SaaS.Clerk.Domain, slog.Default())
			_ = clerkVerifier // Available for auth middleware integration
			slog.Info("Clerk JWT verifier initialized", "domain", cfg.SaaS.Clerk.Domain)
		}

		// 3. Stripe webhook handler and usage reporter.
		stripeWH := saasstripe.NewWebhookHandler(
			tenantStore, cfg.SaaS.Stripe.WebhookSecret, slog.Default(),
		)
		adminAPI.SetStripeWebhook(stripeWH)

		usageReporter := saasstripe.NewUsageReporter(
			tenantStore, cfg.SaaS.Stripe.SecretKey, slog.Default(),
		)
		go usageReporter.RunPeriodicReporting(rootCtx, 1*time.Hour)

		// 4. Provisioning orchestrator.
		flyClient := provisioning.NewFlyClient(
			cfg.SaaS.Provisioning.FlyAPIToken,
			cfg.SaaS.Provisioning.FlyOrg,
			cfg.SaaS.Provisioning.FlyRegion,
			cfg.SaaS.Provisioning.Image,
		)
		cfClient := provisioning.NewCloudflareClient(
			cfg.SaaS.Cloudflare.APIToken,
			cfg.SaaS.Cloudflare.ZoneID,
		)
		orchestrator := provisioning.NewOrchestrator(flyClient, cfClient, tenantStore)
		_ = orchestrator // Available for tenant lifecycle operations
		slog.Info("SaaS provisioning orchestrator initialized")

		// 5. Quota enforcement middleware (applied to gateway handler below).
		quotaMiddleware = quota.New(tenantStore)

		slog.Info("SaaS mode fully initialized",
			"tenant_store", mode,
			"clerk_webhooks", cfg.SaaS.Clerk.WebhookSecret != "",
			"stripe_webhooks", cfg.SaaS.Stripe.WebhookSecret != "",
		)
	}

	gw := gateway.NewWithIAM(cfg, registry, store, engine)
	gw.SetEventBus(bus)
	gw.SetPluginManager(pluginMgr)
	var handler http.Handler = gw
	// Wrap with chaos middleware for fault injection
	handler = gateway.ChaosMiddleware(handler, chaosEngine)
	// Enable CORS by default (disable with CLOUDMOCK_CORS=false)
	corsEnabled := os.Getenv("CLOUDMOCK_CORS")
	if corsEnabled != "false" && corsEnabled != "0" {
		handler = gateway.CORSMiddleware(handler)
	}
	if cfg.RateLimit.Enabled {
		limiter := ratelimit.New(cfg.RateLimit.RequestsPerSecond, cfg.RateLimit.Burst)
		handler = limiter.Middleware(handler)
	}
	// Apply SaaS quota enforcement (after rate limiting, before logging).
	if quotaMiddleware != nil {
		handler = quotaMiddleware.Handler(handler)
	}
	loggedGW := gateway.LoggingMiddlewareWithOpts(handler, requestLog, requestStats, gateway.LoggingMiddlewareOpts{
		Broadcaster: adminAPI.Broadcaster(),
		TraceStore:  traceStore,
		SLOEngine:   sloEngine,
		DataPlane:   dp,
		OnRequest: func(service string, latencyMs float64, statusCode int) {
			// Feed latency and error rate into anomaly detector baselines.
			anomalyDetector.UpdateBaseline(service, "latency_p50", latencyMs)
			errorVal := 0.0
			if statusCode >= 500 {
				errorVal = 1.0
			}
			anomalyDetector.UpdateBaseline(service, "error_rate", errorVal)

			// Check for anomalies and notify if detected.
			if anom := anomalyDetector.Check(service, "latency_p50", latencyMs); anom != nil {
				if anom.Severity == "critical" || anom.Severity == "warning" {
					go func() {
						if nr := adminAPI.NotifyRouter(); nr != nil {
							nr.Notify(rootCtx, notifypkg.Notification{
								Title:     anom.Description,
								Severity:  anom.Severity,
								Service:   anom.Service,
								Type:      "anomaly",
								Timestamp: anom.DetectedAt,
								Fields: map[string]string{
									"Metric":    anom.Metric,
									"Observed":  fmt.Sprintf("%.2f", anom.Observed),
									"Expected":  fmt.Sprintf("%.2f", anom.Expected),
									"Deviation": fmt.Sprintf("%.1f sigma", anom.Deviation),
								},
							})
						}
					}()
				}
			}
		},
	})

	var adminHandler http.Handler = adminAPI
	if cfg.Auth.Enabled {
		adminHandler = auth.Middleware([]byte(cfg.Auth.Secret))(adminHandler)
	}
	if cfg.AdminAuth.Enabled && cfg.AdminAuth.APIKey != "" {
		adminHandler = admin.AdminAuthMiddleware(adminHandler, cfg.AdminAuth.APIKey)
	}
	if corsEnabled != "false" && corsEnabled != "0" {
		adminHandler = gateway.CORSMiddleware(adminHandler)
	}
	adminAddr := fmt.Sprintf(":%d", cfg.Admin.Port)
	adminServer := &http.Server{
		Addr:              adminAddr,
		Handler:           adminHandler,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		slog.Info("cloudmock admin API starting", "addr", adminAddr)
		if err := adminServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("admin API exited", "error", err)
		}
	}()

	// Source server: accepts TCP connections from @cloudmock/node SDK
	// SDK-captured HTTP requests (e.g. BFF inbound traffic) are injected into RequestLog
	sourceServer := admin.NewSourceServer(requestLog, requestStats, adminAPI.Broadcaster())
	adminAPI.SetSourceServer(sourceServer)
	sourceAddr := ":4580"
	go func() {
		if err := sourceServer.ListenAndServe(sourceAddr); err != nil {
			slog.Error("source server exited", "error", err)
		}
	}()

	// OTLP/HTTP ingestion server — accepts OpenTelemetry traces, metrics, and logs.
	var otlpServer *http.Server
	if cfg.OTLP.Enabled {
		otlpHandler := otlp.NewServer(dp, bus, cfg.Region, cfg.AccountID)
		otlpAddr := fmt.Sprintf(":%d", cfg.OTLP.Port)
		otlpServer = &http.Server{
			Addr:              otlpAddr,
			Handler:           otlpHandler,
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      60 * time.Second,
			IdleTimeout:       120 * time.Second,
			ReadHeaderTimeout: 10 * time.Second,
		}
		go func() {
			slog.Info("cloudmock OTLP/HTTP server starting", "addr", otlpAddr)
			if err := otlpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				slog.Error("OTLP server exited", "error", err)
			}
		}()
	}

	// Dashboard — serves SPA + admin API on a single origin (no CORS needed)
	var dashServer *http.Server
	if cfg.Dashboard.Enabled {
		dashboardHandler := dashboard.New(cfg.Admin.Port)
		dashboardHandler.SetAdminHandler(adminAPI)
		dashAddr := fmt.Sprintf(":%d", cfg.Dashboard.Port)
		dashServer = &http.Server{
			Addr:              dashAddr,
			Handler:           dashboardHandler,
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      60 * time.Second,
			IdleTimeout:       120 * time.Second,
			ReadHeaderTimeout: 10 * time.Second,
		}
		go func() {
			slog.Info("cloudmock dashboard starting", "addr", dashAddr)
			if err := dashServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				slog.Error("dashboard exited", "error", err)
			}
		}()
	}

	// Proxy mode: start the virtual-host reverse proxy and DNS servers.
	// Domains are read from env vars set by the orchestrator (sourced from Pulumi config).
	if os.Getenv("CLOUDMOCK_PROXY") == "true" || os.Getenv("CLOUDMOCK_PROXY") == "1" {
		autotendDomain := os.Getenv("CLOUDMOCK_DOMAIN_AUTOTEND")
		if autotendDomain == "" {
			autotendDomain = "autotend.io"
		}
		cloudmockDomain := os.Getenv("CLOUDMOCK_DOMAIN_CLOUDMOCK")
		if cloudmockDomain == "" {
			cloudmockDomain = "cloudmock.io"
		}

		routes := gateway.BuildRoutes(autotendDomain, cloudmockDomain)
		certs, certsErr := gateway.EnsureCerts(autotendDomain, cloudmockDomain)
		if certsErr != nil {
			slog.Warn("proxy: TLS certs unavailable, starting HTTP only", "error", certsErr)
			certs = nil
		}
		gateway.StartProxyWithOpts(routes, certs, gateway.ProxyOpts{
			RequestLog:  requestLog,
			Stats:       requestStats,
			Broadcaster: adminAPI.Broadcaster(),
		})

		// DNS servers resolve *.localhost.<domain> → 127.0.0.1
		go gateway.StartDNSServer(15353, "localhost."+autotendDomain)
		go gateway.StartDNSServer(15354, "localhost."+cloudmockDomain)
	}

	addr := fmt.Sprintf(":%d", cfg.Gateway.Port)
	gwServer := &http.Server{
		Addr:              addr,
		Handler:           loggedGW,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
	}
	slog.Info("cloudmock gateway starting",
		"addr", addr, "region", cfg.Region, "account", cfg.AccountID,
		"iam_mode", cfg.IAM.Mode, "services", len(registry.List()))

	go func() {
		if err := gwServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("gateway: %v", err)
		}
	}()

	// Startup banner — printed after all servers are launched.
	fmt.Printf("\nCloudMock v1.0.0\n")
	fmt.Printf("  Gateway:    http://localhost:%d\n", cfg.Gateway.Port)
	fmt.Printf("  Devtools:   http://localhost:%d  <-- open in browser\n", cfg.Dashboard.Port)
	fmt.Printf("  Admin API:  http://localhost:%d\n", cfg.Admin.Port)
	if cfg.OTLP.Enabled {
		fmt.Printf("  OTLP/HTTP:  http://localhost:%d  <-- set OTEL_EXPORTER_OTLP_ENDPOINT\n", cfg.OTLP.Port)
	}
	fmt.Printf("  Services:   %d active (%s profile)\n", len(registry.List()), cfg.Profile)
	fmt.Println()
	fmt.Printf("Ready. Point your AWS SDK at http://localhost:%d\n\n", cfg.Gateway.Port)

	// Wait for termination signal.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	sig := <-sigCh
	slog.Info("shutdown signal received", "signal", sig)

	// Cancel the root context so background goroutines (e.g. SLO callbacks) stop.
	rootCancel()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := gwServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("gateway shutdown error", "error", err)
	}
	if err := adminServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("admin shutdown error", "error", err)
	}
	if dashServer != nil {
		if err := dashServer.Shutdown(shutdownCtx); err != nil {
			slog.Error("dashboard shutdown error", "error", err)
		}
	}
	if otlpServer != nil {
		if err := otlpServer.Shutdown(shutdownCtx); err != nil {
			slog.Error("OTLP server shutdown error", "error", err)
		}
	}

	// Explicit resource cleanup (replaces deferred calls).
	if regEngine != nil {
		regEngine.Stop()
	}
	if otelShutdown != nil {
		if err := otelShutdown(shutdownCtx); err != nil {
			slog.Error("otel shutdown error", "error", err)
		}
	}
	if duckClient != nil {
		duckClient.Close()
	}
	if pgPool != nil {
		pgPool.Close()
	}

	slog.Info("shutdown complete")
}
