# ECS — Elastic Container Service

**Tier:** 1 (Full Emulation)
**Protocol:** JSON (`X-Amz-Target: AmazonEC2ContainerServiceV20141113.<Action>`)
**Service Name:** `ecs`

## Supported Actions

| Action | Notes |
|--------|-------|
| `CreateCluster` | Creates an ECS cluster |
| `DeleteCluster` | Deletes a cluster |
| `DescribeClusters` | Returns cluster details |
| `ListClusters` | Returns all cluster ARNs |
| `RegisterTaskDefinition` | Registers a task definition revision |
| `DeregisterTaskDefinition` | Marks a task definition as INACTIVE |
| `DescribeTaskDefinition` | Returns a task definition |
| `ListTaskDefinitions` | Returns all task definition ARNs |
| `CreateService` | Creates a long-running service |
| `DeleteService` | Deletes a service |
| `DescribeServices` | Returns service details |
| `ListServices` | Returns all service ARNs in a cluster |
| `UpdateService` | Updates desired count, task definition, etc. |
| `RunTask` | Starts one or more task instances |
| `StopTask` | Stops a running task |
| `DescribeTasks` | Returns task details |
| `ListTasks` | Returns task ARNs in a cluster or service |
| `TagResource` | Adds tags |
| `UntagResource` | Removes tags |
| `ListTagsForResource` | Returns tags for a resource |

## Examples

### AWS CLI

```bash
# Create a cluster
aws ecs create-cluster --cluster-name production

# Register a task definition
aws ecs register-task-definition \
  --family web-server \
  --container-definitions '[{
    "name": "nginx",
    "image": "nginx:latest",
    "portMappings": [{"containerPort": 80}]
  }]' \
  --requires-compatibilities FARGATE \
  --network-mode awsvpc \
  --cpu "256" \
  --memory "512"

# Create a service
aws ecs create-service \
  --cluster production \
  --service-name web \
  --task-definition web-server:1 \
  --desired-count 2 \
  --launch-type FARGATE

# Run a task
aws ecs run-task \
  --cluster production \
  --task-definition web-server:1 \
  --launch-type FARGATE

# List tasks
aws ecs list-tasks --cluster production
```

### Python (boto3)

```python
import boto3

ecs = boto3.client("ecs", endpoint_url="http://localhost:4566",
                   aws_access_key_id="test", aws_secret_access_key="test",
                   region_name="us-east-1")

# Create cluster
ecs.create_cluster(clusterName="dev")

# Register task definition
ecs.register_task_definition(
    family="worker",
    containerDefinitions=[{"name": "app", "image": "myapp:latest", "cpu": 256, "memory": 512}],
    requiresCompatibilities=["FARGATE"],
    networkMode="awsvpc",
    cpu="256",
    memory="512",
)

# Create service
ecs.create_service(
    cluster="dev",
    serviceName="workers",
    taskDefinition="worker:1",
    desiredCount=1,
    launchType="FARGATE",
)

# Describe service
response = ecs.describe_services(cluster="dev", services=["workers"])
svc = response["services"][0]
print(svc["status"], svc["runningCount"])
```

## Notes

- Tasks and services are metadata records. No container runtime is involved.
- `RunTask` returns a task record with status `RUNNING` immediately. No container is started.
- Service auto-scaling and capacity providers are not implemented.
- Container Insights and ECS Anywhere are not implemented.
