.PHONY: build build-gateway build-cli build-tools build-dashboard test lint clean proto docker docker-push docker-up docker-down release help

VERSION ?= 0.1.0

help:
	@echo "Available targets:"
	@echo "  build           - Build both gateway and CLI"
	@echo "  build-gateway   - Build the gateway binary"
	@echo "  build-cli       - Build the CLI binary"
	@echo "  build-tools     - Build AWS tool wrappers and CI helper"
	@echo "  build-dashboard - Build the web dashboard"
	@echo "  test            - Run all tests"
	@echo "  lint            - Run golangci-lint"
	@echo "  clean           - Remove build artifacts"
	@echo "  proto           - Generate protobuf files"
	@echo "  docker          - Build Docker image"
	@echo "  docker-push     - Tag and push Docker image to ghcr.io"
	@echo "  docker-up       - Start Docker containers"
	@echo "  docker-down     - Stop Docker containers"
	@echo "  release         - Build cross-platform release binaries"

build: build-gateway build-cli build-tools

build-gateway:
	@echo "Building gateway..."
	@mkdir -p bin
	@go build -ldflags="-s -w -X github.com/neureaux/cloudmock/pkg/admin.Version=1.0.0 -X github.com/neureaux/cloudmock/pkg/admin.BuildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)" -o bin/gateway ./cmd/gateway

build-cli:
	@echo "Building CLI..."
	@mkdir -p bin
	@go build -o bin/cloudmock ./cmd/cloudmock

build-tools:
	@echo "Building tool wrappers..."
	@mkdir -p bin
	@go build -o bin/cloudmock-aws ./tools/cloudmock-aws
	@go build -o bin/cloudmock-cdk ./tools/cloudmock-cdk
	@go build -o bin/cloudmock-sam ./tools/cloudmock-sam
	@go build -o bin/cloudmock-chalice ./tools/cloudmock-chalice
	@go build -o bin/cloudmock-copilot ./tools/cloudmock-copilot
	@go build -o bin/cloudmock-ci ./tools/cloudmock-ci

test:
	@echo "Running tests..."
	@go test -v -cover ./...

lint:
	@echo "Running linter..."
	@golangci-lint run ./...

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -f coverage.out
	@go clean -testcache

proto:
	@echo "Generating protobuf files..."
	@echo "Proto generation not yet implemented"

docker:
	docker build -t cloudmock:$(VERSION) -t cloudmock:latest .

docker-push:
	docker tag cloudmock:latest ghcr.io/neureaux/cloudmock:$(VERSION)
	docker tag cloudmock:latest ghcr.io/neureaux/cloudmock:latest
	docker push ghcr.io/neureaux/cloudmock:$(VERSION)
	docker push ghcr.io/neureaux/cloudmock:latest

release: build-dashboard
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dist/cloudmock-linux-amd64 ./cmd/gateway
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o dist/cloudmock-linux-arm64 ./cmd/gateway
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o dist/cloudmock-darwin-amd64 ./cmd/gateway
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o dist/cloudmock-darwin-arm64 ./cmd/gateway
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o dist/cloudmock-windows-amd64.exe ./cmd/gateway

build-dashboard:
	cd dashboard && npm run build
	cp -r dashboard/dist/ pkg/dashboard/dist/

docker-up:
	@echo "Starting Docker containers..."
	@docker-compose up -d

docker-down:
	@echo "Stopping Docker containers..."
	@docker-compose down

.PHONY: dev-prod
dev-prod: ## Start production data plane (Docker Compose)
	docker compose -f docker/docker-compose.prod.yml up -d

.PHONY: dev-prod-down
dev-prod-down: ## Stop production data plane
	docker compose -f docker/docker-compose.prod.yml down

.PHONY: config-import
config-import: ## Import cloudmock.yml into PostgreSQL
	go run cmd/configimport/main.go --config cloudmock.yml --pg-url "postgres://cloudmock:cloudmock@localhost:5432/cloudmock"

.PHONY: test-integration
test-integration: ## Run integration tests (requires Docker)
	go test -v -cover ./pkg/dataplane/... -count=1

.PHONY: test-unit
test-unit: ## Run unit tests only (no Docker)
	go test -v -short -cover ./...
