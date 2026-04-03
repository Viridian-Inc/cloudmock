use axum_test::TestServer;
use cloudmock::MockAws;
use std::sync::Arc;

async fn setup() -> (TestServer, MockAws) {
    let cm = MockAws::start().await;
    std::env::set_var("AWS_ENDPOINT_URL", cm.endpoint());

    // Import from binary crate via integration test path
    use {{PROJECT_NAME}}::{build_client, build_router, ensure_table, AppState};

    let client = build_client().await;
    ensure_table(&client).await;
    let state = AppState {
        dynamo: Arc::new(client),
    };
    let app = build_router(state);
    let server = TestServer::new(app).unwrap();
    (server, cm)
}

#[tokio::test]
async fn test_create_item() {
    let (server, _cm) = setup().await;
    let res = server
        .post("/items")
        .json(&serde_json::json!({"id": "1", "name": "Widget"}))
        .await;
    assert_eq!(res.status_code(), 201);
}

#[tokio::test]
async fn test_get_item() {
    let (server, _cm) = setup().await;
    server
        .post("/items")
        .json(&serde_json::json!({"id": "2", "name": "Gadget"}))
        .await;
    let res = server.get("/items/2").await;
    assert_eq!(res.status_code(), 200);
    let body: serde_json::Value = res.json();
    assert_eq!(body["name"], "Gadget");
}

#[tokio::test]
async fn test_list_items() {
    let (server, _cm) = setup().await;
    server
        .post("/items")
        .json(&serde_json::json!({"id": "3", "name": "A"}))
        .await;
    let res = server.get("/items").await;
    assert_eq!(res.status_code(), 200);
    let items: Vec<serde_json::Value> = res.json();
    assert!(!items.is_empty());
}

#[tokio::test]
async fn test_delete_item() {
    let (server, _cm) = setup().await;
    server
        .post("/items")
        .json(&serde_json::json!({"id": "4", "name": "Doomed"}))
        .await;
    let res = server.delete("/items/4").await;
    assert_eq!(res.status_code(), 204);
    let res = server.get("/items/4").await;
    assert_eq!(res.status_code(), 404);
}
