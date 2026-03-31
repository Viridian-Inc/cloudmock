---
title: DynamoDB
description: Amazon DynamoDB emulation in CloudMock
---

## Overview

CloudMock emulates Amazon DynamoDB, a fully managed NoSQL key-value and document database, supporting table management, single-item CRUD, queries, scans, and batch operations.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateTable | Supported | Creates a table with key schema and billing mode |
| DeleteTable | Supported | Deletes a table and all its items |
| DescribeTable | Supported | Returns table metadata and status |
| ListTables | Supported | Returns a list of all table names |
| PutItem | Supported | Inserts or replaces an item |
| GetItem | Supported | Retrieves a single item by primary key |
| DeleteItem | Supported | Removes a single item by primary key |
| UpdateItem | Supported | Partial update using UpdateExpression |
| Query | Supported | Queries items by partition key with optional sort key condition |
| Scan | Supported | Scans all items in a table |
| BatchGetItem | Supported | Retrieves up to 100 items across multiple tables |
| BatchWriteItem | Supported | Puts or deletes up to 25 items across multiple tables |

## Quick Start

### curl

```bash
# Create a table
curl -X POST http://localhost:4566 \
  -H "X-Amz-Target: DynamoDB_20120810.CreateTable" \
  -H "Content-Type: application/x-amz-json-1.0" \
  -d '{
    "TableName": "Users",
    "KeySchema": [{"AttributeName": "UserId", "KeyType": "HASH"}],
    "AttributeDefinitions": [{"AttributeName": "UserId", "AttributeType": "S"}],
    "BillingMode": "PAY_PER_REQUEST"
  }'

# Put an item
curl -X POST http://localhost:4566 \
  -H "X-Amz-Target: DynamoDB_20120810.PutItem" \
  -H "Content-Type: application/x-amz-json-1.0" \
  -d '{
    "TableName": "Users",
    "Item": {"UserId": {"S": "u1"}, "Name": {"S": "Alice"}}
  }'
```

### Node.js

```typescript
import { DynamoDBClient, CreateTableCommand, PutItemCommand } from '@aws-sdk/client-dynamodb';

const ddb = new DynamoDBClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await ddb.send(new CreateTableCommand({
  TableName: 'Users',
  KeySchema: [{ AttributeName: 'UserId', KeyType: 'HASH' }],
  AttributeDefinitions: [{ AttributeName: 'UserId', AttributeType: 'S' }],
  BillingMode: 'PAY_PER_REQUEST',
}));

await ddb.send(new PutItemCommand({
  TableName: 'Users',
  Item: { UserId: { S: 'u1' }, Name: { S: 'Alice' } },
}));
```

### Python

```python
import boto3

dynamodb = boto3.resource('dynamodb', endpoint_url='http://localhost:4566',
                          aws_access_key_id='test', aws_secret_access_key='test',
                          region_name='us-east-1')

table = dynamodb.create_table(
    TableName='Users',
    KeySchema=[{'AttributeName': 'UserId', 'KeyType': 'HASH'}],
    AttributeDefinitions=[{'AttributeName': 'UserId', 'AttributeType': 'S'}],
    BillingMode='PAY_PER_REQUEST',
)

table.put_item(Item={'UserId': 'u1', 'Name': 'Alice'})
response = table.get_item(Key={'UserId': 'u1'})
print(response['Item'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  dynamodb:
    enabled: true
```

No additional service-specific configuration is required.

## Known Differences from AWS

- **Secondary indexes** (GSI/LSI) are accepted during `CreateTable` but queries on them fall back to a full scan.
- **UpdateItem** supports `SET` and `REMOVE` update expressions only. `ADD` and `DELETE` are not implemented.
- **BatchWriteItem** processes all requests in a single call with no retry/unprocessed-items logic.
- **Streams** (DynamoDB Streams) are not implemented.
- **Transactions** (`TransactWriteItems`, `TransactGetItems`) are not implemented.
- **Time-to-Live (TTL)** is not enforced.

## Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| ResourceNotFoundException | 400 | The specified table does not exist |
| ResourceInUseException | 400 | The table already exists |
| ValidationException | 400 | Invalid input (missing required key attributes, etc.) |
| ConditionalCheckFailedException | 400 | A condition expression evaluated to false |
