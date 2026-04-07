package tier1

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"github.com/Viridian-Inc/cloudmock/benchmarks/awsclient"
	"github.com/Viridian-Inc/cloudmock/benchmarks/harness"
)

type dynamoDBSuite struct{}

func NewDynamoDBSuite() harness.Suite { return &dynamoDBSuite{} }

func (s *dynamoDBSuite) Name() string { return "dynamodb" }
func (s *dynamoDBSuite) Tier() int    { return 1 }

func newDDBClient(endpoint string) (*dynamodb.Client, error) {
	cfg, err := awsclient.NewConfig(endpoint)
	if err != nil {
		return nil, err
	}
	client := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		o.BaseEndpoint = awsclient.Endpoint(endpoint)
	})
	return client, nil
}

func createTable(ctx context.Context, endpoint, tableName string) error {
	client, err := newDDBClient(endpoint)
	if err != nil {
		return err
	}
	_, err = client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String(tableName),
		AttributeDefinitions: []ddbtypes.AttributeDefinition{
			{
				AttributeName: aws.String("id"),
				AttributeType: ddbtypes.ScalarAttributeTypeS,
			},
		},
		KeySchema: []ddbtypes.KeySchemaElement{
			{
				AttributeName: aws.String("id"),
				KeyType:       ddbtypes.KeyTypeHash,
			},
		},
		BillingMode: ddbtypes.BillingModePayPerRequest,
	})
	return err
}

func deleteTable(ctx context.Context, endpoint, tableName string) error {
	client, err := newDDBClient(endpoint)
	if err != nil {
		return err
	}
	_, err = client.DeleteTable(ctx, &dynamodb.DeleteTableInput{
		TableName: aws.String(tableName),
	})
	return err
}

