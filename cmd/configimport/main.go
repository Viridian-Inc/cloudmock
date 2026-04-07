package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/Viridian-Inc/cloudmock/pkg/config"
	"github.com/Viridian-Inc/cloudmock/pkg/dataplane/postgres"
)

func main() {
	configPath := flag.String("config", "cloudmock.yml", "path to cloudmock.yml")
	pgURL := flag.String("pg-url", "", "PostgreSQL connection URL (required)")
	flag.Parse()

	if *pgURL == "" {
		log.Fatal("--pg-url is required")
	}

	cfg, err := config.LoadFromFile(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx := context.Background()
	pool, err := postgres.NewPool(ctx, config.PostgreSQLConfig{URL: *pgURL})
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer pool.Close()

	configStore := postgres.NewConfigStore(pool)
	sloStore := postgres.NewSLOStore(pool)

	// Import SLO rules
	if len(cfg.SLO.Rules) > 0 {
		if err := sloStore.SetRules(ctx, cfg.SLO.Rules); err != nil {
			log.Fatalf("import SLO rules: %v", err)
		}
		fmt.Printf("Imported %d SLO rules\n", len(cfg.SLO.Rules))
	}

	// Import full config
	if err := configStore.SetConfig(ctx, cfg); err != nil {
		log.Fatalf("import config: %v", err)
	}
	fmt.Println("Config imported successfully")
}
