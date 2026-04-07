---
title: About CloudMock
description: How CloudMock works — architecture, performance modes, and the full feature set.
---

## What is CloudMock?

CloudMock is a local AWS emulator that runs 100 fully implemented AWS services in a single binary. Point any AWS SDK at `localhost:4566` and your code works as if it's talking to real AWS — but everything runs on your laptop, costs nothing, and starts in under a second.

## Why CloudMock?

**LocalStack** charges for Pro features (chaos testing, CI, Cloud Pods) and uses 583MB of RAM. **Moto** only works in-process for Python and has no DevTools. CloudMock gives you everything — 100 services, chaos engineering, state snapshots, traffic replay, distributed tracing, IaC support, and 10 language SDKs — in one free binary that uses 67MB.

## How it works

```
Your Code → AWS SDK → localhost:4566 → CloudMock Gateway → 100 AWS Services
                                            ↓
                                       DevTools (localhost:4500)
                                       ├── Topology map
                                       ├── Request tracing
                                       ├── Metrics dashboard
                                       ├── Chaos engineering
                                       └── Traffic replay
```

Every request flows through the gateway, which routes to the correct service based on SigV4 credential scope. Services maintain state in memory (or export to JSON via state snapshots). The DevTools dashboard shows real-time topology, traces, and metrics.

## Two performance modes

**HTTP mode** (any language): Start CloudMock as a server. Every AWS SDK works. Sub-millisecond latency.

**In-process mode** (Go only): Import `github.com/Viridian-Inc/cloudmock/sdk`. Zero network. 20 microseconds per operation. The AWS SDK doesn't know it's not hitting a real server.

## The full toolkit

CloudMock isn't just a mock — it's a complete local AWS development platform:

- **100 AWS services** with proper error codes, pagination, and state machines
- **10 language SDKs** (Go, Python, Node, Java, Kotlin, Rust, C/C++, Ruby, C#, Swift)
- **IaC support** (Terraform, CDK, Pulumi — use your existing files)
- **State snapshots** — export to JSON, commit to git, restore on startup
- **Traffic replay** — record real AWS calls, replay against CloudMock
- **Contract testing** — prove your mock matches production
- **Chaos engineering** — inject S3 503s, DynamoDB throttles, Lambda timeouts
- **Distributed tracing** — W3C traceparent propagation, OpenTelemetry ingestion
- **CloudTrail replay** — recreate production state from audit logs
- **Multi-account** — isolated accounts with cross-account STS AssumeRole
- **Compatibility dashboard** — live API compatibility tested nightly
- **Docker Compose stacks** — 8 pre-built templates for common architectures
- **GitHub Action** — one line to add CloudMock to CI
