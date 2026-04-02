---
title: EKS
description: Amazon Elastic Kubernetes Service emulation in CloudMock
---

## Overview

CloudMock emulates Amazon EKS, supporting cluster management, node groups, Fargate profiles, add-ons, and tagging.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateCluster | Supported | Creates an EKS cluster |
| DescribeCluster | Supported | Returns cluster details |
| ListClusters | Supported | Lists all clusters |
| DeleteCluster | Supported | Deletes a cluster |
| UpdateClusterConfig | Supported | Updates cluster configuration |
| CreateNodegroup | Supported | Creates a managed node group |
| DescribeNodegroup | Supported | Returns node group details |
| ListNodegroups | Supported | Lists node groups for a cluster |
| DeleteNodegroup | Supported | Deletes a node group |
| UpdateNodegroupConfig | Supported | Updates node group configuration |
| CreateFargateProfile | Supported | Creates a Fargate profile |
| DescribeFargateProfile | Supported | Returns Fargate profile details |
| ListFargateProfiles | Supported | Lists Fargate profiles |
| DeleteFargateProfile | Supported | Deletes a Fargate profile |
| CreateAddon | Supported | Creates an add-on |
| DescribeAddon | Supported | Returns add-on details |
| ListAddons | Supported | Lists add-ons for a cluster |
| DeleteAddon | Supported | Deletes an add-on |
| TagResource | Supported | Adds tags to a resource |
| UntagResource | Supported | Removes tags from a resource |
| ListTagsForResource | Supported | Lists tags for a resource |

## Quick Start

### Node.js

```typescript
import { EKSClient, CreateClusterCommand } from '@aws-sdk/client-eks';

const client = new EKSClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

await client.send(new CreateClusterCommand({
  name: 'my-cluster',
  roleArn: 'arn:aws:iam::000000000000:role/eks-role',
  resourcesVpcConfig: { subnetIds: ['subnet-12345'], securityGroupIds: ['sg-12345'] },
}));
```

### Python

```python
import boto3

client = boto3.client('eks',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

client.create_cluster(
    name='my-cluster',
    roleArn='arn:aws:iam::000000000000:role/eks-role',
    resourcesVpcConfig={'subnetIds': ['subnet-12345'], 'securityGroupIds': ['sg-12345']})
```

## Configuration

```yaml
# cloudmock.yml
services:
  eks:
    enabled: true
```

## Known Differences from AWS

- No actual Kubernetes control plane is provisioned
- Node groups do not launch real EC2 instances
- Cluster endpoint and certificate authority data are stubs
- Add-ons are stored but not installed
