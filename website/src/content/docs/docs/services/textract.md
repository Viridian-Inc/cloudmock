---
title: Textract
description: Amazon Textract emulation in CloudMock
---

## Overview

CloudMock emulates Amazon Textract, supporting synchronous and asynchronous document text detection, document analysis, expense analysis, and tagging.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| DetectDocumentText | Supported | Synchronous text detection |
| AnalyzeDocument | Supported | Synchronous document analysis |
| StartDocumentTextDetection | Supported | Starts async text detection |
| GetDocumentTextDetection | Supported | Returns async text detection results |
| StartDocumentAnalysis | Supported | Starts async document analysis |
| GetDocumentAnalysis | Supported | Returns async analysis results |
| StartExpenseAnalysis | Supported | Starts async expense analysis |
| GetExpenseAnalysis | Supported | Returns async expense results |
| TagResource | Supported | Adds tags to a resource |
| UntagResource | Supported | Removes tags from a resource |
| ListTagsForResource | Supported | Lists tags for a resource |

## Quick Start

### Node.js

```typescript
import { TextractClient, StartDocumentTextDetectionCommand, GetDocumentTextDetectionCommand } from '@aws-sdk/client-textract';

const client = new TextractClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const { JobId } = await client.send(new StartDocumentTextDetectionCommand({
  DocumentLocation: { S3Object: { Bucket: 'my-bucket', Name: 'document.pdf' } },
}));

const result = await client.send(new GetDocumentTextDetectionCommand({ JobId }));
console.log(result.JobStatus);
```

### Python

```python
import boto3

client = boto3.client('textract',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

response = client.start_document_text_detection(
    DocumentLocation={'S3Object': {'Bucket': 'my-bucket', 'Name': 'document.pdf'}})

result = client.get_document_text_detection(JobId=response['JobId'])
print(result['JobStatus'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  textract:
    enabled: true
```

## Known Differences from AWS

- No actual OCR or document analysis is performed
- Results return stub/empty block data
- Async jobs complete immediately
