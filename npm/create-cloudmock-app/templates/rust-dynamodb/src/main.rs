use aws_config::BehaviorVersion;
use aws_sdk_dynamodb::{
    config::{Credentials, Region},
    types::{
        AttributeDefinition, AttributeValue, BillingMode, KeySchemaElement, KeyType,
        ScalarAttributeType,
    },
    Client,
};
use axum::{
    extract::{Path, State},
    http::StatusCode,
    response::IntoResponse,
    routing::{delete, get, post},
    Json, Router,
};
use serde::{Deserialize, Serialize};
use serde_json::{json, Value};
use std::{collections::HashMap, env, sync::Arc};

const TABLE: &str = "items";

#[derive(Clone)]
pub struct AppState {
    pub dynamo: Arc<Client>,
}

#[derive(Deserialize, Serialize)]
pub struct Item {
    pub id: String,
    #[serde(flatten)]
    pub extra: HashMap<String, Value>,
}

pub fn build_router(state: AppState) -> Router {
    Router::new()
        .route("/items", post(create_item))
        .route("/items", get(list_items))
        .route("/items/:id", get(get_item))
        .route("/items/:id", delete(delete_item))
        .with_state(state)
}

async fn create_item(
    State(state): State<AppState>,
    Json(item): Json<Item>,
) -> impl IntoResponse {
    let mut av: HashMap<String, AttributeValue> = HashMap::new();
    av.insert("id".into(), AttributeValue::S(item.id.clone()));
    for (k, v) in &item.extra {
        av.insert(k.clone(), AttributeValue::S(v.to_string()));
    }
    match state
        .dynamo
        .put_item()
        .table_name(TABLE)
        .set_item(Some(av))
        .send()
        .await
    {
        Ok(_) => (StatusCode::CREATED, Json(json!({"id": item.id}))).into_response(),
        Err(e) => (
            StatusCode::INTERNAL_SERVER_ERROR,
            Json(json!({"error": e.to_string()})),
        )
            .into_response(),
    }
}

async fn get_item(
    State(state): State<AppState>,
    Path(id): Path<String>,
) -> impl IntoResponse {
    let key = HashMap::from([("id".to_string(), AttributeValue::S(id.clone()))]);
    match state
        .dynamo
        .get_item()
        .table_name(TABLE)
        .set_key(Some(key))
        .send()
        .await
    {
        Ok(out) => match out.item {
            None => (StatusCode::NOT_FOUND, Json(json!({"error": "not found"}))).into_response(),
            Some(item) => {
                let flat: HashMap<String, String> = item
                    .into_iter()
                    .filter_map(|(k, v)| v.as_s().ok().map(|s| (k, s.clone())))
                    .collect();
                (StatusCode::OK, Json(json!(flat))).into_response()
            }
        },
        Err(e) => (
            StatusCode::INTERNAL_SERVER_ERROR,
            Json(json!({"error": e.to_string()})),
        )
            .into_response(),
    }
}

async fn list_items(State(state): State<AppState>) -> impl IntoResponse {
    match state.dynamo.scan().table_name(TABLE).send().await {
        Ok(out) => {
            let items: Vec<HashMap<String, String>> = out
                .items
                .unwrap_or_default()
                .into_iter()
                .map(|row| {
                    row.into_iter()
                        .filter_map(|(k, v)| v.as_s().ok().map(|s| (k, s.clone())))
                        .collect()
                })
                .collect();
            (StatusCode::OK, Json(json!(items))).into_response()
        }
        Err(e) => (
            StatusCode::INTERNAL_SERVER_ERROR,
            Json(json!({"error": e.to_string()})),
        )
            .into_response(),
    }
}

async fn delete_item(
    State(state): State<AppState>,
    Path(id): Path<String>,
) -> impl IntoResponse {
    let key = HashMap::from([("id".to_string(), AttributeValue::S(id))]);
    match state
        .dynamo
        .delete_item()
        .table_name(TABLE)
        .set_key(Some(key))
        .send()
        .await
    {
        Ok(_) => StatusCode::NO_CONTENT.into_response(),
        Err(e) => (
            StatusCode::INTERNAL_SERVER_ERROR,
            Json(json!({"error": e.to_string()})),
        )
            .into_response(),
    }
}

pub async fn build_client() -> Client {
    let endpoint = env::var("AWS_ENDPOINT_URL")
        .unwrap_or_else(|_| "http://localhost:4566".to_string());
    let region = env::var("AWS_REGION").unwrap_or_else(|_| "us-east-1".to_string());
    let access_key = env::var("AWS_ACCESS_KEY_ID").unwrap_or_else(|_| "test".to_string());
    let secret_key = env::var("AWS_SECRET_ACCESS_KEY").unwrap_or_else(|_| "test".to_string());

    let creds = Credentials::new(access_key, secret_key, None, None, "static");
    let config = aws_config::defaults(BehaviorVersion::latest())
        .region(Region::new(region))
        .credentials_provider(creds)
        .endpoint_url(endpoint)
        .load()
        .await;
    Client::new(&config)
}

pub async fn ensure_table(client: &Client) {
    let exists = client
        .list_tables()
        .send()
        .await
        .map(|r| r.table_names.unwrap_or_default().contains(&TABLE.to_string()))
        .unwrap_or(false);

    if !exists {
        client
            .create_table()
            .table_name(TABLE)
            .key_schema(
                KeySchemaElement::builder()
                    .attribute_name("id")
                    .key_type(KeyType::Hash)
                    .build()
                    .unwrap(),
            )
            .attribute_definitions(
                AttributeDefinition::builder()
                    .attribute_name("id")
                    .attribute_type(ScalarAttributeType::S)
                    .build()
                    .unwrap(),
            )
            .billing_mode(BillingMode::PayPerRequest)
            .send()
            .await
            .ok();
    }
}

#[tokio::main]
async fn main() {
    tracing_subscriber::fmt::init();
    let client = build_client().await;
    ensure_table(&client).await;

    let state = AppState {
        dynamo: Arc::new(client),
    };
    let app = build_router(state);
    let port = env::var("PORT").unwrap_or_else(|_| "3000".to_string());
    let listener = tokio::net::TcpListener::bind(format!("0.0.0.0:{port}"))
        .await
        .unwrap();
    tracing::info!("Listening on http://localhost:{port}");
    axum::serve(listener, app).await.unwrap();
}
