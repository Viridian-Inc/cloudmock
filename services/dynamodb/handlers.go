package dynamodb

import (
	"encoding/json"
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// ---- JSON request/response types ----

type createTableRequest struct {
	TableName              string                 `json:"TableName"`
	KeySchema              []KeySchemaElement      `json:"KeySchema"`
	AttributeDefinitions   []AttributeDefinition   `json:"AttributeDefinitions"`
	BillingMode            string                  `json:"BillingMode"`
	ProvisionedThroughput  *ProvisionedThroughput  `json:"ProvisionedThroughput"`
	GlobalSecondaryIndexes []GSI                   `json:"GlobalSecondaryIndexes"`
	LocalSecondaryIndexes  []LSI                   `json:"LocalSecondaryIndexes"`
	StreamSpecification    *StreamSpecification    `json:"StreamSpecification"`
}

type gsiDescription struct {
	IndexName             string                 `json:"IndexName"`
	KeySchema             []KeySchemaElement      `json:"KeySchema"`
	Projection            map[string]any `json:"Projection"`
	IndexStatus           string                 `json:"IndexStatus"`
	ItemCount             int64                  `json:"ItemCount"`
	IndexSizeBytes        int64                  `json:"IndexSizeBytes"`
	IndexArn              string                 `json:"IndexArn"`
	ProvisionedThroughput *ProvisionedThroughput `json:"ProvisionedThroughput,omitempty"`
}

type lsiDescription struct {
	IndexName      string                 `json:"IndexName"`
	KeySchema      []KeySchemaElement      `json:"KeySchema"`
	Projection     map[string]any `json:"Projection"`
	ItemCount      int64                  `json:"ItemCount"`
	IndexSizeBytes int64                  `json:"IndexSizeBytes"`
	IndexArn       string                 `json:"IndexArn"`
}

// warmThroughput is the WarmThroughput sub-struct returned in DescribeTable.
// The Terraform AWS provider v6 polls WarmThroughput.Status when on_demand_throughput
// is configured; it must return "ACTIVE" to unblock the creation waiter.
type warmThroughput struct {
	ReadUnitsPerSecond  int64  `json:"ReadUnitsPerSecond,omitempty"`
	WriteUnitsPerSecond int64  `json:"WriteUnitsPerSecond,omitempty"`
	Status              string `json:"Status"`
}

type tableDescription struct {
	TableName              string                 `json:"TableName"`
	TableStatus            string                 `json:"TableStatus"`
	KeySchema              []KeySchemaElement      `json:"KeySchema"`
	AttributeDefinitions   []AttributeDefinition   `json:"AttributeDefinitions"`
	CreationDateTime       float64                `json:"CreationDateTime"`
	ItemCount              int64                  `json:"ItemCount"`
	TableSizeBytes         int64                  `json:"TableSizeBytes"`
	TableArn               string                 `json:"TableArn"`
	BillingModeSummary     *billingModeSummary    `json:"BillingModeSummary,omitempty"`
	ProvisionedThroughput  *ProvisionedThroughput `json:"ProvisionedThroughput,omitempty"`
	GlobalSecondaryIndexes []gsiDescription       `json:"GlobalSecondaryIndexes,omitempty"`
	LocalSecondaryIndexes  []lsiDescription       `json:"LocalSecondaryIndexes,omitempty"`
	StreamSpecification    *StreamSpecification   `json:"StreamSpecification,omitempty"`
	LatestStreamArn        string                 `json:"LatestStreamArn,omitempty"`
	LatestStreamLabel      string                 `json:"LatestStreamLabel,omitempty"`
	// WarmThroughput is always returned as ACTIVE in CloudMock (no async warm-up period).
	// The Terraform AWS provider v6 polls this field when on_demand_throughput is set.
	WarmThroughput         *warmThroughput        `json:"WarmThroughput,omitempty"`
}

type billingModeSummary struct {
	BillingMode string `json:"BillingMode"`
}

type createTableResponse struct {
	TableDescription tableDescription `json:"TableDescription"`
}

type deleteTableRequest struct {
	TableName string `json:"TableName"`
}

type deleteTableResponse struct {
	TableDescription tableDescription `json:"TableDescription"`
}

// updateTableRequest covers the subset of UpdateTable fields that CloudMock handles.
// Fields related to on_demand_throughput, GSI updates, stream changes, etc. are accepted
// but treated as no-ops (CloudMock is always ACTIVE and doesn't enforce capacity limits).
type updateTableRequest struct {
	TableName                  string                 `json:"TableName"`
	BillingMode                string                 `json:"BillingMode,omitempty"`
	ProvisionedThroughput      *ProvisionedThroughput `json:"ProvisionedThroughput,omitempty"`
	AttributeDefinitions       []AttributeDefinition  `json:"AttributeDefinitions,omitempty"`
	// The following fields are parsed but largely ignored — CloudMock stays ACTIVE.
	GlobalSecondaryIndexUpdates []any                 `json:"GlobalSecondaryIndexUpdates,omitempty"`
	StreamSpecification         *StreamSpecification  `json:"StreamSpecification,omitempty"`
	OnDemandThroughput          map[string]any        `json:"OnDemandThroughput,omitempty"`
	TableClass                  string                `json:"TableClass,omitempty"`
	DeletionProtectionEnabled   *bool                 `json:"DeletionProtectionEnabled,omitempty"`
}

type updateTableResponse struct {
	TableDescription tableDescription `json:"TableDescription"`
}

type putResourcePolicyRequest struct {
	ResourceArn string `json:"ResourceArn"`
	Policy      string `json:"Policy"`
}

type putResourcePolicyResponse struct {
	RevisionId string `json:"RevisionId"`
}

type getResourcePolicyRequest struct {
	ResourceArn string `json:"ResourceArn"`
}

type getResourcePolicyResponse struct {
	Policy     string `json:"Policy"`
	RevisionId string `json:"RevisionId"`
}

type deleteResourcePolicyRequest struct {
	ResourceArn string `json:"ResourceArn"`
}

type deleteResourcePolicyResponse struct {
	RevisionId string `json:"RevisionId"`
}

type describeTableRequest struct {
	TableName string `json:"TableName"`
}

type describeTableResponse struct {
	Table tableDescription `json:"Table"`
}

type listTablesResponse struct {
	TableNames []string `json:"TableNames"`
}

type putItemRequest struct {
	TableName                 string                    `json:"TableName"`
	Item                      Item                      `json:"Item"`
	ConditionExpression       string                    `json:"ConditionExpression"`
	ExpressionAttributeNames  map[string]string         `json:"ExpressionAttributeNames"`
	ExpressionAttributeValues map[string]AttributeValue `json:"ExpressionAttributeValues"`
}

type getItemRequest struct {
	TableName                string            `json:"TableName"`
	Key                      Item              `json:"Key"`
	ProjectionExpression     string            `json:"ProjectionExpression"`
	ExpressionAttributeNames map[string]string `json:"ExpressionAttributeNames"`
}

type getItemResponse struct {
	Item Item `json:"Item,omitempty"`
}

type deleteItemRequest struct {
	TableName                 string                    `json:"TableName"`
	Key                       Item                      `json:"Key"`
	ConditionExpression       string                    `json:"ConditionExpression"`
	ExpressionAttributeNames  map[string]string         `json:"ExpressionAttributeNames"`
	ExpressionAttributeValues map[string]AttributeValue `json:"ExpressionAttributeValues"`
}

type updateItemRequest struct {
	TableName                 string                     `json:"TableName"`
	Key                       Item                       `json:"Key"`
	UpdateExpression          string                     `json:"UpdateExpression"`
	ExpressionAttributeNames  map[string]string          `json:"ExpressionAttributeNames"`
	ExpressionAttributeValues map[string]AttributeValue  `json:"ExpressionAttributeValues"`
	ReturnValues              string                     `json:"ReturnValues"`
}

type updateItemResponse struct {
	Attributes Item `json:"Attributes,omitempty"`
}

type queryRequest struct {
	TableName                 string                     `json:"TableName"`
	IndexName                 string                     `json:"IndexName"`
	KeyConditionExpression    string                     `json:"KeyConditionExpression"`
	FilterExpression          string                     `json:"FilterExpression"`
	ProjectionExpression      string                     `json:"ProjectionExpression"`
	ExpressionAttributeNames  map[string]string          `json:"ExpressionAttributeNames"`
	ExpressionAttributeValues map[string]AttributeValue  `json:"ExpressionAttributeValues"`
	ScanIndexForward          *bool                      `json:"ScanIndexForward"`
	Limit                     int                        `json:"Limit"`
}

type queryResponse struct {
	Items        []Item `json:"Items"`
	Count        int    `json:"Count"`
	ScannedCount int    `json:"ScannedCount"`
}

type scanRequest struct {
	TableName                 string                     `json:"TableName"`
	FilterExpression          string                     `json:"FilterExpression"`
	ProjectionExpression      string                     `json:"ProjectionExpression"`
	ExpressionAttributeNames  map[string]string          `json:"ExpressionAttributeNames"`
	ExpressionAttributeValues map[string]AttributeValue  `json:"ExpressionAttributeValues"`
	Limit                     int                        `json:"Limit"`
}

type scanResponse struct {
	Items        []Item `json:"Items"`
	Count        int    `json:"Count"`
	ScannedCount int    `json:"ScannedCount"`
}

type batchGetItemRequest struct {
	RequestItems map[string]batchGetTableRequest `json:"RequestItems"`
}

type batchGetTableRequest struct {
	Keys                     []Item            `json:"Keys"`
	ProjectionExpression     string            `json:"ProjectionExpression"`
	ExpressionAttributeNames map[string]string `json:"ExpressionAttributeNames"`
}

type batchGetItemResponse struct {
	Responses        map[string][]Item                    `json:"Responses"`
	UnprocessedKeys  map[string]batchGetTableRequest      `json:"UnprocessedKeys"`
}

type batchWriteItemRequest struct {
	RequestItems map[string][]writeRequest `json:"RequestItems"`
}

type writeRequest struct {
	PutRequest    *putRequest    `json:"PutRequest,omitempty"`
	DeleteRequest *deleteRequest `json:"DeleteRequest,omitempty"`
}

type putRequest struct {
	Item Item `json:"Item"`
}

type deleteRequest struct {
	Key Item `json:"Key"`
}

type batchWriteItemResponse struct {
	UnprocessedItems map[string][]writeRequest `json:"UnprocessedItems"`
}

// ---- Transaction types ----

type transactWriteItemsRequest struct {
	TransactItems []transactWriteItem `json:"TransactItems"`
}

type transactWriteItem struct {
	Put            *transactPut            `json:"Put,omitempty"`
	Delete         *transactDelete         `json:"Delete,omitempty"`
	Update         *transactUpdate         `json:"Update,omitempty"`
	ConditionCheck *transactConditionCheck `json:"ConditionCheck,omitempty"`
}

type transactPut struct {
	TableName                 string                    `json:"TableName"`
	Item                      Item                      `json:"Item"`
	ConditionExpression       string                    `json:"ConditionExpression"`
	ExpressionAttributeNames  map[string]string         `json:"ExpressionAttributeNames"`
	ExpressionAttributeValues map[string]AttributeValue `json:"ExpressionAttributeValues"`
}

type transactDelete struct {
	TableName                 string                    `json:"TableName"`
	Key                       Item                      `json:"Key"`
	ConditionExpression       string                    `json:"ConditionExpression"`
	ExpressionAttributeNames  map[string]string         `json:"ExpressionAttributeNames"`
	ExpressionAttributeValues map[string]AttributeValue `json:"ExpressionAttributeValues"`
}

type transactUpdate struct {
	TableName                 string                    `json:"TableName"`
	Key                       Item                      `json:"Key"`
	UpdateExpression          string                    `json:"UpdateExpression"`
	ConditionExpression       string                    `json:"ConditionExpression"`
	ExpressionAttributeNames  map[string]string         `json:"ExpressionAttributeNames"`
	ExpressionAttributeValues map[string]AttributeValue `json:"ExpressionAttributeValues"`
}

type transactConditionCheck struct {
	TableName                 string                    `json:"TableName"`
	Key                       Item                      `json:"Key"`
	ConditionExpression       string                    `json:"ConditionExpression"`
	ExpressionAttributeNames  map[string]string         `json:"ExpressionAttributeNames"`
	ExpressionAttributeValues map[string]AttributeValue `json:"ExpressionAttributeValues"`
}

type transactGetItemsRequest struct {
	TransactItems []transactGetItem `json:"TransactItems"`
}

type transactGetItem struct {
	Get *transactGet `json:"Get"`
}

type transactGet struct {
	TableName                string            `json:"TableName"`
	Key                      Item              `json:"Key"`
	ProjectionExpression     string            `json:"ProjectionExpression"`
	ExpressionAttributeNames map[string]string `json:"ExpressionAttributeNames"`
}

type transactGetItemsResponse struct {
	Responses []transactGetResponse `json:"Responses"`
}

type transactGetResponse struct {
	Item Item `json:"Item,omitempty"`
}

type cancellationReason struct {
	Code    string `json:"Code"`
	Message string `json:"Message,omitempty"`
}

// ---- helpers ----

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       body,
		Format:     service.FormatJSON,
	}, nil
}

func jsonErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
}

func parseJSON(body []byte, v any) *service.AWSError {
	if len(body) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, v); err != nil {
		return service.NewAWSError("ValidationException",
			"Request body is not valid JSON.", http.StatusBadRequest)
	}
	return nil
}

func tableToDescription(t *Table, arn string) tableDescription {
	desc := tableDescription{
		TableName:            t.Name,
		TableStatus:          t.Status,
		KeySchema:            t.KeySchema,
		AttributeDefinitions: t.AttributeDefinitions,
		CreationDateTime:     t.CreationDateTime,
		ItemCount:            t.ItemCount,
		TableArn:             arn,
		// Always report WarmThroughput as ACTIVE — CloudMock has no async warm-up.
		// This prevents the Terraform AWS provider v6 waiter from timing out when
		// on_demand_throughput is configured.
		WarmThroughput: &warmThroughput{Status: "ACTIVE"},
	}
	if t.BillingMode != "" {
		desc.BillingModeSummary = &billingModeSummary{BillingMode: t.BillingMode}
	}
	if t.ProvisionedThroughput != nil {
		desc.ProvisionedThroughput = t.ProvisionedThroughput
	}
	for _, gsi := range t.GSIs {
		itemCount := int64(len(t.GSIItems[gsi.IndexName]))
		desc.GlobalSecondaryIndexes = append(desc.GlobalSecondaryIndexes, gsiDescription{
			IndexName:             gsi.IndexName,
			KeySchema:             gsi.KeySchema,
			Projection:            gsi.Projection,
			IndexStatus:           "ACTIVE",
			ItemCount:             itemCount,
			IndexSizeBytes:        itemCount * 100, // approximate
			IndexArn:              arn + "/index/" + gsi.IndexName,
			ProvisionedThroughput: gsi.ProvisionedThroughput,
		})
	}
	for _, lsi := range t.LSIs {
		itemCount := int64(len(t.LSIItems[lsi.IndexName]))
		desc.LocalSecondaryIndexes = append(desc.LocalSecondaryIndexes, lsiDescription{
			IndexName:      lsi.IndexName,
			KeySchema:      lsi.KeySchema,
			Projection:     lsi.Projection,
			ItemCount:      itemCount,
			IndexSizeBytes: itemCount * 100,
			IndexArn:       arn + "/index/" + lsi.IndexName,
		})
	}
	if t.Stream != nil {
		sd := t.Stream.describe()
		desc.StreamSpecification = &StreamSpecification{
			StreamEnabled:  true,
			StreamViewType: sd.StreamViewType,
		}
		desc.LatestStreamArn = sd.StreamARN
		desc.LatestStreamLabel = sd.StreamLabel
	}
	return desc
}

