# cloudmock IaC, AWS Tools & CI Framework Design

**Date:** 2026-03-21
**Status:** Draft
**License:** Apache 2.0

## Overview

Extend cloudmock with Infrastructure-as-Code provider support (Terraform, Pulumi, Crossplane), Cloud Custodian compliance tooling, AWS native tool wrappers (CLI, CDK, SAM, Chalice, Copilot), and a CI framework supporting 7 CI systems (GitHub Actions, GitLab CI, CircleCI, Bitbucket Pipelines, Buildkite, CodeBuild, Travis CI). All components are written in Go (except a minimal Python plugin for Cloud Custodian). Everything lives in the cloudmock monorepo.

## Goals

- Code-generated providers from a unified schema registry — add a service once, all providers update
- Terraform provider with 98 resources + data sources for all cloudmock services
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
        ├──▶ providers/terraform/generated/   (98 resource files)
        ├──▶ providers/crossplane/generated/   (CRDs + controllers)
        ├──▶ providers/pulumi/ (bridge config from terraform schema)
        └──▶ custodian/scanner/rules.go (resource definitions)
```

### Schema Registry

Single source of truth combining Tier 1 service schemas (extracted from service code via `Schema()` method) and Tier 2 stub schemas (from `ServiceModel` definitions).

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

### Default VPC
Auto-created on first EC2 API call:
- VPC 172.31.0.0/16 with `isDefault=true`
- 3 subnets (172.31.0.0/20, 172.31.16.0/20, 172.31.32.0/20) in us-east-1a/b/c
- Default route table with local + IGW routes
- Default security group (self-referencing ingress, all egress)
- Default NACL (allow-all)
- Internet gateway attached

## Terraform Provider

**Name:** `terraform-provider-cloudmock`
**Framework:** `terraform-plugin-framework`

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

### Generated Per Resource (98 total)
- Schema definition (attributes with types, required/computed/optional)
- CRUD handlers mapping to cloudmock API calls
- Import support by ID
- Cross-resource references as Terraform references

### Data Sources
Generated for all describe/list operations — e.g., `data.cloudmock_vpc.selected`

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
Built using `upjet` (wraps Terraform provider as Crossplane provider).

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
Go binary that runs all policies and generates reports:
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
- `--real-aws` flag to bypass cloudmock
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
| CodeBuild | `buildspec.yml` | Docker run in install phase |
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
    extract.go                    # Extract schemas from Tier 1 services
    merge.go                      # Merge Tier 1 + Tier 2 schemas
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
    generated/                    # 98 generated resource files + data sources
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
