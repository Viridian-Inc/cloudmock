.PHONY: build build-gateway build-cli build-cmk build-tools build-plugins build-plugin-example build-dashboard dev restart test test-all lint clean proto docker docker-push docker-up docker-down release help

VERSION ?= 1.0.0
DEVTOOLS_DIR ?= devtools

help:
	@echo "Available targets:"
	@echo "  dev             - Build everything, embed devtools, restart gateway"
	@echo "  restart         - Kill gateway + restart (no rebuild)"
	@echo "  build           - Build both gateway and CLI"
	@echo "  build-gateway   - Build the gateway binary"
	@echo "  build-cli       - Build the CLI binary"
	@echo "  build-tools     - Build AWS tool wrappers and CI helper"
	@echo "  build-plugins   - Build external plugin binaries"
	@echo "  build-dashboard - Build devtools SPA and embed into Go binary"
	@echo "  test            - Run all tests"
	@echo "  lint            - Run golangci-lint"
	@echo "  clean           - Remove build artifacts"
	@echo "  proto           - Generate protobuf files"
	@echo "  docker          - Build Docker image"
	@echo "  docker-push     - Tag and push Docker image to ghcr.io"
	@echo "  docker-up       - Start Docker containers"
	@echo "  docker-down     - Stop Docker containers"
	@echo "  release         - Build cross-platform release binaries"

## dev: one command to build everything, embed devtools, restart gateway
dev: build-dashboard build-gateway restart

## restart: kill any running gateway and start fresh
restart:
	@pkill -9 -f 'bin/gateway' 2>/dev/null || true
	@for p in 4566 4500 4599 4580 4318; do kill -9 $$(lsof -ti:$$p) 2>/dev/null || true; done
	@sleep 1
	@echo "Starting gateway..."
	@bin/gateway > /tmp/cloudmock.log 2>&1 &
	@sleep 3
	@curl -s -o /dev/null -w "Gateway: %{http_code}\n" http://localhost:4599/api/health
	@echo "Devtools: http://localhost:4500"
	@echo "Gateway:  http://localhost:4566"

## build-dashboard: build devtools SPA and embed into Go binary
build-dashboard:
	@echo "Building devtools SPA..."
	@cd $(DEVTOOLS_DIR) && pnpm build
	@rm -rf pkg/dashboard/dist
	@cp -r $(DEVTOOLS_DIR)/dist pkg/dashboard/dist
	@echo "Devtools embedded into pkg/dashboard/dist/"

build: build-gateway build-cli build-cmk build-tools build-plugins

build-gateway:
	@echo "Building gateway..."
	@mkdir -p bin
	@go build -ldflags="-s -w -X github.com/neureaux/cloudmock/pkg/admin.Version=1.0.0 -X github.com/neureaux/cloudmock/pkg/admin.BuildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)" -o bin/gateway ./cmd/gateway

build-cli:
	@echo "Building CLI..."
	@mkdir -p bin
	@go build -o bin/cloudmock ./cmd/cloudmock

build-cmk:
	@echo "Building cmk..."
	@mkdir -p bin
	@go build -ldflags="-s -w" -o bin/cmk ./cmd/cmk

build-tools:
	@echo "Building tool wrappers..."
	@mkdir -p bin
	@go build -o bin/cloudmock-aws ./tools/cloudmock-aws
	@go build -o bin/cloudmock-cdk ./tools/cloudmock-cdk
	@go build -o bin/cloudmock-sam ./tools/cloudmock-sam
	@go build -o bin/cloudmock-chalice ./tools/cloudmock-chalice
	@go build -o bin/cloudmock-copilot ./tools/cloudmock-copilot
	@go build -o bin/cloudmock-ci ./tools/cloudmock-ci

build-plugins: build-plugin-example
	@echo "Plugins built (in-process plugins are compiled into gateway)"

build-plugin-example:
	@echo "Building example plugin..."
	@mkdir -p bin/plugins
	@go build -o bin/plugins/example ./plugins/example/cmd

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

.PHONY: test-race
test-race: ## Run tests with race detector
	go test -race -count=1 -short ./...

.PHONY: test-all
test-all: ## Run all tests: unit + dataplane + integration
	go test -v -cover -count=1 ./...
	go test -v -cover -count=1 ./pkg/dataplane/...
