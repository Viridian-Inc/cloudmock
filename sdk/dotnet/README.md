# CloudMock .NET SDK

Start a local AWS mock server and get pre-configured clients in your .NET tests.

## Installation

```sh
dotnet add package CloudMock
dotnet add package AWSSDK.S3
dotnet add package AWSSDK.DynamoDBv2
```

## Usage

`CloudMockServer` implements `IDisposable`, so it works cleanly with `using` blocks and xUnit/NUnit fixtures.

### xUnit — class fixture

```csharp
using Amazon.S3;
using Amazon.S3.Model;
using Amazon.Runtime;
using CloudMock;
using Xunit;

public class S3Fixture : IDisposable
{
    public CloudMockServer Server { get; } = new CloudMockServer();

    public AmazonS3Client S3Client()
    {
        var config = new AmazonS3Config
        {
            ServiceURL = Server.Endpoint,
            ForcePathStyle = true,
        };
        return new AmazonS3Client(new BasicAWSCredentials("test", "test"), config);
    }

    public void Dispose() => Server.Dispose();
}

public class S3Tests : IClassFixture<S3Fixture>
{
    private readonly S3Fixture _fixture;

    public S3Tests(S3Fixture fixture) => _fixture = fixture;

    [Fact]
    public async Task CreateAndListBucket()
    {
        var s3 = _fixture.S3Client();
        await s3.PutBucketAsync("my-test-bucket");

        var resp = await s3.ListBucketsAsync();
        Assert.Contains(resp.Buckets, b => b.BucketName == "my-test-bucket");
    }
}
```

### DynamoDB example

```csharp
using Amazon.DynamoDBv2;
using Amazon.DynamoDBv2.Model;
using Amazon.Runtime;
using CloudMock;

using var server = new CloudMockServer(region: "eu-west-1");

var config = new AmazonDynamoDBConfig { ServiceURL = server.Endpoint };
var client = new AmazonDynamoDBClient(new BasicAWSCredentials("test", "test"), config);

await client.CreateTableAsync(new CreateTableRequest
{
    TableName = "orders",
    AttributeDefinitions = new() { new("id", ScalarAttributeType.S) },
    KeySchema = new() { new("id", KeyType.HASH) },
    BillingMode = BillingMode.PAY_PER_REQUEST,
});

await client.PutItemAsync("orders", new()
{
    ["id"]    = new AttributeValue("order-1"),
    ["total"] = new AttributeValue { N = "99.99" },
});
```

### Fixed port

```csharp
using var server = new CloudMockServer(port: 4566, region: "us-west-2");
```

## Configuration

| Parameter | Default      | Description                       |
|-----------|--------------|-----------------------------------|
| `port`    | random       | TCP port for the mock server      |
| `region`  | `us-east-1`  | AWS region reported by the server |
| `profile` | `minimal`    | CloudMock service profile         |

## License

BSL-1.1 — see [LICENSE](../../LICENSE).
