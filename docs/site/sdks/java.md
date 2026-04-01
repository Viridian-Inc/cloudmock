# Java Integration

Java works with CloudMock through the AWS SDK for Java v2 and the OpenTelemetry Java Agent. No CloudMock-specific SDK is needed.

## AWS SDK for Java v2

### Maven

```xml
<dependencyManagement>
  <dependencies>
    <dependency>
      <groupId>software.amazon.awssdk</groupId>
      <artifactId>bom</artifactId>
      <version>2.25.0</version>
      <type>pom</type>
      <scope>import</scope>
    </dependency>
  </dependencies>
</dependencyManagement>

<dependencies>
  <dependency>
    <groupId>software.amazon.awssdk</groupId>
    <artifactId>s3</artifactId>
  </dependency>
  <dependency>
    <groupId>software.amazon.awssdk</groupId>
    <artifactId>dynamodb</artifactId>
  </dependency>
  <dependency>
    <groupId>software.amazon.awssdk</groupId>
    <artifactId>dynamodb-enhanced</artifactId>
  </dependency>
</dependencies>
```

### Configuration

Point the SDK at CloudMock using the endpoint override:

```java
import software.amazon.awssdk.auth.credentials.StaticCredentialsProvider;
import software.amazon.awssdk.auth.credentials.AwsBasicCredentials;
import software.amazon.awssdk.regions.Region;
import java.net.URI;

// Shared configuration for all clients
StaticCredentialsProvider credentials = StaticCredentialsProvider.create(
    AwsBasicCredentials.create("test", "test")
);
URI endpoint = URI.create("http://localhost:4566");
Region region = Region.US_EAST_1;
```

Or set the environment variable (SDK v2.20.162+):

```bash
export AWS_ENDPOINT_URL=http://localhost:4566
```

### Examples

**S3:**

```java
import software.amazon.awssdk.services.s3.S3Client;
import software.amazon.awssdk.services.s3.model.*;
import software.amazon.awssdk.core.sync.RequestBody;

S3Client s3 = S3Client.builder()
    .region(region)
    .endpointOverride(endpoint)
    .credentialsProvider(credentials)
    .build();

// Create bucket
s3.createBucket(CreateBucketRequest.builder().bucket("my-bucket").build());

// Upload
s3.putObject(
    PutObjectRequest.builder().bucket("my-bucket").key("hello.txt").build(),
    RequestBody.fromString("Hello from Java!")
);

// Download
ResponseInputStream<GetObjectResponse> response = s3.getObject(
    GetObjectRequest.builder().bucket("my-bucket").key("hello.txt").build()
);
String content = new String(response.readAllBytes());
```

**DynamoDB:**

```java
import software.amazon.awssdk.services.dynamodb.DynamoDbClient;
import software.amazon.awssdk.services.dynamodb.model.*;

DynamoDbClient dynamodb = DynamoDbClient.builder()
    .region(region)
    .endpointOverride(endpoint)
    .credentialsProvider(credentials)
    .build();

// Create table
dynamodb.createTable(CreateTableRequest.builder()
    .tableName("users")
    .keySchema(KeySchemaElement.builder()
        .attributeName("userId").keyType(KeyType.HASH).build())
    .attributeDefinitions(AttributeDefinition.builder()
        .attributeName("userId").attributeType(ScalarAttributeType.S).build())
    .billingMode(BillingMode.PAY_PER_REQUEST)
    .build());

// Put item
dynamodb.putItem(PutItemRequest.builder()
    .tableName("users")
    .item(Map.of(
        "userId", AttributeValue.builder().s("user-1").build(),
        "name", AttributeValue.builder().s("Alice").build(),
        "email", AttributeValue.builder().s("alice@example.com").build()
    ))
    .build());

// Get item
GetItemResponse response = dynamodb.getItem(GetItemRequest.builder()
    .tableName("users")
    .key(Map.of("userId", AttributeValue.builder().s("user-1").build()))
    .build());
```

**DynamoDB Enhanced Client:**

```java
import software.amazon.awssdk.enhanced.dynamodb.*;
import software.amazon.awssdk.enhanced.dynamodb.mapper.annotations.*;

@DynamoDbBean
public class User {
    private String userId;
    private String name;
    private String email;

    @DynamoDbPartitionKey
    public String getUserId() { return userId; }
    public void setUserId(String userId) { this.userId = userId; }
    // ... getters/setters
}

DynamoDbEnhancedClient enhanced = DynamoDbEnhancedClient.builder()
    .dynamoDbClient(dynamodb)
    .build();

DynamoDbTable<User> table = enhanced.table("users", TableSchema.fromBean(User.class));

User user = new User();
user.setUserId("user-1");
user.setName("Alice");
user.setEmail("alice@example.com");
table.putItem(user);

User retrieved = table.getItem(Key.builder().partitionValue("user-1").build());
```

**SQS:**

