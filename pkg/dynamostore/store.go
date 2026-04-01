// Package dynamostore provides a generic single-table DynamoDB store
// with tenant isolation via partition key prefix.
//
// Table design:
//
//	PK: TENANT#{org_id}
//	SK: {FEATURE}#{id}
//	GSI1PK: {FEATURE}      (feature-time-index)
//	GSI1SK: {created_at}
//
// All features share one table; tenant isolation is enforced by scoping
// every read and write to the tenant's partition key.
package dynamostore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// ErrNotFound is returned when a requested item does not exist.
var ErrNotFound = errors.New("dynamostore: not found")

// ErrNoTenant is returned when a tenant context has not been set.
var ErrNoTenant = errors.New("dynamostore: tenant ID not set")

// Item is the raw DynamoDB item shape stored in the table.
type Item struct {
	PK        string `dynamodbav:"pk"`
	SK        string `dynamodbav:"sk"`
	Feature   string `dynamodbav:"feature"`
	Data      string `dynamodbav:"data"` // JSON-encoded payload
	CreatedAt string `dynamodbav:"created_at"`
	UpdatedAt string `dynamodbav:"updated_at"`
	TTL       *int64 `dynamodbav:"ttl,omitempty"`
}

// QueryOpts controls the behaviour of Query.
type QueryOpts struct {
	Limit     int32
	StartTime *time.Time // inclusive lower bound on created_at
	EndTime   *time.Time // exclusive upper bound on created_at
	UseGSI    bool       // if true, queries the feature-time-index GSI
}

// Store is a generic DynamoDB store scoped to a single table.
// Create one Store per application and use WithTenant to produce
// per-request copies bound to a specific tenant.
type Store struct {
	client    *dynamodb.Client
	tableName string
	tenantID  string
}

// New creates a new Store for the given table.
func New(client *dynamodb.Client, tableName string) *Store {
	return &Store{
		client:    client,
		tableName: tableName,
	}
}

// WithTenant returns a shallow copy of the Store bound to the given tenant.
// This must be called per-request from auth middleware.
func (s *Store) WithTenant(tenantID string) *Store {
	return &Store{
		client:    s.client,
		tableName: s.tableName,
		tenantID:  tenantID,
	}
}

// TenantID returns the current tenant context.
func (s *Store) TenantID() string {
	return s.tenantID
}

// pk returns the partition key for the current tenant.
func (s *Store) pk() (string, error) {
	if s.tenantID == "" {
		return "", ErrNoTenant
	}
	return "TENANT#" + s.tenantID, nil
}

// sk returns the sort key for a given feature and ID.
func sk(feature, id string) string {
	return feature + "#" + id
}

// Put stores an item, serialising it to JSON in the data column.
func (s *Store) Put(ctx context.Context, feature, id string, item any) error {
	return s.PutWithTTL(ctx, feature, id, item, nil)
}

// PutWithTTL stores an item with an optional TTL (epoch seconds).
func (s *Store) PutWithTTL(ctx context.Context, feature, id string, item any, ttl *int64) error {
	pk, err := s.pk()
	if err != nil {
		return err
	}

	data, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("dynamostore: marshal: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)

	ddbItem := Item{
		PK:        pk,
		SK:        sk(feature, id),
		Feature:   feature,
		Data:      string(data),
		CreatedAt: now,
		UpdatedAt: now,
		TTL:       ttl,
	}

	av, err := attributevalue.MarshalMap(ddbItem)
	if err != nil {
		return fmt.Errorf("dynamostore: marshal item: %w", err)
	}

	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      av,
	})
	if err != nil {
		return fmt.Errorf("dynamostore: put: %w", err)
	}
	return nil
}

// Get retrieves a single item and unmarshals it into out.
func (s *Store) Get(ctx context.Context, feature, id string, out any) error {
	pk, err := s.pk()
	if err != nil {
		return err
	}

	result, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]ddbtypes.AttributeValue{
			"pk": &ddbtypes.AttributeValueMemberS{Value: pk},
			"sk": &ddbtypes.AttributeValueMemberS{Value: sk(feature, id)},
		},
	})
	if err != nil {
		return fmt.Errorf("dynamostore: get: %w", err)
	}
	if result.Item == nil {
		return ErrNotFound
	}

	var item Item
	if err := attributevalue.UnmarshalMap(result.Item, &item); err != nil {
		return fmt.Errorf("dynamostore: unmarshal item: %w", err)
	}

	return json.Unmarshal([]byte(item.Data), out)
}

