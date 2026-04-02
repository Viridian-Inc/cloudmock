---
title: IoT Data Plane
description: AWS IoT Data Plane emulation in CloudMock
---

## Overview

CloudMock emulates the AWS IoT Data Plane, supporting device shadow operations and MQTT message publishing.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| GetThingShadow | Supported | Returns a thing's shadow document |
| UpdateThingShadow | Supported | Updates a thing's shadow |
| DeleteThingShadow | Supported | Deletes a thing's shadow |
| ListNamedShadowsForThing | Supported | Lists named shadows for a thing |
| Publish | Supported | Publishes a message to a topic (stub) |

## Quick Start

### Node.js

```typescript
import { IoTDataPlaneClient, UpdateThingShadowCommand, GetThingShadowCommand } from '@aws-sdk/client-iot-data-plane';

const client = new IoTDataPlaneClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await client.send(new UpdateThingShadowCommand({
  thingName: 'my-sensor',
  payload: JSON.stringify({ state: { desired: { temp: 72 } } }),
}));

const shadow = await client.send(new GetThingShadowCommand({
  thingName: 'my-sensor',
}));
console.log(new TextDecoder().decode(shadow.payload));
```

### Python

```python
import boto3
import json

client = boto3.client('iot-data',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

client.update_thing_shadow(
    thingName='my-sensor',
    payload=json.dumps({'state': {'desired': {'temp': 72}}}))

response = client.get_thing_shadow(thingName='my-sensor')
print(json.loads(response['payload'].read()))
```

## Configuration

```yaml
# cloudmock.yml
services:
  iotdata:
    enabled: true
```

## Known Differences from AWS

- Publish does not deliver messages to MQTT subscribers
- Shadow versioning is simplified
- Shadow delta calculations may not match AWS behavior exactly
