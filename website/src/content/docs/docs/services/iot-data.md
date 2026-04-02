---
title: IoT Data Plane
description: AWS IoT Data Plane emulation in CloudMock
---

## Overview

CloudMock emulates the AWS IoT Data Plane, supporting device shadow operations and MQTT message publishing. Device shadows are JSON documents with `state` (desired/reported), `metadata`, and `version` fields. Versions increment on each update, and delta computation between desired and reported state is supported.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| GetThingShadow | Supported | Returns shadow document with state, metadata, version, timestamp |
| UpdateThingShadow | Supported | Merges state, increments version, computes delta |
| DeleteThingShadow | Supported | Deletes a shadow document |
| ListNamedShadowsForThing | Supported | Lists named shadows (excludes classic shadow) |
| Publish | Supported | Stores message to topic (no actual MQTT delivery) |

## Shadow Document Structure

```json
{
  "state": {
    "desired": { "temp": 72, "mode": "auto" },
    "reported": { "temp": 70 },
    "delta": { "temp": 72 }
  },
  "metadata": {},
  "version": 3,
  "timestamp": 1234567890
}
```

## Quick Start

### Node.js

```typescript
import { IoTDataPlaneClient, UpdateThingShadowCommand, GetThingShadowCommand } from '@aws-sdk/client-iot-data-plane';

const client = new IoTDataPlaneClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

// Update shadow (desired and reported)
await client.send(new UpdateThingShadowCommand({
  thingName: 'my-sensor',
  payload: JSON.stringify({
    state: {
      desired: { temp: 72, mode: 'auto' },
      reported: { temp: 70 },
    }
  }),
}));

// Get shadow (includes delta when desired != reported)
const shadow = await client.send(new GetThingShadowCommand({
  thingName: 'my-sensor',
}));
const doc = JSON.parse(new TextDecoder().decode(shadow.payload));
console.log('delta:', doc.state.delta); // { temp: 72 }
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

# Update named shadow
client.update_thing_shadow(
    thingName='my-sensor',
    shadowName='config',
    payload=json.dumps({'state': {'desired': {'interval': 30}}}))

# List named shadows
response = client.list_named_shadows_for_thing(thingName='my-sensor')
print(response['results'])  # ['config']

# Publish to topic
client.publish(topic='sensors/temp', payload=b'{"value": 72}', qos=0)
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
- Shadow versioning does not reject stale versions (no version conflict detection)
- Shadow metadata timestamps are set to creation time, not per-field timestamps
</content>
