package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Viridian-Inc/cloudmock/services/cloud-ingest/internal/ingest"
	"github.com/Viridian-Inc/cloudmock/services/cloud-ingest/internal/query"
	"github.com/Viridian-Inc/cloudmock/services/cloud-ingest/internal/store"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}

func run() error {
	// -----------------------------------------------------------------------
	// Configuration from environment
	// -----------------------------------------------------------------------
	dbURL := requireEnv("DATABASE_URL")
	migrationsPath := envOr("MIGRATIONS_PATH", "/migrations")
	addr := envOr("ADDR", ":8080")

	// -----------------------------------------------------------------------
	// Database connection pool
	// -----------------------------------------------------------------------
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("open db pool: %w", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping db: %w", err)
	}
	log.Println("database connected")

	// -----------------------------------------------------------------------
	// Run migrations
	// -----------------------------------------------------------------------
	if err := runMigrations(dbURL, migrationsPath); err != nil {
		return fmt.Errorf("migrations: %w", err)
	}

	// -----------------------------------------------------------------------
	// Wire up handlers
	// -----------------------------------------------------------------------
	spanStore := store.NewSpanStore(pool)

	ingestHandler, stopIngest := ingest.New(spanStore)
	queryHandler := query.New(spanStore)

	mux := http.NewServeMux()
	ingestHandler.RegisterRoutes(mux)
	queryHandler.RegisterRoutes(mux)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// -----------------------------------------------------------------------
	// Graceful shutdown
	// -----------------------------------------------------------------------
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("listening on %s", addr)
		serverErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server error: %w", err)
		}
	case sig := <-quit:
		log.Printf("received signal %s — shutting down", sig)

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("http shutdown error: %v", err)
		}

		// Flush remaining buffered spans before exiting.
		stopIngest()
		log.Println("shutdown complete")
	}

	return nil
}

// runMigrations applies all pending up-migrations using golang-migrate.
func runMigrations(dbURL, migrationsPath string) error {
	// golang-migrate expects the postgres:// scheme with ?sslmode param.
	m, err := migrate.New(
		"file://"+migrationsPath,
		dbURL,
	)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer func() {
		srcErr, dbErr := m.Close()
		if srcErr != nil {
			log.Printf("migrate close source: %v", srcErr)
		}
		if dbErr != nil {
			log.Printf("migrate close db: %v", dbErr)
		}
	}()

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Println("migrations: no changes")
			return nil
		}
		return fmt.Errorf("apply: %w", err)
	}
	log.Println("migrations: applied")
	return nil
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required environment variable %s is not set", key)
	}
	return v
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
