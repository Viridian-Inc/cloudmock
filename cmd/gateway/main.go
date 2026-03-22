package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/neureaux/cloudmock/pkg/admin"
	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/dashboard"
	"github.com/neureaux/cloudmock/pkg/eventbus"
	"github.com/neureaux/cloudmock/pkg/gateway"
	iampkg "github.com/neureaux/cloudmock/pkg/iam"
	"github.com/neureaux/cloudmock/pkg/integration"
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
	"github.com/neureaux/cloudmock/services/stubs"
)

func main() {
	configPath := flag.String("config", "", "path to cloudmock YAML config file")
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
	// "standard" — all Tier 1 services.
	// "full"     — all Tier 1 services (Tier 2 stubs are lazy regardless).
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

	// Tier 2 stub services — always lazy to avoid initializing ~73 services at startup.
	stubs.RegisterAllLazy(registry, cfg.AccountID, cfg.Region)

	requestLog := gateway.NewRequestLog(1000)
	requestStats := gateway.NewRequestStats()
	traceStore := gateway.NewTraceStore(500)

	// Chaos engine for fault injection
	chaosEngine := gateway.NewChaosEngine()

	// Admin API (with CORS for dashboard cross-origin access)
	adminAPI := admin.New(cfg, registry, requestLog, requestStats)
	adminAPI.SetTraceStore(traceStore)
	adminAPI.SetChaosEngine(chaosEngine)

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

	gw := gateway.NewWithIAM(cfg, registry, store, engine)
	var handler http.Handler = gw
	// Wrap with chaos middleware for fault injection
	handler = gateway.ChaosMiddleware(handler, chaosEngine)
	// Enable CORS by default (disable with CLOUDMOCK_CORS=false)
	corsEnabled := os.Getenv("CLOUDMOCK_CORS")
	if corsEnabled != "false" && corsEnabled != "0" {
		handler = gateway.CORSMiddleware(handler)
	}
	loggedGW := gateway.LoggingMiddlewareWithOpts(handler, requestLog, requestStats, gateway.LoggingMiddlewareOpts{
		Broadcaster: adminAPI.Broadcaster(),
		TraceStore:  traceStore,
	})

	var adminHandler http.Handler = adminAPI
	if corsEnabled != "false" && corsEnabled != "0" {
		adminHandler = gateway.CORSMiddleware(adminHandler)
	}
	adminAddr := fmt.Sprintf(":%d", cfg.Admin.Port)
	go func() {
		log.Printf("cloudmock admin API starting on %s", adminAddr)
		if err := http.ListenAndServe(adminAddr, adminHandler); err != nil {
			log.Printf("admin API exited: %v", err)
		}
	}()

	// Dashboard
	if cfg.Dashboard.Enabled {
		dashboardHandler := dashboard.New(cfg.Admin.Port)
		dashAddr := fmt.Sprintf(":%d", cfg.Dashboard.Port)
		go func() {
			log.Printf("cloudmock dashboard starting on %s", dashAddr)
			if err := http.ListenAndServe(dashAddr, dashboardHandler); err != nil {
				log.Printf("dashboard exited: %v", err)
			}
		}()
	}

	addr := fmt.Sprintf(":%d", cfg.Gateway.Port)
	log.Printf("cloudmock gateway starting on %s (region=%s, account=%s, iam_mode=%s, services=%d)",
		addr, cfg.Region, cfg.AccountID, cfg.IAM.Mode, len(registry.List()))

	if err := http.ListenAndServe(addr, loggedGW); err != nil {
		log.Fatalf("gateway exited: %v", err)
	}
}
