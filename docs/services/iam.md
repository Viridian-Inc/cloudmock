# IAM — Identity and Access Management

**Tier:** 1 (Full Emulation)
**Protocol:** Internal (embedded in gateway)
**Service Name:** `iam`

IAM is implemented as an embedded engine within the gateway rather than as a standalone HTTP service. The IAM store manages users, access keys, and policies. The IAM engine evaluates policies for every request when running in `enforce` mode.

## Supported Operations

| Operation | Description |
|-----------|-------------|
| User management | `CreateUser`, `GetUser`, `DeleteUser` (via store API) |
| Access key management | `CreateAccessKey`, `LookupAccessKey` |
| Policy attachment | `AttachUserPolicy`, `GetUserPolicies` |
| Root credentials | Configured via `iam.root_access_key` / `iam.root_secret_key` |
| IAM seed file | Bulk-load users, keys, and policies at startup |
| Policy evaluation | Allow/Deny with wildcard action and resource matching |

## IAM Seed File

The most practical way to configure IAM in cloudmock is via a seed file:

```json
{
  "users": [
    {
      "name": "ci",
      "access_key_id": "AKIAIOSFODNN7EXAMPLE",
      "secret_access_key": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
      "policies": [
        {
          "name": "AllowAll",
          "document": {
            "Version": "2012-10-17",
            "Statement": [
              {
                "Effect": "Allow",
                "Action": "*",
                "Resource": "*"
              }
            ]
          }
        }
      ]
    },
    {
      "name": "readonly",
      "access_key_id": "AKIAI44QH8DHBEXAMPLE",
      "secret_access_key": "je7MtGbClwBF/2Zp9Utk/h3yCo8nvbEXAMPLEKEY",
      "policies": [
        {
          "name": "ReadOnlyS3",
          "document": {
            "Version": "2012-10-17",
            "Statement": [
              {
                "Effect": "Allow",
                "Action": ["s3:GetObject", "s3:ListBucket", "s3:HeadObject"],
                "Resource": "*"
              }
            ]
          }
        }
      ]
    }
  ]
}
```

Configure the path in `cloudmock.yml`:

```yaml
iam:
  mode: enforce
  seed_file: ./iam-seed.json
```

## Policy Evaluation Rules

1. The root user (`root_access_key` credential) bypasses all policy checks.
2. All policies attached to the calling user are evaluated together.
3. If any matching statement has `Effect: Deny` — the request is denied (explicit deny wins).
4. If any matching statement has `Effect: Allow` — the request is allowed.
5. Otherwise — implicit deny.

## Action Wildcard Examples

```json
"Action": "*"            // all actions on all services
"Action": "s3:*"         // all S3 actions
"Action": "s3:Get*"      // all S3 Get* actions
"Action": ["s3:GetObject", "s3:PutObject"]  // specific actions
```

## Usage with AWS CLI

```bash
# Use root credentials (always allowed)
AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test \
  aws s3 ls

# Use a restricted user's credentials
AWS_ACCESS_KEY_ID=AKIAI44QH8DHBEXAMPLE \
AWS_SECRET_ACCESS_KEY=je7MtGbClwBF/2Zp9Utk/h3yCo8nvbEXAMPLEKEY \
  aws s3 ls                          # succeeds
  aws s3 cp file.txt s3://bucket/    # fails (no PutObject)
```

## IAM Modes

```bash
# Development: skip all auth
CLOUDMOCK_IAM_MODE=none cloudmock start

# CI: verify credentials, skip policy eval
CLOUDMOCK_IAM_MODE=authenticate cloudmock start

# Production-like: full evaluation
CLOUDMOCK_IAM_MODE=enforce cloudmock start
```
