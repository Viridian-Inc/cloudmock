---
title: Transcribe
description: Amazon Transcribe emulation in CloudMock
---

## Overview

CloudMock emulates Amazon Transcribe, supporting transcription job management, custom vocabularies, and tagging.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| StartTranscriptionJob | Supported | Starts a transcription job |
| GetTranscriptionJob | Supported | Returns transcription job details |
| ListTranscriptionJobs | Supported | Lists transcription jobs |
| DeleteTranscriptionJob | Supported | Deletes a transcription job |
| CreateVocabulary | Supported | Creates a custom vocabulary |
| GetVocabulary | Supported | Returns vocabulary details |
| ListVocabularies | Supported | Lists custom vocabularies |
| DeleteVocabulary | Supported | Deletes a vocabulary |
| UpdateVocabulary | Supported | Updates a vocabulary |
| TagResource | Supported | Adds tags to a resource |
| UntagResource | Supported | Removes tags from a resource |
| ListTagsForResource | Supported | Lists tags for a resource |

## Quick Start

### Node.js

```typescript
import { TranscribeClient, StartTranscriptionJobCommand } from '@aws-sdk/client-transcribe';

const client = new TranscribeClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await client.send(new StartTranscriptionJobCommand({
  TranscriptionJobName: 'my-job',
  LanguageCode: 'en-US',
  Media: { MediaFileUri: 's3://my-bucket/audio.mp3' },
}));
```

### Python

```python
import boto3

client = boto3.client('transcribe',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

client.start_transcription_job(
    TranscriptionJobName='my-job',
    LanguageCode='en-US',
    Media={'MediaFileUri': 's3://my-bucket/audio.mp3'})
```

## Configuration

```yaml
# cloudmock.yml
services:
  transcribe:
    enabled: true
```

## Known Differences from AWS

- No actual speech-to-text transcription is performed
- Job results contain stub transcript data
- Vocabularies are stored but not used in transcription
