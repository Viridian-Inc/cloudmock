# CloudMock Kotlin SDK

Start a local AWS mock server and get pre-configured clients in your Kotlin/JVM tests.

## Installation

### Gradle (Kotlin DSL)

```kotlin
dependencies {
    testImplementation("dev.cloudmock:cloudmock:1.0.0")
    testImplementation("aws.sdk.kotlin:s3:1.0.0")
    testImplementation("aws.sdk.kotlin:dynamodb:1.0.0")
}
```

### Maven

```xml
<dependency>
    <groupId>dev.cloudmock</groupId>
    <artifactId>cloudmock</artifactId>
    <version>1.0.0</version>
    <scope>test</scope>
</dependency>
```

## Usage

`CloudMock` implements `AutoCloseable`, so it works cleanly with `use {}` blocks and JUnit lifecycle hooks.

### JUnit 5 — per-class lifecycle

```kotlin
package com.example

import aws.sdk.kotlin.runtime.auth.credentials.StaticCredentialsProvider
import aws.sdk.kotlin.services.s3.S3Client
import aws.sdk.kotlin.services.s3.createBucket
import aws.sdk.kotlin.services.s3.listBuckets
import dev.cloudmock.CloudMock
import org.junit.jupiter.api.*
import org.junit.jupiter.api.Assertions.*

@TestInstance(TestInstance.Lifecycle.PER_CLASS)
class S3Tests {
    private val mock = CloudMock.start()

    @AfterAll
    fun tearDown() = mock.close()

    @Test
    fun `create and list bucket`() = runTest {
        S3Client {
            endpointUrl = mock.endpoint.toURL()
            region = mock.region
            credentialsProvider = StaticCredentialsProvider {
                accessKeyId = "test"
                secretAccessKey = "test"
            }
            forcePathStyle = true
        }.use { s3 ->
            s3.createBucket { bucket = "my-test-bucket" }
            val names = s3.listBuckets {}.buckets?.mapNotNull { it.name } ?: emptyList()
            assertTrue(names.contains("my-test-bucket"))
        }
    }
}
```

### DynamoDB example

```kotlin
import aws.sdk.kotlin.services.dynamodb.DynamoDbClient
import aws.sdk.kotlin.services.dynamodb.model.*
import dev.cloudmock.CloudMock

CloudMock.start(region = "eu-west-1").use { mock ->
    DynamoDbClient {
        endpointUrl = mock.endpoint.toURL()
        region = mock.region
    }.use { dynamo ->
        dynamo.createTable {
            tableName = "orders"
            attributeDefinitions = listOf(AttributeDefinition {
                attributeName = "id"; attributeType = ScalarAttributeType.S
            })
            keySchema = listOf(KeySchemaElement {
                attributeName = "id"; keyType = KeyType.Hash
            })
            billingMode = BillingMode.PayPerRequest
        }

        dynamo.putItem {
            tableName = "orders"
            item = mapOf(
                "id"    to AttributeValue.S("order-1"),
                "total" to AttributeValue.N("99.99")
            )
        }
    }
}
```

### Fixed port

```kotlin
val mock = CloudMock(port = 4566, region = "us-west-2").start()
```

## Configuration

| Parameter | Default      | Description                        |
|-----------|--------------|------------------------------------|
| `port`    | random       | TCP port for the mock server       |
| `region`  | `us-east-1`  | AWS region reported by the server  |
| `profile` | `minimal`    | CloudMock service profile to load  |

## Android / Mobile RUM

For Android telemetry (OTel, RUM, BLE mesh), use `io.cloudmock.CloudMock.initialize()` — see the Android SDK sources.

## License

BSL-1.1 — see [LICENSE](../../LICENSE).
