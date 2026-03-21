# KMS — Key Management Service

**Tier:** 1 (Full Emulation)
**Protocol:** JSON (`X-Amz-Target: TrentService.<Action>`)
**Service Name:** `kms`

## Supported Actions

| Action | Notes |
|--------|-------|
| `CreateKey` | Creates a symmetric CMK and returns its key ID and ARN |
| `DescribeKey` | Returns key metadata |
| `ListKeys` | Returns all key IDs and ARNs |
| `Encrypt` | Encrypts plaintext using a CMK; returns base64 ciphertext |
| `Decrypt` | Decrypts ciphertext; returns plaintext |
| `CreateAlias` | Associates a friendly name with a key |
| `ListAliases` | Returns all aliases |
| `EnableKey` | Sets key state to Enabled |
| `DisableKey` | Sets key state to Disabled |
| `ScheduleKeyDeletion` | Marks a key for deletion after a waiting period |

## Examples

### AWS CLI

```bash
# Create a key
aws kms create-key --description "My test key"

# Encrypt
aws kms encrypt \
  --key-id arn:aws:kms:us-east-1:000000000000:key/<key-id> \
  --plaintext "Hello, KMS" \
  --query CiphertextBlob \
  --output text

# Decrypt
aws kms decrypt \
  --ciphertext-blob fileb://ciphertext.bin \
  --query Plaintext \
  --output text | base64 --decode

# Create an alias
aws kms create-alias \
  --alias-name alias/my-key \
  --target-key-id <key-id>

# List aliases
aws kms list-aliases
```

### Python (boto3)

```python
import boto3, base64

kms = boto3.client("kms", endpoint_url="http://localhost:4566",
                   aws_access_key_id="test", aws_secret_access_key="test",
                   region_name="us-east-1")

# Create key
key = kms.create_key(Description="test key")
key_id = key["KeyMetadata"]["KeyId"]

# Encrypt
enc = kms.encrypt(KeyId=key_id, Plaintext=b"secret value")
ciphertext = enc["CiphertextBlob"]

# Decrypt
dec = kms.decrypt(CiphertextBlob=ciphertext)
print(dec["Plaintext"])  # b"secret value"
```

## Notes

- Encryption is simulated: ciphertext is an encoded form of the plaintext that can be round-tripped through `Decrypt`. It is not cryptographically secure.
- Key rotation and asymmetric key operations are not implemented.
- Envelope encryption patterns (GenerateDataKey) are not implemented.
