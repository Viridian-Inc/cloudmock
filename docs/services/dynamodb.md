# DynamoDB

**Tier:** 1 (Full Emulation)
**Protocol:** JSON (`X-Amz-Target: DynamoDB_20120810.<Action>`)
**Service Name:** `dynamodb`

## Supported Actions

| Action | Notes |
|--------|-------|
| `CreateTable` | Creates a table with key schema and billing mode |
| `DeleteTable` | Deletes a table and all its items |
| `DescribeTable` | Returns table metadata and status |
| `ListTables` | Returns a list of all table names |
| `PutItem` | Inserts or replaces an item |
| `GetItem` | Retrieves a single item by primary key |
| `DeleteItem` | Removes a single item by primary key |
| `UpdateItem` | Partial update using UpdateExpression |
| `Query` | Queries items by partition key with optional sort key condition |
| `Scan` | Scans all items in a table |
| `BatchGetItem` | Retrieves up to 100 items across multiple tables |
| `BatchWriteItem` | Puts or deletes up to 25 items across multiple tables |

## Examples

### AWS CLI

```bash
# Create a table
aws dynamodb create-table \
  --table-name Users \
  --attribute-definitions AttributeName=UserId,AttributeType=S \
  --key-schema AttributeName=UserId,KeyType=HASH \
  --billing-mode PAY_PER_REQUEST

# Put an item
aws dynamodb put-item \
  --table-name Users \
  --item '{"UserId": {"S": "u1"}, "Name": {"S": "Alice"}}'

# Get an item
aws dynamodb get-item \
  --table-name Users \
  --key '{"UserId": {"S": "u1"}}'

# Query
aws dynamodb query \
  --table-name Users \
  --key-condition-expression "UserId = :uid" \
  --expression-attribute-values '{":uid": {"S": "u1"}}'

# Scan
aws dynamodb scan --table-name Users
```

### Python (boto3)

```python
import boto3

dynamodb = boto3.resource("dynamodb", endpoint_url="http://localhost:4566",
                          aws_access_key_id="test", aws_secret_access_key="test",
                          region_name="us-east-1")

# Create table
table = dynamodb.create_table(
    TableName="Orders",
    KeySchema=[
        {"AttributeName": "OrderId", "KeyType": "HASH"},
        {"AttributeName": "CreatedAt", "KeyType": "RANGE"},
    ],
    AttributeDefinitions=[
        {"AttributeName": "OrderId", "AttributeType": "S"},
        {"AttributeName": "CreatedAt", "AttributeType": "S"},
    ],
    BillingMode="PAY_PER_REQUEST",
)

# Put item
table.put_item(Item={"OrderId": "o1", "CreatedAt": "2026-01-01", "Total": 99})

# Get item
response = table.get_item(Key={"OrderId": "o1", "CreatedAt": "2026-01-01"})
print(response["Item"])
```

## Notes

- Secondary indexes (GSI/LSI) are accepted during `CreateTable` but queries on them fall back to a full scan.
- `UpdateItem` supports `SET` and `REMOVE` update expressions.
- `BatchWriteItem` processes all requests in a single call; there is no retry/unprocessed-items logic.
