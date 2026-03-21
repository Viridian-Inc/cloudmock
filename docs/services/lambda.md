# Lambda

**Tier:** 1 (Full Emulation — resource management; invocation is a stub)
**Protocol:** REST-JSON
**Service Name:** `lambda`

> **Note:** Lambda function management (create, update, delete, list) is fully implemented. Actual function invocation executes a no-op stub that returns an empty 200 response. Runtime execution is not supported.

## Supported Actions

| Action | Notes |
|--------|-------|
| `CreateFunction` | Stores function configuration and code reference |
| `DeleteFunction` | Removes a function |
| `GetFunction` | Returns function configuration |
| `ListFunctions` | Returns all functions |
| `UpdateFunctionCode` | Updates the code reference |
| `UpdateFunctionConfiguration` | Updates runtime, handler, environment, etc. |
| `InvokeFunction` | Returns a stub 200 response; does not execute code |
| `AddPermission` | Stores a resource-based policy statement |
| `RemovePermission` | Removes a policy statement |
| `CreateEventSourceMapping` | Stores an event source mapping |
| `ListEventSourceMappings` | Returns all event source mappings |
| `TagResource` | Adds tags to a function |
| `UntagResource` | Removes tags from a function |

## Examples

### AWS CLI

```bash
# Create a function (zip is stored but not executed)
aws lambda create-function \
  --function-name my-function \
  --runtime nodejs20.x \
  --role arn:aws:iam::000000000000:role/lambda-role \
  --handler index.handler \
  --zip-file fileb://function.zip

# List functions
aws lambda list-functions

# Get function details
aws lambda get-function --function-name my-function

# Invoke (returns stub response)
aws lambda invoke \
  --function-name my-function \
  --payload '{"key":"value"}' \
  output.json

# Delete function
aws lambda delete-function --function-name my-function
```

### Python (boto3)

```python
import boto3, zipfile, io

client = boto3.client("lambda", endpoint_url="http://localhost:4566",
                      aws_access_key_id="test", aws_secret_access_key="test",
                      region_name="us-east-1")

# Create a minimal zip
buf = io.BytesIO()
with zipfile.ZipFile(buf, "w") as zf:
    zf.writestr("index.js", "exports.handler = async () => ({ statusCode: 200 });")
buf.seek(0)

client.create_function(
    FunctionName="hello",
    Runtime="nodejs20.x",
    Role="arn:aws:iam::000000000000:role/lambda-role",
    Handler="index.handler",
    Code={"ZipFile": buf.read()},
)

# Invoke (stub — returns empty payload)
response = client.invoke(FunctionName="hello", Payload=b"{}")
print(response["StatusCode"])  # 200
```

## Notes

- `runtimes` per-service configuration is accepted but has no effect on execution.
- Event source mappings are stored and returned by `ListEventSourceMappings` but no trigger logic runs.
- To test Lambda-triggered workflows, invoke your function logic directly in your test code and use cloudmock for the downstream services (SQS, DynamoDB, S3, etc.).
