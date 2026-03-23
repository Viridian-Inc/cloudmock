package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/neureaux/cloudmock/pkg/admin"
	"github.com/neureaux/cloudmock/pkg/auth"
	authmemory "github.com/neureaux/cloudmock/pkg/auth/memory"
	authpg "github.com/neureaux/cloudmock/pkg/auth/postgres"
	"github.com/neureaux/cloudmock/pkg/ratelimit"
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
	incmemory "github.com/neureaux/cloudmock/pkg/incident/memory"
	incpg "github.com/neureaux/cloudmock/pkg/incident/postgres"
	"github.com/neureaux/cloudmock/pkg/regression"
	"github.com/neureaux/cloudmock/pkg/webhook"
	whmemory "github.com/neureaux/cloudmock/pkg/webhook/memory"
	whpg "github.com/neureaux/cloudmock/pkg/webhook/postgres"
	regmemory "github.com/neureaux/cloudmock/pkg/regression/memory"
	regpg "github.com/neureaux/cloudmock/pkg/regression/postgres"
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
			Config:   memory.NewConfigStore(cfg),
			Topology: memory.NewTopologyStore(),
			Mode:     "local",
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
		defer duckClient.Close()
		if err := duckClient.InitSchema(); err != nil {
			log.Fatalf("duckdb schema: %v", err)
		}

		pgPool, err = pgImpl.NewPool(ctx, cfg.DataPlane.PostgreSQL)
		if err != nil {
			log.Fatalf("postgres: %v", err)
		}
		defer pgPool.Close()

		promClient, err = promImpl.NewClient(cfg.DataPlane.Prometheus)
		if err != nil {
			log.Fatalf("prometheus: %v", err)
		}

		shutdown, err := dataplane.InitTracer(ctx, cfg.DataPlane.OTel)
		if err != nil {
			log.Fatalf("otel: %v", err)
		}
		defer shutdown(ctx)

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
			Config:   pgImpl.NewConfigStore(pgPool),
			Topology: pgImpl.NewTopologyStore(pgPool),
			Mode:     "production",
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

	// Chaos engine for fault injection
	chaosEngine := gateway.NewChaosEngine()

	// Admin API (with CORS for dashboard cross-origin access)
	adminAPI := admin.NewWithDataPlane(cfg, registry, dp)
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
		defer regEngine.Stop()

		adminAPI.SetRegressionEngine(regEngine)
	}

	// Incident management service
	if cfg.Incidents.Enabled {
		var incStore incident.IncidentStore
		switch mode {
		case "local":
			incStore = incmemory.NewStore()
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
				incService.OnSLOBreach(context.Background(), service, action, burnRate, budgetUsed)
			})
		}

		adminAPI.SetIncidentService(incService)

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
	if cfg.RateLimit.Enabled {
		limiter := ratelimit.New(cfg.RateLimit.RequestsPerSecond, cfg.RateLimit.Burst)
		handler = limiter.Middleware(handler)
	}
	loggedGW := gateway.LoggingMiddlewareWithOpts(handler, requestLog, requestStats, gateway.LoggingMiddlewareOpts{
		Broadcaster: adminAPI.Broadcaster(),
		TraceStore:  traceStore,
		SLOEngine:   sloEngine,
		DataPlane:   dp,
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
			log.Printf("proxy: TLS certs unavailable (%v) — starting HTTP only", certsErr)
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
	log.Printf("cloudmock gateway starting on %s (region=%s, account=%s, iam_mode=%s, services=%d)",
		addr, cfg.Region, cfg.AccountID, cfg.IAM.Mode, len(registry.List()))

	if err := http.ListenAndServe(addr, loggedGW); err != nil {
		log.Fatalf("gateway exited: %v", err)
	}
}
