# Terraform + CloudMock

Use Terraform to provision infrastructure against CloudMock. Validate your IaC locally before deploying to AWS.

## Start CloudMock

```bash
docker compose up -d
```

## Run Terraform

Terraform runs on your host (no container needed):

```bash
cd infra
terraform init
terraform plan
terraform apply -auto-approve
```

This creates:
- S3 bucket `my-app-data` (versioning enabled)
- DynamoDB table `items` (on-demand billing)
- SQS queue `jobs` with a dead-letter queue after 3 failures

## Verify

```bash
# Check S3 bucket
aws s3 ls --endpoint-url http://localhost:4566 --no-sign-request

# Check DynamoDB table
aws dynamodb list-tables --endpoint-url http://localhost:4566 --region us-east-1 --no-sign-request

# Check SQS queues
aws sqs list-queues --endpoint-url http://localhost:4566 --region us-east-1 --no-sign-request
```

## Tear down

```bash
terraform destroy -auto-approve
docker compose down
```

## Tips

- Add more resources to `infra/main.tf` — all 99 CloudMock services work as Terraform targets
- Use `terraform output` to get endpoint URLs into your app config
- Commit `terraform.tfstate` to share the infra definition with your team (it points at localhost, so it's safe)
- See the CloudMock [Terraform guide](https://cloudmock.app/docs/guides/terraform) for advanced patterns
