---
title: AppSync
description: AWS AppSync (GraphQL) emulation in CloudMock
---

## Overview

CloudMock emulates AWS AppSync, supporting GraphQL API management, data source configuration, resolver and function management, API key lifecycle, and tagging.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateGraphqlApi | Supported | Creates a GraphQL API |
| GetGraphqlApi | Supported | Returns API details |
| ListGraphqlApis | Supported | Returns all GraphQL APIs |
| UpdateGraphqlApi | Supported | Updates API configuration |
| DeleteGraphqlApi | Supported | Deletes a GraphQL API |
| CreateDataSource | Supported | Creates a data source for an API |
| GetDataSource | Supported | Returns data source details |
| ListDataSources | Supported | Returns all data sources for an API |
| UpdateDataSource | Supported | Updates a data source |
| DeleteDataSource | Supported | Deletes a data source |
| CreateResolver | Supported | Creates a resolver for a type/field |
| GetResolver | Supported | Returns resolver details |
| ListResolvers | Supported | Returns all resolvers for a type |
| UpdateResolver | Supported | Updates a resolver |
| DeleteResolver | Supported | Deletes a resolver |
| CreateFunction | Supported | Creates a pipeline function |
| GetFunction | Supported | Returns function details |
| ListFunctions | Supported | Returns all functions for an API |
| UpdateFunction | Supported | Updates a function |
| DeleteFunction | Supported | Deletes a function |
| CreateApiKey | Supported | Creates an API key |
| ListApiKeys | Supported | Returns all API keys |
| UpdateApiKey | Supported | Updates an API key expiration |
| DeleteApiKey | Supported | Deletes an API key |
| TagResource | Supported | Adds tags to a resource |
| UntagResource | Supported | Removes tags |
| ListTagsForResource | Supported | Returns tags for a resource |

## Quick Start

### curl

```bash
# Create a GraphQL API
curl -X POST http://localhost:4566/v1/apis \
  -H "Content-Type: application/json" \
  -d '{
    "name": "MyAPI",
    "authenticationType": "API_KEY"
  }'

# Create an API key
curl -X POST http://localhost:4566/v1/apis/<api-id>/apikeys \
  -H "Content-Type: application/json" \
  -d '{"description": "dev key"}'
```

### Node.js

```typescript
import { AppSyncClient, CreateGraphqlApiCommand, CreateDataSourceCommand, CreateResolverCommand } from '@aws-sdk/client-appsync';

const appsync = new AppSyncClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const api = await appsync.send(new CreateGraphqlApiCommand({
  name: 'MyAPI', authenticationType: 'API_KEY',
}));
const apiId = api.graphqlApi!.apiId!;

await appsync.send(new CreateDataSourceCommand({
  apiId, name: 'usersTable', type: 'AMAZON_DYNAMODB',
  dynamodbConfig: { tableName: 'Users', awsRegion: 'us-east-1' },
  serviceRoleArn: 'arn:aws:iam::000000000000:role/appsync-role',
}));

await appsync.send(new CreateResolverCommand({
  apiId, typeName: 'Query', fieldName: 'getUser',
  dataSourceName: 'usersTable',
  requestMappingTemplate: '{"version":"2017-02-28","operation":"GetItem","key":{"id":{"S":"$ctx.args.id"}}}',
  responseMappingTemplate: '$util.toJson($ctx.result)',
}));
```

### Python

```python
import boto3

appsync = boto3.client('appsync', endpoint_url='http://localhost:4566',
                       aws_access_key_id='test', aws_secret_access_key='test',
                       region_name='us-east-1')

api = appsync.create_graphql_api(name='MyAPI', authenticationType='API_KEY')
api_id = api['graphqlApi']['apiId']

appsync.create_data_source(
    apiId=api_id, name='usersTable', type='AMAZON_DYNAMODB',
    dynamodbConfig={'tableName': 'Users', 'awsRegion': 'us-east-1'},
    serviceRoleArn='arn:aws:iam::000000000000:role/appsync-role',
)

appsync.create_resolver(
    apiId=api_id, typeName='Query', fieldName='getUser',
    dataSourceName='usersTable',
    requestMappingTemplate='{"version":"2017-02-28","operation":"GetItem","key":{"id":{"S":"$ctx.args.id"}}}',
    responseMappingTemplate='$util.toJson($ctx.result)',
)

keys = appsync.create_api_key(apiId=api_id, description='dev key')
print(keys['apiKey']['id'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  appsync:
    enabled: true
```

No additional service-specific configuration is required.

## Known Differences from AWS

- Only the **management API** is emulated. GraphQL query execution against the API endpoint is not supported.
- **Resolvers and functions** are stored as metadata but mapping templates are not executed.
- **Subscriptions** (real-time WebSocket) are not implemented.
- **Caching**, **logging**, and **WAF integration** are not implemented.
- **Merged APIs** and **custom domains** are not implemented.
- Data source connections to DynamoDB, Lambda, etc. are recorded but not used for query resolution.

## Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| NotFoundException | 404 | The specified API, data source, or resolver does not exist |
| ConcurrentModificationException | 409 | The resource was modified by another request |
| BadRequestException | 400 | The request is not valid |
| ApiKeyLimitExceededException | 400 | Too many API keys for this API |
| ApiLimitExceededException | 400 | Too many APIs in this account |
