.PHONY: build build-gateway build-cli test lint clean proto docker docker-up docker-down help

help:
	@echo "Available targets:"
	@echo "  build           - Build both gateway and CLI"
	@echo "  build-gateway   - Build the gateway binary"
	@echo "  build-cli       - Build the CLI binary"
	@echo "  test            - Run all tests"
	@echo "  lint            - Run golangci-lint"
	@echo "  clean           - Remove build artifacts"
	@echo "  proto           - Generate protobuf files"
	@echo "  docker          - Build Docker image"
	@echo "  docker-up       - Start Docker containers"
	@echo "  docker-down     - Stop Docker containers"

build: build-gateway build-cli

build-gateway:
	@echo "Building gateway..."
	@mkdir -p bin
	@go build -o bin/gateway ./cmd/gateway

build-cli:
	@echo "Building CLI..."
	@mkdir -p bin
	@go build -o bin/cloudmock ./cmd/cloudmock

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
	@echo "Building Docker image..."
	@docker build -t cloudmock:latest .

docker-up:
	@echo "Starting Docker containers..."
	@docker-compose up -d

docker-down:
	@echo "Stopping Docker containers..."
	@docker-compose down