func (s *dynamoDBSuite) Operations() []harness.Operation {
	return []harness.Operation{
		{
			Name: "CreateTable",
			Run: func(ctx context.Context, endpoint string) (any, error) {
				tableName := fmt.Sprintf("bench-%s", uuid.New().String()[:8])
				client, err := newDDBClient(endpoint)
				if err != nil {
					return nil, err
				}
				out, err := client.CreateTable(ctx, &dynamodb.CreateTableInput{
					TableName: aws.String(tableName),
					AttributeDefinitions: []ddbtypes.AttributeDefinition{
						{
							AttributeName: aws.String("id"),
							AttributeType: ddbtypes.ScalarAttributeTypeS,
						},
					},
					KeySchema: []ddbtypes.KeySchemaElement{
						{
							AttributeName: aws.String("id"),
							KeyType:       ddbtypes.KeyTypeHash,
						},
					},
					BillingMode: ddbtypes.BillingModePayPerRequest,
				})
				if err != nil {
					return nil, err
				}
				// Clean up the created table
				client.DeleteTable(ctx, &dynamodb.DeleteTableInput{TableName: aws.String(tableName)})
				return out, nil
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "CreateTableOutput")}
			},
		},
		{
			Name: "PutItem",
			Setup: func(ctx context.Context, endpoint string) error {
				return createTable(ctx, endpoint, "bench-putitem")
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newDDBClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.PutItem(ctx, &dynamodb.PutItemInput{
					TableName: aws.String("bench-putitem"),
					Item: map[string]ddbtypes.AttributeValue{
						"id":    &ddbtypes.AttributeValueMemberS{Value: uuid.New().String()},
						"data":  &ddbtypes.AttributeValueMemberS{Value: "benchmark-data"},
						"count": &ddbtypes.AttributeValueMemberN{Value: "1"},
					},
				})
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "PutItemOutput")}
			},
			Teardown: func(ctx context.Context, endpoint string) error {
				return deleteTable(ctx, endpoint, "bench-putitem")
			},
		},
		{
			Name: "GetItem",
			Setup: func(ctx context.Context, endpoint string) error {
				if err := createTable(ctx, endpoint, "bench-getitem"); err != nil {
					return err
				}
				client, err := newDDBClient(endpoint)
				if err != nil {
					return err
				}
				_, err = client.PutItem(ctx, &dynamodb.PutItemInput{
					TableName: aws.String("bench-getitem"),
					Item: map[string]ddbtypes.AttributeValue{
						"id":   &ddbtypes.AttributeValueMemberS{Value: "bench-pk"},
						"data": &ddbtypes.AttributeValueMemberS{Value: "benchmark-data"},
					},
				})
				return err
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newDDBClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.GetItem(ctx, &dynamodb.GetItemInput{
					TableName: aws.String("bench-getitem"),
					Key: map[string]ddbtypes.AttributeValue{
						"id": &ddbtypes.AttributeValueMemberS{Value: "bench-pk"},
					},
				})
			},
			Validate: func(resp any) []harness.Finding {
				out, ok := resp.(*dynamodb.GetItemOutput)
				if !ok || out == nil {
					return []harness.Finding{harness.CheckNotNil(nil, "GetItemOutput")}
				}
				findings := []harness.Finding{harness.CheckNotNil(out, "GetItemOutput")}
				pkVal, exists := out.Item["id"]
				if !exists {
					findings = append(findings, harness.Finding{
						Field:    "id",
						Expected: "bench-pk",
						Actual:   "<missing>",
						Grade:    harness.GradeFail,
					})
				} else if v, ok := pkVal.(*ddbtypes.AttributeValueMemberS); !ok || v.Value != "bench-pk" {
					actual := "<wrong type>"
					if ok {
						actual = v.Value
					}
					findings = append(findings, harness.Finding{
						Field:    "id",
						Expected: "bench-pk",
						Actual:   actual,
						Grade:    harness.GradeFail,
					})
				} else {
					findings = append(findings, harness.Finding{
						Field:    "id",
						Expected: "bench-pk",
						Actual:   v.Value,
						Grade:    harness.GradePass,
					})
				}
				return findings
			},
			Teardown: func(ctx context.Context, endpoint string) error {
				return deleteTable(ctx, endpoint, "bench-getitem")
			},
		},
		{
			Name: "Query",
			Setup: func(ctx context.Context, endpoint string) error {
				if err := createTable(ctx, endpoint, "bench-query"); err != nil {
					return err
				}
				client, err := newDDBClient(endpoint)
				if err != nil {
					return err
				}
				_, err = client.PutItem(ctx, &dynamodb.PutItemInput{
					TableName: aws.String("bench-query"),
					Item: map[string]ddbtypes.AttributeValue{
						"id":   &ddbtypes.AttributeValueMemberS{Value: "bench-pk"},
						"data": &ddbtypes.AttributeValueMemberS{Value: "benchmark-data"},
					},
				})
				return err
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newDDBClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.Query(ctx, &dynamodb.QueryInput{
					TableName:              aws.String("bench-query"),
					KeyConditionExpression: aws.String("#pk = :val"),
					ExpressionAttributeNames: map[string]string{
						"#pk": "id",
					},
					ExpressionAttributeValues: map[string]ddbtypes.AttributeValue{
						":val": &ddbtypes.AttributeValueMemberS{Value: "bench-pk"},
					},
				})
			},
			Validate: func(resp any) []harness.Finding {
				out, ok := resp.(*dynamodb.QueryOutput)
				if !ok || out == nil {
					return []harness.Finding{harness.CheckNotNil(nil, "QueryOutput")}
				}
				findings := []harness.Finding{harness.CheckNotNil(out, "QueryOutput")}
				if out.Count < 1 {
					findings = append(findings, harness.Finding{
						Field:    "Count",
						Expected: ">= 1",
						Actual:   fmt.Sprintf("%d", out.Count),
						Grade:    harness.GradeFail,
					})
				} else {
					findings = append(findings, harness.Finding{
						Field:    "Count",
						Expected: ">= 1",
						Actual:   fmt.Sprintf("%d", out.Count),
						Grade:    harness.GradePass,
					})
				}
				return findings
			},
			Teardown: func(ctx context.Context, endpoint string) error {
				return deleteTable(ctx, endpoint, "bench-query")
			},
		},
		{
			Name: "Scan",
			Setup: func(ctx context.Context, endpoint string) error {
				if err := createTable(ctx, endpoint, "bench-scan"); err != nil {
					return err
				}
				client, err := newDDBClient(endpoint)
				if err != nil {
					return err
				}
				_, err = client.PutItem(ctx, &dynamodb.PutItemInput{
					TableName: aws.String("bench-scan"),
					Item: map[string]ddbtypes.AttributeValue{
						"id":   &ddbtypes.AttributeValueMemberS{Value: "bench-pk"},
						"data": &ddbtypes.AttributeValueMemberS{Value: "benchmark-data"},
					},
				})
				return err
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newDDBClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.Scan(ctx, &dynamodb.ScanInput{
					TableName: aws.String("bench-scan"),
				})
			},
			Validate: func(resp any) []harness.Finding {
				out, ok := resp.(*dynamodb.ScanOutput)
				if !ok || out == nil {
					return []harness.Finding{harness.CheckNotNil(nil, "ScanOutput")}
				}
				findings := []harness.Finding{harness.CheckNotNil(out, "ScanOutput")}
				if out.Count < 1 {
					findings = append(findings, harness.Finding{
						Field:    "Count",
						Expected: ">= 1",
						Actual:   fmt.Sprintf("%d", out.Count),
						Grade:    harness.GradeFail,
					})
				} else {
					findings = append(findings, harness.Finding{
						Field:    "Count",
						Expected: ">= 1",
						Actual:   fmt.Sprintf("%d", out.Count),
						Grade:    harness.GradePass,
					})
				}
				return findings
			},
			Teardown: func(ctx context.Context, endpoint string) error {
				return deleteTable(ctx, endpoint, "bench-scan")
			},
		},
		{
			Name: "UpdateItem",
			Setup: func(ctx context.Context, endpoint string) error {
				if err := createTable(ctx, endpoint, "bench-update"); err != nil {
					return err
				}
				client, err := newDDBClient(endpoint)
				if err != nil {
					return err
				}
				_, err = client.PutItem(ctx, &dynamodb.PutItemInput{
					TableName: aws.String("bench-update"),
					Item: map[string]ddbtypes.AttributeValue{
						"id":   &ddbtypes.AttributeValueMemberS{Value: "bench-pk"},
						"data": &ddbtypes.AttributeValueMemberS{Value: "original"},
					},
				})
				return err
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newDDBClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
					TableName: aws.String("bench-update"),
					Key: map[string]ddbtypes.AttributeValue{
						"id": &ddbtypes.AttributeValueMemberS{Value: "bench-pk"},
					},
					UpdateExpression: aws.String("SET #d = :newval"),
					ExpressionAttributeNames: map[string]string{
						"#d": "data",
					},
					ExpressionAttributeValues: map[string]ddbtypes.AttributeValue{
						":newval": &ddbtypes.AttributeValueMemberS{Value: "updated"},
					},
				})
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "UpdateItemOutput")}
			},
			Teardown: func(ctx context.Context, endpoint string) error {
				return deleteTable(ctx, endpoint, "bench-update")
			},
		},
		{
			Name: "DeleteItem",
			Setup: func(ctx context.Context, endpoint string) error {
				if err := createTable(ctx, endpoint, "bench-delete"); err != nil {
					return err
				}
				client, err := newDDBClient(endpoint)
				if err != nil {
					return err
				}
				_, err = client.PutItem(ctx, &dynamodb.PutItemInput{
					TableName: aws.String("bench-delete"),
					Item: map[string]ddbtypes.AttributeValue{
						"id":   &ddbtypes.AttributeValueMemberS{Value: "bench-pk"},
						"data": &ddbtypes.AttributeValueMemberS{Value: "to-be-deleted"},
					},
				})
				return err
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newDDBClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
					TableName: aws.String("bench-delete"),
					Key: map[string]ddbtypes.AttributeValue{
						"id": &ddbtypes.AttributeValueMemberS{Value: "bench-pk"},
					},
				})
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "DeleteItemOutput")}
			},
			Teardown: func(ctx context.Context, endpoint string) error {
				return deleteTable(ctx, endpoint, "bench-delete")
			},
		},
	}
}
