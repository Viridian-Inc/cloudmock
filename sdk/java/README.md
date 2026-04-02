# CloudMock Java SDK

Local AWS emulation for Java tests. 98 services.

## Maven

```xml
<dependency>
    <groupId>dev.cloudmock</groupId>
    <artifactId>cloudmock-sdk</artifactId>
    <version>1.0.0</version>
    <scope>test</scope>
</dependency>
```

## Usage

```java
import dev.cloudmock.CloudMock;
import software.amazon.awssdk.services.s3.S3Client;
import software.amazon.awssdk.regions.Region;
import software.amazon.awssdk.auth.credentials.*;

try (CloudMock cm = CloudMock.start()) {
    S3Client s3 = S3Client.builder()
        .endpointOverride(cm.endpoint())
        .region(Region.US_EAST_1)
        .credentialsProvider(StaticCredentialsProvider.create(
            AwsBasicCredentials.create("test", "test")))
        .forcePathStyle(true)
        .build();

    s3.createBucket(b -> b.bucket("my-bucket"));
    s3.putObject(
        b -> b.bucket("my-bucket").key("hello.txt"),
        software.amazon.awssdk.core.sync.RequestBody.fromString("world"));
}
```

### JUnit 5 Extension

```java
class MyTest {
    static CloudMock cm;

    @BeforeAll
    static void setup() throws Exception {
        cm = CloudMock.start();
    }

    @AfterAll
    static void teardown() {
        cm.close();
    }

    @Test
    void testDynamoDB() {
        var ddb = DynamoDbClient.builder()
            .endpointOverride(cm.endpoint())
            .region(Region.US_EAST_1)
            .credentialsProvider(StaticCredentialsProvider.create(
                AwsBasicCredentials.create("test", "test")))
            .build();
        // ...
    }
}
```
