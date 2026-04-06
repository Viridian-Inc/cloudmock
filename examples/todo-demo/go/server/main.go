package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/google/uuid"
)

var (
	ddb      *dynamodb.Client
	s3c      *s3.Client
	snsc     *sns.Client
	sqsc     *sqs.Client
	topicArn string
	queueURL string
)

type Todo struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	ImageKey    string `json:"imageKey,omitempty"`
	CreatedAt   string `json:"createdAt"`
}

func itemToTodo(item map[string]ddbtypes.AttributeValue) Todo {
	t := Todo{
		ID:        item["id"].(*ddbtypes.AttributeValueMemberS).Value,
		Title:     item["title"].(*ddbtypes.AttributeValueMemberS).Value,
		Status:    item["status"].(*ddbtypes.AttributeValueMemberS).Value,
		CreatedAt: item["createdAt"].(*ddbtypes.AttributeValueMemberS).Value,
	}
	if v, ok := item["description"]; ok {
		t.Description = v.(*ddbtypes.AttributeValueMemberS).Value
	}
	if v, ok := item["imageKey"]; ok {
		t.ImageKey = v.(*ddbtypes.AttributeValueMemberS).Value
	}
	return t
}

func setup(ctx context.Context) {
	endpoint := os.Getenv("CLOUDMOCK_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:4566"
	}

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
		config.WithBaseEndpoint(endpoint),
	)
	if err != nil {
		log.Fatal(err)
	}

	ddb = dynamodb.NewFromConfig(cfg)
	s3c = s3.NewFromConfig(cfg, func(o *s3.Options) { o.UsePathStyle = true })
	snsc = sns.NewFromConfig(cfg)
	sqsc = sqs.NewFromConfig(cfg)

	ddb.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName:            aws.String("todos"),
		KeySchema:            []ddbtypes.KeySchemaElement{{AttributeName: aws.String("id"), KeyType: ddbtypes.KeyTypeHash}},
		AttributeDefinitions: []ddbtypes.AttributeDefinition{{AttributeName: aws.String("id"), AttributeType: ddbtypes.ScalarAttributeTypeS}},
		BillingMode:          ddbtypes.BillingModePayPerRequest,
	})
	s3c.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String("todo-attachments")})
	topicOut, _ := snsc.CreateTopic(ctx, &sns.CreateTopicInput{Name: aws.String("todo-completed")})
	topicArn = *topicOut.TopicArn
	queueOut, _ := sqsc.CreateQueue(ctx, &sqs.CreateQueueInput{QueueName: aws.String("todo-notifications")})
	queueURL = *queueOut.QueueUrl
	attrsOut, _ := sqsc.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
		QueueUrl:       &queueURL,
		AttributeNames: []sqstypes.QueueAttributeName{sqstypes.QueueAttributeNameQueueArn},
	})
	snsc.Subscribe(ctx, &sns.SubscribeInput{TopicArn: &topicArn, Protocol: aws.String("sqs"), Endpoint: aws.String(attrsOut.Attributes["QueueArn"])})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func main() {
	ctx := context.Background()
	setup(ctx)

	mux := http.NewServeMux()

	mux.HandleFunc("POST /todos", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Title       string `json:"title"`
			Description string `json:"description"`
		}
		json.NewDecoder(r.Body).Decode(&body)

		id := uuid.NewString()
		now := time.Now().UTC().Format(time.RFC3339)
		ddb.PutItem(r.Context(), &dynamodb.PutItemInput{
			TableName: aws.String("todos"),
			Item: map[string]ddbtypes.AttributeValue{
				"id":          &ddbtypes.AttributeValueMemberS{Value: id},
				"title":       &ddbtypes.AttributeValueMemberS{Value: body.Title},
				"description": &ddbtypes.AttributeValueMemberS{Value: body.Description},
				"status":      &ddbtypes.AttributeValueMemberS{Value: "pending"},
				"createdAt":   &ddbtypes.AttributeValueMemberS{Value: now},
			},
		})
		writeJSON(w, 201, Todo{ID: id, Title: body.Title, Description: body.Description, Status: "pending", CreatedAt: now})
	})

	mux.HandleFunc("GET /todos", func(w http.ResponseWriter, r *http.Request) {
		out, _ := ddb.Scan(r.Context(), &dynamodb.ScanInput{TableName: aws.String("todos")})
		todos := make([]Todo, 0, len(out.Items))
		for _, item := range out.Items {
			todos = append(todos, itemToTodo(item))
		}
		writeJSON(w, 200, todos)
	})

	mux.HandleFunc("GET /todos/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		out, _ := ddb.GetItem(r.Context(), &dynamodb.GetItemInput{
			TableName: aws.String("todos"),
			Key:       map[string]ddbtypes.AttributeValue{"id": &ddbtypes.AttributeValueMemberS{Value: id}},
		})
		if out.Item == nil {
			writeJSON(w, 404, map[string]string{"error": "Not found"})
			return
		}
		writeJSON(w, 200, itemToTodo(out.Item))
	})

	mux.HandleFunc("PUT /todos/{id}/image", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		imageKey := fmt.Sprintf("todos/%s/image.jpg", id)
		body, _ := io.ReadAll(r.Body)
		s3c.PutObject(r.Context(), &s3.PutObjectInput{
			Bucket:      aws.String("todo-attachments"),
			Key:         aws.String(imageKey),
			Body:        bytes.NewReader(body),
			ContentType: aws.String(r.Header.Get("Content-Type")),
		})
		ddb.UpdateItem(r.Context(), &dynamodb.UpdateItemInput{
			TableName:                 aws.String("todos"),
			Key:                       map[string]ddbtypes.AttributeValue{"id": &ddbtypes.AttributeValueMemberS{Value: id}},
			UpdateExpression:          aws.String("SET imageKey = :key"),
			ExpressionAttributeValues: map[string]ddbtypes.AttributeValue{":key": &ddbtypes.AttributeValueMemberS{Value: imageKey}},
		})
		writeJSON(w, 200, map[string]string{"imageKey": imageKey})
	})

	mux.HandleFunc("PUT /todos/{id}/complete", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		out, _ := ddb.GetItem(r.Context(), &dynamodb.GetItemInput{
			TableName: aws.String("todos"),
			Key:       map[string]ddbtypes.AttributeValue{"id": &ddbtypes.AttributeValueMemberS{Value: id}},
		})
		if out.Item == nil {
			writeJSON(w, 404, map[string]string{"error": "Not found"})
			return
		}
		ddb.UpdateItem(r.Context(), &dynamodb.UpdateItemInput{
			TableName:                 aws.String("todos"),
			Key:                       map[string]ddbtypes.AttributeValue{"id": &ddbtypes.AttributeValueMemberS{Value: id}},
			UpdateExpression:          aws.String("SET #s = :s"),
			ExpressionAttributeNames:  map[string]string{"#s": "status"},
			ExpressionAttributeValues: map[string]ddbtypes.AttributeValue{":s": &ddbtypes.AttributeValueMemberS{Value: "completed"}},
		})
		title := out.Item["title"].(*ddbtypes.AttributeValueMemberS).Value
		msgBody, _ := json.Marshal(map[string]string{"todoId": id, "title": title, "completedAt": time.Now().UTC().Format(time.RFC3339)})
		snsc.Publish(r.Context(), &sns.PublishInput{TopicArn: &topicArn, Subject: aws.String("Todo Completed"), Message: aws.String(string(msgBody))})
		writeJSON(w, 200, map[string]string{"status": "completed"})
	})

	mux.HandleFunc("GET /notifications", func(w http.ResponseWriter, r *http.Request) {
		out, _ := sqsc.ReceiveMessage(r.Context(), &sqs.ReceiveMessageInput{QueueUrl: &queueURL, MaxNumberOfMessages: 10, WaitTimeSeconds: 1})
		notifications := make([]map[string]string, 0)
		for _, msg := range out.Messages {
			var envelope struct{ Message string }
			json.Unmarshal([]byte(*msg.Body), &envelope)
			var n map[string]string
			json.Unmarshal([]byte(envelope.Message), &n)
			notifications = append(notifications, n)
			sqsc.DeleteMessage(r.Context(), &sqs.DeleteMessageInput{QueueUrl: &queueURL, ReceiptHandle: msg.ReceiptHandle})
		}
		writeJSON(w, 200, notifications)
	})

	fmt.Println("Todo API running on http://localhost:3000")
	log.Fatal(http.ListenAndServe(":3000", mux))
}
