# Contributing to CloudMock

Thank you for your interest in contributing to CloudMock! This guide will help you get started.

## Code of Conduct

By participating in this project, you agree to maintain a respectful and inclusive environment. Be kind, constructive, and professional in all interactions.

## Development Setup

### Prerequisites

- Go 1.22+
- Make
- Docker (for integration tests)
- Node.js 18+ and pnpm (for DevTools UI, optional)

### Clone and Build

```bash
git clone https://github.com/neureaux/cloudmock.git
cd cloudmock
make build
```

This produces binaries in `bin/`:
- `bin/gateway` -- the main CloudMock server
- `bin/cloudmock` -- the CLI
- `bin/cmk` -- the AWS CLI wrapper

### Run Locally

```bash
./bin/gateway
# Gateway on :4566, DevTools on :4500, Admin API on :4599
```

## Running Tests

```bash
# Unit tests (no Docker required)
make test-unit

# All tests including integration (requires Docker)
make test-all

# Tests with race detector
make test-race

# Lint
make lint
```

## Project Structure

```
cmd/
  gateway/       Main server entry point
  cloudmock/     CLI entry point
  cmk/           AWS CLI wrapper
gateway/         HTTP router, AWS request parsing, middleware
services/        One package per AWS service (98 total)
pkg/
  otel/          OpenTelemetry collector and trace storage
  dashboard/     Embedded DevTools web UI
  admin/         Admin API (chaos, snapshots, config)
  dataplane/     Production data plane (PostgreSQL-backed)
plugins/         External plugin system
tools/           AWS tool wrappers (CDK, SAM, Chalice, Copilot)
codegen/         Code generation for service stubs
docs/            Documentation
tests/           Integration and end-to-end tests
```

## Adding a New AWS Service

CloudMock uses a consistent pattern for all services. Here is how to add one:

### 1. Generate the Scaffold

```bash
# The codegen tool creates the boilerplate
go run codegen/main.go --service <service-name>
```

This creates `services/<service-name>/` with:
- `handler.go` -- request router and action dispatch
- `store.go` -- in-memory data store
- `types.go` -- request/response structs
- `<service>_test.go` -- test file

### 2. Implement Actions

Each AWS API action maps to a handler function:

```go
// services/myservice/handler.go
func (h *Handler) CreateThing(ctx context.Context, input *CreateThingInput) (*CreateThingOutput, error) {
    thing := &Thing{
        ThingId:   generateID(),
        ThingName: input.ThingName,
        CreatedAt: time.Now(),
    }
    h.store.Put(thing.ThingId, thing)
    return &CreateThingOutput{Thing: thing}, nil
}
```

### 3. Register the Service

Add the service to the gateway's service registry in `gateway/registry.go`:

```go
registry.Register("myservice", myservice.New)
```

### 4. Add Tests

Write tests that exercise the service through the AWS SDK:

```go
func TestCreateThing(t *testing.T) {
    gw := testutil.StartGateway(t)
    client := myservice.NewFromConfig(gw.AWSConfig())

    out, err := client.CreateThing(context.Background(), &myservice.CreateThingInput{
        ThingName: aws.String("test-thing"),
    })
    require.NoError(t, err)
    assert.Equal(t, "test-thing", *out.Thing.ThingName)
}
```

### 5. Document It

Add the service to `docs/compatibility-matrix.md` with the list of supported operations.

## Adding an Admin API Endpoint

The admin API (port 4599) handles DevTools and management operations:

1. Add a handler in `pkg/admin/`
2. Register the route in `pkg/admin/router.go`
3. Add tests
4. Document in `docs/admin-api.md`

## Pull Request Guidelines

### Before Submitting

- Run `make test-unit` and `make lint` -- both must pass
- Add tests for new functionality
- Update documentation if behavior changes
- Keep commits focused -- one logical change per commit

### PR Process

1. Fork the repository
2. Create a feature branch from `main` (`git checkout -b feature/my-feature`)
3. Make your changes
4. Push to your fork and open a pull request against `main`
5. Fill in the PR template with a description of changes and test plan
6. Wait for CI to pass and a maintainer to review

### What We Look For

- **Tests** -- new features need tests, bug fixes need regression tests
- **Documentation** -- user-facing changes should update docs
- **Backwards compatibility** -- avoid breaking existing APIs
- **Code style** -- follow existing patterns; `go vet` and `golangci-lint` must pass
- **Commit messages** -- clear, concise descriptions of what changed and why

### Small PRs Are Better

Large PRs are hard to review. If your change is big, consider splitting it into smaller, incremental PRs that each make sense on their own.

## Reporting Bugs

Use [GitHub Issues](https://github.com/neureaux/cloudmock/issues) with the bug report template. Include:

- CloudMock version (`cloudmock --version`)
- Steps to reproduce
- Expected vs actual behavior
- Relevant logs or traces

## Requesting Features

Use [GitHub Issues](https://github.com/neureaux/cloudmock/issues) with the feature request template. Describe the use case, not just the solution.

## License

By contributing to CloudMock, you agree that your contributions will be licensed under the Apache License 2.0.
