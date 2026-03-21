# SSM — Systems Manager Parameter Store

**Tier:** 1 (Full Emulation)
**Protocol:** JSON (`X-Amz-Target: AmazonSSM.<Action>`)
**Service Name:** `ssm`

## Supported Actions

| Action | Notes |
|--------|-------|
| `PutParameter` | Creates or updates a parameter (String, StringList, SecureString) |
| `GetParameter` | Returns a single parameter by name |
| `GetParameters` | Returns multiple parameters by name list |
| `GetParametersByPath` | Returns all parameters under a path prefix |
| `DeleteParameter` | Deletes a single parameter |
| `DeleteParameters` | Deletes multiple parameters |
| `DescribeParameters` | Returns parameter metadata (no values) |

## Examples

### AWS CLI

```bash
# Create a String parameter
aws ssm put-parameter \
  --name /app/env \
  --value production \
  --type String

# Create a SecureString parameter
aws ssm put-parameter \
  --name /app/api-key \
  --value "abc123" \
  --type SecureString

# Get a single parameter
aws ssm get-parameter --name /app/env

# Get with decryption (SecureString)
aws ssm get-parameter --name /app/api-key --with-decryption

# Get all parameters under a path
aws ssm get-parameters-by-path --path /app

# Delete a parameter
aws ssm delete-parameter --name /app/env
```

### Python (boto3)

```python
import boto3

ssm = boto3.client("ssm", endpoint_url="http://localhost:4566",
                   aws_access_key_id="test", aws_secret_access_key="test",
                   region_name="us-east-1")

# Put parameters
ssm.put_parameter(Name="/service/db-host", Value="localhost", Type="String")
ssm.put_parameter(Name="/service/db-pass", Value="secret", Type="SecureString")

# Get by path
response = ssm.get_parameters_by_path(Path="/service", WithDecryption=True)
for param in response["Parameters"]:
    print(param["Name"], "=", param["Value"])

# Get multiple at once
response = ssm.get_parameters(
    Names=["/service/db-host", "/service/db-pass"],
    WithDecryption=True,
)
params = {p["Name"]: p["Value"] for p in response["Parameters"]}
```

## Notes

- SecureString type is accepted and stored; encryption is simulated (not cryptographically secure).
- `GetParametersByPath` supports recursive path lookups.
- Parameter versioning stores each `PutParameter` call as a new version; the latest version is returned by default.
- Parameter history retrieval (`GetParameterHistory`) is not implemented.
