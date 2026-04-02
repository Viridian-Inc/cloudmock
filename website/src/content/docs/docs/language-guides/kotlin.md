---
title: Kotlin / Android
description: Using CloudMock with Kotlin, the AWS SDK for Kotlin, and OkHttp
---

CloudMock does not require a custom Kotlin SDK. Configure the AWS SDK for Kotlin (or OkHttp) to point at your CloudMock gateway. This works for Android apps, server-side Kotlin, and Kotlin Multiplatform projects.

## AWS SDK for Kotlin configuration

### Add dependencies

In your `build.gradle.kts`:

```kotlin
dependencies {
    implementation("aws.sdk.kotlin:s3:1.3.+")
    implementation("aws.sdk.kotlin:dynamodb:1.3.+")
    implementation("aws.sdk.kotlin:sqs:1.3.+")
}
```

### Configure the endpoint

Point each service client at CloudMock by setting the endpoint URL:

```kotlin
import aws.sdk.kotlin.services.s3.S3Client
import aws.sdk.kotlin.services.s3.model.CreateBucketRequest
import aws.sdk.kotlin.services.dynamodb.DynamoDbClient
import aws.sdk.kotlin.services.dynamodb.model.*
import aws.smithy.kotlin.runtime.auth.awscredentials.Credentials
import aws.smithy.kotlin.runtime.auth.awscredentials.CredentialsProvider
import aws.smithy.kotlin.runtime.net.url.Url

// Static credentials for CloudMock
class CloudMockCredentials : CredentialsProvider {
    override suspend fun resolve(attributes: aws.smithy.kotlin.runtime.collections.Attributes): Credentials {
        return Credentials("test", "test")
    }
}

val cloudmockEndpoint = Url.parse("http://10.0.2.2:4566") // Android emulator
// For local JVM: Url.parse("http://localhost:4566")

// S3
val s3 = S3Client {
    region = "us-east-1"
    endpointUrl = cloudmockEndpoint
    credentialsProvider = CloudMockCredentials()
    forcePathStyle = true
}

suspend fun createBucket() {
    s3.createBucket(CreateBucketRequest { bucket = "my-bucket" })
}

// DynamoDB
val dynamodb = DynamoDbClient {
    region = "us-east-1"
    endpointUrl = cloudmockEndpoint
    credentialsProvider = CloudMockCredentials()
}

suspend fun createTable() {
    dynamodb.createTable(CreateTableInput {
        tableName = "Users"
        keySchema = listOf(
            KeySchemaElement {
                attributeName = "UserId"
                keyType = KeyType.Hash
            }
        )
        attributeDefinitions = listOf(
            AttributeDefinition {
                attributeName = "UserId"
                attributeType = ScalarAttributeType.S
            }
        )
        billingMode = BillingMode.PayPerRequest
    })
}
```

### Conditional endpoint (debug vs. release)

```kotlin
object AwsConfig {
    val endpoint: Url? = if (BuildConfig.DEBUG) {
        Url.parse("http://10.0.2.2:4566")
    } else {
        null // Use default AWS endpoints
    }

    val credentials: CredentialsProvider? = if (BuildConfig.DEBUG) {
        CloudMockCredentials()
    } else {
        null // Use default credential chain
    }
}

val s3 = S3Client {
    region = "us-east-1"
    AwsConfig.endpoint?.let { endpointUrl = it }
    AwsConfig.credentials?.let { credentialsProvider = it }
    forcePathStyle = true
}
```

## Testing with JUnit 5 (server-side Kotlin)

For server-side Kotlin or Kotlin Multiplatform, use a JUnit 5 test class with `runBlocking` (or `runTest` from `kotlinx-coroutines-test`):