// ---- handlers ----

func handleCreateTable(ctx *service.RequestContext, store *TableStore) (*service.Response, error) {
	var req createTableRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.TableName == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"TableName is required.", http.StatusBadRequest))
	}
	if len(req.KeySchema) == 0 {
		return jsonErr(service.NewAWSError("ValidationException",
			"KeySchema is required.", http.StatusBadRequest))
	}

	table, awsErr := store.CreateTable(req.TableName, req.KeySchema, req.AttributeDefinitions, req.BillingMode, req.ProvisionedThroughput, req.GlobalSecondaryIndexes, req.LocalSecondaryIndexes, req.StreamSpecification)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(createTableResponse{
		TableDescription: tableToDescription(table, store.tableARN(req.TableName)),
	})
}

func handleDeleteTable(ctx *service.RequestContext, store *TableStore) (*service.Response, error) {
	var req deleteTableRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.TableName == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"TableName is required.", http.StatusBadRequest))
	}

	table, awsErr := store.DeleteTable(req.TableName)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(deleteTableResponse{
		TableDescription: tableToDescription(table, store.tableARN(req.TableName)),
	})
}

func handleDescribeTable(ctx *service.RequestContext, store *TableStore) (*service.Response, error) {
	var req describeTableRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.TableName == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"TableName is required.", http.StatusBadRequest))
	}

	table, awsErr := store.DescribeTable(req.TableName)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(describeTableResponse{
		Table: tableToDescription(table, store.tableARN(req.TableName)),
	})
}

