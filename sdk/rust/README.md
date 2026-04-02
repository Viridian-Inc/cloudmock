# cloudmock

Local AWS emulation for Rust. 98 services.

## Usage

```rust
use cloudmock::CloudMock;

#[tokio::test]
async fn test_dynamodb() {
    let cm = CloudMock::start().await.unwrap();
    
    let config = aws_config::defaults(aws_config::BehaviorVersion::latest())
        .endpoint_url(cm.endpoint())
        .credentials_provider(aws_credential_types::Credentials::new(
            "test", "test", None, None, "cloudmock"
        ))
        .region(aws_config::Region::new("us-east-1"))
        .load()
        .await;
    
    let ddb = aws_sdk_dynamodb::Client::new(&config);
    
    ddb.create_table()
        .table_name("users")
        .key_schema(aws_sdk_dynamodb::types::KeySchemaElement::builder()
            .attribute_name("pk")
            .key_type(aws_sdk_dynamodb::types::KeyType::Hash)
            .build().unwrap())
        .attribute_definitions(aws_sdk_dynamodb::types::AttributeDefinition::builder()
            .attribute_name("pk")
            .attribute_type(aws_sdk_dynamodb::types::ScalarAttributeType::S)
            .build().unwrap())
        .billing_mode(aws_sdk_dynamodb::types::BillingMode::PayPerRequest)
        .send()
        .await
        .unwrap();
    
    cm.stop().await;
}
```
