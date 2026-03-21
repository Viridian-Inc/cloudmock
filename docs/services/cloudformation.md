# CloudFormation

**Tier:** 1 (Full Emulation)
**Protocol:** Query (`Action=<Action>`)
**Service Name:** `cloudformation`

## Supported Actions

| Action | Notes |
|--------|-------|
| `CreateStack` | Creates a stack from a template |
| `DeleteStack` | Deletes a stack |
| `DescribeStacks` | Returns stack metadata and outputs |
| `ListStacks` | Returns stack summaries with optional status filter |
| `DescribeStackResources` | Returns the resources in a stack |
| `DescribeStackEvents` | Returns the event history for a stack |
| `GetTemplate` | Returns the template body for a stack |
| `ValidateTemplate` | Validates a template and returns parameter names |
| `ListExports` | Returns all stack exports |
| `CreateChangeSet` | Creates a change set for a stack |
| `DescribeChangeSet` | Returns change set details |
| `ExecuteChangeSet` | Applies a change set to a stack |
| `DeleteChangeSet` | Discards a change set |

## Examples

### AWS CLI

```bash
# Validate a template
aws cloudformation validate-template \
  --template-body file://template.yml

# Create a stack
aws cloudformation create-stack \
  --stack-name my-stack \
  --template-body file://template.yml \
  --parameters ParameterKey=Env,ParameterValue=dev

# Describe the stack
aws cloudformation describe-stacks --stack-name my-stack

# Describe resources
aws cloudformation describe-stack-resources --stack-name my-stack

# Create a change set
aws cloudformation create-change-set \
  --stack-name my-stack \
  --change-set-name update-1 \
  --template-body file://updated-template.yml

# Execute change set
aws cloudformation execute-change-set \
  --change-set-name update-1 \
  --stack-name my-stack

# Delete stack
aws cloudformation delete-stack --stack-name my-stack
```

### Python (boto3)

```python
import boto3

cf = boto3.client("cloudformation", endpoint_url="http://localhost:4566",
                  aws_access_key_id="test", aws_secret_access_key="test",
                  region_name="us-east-1")

template = """
AWSTemplateFormatVersion: '2010-09-09'
Parameters:
  BucketName:
    Type: String
Resources:
  MyBucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: !Ref BucketName
Outputs:
  BucketArn:
    Value: !GetAtt MyBucket.Arn
"""

# Create stack
cf.create_stack(
    StackName="infra",
    TemplateBody=template,
    Parameters=[{"ParameterKey": "BucketName", "ParameterValue": "my-cf-bucket"}],
)

# Check status
response = cf.describe_stacks(StackName="infra")
stack = response["Stacks"][0]
print(stack["StackStatus"])  # CREATE_COMPLETE
```

## Notes

- Stack resources listed in the template are stored as metadata. CloudFormation does not create actual resources in cloudmock (e.g., an `AWS::S3::Bucket` in the template does not create a real S3 bucket in the emulator).
- Stacks immediately transition to `CREATE_COMPLETE` after `CreateStack`.
- Change sets transition to `CREATE_COMPLETE` immediately and can be executed without waits.
- Nested stacks, stack sets, and drift detection are not implemented.
