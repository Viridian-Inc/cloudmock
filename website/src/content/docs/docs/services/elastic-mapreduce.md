---
title: EMR
description: Amazon EMR (Elastic MapReduce) emulation in CloudMock
---

## Overview

CloudMock emulates Amazon EMR, supporting cluster lifecycle, job flow steps, instance groups, security configurations, and tagging. Cluster state transitions are simulated using the lifecycle state machine.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| RunJobFlow | Supported | Creates an EMR cluster; state: STARTING → BOOTSTRAPPING → RUNNING |
| DescribeCluster | Supported | Returns cluster details and status |
| ListClusters | Supported | Lists clusters with state filtering |
| TerminateJobFlows | Supported | Terminates one or more clusters |
| AddJobFlowSteps | Supported | Adds steps to a running cluster |
| ListSteps | Supported | Lists steps for a cluster |
| DescribeStep | Supported | Returns step details and status |
| AddInstanceGroups | Supported | Adds MASTER, CORE, or TASK groups |
| ListInstanceGroups | Supported | Lists instance groups for a cluster |
| ModifyInstanceGroups | Supported | Updates instance counts |
| SetTerminationProtection | Supported | Enables or disables termination protection |
| SetVisibleToAllUsers | Supported | Sets cluster visibility |
| AddTags | Supported | Adds tags to a cluster |
| RemoveTags | Supported | Removes tags from a cluster |
| CreateSecurityConfiguration | Supported | Stores JSON security configuration |
| DescribeSecurityConfiguration | Supported | Returns security configuration by name |
| ListSecurityConfigurations | Supported | Lists all security configurations |
| DeleteSecurityConfiguration | Supported | Deletes a security configuration |

## Cluster State Machine

```
STARTING → BOOTSTRAPPING → RUNNING → WAITING → TERMINATING → TERMINATED
```

## Step State Machine

```
PENDING → RUNNING → COMPLETED / FAILED
```

## Quick Start

### Node.js

```typescript
import { EMRClient, RunJobFlowCommand, CreateSecurityConfigurationCommand } from '@aws-sdk/client-emr';

const client = new EMRClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await client.send(new CreateSecurityConfigurationCommand({
  Name: 'my-security-config',
  SecurityConfiguration: JSON.stringify({
    EncryptionConfiguration: { EnableInTransitEncryption: true },
  }),
}));

const { JobFlowId } = await client.send(new RunJobFlowCommand({
  Name: 'my-cluster',
  ReleaseLabel: 'emr-6.10.0',
  SecurityConfiguration: 'my-security-config',
  Instances: { MasterInstanceType: 'm5.xlarge', InstanceCount: 3 },
  ServiceRole: 'EMR_DefaultRole',
  JobFlowRole: 'EMR_EC2_DefaultRole',
}));
console.log(JobFlowId);
```

### Python

```python
import boto3, json

client = boto3.client('emr',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

client.create_security_configuration(
    Name='my-security-config',
    SecurityConfiguration=json.dumps({'EncryptionConfiguration': {'EnableInTransitEncryption': True}}))

response = client.run_job_flow(
    Name='my-cluster',
    ReleaseLabel='emr-6.10.0',
    Instances={'MasterInstanceType': 'm5.xlarge', 'InstanceCount': 3},
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
- Steps are stored but not executed against real compute
- Instance groups do not launch real EC2 instances
- Security configurations are stored as JSON blobs without enforcement
