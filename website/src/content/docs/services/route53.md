---
title: Route 53
description: Amazon Route 53 DNS emulation in CloudMock
---

## Overview

CloudMock emulates Amazon Route 53, supporting hosted zone management and DNS record set operations. Records are stored for reference but DNS resolution is not performed.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateHostedZone | Supported | Creates a public or private hosted zone |
| ListHostedZones | Supported | Returns all hosted zones |
| GetHostedZone | Supported | Returns details for a specific zone |
| DeleteHostedZone | Supported | Deletes a hosted zone |
| ChangeResourceRecordSets | Supported | Creates, updates, or deletes DNS records |
| ListResourceRecordSets | Supported | Returns all DNS records in a zone |

## Quick Start

### curl

```bash
# Create a hosted zone
curl -X POST http://localhost:4566/2013-04-01/hostedzone \
  -H "Content-Type: application/xml" \
  -d '<CreateHostedZoneRequest xmlns="https://route53.amazonaws.com/doc/2013-04-01/">
    <Name>example.com</Name>
    <CallerReference>unique-ref-1</CallerReference>
  </CreateHostedZoneRequest>'

# List hosted zones
curl http://localhost:4566/2013-04-01/hostedzone
```

### Node.js

```typescript
import { Route53Client, CreateHostedZoneCommand, ChangeResourceRecordSetsCommand } from '@aws-sdk/client-route-53';

const r53 = new Route53Client({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const zone = await r53.send(new CreateHostedZoneCommand({
  Name: 'example.com', CallerReference: 'ref-1',
}));

await r53.send(new ChangeResourceRecordSetsCommand({
  HostedZoneId: zone.HostedZone!.Id!,
  ChangeBatch: {
    Changes: [{
      Action: 'UPSERT',
      ResourceRecordSet: {
        Name: 'api.example.com', Type: 'A', TTL: 300,
        ResourceRecords: [{ Value: '10.0.0.1' }],
      },
    }],
  },
}));
```

### Python

```python
import boto3

r53 = boto3.client('route53', endpoint_url='http://localhost:4566',
                   aws_access_key_id='test', aws_secret_access_key='test',
                   region_name='us-east-1')

zone = r53.create_hosted_zone(Name='example.com', CallerReference='ref-1')
zone_id = zone['HostedZone']['Id']

r53.change_resource_record_sets(
    HostedZoneId=zone_id,
    ChangeBatch={
        'Changes': [{
            'Action': 'UPSERT',
            'ResourceRecordSet': {
                'Name': 'api.example.com', 'Type': 'A', 'TTL': 300,
                'ResourceRecords': [{'Value': '10.0.0.1'}],
            },
        }],
    },
)

records = r53.list_resource_record_sets(HostedZoneId=zone_id)
for rrs in records['ResourceRecordSets']:
    print(rrs['Name'], rrs['Type'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  route53:
    enabled: true
```

No additional service-specific configuration is required.

## Known Differences from AWS

- **DNS resolution** is not performed -- records are stored for reference only.
- `ChangeResourceRecordSets` supports `CREATE`, `DELETE`, and `UPSERT` actions.
- **Alias records** are accepted and stored but not resolved.
- **Traffic policies**, **health checks**, and **DNSSEC** are not implemented.
- **Hosted zone delegation** is not implemented.

## Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| NoSuchHostedZone | 404 | The specified hosted zone does not exist |
| HostedZoneAlreadyExists | 409 | A hosted zone with this name already exists |
| InvalidChangeBatch | 400 | The change batch is not valid |
| InvalidInput | 400 | An input parameter is not valid |