func handleUpdateTable(ctx *service.RequestContext, store *TableStore) (*service.Response, error) {
	var req updateTableRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.TableName == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"TableName is required.", http.StatusBadRequest))
	}

	table, awsErr := store.UpdateTable(req.TableName, req.BillingMode, req.ProvisionedThroughput, req.AttributeDefinitions)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(updateTableResponse{
		TableDescription: tableToDescription(table, store.tableARN(req.TableName)),
	})
}

func handleListTables(_ *service.RequestContext, store *TableStore) (*service.Response, error) {
	names := store.ListTables()
	return jsonOK(listTablesResponse{TableNames: names})
}

func handlePutItem(ctx *service.RequestContext, store *TableStore) (*service.Response, error) {
	var req putItemRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.TableName == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"TableName is required.", http.StatusBadRequest))
	}

	if awsErr := store.PutItem(req.TableName, req.Item, req.ConditionExpression); awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(struct{}{})
}

func handleGetItem(ctx *service.RequestContext, store *TableStore) (*service.Response, error) {
	var req getItemRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.TableName == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"TableName is required.", http.StatusBadRequest))
	}

	item, awsErr := store.GetItem(req.TableName, req.Key, req.ProjectionExpression, req.ExpressionAttributeNames)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(getItemResponse{Item: item})
}

