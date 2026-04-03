---
title: Multi-Account Support
description: Simulate multiple AWS accounts with per-account resource isolation and cross-account STS AssumeRole.
---

CloudMock supports multi-account AWS environments where each account has isolated resources. This is essential for testing landing zone architectures, cross-account IAM roles, and Organizations-based workflows.

## Configuration

Define accounts in your `cloudmock.yml`:

```yaml
region: us-east-1
account_id: "111111111111"

accounts:
  - id: "222222222222"
    name: "Development"
  - id: "333333333333"
    name: "Staging"
  - id: "444444444444"
    name: "Production"
```

The `account_id` field is the management (default) account. Each entry in `accounts` provisions an additional isolated account with its own service instances.

## STS AssumeRole Across Accounts

Cross-account role assumption works like real AWS. When you call `sts:AssumeRole` with a role ARN targeting a different account, the returned temporary credentials are bound to that account:

```python
import boto3

# Start with credentials for account 111111111111
sts = boto3.client('sts', endpoint_url='http://localhost:4566')

# Assume a role in the development account
response = sts.assume_role(
    RoleArn='arn:aws:iam::222222222222:role/DevAdmin',
    RoleSessionName='cross-account-session'
)

# Use the temporary credentials — requests now target account 222222222222
dev_session = boto3.Session(
    aws_access_key_id=response['Credentials']['AccessKeyId'],
    aws_secret_access_key=response['Credentials']['SecretAccessKey'],
    aws_session_token=response['Credentials']['SessionToken'],
)

# This S3 client operates in the dev account's isolated namespace
s3 = dev_session.client('s3', endpoint_url='http://localhost:4566')
s3.create_bucket(Bucket='dev-data')
```

## Resource Isolation

Each account gets independent service instances. A DynamoDB table created in account `222222222222` is not visible from account `333333333333`. This matches real AWS behavior where accounts are hard isolation boundaries.

Services are created lazily -- only when a request targets a specific account and service combination. This keeps memory usage low even with many accounts configured.

## Organizations Integration

When multi-account mode is active, the Organizations `CreateAccount` API automatically provisions a new isolated account in the registry:

```python
orgs = boto3.client('organizations', endpoint_url='http://localhost:4566')

# Create organization first
orgs.create_organization(FeatureSet='ALL')

# This both records the account in Organizations AND provisions it
# in the account registry with isolated services
response = orgs.create_account(
    AccountName='New Team Account',
    Email='team@example.com'
)

new_account_id = response['CreateAccountStatus']['AccountId']
# You can now AssumeRole into this account
```

## Testing Landing Zone Architectures

Multi-account support is designed for testing Control Tower and landing zone patterns:

1. **Management account** -- runs Organizations, creates OUs and SCPs
2. **Security account** -- centralized CloudTrail, GuardDuty
3. **Shared services account** -- shared VPCs, Transit Gateway
4. **Workload accounts** -- application resources

```yaml
accounts:
  - id: "222222222222"
    name: "Security"
  - id: "333333333333"
    name: "Shared Services"
  - id: "444444444444"
    name: "Workload-Dev"
  - id: "555555555555"
    name: "Workload-Prod"
```

## Backward Compatibility

Multi-account mode is opt-in. When no `accounts` are configured in `cloudmock.yml`, everything works exactly as before with a single shared account. The feature activates only when the `accounts` list is non-empty.
