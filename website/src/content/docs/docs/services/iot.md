---
title: IoT Core
description: AWS IoT Core emulation in CloudMock
---

## Overview

CloudMock emulates AWS IoT Core, supporting things, thing types, thing groups, policies, certificates, topic rules, jobs, and tagging. The Thing → Certificate → Policy relationship chain is fully implemented with certificate state management (ACTIVE, INACTIVE, REVOKED) and job lifecycle (IN_PROGRESS, COMPLETED, CANCELLED).

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateThing | Supported | Creates a thing with optional type and attributes |
| DescribeThing | Supported | Returns thing details including version |
| ListThings | Supported | Lists all things |
| UpdateThing | Supported | Updates thing type and attributes, increments version |
| DeleteThing | Supported | Deletes a thing |
| CreateThingType | Supported | Creates a thing type |
| DescribeThingType | Supported | Returns thing type details with metadata |
| ListThingTypes | Supported | Lists thing types |
| DeleteThingType | Supported | Deletes a thing type |
| CreateThingGroup | Supported | Creates a thing group with optional parent |
| DescribeThingGroup | Supported | Returns thing group details |
| ListThingGroups | Supported | Lists thing groups |
| DeleteThingGroup | Supported | Deletes a thing group |
| AddThingToThingGroup | Supported | Adds a thing to a group |
| RemoveThingFromThingGroup | Supported | Removes a thing from a group |
| CreatePolicy | Supported | Creates an IoT policy |
| GetPolicy | Supported | Returns policy details |
| ListPolicies | Supported | Lists policies |
| DeletePolicy | Supported | Deletes a policy |
| AttachPolicy | Supported | Attaches a policy to a target (cert ARN) |
| DetachPolicy | Supported | Detaches a policy from a target |
| ListAttachedPolicies | Supported | Lists policies attached to a target |
| ListTargetsForPolicy | Supported | Lists targets a policy is attached to |
| CreateKeysAndCertificate | Supported | Creates mock keys and certificate |
| DescribeCertificate | Supported | Returns certificate details and status |
| ListCertificates | Supported | Lists certificates |
| UpdateCertificate | Supported | Changes certificate status (ACTIVE/INACTIVE/REVOKED) |
| DeleteCertificate | Supported | Deletes a certificate |
| AttachThingPrincipal | Supported | Attaches a certificate to a thing |
| DetachThingPrincipal | Supported | Detaches a certificate from a thing |
| ListThingPrincipals | Supported | Lists certificates attached to a thing |
| CreateTopicRule | Supported | Creates a topic rule (SQL must start with SELECT) |
| GetTopicRule | Supported | Returns topic rule details |
| ListTopicRules | Supported | Lists topic rules |
| DeleteTopicRule | Supported | Deletes a topic rule |
| CreateJob | Supported | Creates a job targeting things/groups |
| DescribeJob | Supported | Returns job details and status |
| ListJobs | Supported | Lists all jobs |
| CancelJob | Supported | Cancels an IN_PROGRESS job |
| TagResource | Supported | Adds tags to a resource |
| UntagResource | Supported | Removes tags from a resource |
| ListTagsForResource | Supported | Lists tags for a resource |

## Quick Start

### Node.js

```typescript
import { IoTClient, CreateThingCommand, CreateKeysAndCertificateCommand, AttachThingPrincipalCommand } from '@aws-sdk/client-iot';

const client = new IoTClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

// Create a thing
const { thingName, thingArn } = await client.send(new CreateThingCommand({
  thingName: 'my-sensor',
}));

// Create and attach certificate
const { certificateArn, certificateId } = await client.send(
  new CreateKeysAndCertificateCommand({ setAsActive: true })
);

await client.send(new AttachThingPrincipalCommand({
  thingName: 'my-sensor',
  principal: certificateArn,
}));
console.log(thingArn, certificateArn);
```

### Python

```python
import boto3

client = boto3.client('iot',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

thing = client.create_thing(thingName='my-sensor')
cert = client.create_keys_and_certificate(setAsActive=True)
client.attach_thing_principal(
    thingName='my-sensor',
    principal=cert['certificateArn']
)

# Create a job
client.create_job(
    jobId='firmware-update-001',
    targets=[thing['thingArn']],
    description='Firmware update',
    documentSource='s3://my-bucket/firmware-job.json',
)
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
- Job execution tracking per-device is not implemented (only job-level status)
</content>