// List retrieves all items for a given feature within the current tenant
// and unmarshals them into out (which must be a pointer to a slice).
func (s *Store) List(ctx context.Context, feature string, out any) error {
	pk, err := s.pk()
	if err != nil {
		return err
	}

	skPrefix := feature + "#"

	result, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.tableName),
		KeyConditionExpression: aws.String("pk = :pk AND begins_with(sk, :skPrefix)"),
		ExpressionAttributeValues: map[string]ddbtypes.AttributeValue{
			":pk":       &ddbtypes.AttributeValueMemberS{Value: pk},
			":skPrefix": &ddbtypes.AttributeValueMemberS{Value: skPrefix},
		},
	})
	if err != nil {
		return fmt.Errorf("dynamostore: list: %w", err)
	}

	return s.unmarshalItems(result.Items, out)
}

// Delete removes a single item.
func (s *Store) Delete(ctx context.Context, feature, id string) error {
	pk, err := s.pk()
	if err != nil {
		return err
	}

	_, err = s.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]ddbtypes.AttributeValue{
			"pk": &ddbtypes.AttributeValueMemberS{Value: pk},
			"sk": &ddbtypes.AttributeValueMemberS{Value: sk(feature, id)},
		},
	})
	if err != nil {
		return fmt.Errorf("dynamostore: delete: %w", err)
	}
	return nil
}

// Query retrieves items for a feature, optionally using the GSI and time filters.
func (s *Store) Query(ctx context.Context, feature string, opts QueryOpts) ([]Item, error) {
	if opts.UseGSI {
		return s.queryGSI(ctx, feature, opts)
	}

	// Default: query within the tenant partition.
	pk, err := s.pk()
	if err != nil {
		return nil, err
	}

	skPrefix := feature + "#"
	input := &dynamodb.QueryInput{
		TableName:              aws.String(s.tableName),
		KeyConditionExpression: aws.String("pk = :pk AND begins_with(sk, :skPrefix)"),
		ExpressionAttributeValues: map[string]ddbtypes.AttributeValue{
			":pk":       &ddbtypes.AttributeValueMemberS{Value: pk},
			":skPrefix": &ddbtypes.AttributeValueMemberS{Value: skPrefix},
		},
		ScanIndexForward: aws.Bool(false), // newest first
	}
	if opts.Limit > 0 {
		input.Limit = aws.Int32(opts.Limit)
	}

	// Optional time-range filter on created_at attribute.
	if opts.StartTime != nil || opts.EndTime != nil {
		filter, vals := s.buildTimeFilter(opts)
		input.FilterExpression = aws.String(filter)
		for k, v := range vals {
			input.ExpressionAttributeValues[k] = v
		}
	}

	result, err := s.client.Query(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("dynamostore: query: %w", err)
	}

	return s.marshalRawItems(result.Items)
}

// queryGSI uses the feature-time-index GSI for cross-tenant time-range queries.
func (s *Store) queryGSI(ctx context.Context, feature string, opts QueryOpts) ([]Item, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(s.tableName),
		IndexName:              aws.String("feature-time-index"),
		KeyConditionExpression: aws.String("feature = :f"),
		ExpressionAttributeValues: map[string]ddbtypes.AttributeValue{
			":f": &ddbtypes.AttributeValueMemberS{Value: feature},
		},
		ScanIndexForward: aws.Bool(false),
	}
	if opts.Limit > 0 {
		input.Limit = aws.Int32(opts.Limit)
	}

	// Time-range on the GSI sort key (created_at).
	if opts.StartTime != nil && opts.EndTime != nil {
		input.KeyConditionExpression = aws.String("feature = :f AND created_at BETWEEN :start AND :end")
		input.ExpressionAttributeValues[":start"] = &ddbtypes.AttributeValueMemberS{Value: opts.StartTime.UTC().Format(time.RFC3339)}
		input.ExpressionAttributeValues[":end"] = &ddbtypes.AttributeValueMemberS{Value: opts.EndTime.UTC().Format(time.RFC3339)}
	} else if opts.StartTime != nil {
		input.KeyConditionExpression = aws.String("feature = :f AND created_at >= :start")
		input.ExpressionAttributeValues[":start"] = &ddbtypes.AttributeValueMemberS{Value: opts.StartTime.UTC().Format(time.RFC3339)}
	} else if opts.EndTime != nil {
		input.KeyConditionExpression = aws.String("feature = :f AND created_at < :end")
		input.ExpressionAttributeValues[":end"] = &ddbtypes.AttributeValueMemberS{Value: opts.EndTime.UTC().Format(time.RFC3339)}
	}

	result, err := s.client.Query(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("dynamostore: query gsi: %w", err)
	}

	return s.marshalRawItems(result.Items)
}

