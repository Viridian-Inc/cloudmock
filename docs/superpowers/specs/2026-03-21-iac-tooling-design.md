# cloudmock IaC, AWS Tools & CI Framework Design

**Date:** 2026-03-21
**Status:** Draft
**License:** Apache 2.0

## Overview

Extend cloudmock with Infrastructure-as-Code provider support (Terraform, Pulumi, Crossplane), Cloud Custodian compliance tooling, AWS native tool wrappers (CLI, CDK, SAM, Chalice, Copilot), and a CI framework supporting 7 CI systems (GitHub Actions, GitLab CI, CircleCI, Bitbucket Pipelines, Buildkite, CodeBuild, Travis CI). All components are written in Go (except a minimal Python plugin for Cloud Custodian). Everything lives in the cloudmock monorepo.

## Goals

- Code-generated providers from a unified schema registry — add a service once, all providers update
- Terraform provider covering all 98 services with ~200+ resources + data sources (services like EC2 and S3 have multiple resource types each)
- Pulumi provider bridging Terraform for TypeScript, Python, Go, C#, Java, YAML
- Crossplane provider via upjet + AWS provider endpoint configuration
- Full EC2/VPC networking promoted to Tier 1 with 50+ actions and referential integrity
- Cloud Custodian policy library + Go compliance scanner
- 5 Go wrapper CLIs for AWS native tools with cloudmock auto-configuration
- `cloudmock-ci` binary + YAML templates for 7 CI systems
- Wire-compatible with real AWS providers via endpoint redirect as an alternative path
- Everything in Go, monorepo

## Architecture

### Code Generation Pipeline

```
pkg/schema/registry.go (unified schemas for 98 services)
        │
        ▼
codegen/main.go (reads schemas, emits code)
        │
        ├──▶ providers/terraform/generated/   (~200 resource files via SDKv2)
        ├──▶ providers/crossplane/config/     (upjet config files — upjet then generates CRDs)
        ├──▶ providers/pulumi/provider.go     (tfbridge mapping from Terraform schema)
        └──▶ custodian/scanner/rules.go       (compliance rule definitions)

Crossplane generation is a two-step process:
  1. codegen emits upjet configuration (resource overrides, external names, group mappings)
  2. upjet reads Terraform provider binary schema and generates CRDs + controllers
```

### Schema Registry Bootstrap

The current codebase has no schema extraction capability. Building the registry requires two foundational changes:

**1. New `SchemaProvider` interface (opt-in, not breaking):**
```go
// pkg/service/schema.go
type SchemaProvider interface {
    ResourceSchemas() []schema.ResourceSchema
}
```
Each Tier 1 service implements this interface alongside `Service`. The schema registry checks for `SchemaProvider` via type assertion — services that don't implement it are skipped (not a breaking change to the `Service` interface). All 23 Tier 1 services must be retrofitted to implement `SchemaProvider`.

**2. Augmented `ServiceModel` for Tier 2 stubs:**
The current `Field` type in `pkg/stub/model.go` only has Name, Type, Required. The following fields must be added: `Computed bool`, `ForceNew bool`, `Default interface{}`, `RefTo string`. The `ResourceType` must gain `ArnPattern`, `ImportID`, and `References []ResourceRef`. All 74 stub model definitions in `services/stubs/catalog.go` must be updated with this metadata.

**Retrofitting scope:** ~23 Tier 1 services × ~20 lines each (schema method) + ~74 Tier 2 models × ~5 additional field annotations each. This is mechanical work suitable for code generation or a single pass.

### Schema Registry

Single source of truth combining Tier 1 service schemas (extracted via opt-in `SchemaProvider` interface) and Tier 2 stub schemas (from augmented `ServiceModel` definitions).

```go
type ResourceSchema struct {
    ServiceName    string
    ResourceType   string              // e.g., "aws_s3_bucket"
    TerraformType  string              // e.g., "cloudmock_s3_bucket"
    AWSType        string              // e.g., "AWS::S3::Bucket"
    Attributes     []AttributeSchema
    CreateAction   string
    ReadAction     string
    UpdateAction   string
    DeleteAction   string
    ListAction     string
    ImportID       string              // which attribute is the import key
    References     []ResourceRef       // cross-resource references
}

type AttributeSchema struct {
    Name       string
    Type       string    // string, int, bool, list, map, set
    Required   bool
    Computed   bool      // server-generated (ARN, ID, timestamps)
    ForceNew   bool      // changing requires replacement
    Default    interface{}
    RefTo      string    // references another resource (e.g., "aws_vpc.id")
}

type ResourceRef struct {
    FromAttr   string   // attribute on this resource
    ToResource string   // target resource type
    ToAttr     string   // target attribute
}
```

