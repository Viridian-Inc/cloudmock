# Secrets Manager

**Tier:** 1 (Full Emulation)
**Protocol:** JSON (`X-Amz-Target: secretsmanager.<Action>`)
**Service Name:** `secretsmanager`

## Supported Actions

| Action | Notes |
|--------|-------|
| `CreateSecret` | Creates a secret with string or binary value |
| `GetSecretValue` | Returns the current secret value |
| `PutSecretValue` | Adds a new version of the secret |
| `UpdateSecret` | Updates secret metadata (description, KMS key) |
| `DeleteSecret` | Marks the secret for deletion (immediate in emulator) |
| `RestoreSecret` | Cancels a pending deletion |
| `DescribeSecret` | Returns secret metadata without the value |
| `ListSecrets` | Returns all secrets |
| `TagResource` | Adds tags to a secret |
| `UntagResource` | Removes tags from a secret |

## Examples

### AWS CLI

```bash
# Create a secret
aws secretsmanager create-secret \
  --name /app/db-password \
  --secret-string "supersecret"

# Get the value
aws secretsmanager get-secret-value \
  --secret-id /app/db-password

# Update the value
aws secretsmanager put-secret-value \
  --secret-id /app/db-password \
  --secret-string "newpassword"

# List secrets
aws secretsmanager list-secrets

# Delete a secret
aws secretsmanager delete-secret \
  --secret-id /app/db-password \
  --force-delete-without-recovery
```

### Python (boto3)

```python
import boto3, json

sm = boto3.client("secretsmanager", endpoint_url="http://localhost:4566",
                  aws_access_key_id="test", aws_secret_access_key="test",
                  region_name="us-east-1")

# Create a JSON secret
sm.create_secret(
    Name="/app/config",
    SecretString=json.dumps({"host": "db.local", "password": "s3cr3t"}),
)

# Retrieve and parse
response = sm.get_secret_value(SecretId="/app/config")
config = json.loads(response["SecretString"])
print(config["host"])  # db.local
```

## Notes

- Secret versioning is tracked via version IDs but only the latest version is accessible without specifying a version ID.
- Automatic secret rotation is not implemented.
- Binary secrets (`SecretBinary`) are stored but returned as-is without base64 processing.
