package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

var sqsClient *sqs.Client
var queueURL string

func main() {
	endpoint := os.Getenv("AWS_ENDPOINT_URL")
	if endpoint == "" {
		endpoint = "http://cloudmock:4566"
	}
	queueURL = os.Getenv("NOTIFICATIONS_QUEUE_URL")
	if queueURL == "" {
		queueURL = endpoint + "/000000000000/notifications"
	}

	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion("us-east-1"),
		config.WithBaseEndpoint(endpoint),
		config.WithCredentialsProvider(aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{AccessKeyID: "test", SecretAccessKey: "test"}, nil
		})),
	)
	if err != nil {
		log.Fatal(err)
	}
	sqsClient = sqs.NewFromConfig(cfg)

	go pollQueue()

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"status":"ok"}`)
	})

	log.Println("Notification service on :3003")
	log.Fatal(http.ListenAndServe(":3003", nil))
}

func pollQueue() {
	for {
		out, err := sqsClient.ReceiveMessage(context.Background(), &sqs.ReceiveMessageInput{
			QueueUrl:            &queueURL,
			WaitTimeSeconds:     5,
			MaxNumberOfMessages: 10,
		})
		if err != nil {
			log.Printf("receive error: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}

		for _, msg := range out.Messages {
			var payload map[string]interface{}
			if err := json.Unmarshal([]byte(*msg.Body), &payload); err != nil {
				log.Printf("parse error: %v", err)
				continue
			}
			log.Printf("Sending notification for order %v", payload["id"])

			sqsClient.DeleteMessage(context.Background(), &sqs.DeleteMessageInput{
				QueueUrl:      &queueURL,
				ReceiptHandle: msg.ReceiptHandle,
			})
		}
	}
}
