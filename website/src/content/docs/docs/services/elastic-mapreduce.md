---
title: EMR
description: Amazon EMR (Elastic MapReduce) emulation in CloudMock
---

## Overview

CloudMock emulates Amazon EMR, supporting cluster lifecycle, job flow steps, instance groups, and tagging.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| RunJobFlow | Supported | Creates an EMR cluster |
| DescribeCluster | Supported | Returns cluster details |
| ListClusters | Supported | Lists clusters |
| TerminateJobFlows | Supported | Terminates clusters |
| AddJobFlowSteps | Supported | Adds steps to a cluster |
| ListSteps | Supported | Lists steps for a cluster |
| DescribeStep | Supported | Returns step details |
| AddInstanceGroups | Supported | Adds instance groups |
| ListInstanceGroups | Supported | Lists instance groups |
| ModifyInstanceGroups | Supported | Modifies instance group capacity |
| SetTerminationProtection | Supported | Enables or disables termination protection |
| SetVisibleToAllUsers | Supported | Sets cluster visibility |
| AddTags | Supported | Adds tags to a cluster |
| RemoveTags | Supported | Removes tags from a cluster |

## Quick Start

### Node.js

```typescript
import { EMRClient, RunJobFlowCommand } from '@aws-sdk/client-emr';

const client = new EMRClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const { JobFlowId } = await client.send(new RunJobFlowCommand({
  Name: 'my-cluster',
  ReleaseLabel: 'emr-6.10.0',
  Instances: {
    MasterInstanceType: 'm5.xlarge',
    SlaveInstanceType: 'm5.xlarge',
    InstanceCount: 3,
  },
  ServiceRole: 'EMR_DefaultRole',
  JobFlowRole: 'EMR_EC2_DefaultRole',
}));
console.log(JobFlowId);
```

### Python

```python
import boto3

client = boto3.client('emr',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

response = client.run_job_flow(
    Name='my-cluster',
    ReleaseLabel='emr-6.10.0',
    Instances={
        'MasterInstanceType': 'm5.xlarge',
        'SlaveInstanceType': 'm5.xlarge',
        'InstanceCount': 3,
    },
    ServiceRole='EMR_DefaultRole',
    JobFlowRole='EMR_EC2_DefaultRole')
print(response['JobFlowId'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  elasticmapreduce:
    enabled: true
```

## Known Differences from AWS

- No actual Hadoop/Spark clusters are provisioned
- Steps are stored but not executed
- Instance groups do not launch real EC2 instances
