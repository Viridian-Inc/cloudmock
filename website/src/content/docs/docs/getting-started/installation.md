---
title: Installation
description: Install CloudMock on macOS, Linux, or Windows
---

CloudMock is a single binary with zero external dependencies. Pick the install method that fits your workflow.

## npx (quickest)

Run without installing:

```bash
npx cloudmock start
```

This downloads the latest release binary and starts the gateway. No global install required.

## Snap (Linux)

```bash
sudo snap install cloudmock
cloudmock start
```

## Homebrew (macOS / Linux)

```bash
brew install viridian-inc/tap/cloudmock
cloudmock start
```

To upgrade later:

```bash
brew upgrade cloudmock
```

## Docker

```bash
docker run -p 4566:4566 -p 4500:4500 ghcr.io/Viridian-Inc/cloudmock:latest
```

To run in the background:

```bash
docker run -d --name cloudmock -p 4566:4566 -p 4500:4500 ghcr.io/Viridian-Inc/cloudmock:latest
```

With Docker Compose, add this to your `docker-compose.yml`:

```yaml
services:
  cloudmock:
    image: ghcr.io/Viridian-Inc/cloudmock:latest
    ports:
      - "4566:4566"
      - "4500:4500"
```

Then `docker compose up -d`.

## Shell installer (macOS / Linux)

```bash
curl -fsSL https://cloudmock.app/install.sh | bash
cloudmock start
```

## apt (Debian / Ubuntu)

```bash
curl -LO https://github.com/Viridian-Inc/cloudmock/releases/download/v1.0.3/cloudmock_1.0.3_amd64.deb && sudo apt install cloudmock_1.0.3_amd64.deb
cloudmock start
```

ARM64:

```bash
curl -LO https://github.com/Viridian-Inc/cloudmock/releases/download/v1.0.3/cloudmock_1.0.3_arm64.deb && sudo apt install cloudmock_1.0.3_arm64.deb
```

## go install (from source)

Requires Go 1.26 or later:

```bash
go install github.com/Viridian-Inc/cloudmock/cmd/gateway@latest
cloudmock start
```

## System requirements

| Platform | Support |
|----------|---------|
| macOS (arm64, amd64) | Native binary |
| Linux (arm64, amd64) | Native binary |
| Windows | Docker (recommended), or WSL2 with the Linux binary |

CloudMock has no runtime dependencies. No database, no JVM, no Docker requirement (unless you choose the Docker install method).

## Verify it works

After starting CloudMock, you should see output like this:

```
Starting cloudmock gateway (config=cloudmock.yml)

  AWS Gateway:  http://localhost:4566
  Dashboard:    http://localhost:4500
  Admin API:    http://localhost:4599

  Profile:      minimal (8 services)
  IAM mode:     enforce
  Persistence:  off

  Ready.
```

The gateway listens on three ports:

| Port | Purpose |
|------|---------|
| 4566 | AWS API endpoint -- point your SDKs and CLI here |
| 4500 | Web dashboard for inspecting services and resources |
| 4599 | Admin API for health checks, resets, and configuration |

## Configuration

CloudMock reads configuration from `cloudmock.yml` in the working directory. The defaults work for most use cases. See the [Configuration Reference](/docs/reference/configuration/) for all options.

Common overrides via environment variables:

```bash
# Change ports
CLOUDMOCK_GATEWAY_PORT=5566 cloudmock start

# Use a different service profile
CLOUDMOCK_PROFILE=standard cloudmock start

# Disable IAM enforcement for quick prototyping
CLOUDMOCK_IAM_MODE=none cloudmock start
```

## Try the demo

Clone the example project and run it in under a minute:

```bash
git clone https://github.com/Viridian-Inc/cloudmock-todo-demo.git
cd cloudmock-todo-demo/node
npx cloudmock &
npm install && npx tsx quickstart.ts
```

See the [full demo project](https://github.com/Viridian-Inc/cloudmock-todo-demo) for Python and Go examples.

## Next step

You have CloudMock running. Now [make your first request](/docs/getting-started/first-request/).