func handleDeleteItem(ctx *service.RequestContext, store *TableStore) (*service.Response, error) {
	var req deleteItemRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.TableName == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"TableName is required.", http.StatusBadRequest))
	}

	if awsErr := store.DeleteItem(req.TableName, req.Key, req.ConditionExpression); awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(struct{}{})
}

func handleUpdateItem(ctx *service.RequestContext, store *TableStore) (*service.Response, error) {
	var req updateItemRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.TableName == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"TableName is required.", http.StatusBadRequest))
	}

	attrs, awsErr := store.UpdateItem(req.TableName, req.Key, req.UpdateExpression, req.ExpressionAttributeNames, req.ExpressionAttributeValues, req.ReturnValues)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(updateItemResponse{Attributes: attrs})
}

func handleQuery(ctx *service.RequestContext, store *TableStore) (*service.Response, error) {
	var req queryRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.TableName == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"TableName is required.", http.StatusBadRequest))
	}
	if req.KeyConditionExpression == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"KeyConditionExpression is required.", http.StatusBadRequest))
	}

	items, count, scanned, awsErr := store.Query(req.TableName, req.IndexName, req.KeyConditionExpression, req.FilterExpression, req.ProjectionExpression, req.ExpressionAttributeNames, req.ExpressionAttributeValues, req.ScanIndexForward, req.Limit)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	if items == nil {
		items = []Item{}
	}

	return jsonOK(queryResponse{
		Items:        items,
		Count:        count,
		ScannedCount: scanned,
	})
}

func handleScan(ctx *service.RequestContext, store *TableStore) (*service.Response, error) {
	var req scanRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.TableName == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"TableName is required.", http.StatusBadRequest))
	}

	items, count, scanned, awsErr := store.Scan(req.TableName, req.FilterExpression, req.ProjectionExpression, req.ExpressionAttributeNames, req.ExpressionAttributeValues, req.Limit)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	if items == nil {
		items = []Item{}
	}

	return jsonOK(scanResponse{
		Items:        items,
		Count:        count,
		ScannedCount: scanned,
	})
}

