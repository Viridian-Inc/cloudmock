# CloudMock Swift SDK

Start a local AWS mock server in your Swift tests using `CloudMockServer`, or initialize mobile telemetry with `CloudMock`.

## Installation

Add the package to your `Package.swift`:

```swift
dependencies: [
    .package(url: "https://github.com/Viridian-Inc/cloudmock", from: "1.0.0"),
],
targets: [
    .testTarget(name: "MyAppTests", dependencies: [
        .product(name: "CloudMock", package: "cloudmock"),
        "SotoS3", "SotoCore",
    ]),
]
```

Or via Xcode: **File > Add Package Dependencies** and enter the repository URL.

## Usage with Soto (AWS SDK for Swift)

```swift
import XCTest
import SotoCore
import SotoS3
import SotoDynamoDB
import CloudMock

final class S3Tests: XCTestCase {
    var server: CloudMockServer!
    var client: AWSClient!

    override func setUp() async throws {
        server = CloudMockServer()
        try server.start()

        client = AWSClient(
            credentialProvider: .static(accessKeyId: "test", secretAccessKey: "test"),
            httpClientProvider: .createNew
        )
    }

    override func tearDown() async throws {
        try await client.shutdown()
        server.stop()
    }

    func testCreateBucket() async throws {
        let s3 = S3(client: client, endpoint: server.endpoint, region: .useast1)
        _ = try await s3.createBucket(.init(bucket: "my-test-bucket"))

        let resp = try await s3.listBuckets(.init())
        let names = resp.buckets?.compactMap(\.name) ?? []
        XCTAssertTrue(names.contains("my-test-bucket"))
    }

    func testDynamoDB() async throws {
        let dynamo = DynamoDB(client: client, endpoint: server.endpoint, region: .useast1)

        _ = try await dynamo.createTable(.init(
            attributeDefinitions: [.init(attributeName: "id", attributeType: .s)],
            billingMode: .payPerRequest,
            keySchema: [.init(attributeName: "id", keyType: .hash)],
            tableName: "items"
        ))

        _ = try await dynamo.putItem(.init(
            item: ["id": .s("item-1"), "value": .s("hello")],
            tableName: "items"
        ))

        let got = try await dynamo.getItem(.init(
            key: ["id": .s("item-1")],
            tableName: "items"
        ))
        XCTAssertEqual(got.item?["value"], .s("hello"))
    }
}
```

### Fixed port

```swift
let server = CloudMockServer(port: 4566)
try server.start()
defer { server.stop() }
```

## Configuration

| Parameter | Default   | Description                        |
|-----------|-----------|------------------------------------|
| `port`    | random    | TCP port for the mock server       |
| `profile` | `minimal` | CloudMock service profile to load  |

## Mobile RUM / Telemetry

For iOS/macOS telemetry (OTel, RUM, BLE mesh), use `CloudMock.initialize(config:)` — see the main SDK sources.

## License

BSL-1.1 — see [LICENSE](../../LICENSE).