```kotlin
import aws.sdk.kotlin.services.dynamodb.DynamoDbClient
import aws.sdk.kotlin.services.dynamodb.model.*
import aws.sdk.kotlin.services.s3.S3Client
import aws.sdk.kotlin.services.s3.model.*
import aws.smithy.kotlin.runtime.net.url.Url
import kotlinx.coroutines.runBlocking
import org.junit.jupiter.api.*

@TestInstance(TestInstance.Lifecycle.PER_CLASS)
class CloudMockKotlinTest {

    private val endpoint = Url.parse("http://localhost:4566")
    private val credentials = CloudMockCredentials()

    private val s3 = S3Client {
        region = "us-east-1"
        endpointUrl = endpoint
        credentialsProvider = credentials
        forcePathStyle = true
    }

    private val ddb = DynamoDbClient {
        region = "us-east-1"
        endpointUrl = endpoint
        credentialsProvider = credentials
    }

    @AfterAll
    fun teardown() {
        s3.close()
        ddb.close()
    }

    @Test
    fun `create and list S3 bucket`() = runBlocking {
        s3.createBucket(CreateBucketRequest { bucket = "test" })
        val buckets = s3.listBuckets(ListBucketsRequest {}).buckets ?: emptyList()
        Assertions.assertTrue(buckets.any { it.name == "test" })
    }

    @Test
    fun `DynamoDB put and get item`() = runBlocking {
        ddb.createTable(CreateTableRequest {
            tableName = "users"
            keySchema = listOf(KeySchemaElement {
                attributeName = "pk"
                keyType = KeyType.Hash
            })
            attributeDefinitions = listOf(AttributeDefinition {
                attributeName = "pk"
                attributeType = ScalarAttributeType.S
            })
            billingMode = BillingMode.PayPerRequest
        })

        ddb.putItem(PutItemRequest {
            tableName = "users"
            item = mapOf(
                "pk" to AttributeValue.S("user-1"),
                "name" to AttributeValue.S("Alice"),
            )
        })

        val resp = ddb.getItem(GetItemRequest {
            tableName = "users"
            key = mapOf("pk" to AttributeValue.S("user-1"))
        })

        Assertions.assertEquals("Alice", (resp.item?.get("name") as? AttributeValue.S)?.value)
    }
}
```

Start CloudMock before running tests:

```bash
npx cloudmock start &
./gradlew test
```

## OkHttp endpoint override

For applications that call AWS APIs directly via HTTP (without the AWS SDK), configure OkHttp to route requests to CloudMock:

```kotlin
import okhttp3.OkHttpClient
import okhttp3.Request

val client = OkHttpClient()

// List S3 buckets via the CloudMock gateway
val request = Request.Builder()
    .url("http://10.0.2.2:4566/")
    .header("Authorization", "AWS4-HMAC-SHA256 ...")
    .build()

val response = client.newCall(request).execute()
```

For most use cases, the AWS SDK for Kotlin handles request signing and serialization automatically. Direct OkHttp usage is only needed if you are testing raw HTTP behavior.

## Android emulator networking

The Android emulator uses a virtual network. `localhost` inside the emulator refers to the emulator itself, not the host machine. To reach CloudMock running on the host:

| Environment | CloudMock URL |
|-------------|---------------|
| Android Emulator | `http://10.0.2.2:4566` |
| Genymotion | `http://10.0.3.2:4566` |
| Physical device (same Wi-Fi) | `http://<host-ip>:4566` |
| Local JVM (tests, server) | `http://localhost:4566` |

### Network security config

Android 9+ blocks cleartext (non-HTTPS) traffic by default. Add a network security config to allow HTTP connections to CloudMock during development.

Create `res/xml/network_security_config.xml`:

```xml
<?xml version="1.0" encoding="utf-8"?>
<network-security-config>
    <domain-config cleartextTrafficPermitted="true">
        <domain includeSubdomains="true">10.0.2.2</domain>
        <domain includeSubdomains="true">localhost</domain>
    </domain-config>
</network-security-config>
```

Reference it in `AndroidManifest.xml`:

```xml
<application
    android:networkSecurityConfig="@xml/network_security_config"
    ... >
```

## Server-side Kotlin (Ktor, Spring Boot)

For server-side Kotlin applications, use `localhost:4566` directly:

```kotlin
// Ktor application
val s3 = S3Client {
    region = "us-east-1"
    endpointUrl = Url.parse("http://localhost:4566")
    credentialsProvider = CloudMockCredentials()
    forcePathStyle = true
}
```

No special networking configuration is needed for server-side Kotlin.

## Common issues

### S3 path style

Always set `forcePathStyle = true` on the S3 client. CloudMock does not support virtual-hosted style S3 URLs.

### Emulator IP address

Remember to use `10.0.2.2` (not `localhost`) when running inside the Android emulator. This is the standard alias for the host machine's loopback interface.

### Coroutine scope

The AWS SDK for Kotlin uses suspend functions. Make sure you call AWS operations from a coroutine scope (e.g., `viewModelScope`, `lifecycleScope`, or `runBlocking` in tests).
