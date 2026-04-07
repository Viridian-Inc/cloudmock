package stress

import (
	"context"
	"fmt"
	"math/rand"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"github.com/Viridian-Inc/cloudmock/benchmarks/awsclient"
	"github.com/Viridian-Inc/cloudmock/benchmarks/harness"
)

type dynamoDBStressSuite struct{}

func NewDynamoDBStressSuite() harness.Suite { return &dynamoDBStressSuite{} }

func (s *dynamoDBStressSuite) Name() string { return "dynamodb-stress" }
func (s *dynamoDBStressSuite) Tier() int    { return 1 }

func newStressDDBClient(endpoint string) (*dynamodb.Client, error) {
	cfg, err := awsclient.NewConfig(endpoint)
	if err != nil {
		return nil, err
	}
	client := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		o.BaseEndpoint = awsclient.Endpoint(endpoint)
	})
	return client, nil
}

func createStressTable(ctx context.Context, endpoint, tableName string) error {
	client, err := newStressDDBClient(endpoint)
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

func createStressTableWithSort(ctx context.Context, endpoint, tableName string) error {
	client, err := newStressDDBClient(endpoint)
	if err != nil {
		return err
	}
	_, err = client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String(tableName),
		AttributeDefinitions: []ddbtypes.AttributeDefinition{
			{
				AttributeName: aws.String("pk"),
				AttributeType: ddbtypes.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("sk"),
				AttributeType: ddbtypes.ScalarAttributeTypeS,
			},
		},
		KeySchema: []ddbtypes.KeySchemaElement{
			{
				AttributeName: aws.String("pk"),
				KeyType:       ddbtypes.KeyTypeHash,
			},
			{
				AttributeName: aws.String("sk"),
				KeyType:       ddbtypes.KeyTypeRange,
			},
		},
		BillingMode: ddbtypes.BillingModePayPerRequest,
	})
	return err
}

func deleteStressTable(ctx context.Context, endpoint, tableName string) error {
	client, err := newStressDDBClient(endpoint)
	if err != nil {
		return err
	}
	_, err = client.DeleteTable(ctx, &dynamodb.DeleteTableInput{
		TableName: aws.String(tableName),
	})
	return err
}

