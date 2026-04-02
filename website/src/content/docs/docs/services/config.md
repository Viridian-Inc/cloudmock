---
title: Config
description: AWS Config emulation in CloudMock
---

## Overview

CloudMock emulates AWS Config, supporting config rules, configuration recorders, delivery channels, compliance evaluation, and conformance packs.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| PutConfigRule | Supported | Creates or updates a config rule |
| DescribeConfigRules | Supported | Lists config rules |
| DeleteConfigRule | Supported | Deletes a config rule |
| PutConfigurationRecorder | Supported | Creates or updates a configuration recorder |
| DescribeConfigurationRecorders | Supported | Lists configuration recorders |
| DeleteConfigurationRecorder | Supported | Deletes a configuration recorder |
| PutDeliveryChannel | Supported | Creates or updates a delivery channel |
| DescribeDeliveryChannels | Supported | Lists delivery channels |
| DeleteDeliveryChannel | Supported | Deletes a delivery channel |
| StartConfigurationRecorder | Supported | Starts recording |
| StopConfigurationRecorder | Supported | Stops recording |
| GetComplianceDetailsByConfigRule | Supported | Returns compliance details |
| DescribeComplianceByConfigRule | Supported | Returns compliance summary by rule |
| PutConformancePack | Supported | Creates or updates a conformance pack |
| DescribeConformancePacks | Supported | Lists conformance packs |
| DeleteConformancePack | Supported | Deletes a conformance pack |
| PutEvaluations | Supported | Submits evaluation results |

## Quick Start

### Node.js

```typescript
import { ConfigServiceClient, PutConfigRuleCommand } from '@aws-sdk/client-config-service';

const client = new ConfigServiceClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await client.send(new PutConfigRuleCommand({
  ConfigRule: {
    ConfigRuleName: 's3-bucket-versioning',
    Source: { Owner: 'AWS', SourceIdentifier: 'S3_BUCKET_VERSIONING_ENABLED' },
  },
}));
```

### Python

```python
import boto3

client = boto3.client('config',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

client.put_config_rule(ConfigRule={
    'ConfigRuleName': 's3-bucket-versioning',
    'Source': {'Owner': 'AWS', 'SourceIdentifier': 'S3_BUCKET_VERSIONING_ENABLED'},
})
```

## Configuration

```yaml
# cloudmock.yml
services:
  config:
    enabled: true
```

## Known Differences from AWS

- Config rules do not evaluate resources automatically
- Configuration recorder does not capture real resource changes
- Compliance results are based on submitted evaluations only
