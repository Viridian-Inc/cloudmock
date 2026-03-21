# STS — Security Token Service

**Tier:** 1 (Full Emulation)
**Protocol:** Query (`Action=<Action>`)
**Service Name:** `sts`

## Supported Actions

| Action | Notes |
|--------|-------|
| `GetCallerIdentity` | Returns the account ID, ARN, and user ID of the caller |
| `AssumeRole` | Returns temporary credentials for the specified role ARN |
| `GetSessionToken` | Returns temporary credentials for the current user |

## Examples

### AWS CLI

```bash
# Get caller identity
aws sts get-caller-identity
# {
#   "UserId": "AKIAIOSFODNN7EXAMPLE",
#   "Account": "000000000000",
#   "Arn": "arn:aws:iam::000000000000:root"
# }

# Assume a role
aws sts assume-role \
  --role-arn arn:aws:iam::000000000000:role/my-role \
  --role-session-name my-session

# Get session token
aws sts get-session-token --duration-seconds 3600
```

### Python (boto3)

```python
import boto3

sts = boto3.client("sts", endpoint_url="http://localhost:4566",
                   aws_access_key_id="test", aws_secret_access_key="test",
                   region_name="us-east-1")

# Verify identity
identity = sts.get_caller_identity()
print(identity["Arn"])  # arn:aws:iam::000000000000:root

# Assume role and use credentials
response = sts.assume_role(
    RoleArn="arn:aws:iam::000000000000:role/my-role",
    RoleSessionName="test",
)
creds = response["Credentials"]

# Use assumed-role credentials with another client
s3 = boto3.client(
    "s3",
    endpoint_url="http://localhost:4566",
    aws_access_key_id=creds["AccessKeyId"],
    aws_secret_access_key=creds["SecretAccessKey"],
    aws_session_token=creds["SessionToken"],
)
```

## Notes

- `AssumeRole` returns synthetic temporary credentials with a configurable expiration (default 1 hour). The returned session token is accepted by the IAM middleware for subsequent requests.
- `GetSessionToken` generates temporary credentials for the current IAM user.
- Cross-account role assumption is accepted but no cross-account isolation exists in the emulator.
