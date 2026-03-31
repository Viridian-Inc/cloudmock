---
title: First Request
description: Create an S3 bucket, upload a file, and list objects in 30 seconds
---

This tutorial takes 30 seconds. You will create an S3 bucket, upload a file, and read it back -- first with curl, then with the AWS CLI.

## Prerequisites

CloudMock must be running on `localhost:4566`. If it is not, see [Installation](/getting-started/installation/).

## Set credentials

CloudMock's default root credentials are `test` / `test`. Export them so every command in this tutorial authenticates correctly:

```bash
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_DEFAULT_REGION=us-east-1
export AWS_ENDPOINT_URL=http://localhost:4566
```

## Option A: curl

### Create a bucket

```bash
curl -X PUT http://localhost:4566/my-bucket
```

Expected output:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<CreateBucketResult>
  <Location>/my-bucket</Location>
</CreateBucketResult>
```

### Upload a file

```bash
echo "Hello, CloudMock!" | curl -X PUT --data-binary @- \
  http://localhost:4566/my-bucket/hello.txt
```

Expected output: HTTP 200 with an empty body and an `ETag` header.

### List objects

```bash
curl http://localhost:4566/my-bucket?list-type=2
```

Expected output:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<ListBucketResult>
  <Name>my-bucket</Name>
  <Contents>
    <Key>hello.txt</Key>
    <Size>18</Size>
    <StorageClass>STANDARD</StorageClass>
  </Contents>
</ListBucketResult>
```

### Read the file back

```bash
curl http://localhost:4566/my-bucket/hello.txt
```

Expected output:

```
Hello, CloudMock!
```

## Option B: AWS CLI

The AWS CLI is the standard way to interact with AWS services. If you have it installed, it works with CloudMock out of the box once `AWS_ENDPOINT_URL` is set.

### Create a bucket

```bash
aws s3 mb s3://my-bucket
```

Expected output:

```
make_bucket: my-bucket
```

### Upload a file

```bash
echo "Hello, CloudMock!" > hello.txt
aws s3 cp hello.txt s3://my-bucket/hello.txt
```

Expected output:

```
upload: ./hello.txt to s3://my-bucket/hello.txt
```

### List objects

```bash
aws s3 ls s3://my-bucket
```

Expected output:

```
2026-03-31 00:00:00         18 hello.txt
```

### Read the file back

```bash
aws s3 cp s3://my-bucket/hello.txt -
```

Expected output:

```
Hello, CloudMock!
```

## What just happened

You pointed standard AWS tools at `localhost:4566` and used them exactly as you would against real AWS. CloudMock handled the S3 API calls -- creating the bucket in memory, storing the object, and returning it on request.

No AWS account. No internet. No cost.

## Next step

You have made your first request. Now [configure your SDK](/getting-started/with-your-stack/) to use CloudMock in your application code.
