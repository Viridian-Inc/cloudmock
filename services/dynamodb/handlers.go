package dynamodb

import (
	"encoding/json"
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// ---- JSON request/response types ----

type createTableRequest struct {
	TableName             string                 `json:"TableName"`
	KeySchema             []KeySchemaElement      `json:"KeySchema"`
	AttributeDefinitions  []AttributeDefinition   `json:"AttributeDefinitions"`
	BillingMode           string                  `json:"BillingMode"`
	ProvisionedThroughput *ProvisionedThroughput  `json:"ProvisionedThroughput"`
}

type tableDescription struct {
	TableName             string                 `json:"TableName"`
	TableStatus           string                 `json:"TableStatus"`
	KeySchema             []KeySchemaElement      `json:"KeySchema"`
	AttributeDefinitions  []AttributeDefinition   `json:"AttributeDefinitions"`
	CreationDateTime      float64                `json:"CreationDateTime"`
	ItemCount             int64                  `json:"ItemCount"`
	TableArn              string                 `json:"TableArn"`
	BillingModeSummary    *billingModeSummary    `json:"BillingModeSummary,omitempty"`
	ProvisionedThroughput *ProvisionedThroughput `json:"ProvisionedThroughput,omitempty"`
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
	TableName string `json:"TableName"`
	Item      Item   `json:"Item"`
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
	TableName string `json:"TableName"`
	Key       Item   `json:"Key"`
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

// ---- helpers ----

func jsonOK(body interface{}) (*service.Response, error) {
	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       body,
		Format:     service.FormatJSON,
	}, nil
}

func jsonErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
}

func parseJSON(body []byte, v interface{}) *service.AWSError {
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
	}
	if t.BillingMode != "" {
		desc.BillingModeSummary = &billingModeSummary{BillingMode: t.BillingMode}
	}
	if t.ProvisionedThroughput != nil {
		desc.ProvisionedThroughput = t.ProvisionedThroughput
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

	table, awsErr := store.CreateTable(req.TableName, req.KeySchema, req.AttributeDefinitions, req.BillingMode, req.ProvisionedThroughput)
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

	if awsErr := store.PutItem(req.TableName, req.Item); awsErr != nil {
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

	if awsErr := store.DeleteItem(req.TableName, req.Key); awsErr != nil {
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

	items, count, scanned, awsErr := store.Query(req.TableName, req.KeyConditionExpression, req.FilterExpression, req.ProjectionExpression, req.ExpressionAttributeNames, req.ExpressionAttributeValues, req.ScanIndexForward, req.Limit)
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