func handleBatchGetItem(ctx *service.RequestContext, store *TableStore) (*service.Response, error) {
	var req batchGetItemRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	responses := make(map[string][]Item)
	for tableName, tableReq := range req.RequestItems {
		var items []Item
		for _, key := range tableReq.Keys {
			item, awsErr := store.GetItem(tableName, key, tableReq.ProjectionExpression, tableReq.ExpressionAttributeNames)
			if awsErr != nil {
				return jsonErr(awsErr)
			}
			if item != nil {
				items = append(items, item)
			}
		}
		if items == nil {
			items = []Item{}
		}
		responses[tableName] = items
	}

	return jsonOK(batchGetItemResponse{
		Responses:       responses,
		UnprocessedKeys: map[string]batchGetTableRequest{},
	})
}

func handleBatchWriteItem(ctx *service.RequestContext, store *TableStore) (*service.Response, error) {
	var req batchWriteItemRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	for tableName, requests := range req.RequestItems {
		for _, wr := range requests {
			if wr.PutRequest != nil {
				if awsErr := store.PutItem(tableName, wr.PutRequest.Item); awsErr != nil {
					return jsonErr(awsErr)
				}
			}
			if wr.DeleteRequest != nil {
				if awsErr := store.DeleteItem(tableName, wr.DeleteRequest.Key); awsErr != nil {
					return jsonErr(awsErr)
				}
			}
		}
	}

	return jsonOK(batchWriteItemResponse{
		UnprocessedItems: map[string][]writeRequest{},
	})
}

func handleTransactWriteItems(ctx *service.RequestContext, store *TableStore) (*service.Response, error) {
	var req transactWriteItemsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if len(req.TransactItems) == 0 {
		return jsonErr(service.NewAWSError("ValidationException",
			"TransactItems is required and must not be empty.", http.StatusBadRequest))
	}

	if awsErr := store.TransactWriteItems(req.TransactItems); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(struct{}{})
}

func handleTransactGetItems(ctx *service.RequestContext, store *TableStore) (*service.Response, error) {
	var req transactGetItemsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if len(req.TransactItems) == 0 {
		return jsonErr(service.NewAWSError("ValidationException",
			"TransactItems is required and must not be empty.", http.StatusBadRequest))
	}

	responses, awsErr := store.TransactGetItems(req.TransactItems)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(transactGetItemsResponse{Responses: responses})
}

func handlePutResourcePolicy(ctx *service.RequestContext, store *TableStore) (*service.Response, error) {
	var req putResourcePolicyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ResourceArn == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"ResourceArn is required.", http.StatusBadRequest))
	}
	table, awsErr := store.PutResourcePolicy(req.ResourceArn, req.Policy)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(putResourcePolicyResponse{RevisionId: table.ResourcePolicyRevisionID})
}

func handleGetResourcePolicy(ctx *service.RequestContext, store *TableStore) (*service.Response, error) {
	var req getResourcePolicyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ResourceArn == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"ResourceArn is required.", http.StatusBadRequest))
	}
	table, awsErr := store.GetResourcePolicy(req.ResourceArn)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(getResourcePolicyResponse{
		Policy:     table.ResourcePolicy,
		RevisionId: table.ResourcePolicyRevisionID,
	})
}

func handleDeleteResourcePolicy(ctx *service.RequestContext, store *TableStore) (*service.Response, error) {
	var req deleteResourcePolicyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ResourceArn == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"ResourceArn is required.", http.StatusBadRequest))
	}
	if awsErr := store.DeleteResourcePolicy(req.ResourceArn); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(deleteResourcePolicyResponse{})
}

// continuousBackupsDescription is the response body for DescribeContinuousBackups.
// CloudMock always reports PITR as disabled and backups as ENABLED (available).
type continuousBackupsDescription struct {
	ContinuousBackupsStatus          string `json:"ContinuousBackupsStatus"`
	PointInTimeRecoveryDescription   struct {
		PointInTimeRecoveryStatus string `json:"PointInTimeRecoveryStatus"`
	} `json:"PointInTimeRecoveryDescription"`
}

