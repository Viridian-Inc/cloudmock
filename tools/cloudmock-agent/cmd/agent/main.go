package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/neureaux/cloudmock/tools/cloudmock-agent/sdk"
)

func main() {
	var (
		listenAddr   = flag.String("listen", ":4577", "Address to listen on")
		otelEndpoint = flag.String("endpoint", "localhost:4318", "OTLP endpoint")
		apiKey       = flag.String("api-key", "", "CloudMock API key")
		environment  = flag.String("env", "production", "Environment name")
		orgID        = flag.String("org-id", "", "Organization ID")
		appID        = flag.String("app-id", "", "App ID")
	)
	flag.Parse()

	// Override from env vars
	if v := os.Getenv("CLOUDMOCK_OTEL_ENDPOINT"); v != "" {
		*otelEndpoint = v
	}
	if v := os.Getenv("CLOUDMOCK_API_KEY"); v != "" {
		*apiKey = v
	}
	if v := os.Getenv("CLOUDMOCK_ENVIRONMENT"); v != "" {
		*environment = v
	}
	if v := os.Getenv("CLOUDMOCK_ORG_ID"); v != "" {
		*orgID = v
	}
	if v := os.Getenv("CLOUDMOCK_APP_ID"); v != "" {
		*appID = v
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	// Setup OTel
	shutdown, err := sdk.SetupOTel(
		context.Background(), *otelEndpoint, *apiKey, "cloudmock-agent",
	)
	if err != nil {
		slog.Error("failed to setup OTel", "error", err)
		os.Exit(1)
	}
	defer shutdown()

	tracer := otel.Tracer("cloudmock-agent-proxy")

	// Create reverse proxy to real AWS
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			// Route to the real AWS endpoint based on the service
			service := extractServiceFromRequest(req)
			region := extractRegionFromRequest(req)
			if region == "" {
				region = "us-east-1"
			}

			target := fmt.Sprintf("https://%s.%s.amazonaws.com", service, region)
			u, _ := url.Parse(target)
			req.URL.Scheme = u.Scheme
			req.URL.Host = u.Host
			req.Host = u.Host
		},
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{},
			MaxIdleConns:    100,
			IdleConnTimeout: 90 * time.Second,
		},
	}

	// Capture flag values for use in closure
	env := *environment
	org := *orgID
	app := *appID

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		service, action := sdk.DetectServiceAction(r)

		_, span := tracer.Start(r.Context(), service+"."+action,
			trace.WithSpanKind(trace.SpanKindClient),
		)
		defer span.End()

		span.SetAttributes(
			attribute.String("aws.service", service),
			attribute.String("aws.action", action),
			attribute.String("aws.region", extractRegionFromRequest(r)),
			attribute.String("cloudmock.environment", env),
			attribute.String("cloudmock.source", "agent-proxy"),
			attribute.String("cloudmock.org_id", org),
			attribute.String("cloudmock.app_id", app),
		)

		start := time.Now()
		proxy.ServeHTTP(w, r)
		duration := time.Since(start)

		span.SetAttributes(attribute.Float64("duration_ms", float64(duration.Milliseconds())))
	})

	slog.Info("cloudmock agent starting",
		"listen", *listenAddr,
		"endpoint", *otelEndpoint,
		"environment", *environment,
	)

	fmt.Printf("\nCloudMock Agent\n")
	fmt.Printf("  Proxy:     http://localhost%s\n", *listenAddr)
	fmt.Printf("  OTLP:      %s\n", *otelEndpoint)
	fmt.Printf("  Env:       %s\n\n", *environment)
	fmt.Printf("Set AWS_ENDPOINT_URL=http://localhost%s to route through the agent\n\n", *listenAddr)

	if err := http.ListenAndServe(*listenAddr, handler); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func extractServiceFromRequest(req *http.Request) string {
	s, _ := sdk.DetectServiceAction(req)
	return s
}

func extractRegionFromRequest(req *http.Request) string {
	return sdk.ExtractRegion(req)
}