// buildTimeFilter creates a filter expression for time-range queries on the primary index.
func (s *Store) buildTimeFilter(opts QueryOpts) (string, map[string]ddbtypes.AttributeValue) {
	vals := make(map[string]ddbtypes.AttributeValue)
	var parts []string

	if opts.StartTime != nil {
		parts = append(parts, "created_at >= :start")
		vals[":start"] = &ddbtypes.AttributeValueMemberS{Value: opts.StartTime.UTC().Format(time.RFC3339)}
	}
	if opts.EndTime != nil {
		parts = append(parts, "created_at < :end")
		vals[":end"] = &ddbtypes.AttributeValueMemberS{Value: opts.EndTime.UTC().Format(time.RFC3339)}
	}

	filter := parts[0]
	if len(parts) == 2 {
		filter = parts[0] + " AND " + parts[1]
	}
	return filter, vals
}

// unmarshalItems deserialises a list of DynamoDB items into a typed slice.
func (s *Store) unmarshalItems(items []map[string]ddbtypes.AttributeValue, out any) error {
	var rawItems []Item
	for _, av := range items {
		var item Item
		if err := attributevalue.UnmarshalMap(av, &item); err != nil {
			return fmt.Errorf("dynamostore: unmarshal: %w", err)
		}
		rawItems = append(rawItems, item)
	}

	// Collect the JSON data blobs and unmarshal them as a JSON array.
	var dataArray []json.RawMessage
	for _, item := range rawItems {
		dataArray = append(dataArray, json.RawMessage(item.Data))
	}

	arrayJSON, err := json.Marshal(dataArray)
	if err != nil {
		return fmt.Errorf("dynamostore: marshal array: %w", err)
	}

	return json.Unmarshal(arrayJSON, out)
}

// marshalRawItems converts DynamoDB attribute maps to Item structs.
func (s *Store) marshalRawItems(items []map[string]ddbtypes.AttributeValue) ([]Item, error) {
	var result []Item
	for _, av := range items {
		var item Item
		if err := attributevalue.UnmarshalMap(av, &item); err != nil {
			return nil, fmt.Errorf("dynamostore: unmarshal: %w", err)
		}
		result = append(result, item)
	}
	return result, nil
}

// UpdateData performs a Put that preserves the original created_at timestamp.
func (s *Store) UpdateData(ctx context.Context, feature, id string, item any) error {
	pk, err := s.pk()
	if err != nil {
		return err
	}

	data, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("dynamostore: marshal: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)

	_, err = s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]ddbtypes.AttributeValue{
			"pk": &ddbtypes.AttributeValueMemberS{Value: pk},
			"sk": &ddbtypes.AttributeValueMemberS{Value: sk(feature, id)},
		},
		UpdateExpression: aws.String("SET #data = :data, updated_at = :now"),
		ExpressionAttributeNames: map[string]string{
			"#data": "data",
		},
		ExpressionAttributeValues: map[string]ddbtypes.AttributeValue{
			":data": &ddbtypes.AttributeValueMemberS{Value: string(data)},
			":now":  &ddbtypes.AttributeValueMemberS{Value: now},
		},
		ConditionExpression: aws.String("attribute_exists(pk)"),
	})
	if err != nil {
		var condErr *ddbtypes.ConditionalCheckFailedException
		if errors.As(err, &condErr) {
			return ErrNotFound
		}
		return fmt.Errorf("dynamostore: update: %w", err)
	}
	return nil
}
