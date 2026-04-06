use dashmap::DashMap;
use parking_lot::RwLock;
use std::collections::HashMap;
use std::sync::atomic::{AtomicI64, Ordering};

use crate::types::*;

/// Extract a comparable string key from an AttributeValue.
fn attr_key(av: &AttributeValue) -> String {
    match av {
        serde_json::Value::Object(map) => {
            if let Some(v) = map.get("S") {
                return format!("S:{}", v.as_str().unwrap_or(""));
            }
            if let Some(v) = map.get("N") {
                return format!("N:{}", v.as_str().unwrap_or(""));
            }
            if let Some(v) = map.get("B") {
                return format!("B:{}", v.as_str().unwrap_or(""));
            }
            String::new()
        }
        _ => String::new(),
    }
}

/// A single partition: items sharing the same hash key value.
struct Partition {
    /// For hash-only tables: single item.
    /// For hash+range tables: map of range_key_string → Item.
    items: HashMap<String, Item>,
    range_key_name: Option<String>,
}

impl Partition {
    fn new(range_key_name: Option<String>) -> Self {
        Self {
            items: HashMap::new(),
            range_key_name,
        }
    }

    fn put(&mut self, item: Item) -> Option<Item> {
        let key = match &self.range_key_name {
            Some(rk) => item.get(rk).map(attr_key).unwrap_or_default(),
            None => String::new(),
        };
        self.items.insert(key, item)
    }

    fn get(&self, key_item: &Item) -> Option<&Item> {
        let key = match &self.range_key_name {
            Some(rk) => key_item.get(rk).map(attr_key).unwrap_or_default(),
            None => String::new(),
        };
        self.items.get(&key)
    }

    fn delete(&mut self, key_item: &Item) -> Option<Item> {
        let key = match &self.range_key_name {
            Some(rk) => key_item.get(rk).map(attr_key).unwrap_or_default(),
            None => String::new(),
        };
        self.items.remove(&key)
    }

    fn len(&self) -> usize {
        self.items.len()
    }

    fn all_items(&self) -> Vec<&Item> {
        self.items.values().collect()
    }
}

/// A DynamoDB table with partitioned storage.
struct Table {
    name: String,
    key_schema: Vec<KeySchemaElement>,
    attribute_definitions: Vec<AttributeDefinition>,
    billing_mode: String,
    hash_key_name: String,
    range_key_name: Option<String>,
    partitions: DashMap<String, RwLock<Partition>>,
    item_count: AtomicI64,
    arn: String,
}

impl Table {
    fn new(
        name: String,
        key_schema: Vec<KeySchemaElement>,
        attribute_definitions: Vec<AttributeDefinition>,
        billing_mode: String,
        arn: String,
    ) -> Self {
        let hash_key_name = key_schema
            .iter()
            .find(|k| k.key_type == "HASH")
            .map(|k| k.attribute_name.clone())
            .unwrap_or_default();
        let range_key_name = key_schema
            .iter()
            .find(|k| k.key_type == "RANGE")
            .map(|k| k.attribute_name.clone());

        Self {
            name,
            key_schema,
            attribute_definitions,
            billing_mode,
            hash_key_name,
            range_key_name,
            partitions: DashMap::new(),
            item_count: AtomicI64::new(0),
            arn,
        }
    }

    fn put_item(&self, item: Item) -> Option<Item> {
        let pk_val = item
            .get(&self.hash_key_name)
            .map(attr_key)
            .unwrap_or_default();

        let entry = self
            .partitions
            .entry(pk_val)
            .or_insert_with(|| RwLock::new(Partition::new(self.range_key_name.clone())));

        let mut partition = entry.write();
        let old = partition.put(item);
        if old.is_none() {
            self.item_count.fetch_add(1, Ordering::Relaxed);
        }
        old
    }

    fn get_item(&self, key: &Item) -> Option<Item> {
        let pk_val = key
            .get(&self.hash_key_name)
            .map(attr_key)
            .unwrap_or_default();

        let entry = self.partitions.get(&pk_val)?;
        let partition = entry.read();
        partition.get(key).cloned()
    }

    fn delete_item(&self, key: &Item) -> Option<Item> {
        let pk_val = key
            .get(&self.hash_key_name)
            .map(attr_key)
            .unwrap_or_default();

        let entry = self.partitions.get(&pk_val)?;
        let mut partition = entry.write();
        let old = partition.delete(key);
        if old.is_some() {
            self.item_count.fetch_sub(1, Ordering::Relaxed);
        }
        old
    }

    fn item_count(&self) -> i64 {
        self.item_count.load(Ordering::Relaxed)
    }

    fn description(&self) -> TableDescription {
        TableDescription {
            table_name: self.name.clone(),
            table_status: "ACTIVE".to_string(),
            key_schema: self.key_schema.clone(),
            attribute_definitions: self.attribute_definitions.clone(),
            item_count: self.item_count(),
            table_arn: self.arn.clone(),
            billing_mode_summary: BillingModeSummary {
                billing_mode: self.billing_mode.clone(),
            },
        }
    }
}

/// The top-level DynamoDB store. Lock-free table lookup via DashMap.
pub struct DdbStore {
    tables: DashMap<String, Table>,
    account_id: String,
    region: String,
}

impl DdbStore {
    pub fn new(account_id: String, region: String) -> Self {
        Self {
            tables: DashMap::new(),
            account_id,
            region,
        }
    }

