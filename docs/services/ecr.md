# ECR — Elastic Container Registry

**Tier:** 1 (Full Emulation)
**Protocol:** JSON (`X-Amz-Target: AmazonEC2ContainerRegistry_V20150921.<Action>`)
**Service Name:** `ecr`

## Supported Actions

| Action | Notes |
|--------|-------|
| `CreateRepository` | Creates a container image repository |
| `DeleteRepository` | Deletes a repository |
| `DescribeRepositories` | Returns repository metadata |
| `ListImages` | Returns image tags in a repository |
| `BatchGetImage` | Returns image manifests by tag or digest |
| `PutImage` | Stores an image manifest |
| `BatchDeleteImage` | Deletes images by tag or digest |
| `GetAuthorizationToken` | Returns a Docker login token |
| `DescribeImageScanFindings` | Returns a stub scan result |
| `TagResource` | Adds tags to a repository |
| `UntagResource` | Removes tags |
| `ListTagsForResource` | Returns tags for a repository |

## Examples

### AWS CLI

```bash
# Create a repository
aws ecr create-repository --repository-name my-app

# Get login token
aws ecr get-authorization-token \
  --query 'authorizationData[0].authorizationToken' \
  --output text | base64 --decode
# Output: AWS:<token>

# Log in Docker
aws ecr get-login-password | \
  docker login --username AWS --password-stdin \
  000000000000.dkr.ecr.us-east-1.localhost:4566

# List images
aws ecr list-images --repository-name my-app

# Delete an image
aws ecr batch-delete-image \
  --repository-name my-app \
  --image-ids imageTag=latest
```

### Python (boto3)

```python
import boto3

ecr = boto3.client("ecr", endpoint_url="http://localhost:4566",
                   aws_access_key_id="test", aws_secret_access_key="test",
                   region_name="us-east-1")

# Create repo
repo = ecr.create_repository(repositoryName="backend")
uri = repo["repository"]["repositoryUri"]
print(uri)  # 000000000000.dkr.ecr.us-east-1.localhost:4566/backend

# Get auth token
token_resp = ecr.get_authorization_token()
token = token_resp["authorizationData"][0]["authorizationToken"]

# List images
images = ecr.list_images(repositoryName="backend")
for img in images.get("imageIds", []):
    print(img)
```

## Notes

- `GetAuthorizationToken` returns a synthetic token. Docker push/pull to the cloudmock ECR endpoint requires a container registry proxy and is not natively supported.
- Image layers are not stored. `PutImage` and `BatchGetImage` operate on manifest metadata only.
- Image vulnerability scanning results from `DescribeImageScanFindings` are stubs.
- Lifecycle policies are not implemented.
