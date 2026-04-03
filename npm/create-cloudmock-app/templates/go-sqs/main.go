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
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

var (
	queueURL = envOr("SQS_QUEUE_URL", "http://localhost:4566/000000000000/messages")
	port     = envOr("PORT", "3000")
)

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func newSQSClient(ctx context.Context) *sqs.Client {
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
	return sqs.NewFromConfig(cfg)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func errorJSON(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// POST /messages — send a message
func handleSend(client *sqs.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			errorJSON(w, 400, "invalid JSON")
			return
		}
		if _, ok := payload["message"]; !ok {
			errorJSON(w, 400, "message is required")
			return
		}
		body, _ := json.Marshal(payload)
		result, err := client.SendMessage(r.Context(), &sqs.SendMessageInput{
			QueueUrl:    aws.String(queueURL),
			MessageBody: aws.String(string(body)),
		})
		if err != nil {
			errorJSON(w, 500, err.Error())
			return
		}
		writeJSON(w, 202, map[string]string{"messageId": aws.ToString(result.MessageId)})
	}
}

// GET /messages — receive and delete messages
func handleReceive(client *sqs.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := client.ReceiveMessage(r.Context(), &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(queueURL),
			MaxNumberOfMessages: 10,
			WaitTimeSeconds:     1,
		})
		if err != nil {
			errorJSON(w, 500, err.Error())
			return
		}

		messages := make([]map[string]any, 0, len(result.Messages))
		for _, msg := range result.Messages {
			// Delete after receiving
			_, _ = client.DeleteMessage(r.Context(), &sqs.DeleteMessageInput{
				QueueUrl:      aws.String(queueURL),
				ReceiptHandle: msg.ReceiptHandle,
			})

			var payload map[string]any
			if err := json.Unmarshal([]byte(aws.ToString(msg.Body)), &payload); err != nil {
				payload = map[string]any{"body": aws.ToString(msg.Body)}
			}
			messages = append(messages, payload)
		}
		writeJSON(w, 200, messages)
	}
}

func NewServer(client *sqs.Client) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /messages", handleSend(client))
	mux.HandleFunc("GET /messages", handleReceive(client))
	return mux
}

func main() {
	ctx := context.Background()
	client := newSQSClient(ctx)

	// Create queue if it doesn't exist
	_, err := client.CreateQueue(ctx, &sqs.CreateQueueInput{
		QueueName: aws.String("messages"),
	})
	if err != nil {
		log.Printf("create queue: %v (may already exist)", err)
	}

	mux := NewServer(client)
	log.Printf("Listening on http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
