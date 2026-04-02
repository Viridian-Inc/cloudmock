---
title: Swift / iOS
description: Using CloudMock with Swift and the AWS SDK for Swift
---

CloudMock does not require a custom Swift SDK. Instead, configure the AWS SDK for Swift (or raw URLSession calls) to point at your CloudMock gateway running on localhost. This works for iOS simulators, macOS apps, and server-side Swift.

## AWS SDK for Swift configuration

### Install the AWS SDK for Swift

Add the AWS SDK to your `Package.swift`:

```swift
dependencies: [
    .package(url: "https://github.com/awslabs/aws-sdk-swift.git", from: "1.0.0"),
],
targets: [
    .target(
        name: "MyApp",
        dependencies: [
            .product(name: "AWSS3", package: "aws-sdk-swift"),
            .product(name: "AWSDynamoDB", package: "aws-sdk-swift"),
        ]
    ),
]
```

### Configure the endpoint

Point each service client at CloudMock by setting a custom endpoint:

```swift
import AWSS3
import AWSDynamoDB
import AWSClientRuntime
import SmithyIdentity

// CloudMock endpoint -- use your Mac's IP when running on iOS Simulator
let cloudmockEndpoint = "http://localhost:4566"

// For iOS Simulator, use the host machine's IP instead of localhost:
// let cloudmockEndpoint = "http://192.168.1.100:4566"

let credentials = AWSCredentialIdentity(
    accessKey: "test",
    secret: "test"
)

let credentialResolver = try StaticAWSCredentialIdentityResolver(credentials)

// S3
let s3Config = try await S3Client.S3ClientConfiguration(
    awsCredentialIdentityResolver: credentialResolver,
    region: "us-east-1",
    endpoint: cloudmockEndpoint
)
s3Config.forcePathStyle = true
let s3 = S3Client(config: s3Config)

// Create a bucket
try await s3.createBucket(input: CreateBucketInput(bucket: "my-bucket"))

// DynamoDB
let ddbConfig = try await DynamoDBClient.DynamoDBClientConfiguration(
    awsCredentialIdentityResolver: credentialResolver,
    region: "us-east-1",
    endpoint: cloudmockEndpoint
)
let dynamodb = DynamoDBClient(config: ddbConfig)
```

### Conditional endpoint (debug vs. release)

```swift
func makeS3Client() async throws -> S3Client {
    #if DEBUG
    let config = try await S3Client.S3ClientConfiguration(
        awsCredentialIdentityResolver: try StaticAWSCredentialIdentityResolver(
            AWSCredentialIdentity(accessKey: "test", secret: "test")
        ),
        region: "us-east-1",
        endpoint: "http://localhost:4566"
    )
    config.forcePathStyle = true
    return S3Client(config: config)
    #else
    let config = try await S3Client.S3ClientConfiguration(region: "us-east-1")
    return S3Client(config: config)
    #endif
}
```

## Testing with XCTest

Use `XCTestCase` with `async/await` to test against CloudMock. Start CloudMock on your Mac before running the test suite.

```swift
import XCTest
import AWSS3
import AWSDynamoDB
import SmithyIdentity

final class CloudMockTests: XCTestCase {

    var s3: S3Client!
    var ddb: DynamoDBClient!

    override func setUp() async throws {
        let credentialResolver = try StaticAWSCredentialIdentityResolver(
            AWSCredentialIdentity(accessKey: "test", secret: "test")
        )

        let s3Config = try await S3Client.S3ClientConfiguration(
            awsCredentialIdentityResolver: credentialResolver,
            region: "us-east-1",
            endpoint: "http://localhost:4566"
        )
        s3Config.forcePathStyle = true
        s3 = S3Client(config: s3Config)

        let ddbConfig = try await DynamoDBClient.DynamoDBClientConfiguration(
            awsCredentialIdentityResolver: credentialResolver,
            region: "us-east-1",
            endpoint: "http://localhost:4566"
        )
        ddb = DynamoDBClient(config: ddbConfig)
    }

    func testCreateAndListBucket() async throws {
        try await s3.createBucket(input: CreateBucketInput(bucket: "test"))
        let output = try await s3.listBuckets(input: ListBucketsInput())
        let names = output.buckets?.compactMap { $0.name } ?? []
        XCTAssertTrue(names.contains("test"))
    }

    func testDynamoDBPutAndGet() async throws {
        let createInput = CreateTableInput(
            attributeDefinitions: [
                AttributeDefinition(attributeName: "pk", attributeType: .s)
            ],
            billingMode: .payPerRequest,
            keySchema: [
                KeySchemaElement(attributeName: "pk", keyType: .hash)
            ],
            tableName: "users"
        )
        _ = try await ddb.createTable(input: createInput)

        let putInput = PutItemInput(
            item: [
                "pk": AttributeValue.s("user-1"),
                "name": AttributeValue.s("Alice"),
            ],
            tableName: "users"
        )
        _ = try await ddb.putItem(input: putInput)

        let getInput = GetItemInput(
            key: ["pk": AttributeValue.s("user-1")],
            tableName: "users"
        )
        let getOutput = try await ddb.getItem(input: getInput)

        if case .s(let name) = getOutput.item?["name"] {
            XCTAssertEqual(name, "Alice")
        } else {
            XCTFail("Expected name attribute")
        }
    }
}
```

Start CloudMock before running:

```bash
npx cloudmock start &
xcodebuild test -scheme MyApp -destination 'platform=macOS'
```

## URLSession configuration

For direct HTTP calls to CloudMock (without the AWS SDK), use a standard URLSession:

```swift
let endpoint = URL(string: "http://localhost:4566")!

// List S3 buckets
var request = URLRequest(url: endpoint)
request.httpMethod = "GET"
request.setValue("us-east-1", forHTTPHeaderField: "X-Amz-Region")

let (data, response) = try await URLSession.shared.data(for: request)
```

## iOS Simulator networking

When running on the iOS Simulator, `localhost` refers to the simulator's loopback interface, which is the same as the host Mac. CloudMock running on the Mac is accessible at `http://localhost:4566` from the simulator without any special configuration.

For physical iOS devices on the same network, use the Mac's local IP address (e.g., `http://192.168.1.100:4566`).

### App Transport Security

During development, you may need to allow insecure HTTP connections in your `Info.plist`:

```xml
<key>NSAppTransportSecurity</key>
<dict>
    <key>NSAllowsLocalNetworking</key>
    <true/>
</dict>
```

The `NSAllowsLocalNetworking` key permits HTTP connections to localhost and `.local` domains without requiring HTTPS.

## Supported services

All 98 AWS services emulated by CloudMock are accessible from Swift. The most commonly used from iOS/macOS apps:

- **S3** -- File uploads, downloads, presigned URLs
- **DynamoDB** -- NoSQL database operations
- **Cognito** -- User authentication and sign-up flows
- **SNS** -- Push notification registration
- **SQS** -- Message queue operations
- **Lambda** -- Invoke serverless functions
- **API Gateway** -- REST API calls (though you may call your API server directly)

## Common issues

### S3 path style

Always set `forcePathStyle = true` on the S3 client configuration. CloudMock does not support virtual-hosted style S3 URLs.

### Simulator vs. device

The iOS Simulator shares the host Mac's network stack, so `localhost` works. Physical devices require the Mac's IP address on the local network.
