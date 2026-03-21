# RDS — Relational Database Service

**Tier:** 1 (Full Emulation)
**Protocol:** Query (`Action=<Action>`)
**Service Name:** `rds`

## Supported Actions

| Action | Notes |
|--------|-------|
| `CreateDBInstance` | Creates a DB instance record |
| `DeleteDBInstance` | Deletes a DB instance |
| `DescribeDBInstances` | Returns all DB instances |
| `ModifyDBInstance` | Updates instance attributes |
| `CreateDBCluster` | Creates an Aurora cluster |
| `DeleteDBCluster` | Deletes a cluster |
| `DescribeDBClusters` | Returns all clusters |
| `CreateDBSnapshot` | Creates a snapshot of a DB instance |
| `DeleteDBSnapshot` | Deletes a snapshot |
| `DescribeDBSnapshots` | Returns all snapshots |
| `CreateDBSubnetGroup` | Creates a subnet group |
| `DescribeDBSubnetGroups` | Returns all subnet groups |
| `DeleteDBSubnetGroup` | Deletes a subnet group |
| `AddTagsToResource` | Adds tags to a resource |
| `RemoveTagsFromResource` | Removes tags |
| `ListTagsForResource` | Returns tags for a resource |

## Examples

### AWS CLI

```bash
# Create a DB instance
aws rds create-db-instance \
  --db-instance-identifier my-db \
  --db-instance-class db.t3.micro \
  --engine mysql \
  --master-username admin \
  --master-user-password "Password123" \
  --allocated-storage 20

# Describe instances
aws rds describe-db-instances

# Create a snapshot
aws rds create-db-snapshot \
  --db-instance-identifier my-db \
  --db-snapshot-identifier my-db-snap-1

# Delete the instance
aws rds delete-db-instance \
  --db-instance-identifier my-db \
  --skip-final-snapshot
```

### Python (boto3)

```python
import boto3

rds = boto3.client("rds", endpoint_url="http://localhost:4566",
                   aws_access_key_id="test", aws_secret_access_key="test",
                   region_name="us-east-1")

# Create instance
rds.create_db_instance(
    DBInstanceIdentifier="test-db",
    DBInstanceClass="db.t3.micro",
    Engine="postgres",
    MasterUsername="postgres",
    MasterUserPassword="postgres123",
    AllocatedStorage=20,
)

# Describe
response = rds.describe_db_instances()
for db in response["DBInstances"]:
    print(db["DBInstanceIdentifier"], db["DBInstanceStatus"])
```

## Notes

- DB instances are metadata-only records. No actual database engine is started.
- Connection strings and endpoints returned in `DescribeDBInstances` are synthetic and not connectable.
- Parameter groups, option groups, and enhanced monitoring are accepted but not enforced.
- Aurora serverless v2, Proxy, and Blue/Green deployments are not implemented.
