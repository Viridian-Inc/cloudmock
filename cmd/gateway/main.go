package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/gateway"
	iampkg "github.com/neureaux/cloudmock/pkg/iam"
	"github.com/neureaux/cloudmock/pkg/routing"
	cwsvc "github.com/neureaux/cloudmock/services/cloudwatch"
	logssvc "github.com/neureaux/cloudmock/services/cloudwatchlogs"
	dynamodbsvc "github.com/neureaux/cloudmock/services/dynamodb"
	kmssvc "github.com/neureaux/cloudmock/services/kms"
	s3svc "github.com/neureaux/cloudmock/services/s3"
	secretssvc "github.com/neureaux/cloudmock/services/secretsmanager"
	snssvc "github.com/neureaux/cloudmock/services/sns"
	sqssvc "github.com/neureaux/cloudmock/services/sqs"
	ssmsvc "github.com/neureaux/cloudmock/services/ssm"
	stssvc "github.com/neureaux/cloudmock/services/sts"
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

	// Service registry
	registry := routing.NewRegistry()
	registry.Register(s3svc.New())
	registry.Register(stssvc.New(cfg.AccountID))
	registry.Register(kmssvc.New(cfg.AccountID, cfg.Region))
	registry.Register(secretssvc.New(cfg.AccountID, cfg.Region))
	registry.Register(ssmsvc.New(cfg.AccountID, cfg.Region))
	registry.Register(sqssvc.New(cfg.AccountID, cfg.Region))
	registry.Register(snssvc.New(cfg.AccountID, cfg.Region))
	registry.Register(dynamodbsvc.New(cfg.AccountID, cfg.Region))
	registry.Register(logssvc.New(cfg.AccountID, cfg.Region))
	registry.Register(cwsvc.New(cfg.AccountID, cfg.Region))

	gw := gateway.NewWithIAM(cfg, registry, store, engine)

	addr := fmt.Sprintf(":%d", cfg.Gateway.Port)
	log.Printf("cloudmock gateway starting on %s (region=%s, account=%s, iam_mode=%s, services=%d)",
		addr, cfg.Region, cfg.AccountID, cfg.IAM.Mode, len(registry.List()))

	if err := http.ListenAndServe(addr, gw); err != nil {
		log.Fatalf("gateway exited: %v", err)
	}
}
