# Step Functions

**Tier:** 1 (Full Emulation)
**Protocol:** JSON (`X-Amz-Target: AmazonStatesService.<Action>`)
**Service Name:** `states`

## Supported Actions

| Action | Notes |
|--------|-------|
| `CreateStateMachine` | Creates a state machine from an ASL definition |
| `DeleteStateMachine` | Deletes a state machine |
| `DescribeStateMachine` | Returns state machine definition and metadata |
| `ListStateMachines` | Returns all state machines |
| `UpdateStateMachine` | Updates the definition or role ARN |
| `StartExecution` | Starts an execution and returns its ARN |
| `DescribeExecution` | Returns execution status and input/output |
| `StopExecution` | Stops a running execution |
| `ListExecutions` | Returns executions for a state machine |
| `GetExecutionHistory` | Returns the event history of an execution |
| `TagResource` | Adds tags to a state machine |
| `UntagResource` | Removes tags |
| `ListTagsForResource` | Returns tags for a resource |

## Examples

### AWS CLI

```bash
# Create a state machine
aws stepfunctions create-state-machine \
  --name HelloWorld \
  --definition '{"Comment":"Test","StartAt":"Hello","States":{"Hello":{"Type":"Pass","End":true}}}' \
  --role-arn arn:aws:iam::000000000000:role/step-functions-role

# Start an execution
aws stepfunctions start-execution \
  --state-machine-arn arn:aws:states:us-east-1:000000000000:stateMachine:HelloWorld \
  --name run-1 \
  --input '{"key":"value"}'

# Check execution status
aws stepfunctions describe-execution \
  --execution-arn arn:aws:states:us-east-1:000000000000:execution:HelloWorld:run-1

# List executions
aws stepfunctions list-executions \
  --state-machine-arn arn:aws:states:us-east-1:000000000000:stateMachine:HelloWorld
```

### Python (boto3)

```python
import boto3, json

sfn = boto3.client("stepfunctions", endpoint_url="http://localhost:4566",
                   aws_access_key_id="test", aws_secret_access_key="test",
                   region_name="us-east-1")

definition = {
    "Comment": "Simple pass-through",
    "StartAt": "PassState",
    "States": {
        "PassState": {"Type": "Pass", "End": True}
    },
}

# Create
sm = sfn.create_state_machine(
    name="MyFlow",
    definition=json.dumps(definition),
    roleArn="arn:aws:iam::000000000000:role/sfn-role",
)
sm_arn = sm["stateMachineArn"]

# Execute
execution = sfn.start_execution(
    stateMachineArn=sm_arn,
    name="exec-1",
    input=json.dumps({"orderId": "o-123"}),
)

# Poll status
result = sfn.describe_execution(executionArn=execution["executionArn"])
print(result["status"])  # RUNNING | SUCCEEDED | FAILED
```

## Notes

- Executions are recorded with their input and status but the state machine definition is not actually interpreted. Executions immediately transition to `SUCCEEDED`.
- `GetExecutionHistory` returns a minimal event list reflecting the start and end of the execution.
- Express workflows are accepted but behave identically to standard workflows.
- Activity tasks and heartbeats are not implemented.
