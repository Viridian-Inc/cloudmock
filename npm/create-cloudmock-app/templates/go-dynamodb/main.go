package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var (
	tableName = envOr("TABLE_NAME", "items")
	port      = envOr("PORT", "3000")
)

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func newDynamoClient(ctx context.Context) *dynamodb.Client {
	endpointURL := envOr("AWS_ENDPOINT_URL", "http://localhost:4566")
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(envOr("AWS_REGION", "us-east-1")),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			envOr("AWS_ACCESS_KEY_ID", "test"),
			envOr("AWS_SECRET_ACCESS_KEY", "test"),
			"",
		)),
		config.WithBaseEndpoint(endpointURL),
	)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	return dynamodb.NewFromConfig(cfg)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func errorJSON(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// POST /items
func handleCreate(client *dynamodb.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var item map[string]any
		if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
			errorJSON(w, 400, "invalid JSON")
			return
		}
		if _, ok := item["id"]; !ok {
			errorJSON(w, 400, "id is required")
			return
		}
		av, err := attributevalue.MarshalMap(item)
		if err != nil {
			errorJSON(w, 500, err.Error())
			return
		}
		_, err = client.PutItem(r.Context(), &dynamodb.PutItemInput{
			TableName: &tableName,
			Item:      av,
		})
		if err != nil {
			errorJSON(w, 500, err.Error())
			return
		}
		writeJSON(w, 201, item)
	}
}

// GET /items/{id}
func handleGet(client *dynamodb.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		result, err := client.GetItem(r.Context(), &dynamodb.GetItemInput{
			TableName: &tableName,
			Key: map[string]types.AttributeValue{
				"id": &types.AttributeValueMemberS{Value: id},
			},
		})
		if err != nil {
			errorJSON(w, 500, err.Error())
			return
		}
		if result.Item == nil {
			errorJSON(w, 404, "not found")
			return
		}
		var item map[string]any
		if err := attributevalue.UnmarshalMap(result.Item, &item); err != nil {
			errorJSON(w, 500, err.Error())
			return
		}
		writeJSON(w, 200, item)
	}
}

// GET /items
func handleList(client *dynamodb.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := client.Scan(r.Context(), &dynamodb.ScanInput{
			TableName: &tableName,
		})
		if err != nil {
			errorJSON(w, 500, err.Error())
			return
		}
		var items []map[string]any
		if err := attributevalue.UnmarshalListOfMaps(result.Items, &items); err != nil {
			errorJSON(w, 500, err.Error())
			return
		}
		if items == nil {
			items = []map[string]any{}
		}
		writeJSON(w, 200, items)
	}
}

// DELETE /items/{id}
func handleDelete(client *dynamodb.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		_, err := client.DeleteItem(r.Context(), &dynamodb.DeleteItemInput{
			TableName: &tableName,
			Key: map[string]types.AttributeValue{
				"id": &types.AttributeValueMemberS{Value: id},
			},
		})
		if err != nil {
			errorJSON(w, 500, err.Error())
			return
		}
		w.WriteHeader(204)
	}
}

func NewServer(client *dynamodb.Client) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /items", handleCreate(client))
	mux.HandleFunc("GET /items/{id}", handleGet(client))
	mux.HandleFunc("GET /items", handleList(client))
	mux.HandleFunc("DELETE /items/{id}", handleDelete(client))
	return mux
}

func main() {
	ctx := context.Background()
	client := newDynamoClient(ctx)

	// Create table if it doesn't exist
	_, err := client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String(tableName),
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String("id"), KeyType: types.KeyTypeHash},
		},
		AttributeDefinitions: []types.AttributeDefinition{
			{AttributeName: aws.String("id"), AttributeType: types.ScalarAttributeTypeS},
		},
		BillingMode: types.BillingModePayPerRequest,
	})
	if err != nil {
		// Ignore ResourceInUseException (table already exists)
		log.Printf("create table: %v (may already exist)", err)
	}

	mux := NewServer(client)
	log.Printf("Listening on http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