```java
import software.amazon.awssdk.services.sqs.SqsClient;
import software.amazon.awssdk.services.sqs.model.*;

SqsClient sqs = SqsClient.builder()
    .region(region)
    .endpointOverride(endpoint)
    .credentialsProvider(credentials)
    .build();

// Create queue
String queueUrl = sqs.createQueue(CreateQueueRequest.builder()
    .queueName("order-processing").build())
    .queueUrl();

// Send message
sqs.sendMessage(SendMessageRequest.builder()
    .queueUrl(queueUrl)
    .messageBody("{\"orderId\": \"order-123\"}")
    .build());

// Receive
ReceiveMessageResponse response = sqs.receiveMessage(ReceiveMessageRequest.builder()
    .queueUrl(queueUrl)
    .maxNumberOfMessages(10)
    .waitTimeSeconds(5)
    .build());
```

## OpenTelemetry

### Java Agent (Zero-Code)

The easiest way to add tracing. No code changes required.

```bash
# Download the agent
curl -L -o opentelemetry-javaagent.jar \
  https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/latest/download/opentelemetry-javaagent.jar
```

Run your application with the agent:

```bash
java -javaagent:opentelemetry-javaagent.jar \
  -Dotel.exporter.otlp.endpoint=http://localhost:4318 \
  -Dotel.exporter.otlp.protocol=http/protobuf \
  -Dotel.service.name=my-java-service \
  -jar my-app.jar
```

The agent auto-instruments:
- Spring Boot (Web, WebFlux, Data)
- JDBC (PostgreSQL, MySQL, Oracle)
- AWS SDK v1 and v2
- HTTP clients (Apache, OkHttp, Java 11+ HttpClient)
- gRPC
- Kafka, RabbitMQ
- Redis (Jedis, Lettuce)
- 100+ more libraries

### Spring Boot Programmatic Setup

```xml
<dependency>
  <groupId>io.opentelemetry</groupId>
  <artifactId>opentelemetry-api</artifactId>
</dependency>
<dependency>
  <groupId>io.opentelemetry</groupId>
  <artifactId>opentelemetry-sdk</artifactId>
</dependency>
<dependency>
  <groupId>io.opentelemetry</groupId>
  <artifactId>opentelemetry-exporter-otlp</artifactId>
</dependency>
```

```java
import io.opentelemetry.api.OpenTelemetry;
import io.opentelemetry.api.trace.Tracer;
import io.opentelemetry.api.trace.Span;
import io.opentelemetry.sdk.OpenTelemetrySdk;
import io.opentelemetry.sdk.trace.SdkTracerProvider;
import io.opentelemetry.sdk.trace.export.BatchSpanProcessor;
import io.opentelemetry.exporter.otlp.http.trace.OtlpHttpSpanExporter;

@Configuration
public class TracingConfig {

    @Bean
    public OpenTelemetry openTelemetry() {
        OtlpHttpSpanExporter exporter = OtlpHttpSpanExporter.builder()
            .setEndpoint("http://localhost:4318/v1/traces")
            .build();

        SdkTracerProvider tracerProvider = SdkTracerProvider.builder()
            .addSpanProcessor(BatchSpanProcessor.builder(exporter).build())
            .setResource(Resource.create(Attributes.of(
                ResourceAttributes.SERVICE_NAME, "order-api"
            )))
            .build();

        return OpenTelemetrySdk.builder()
            .setTracerProvider(tracerProvider)
            .buildAndRegisterGlobal();
    }
}
```

### Custom Spans

```java
import io.opentelemetry.api.GlobalOpenTelemetry;
import io.opentelemetry.api.trace.Tracer;
import io.opentelemetry.api.trace.Span;
import io.opentelemetry.api.trace.StatusCode;

@Service
public class OrderService {
    private final Tracer tracer = GlobalOpenTelemetry.getTracer("order-service");

    public Order processOrder(OrderRequest request) {
        Span span = tracer.spanBuilder("process-order")
            .setAttribute("order.id", request.getId())
            .setAttribute("order.total", request.getTotal())
            .startSpan();

        try (var scope = span.makeCurrent()) {
            Payment payment = paymentService.charge(request);
            span.setAttribute("payment.id", payment.getId());

            notificationService.sendConfirmation(request);
            return new Order(request.getId(), "completed");
        } catch (Exception e) {
            span.recordException(e);
            span.setStatus(StatusCode.ERROR, e.getMessage());
            throw e;
        } finally {
            span.end();
        }
    }
}
```

## Testing

Use CloudMock in integration tests:

```java
@SpringBootTest
@TestPropertySource(properties = {
    "aws.endpoint=http://localhost:4566",
    "aws.region=us-east-1"
})
class OrderServiceTest {

    @BeforeAll
    static void setup() {
        // Ensure CloudMock is running: cmk start
    }

    @AfterEach
    void cleanup() throws Exception {
        // Reset CloudMock state
        HttpClient.newHttpClient().send(
            HttpRequest.newBuilder()
                .uri(URI.create("http://localhost:4599/api/reset"))
                .POST(HttpRequest.BodyPublishers.noBody())
                .build(),
            HttpResponse.BodyHandlers.discarding()
        );
    }

    @Test
    void shouldCreateOrder() {
        // Test against CloudMock -- real AWS SDK calls, no mocks
        Order order = orderService.create(new OrderRequest("order-1", "Alice", 99.99));
        assertThat(order.getStatus()).isEqualTo("pending");
    }
}
```
