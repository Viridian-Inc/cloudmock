---
title: EC2
description: Amazon EC2 emulation in CloudMock
---

## Overview

CloudMock emulates Amazon EC2, providing comprehensive support for VPC networking, instances, security groups, Elastic IPs, network interfaces, NAT gateways, internet gateways, route tables, VPC endpoints, VPC peering, and network ACLs.

## Supported Operations

### VPC & Subnets

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateVpc | Supported | Creates a VPC |
| DescribeVpcs | Supported | Lists VPCs |
| DeleteVpc | Supported | Deletes a VPC |
| ModifyVpcAttribute | Supported | Modifies VPC attributes |
| CreateSubnet | Supported | Creates a subnet in a VPC |
| DescribeSubnets | Supported | Lists subnets |
| DeleteSubnet | Supported | Deletes a subnet |

### Instances

| Operation | Status | Notes |
|-----------|--------|-------|
| RunInstances | Supported | Launches instances |
| DescribeInstances | Supported | Lists instances |
| TerminateInstances | Supported | Terminates instances |
| StopInstances | Supported | Stops instances |
| StartInstances | Supported | Starts stopped instances |
| DescribeInstanceStatus | Supported | Returns instance status |

### Security Groups

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateSecurityGroup | Supported | Creates a security group |
| DescribeSecurityGroups | Supported | Lists security groups |
| DeleteSecurityGroup | Supported | Deletes a security group |
| AuthorizeSecurityGroupIngress | Supported | Adds inbound rules |
| AuthorizeSecurityGroupEgress | Supported | Adds outbound rules |
| RevokeSecurityGroupIngress | Supported | Removes inbound rules |
| RevokeSecurityGroupEgress | Supported | Removes outbound rules |

### Networking

| Operation | Status | Notes |
|-----------|--------|-------|
| AllocateAddress | Supported | Allocates an Elastic IP |
| ReleaseAddress | Supported | Releases an Elastic IP |
| AssociateAddress | Supported | Associates an EIP with an instance |
| DisassociateAddress | Supported | Disassociates an EIP |
| DescribeAddresses | Supported | Lists Elastic IPs |
| CreateNetworkInterface | Supported | Creates an ENI |
| DescribeNetworkInterfaces | Supported | Lists ENIs |
| DeleteNetworkInterface | Supported | Deletes an ENI |
| CreateNetworkAcl | Supported | Creates a network ACL |
| DescribeNetworkAcls | Supported | Lists network ACLs |
| DeleteNetworkAcl | Supported | Deletes a network ACL |
| CreateNetworkAclEntry | Supported | Adds a network ACL entry |
| DeleteNetworkAclEntry | Supported | Removes a network ACL entry |

### Gateways & Route Tables

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateInternetGateway | Supported | Creates an internet gateway |
| AttachInternetGateway | Supported | Attaches IGW to a VPC |
| DetachInternetGateway | Supported | Detaches IGW from a VPC |
| DeleteInternetGateway | Supported | Deletes an internet gateway |
| DescribeInternetGateways | Supported | Lists internet gateways |
| CreateNatGateway | Supported | Creates a NAT gateway |
| DescribeNatGateways | Supported | Lists NAT gateways |
| DeleteNatGateway | Supported | Deletes a NAT gateway |
| CreateRouteTable | Supported | Creates a route table |
| DescribeRouteTables | Supported | Lists route tables |
| DeleteRouteTable | Supported | Deletes a route table |
| CreateRoute | Supported | Adds a route |
| DeleteRoute | Supported | Removes a route |
| ReplaceRoute | Supported | Replaces a route |
| AssociateRouteTable | Supported | Associates a route table with a subnet |
| DisassociateRouteTable | Supported | Disassociates a route table |

### VPC Endpoints & Peering

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateVpcEndpoint | Supported | Creates a VPC endpoint |
| DescribeVpcEndpoints | Supported | Lists VPC endpoints |
| DeleteVpcEndpoints | Supported | Deletes VPC endpoints |
| CreateVpcPeeringConnection | Supported | Creates a peering connection |
| AcceptVpcPeeringConnection | Supported | Accepts a peering connection |
| DeleteVpcPeeringConnection | Supported | Deletes a peering connection |

### Tags

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateTags | Supported | Adds tags to resources |
| DeleteTags | Supported | Removes tags from resources |
| DescribeTags | Supported | Lists tags |

## Quick Start

### Node.js

```typescript
import { EC2Client, RunInstancesCommand, CreateVpcCommand } from '@aws-sdk/client-ec2';

const client = new EC2Client({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const vpc = await client.send(new CreateVpcCommand({ CidrBlock: '10.0.0.0/16' }));
console.log(vpc.Vpc.VpcId);

const instances = await client.send(new RunInstancesCommand({
  ImageId: 'ami-12345678',
  InstanceType: 't2.micro',
  MinCount: 1,
  MaxCount: 1,
}));
console.log(instances.Instances[0].InstanceId);
```

### Python

```python
import boto3

client = boto3.client('ec2',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

vpc = client.create_vpc(CidrBlock='10.0.0.0/16')
print(vpc['Vpc']['VpcId'])

instances = client.run_instances(
    ImageId='ami-12345678',
    InstanceType='t2.micro',
    MinCount=1, MaxCount=1)
print(instances['Instances'][0]['InstanceId'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  ec2:
    enabled: true
```

## Known Differences from AWS

- Instances do not run actual compute workloads
- AMI validation is not performed
- Network traffic between resources is not simulated
- CIDR range validation is performed but some edge cases may differ
