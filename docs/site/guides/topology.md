# Topology View

CloudMock auto-discovers your Infrastructure-as-Code to build a visual map of your architecture. The topology view shows services, resources, and connections in the DevTools dashboard.

## How It Works

CloudMock reads your Pulumi or Terraform project and extracts:

- **Resources** -- S3 buckets, DynamoDB tables, Lambda functions, SQS queues, etc.
- **Connections** -- which resources reference each other (Lambda -> DynamoDB, API Gateway -> Lambda, etc.)
- **Service groups** -- resources grouped by AWS service

This builds a live topology map in DevTools, overlaid with real-time request data.

## Auto-Discovery

CloudMock searches for IaC projects automatically:

1. `Pulumi.yaml` or `Pulumi.*.yaml` in current and parent directories
2. `terraform/` directory containing `.tf` files
3. `infra/pulumi/` directory pattern

```bash
cmk start
# Output: Discovered pulumi project at ./infra
```

Or configure explicitly:

```yaml
# .cloudmock.yaml
iac:
  path: ./infrastructure/pulumi
  env: dev
```

## Viewing the Topology

Open DevTools (`http://localhost:4500`) and navigate to the Topology tab.

### What You See

- **Nodes** -- each AWS resource (S3 bucket, DynamoDB table, Lambda function, etc.)
- **Edges** -- connections between resources (event triggers, IAM references, VPC associations)
- **Request flow** -- animated lines showing live traffic between resources
- **Health indicators** -- green/yellow/red status based on error rates and SLO compliance
- **Cost annotations** -- estimated per-resource costs

### Interacting

- Click a resource node to see its configuration, recent requests, and metrics
- Hover over an edge to see the relationship type and traffic volume
- Use the service filter to focus on specific AWS services
- Zoom and pan to navigate large architectures

## API Access

### Get Full Topology

```bash
curl http://localhost:4599/api/topology | jq '.'
```

Response:

```json
{
  "nodes": [
    {
      "id": "lambda:order-processor",
      "service": "lambda",
      "type": "Function",
      "name": "order-processor",
      "arn": "arn:aws:lambda:us-east-1:000000000000:function:order-processor"
    },
    {
      "id": "dynamodb:orders",
      "service": "dynamodb",
      "type": "Table",
      "name": "orders",
      "arn": "arn:aws:dynamodb:us-east-1:000000000000:table/orders"
    }
  ],
  "edges": [
    {
      "source": "lambda:order-processor",
      "target": "dynamodb:orders",
      "type": "reads_from",
      "label": "DynamoDB Query"
    }
  ]
}
```

### Get Topology Configuration

```bash
curl http://localhost:4599/api/topology/config | jq '.'
```

Returns the IaC source path and detected resources.

### Get Resources by Service

```bash
curl http://localhost:4599/api/resources/lambda | jq '.'
curl http://localhost:4599/api/resources/dynamodb | jq '.'
curl http://localhost:4599/api/resources/s3 | jq '.'
```

## Blast Radius Analysis

See what's affected when a resource fails:

```bash
curl -X POST http://localhost:4599/api/blast-radius \
  -H "Content-Type: application/json" \
  -d '{"resource": "dynamodb:orders"}'
```

Response:

```json
{
  "resource": "dynamodb:orders",
  "affected": [
    {"id": "lambda:order-processor", "impact": "direct", "reason": "reads from table"},
    {"id": "lambda:order-api", "impact": "indirect", "reason": "invokes order-processor"},
    {"id": "apigateway:orders-api", "impact": "indirect", "reason": "routes to order-api"}
  ],
  "total_affected": 3
}
```

This is useful for understanding the impact of a failure or planned maintenance.

## Topology Without IaC

If you don't use Pulumi or Terraform, CloudMock still builds a topology from the AWS resources you create at runtime. Every `CreateTable`, `CreateFunction`, `CreateBucket`, etc. call adds a node to the graph.

Connections are inferred from:

- Lambda event source mappings (SQS, Kinesis, DynamoDB Streams)
- API Gateway route integrations
- Step Functions state machine definitions
- SNS topic subscriptions
- IAM policy resource references
