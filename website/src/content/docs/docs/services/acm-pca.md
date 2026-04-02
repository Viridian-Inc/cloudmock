---
title: ACM PCA
description: AWS Certificate Manager Private Certificate Authority emulation in CloudMock
---

## Overview

CloudMock emulates AWS Certificate Manager Private Certificate Authority (ACM PCA), supporting private CA lifecycle management, certificate issuance, revocation, and permissions.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateCertificateAuthority | Supported | Creates a private CA |
| DescribeCertificateAuthority | Supported | Returns CA details |
| ListCertificateAuthorities | Supported | Lists all private CAs |
| DeleteCertificateAuthority | Supported | Deletes a private CA |
| UpdateCertificateAuthority | Supported | Updates CA configuration |
| IssueCertificate | Supported | Issues a certificate from the CA |
| GetCertificate | Supported | Returns an issued certificate |
| GetCertificateAuthorityCertificate | Supported | Returns the CA certificate and chain |
| RevokeCertificate | Supported | Revokes a certificate |
| GetCertificateAuthorityCsr | Supported | Returns the CSR for a CA (requires PENDING_CERTIFICATE state) |
| TagCertificateAuthority | Supported | Adds tags to a CA |
| UntagCertificateAuthority | Supported | Removes tags from a CA |
| ListTags | Supported | Lists tags for a CA |
| CreatePermission | Supported | Grants permissions on a CA |
| ListPermissions | Supported | Lists permissions for a CA |
| DeletePermission | Supported | Removes permissions from a CA |

## Quick Start

### Node.js

```typescript
import { ACMPCAClient, CreateCertificateAuthorityCommand } from '@aws-sdk/client-acm-pca';

const client = new ACMPCAClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const result = await client.send(new CreateCertificateAuthorityCommand({
  CertificateAuthorityConfiguration: {
    KeyAlgorithm: 'RSA_2048',
    SigningAlgorithm: 'SHA256WITHRSA',
    Subject: { CommonName: 'My Private CA' },
  },
  CertificateAuthorityType: 'ROOT',
}));
console.log(result.CertificateAuthorityArn);
```

### Python

```python
import boto3

client = boto3.client('acm-pca',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

response = client.create_certificate_authority(
    CertificateAuthorityConfiguration={
        'KeyAlgorithm': 'RSA_2048',
        'SigningAlgorithm': 'SHA256WITHRSA',
        'Subject': {'CommonName': 'My Private CA'},
    },
    CertificateAuthorityType='ROOT')
print(response['CertificateAuthorityArn'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  acmpca:
    enabled: true
```

## Known Differences from AWS

- PKI operations generate stub certificates rather than cryptographically valid ones
- CSR generation returns placeholder data
- Certificate revocation updates status but does not maintain a real CRL