    fn table_arn(&self, name: &str) -> String {
        format!(
            "arn:aws:dynamodb:{}:{}:table/{}",
            self.region, self.account_id, name
        )
    }

    /// Handle a DynamoDB request. Returns (status_code, response_json_bytes).
    pub fn handle(&self, action: &str, body: &[u8]) -> (u16, Vec<u8>) {
        match action {
            "CreateTable" => self.create_table(body),
            "DeleteTable" => self.delete_table(body),
            "ListTables" => self.list_tables(),
            "PutItem" => self.put_item(body),
            "GetItem" => self.get_item(body),
            "DeleteItem" => self.delete_item(body),
            _ => {
                // Fall through to Go for unsupported actions
                (0, Vec::new())
            }
        }
    }

    fn create_table(&self, body: &[u8]) -> (u16, Vec<u8>) {
        let req: CreateTableRequest = match serde_json::from_slice(body) {
            Ok(r) => r,
            Err(_) => return self.error(400, "ValidationException", "Invalid JSON"),
        };

        if req.table_name.is_empty() {
            return self.error(400, "ValidationException", "TableName is required.");
        }

        let arn = self.table_arn(&req.table_name);
        let billing = req.billing_mode.unwrap_or_else(|| "PAY_PER_REQUEST".to_string());
        let table = Table::new(
            req.table_name.clone(),
            req.key_schema,
            req.attribute_definitions,
            billing,
            arn,
        );

        // Atomic insert — fail if already exists.
        if self.tables.contains_key(&req.table_name) {
            return self.error(
                400,
                "ResourceInUseException",
                &format!("Table already exists: {}", req.table_name),
            );
        }

        let desc = table.description();
        self.tables.insert(req.table_name, table);

        let resp = CreateTableResponse {
            table_description: desc,
        };
        (200, serde_json::to_vec(&resp).unwrap_or_default())
    }

    fn delete_table(&self, body: &[u8]) -> (u16, Vec<u8>) {
        let req: DeleteTableRequest = match serde_json::from_slice(body) {
            Ok(r) => r,
            Err(_) => return self.error(400, "ValidationException", "Invalid JSON"),
        };

        match self.tables.remove(&req.table_name) {
            Some((_, table)) => {
                let mut desc = table.description();
                desc.table_status = "DELETING".to_string();
                let resp = DeleteTableResponse {
                    table_description: desc,
                };
                (200, serde_json::to_vec(&resp).unwrap_or_default())
            }
            None => self.error(
                400,
                "ResourceNotFoundException",
                &format!("Requested resource not found: Table: {} not found", req.table_name),
            ),
        }
    }

    fn list_tables(&self) -> (u16, Vec<u8>) {
        let names: Vec<String> = self.tables.iter().map(|e| e.key().clone()).collect();
        let resp = ListTablesResponse { table_names: names };
        (200, serde_json::to_vec(&resp).unwrap_or_default())
    }

    fn put_item(&self, body: &[u8]) -> (u16, Vec<u8>) {
        let req: PutItemRequest = match serde_json::from_slice(body) {
            Ok(r) => r,
            Err(_) => return self.error(400, "ValidationException", "Invalid JSON"),
        };

        if req.table_name.is_empty() {
            return self.error(400, "ValidationException", "TableName is required.");
        }

        match self.tables.get(&req.table_name) {
            Some(table) => {
                table.put_item(req.item);
                (200, b"{}".to_vec())
            }
            None => self.error(
                400,
                "ResourceNotFoundException",
                &format!("Requested resource not found: Table: {} not found", req.table_name),
            ),
        }
    }

    fn get_item(&self, body: &[u8]) -> (u16, Vec<u8>) {
        let req: GetItemRequest = match serde_json::from_slice(body) {
            Ok(r) => r,
            Err(_) => return self.error(400, "ValidationException", "Invalid JSON"),
        };

        if req.table_name.is_empty() {
            return self.error(400, "ValidationException", "TableName is required.");
        }

        match self.tables.get(&req.table_name) {
            Some(table) => {
                let item = table.get_item(&req.key);
                let resp = GetItemResponse { item };
                (200, serde_json::to_vec(&resp).unwrap_or_default())
            }
            None => self.error(
                400,
                "ResourceNotFoundException",
                &format!("Requested resource not found: Table: {} not found", req.table_name),
            ),
        }
    }

    fn delete_item(&self, body: &[u8]) -> (u16, Vec<u8>) {
        let req: DeleteItemRequest = match serde_json::from_slice(body) {
            Ok(r) => r,
            Err(_) => return self.error(400, "ValidationException", "Invalid JSON"),
        };

        if req.table_name.is_empty() {
            return self.error(400, "ValidationException", "TableName is required.");
        }

        match self.tables.get(&req.table_name) {
            Some(table) => {
                table.delete_item(&req.key);
                (200, b"{}".to_vec())
            }
            None => self.error(
                400,
                "ResourceNotFoundException",
                &format!("Requested resource not found: Table: {} not found", req.table_name),
            ),
        }
    }

    fn error(&self, status: u16, code: &str, message: &str) -> (u16, Vec<u8>) {
        let err = DdbError {
            error_type: code.to_string(),
            message: message.to_string(),
        };
        (status, serde_json::to_vec(&err).unwrap_or_default())
    }
}
