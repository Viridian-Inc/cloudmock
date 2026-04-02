---
title: IoT Core
description: AWS IoT Core emulation in CloudMock
---

## Overview

CloudMock emulates AWS IoT Core, supporting things, thing types, thing groups, policies, certificates, topic rules, and tagging.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateThing | Supported | Creates a thing |
| DescribeThing | Supported | Returns thing details |
| ListThings | Supported | Lists all things |
| UpdateThing | Supported | Updates a thing |
| DeleteThing | Supported | Deletes a thing |
| CreateThingType | Supported | Creates a thing type |
| DescribeThingType | Supported | Returns thing type details |
| ListThingTypes | Supported | Lists thing types |
| DeleteThingType | Supported | Deletes a thing type |
| CreateThingGroup | Supported | Creates a thing group |
| DescribeThingGroup | Supported | Returns thing group details |
| ListThingGroups | Supported | Lists thing groups |
| DeleteThingGroup | Supported | Deletes a thing group |
| AddThingToThingGroup | Supported | Adds a thing to a group |
| RemoveThingFromThingGroup | Supported | Removes a thing from a group |
| CreatePolicy | Supported | Creates an IoT policy |
| GetPolicy | Supported | Returns policy details |
| ListPolicies | Supported | Lists policies |
| DeletePolicy | Supported | Deletes a policy |
| AttachPolicy | Supported | Attaches a policy to a target |
| DetachPolicy | Supported | Detaches a policy |
| ListAttachedPolicies | Supported | Lists attached policies |
| CreateKeysAndCertificate | Supported | Creates keys and certificate |
| DescribeCertificate | Supported | Returns certificate details |
| ListCertificates | Supported | Lists certificates |
| DeleteCertificate | Supported | Deletes a certificate |
| AttachThingPrincipal | Supported | Attaches a principal to a thing |
| DetachThingPrincipal | Supported | Detaches a principal |
| ListThingPrincipals | Supported | Lists thing principals |
| CreateTopicRule | Supported | Creates a topic rule |
| GetTopicRule | Supported | Returns topic rule details |
| ListTopicRules | Supported | Lists topic rules |
| DeleteTopicRule | Supported | Deletes a topic rule |
| TagResource | Supported | Adds tags to a resource |
| UntagResource | Supported | Removes tags from a resource |
| ListTagsForResource | Supported | Lists tags for a resource |

## Quick Start

### Node.js

```typescript
import { IoTClient, CreateThingCommand } from '@aws-sdk/client-iot';

const client = new IoTClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const { thingName, thingArn } = await client.send(new CreateThingCommand({
  thingName: 'my-sensor',
}));
console.log(thingArn);
```

### Python

```python
import boto3

client = boto3.client('iot',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

response = client.create_thing(thingName='my-sensor')
print(response['thingArn'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  iot:
    enabled: true
```

## Known Differences from AWS

- Certificates are stubs and cannot be used for real MQTT connections
- Topic rules are stored but do not trigger actions
- MQTT message broker is not implemented
