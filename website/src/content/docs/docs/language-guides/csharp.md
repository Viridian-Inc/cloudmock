---
title: C# / .NET
description: Using CloudMock with the AWS SDK for .NET
---

# C# / .NET

## CloudMock .NET SDK

The CloudMock SDK auto-manages the server lifecycle:

```csharp
using CloudMock;

using var cm = new CloudMockServer();
// cm.Endpoint contains the URL
```

Install: `dotnet add package CloudMock`

## Manual Setup

If you prefer to manage CloudMock yourself:

```csharp
using Amazon.S3;
using Amazon.Runtime;

var config = new AmazonS3Config
{
    ServiceURL = "http://localhost:4566",
    ForcePathStyle = true
};

var creds = new BasicAWSCredentials("test", "test");
var s3 = new AmazonS3Client(creds, config);

await s3.PutBucketAsync("my-bucket");
```

## DynamoDB Example

```csharp
using Amazon.DynamoDBv2;
using Amazon.DynamoDBv2.Model;

var ddbConfig = new AmazonDynamoDBConfig { ServiceURL = "http://localhost:4566" };
var ddb = new AmazonDynamoDBClient(new BasicAWSCredentials("test", "test"), ddbConfig);

await ddb.CreateTableAsync(new CreateTableRequest
{
    TableName = "users",
    KeySchema = new List<KeySchemaElement>
    {
        new("pk", KeyType.HASH)
    },
    AttributeDefinitions = new List<AttributeDefinition>
    {
        new("pk", ScalarAttributeType.S)
    },
    BillingMode = BillingMode.PAY_PER_REQUEST
});
```

## Testing with xUnit

```csharp
using CloudMock;
using Xunit;

public class AWSTests : IClassFixture<CloudMockFixture>
{
    private readonly CloudMockServer _cm;

    public AWSTests(CloudMockFixture fixture) => _cm = fixture.Server;

    [Fact]
    public async Task CanCreateBucket()
    {
        var s3 = new AmazonS3Client(
            new BasicAWSCredentials("test", "test"),
            new AmazonS3Config { ServiceURL = _cm.Endpoint, ForcePathStyle = true });
        await s3.PutBucketAsync("test");
    }
}

public class CloudMockFixture : IDisposable
{
    public CloudMockServer Server { get; } = new();
    public void Dispose() => Server.Dispose();
}
```
