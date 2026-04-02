---
title: MSK (Kafka)
description: Amazon Managed Streaming for Apache Kafka emulation in CloudMock
---

## Overview

CloudMock emulates Amazon MSK, supporting cluster management, broker operations, configurations, cluster operations tracking, and tagging.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateCluster | Supported | Creates an MSK cluster |
| DescribeCluster | Supported | Returns cluster details |
| ListClusters | Supported | Lists all clusters |
| DeleteCluster | Supported | Deletes a cluster |
| UpdateBrokerCount | Supported | Scales the number of brokers |
| UpdateBrokerStorage | Supported | Updates broker storage |
| UpdateClusterConfiguration | Supported | Updates cluster configuration |
| RebootBroker | Supported | Reboots a broker |
| GetBootstrapBrokers | Supported | Returns bootstrap broker connection strings |
| ListNodes | Supported | Lists broker nodes |
| CreateConfiguration | Supported | Creates a configuration |
| DescribeConfiguration | Supported | Returns configuration details |
| ListConfigurations | Supported | Lists configurations |
| UpdateConfiguration | Supported | Updates a configuration |
| DeleteConfiguration | Supported | Deletes a configuration |
| ListClusterOperations | Supported | Lists cluster operations |
| DescribeClusterOperation | Supported | Returns operation details |
| TagResource | Supported | Adds tags to a resource |
| UntagResource | Supported | Removes tags from a resource |
| ListTagsForResource | Supported | Lists tags for a resource |

## Quick Start

### Node.js

```typescript
import { KafkaClient, CreateClusterCommand } from '@aws-sdk/client-kafka';

const client = new KafkaClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const { ClusterArn } = await client.send(new CreateClusterCommand({
  ClusterName: 'my-kafka-cluster',
  KafkaVersion: '3.5.1',
  NumberOfBrokerNodes: 3,
  BrokerNodeGroupInfo: {
    InstanceType: 'kafka.m5.large',
    ClientSubnets: ['subnet-1', 'subnet-2', 'subnet-3'],
  },
}));
console.log(ClusterArn);
```

### Python

```python
import boto3

client = boto3.client('kafka',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

response = client.create_cluster(
    ClusterName='my-kafka-cluster',
    KafkaVersion='3.5.1',
    NumberOfBrokerNodes=3,
    BrokerNodeGroupInfo={
        'InstanceType': 'kafka.m5.large',
        'ClientSubnets': ['subnet-1', 'subnet-2', 'subnet-3'],
    })
print(response['ClusterArn'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  kafka:
    enabled: true
```

## Known Differences from AWS

- No actual Kafka brokers are provisioned
- Bootstrap broker strings are placeholders
- Broker reboots and scaling are status changes only
