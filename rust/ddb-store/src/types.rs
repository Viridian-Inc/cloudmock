use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::collections::HashMap;

/// DynamoDB AttributeValue in wire format: {"S": "hello"}, {"N": "42"}, etc.
pub type AttributeValue = Value;

/// A DynamoDB item: map of attribute name to typed value.
pub type Item = HashMap<String, AttributeValue>;

/// Key schema element.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct KeySchemaElement {
    #[serde(rename = "AttributeName")]
    pub attribute_name: String,
    #[serde(rename = "KeyType")]
    pub key_type: String, // HASH or RANGE
}

/// Attribute definition.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AttributeDefinition {
    #[serde(rename = "AttributeName")]
    pub attribute_name: String,
    #[serde(rename = "AttributeType")]
    pub attribute_type: String, // S, N, or B
}

// ── Request types ──

#[derive(Debug, Deserialize)]
pub struct CreateTableRequest {
    #[serde(rename = "TableName")]
    pub table_name: String,
    #[serde(rename = "KeySchema")]
    pub key_schema: Vec<KeySchemaElement>,
    #[serde(rename = "AttributeDefinitions")]
    pub attribute_definitions: Vec<AttributeDefinition>,
    #[serde(rename = "BillingMode", default)]
    pub billing_mode: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct PutItemRequest {
    #[serde(rename = "TableName")]
    pub table_name: String,
    #[serde(rename = "Item")]
    pub item: Item,
}

#[derive(Debug, Deserialize)]
pub struct GetItemRequest {
    #[serde(rename = "TableName")]
    pub table_name: String,
    #[serde(rename = "Key")]
    pub key: Item,
}

#[derive(Debug, Deserialize)]
pub struct DeleteItemRequest {
    #[serde(rename = "TableName")]
    pub table_name: String,
    #[serde(rename = "Key")]
    pub key: Item,
}

#[derive(Debug, Deserialize)]
pub struct ListTablesRequest {}

#[derive(Debug, Deserialize)]
pub struct DeleteTableRequest {
    #[serde(rename = "TableName")]
    pub table_name: String,
}

// ── Response types ──

#[derive(Debug, Serialize)]
pub struct ListTablesResponse {
    #[serde(rename = "TableNames")]
    pub table_names: Vec<String>,
}

#[derive(Debug, Serialize)]
pub struct GetItemResponse {
    #[serde(rename = "Item", skip_serializing_if = "Option::is_none")]
    pub item: Option<Item>,
}

#[derive(Debug, Serialize)]
pub struct CreateTableResponse {
    #[serde(rename = "TableDescription")]
    pub table_description: TableDescription,
}

#[derive(Debug, Serialize)]
pub struct DeleteTableResponse {
    #[serde(rename = "TableDescription")]
    pub table_description: TableDescription,
}

#[derive(Debug, Serialize)]
pub struct TableDescription {
    #[serde(rename = "TableName")]
    pub table_name: String,
    #[serde(rename = "TableStatus")]
    pub table_status: String,
    #[serde(rename = "KeySchema")]
    pub key_schema: Vec<KeySchemaElement>,
    #[serde(rename = "AttributeDefinitions")]
    pub attribute_definitions: Vec<AttributeDefinition>,
    #[serde(rename = "ItemCount")]
    pub item_count: i64,
    #[serde(rename = "TableArn")]
    pub table_arn: String,
    #[serde(rename = "BillingModeSummary")]
    pub billing_mode_summary: BillingModeSummary,
}

#[derive(Debug, Serialize)]
pub struct BillingModeSummary {
    #[serde(rename = "BillingMode")]
    pub billing_mode: String,
}

// ── Error ──

#[derive(Debug, Serialize)]
pub struct DdbError {
    #[serde(rename = "__type")]
    pub error_type: String,
    #[serde(rename = "Message")]
    pub message: String,
}