func bulkLoad(ctx context.Context, endpoint, tableName string, count int) error {
	client, err := newStressDDBClient(endpoint)
	if err != nil {
		return err
	}
	for i := 0; i < count; i++ {
		_, err = client.PutItem(ctx, &dynamodb.PutItemInput{
			TableName: aws.String(tableName),
			Item: map[string]ddbtypes.AttributeValue{
				"id":   &ddbtypes.AttributeValueMemberS{Value: fmt.Sprintf("key-%d", i)},
				"data": &ddbtypes.AttributeValueMemberS{Value: fmt.Sprintf("value-%d", i)},
			},
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func bulkLoadWithSort(ctx context.Context, endpoint, tableName, partitionKey string, count int) error {
	client, err := newStressDDBClient(endpoint)
	if err != nil {
		return err
	}
	for i := 0; i < count; i++ {
		_, err = client.PutItem(ctx, &dynamodb.PutItemInput{
			TableName: aws.String(tableName),
			Item: map[string]ddbtypes.AttributeValue{
				"pk":   &ddbtypes.AttributeValueMemberS{Value: partitionKey},
				"sk":   &ddbtypes.AttributeValueMemberS{Value: fmt.Sprintf("sort-%05d", i)},
				"data": &ddbtypes.AttributeValueMemberS{Value: fmt.Sprintf("value-%d", i)},
			},
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *dynamoDBStressSuite) Operations() []harness.Operation {
	const (
		bulkPutTable      = "stress-bulk-put"
		getScaleTable     = "stress-get-scale"
		queryScaleTable   = "stress-query-scale"
		concurrentTable   = "stress-concurrent"
		queryPartitionKey = "stress-pk"
	)

	return []harness.Operation{
		{
			Name: "BulkPutItem",
			Setup: func(ctx context.Context, endpoint string) error {
				return createStressTable(ctx, endpoint, bulkPutTable)
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newStressDDBClient(endpoint)
				if err != nil {
					return nil, err
				}
				var lastOut *dynamodb.PutItemOutput
				for i := 0; i < 1000; i++ {
					out, err := client.PutItem(ctx, &dynamodb.PutItemInput{
						TableName: aws.String(bulkPutTable),
						Item: map[string]ddbtypes.AttributeValue{
							"id":   &ddbtypes.AttributeValueMemberS{Value: fmt.Sprintf("bulk-key-%d", i)},
							"data": &ddbtypes.AttributeValueMemberS{Value: fmt.Sprintf("bulk-value-%d", i)},
						},
					})
					if err != nil {
						return nil, err
					}
					lastOut = out
				}
				return lastOut, nil
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "PutItemOutput")}
			},
			Teardown: func(ctx context.Context, endpoint string) error {
				return deleteStressTable(ctx, endpoint, bulkPutTable)
			},
		},
		{
			Name: "GetItemAtScale",
			Setup: func(ctx context.Context, endpoint string) error {
				if err := createStressTable(ctx, endpoint, getScaleTable); err != nil {
					return err
				}
				return bulkLoad(ctx, endpoint, getScaleTable, 10000)
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newStressDDBClient(endpoint)
				if err != nil {
					return nil, err
				}
				key := fmt.Sprintf("key-%d", rand.Intn(10000))
				return client.GetItem(ctx, &dynamodb.GetItemInput{
					TableName: aws.String(getScaleTable),
					Key: map[string]ddbtypes.AttributeValue{
						"id": &ddbtypes.AttributeValueMemberS{Value: key},
					},
				})
			},
			Validate: func(resp any) []harness.Finding {
				out, ok := resp.(*dynamodb.GetItemOutput)
				if !ok || out == nil {
					return []harness.Finding{harness.CheckNotNil(nil, "GetItemOutput")}
				}
				findings := []harness.Finding{harness.CheckNotNil(out, "GetItemOutput")}
				if len(out.Item) == 0 {
					findings = append(findings, harness.Finding{
						Field:    "Item",
						Expected: "<non-empty>",
						Actual:   "<empty>",
						Grade:    harness.GradeFail,
					})
				} else {
					findings = append(findings, harness.Finding{
						Field:    "Item",
						Expected: "<non-empty>",
						Actual:   "<non-empty>",
						Grade:    harness.GradePass,
					})
				}
				return findings
			},
			Teardown: func(ctx context.Context, endpoint string) error {
				return deleteStressTable(ctx, endpoint, getScaleTable)
			},
		},
		{
			Name: "QueryAtScale",
			Setup: func(ctx context.Context, endpoint string) error {
				if err := createStressTableWithSort(ctx, endpoint, queryScaleTable); err != nil {
					return err
				}
				return bulkLoadWithSort(ctx, endpoint, queryScaleTable, queryPartitionKey, 10000)
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newStressDDBClient(endpoint)
				if err != nil {
					return nil, err
				}
				return client.Query(ctx, &dynamodb.QueryInput{
					TableName:              aws.String(queryScaleTable),
					KeyConditionExpression: aws.String("#pk = :pkval"),
					ExpressionAttributeNames: map[string]string{
						"#pk": "pk",
					},
					ExpressionAttributeValues: map[string]ddbtypes.AttributeValue{
						":pkval": &ddbtypes.AttributeValueMemberS{Value: queryPartitionKey},
					},
					Limit: aws.Int32(100),
				})
			},
			Validate: func(resp any) []harness.Finding {
				out, ok := resp.(*dynamodb.QueryOutput)
				if !ok || out == nil {
					return []harness.Finding{harness.CheckNotNil(nil, "QueryOutput")}
				}
				findings := []harness.Finding{harness.CheckNotNil(out, "QueryOutput")}
				if out.Count != 100 {
					findings = append(findings, harness.Finding{
						Field:    "Count",
						Expected: "100",
						Actual:   fmt.Sprintf("%d", out.Count),
						Grade:    harness.GradeFail,
					})
				} else {
					findings = append(findings, harness.Finding{
						Field:    "Count",
						Expected: "100",
						Actual:   fmt.Sprintf("%d", out.Count),
						Grade:    harness.GradePass,
					})
				}
				return findings
			},
			Teardown: func(ctx context.Context, endpoint string) error {
				return deleteStressTable(ctx, endpoint, queryScaleTable)
			},
		},
		{
			Name: "ConcurrentMixedOps",
			Setup: func(ctx context.Context, endpoint string) error {
				if err := createStressTable(ctx, endpoint, concurrentTable); err != nil {
					return err
				}
				return bulkLoad(ctx, endpoint, concurrentTable, 1000)
			},
			Run: func(ctx context.Context, endpoint string) (any, error) {
				client, err := newStressDDBClient(endpoint)
				if err != nil {
					return nil, err
				}

				const goroutines = 10
				var wg sync.WaitGroup
				errs := make([]error, goroutines)

				for i := 0; i < goroutines; i++ {
					wg.Add(1)
					go func(idx int) {
						defer wg.Done()
						op := rand.Intn(3)
						switch op {
						case 0: // Get
							key := fmt.Sprintf("key-%d", rand.Intn(1000))
							_, errs[idx] = client.GetItem(ctx, &dynamodb.GetItemInput{
								TableName: aws.String(concurrentTable),
								Key: map[string]ddbtypes.AttributeValue{
									"id": &ddbtypes.AttributeValueMemberS{Value: key},
								},
							})
						case 1: // Put
							_, errs[idx] = client.PutItem(ctx, &dynamodb.PutItemInput{
								TableName: aws.String(concurrentTable),
								Item: map[string]ddbtypes.AttributeValue{
									"id":   &ddbtypes.AttributeValueMemberS{Value: uuid.New().String()},
									"data": &ddbtypes.AttributeValueMemberS{Value: "concurrent-write"},
								},
							})
						case 2: // Delete
							key := fmt.Sprintf("key-%d", rand.Intn(1000))
							_, errs[idx] = client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
								TableName: aws.String(concurrentTable),
								Key: map[string]ddbtypes.AttributeValue{
									"id": &ddbtypes.AttributeValueMemberS{Value: key},
								},
							})
						}
					}(i)
				}

				wg.Wait()

				for _, e := range errs {
					if e != nil {
						return nil, e
					}
				}

				return &struct{ OK bool }{OK: true}, nil
			},
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, "ConcurrentMixedOpsResult")}
			},
			Teardown: func(ctx context.Context, endpoint string) error {
				return deleteStressTable(ctx, endpoint, concurrentTable)
			},
		},
	}
}
