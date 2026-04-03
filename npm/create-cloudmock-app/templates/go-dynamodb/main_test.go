package main_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	sdk "github.com/neureaux/cloudmock/sdk"
)

func setupTest(t *testing.T) (*httptest.Server, func()) {
	t.Helper()
	cm := sdk.New()
	cfg := cm.Config()

	client := dynamodb.NewFromConfig(cfg)
	ctx := context.Background()

	_, err := client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String("items"),
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String("id"), KeyType: types.KeyTypeHash},
		},
		AttributeDefinitions: []types.AttributeDefinition{
			{AttributeName: aws.String("id"), AttributeType: types.ScalarAttributeTypeS},
		},
		BillingMode: types.BillingModePayPerRequest,
	})
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	mux := NewServer(client)
	srv := httptest.NewServer(mux)
	return srv, func() {
		srv.Close()
		cm.Stop()
	}
}

func postJSON(t *testing.T, srv *httptest.Server, path string, body any) *http.Response {
	t.Helper()
	b, _ := json.Marshal(body)
	res, err := http.Post(srv.URL+path, "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestCreateItem(t *testing.T) {
	srv, cleanup := setupTest(t)
	defer cleanup()

	res := postJSON(t, srv, "/items", map[string]any{"id": "1", "name": "Widget"})
	if res.StatusCode != 201 {
		t.Fatalf("expected 201, got %d", res.StatusCode)
	}
}

func TestGetItem(t *testing.T) {
	srv, cleanup := setupTest(t)
	defer cleanup()

	postJSON(t, srv, "/items", map[string]any{"id": "2", "name": "Gadget"})

	res, err := http.Get(srv.URL + "/items/2")
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var item map[string]any
	json.NewDecoder(res.Body).Decode(&item)
	if item["name"] != "Gadget" {
		t.Errorf("expected name=Gadget, got %v", item["name"])
	}
}

func TestListItems(t *testing.T) {
	srv, cleanup := setupTest(t)
	defer cleanup()

	postJSON(t, srv, "/items", map[string]any{"id": "3", "name": "A"})
	postJSON(t, srv, "/items", map[string]any{"id": "4", "name": "B"})

	res, err := http.Get(srv.URL + "/items")
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var items []any
	json.NewDecoder(res.Body).Decode(&items)
	if len(items) < 2 {
		t.Errorf("expected at least 2 items, got %d", len(items))
	}
}

func TestDeleteItem(t *testing.T) {
	srv, cleanup := setupTest(t)
	defer cleanup()

	postJSON(t, srv, "/items", map[string]any{"id": "5", "name": "Doomed"})

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/items/5", nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 204 {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}

	res, _ = http.Get(srv.URL + "/items/5")
	if res.StatusCode != 404 {
		t.Fatalf("expected 404, got %d", res.StatusCode)
	}
}
