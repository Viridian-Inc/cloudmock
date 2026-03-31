# CloudMock

Local AWS. 25 services. One binary.

## Install

```bash
npx cloudmock                   # zero install
brew install cloudmock           # macOS / Linux
docker run -p 4566:4566 -p 4500:4500 ghcr.io/neureaux/cloudmock
go install github.com/neureaux/cloudmock/cmd/gateway@latest
```

## Point your SDK

### Node.js

```js
const { S3Client, CreateBucketCommand } = require("@aws-sdk/client-s3");

const client = new S3Client({
  endpoint: "http://localhost:4566",
  region: "us-east-1",
  credentials: { accessKeyId: "test", secretAccessKey: "test" },
  forcePathStyle: true,
});

await client.send(new CreateBucketCommand({ Bucket: "my-bucket" }));
```

### Python

```python
import boto3

s3 = boto3.client(
    "s3",
    endpoint_url="http://localhost:4566",
    aws_access_key_id="test",
    aws_secret_access_key="test",
    region_name="us-east-1",
)

s3.create_bucket(Bucket="my-bucket")
```

### Go

```go
cfg, _ := config.LoadDefaultConfig(context.TODO(),
    config.WithRegion("us-east-1"),
    config.WithBaseEndpoint("http://localhost:4566"),
    config.WithCredentialsProvider(aws.CredentialsProviderFunc(
        func(ctx context.Context) (aws.Credentials, error) {
            return aws.Credentials{
                AccessKeyID: "test", SecretAccessKey: "test",
            }, nil
        },
    )),
)
client := s3.NewFromConfig(cfg)
```

## cmk CLI

`cmk` wraps the AWS CLI with `--endpoint-url` pointed at CloudMock.

```bash
# Instead of:
aws --endpoint-url=http://localhost:4566 s3 ls

# Use:
cmk s3 ls
cmk dynamodb list-tables
cmk sqs create-queue --queue-name jobs
```

Set `CLOUDMOCK_ENDPOINT` to override the default (`http://localhost:4566`).

## Devtools

Open http://localhost:4500 for the built-in devtools UI. It shows service topology, request traces, metrics, and chaos injection controls.

## Services

25 Tier 1 services with full API emulation. 429 operations total.

| Service | Endpoint name | Operations |
|---|---|---|
| API Gateway | `apigateway` | 14 |
| CloudFormation | `cloudformation` | 13 |
| CloudWatch | `monitoring` | 11 |
| CloudWatch Logs | `logs` | 14 |
| Cognito | `cognito-idp` | 14 |
| DynamoDB | `dynamodb` | 28 |
| EC2 | `ec2` | 90 |
| ECR | `ecr` | 12 |
| ECS | `ecs` | 20 |
| EventBridge | `events` | 17 |
| IAM | `iam` | 37 |
| Kinesis | `kinesis` | 13 |
| KMS | `kms` | 10 |
| Lambda | `lambda` | 12 |
| Data Firehose | `firehose` | 10 |
| RDS | `rds` | 16 |
| Route 53 | `route53` | 6 |
| S3 | `s3` | 27 |
| Secrets Manager | `secretsmanager` | 10 |
| SES | `ses` | 7 |
| SNS | `sns` | 12 |
| SQS | `sqs` | 13 |
| SSM Parameter Store | `ssm` | 7 |
| Step Functions | `states` | 13 |
| STS | `sts` | 3 |

An additional 73 services are available as CRUD stubs (create, describe, list, delete). See [docs/compatibility-matrix.md](docs/compatibility-matrix.md) for the full list.

## Configuration

CloudMock reads `cloudmock.yml` from the working directory. Profiles (`minimal`, `standard`, `full`) control which services start. Ports for the gateway (default 4566), devtools (4500), and admin API (4599) are configurable. Persistent state snapshots can be enabled to survive restarts.

See [docs/configuration.md](docs/configuration.md) for the full reference.

## Links

- [Documentation](https://cloudmock.io/docs)
- [GitHub Issues](https://github.com/neureaux/cloudmock/issues)
- [Comparison with LocalStack, Moto, et al.](https://cloudmock.io/docs/reference/comparison)

## Contributing

1. Fork the repository and create a feature branch.
2. Run tests: `make test`
3. Add or update service code under `services/<name>/`.
4. Submit a pull request.

See [docs/architecture.md](docs/architecture.md) for how the service framework works.

## License

Apache License 2.0. See [LICENSE](LICENSE).
