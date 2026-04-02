---
title: ACM
description: AWS Certificate Manager emulation in CloudMock
---

## Overview

CloudMock emulates AWS Certificate Manager (ACM), supporting certificate request, import, renewal, export, and tagging operations.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| RequestCertificate | Supported | Creates a certificate request |
| DescribeCertificate | Supported | Returns certificate details |
| ListCertificates | Supported | Lists all certificates |
| DeleteCertificate | Supported | Deletes a certificate |
| ImportCertificate | Supported | Imports an external certificate |
| RenewCertificate | Supported | Triggers certificate renewal |
| ExportCertificate | Supported | Exports a certificate and private key |
| GetCertificate | Supported | Returns the certificate body and chain |
| AddTagsToCertificate | Supported | Adds tags to a certificate |
| RemoveTagsFromCertificate | Supported | Removes tags from a certificate |
| ListTagsForCertificate | Supported | Lists tags for a certificate |

## Quick Start

### Node.js

```typescript
import { ACMClient, RequestCertificateCommand, DescribeCertificateCommand } from '@aws-sdk/client-acm';

const client = new ACMClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const { CertificateArn } = await client.send(new RequestCertificateCommand({
  DomainName: 'example.com',
}));

const cert = await client.send(new DescribeCertificateCommand({
  CertificateArn,
}));
console.log(cert.Certificate);
```

### Python

```python
import boto3

client = boto3.client('acm',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

response = client.request_certificate(DomainName='example.com')
arn = response['CertificateArn']

cert = client.describe_certificate(CertificateArn=arn)
print(cert['Certificate'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  acm:
    enabled: true
```

## Known Differences from AWS

- Certificates are not actually validated via DNS or email
- Renewal does not perform real re-issuance; it updates the certificate status
- Export returns stub certificate data