## EC2/VPC — Tier 1 Promotion

EC2 is promoted from Tier 2 stub to full Tier 1 service with complete VPC networking.

### Protocol
Query-string/form-encoded with `Action` parameter. XML responses. Namespace: `http://ec2.amazonaws.com/doc/2016-11-15/`

### Resources

| Resource | ID Format | Key References |
|----------|-----------|---------------|
| VPC | vpc-{17hex} | Creates default RT + SG + NACL on create |
| Subnet | subnet-{17hex} | VPC ID, CIDR within VPC CIDR, AZ assignment |
| Security Group | sg-{17hex} | VPC ID, default egress on create |
| Security Group Rule | sgr-{17hex} | SG ID, referenced SG or CIDR |
| Route Table | rtb-{17hex} | VPC ID |
| Route Table Association | rtbassoc-{17hex} | RT ID, Subnet ID |
| Internet Gateway | igw-{17hex} | 1:1 VPC attachment |
| NAT Gateway | nat-{17hex} | Subnet ID, EIP Allocation ID |
| Elastic IP | eipalloc-{17hex} | Association to instance or ENI |
| Network ACL | acl-{17hex} | VPC ID, ordered rules |
| VPC Endpoint | vpce-{17hex} | VPC ID, Service Name, Gateway or Interface |
| Network Interface | eni-{17hex} | Subnet ID, SG IDs, private IP from CIDR |
| VPC Peering | pcx-{17hex} | Requester VPC, Accepter VPC |
| Instance | i-{17hex} | Subnet, SGs, VPC, state machine |

### Actions (50+)

**VPC Core:** CreateVpc, DescribeVpcs, DeleteVpc, ModifyVpcAttribute, CreateSubnet, DescribeSubnets, DeleteSubnet, CreateInternetGateway, AttachInternetGateway, DetachInternetGateway, DeleteInternetGateway, CreateNatGateway, DescribeNatGateways, DeleteNatGateway

**Route Tables:** CreateRouteTable, DescribeRouteTables, DeleteRouteTable, CreateRoute, DeleteRoute, ReplaceRoute, AssociateRouteTable, DisassociateRouteTable

**Security Groups:** CreateSecurityGroup, DescribeSecurityGroups, DeleteSecurityGroup, AuthorizeSecurityGroupIngress, AuthorizeSecurityGroupEgress, RevokeSecurityGroupIngress, RevokeSecurityGroupEgress

**Network:** AllocateAddress, ReleaseAddress, AssociateAddress, DisassociateAddress, CreateNetworkInterface, DescribeNetworkInterfaces, DeleteNetworkInterface, CreateNetworkAcl, DescribeNetworkAcls, DeleteNetworkAcl, CreateNetworkAclEntry, DeleteNetworkAclEntry, CreateVpcEndpoint, DescribeVpcEndpoints, DeleteVpcEndpoints, CreateVpcPeeringConnection, AcceptVpcPeeringConnection, DeleteVpcPeeringConnection

**Instances:** RunInstances, DescribeInstances, TerminateInstances, StopInstances, StartInstances, DescribeInstanceStatus

**Tagging:** CreateTags, DeleteTags, DescribeTags

### Referential Integrity
- Subnet CIDR must fall within parent VPC CIDR
- Security group rules validate referenced SG exists
- Route targets validate gateway/endpoint exists
- Delete operations return `DependencyViolation` if dependents exist
- DeleteVpc fails if subnets, IGWs, or ENIs still exist
- Cross-service IAM referential integrity (instance profiles, service-linked roles) is NOT enforced — EC2 accepts any role ARN string without validating it exists in IAM. This is a deliberate simplification.

### Default VPC
Auto-created at EC2 service startup (matching AWS behavior where default VPC exists at account creation time, not on first API call). This avoids read operations (DescribeVpcs) having side effects and prevents Terraform data sources from triggering creation during plan.
- VPC 172.31.0.0/16 with `isDefault=true`
- 3 subnets (172.31.0.0/20, 172.31.16.0/20, 172.31.32.0/20) in us-east-1a/b/c
- Default route table with local + IGW routes
- Default security group (self-referencing ingress, all egress)
- Default NACL (allow-all)
- Internet gateway attached

## Terraform Provider

**Name:** `terraform-provider-cloudmock`
**Framework:** `terraform-plugin-sdk/v2` (not plugin-framework)

Note: We use SDKv2 rather than the newer plugin-framework because Pulumi's tfbridge requires SDKv2 for full compatibility. The tfbridge SDK (v3.x) has experimental plugin-framework support but SDKv2 is the stable, proven path for bridging.

