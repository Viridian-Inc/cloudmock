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

	// Register S3 with event bus support
	registry.Register(s3svc.NewWithBus(bus))

	registry.Register(stssvc.New(cfg.AccountID))
	registry.Register(kmssvc.New(cfg.AccountID, cfg.Region))
	registry.Register(secretssvc.New(cfg.AccountID, cfg.Region))
	registry.Register(ssmsvc.New(cfg.AccountID, cfg.Region))
	registry.Register(sqssvc.New(cfg.AccountID, cfg.Region))
	registry.Register(sessvc.New(cfg.AccountID, cfg.Region))

	// Register SNS with service locator for SNS → SQS fan-out
	snsService := snssvc.New(cfg.AccountID, cfg.Region)
	registry.Register(snsService)

	registry.Register(dynamodbsvc.New(cfg.AccountID, cfg.Region))
	registry.Register(logssvc.New(cfg.AccountID, cfg.Region))
	registry.Register(cwsvc.New(cfg.AccountID, cfg.Region))
	registry.Register(firehosesvc.New(cfg.AccountID, cfg.Region))
	registry.Register(kinesissvc.New(cfg.AccountID, cfg.Region))
	registry.Register(r53svc.New(cfg.AccountID, cfg.Region))
	registry.Register(ecrsvc.New(cfg.AccountID, cfg.Region))
	registry.Register(ecssvc.New(cfg.AccountID, cfg.Region))
	registry.Register(cognitosvc.New(cfg.AccountID, cfg.Region))

	// Register EventBridge with service locator for target delivery
	ebService := ebsvc.New(cfg.AccountID, cfg.Region)
	registry.Register(ebService)

	registry.Register(sfnsvc.New(cfg.AccountID, cfg.Region))
	registry.Register(rdssvc.New(cfg.AccountID, cfg.Region))
	registry.Register(apigwsvc.New(cfg.AccountID, cfg.Region))
	registry.Register(cfnsvc.New(cfg.AccountID, cfg.Region))

	// Register Lambda with service locator for S3 code source
	lambdaService := lambdasvc.New(cfg.AccountID, cfg.Region)
	registry.Register(lambdaService)

	// Set service locators now that all services are registered.
	// (This breaks the circular dependency: services need the registry,
	// but the registry needs the services.)
	snsService.SetLocator(registry)
	ebService.SetLocator(registry)
	lambdaService.SetLocator(registry)

	// Wire cross-service integrations via event bus
	integration.WireIntegrations(bus, registry, cfg.AccountID, cfg.Region)

	registry.Register(ec2svc.New(cfg.AccountID, cfg.Region))

	// Tier 2 stub services
	stubs.RegisterAll(registry, cfg.AccountID, cfg.Region)

	requestLog := gateway.NewRequestLog(1000)
	requestStats := gateway.NewRequestStats()

	gw := gateway.NewWithIAM(cfg, registry, store, engine)
	var handler http.Handler = gw
	// Enable CORS by default (disable with CLOUDMOCK_CORS=false)
	corsEnabled := os.Getenv("CLOUDMOCK_CORS")
	if corsEnabled != "false" && corsEnabled != "0" {
		handler = gateway.CORSMiddleware(handler)
	}
	loggedGW := gateway.LoggingMiddleware(handler, requestLog, requestStats)

	// Admin API (with CORS for dashboard cross-origin access)
	adminAPI := admin.New(cfg, registry, requestLog, requestStats)
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