type describeContinuousBackupsRequest struct {
	TableName string `json:"TableName"`
}

type describeContinuousBackupsResponse struct {
	ContinuousBackupsDescription continuousBackupsDescription `json:"ContinuousBackupsDescription"`
}

type updateContinuousBackupsRequest struct {
	TableName                  string `json:"TableName"`
	PointInTimeRecoverySpecification struct {
		PointInTimeRecoveryEnabled bool `json:"PointInTimeRecoveryEnabled"`
	} `json:"PointInTimeRecoverySpecification"`
}

func handleDescribeContinuousBackups(ctx *service.RequestContext, store *TableStore) (*service.Response, error) {
	var req describeContinuousBackupsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.TableName == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"TableName is required.", http.StatusBadRequest))
	}
	// Verify table exists.
	if _, awsErr := store.DescribeTable(req.TableName); awsErr != nil {
		return jsonErr(awsErr)
	}
	desc := continuousBackupsDescription{
		ContinuousBackupsStatus: "ENABLED",
	}
	desc.PointInTimeRecoveryDescription.PointInTimeRecoveryStatus = "DISABLED"
	return jsonOK(describeContinuousBackupsResponse{ContinuousBackupsDescription: desc})
}

func handleUpdateContinuousBackups(ctx *service.RequestContext, store *TableStore) (*service.Response, error) {
	var req updateContinuousBackupsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.TableName == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"TableName is required.", http.StatusBadRequest))
	}
	// Verify table exists.
	if _, awsErr := store.DescribeTable(req.TableName); awsErr != nil {
		return jsonErr(awsErr)
	}
	status := "DISABLED"
	if req.PointInTimeRecoverySpecification.PointInTimeRecoveryEnabled {
		status = "ENABLED"
	}
	desc := continuousBackupsDescription{
		ContinuousBackupsStatus: "ENABLED",
	}
	desc.PointInTimeRecoveryDescription.PointInTimeRecoveryStatus = status
	return jsonOK(describeContinuousBackupsResponse{ContinuousBackupsDescription: desc})
}

// ---- DynamoDB tagging handlers ----

type tag struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

type listTagsOfResourceRequest struct {
	ResourceArn string `json:"ResourceArn"`
}

type listTagsOfResourceResponse struct {
	Tags []tag `json:"Tags"`
}

type tagResourceRequest struct {
	ResourceArn string `json:"ResourceArn"`
	Tags        []tag  `json:"Tags"`
}

type untagResourceRequest struct {
	ResourceArn string   `json:"ResourceArn"`
	TagKeys     []string `json:"TagKeys"`
}

func tableNameFromARN(arn string) string {
	// arn:aws:dynamodb:region:account:table/name
	for i := len(arn) - 1; i >= 0; i-- {
		if arn[i] == '/' {
			return arn[i+1:]
		}
	}
	return arn
}

func handleListTagsOfResource(ctx *service.RequestContext, store *TableStore) (*service.Response, error) {
	var req listTagsOfResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	tableName := tableNameFromARN(req.ResourceArn)
	table, awsErr := store.DescribeTable(tableName)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	var tags []tag
	for k, v := range table.Tags {
		tags = append(tags, tag{Key: k, Value: v})
	}
	if tags == nil {
		tags = []tag{}
	}
	return jsonOK(listTagsOfResourceResponse{Tags: tags})
}

func handleTagResource(ctx *service.RequestContext, store *TableStore) (*service.Response, error) {
	var req tagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	tableName := tableNameFromARN(req.ResourceArn)
	table, awsErr := store.DescribeTable(tableName)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	if table.Tags == nil {
		table.Tags = make(map[string]string)
	}
	for _, t := range req.Tags {
		table.Tags[t.Key] = t.Value
	}
	return jsonOK(struct{}{})
}

func handleUntagResource(ctx *service.RequestContext, store *TableStore) (*service.Response, error) {
	var req untagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	tableName := tableNameFromARN(req.ResourceArn)
	table, awsErr := store.DescribeTable(tableName)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	for _, k := range req.TagKeys {
		delete(table.Tags, k)
	}
	return jsonOK(struct{}{})
}