### Provider Configuration
```hcl
provider "cloudmock" {
  endpoint   = "http://localhost:4566"
  region     = "us-east-1"
  access_key = "test"
  secret_key = "test"
}
```

### Resource Naming
`cloudmock_<service>_<resource>` — e.g., `cloudmock_s3_bucket`, `cloudmock_vpc`, `cloudmock_dynamodb_table`

### Generated Per Resource (~200+ total, one per resource type across 98 services)
EC2 alone has 14 resource types (VPC, Subnet, SecurityGroup, etc.). S3 has bucket + bucket_policy + bucket_notification. Most services have 1-3 resource types. Total estimated: ~200 resources.
- Schema definition (attributes with types, required/computed/optional)
- CRUD handlers mapping to cloudmock API calls
- Import support by ID
- Cross-resource references as Terraform references

### Data Sources
Generated for all describe/list operations — e.g., `data.cloudmock_vpc.selected`

### Provider-to-cloudmock Communication
- HTTP client: AWS SDK Go v2 configured with cloudmock endpoint (reuses SDK's request signing, serialization, error handling)
- Authentication: SigV4 with configurable access key/secret (default test/test)
- All requests go through the gateway (port 4566), not direct service endpoints
- Timeouts: 30s default, configurable per provider block
- Retries: disabled by default (cloudmock is local, retries mask bugs)

### Acceptance Tests
Each resource gets a test that runs against live cloudmock: create → read → update → delete → import.

### AWS Provider Compatibility
Also works via real AWS provider endpoint redirect:
```hcl
provider "aws" {
  endpoints { s3 = "http://localhost:4566" }
  access_key                  = "test"
  secret_key                  = "test"
  region                      = "us-east-1"
  skip_credentials_validation = true
  skip_metadata_api_check     = true
  skip_requesting_account_id  = true
}
```

## Pulumi Provider

**Name:** `pulumi-resource-cloudmock`
**Approach:** Terraform Bridge (`tfbridge`)

Wraps the Terraform provider automatically. Generates language SDKs:
- TypeScript: `@cloudmock/pulumi` (npm)
- Python: `pulumi-cloudmock` (PyPI)
- Go: `github.com/neureaux/cloudmock/providers/pulumi/sdk/go`
- C#, Java, YAML

### Usage
```typescript
import * as cloudmock from "@cloudmock/pulumi";

const vpc = new cloudmock.ec2.Vpc("main", { cidrBlock: "10.0.0.0/16" });
const subnet = new cloudmock.ec2.Subnet("public", {
    vpcId: vpc.id,
    cidrBlock: "10.0.1.0/24",
});
```

### Provider Configuration
```typescript
const provider = new cloudmock.Provider("local", {
    endpoint: "http://localhost:4566",
    region: "us-east-1",
});
```

## Crossplane Provider

### Mode 1: AWS Provider Config (Quick Start)
Configuration package patching `provider-aws` to use cloudmock endpoints. Includes Helm chart for deploying cloudmock into Kubernetes.

```yaml
apiVersion: aws.upbound.io/v1beta1
kind: ProviderConfig
metadata:
  name: cloudmock
spec:
  endpoint:
    url:
      type: Static
      static: http://cloudmock:4566
  credentials:
    source: Secret
    secretRef:
      name: cloudmock-creds
      namespace: crossplane-system
      key: credentials
```

### Mode 2: Native Provider (`provider-cloudmock`)
Built using `upjet` v1.x (wraps Terraform provider as Crossplane provider).

**upjet workflow:**
1. Build and install `terraform-provider-cloudmock` locally
2. upjet reads the provider's schema via `terraform providers schema -json` (requires Terraform binary)
3. upjet generates Go controllers + CRD YAML for each Terraform resource
4. The code generator (`codegen/crossplane.go`) generates upjet configuration files (resource overrides, external name configs, API group mappings), NOT the CRDs directly
5. upjet then generates the final CRDs + controllers from those configs

**API group convention:** `<service>.cloudmock.io/v1alpha1` — e.g., `s3.cloudmock.io`, `ec2.cloudmock.io`, `dynamodb.cloudmock.io`. The service name matches the Terraform provider's resource prefix after `cloudmock_`.

**Requirements:** Terraform >= 1.5, upjet v1.4+, local provider binary (not registry-published).

```yaml
apiVersion: s3.cloudmock.io/v1alpha1
kind: Bucket
metadata:
  name: my-bucket
spec:
  forProvider:
    bucket: my-bucket
    region: us-east-1
  providerConfigRef:
    name: cloudmock

---
apiVersion: ec2.cloudmock.io/v1alpha1
kind: VPC
metadata:
  name: main
spec:
  forProvider:
    cidrBlock: "10.0.0.0/16"
  providerConfigRef:
    name: cloudmock
```

Includes Compositions for common patterns (VPC with subnets, EKS cluster, RDS with subnet group).

## Cloud Custodian

### Policy Library (`custodian/policies/`)
Curated YAML policies organized by domain:

| Domain | Policies |
|--------|----------|
| S3 Security | Block public access, require encryption, require versioning |
| IAM Hygiene | Unused keys, overly permissive policies, root usage |
| Network Security | Open SGs, unrestricted SSH/RDP, missing flow logs |
| Encryption | Unencrypted EBS/RDS/DynamoDB, KMS rotation |
| Tagging | Required tags, value validation |
| Cost | Stopped instances, unused EIPs, idle LBs |
| Logging | CloudTrail, CloudWatch alarms, VPC flow logs |

### Plugin (`c7n-cloudmock`)
Minimal Python plugin (required by Custodian's architecture):
- Custom resources: `cloudmock-service`, `cloudmock-request-log`
- Custom filters: `cloudmock-tier`, `cloudmock-stub`
- Custom actions: `cloudmock-reset`, `cloudmock-seed`, `cloudmock-snapshot`
- Auto endpoint configuration

### Compliance Scanner (`custodian/scanner/`)
Go binary with its own rule engine (independent of Cloud Custodian). This is NOT a wrapper around Custodian — it is a standalone tool that queries cloudmock's API directly and evaluates rules written in Go. This avoids Python as a runtime dependency.

The Go scanner and the Custodian policy library serve different audiences:
- **Go scanner:** CI pipelines and developers who want fast, zero-dependency compliance checks
- **Custodian policies:** Teams already using Cloud Custodian who want to test their policies against cloudmock

Rule drift between the two is accepted — the Go scanner covers the most common checks, while Custodian policies can be arbitrarily complex.

```bash
cloudmock-compliance scan --endpoint http://localhost:4566 --format html
cloudmock-compliance scan --format json --output report.json
cloudmock-compliance scan --format junit --output compliance.xml
```

## AWS Native Tool Wrappers

Five Go binaries sharing `tools/common/` for endpoint detection, credential injection, health checks, and exec wrapping.

### Shared Base (`tools/common/`)
```go
// Detect cloudmock, configure environment, exec tool
func ExecWithCloudmock(tool string, args []string) error
func DetectEndpoint() (string, error)
func InjectCredentials(env []string) []string
func WaitForHealth(endpoint string, timeout time.Duration) error
```

### `cloudmock-aws`
Wraps AWS CLI. Sets `AWS_ENDPOINT_URL` + credentials, execs `aws`. Extra commands: `configure` (set up named profile), `reset`, `status`.

### `cloudmock-cdk`
Wraps CDK. Injects endpoint via CDK context/environment. Extra commands: `reset` (clear state + redeploy).

### `cloudmock-sam`
Wraps SAM CLI. Routes `local invoke` to cloudmock Lambda, `deploy` to cloudmock CloudFormation. Extra commands: `logs`.

### `cloudmock-chalice`
Wraps Chalice. Configures deployer endpoints. Creates API Gateway + Lambda + IAM in cloudmock.

### `cloudmock-copilot`
Wraps Copilot. Points at cloudmock ECS/ECR/CloudFormation.

### All wrappers support:
- `--real-aws` flag to bypass cloudmock (requires confirmation prompt: "You are about to run against REAL AWS. Continue? [y/N]". Skippable with `--real-aws --yes` or `CLOUDMOCK_ALLOW_REAL_AWS=1` env var)
- Auto health check before execution
- `--endpoint` flag to override default

## CI Framework

### `cloudmock-ci` Binary
```
cloudmock-ci init --ci <system>           # Scaffold CI config
cloudmock-ci wait --timeout 30s           # Poll health until ready
cloudmock-ci run --tool <terraform|pulumi|cdk|sam|custodian>  # Execute tool
cloudmock-ci seed --file data.json        # Pre-populate test data
cloudmock-ci diff                         # State diff before/after
cloudmock-ci report --format <markdown|json|junit>  # Generate report
```

### Supported CI Systems (7)

| System | Config File | Service Mechanism |
|--------|------------|-------------------|
| GitHub Actions | `.github/workflows/*.yml` | `services:` container |
| GitLab CI | `.gitlab-ci.yml` | `services:` sidecar |
| CircleCI | `.circleci/config.yml` | Secondary Docker image |
| Bitbucket Pipelines | `bitbucket-pipelines.yml` | `definitions.services` |
| Buildkite | `.buildkite/pipeline.yml` | docker-compose plugin |
| CodeBuild | `buildspec.yml` | Docker run in install phase (requires privileged mode for DinD) |
| Travis CI | `.travis.yml` | Docker service |

### Template Features (all systems)
- Start cloudmock as a service container
- Wait for health check
- Run IaC tool (terraform/pulumi/cdk/sam)
- Generate test report (JUnit XML for CI integration)
- PR comment with results (where supported)

### GitHub Actions — Additional
Composite actions for reuse:
- `cloudmock/setup` — Start cloudmock service
- `cloudmock/terraform` — Setup + terraform init/plan/apply + report
- `cloudmock/pulumi` — Setup + pulumi preview/up + report
- `cloudmock/cdk` — Setup + cdk synth/deploy + report

Reusable workflows:
- `cloudmock/ci-terraform` — Complete terraform CI pipeline
- `cloudmock/ci-pulumi` — Complete pulumi CI pipeline

## Project Structure

```
cloudmock/
  # Existing
  cmd/gateway/                    # Gateway binary
  cmd/cloudmock/                  # CLI binary
  pkg/                            # Core packages
  services/                       # Services

  # Schema & Code Generation
  pkg/schema/
    registry.go                   # Unified schema registry
    types.go                      # ResourceSchema, AttributeSchema, ResourceRef
    extract.go                    # Type-assert SchemaProvider on each service, call ResourceSchemas()
    merge.go                      # Combine Tier 1 schemas + Tier 2 stub models. For promoted services (EC2), Tier 1 schema takes precedence and Tier 2 stub is dropped.
  codegen/
    main.go                       # Code generator entrypoint
    terraform.go                  # Emit Terraform resources
    crossplane.go                 # Emit Crossplane CRDs + controllers
    pulumi.go                     # Emit Pulumi bridge config
    custodian.go                  # Emit Custodian resource defs

  # Terraform Provider
  providers/terraform/
    main.go                       # Provider entrypoint
    provider.go                   # Provider configuration
    generated/                    # ~200 generated resource files + data sources
    tests/                        # Acceptance tests

  # Pulumi Provider
  providers/pulumi/
    cmd/pulumi-resource-cloudmock/main.go
    provider.go                   # tfbridge config
    sdk/go/ sdk/nodejs/ sdk/python/

  # Crossplane Provider
  providers/crossplane/
    cmd/provider/main.go          # Controller manager
    config/                       # Metadata + overrides
    generated/                    # upjet-generated CRDs + controllers
    package/crossplane.yaml
    examples/                     # Compositions
  providers/crossplane-aws-config/
    provider-config.yaml
    helm/                         # Helm chart

  # Cloud Custodian
  custodian/
    plugin/                       # c7n-cloudmock (Python, minimal)
    policies/                     # Policy YAML library
    scanner/                      # Go compliance scanner

  # AWS Tool Wrappers
  tools/common/                   # Shared wrapper logic
  tools/cloudmock-aws/main.go
  tools/cloudmock-cdk/main.go
  tools/cloudmock-sam/main.go
  tools/cloudmock-chalice/main.go
  tools/cloudmock-copilot/main.go

  # CI Framework
  tools/cloudmock-ci/
    main.go                       # CI helper binary
    init.go run.go report.go wait.go seed.go diff.go
    templates/                    # Embedded CI YAML templates (7 systems)

  # EC2/VPC (promoted to Tier 1)
  services/ec2/
    service.go handlers.go store.go
    vpc.go security_group.go network.go
    service_test.go
```

## Build Order

1. EC2/VPC deepening (Tier 1 promotion, 50+ actions, default VPC)
2. Schema registry (`pkg/schema/`)
3. Code generator (`codegen/`)
4. Terraform provider (generated from schemas)
5. Pulumi provider (bridge from Terraform)
6. Crossplane provider (upjet from Terraform + AWS provider config)
7. AWS tool wrappers (5 Go binaries)
8. Cloud Custodian (Python plugin + Go scanner + policy library)
9. CI framework (`cloudmock-ci` binary + 7 template sets)

## Non-Goals

- Real container orchestration for ECS/EKS (metadata only)
- Real DNS resolution for Route 53 outside cloudmock
- Real VPC network isolation between resources (all resources share one flat namespace)
- Terraform state locking (single-user local use)
- Pulumi state backend (uses local filesystem)
- Crossplane continuous reconciliation loops (one-shot apply only in testing)
