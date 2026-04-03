package com.example;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.context.annotation.Bean;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;
import software.amazon.awssdk.auth.credentials.AwsBasicCredentials;
import software.amazon.awssdk.auth.credentials.StaticCredentialsProvider;
import software.amazon.awssdk.regions.Region;
import software.amazon.awssdk.services.dynamodb.DynamoDbClient;
import software.amazon.awssdk.services.dynamodb.model.*;

import jakarta.annotation.PostConstruct;
import java.net.URI;
import java.util.*;

@SpringBootApplication
@RestController
@RequestMapping("/items")
public class App {

    private final DynamoDbClient dynamoDb;
    private final String tableName;

    public App(DynamoDbClient dynamoDb) {
        this.dynamoDb = dynamoDb;
        this.tableName = System.getenv().getOrDefault("TABLE_NAME", "items");
    }

    @PostConstruct
    public void createTableIfNeeded() {
        try {
            dynamoDb.createTable(CreateTableRequest.builder()
                .tableName(tableName)
                .keySchema(KeySchemaElement.builder()
                    .attributeName("id").keyType(KeyType.HASH).build())
                .attributeDefinitions(AttributeDefinition.builder()
                    .attributeName("id").attributeType(ScalarAttributeType.S).build())
                .billingMode(BillingMode.PAY_PER_REQUEST)
                .build());
        } catch (ResourceInUseException e) {
            // Table already exists — ignore
        }
    }

    @PostMapping
    public ResponseEntity<Map<String, Object>> createItem(@RequestBody Map<String, Object> item) {
        if (!item.containsKey("id")) {
            return ResponseEntity.badRequest().body(Map.of("error", "id is required"));
        }
        Map<String, AttributeValue> av = new HashMap<>();
        item.forEach((k, v) -> av.put(k, AttributeValue.builder().s(String.valueOf(v)).build()));
        dynamoDb.putItem(PutItemRequest.builder().tableName(tableName).item(av).build());
        return ResponseEntity.status(HttpStatus.CREATED).body(item);
    }

    @GetMapping("/{id}")
    public ResponseEntity<Map<String, String>> getItem(@PathVariable String id) {
        GetItemResponse result = dynamoDb.getItem(GetItemRequest.builder()
            .tableName(tableName)
            .key(Map.of("id", AttributeValue.builder().s(id).build()))
            .build());
        if (!result.hasItem()) {
            return ResponseEntity.notFound().build();
        }
        Map<String, String> item = new HashMap<>();
        result.item().forEach((k, v) -> item.put(k, v.s()));
        return ResponseEntity.ok(item);
    }

    @GetMapping
    public List<Map<String, String>> listItems() {
        ScanResponse result = dynamoDb.scan(ScanRequest.builder().tableName(tableName).build());
        List<Map<String, String>> items = new ArrayList<>();
        for (Map<String, AttributeValue> raw : result.items()) {
            Map<String, String> item = new HashMap<>();
            raw.forEach((k, v) -> item.put(k, v.s()));
            items.add(item);
        }
        return items;
    }

    @DeleteMapping("/{id}")
    public ResponseEntity<Void> deleteItem(@PathVariable String id) {
        dynamoDb.deleteItem(DeleteItemRequest.builder()
            .tableName(tableName)
            .key(Map.of("id", AttributeValue.builder().s(id).build()))
            .build());
        return ResponseEntity.noContent().build();
    }

    public static void main(String[] args) {
        SpringApplication.run(App.class, args);
    }

    @Bean
    public DynamoDbClient dynamoDbClient() {
        String endpoint = System.getenv().getOrDefault("AWS_ENDPOINT_URL", "http://localhost:4566");
        String region = System.getenv().getOrDefault("AWS_REGION", "us-east-1");
        String accessKey = System.getenv().getOrDefault("AWS_ACCESS_KEY_ID", "test");
        String secretKey = System.getenv().getOrDefault("AWS_SECRET_ACCESS_KEY", "test");
        return DynamoDbClient.builder()
            .endpointOverride(URI.create(endpoint))
            .region(Region.of(region))
            .credentialsProvider(StaticCredentialsProvider.create(
                AwsBasicCredentials.create(accessKey, secretKey)))
            .build();
    }
}
