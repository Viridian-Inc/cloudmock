# Cognito User Pools

**Tier:** 1 (Full Emulation)
**Protocol:** JSON (`X-Amz-Target: AWSCognitoIdentityProviderService.<Action>`)
**Service Name:** `cognito-idp`

## Supported Actions

| Action | Notes |
|--------|-------|
| `CreateUserPool` | Creates a user pool |
| `DeleteUserPool` | Deletes a user pool and all users |
| `DescribeUserPool` | Returns user pool details |
| `ListUserPools` | Returns all user pools |
| `CreateUserPoolClient` | Creates an app client for a user pool |
| `DescribeUserPoolClient` | Returns app client details |
| `ListUserPoolClients` | Returns all clients for a user pool |
| `AdminCreateUser` | Creates a user in a pool (admin API) |
| `AdminGetUser` | Returns user details (admin API) |
| `AdminDeleteUser` | Deletes a user (admin API) |
| `AdminSetUserPassword` | Sets a user's password (admin API) |
| `SignUp` | User self-registration |
| `InitiateAuth` | Starts the authentication flow |
| `AdminConfirmSignUp` | Confirms a user's registration (admin API) |

## Examples

### AWS CLI

```bash
# Create user pool
aws cognito-idp create-user-pool --pool-name MyApp

# Create app client
aws cognito-idp create-user-pool-client \
  --user-pool-id us-east-1_XXXXXXXX \
  --client-name web-client \
  --no-generate-secret

# Create a user
aws cognito-idp admin-create-user \
  --user-pool-id us-east-1_XXXXXXXX \
  --username alice \
  --temporary-password "Temp123!"

# Set permanent password
aws cognito-idp admin-set-user-password \
  --user-pool-id us-east-1_XXXXXXXX \
  --username alice \
  --password "Perm456!" \
  --permanent

# Initiate auth
aws cognito-idp initiate-auth \
  --auth-flow USER_PASSWORD_AUTH \
  --client-id <client-id> \
  --auth-parameters USERNAME=alice,PASSWORD="Perm456!"
```

### Python (boto3)

```python
import boto3

idp = boto3.client("cognito-idp", endpoint_url="http://localhost:4566",
                   aws_access_key_id="test", aws_secret_access_key="test",
                   region_name="us-east-1")

# Create pool and client
pool = idp.create_user_pool(PoolName="MyApp")
pool_id = pool["UserPool"]["Id"]

client = idp.create_user_pool_client(
    UserPoolId=pool_id,
    ClientName="backend",
    ExplicitAuthFlows=["ALLOW_USER_PASSWORD_AUTH", "ALLOW_REFRESH_TOKEN_AUTH"],
)
client_id = client["UserPoolClient"]["ClientId"]

# Create and confirm user
idp.admin_create_user(UserPoolId=pool_id, Username="bob", TemporaryPassword="Tmp1!")
idp.admin_set_user_password(UserPoolId=pool_id, Username="bob", Password="Perm1!", Permanent=True)

# Authenticate
response = idp.initiate_auth(
    AuthFlow="USER_PASSWORD_AUTH",
    ClientId=client_id,
    AuthParameters={"USERNAME": "bob", "PASSWORD": "Perm1!"},
)
print(response["AuthenticationResult"]["AccessToken"])
```

## Notes

- Tokens returned by `InitiateAuth` are synthetic JWTs. They are accepted by the cloudmock IAM middleware but cannot be verified against a real Cognito JWKS endpoint.
- MFA, custom authentication flows, and identity pools (Cognito Federated Identities) are not implemented.
- Password policies are accepted in `CreateUserPool` but not enforced.
