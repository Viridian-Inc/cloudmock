---
title: KMS
description: AWS KMS (Key Management Service) emulation in CloudMock
---

## Overview

CloudMock emulates AWS KMS, supporting symmetric key management, simulated encrypt/decrypt operations, key aliases, and key lifecycle (enable, disable, schedule deletion).

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateKey | Supported | Creates a symmetric CMK and returns its key ID and ARN |
| DescribeKey | Supported | Returns key metadata |
| ListKeys | Supported | Returns all key IDs and ARNs |
| Encrypt | Supported | Encrypts plaintext using a CMK; returns base64 ciphertext |
| Decrypt | Supported | Decrypts ciphertext; returns plaintext |
| CreateAlias | Supported | Associates a friendly name with a key |
| ListAliases | Supported | Returns all aliases |
| EnableKey | Supported | Sets key state to Enabled |
| DisableKey | Supported | Sets key state to Disabled |
| ScheduleKeyDeletion | Supported | Marks a key for deletion after a waiting period |

## Quick Start

### curl

```bash
# Create a key
curl -X POST http://localhost:4566 \
  -H "X-Amz-Target: TrentService.CreateKey" \
  -H "Content-Type: application/x-amz-json-1.1" \
  -d '{"Description": "My test key"}'

# Encrypt
curl -X POST http://localhost:4566 \
  -H "X-Amz-Target: TrentService.Encrypt" \
  -H "Content-Type: application/x-amz-json-1.1" \
  -d '{"KeyId": "<key-id>", "Plaintext": "SGVsbG8sIEtNUw=="}'
```

### Node.js

```typescript
import { KMSClient, CreateKeyCommand, EncryptCommand, DecryptCommand } from '@aws-sdk/client-kms';

const kms = new KMSClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const key = await kms.send(new CreateKeyCommand({ Description: 'test key' }));
const keyId = key.KeyMetadata!.KeyId!;

const enc = await kms.send(new EncryptCommand({
  KeyId: keyId, Plaintext: Buffer.from('secret value'),
}));

const dec = await kms.send(new DecryptCommand({
  CiphertextBlob: enc.CiphertextBlob!,
}));
console.log(Buffer.from(dec.Plaintext!).toString()); // secret value
```

### Python

```python
import boto3

kms = boto3.client('kms', endpoint_url='http://localhost:4566',
                   aws_access_key_id='test', aws_secret_access_key='test',
                   region_name='us-east-1')

key = kms.create_key(Description='test key')
key_id = key['KeyMetadata']['KeyId']

enc = kms.encrypt(KeyId=key_id, Plaintext=b'secret value')
ciphertext = enc['CiphertextBlob']

dec = kms.decrypt(CiphertextBlob=ciphertext)
print(dec['Plaintext'])  # b'secret value'
```

## Configuration

```yaml
# cloudmock.yml
services:
  kms:
    enabled: true
```

No additional service-specific configuration is required.

## Known Differences from AWS

- Encryption is **simulated**: ciphertext is an encoded form of the plaintext that can be round-tripped through `Decrypt`. It is not cryptographically secure.
- **Key rotation** and **asymmetric key** operations are not implemented.
- **Envelope encryption** patterns (`GenerateDataKey`, `GenerateDataKeyWithoutPlaintext`) are not implemented.
- **Grants** and **key policies** are not implemented.
- **Multi-region keys** are not implemented.

## Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| NotFoundException | 400 | The specified key does not exist |
| DisabledException | 400 | The key is disabled |
| InvalidCiphertextException | 400 | The ciphertext is not valid |
| KMSInvalidStateException | 400 | The key is in an invalid state for this operation |
| AlreadyExistsException | 400 | An alias with this name already exists |
