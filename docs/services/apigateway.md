# API Gateway (REST API)

**Tier:** 1 (Full Emulation)
**Protocol:** REST-JSON (path-based routing)
**Service Name:** `apigateway`

## Supported Actions

| Action | Notes |
|--------|-------|
| `CreateRestApi` | Creates a REST API and returns its ID |
| `GetRestApis` | Returns all REST APIs |
| `GetRestApi` | Returns a specific REST API |
| `DeleteRestApi` | Deletes a REST API |
| `CreateResource` | Creates a resource (path part) under a parent |
| `GetResources` | Returns all resources in an API |
| `DeleteResource` | Deletes a resource |
| `PutMethod` | Adds an HTTP method to a resource |
| `GetMethod` | Returns method configuration |
| `PutIntegration` | Associates an integration (Lambda, HTTP, Mock) with a method |
| `CreateDeployment` | Deploys an API to a stage |
| `GetDeployments` | Returns all deployments |
| `CreateStage` | Creates a stage pointing at a deployment |
| `GetStages` | Returns all stages for an API |

## Examples

### AWS CLI

```bash
# Create API
aws apigateway create-rest-api --name "MyAPI"

# Get root resource ID
aws apigateway get-resources --rest-api-id <api-id>

# Create resource
aws apigateway create-resource \
  --rest-api-id <api-id> \
  --parent-id <root-resource-id> \
  --path-part "items"

# Add GET method
aws apigateway put-method \
  --rest-api-id <api-id> \
  --resource-id <resource-id> \
  --http-method GET \
  --authorization-type NONE

# Add mock integration
aws apigateway put-integration \
  --rest-api-id <api-id> \
  --resource-id <resource-id> \
  --http-method GET \
  --type MOCK

# Deploy
aws apigateway create-deployment \
  --rest-api-id <api-id> \
  --stage-name prod

# Create stage
aws apigateway create-stage \
  --rest-api-id <api-id> \
  --stage-name v1 \
  --deployment-id <deployment-id>
```

### Python (boto3)

```python
import boto3

apigw = boto3.client("apigateway", endpoint_url="http://localhost:4566",
                     aws_access_key_id="test", aws_secret_access_key="test",
                     region_name="us-east-1")

# Create API
api = apigw.create_rest_api(name="TestAPI")
api_id = api["id"]

# Get resources (root resource)
resources = apigw.get_resources(restApiId=api_id)
root_id = next(r["id"] for r in resources["items"] if r["path"] == "/")

# Create /hello resource
resource = apigw.create_resource(restApiId=api_id, parentId=root_id, pathPart="hello")
resource_id = resource["id"]

# PUT method and integration
apigw.put_method(restApiId=api_id, resourceId=resource_id,
                 httpMethod="GET", authorizationType="NONE")
apigw.put_integration(restApiId=api_id, resourceId=resource_id,
                      httpMethod="GET", type="MOCK")

# Deploy
deployment = apigw.create_deployment(restApiId=api_id, stageName="dev")
```

## Notes

- API invocation through the emulated endpoint is not supported — only the management API is emulated.
- HTTP and Lambda proxy integrations are recorded but not executed.
- API Gateway v2 (HTTP APIs and WebSocket APIs) is not implemented.
- Usage plans, API keys, and authorizers are not implemented.
