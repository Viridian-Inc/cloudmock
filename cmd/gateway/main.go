package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/gateway"
	"github.com/neureaux/cloudmock/pkg/routing"
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

	registry := routing.NewRegistry()
	gw := gateway.New(cfg, registry)

	addr := fmt.Sprintf(":%d", cfg.Gateway.Port)
	log.Printf("cloudmock gateway starting on %s (region=%s, account=%s, iam_mode=%s)",
		addr, cfg.Region, cfg.AccountID, cfg.IAM.Mode)

	if err := http.ListenAndServe(addr, gw); err != nil {
		log.Fatalf("gateway exited: %v", err)
	}
}
