---
title: Cognito
description: Amazon Cognito User Pools emulation in CloudMock
---

## Overview

CloudMock emulates Amazon Cognito User Pools, supporting user pool and client management, admin user operations, self-service sign-up, and authentication with synthetic JWT tokens.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateUserPool | Supported | Creates a user pool |
| DeleteUserPool | Supported | Deletes a user pool and all users |
| DescribeUserPool | Supported | Returns user pool details |
| ListUserPools | Supported | Returns all user pools |
| CreateUserPoolClient | Supported | Creates an app client for a user pool |
| DescribeUserPoolClient | Supported | Returns app client details |
| ListUserPoolClients | Supported | Returns all clients for a user pool |
| AdminCreateUser | Supported | Creates a user in a pool (admin API) |
| AdminGetUser | Supported | Returns user details (admin API) |
| AdminDeleteUser | Supported | Deletes a user (admin API) |
| AdminSetUserPassword | Supported | Sets a user's password (admin API) |
| SignUp | Supported | User self-registration |
| InitiateAuth | Supported | Starts the authentication flow |
| AdminConfirmSignUp | Supported | Confirms a user's registration (admin API) |

## Quick Start

### curl

```bash
# Create a user pool
curl -X POST http://localhost:4566 \
  -H "X-Amz-Target: AWSCognitoIdentityProviderService.CreateUserPool" \
  -H "Content-Type: application/x-amz-json-1.1" \
  -d '{"PoolName": "MyApp"}'

# Create an app client
curl -X POST http://localhost:4566 \
  -H "X-Amz-Target: AWSCognitoIdentityProviderService.CreateUserPoolClient" \
  -H "Content-Type: application/x-amz-json-1.1" \
  -d '{"UserPoolId": "us-east-1_XXXXXXXX", "ClientName": "web-client"}'
```

### Node.js

```typescript
import { CognitoIdentityProviderClient, CreateUserPoolCommand, AdminCreateUserCommand, InitiateAuthCommand } from '@aws-sdk/client-cognito-identity-provider';

const cognito = new CognitoIdentityProviderClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const pool = await cognito.send(new CreateUserPoolCommand({ PoolName: 'MyApp' }));
const poolId = pool.UserPool!.Id!;

await cognito.send(new AdminCreateUserCommand({
  UserPoolId: poolId, Username: 'alice', TemporaryPassword: 'Temp123!',
}));
```

### Python

```python
import boto3

idp = boto3.client('cognito-idp', endpoint_url='http://localhost:4566',
                   aws_access_key_id='test', aws_secret_access_key='test',
                   region_name='us-east-1')

pool = idp.create_user_pool(PoolName='MyApp')
pool_id = pool['UserPool']['Id']

client = idp.create_user_pool_client(
    UserPoolId=pool_id, ClientName='backend',
    ExplicitAuthFlows=['ALLOW_USER_PASSWORD_AUTH', 'ALLOW_REFRESH_TOKEN_AUTH'],
)
client_id = client['UserPoolClient']['ClientId']

idp.admin_create_user(UserPoolId=pool_id, Username='bob', TemporaryPassword='Tmp1!')
idp.admin_set_user_password(UserPoolId=pool_id, Username='bob', Password='Perm1!', Permanent=True)

response = idp.initiate_auth(
    AuthFlow='USER_PASSWORD_AUTH', ClientId=client_id,
    AuthParameters={'USERNAME': 'bob', 'PASSWORD': 'Perm1!'},
)
print(response['AuthenticationResult']['AccessToken'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  cognito:
    enabled: true
```

No additional service-specific configuration is required.

## Known Differences from AWS

- Tokens returned by `InitiateAuth` are **synthetic JWTs**. They are accepted by the CloudMock IAM middleware but cannot be verified against a real Cognito JWKS endpoint.
- **MFA**, custom authentication flows, and **identity pools** (Cognito Federated Identities) are not implemented.
- **Password policies** are accepted in `CreateUserPool` but not enforced.
- **Email/SMS verification** is not performed; users can be confirmed via `AdminConfirmSignUp`.
- **Hosted UI** and OAuth flows are not implemented.

## Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| ResourceNotFoundException | 400 | The specified user pool does not exist |
| UsernameExistsException | 400 | A user with this username already exists |
| NotAuthorizedException | 400 | Invalid username or password |
| UserNotFoundException | 400 | The specified user does not exist |
| InvalidParameterException | 400 | An input parameter is not valid |
