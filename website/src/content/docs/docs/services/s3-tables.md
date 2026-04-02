---
title: S3 Tables
description: AWS S3 Tables emulation in CloudMock
---

## Overview

CloudMock emulates AWS S3 Tables, supporting the full TableBucket → Namespace → Table hierarchy with table policies and metadata location updates. S3 Tables is designed for Apache Iceberg table management.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateTableBucket | Supported | Validates bucket name (lowercase, no underscores) |
| GetTableBucket | Supported | Returns bucket details |
| ListTableBuckets | Supported | Lists all table buckets |
| DeleteTableBucket | Supported | Deletes bucket and its tables |
| CreateNamespace | Supported | Creates namespace within a bucket |
| GetNamespace | Supported | Returns namespace details |
| ListNamespaces | Supported | Lists namespaces for a bucket |
| DeleteNamespace | Supported | Deletes a namespace |
| CreateTable | Supported | Creates table in bucket/namespace |
| GetTable | Supported | Returns table details |
| ListTables | Supported | Lists tables for a bucket |
| UpdateTableMetadataLocation | Supported | Updates Iceberg metadata location |
| DeleteTable | Supported | Deletes a table |
| GetTablePolicy | Supported | Returns table resource policy |
| PutTablePolicy | Supported | Sets table policy (validates JSON) |
| DeleteTablePolicy | Supported | Deletes table policy |

## Resource Hierarchy

```
TableBucket (arn:aws:s3tables:{region}:{account}:bucket/{name})
  └── Namespace (analytics, ml, etc.)
       └── Table (orders, customers, etc.)
            └── TablePolicy (IAM resource policy)
```

## Quick Start

```python
import boto3

client = boto3.client('s3tables',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

# Create table bucket
bucket = client.create_table_bucket(name='my-lakehouse')
bucket_arn = bucket['arn']

# Create namespace
client.create_namespace(
    tableBucketARN=bucket_arn,
    namespace=['analytics'],
)

# Create Iceberg table
client.create_table(
    tableBucketARN=bucket_arn,
    namespace='analytics',
    name='events',
    format='ICEBERG',
)

# Update metadata location after Iceberg write
client.update_table_metadata_location(
    tableBucketARN=bucket_arn,
    namespace='analytics',
    name='events',
    metadataLocation='s3://bucket/warehouse/analytics/events/metadata/v1.metadata.json',
    versionToken='old-token',
)
```

## Configuration

```yaml
services:
  s3tables:
    enabled: true
```

## Known Differences from AWS

- No actual Iceberg table files are created or managed
- MetadataLocation updates store the value but do not validate S3 paths
- Namespace and table ARNs are derived from bucket ARN
