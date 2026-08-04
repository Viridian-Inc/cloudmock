package cosmosdb

import (
	"math"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

func TestAccountSQLDatabaseAndContainerLifecycle(t *testing.T) {
	svc := New()

	accountURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.DocumentDB/databaseAccounts/acct-a?api-version=2025-05-01"
	accountPayload := []byte(`{
		"location":"eastus",
		"tags":{"env":"test"},
		"properties":{
			"databaseAccountOfferType":"Standard",
			"locations":[{"locationName":"East US","failoverPriority":0}],
			"consistencyPolicy":{"defaultConsistencyLevel":"Session"}
		}
	}`)
	createAccountResp, err := svc.HandleRequest(cosmosCtx(t, http.MethodPut, accountURL, accountPayload))
	if err != nil {
		t.Fatalf("create account returned error: %v", err)
	}
	if createAccountResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create account status 201, got %d; body=%s", createAccountResp.StatusCode, string(createAccountResp.RawBody))
	}
	account := decodeCosmosResponse(t, createAccountResp)
	if account["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.DocumentDB/databaseAccounts/acct-a" {
		t.Fatalf("unexpected account id: %v", account["id"])
	}
	if account["name"] != "acct-a" || account["type"] != "Microsoft.DocumentDB/databaseAccounts" || account["location"] != "eastus" {
		t.Fatalf("unexpected account identity fields: %v", account)
	}
	accountProps := account["properties"].(map[string]any)
	if accountProps["provisioningState"] != "Succeeded" || accountProps["documentEndpoint"] != "https://acct-a.documents.azure.com:443/" {
		t.Fatalf("unexpected account properties: %v", accountProps)
	}

	listAccountsResp, err := svc.HandleRequest(cosmosCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.DocumentDB/databaseAccounts?api-version=2025-05-01", nil))
	if err != nil {
		t.Fatalf("list accounts returned error: %v", err)
	}
	listedAccounts := decodeCosmosResponse(t, listAccountsResp)
	if len(listedAccounts["value"].([]any)) != 1 {
		t.Fatalf("expected one account in list, got %v", listedAccounts)
	}

	databaseURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.DocumentDB/databaseAccounts/acct-a/sqlDatabases/db-a?api-version=2025-05-01"
	databasePayload := []byte(`{
		"properties":{"resource":{"id":"db-a"}},
		"options":{"throughput":400}
	}`)
	createDatabaseResp, err := svc.HandleRequest(cosmosCtx(t, http.MethodPut, databaseURL, databasePayload))
	if err != nil {
		t.Fatalf("create database returned error: %v", err)
	}
	if createDatabaseResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create database status 201, got %d; body=%s", createDatabaseResp.StatusCode, string(createDatabaseResp.RawBody))
	}
	database := decodeCosmosResponse(t, createDatabaseResp)
	if database["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.DocumentDB/databaseAccounts/acct-a/sqlDatabases/db-a" {
		t.Fatalf("unexpected database id: %v", database["id"])
	}
	if database["name"] != "acct-a/db-a" || database["type"] != "Microsoft.DocumentDB/databaseAccounts/sqlDatabases" {
		t.Fatalf("unexpected database identity fields: %v", database)
	}
	databaseProps := database["properties"].(map[string]any)
	if databaseProps["provisioningState"] != "Succeeded" {
		t.Fatalf("unexpected database properties: %v", databaseProps)
	}

	containerURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.DocumentDB/databaseAccounts/acct-a/sqlDatabases/db-a/containers/container-a?api-version=2025-05-01"
	containerPayload := []byte(`{
		"properties":{
			"resource":{
				"id":"container-a",
				"partitionKey":{"paths":["/pk"],"kind":"Hash"},
				"indexingPolicy":{"indexingMode":"consistent"}
			}
		}
	}`)
	createContainerResp, err := svc.HandleRequest(cosmosCtx(t, http.MethodPut, containerURL, containerPayload))
	if err != nil {
		t.Fatalf("create container returned error: %v", err)
	}
	if createContainerResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create container status 201, got %d; body=%s", createContainerResp.StatusCode, string(createContainerResp.RawBody))
	}
	container := decodeCosmosResponse(t, createContainerResp)
	if container["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.DocumentDB/databaseAccounts/acct-a/sqlDatabases/db-a/containers/container-a" {
		t.Fatalf("unexpected container id: %v", container["id"])
	}
	if container["name"] != "acct-a/db-a/container-a" || container["type"] != "Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers" {
		t.Fatalf("unexpected container identity fields: %v", container)
	}
	containerProps := container["properties"].(map[string]any)
	if containerProps["provisioningState"] != "Succeeded" {
		t.Fatalf("unexpected container properties: %v", containerProps)
	}

	listContainersResp, err := svc.HandleRequest(cosmosCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.DocumentDB/databaseAccounts/acct-a/sqlDatabases/db-a/containers?api-version=2025-05-01", nil))
	if err != nil {
		t.Fatalf("list containers returned error: %v", err)
	}
	listedContainers := decodeCosmosResponse(t, listContainersResp)
	if len(listedContainers["value"].([]any)) != 1 {
		t.Fatalf("expected one container in list, got %v", listedContainers)
	}

	deleteContainerResp, err := svc.HandleRequest(cosmosCtx(t, http.MethodDelete, containerURL, nil))
	if err != nil {
		t.Fatalf("delete container returned error: %v", err)
	}
	if deleteContainerResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete container status 202, got %d; body=%s", deleteContainerResp.StatusCode, string(deleteContainerResp.RawBody))
	}
}

func TestSQLDatabaseAndContainerTemplateProvisioning(t *testing.T) {
	svc := New()

	accountResource := map[string]any{
		"type":     "Microsoft.DocumentDB/databaseAccounts",
		"name":     "acct-a",
		"location": "eastus",
		"tags":     map[string]any{"env": "test"},
		"properties": map[string]any{
			"databaseAccountOfferType": "Standard",
		},
	}
	if _, err := svc.ProvisionTemplateResource("sub-1", "rg-a", accountResource); err != nil {
		t.Fatalf("provision account returned error: %v", err)
	}

	databaseResource := map[string]any{
		"type": "Microsoft.DocumentDB/databaseAccounts/sqlDatabases",
		"name": "acct-a/db-a",
		"properties": map[string]any{
			"resource": map[string]any{"id": "db-a"},
		},
		"options": map[string]any{"throughput": 400},
	}
	if _, err := svc.ProvisionTemplateResource("sub-1", "rg-a", databaseResource); err != nil {
		t.Fatalf("provision database returned error: %v", err)
	}

	containerResource := map[string]any{
		"type": "Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers",
		"name": "acct-a/db-a/container-a",
		"properties": map[string]any{
			"resource": map[string]any{
				"id":           "container-a",
				"partitionKey": map[string]any{"paths": []any{"/pk"}, "kind": "Hash"},
			},
		},
	}
	containerResult, err := svc.ProvisionTemplateResource("sub-1", "rg-a", containerResource)
	if err != nil {
		t.Fatalf("provision container returned error: %v", err)
	}
	container := containerResult.(map[string]any)
	if container["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.DocumentDB/databaseAccounts/acct-a/sqlDatabases/db-a/containers/container-a" {
		t.Fatalf("unexpected provisioned container id: %v", container["id"])
	}
	if container["type"] != "Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers" {
		t.Fatalf("unexpected provisioned container type: %v", container["type"])
	}
}

func TestCosmosSQLDataPlaneDatabaseContainerAndDocumentLifecycle(t *testing.T) {
	svc := New()

	createDB, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	if err != nil {
		t.Fatalf("create database returned error: %v", err)
	}
	if createDB.StatusCode != http.StatusCreated {
		t.Fatalf("expected create database status 201, got %d; body=%s", createDB.StatusCode, string(createDB.RawBody))
	}
	db := decodeCosmosResponse(t, createDB)
	if db["id"] != "app" || db["_etag"] == "" || db["_colls"] != "colls/" {
		t.Fatalf("unexpected database body: %v", db)
	}

	duplicateDB, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	if err != nil {
		t.Fatalf("duplicate database returned error: %v", err)
	}
	if duplicateDB.StatusCode != http.StatusConflict {
		t.Fatalf("expected duplicate database status 409, got %d; body=%s", duplicateDB.StatusCode, string(duplicateDB.RawBody))
	}

	listDBs, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodGet, "https://acct.documents.azure.com/dbs", nil, nil))
	if err != nil {
		t.Fatalf("list databases returned error: %v", err)
	}
	listedDBs := decodeCosmosResponse(t, listDBs)
	if listedDBs["_count"] != float64(1) {
		t.Fatalf("expected one database, got %v", listedDBs)
	}

	createCollection, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	if err != nil {
		t.Fatalf("create collection returned error: %v", err)
	}
	if createCollection.StatusCode != http.StatusCreated {
		t.Fatalf("expected create collection status 201, got %d; body=%s", createCollection.StatusCode, string(createCollection.RawBody))
	}
	collection := decodeCosmosResponse(t, createCollection)
	if collection["id"] != "items" || collection["_docs"] != "docs/" || createCollection.Headers["x-ms-alt-content-path"] != "dbs/app" {
		t.Fatalf("unexpected collection response: body=%v headers=%v", collection, createCollection.Headers)
	}

	createDocument, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"id":"laptop-1","category":"electronics","name":"Laptop Pro","price":1299}`), nil))
	if err != nil {
		t.Fatalf("create document returned error: %v", err)
	}
	if createDocument.StatusCode != http.StatusCreated {
		t.Fatalf("expected create document status 201, got %d; body=%s", createDocument.StatusCode, string(createDocument.RawBody))
	}
	created := decodeCosmosResponse(t, createDocument)
	if created["id"] != "laptop-1" || created["_etag"] == "" || created["_ts"] == nil || createDocument.Headers["x-ms-alt-content-path"] != "dbs/app/colls/items" {
		t.Fatalf("unexpected created document: body=%v headers=%v", created, createDocument.Headers)
	}
	firstETag := createDocument.Headers["etag"]
	if firstETag == "" {
		t.Fatalf("expected document etag header")
	}

	readDocument, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodGet, "https://acct.documents.azure.com/dbs/app/colls/items/docs/laptop-1", nil, map[string]string{
		"x-ms-documentdb-partitionkey": `["electronics"]`,
	}))
	if err != nil {
		t.Fatalf("read document returned error: %v", err)
	}
	read := decodeCosmosResponse(t, readDocument)
	if read["name"] != "Laptop Pro" || read["price"] != float64(1299) {
		t.Fatalf("unexpected read document: %v", read)
	}

	replaceDocument, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPut, "https://acct.documents.azure.com/dbs/app/colls/items/docs/laptop-1", []byte(`{"id":"laptop-1","category":"electronics","name":"Laptop Pro","price":999}`), map[string]string{
		"x-ms-documentdb-partitionkey": `["electronics"]`,
		"If-Match":                     firstETag,
	}))
	if err != nil {
		t.Fatalf("replace document returned error: %v", err)
	}
	if replaceDocument.StatusCode != http.StatusOK {
		t.Fatalf("expected replace document status 200, got %d; body=%s", replaceDocument.StatusCode, string(replaceDocument.RawBody))
	}
	if replaceDocument.Headers["etag"] == "" || replaceDocument.Headers["etag"] == firstETag {
		t.Fatalf("expected changed etag, before=%q after=%q", firstETag, replaceDocument.Headers["etag"])
	}

	upsertDocument, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"id":"laptop-1","category":"electronics","name":"Laptop Pro","price":799}`), map[string]string{
		"x-ms-documentdb-is-upsert": "True",
	}))
	if err != nil {
		t.Fatalf("upsert document returned error: %v", err)
	}
	if upsertDocument.StatusCode != http.StatusOK {
		t.Fatalf("expected upsert replace status 200, got %d; body=%s", upsertDocument.StatusCode, string(upsertDocument.RawBody))
	}

	listDocuments, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodGet, "https://acct.documents.azure.com/dbs/app/colls/items/docs", nil, nil))
	if err != nil {
		t.Fatalf("list documents returned error: %v", err)
	}
	listedDocs := decodeCosmosResponse(t, listDocuments)
	if listedDocs["_count"] != float64(1) || len(listedDocs["Documents"].([]any)) != 1 {
		t.Fatalf("unexpected list documents response: %v", listedDocs)
	}

	deleteDocument, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodDelete, "https://acct.documents.azure.com/dbs/app/colls/items/docs/laptop-1", nil, map[string]string{
		"x-ms-documentdb-partitionkey": `["electronics"]`,
	}))
	if err != nil {
		t.Fatalf("delete document returned error: %v", err)
	}
	if deleteDocument.StatusCode != http.StatusNoContent {
		t.Fatalf("expected delete document status 204, got %d; body=%s", deleteDocument.StatusCode, string(deleteDocument.RawBody))
	}

	missingDocument, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodGet, "https://acct.documents.azure.com/dbs/app/colls/items/docs/laptop-1", nil, nil))
	if err != nil {
		t.Fatalf("read missing document returned error: %v", err)
	}
	if missingDocument.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing document status 404, got %d; body=%s", missingDocument.StatusCode, string(missingDocument.RawBody))
	}

	deleteDB, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodDelete, "https://acct.documents.azure.com/dbs/app", nil, nil))
	if err != nil {
		t.Fatalf("delete database returned error: %v", err)
	}
	if deleteDB.StatusCode != http.StatusNoContent {
		t.Fatalf("expected delete database status 204, got %d; body=%s", deleteDB.StatusCode, string(deleteDB.RawBody))
	}
}

func TestCosmosSQLDataPlaneQueryPaginationAndPatch(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	for _, payload := range []string{
		`{"id":"a","category":"sort","rank":3,"price":10,"removable":true}`,
		`{"id":"b","category":"sort","rank":1,"price":200}`,
		`{"id":"c","category":"sort","rank":2,"price":300}`,
	} {
		_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(payload), nil))
	}

	query, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT * FROM c WHERE c.price > @minPrice ORDER BY c.rank ASC","parameters":[{"name":"@minPrice","value":100}]}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
		"x-ms-max-item-count":     "1",
	}))
	if err != nil {
		t.Fatalf("query documents returned error: %v", err)
	}
	queried := decodeCosmosResponse(t, query)
	docs := queried["Documents"].([]any)
	if len(docs) != 1 || docs[0].(map[string]any)["id"] != "b" || query.Headers["x-ms-continuation"] == "" {
		t.Fatalf("unexpected first query page: body=%v headers=%v", queried, query.Headers)
	}

	nextQuery, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT * FROM c WHERE c.price > @minPrice ORDER BY c.rank ASC","parameters":[{"name":"@minPrice","value":100}]}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
		"x-ms-max-item-count":     "1",
		"x-ms-continuation":       query.Headers["x-ms-continuation"],
	}))
	if err != nil {
		t.Fatalf("next query page returned error: %v", err)
	}
	nextDocs := decodeCosmosResponse(t, nextQuery)["Documents"].([]any)
	if len(nextDocs) != 1 || nextDocs[0].(map[string]any)["id"] != "c" {
		t.Fatalf("unexpected second query page: %v", nextDocs)
	}

	countQuery, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT VALUE COUNT(1) FROM c"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("count query returned error: %v", err)
	}
	countDocs := decodeCosmosResponse(t, countQuery)["Documents"].([]any)
	if len(countDocs) != 1 || countDocs[0] != float64(3) {
		t.Fatalf("unexpected count query response: %v", countDocs)
	}

	patch, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPatch, "https://acct.documents.azure.com/dbs/app/colls/items/docs/a", []byte(`{"operations":[{"op":"set","path":"/name","value":"Patched"},{"op":"incr","path":"/price","value":5},{"op":"remove","path":"/removable"}]}`), map[string]string{
		"x-ms-documentdb-partitionkey": `["sort"]`,
	}))
	if err != nil {
		t.Fatalf("patch document returned error: %v", err)
	}
	patched := decodeCosmosResponse(t, patch)
	if patched["name"] != "Patched" || patched["price"] != float64(15) {
		t.Fatalf("unexpected patched document: %v", patched)
	}
	if _, exists := patched["removable"]; exists {
		t.Fatalf("expected removable to be deleted: %v", patched)
	}
}

func TestCosmosSQLDataPlaneQueryDistinctAndStringFunctions(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	for _, payload := range []string{
		`{"id":"item-1","category":"Electronics","name":"Laptop Pro","first":"John","last":"Doe"}`,
		`{"id":"item-2","category":"books","name":"Guide","first":"Jane","last":"Smith"}`,
		`{"id":"item-3","category":"books","name":"Reference","first":"Ada","last":"Lovelace"}`,
	} {
		_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(payload), nil))
	}

	distinctResp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT DISTINCT c.category FROM c"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("distinct query returned error: %v", err)
	}
	distinctDocs := decodeCosmosResponse(t, distinctResp)["Documents"].([]any)
	if len(distinctDocs) != 2 {
		t.Fatalf("expected two distinct categories, got %v", distinctDocs)
	}
	categories := []string{
		distinctDocs[0].(map[string]any)["category"].(string),
		distinctDocs[1].(map[string]any)["category"].(string),
	}
	sort.Strings(categories)
	if !reflect.DeepEqual(categories, []string{"Electronics", "books"}) {
		t.Fatalf("unexpected distinct categories: %v", distinctDocs)
	}

	lowerResp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT * FROM c WHERE LOWER(c.category) = 'electronics'"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("lower query returned error: %v", err)
	}
	lowerDocs := decodeCosmosResponse(t, lowerResp)["Documents"].([]any)
	if len(lowerDocs) != 1 || lowerDocs[0].(map[string]any)["id"] != "item-1" {
		t.Fatalf("unexpected LOWER query results: %v", lowerDocs)
	}

	upperResp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT * FROM c WHERE UPPER(c.category) = 'BOOKS' ORDER BY c.id ASC"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("upper query returned error: %v", err)
	}
	upperDocs := decodeCosmosResponse(t, upperResp)["Documents"].([]any)
	if len(upperDocs) != 2 || upperDocs[0].(map[string]any)["id"] != "item-2" || upperDocs[1].(map[string]any)["id"] != "item-3" {
		t.Fatalf("unexpected UPPER query results: %v", upperDocs)
	}

	lengthResp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT LOWER(c.category) AS cat, LENGTH(c.name) AS nlen, CONCAT(c.first, ' ', c.last) AS full_name FROM c WHERE c.id = 'item-1'"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("string function projection query returned error: %v", err)
	}
	projected := decodeCosmosResponse(t, lengthResp)["Documents"].([]any)
	if len(projected) != 1 {
		t.Fatalf("expected one projected document, got %v", projected)
	}
	row := projected[0].(map[string]any)
	if row["cat"] != "electronics" || row["nlen"] != float64(10) || row["full_name"] != "John Doe" {
		t.Fatalf("unexpected string projection row: %v", row)
	}
}

func TestCosmosSQLDataPlaneQueryLikeAndOffsetLimit(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	for _, payload := range []string{
		`{"id":"item-0","category":"query","name":"Laptop Pro","rank":0}`,
		`{"id":"item-1","category":"query","name":"Keyboard","rank":1}`,
		`{"id":"item-2","category":"query","name":"Monitor Pro","rank":2}`,
		`{"id":"item-3","category":"query","name":"Mouse","rank":3}`,
		`{"id":"item-4","category":"query","name":"Dock","rank":4}`,
	} {
		_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(payload), nil))
	}

	likeResp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT * FROM c WHERE c.name LIKE '%Pro%' ORDER BY c.rank ASC"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("LIKE query returned error: %v", err)
	}
	likeDocs := decodeCosmosResponse(t, likeResp)["Documents"].([]any)
	if len(likeDocs) != 2 || likeDocs[0].(map[string]any)["id"] != "item-0" || likeDocs[1].(map[string]any)["id"] != "item-2" {
		t.Fatalf("unexpected LIKE query results: %v", likeDocs)
	}

	pageResp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT * FROM c ORDER BY c.rank ASC OFFSET 1 LIMIT 2"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("OFFSET LIMIT query returned error: %v", err)
	}
	pageDocs := decodeCosmosResponse(t, pageResp)["Documents"].([]any)
	if len(pageDocs) != 2 || pageDocs[0].(map[string]any)["id"] != "item-1" || pageDocs[1].(map[string]any)["id"] != "item-2" {
		t.Fatalf("unexpected OFFSET LIMIT query results: %v", pageDocs)
	}
}

func TestCosmosSQLDataPlaneQueryGroupByCount(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	for _, payload := range []string{
		`{"id":"food-0","category":"food"}`,
		`{"id":"food-1","category":"food"}`,
		`{"id":"food-2","category":"food"}`,
		`{"id":"book-0","category":"books"}`,
		`{"id":"book-1","category":"books"}`,
	} {
		_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(payload), nil))
	}

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT c.category, COUNT(1) as count FROM c GROUP BY c.category"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("GROUP BY query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	counts := make(map[string]int)
	for _, doc := range docs {
		row := doc.(map[string]any)
		category, categoryOK := row["category"].(string)
		count, countOK := row["count"].(float64)
		if !categoryOK || !countOK {
			t.Fatalf("unexpected GROUP BY row shape: %v", row)
		}
		counts[category] = int(count)
	}
	if !reflect.DeepEqual(counts, map[string]int{"books": 2, "food": 3}) {
		t.Fatalf("unexpected GROUP BY counts: %v", docs)
	}
}

func TestCosmosSQLDataPlaneQueryCountScalarExpression(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"products","partitionKey":{"paths":["/detailCategory"],"kind":"Hash"}}`), nil))
	for _, payload := range []string{
		`{"id":"helmet-a","detailCategory":"gear-cycle-helmets","name":"Kameni Adult Bike Helmet"}`,
		`{"id":"helmet-b","detailCategory":"gear-cycle-helmets","name":"Rockmak Full Face Helmet"}`,
		`{"id":"helmet-c","detailCategory":"gear-cycle-helmets"}`,
		`{"id":"watch-a","detailCategory":"apparel-accessories-watches","name":"Diannis Watch"}`,
	} {
		_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/products/docs", []byte(payload), nil))
	}

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/products/docs", []byte(`{"query":"SELECT VALUE COUNT(p.name) FROM products p WHERE p.detailCategory = 'gear-cycle-helmets'"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("COUNT field query returned error: %v", err)
	}
	values := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if !reflect.DeepEqual(values, []any{float64(2)}) {
		t.Fatalf("unexpected COUNT field result: %v", values)
	}

	resp, err = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/products/docs", []byte(`{"query":"SELECT VALUE COUNT(2 + 3) FROM products p WHERE p.detailCategory = 'gear-cycle-helmets'"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("COUNT scalar expression query returned error: %v", err)
	}
	values = decodeCosmosResponse(t, resp)["Documents"].([]any)
	if !reflect.DeepEqual(values, []any{float64(3)}) {
		t.Fatalf("unexpected COUNT scalar expression result: %v", values)
	}
}

func TestCosmosSQLDataPlaneQueryGroupByAggregateFunctions(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	for _, payload := range []string{
		`{"id":"gear-a","category":"gear","quantity":4,"price":12}`,
		`{"id":"gear-b","category":"gear","quantity":6,"price":18}`,
		`{"id":"food-a","category":"food","quantity":2,"price":5}`,
		`{"id":"food-b","category":"food","quantity":8,"price":15}`,
	} {
		_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(payload), nil))
	}

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT c.category, SUM(c.quantity) AS totalQuantity, AVG(c.price) AS averagePrice, MIN(c.price) AS minPrice, MAX(c.price) AS maxPrice FROM c GROUP BY c.category"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("GROUP BY aggregate query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if len(docs) != 2 {
		t.Fatalf("expected two GROUP BY aggregate rows, got %v", docs)
	}
	rows := make(map[string]map[string]any)
	for _, doc := range docs {
		row := doc.(map[string]any)
		category, ok := row["category"].(string)
		if !ok {
			t.Fatalf("unexpected GROUP BY aggregate row: %v", row)
		}
		rows[category] = row
	}
	if !reflect.DeepEqual(rows["gear"], map[string]any{"category": "gear", "totalQuantity": float64(10), "averagePrice": float64(15), "minPrice": float64(12), "maxPrice": float64(18)}) {
		t.Fatalf("unexpected gear GROUP BY aggregate row: %v", rows["gear"])
	}
	if !reflect.DeepEqual(rows["food"], map[string]any{"category": "food", "totalQuantity": float64(10), "averagePrice": float64(10), "minPrice": float64(5), "maxPrice": float64(15)}) {
		t.Fatalf("unexpected food GROUP BY aggregate row: %v", rows["food"])
	}
}

func TestCosmosSQLDataPlaneQueryAggregateNumericExpressionsExcludeUndefined(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"products","partitionKey":{"paths":["/detailCategory"],"kind":"Hash"}}`), nil))
	for _, payload := range []string{
		`{"id":"pack-a","detailCategory":"gear-hike-backpacks","quantity":0,"price":98}`,
		`{"id":"pack-b","detailCategory":"gear-hike-backpacks","quantity":230,"price":105}`,
		`{"id":"pack-c","detailCategory":"gear-hike-backpacks","quantity":14}`,
		`{"id":"pack-d","detailCategory":"gear-hike-backpacks","quantity":232,"price":97}`,
		`{"id":"watch-a","detailCategory":"apparel-accessories-watches","quantity":5,"price":75}`,
	} {
		_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/products/docs", []byte(payload), nil))
	}

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/products/docs", []byte(`{"query":"SELECT VALUE SUM(p.quantity + 1) FROM products p WHERE p.detailCategory = 'gear-hike-backpacks'"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("SUM numeric expression query returned error: %v", err)
	}
	values := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if !reflect.DeepEqual(values, []any{float64(480)}) {
		t.Fatalf("unexpected SUM numeric expression result: %v", values)
	}

	resp, err = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/products/docs", []byte(`{"query":"SELECT VALUE AVG(p.price) FROM products p WHERE p.detailCategory = 'gear-hike-backpacks'"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("AVG undefined-exclusion query returned error: %v", err)
	}
	values = decodeCosmosResponse(t, resp)["Documents"].([]any)
	if !reflect.DeepEqual(values, []any{float64(100)}) {
		t.Fatalf("unexpected AVG undefined-exclusion result: %v", values)
	}
}

func TestCosmosSQLDataPlaneQueryTopLevelAggregateProjection(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"products","partitionKey":{"paths":["/detailCategory"],"kind":"Hash"}}`), nil))
	for _, payload := range []string{
		`{"id":"watch-a","detailCategory":"apparel-accessories-watches","price":98,"quantity":3}`,
		`{"id":"watch-b","detailCategory":"apparel-accessories-watches","price":105,"quantity":7}`,
		`{"id":"pack-a","detailCategory":"gear-hike-backpacks","price":75,"quantity":2}`,
	} {
		_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/products/docs", []byte(payload), nil))
	}

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/products/docs", []byte(`{"query":"SELECT AVG(p.price) AS averagePrice, SUM(p.quantity) AS totalQuantity, MIN(p.price) AS minPrice, MAX(p.price) AS maxPrice FROM products p WHERE p.detailCategory = 'apparel-accessories-watches'"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("top-level aggregate projection query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	expected := []any{map[string]any{
		"averagePrice":  float64(101.5),
		"totalQuantity": float64(10),
		"minPrice":      float64(98),
		"maxPrice":      float64(105),
	}}
	if !reflect.DeepEqual(docs, expected) {
		t.Fatalf("unexpected top-level aggregate projection result: %v", docs)
	}
}

func TestCosmosSQLDataPlaneQueryAggregateValueObjectProjection(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"products","partitionKey":{"paths":["/detailCategory"],"kind":"Hash"}}`), nil))
	for _, payload := range []string{
		`{"id":"watch-a","detailCategory":"apparel-accessories-watches","price":98,"quantity":3}`,
		`{"id":"watch-b","detailCategory":"apparel-accessories-watches","price":105,"quantity":7}`,
		`{"id":"pack-a","detailCategory":"gear-hike-backpacks","price":75,"quantity":2}`,
	} {
		_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/products/docs", []byte(payload), nil))
	}

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/products/docs", []byte(`{"query":"SELECT VALUE { averagePrice: AVG(p.price), totalQuantity: SUM(p.quantity), minPrice: MIN(p.price), maxPrice: MAX(p.price) } FROM products p WHERE p.detailCategory = 'apparel-accessories-watches'"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("aggregate value object projection query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	expected := []any{map[string]any{
		"averagePrice":  float64(101.5),
		"totalQuantity": float64(10),
		"minPrice":      float64(98),
		"maxPrice":      float64(105),
	}}
	if !reflect.DeepEqual(docs, expected) {
		t.Fatalf("unexpected aggregate value object projection result: %v", docs)
	}
}

func TestCosmosSQLDataPlaneQueryAggregatesReturnUndefinedForInvalidInputs(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"products","partitionKey":{"paths":["/detailCategory"],"kind":"Hash"}}`), nil))
	for _, payload := range []string{
		`{"id":"pack-a","detailCategory":"gear-hike-backpacks","price":98,"quantity":0}`,
		`{"id":"pack-b","detailCategory":"gear-hike-backpacks","price":"unknown","quantity":true}`,
		`{"id":"pack-c","detailCategory":"gear-hike-backpacks","price":105}`,
		`{"id":"watch-a","detailCategory":"apparel-accessories-watches","price":75,"quantity":3}`,
	} {
		_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/products/docs", []byte(payload), nil))
	}

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/products/docs", []byte(`{"query":"SELECT VALUE { averagePrice: AVG(p.price), totalQuantity: SUM(p.quantity), itemCount: COUNT(1) } FROM products p WHERE p.detailCategory = 'gear-hike-backpacks'"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("invalid aggregate input query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	expected := []any{map[string]any{"itemCount": float64(3)}}
	if !reflect.DeepEqual(docs, expected) {
		t.Fatalf("unexpected invalid aggregate input result: %v", docs)
	}
}

func TestCosmosSQLDataPlaneQueryGroupByMultipleExpressions(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	for _, payload := range []string{
		`{"id":"gear-west-a","category":"gear","region":"west","quantity":4}`,
		`{"id":"gear-west-b","category":"gear","region":"west","quantity":6}`,
		`{"id":"gear-east-a","category":"gear","region":"east","quantity":3}`,
		`{"id":"food-west-a","category":"food","region":"west","quantity":2}`,
		`{"id":"food-east-a","category":"food","region":"east","quantity":5}`,
		`{"id":"food-east-b","category":"food","region":"east","quantity":7}`,
	} {
		_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(payload), nil))
	}

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT c.category, c.region, COUNT(1) AS itemCount, SUM(c.quantity) AS totalQuantity FROM c GROUP BY c.category, c.region"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("multi-expression GROUP BY query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if len(docs) != 4 {
		t.Fatalf("expected four multi-expression GROUP BY rows, got %v", docs)
	}
	rows := make(map[string]map[string]any)
	for _, doc := range docs {
		row := doc.(map[string]any)
		category, categoryOK := row["category"].(string)
		region, regionOK := row["region"].(string)
		if !categoryOK || !regionOK {
			t.Fatalf("unexpected multi-expression GROUP BY row: %v", row)
		}
		rows[category+"|"+region] = row
	}
	expected := map[string]map[string]any{
		"gear|west": {"category": "gear", "region": "west", "itemCount": float64(2), "totalQuantity": float64(10)},
		"gear|east": {"category": "gear", "region": "east", "itemCount": float64(1), "totalQuantity": float64(3)},
		"food|west": {"category": "food", "region": "west", "itemCount": float64(1), "totalQuantity": float64(2)},
		"food|east": {"category": "food", "region": "east", "itemCount": float64(2), "totalQuantity": float64(12)},
	}
	if !reflect.DeepEqual(rows, expected) {
		t.Fatalf("unexpected multi-expression GROUP BY rows: %v", rows)
	}
}

func TestCosmosSQLDataPlaneQueryArrayFunctions(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"id":"array-query","category":"arr","tags":["a","b","c","d"]}`), nil))

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT ARRAY_LENGTH(c.tags) AS len, ARRAY_SLICE(c.tags, 1, 2) AS slice FROM c WHERE c.id = 'array-query'"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("array function query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if len(docs) != 1 {
		t.Fatalf("expected one array function result, got %v", docs)
	}
	row := docs[0].(map[string]any)
	if row["len"] != float64(4) {
		t.Fatalf("unexpected ARRAY_LENGTH result: %v", row)
	}
	if !reflect.DeepEqual(row["slice"], []any{"b", "c"}) {
		t.Fatalf("unexpected ARRAY_SLICE result: %v", row)
	}
}

func TestCosmosSQLDataPlaneQueryArrayLengthReturnsUndefinedForInvalidInputs(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT VALUE { length: ARRAY_LENGTH([70, 86, 92, 99, 85, 90, 82]), emptyLength: ARRAY_LENGTH([]), nullLength: ARRAY_LENGTH(null), stringLength: ARRAY_LENGTH(\"not-array\"), objectLength: ARRAY_LENGTH({ name: \"AdventureWorks\" }), undefinedLength: ARRAY_LENGTH(undefined), missingArgument: ARRAY_LENGTH() }"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("ARRAY_LENGTH invalid-input query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	expected := []any{map[string]any{
		"length":      float64(7),
		"emptyLength": float64(0),
	}}
	if !reflect.DeepEqual(docs, expected) {
		t.Fatalf("unexpected ARRAY_LENGTH invalid-input result: %v", docs)
	}
}

func TestCosmosSQLDataPlaneQueryArraySliceReturnsUndefinedForInvalidInputs(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT VALUE { sliceFromSecond: ARRAY_SLICE([\"Alpha\", \"Bravo\", \"Charlie\", \"Delta\"], 1), sliceFromLast: ARRAY_SLICE([\"Alpha\", \"Bravo\", \"Charlie\", \"Delta\"], -1), sliceLimited: ARRAY_SLICE([\"Alpha\", \"Bravo\", \"Charlie\", \"Delta\"], 1, 2), invalidArray: ARRAY_SLICE(\"Alpha\", 0), invalidStartString: ARRAY_SLICE([\"Alpha\", \"Bravo\"], \"1\"), invalidStartBool: ARRAY_SLICE([\"Alpha\", \"Bravo\"], true), invalidLengthString: ARRAY_SLICE([\"Alpha\", \"Bravo\"], 0, \"1\"), invalidLengthArray: ARRAY_SLICE([\"Alpha\", \"Bravo\"], 0, [1]), invalidMissingStart: ARRAY_SLICE([\"Alpha\", \"Bravo\"]) }"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("ARRAY_SLICE invalid-input query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	expected := []any{map[string]any{
		"sliceFromSecond": []any{"Bravo", "Charlie", "Delta"},
		"sliceFromLast":   []any{"Delta"},
		"sliceLimited":    []any{"Bravo", "Charlie"},
	}}
	if !reflect.DeepEqual(docs, expected) {
		t.Fatalf("unexpected ARRAY_SLICE invalid-input result: %v", docs)
	}
}

func TestCosmosSQLDataPlaneQueryNestedPathsTopAndCompositeWhere(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	for _, payload := range []string{
		`{"id":"jacket","category":"apparel","metadata":{"rating":{"score":19}},"sizes":[{"name":"small"},{"name":"medium"}]}`,
		`{"id":"boots","category":"gear","metadata":{"rating":{"score":15}},"sizes":[{"name":"nine"}]}`,
		`{"id":"rope","category":"gear","metadata":{"rating":{"score":9}},"sizes":[{"name":"short"}]}`,
		`{"id":"snack","category":"food","metadata":{"rating":{"score":20}},"sizes":[{"name":"single"}]}`,
	} {
		_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(payload), nil))
	}

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT TOP 2 c.id, c.metadata.rating.score AS score, c.sizes[0].name AS defaultSize FROM c WHERE c.metadata.rating.score >= 10 AND (c.category = 'gear' OR c.category = 'apparel') ORDER BY c.metadata.rating.score DESC"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("nested path query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if len(docs) != 2 {
		t.Fatalf("expected TOP 2 nested query results, got %v", docs)
	}
	first := docs[0].(map[string]any)
	second := docs[1].(map[string]any)
	if first["id"] != "jacket" || first["score"] != float64(19) || first["defaultSize"] != "small" {
		t.Fatalf("unexpected first nested query row: %v", first)
	}
	if second["id"] != "boots" || second["score"] != float64(15) || second["defaultSize"] != "nine" {
		t.Fatalf("unexpected second nested query row: %v", second)
	}
}

func TestCosmosSQLDataPlaneQueryTypeConditionalAndMathFunctions(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	for _, payload := range []string{
		`{"id":"stocked","category":"math","price":-9,"discount":2.6,"optional":null,"inventory":{"quantity":16}}`,
		`{"id":"text-quantity","category":"math","price":5,"discount":1.2,"inventory":{"quantity":"many"}}`,
		`{"id":"missing-optional","category":"math","price":3,"discount":4.8,"inventory":{"quantity":25}}`,
	} {
		_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(payload), nil))
	}

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT IIF(IS_NULL(c.optional), 'null-value', 'present') AS optionalState, IS_NULL(c.optional) AS optionalNull, ABS(c.price) AS absPrice, CEILING(c.discount) AS discountCeiling, FLOOR(c.discount) AS discountFloor, ROUND(c.discount) AS discountRound, SQRT(c.inventory.quantity) AS quantityRoot FROM c WHERE IS_DEFINED(c.optional) AND IS_NUMBER(c.inventory.quantity)"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("type and math function query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if len(docs) != 1 {
		t.Fatalf("expected one type and math function result, got %v", docs)
	}
	row := docs[0].(map[string]any)
	if row["optionalState"] != "null-value" || row["optionalNull"] != true {
		t.Fatalf("unexpected conditional/type function fields: %v", row)
	}
	if row["absPrice"] != float64(9) || row["discountCeiling"] != float64(3) || row["discountFloor"] != float64(2) || row["discountRound"] != float64(3) || row["quantityRoot"] != float64(4) {
		t.Fatalf("unexpected math function fields: %v", row)
	}
}

func TestCosmosSQLDataPlaneQueryKeywordAndStringPredicates(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	for _, payload := range []string{
		`{"id":"a","category":"gear","price":12,"name":"Pro Jacket","sku":"JK-A"}`,
		`{"id":"b","category":"apparel","price":25,"name":"Trail Boots","sku":"BT-A"}`,
		`{"id":"c","category":"food","price":15,"name":"Pro Snacks","sku":"SN-A"}`,
		`{"id":"d","category":"gear","price":45,"name":"Trail Rope","sku":"RP-A"}`,
		`{"id":"e","category":"gear","price":20,"name":"Basic Hat","sku":"HT-B"}`,
	} {
		_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(payload), nil))
	}

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT c.id FROM c WHERE NOT (c.category IN ('food', 'archived')) AND c.price BETWEEN 10 AND 30 AND (CONTAINS(c.name, 'pro', true) OR STARTSWITH(c.name, 'Trail')) AND ENDSWITH(c.sku, '-A') ORDER BY c.id ASC"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("keyword predicate query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if len(docs) != 2 {
		t.Fatalf("expected two keyword predicate results, got %v", docs)
	}
	ids := []string{docs[0].(map[string]any)["id"].(string), docs[1].(map[string]any)["id"].(string)}
	if !reflect.DeepEqual(ids, []string{"a", "b"}) {
		t.Fatalf("unexpected keyword predicate results: %v", docs)
	}
}

func TestCosmosSQLDataPlaneQueryStringPredicatesReturnUndefinedForInvalidInputs(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT VALUE { containsSuffix: CONTAINS(\"AdventureWorks\", \"Works\"), startsWithIgnoreCase: STARTSWITH(\"AdventureWorks\", \"adventure\", true), endsWithIgnoreCase: ENDSWITH(\"AdventureWorks\", \"works\", true), stringEqualsIgnoreCase: STRINGEQUALS(\"AdventureWorks\", \"adventureworks\", true), regexMatch: REGEXMATCH(\"abcd\", \"ABC\", \"i\"), containsInvalidLeft: CONTAINS(123, \"Adventure\"), containsInvalidRight: CONTAINS(\"AdventureWorks\", 123), startsWithInvalidLeft: STARTSWITH([\"Adventure\"], \"Adventure\"), endsWithInvalidRight: ENDSWITH(\"AdventureWorks\", {suffix: \"Works\"}), stringEqualsInvalidLeft: STRINGEQUALS(123, \"123\"), stringEqualsInvalidRight: STRINGEQUALS(\"AdventureWorks\", [\"AdventureWorks\"]), regexInvalidValue: REGEXMATCH(true, \"true\"), regexInvalidPattern: REGEXMATCH(\"AdventureWorks\", 123), regexInvalidFlags: REGEXMATCH(\"AdventureWorks\", \"Adventure\", true) }"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("invalid string predicate query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	expected := []any{map[string]any{
		"containsSuffix":         true,
		"startsWithIgnoreCase":   true,
		"endsWithIgnoreCase":     true,
		"stringEqualsIgnoreCase": true,
		"regexMatch":             true,
	}}
	if !reflect.DeepEqual(docs, expected) {
		t.Fatalf("unexpected invalid string predicate result: %v", docs)
	}
}

func TestCosmosSQLDataPlaneQueryFullTextPredicates(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	for _, payload := range []string{
		`{"id":"guide","category":"docs","text":"Azure Cosmos DB provides globally distributed NoSQL search with vector and full text capabilities."}`,
		`{"id":"notes","category":"docs","text":"Azure Table Storage offers simple key value access."}`,
	} {
		_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(payload), nil))
	}

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT c.id, FULLTEXTCONTAINS(c.text, 'cosmos db') AS containsPhrase, FULLTEXTCONTAINSALL(c.text, 'azure', 'search') AS containsAll, FULLTEXTCONTAINSALL(c.text, 'azure', 'missing') AS containsAllMissing, FULLTEXTCONTAINSANY(c.text, 'missing', 'vector') AS containsAny, FULLTEXTCONTAINSANY(c.text, 'missing', 'absent') AS containsAnyMissing FROM c WHERE FULLTEXTCONTAINS(c.text, 'cosmos db') AND FULLTEXTCONTAINSALL(c.text, 'azure', 'search') AND FULLTEXTCONTAINSANY(c.text, 'missing', 'vector')"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("full-text predicate query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if len(docs) != 1 {
		t.Fatalf("expected one full-text predicate result, got %v", docs)
	}
	row := docs[0].(map[string]any)
	if row["id"] != "guide" || row["containsPhrase"] != true || row["containsAll"] != true || row["containsAllMissing"] != false || row["containsAny"] != true || row["containsAnyMissing"] != false {
		t.Fatalf("unexpected full-text predicate fields: %v", row)
	}
}

func TestCosmosSQLDataPlaneQueryFullTextScoreRank(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	for _, payload := range []string{
		`{"id":"a-weak","category":"docs","text":"keyword appears once"}`,
		`{"id":"z-strong","category":"docs","text":"keyword keyword keyword with more matching context"}`,
		`{"id":"other","category":"docs","text":"unrelated content"}`,
	} {
		_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(payload), nil))
	}

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT TOP 2 c.id FROM c WHERE FULLTEXTCONTAINS(c.text, 'keyword') ORDER BY RANK FULLTEXTSCORE(c.text, 'keyword')"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("full-text score rank query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if len(docs) != 2 {
		t.Fatalf("expected two full-text score results, got %v", docs)
	}
	ids := []string{docs[0].(map[string]any)["id"].(string), docs[1].(map[string]any)["id"].(string)}
	if !reflect.DeepEqual(ids, []string{"z-strong", "a-weak"}) {
		t.Fatalf("unexpected full-text score ranking: %v", docs)
	}
}

func TestCosmosSQLDataPlaneQueryReciprocalRankFusion(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	for _, payload := range []string{
		`{"id":"text-heavy","category":"hybrid","text":"keyword keyword keyword","vector":[9,9,9]}`,
		`{"id":"vector-match","category":"hybrid","text":"keyword","vector":[1,2,3]}`,
		`{"id":"weak","category":"hybrid","text":"keyword","vector":[5,5,5]}`,
	} {
		_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(payload), nil))
	}

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT TOP 2 c.id FROM c WHERE FULLTEXTCONTAINS(c.text, 'keyword') ORDER BY RANK RRF(FULLTEXTSCORE(c.text, 'keyword'), VECTORDISTANCE(c.vector, [1,2,3], true, {distanceFunction: 'Euclidean'}), [1,4])"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("RRF rank query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if len(docs) != 2 {
		t.Fatalf("expected two RRF results, got %v", docs)
	}
	ids := []string{docs[0].(map[string]any)["id"].(string), docs[1].(map[string]any)["id"].(string)}
	if !reflect.DeepEqual(ids, []string{"vector-match", "text-heavy"}) {
		t.Fatalf("unexpected RRF ranking: %v", docs)
	}
}

func TestCosmosSQLDataPlaneQuerySpatialPointDistance(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"places","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	for _, payload := range []string{
		`{"id":"near","category":"geo","location":{"type":"Point","coordinates":[0.001,0]}}`,
		`{"id":"exact","category":"geo","location":{"type":"Point","coordinates":[0,0]}}`,
		`{"id":"far","category":"geo","location":{"type":"Point","coordinates":[1,0]}}`,
	} {
		_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/places/docs", []byte(payload), nil))
	}

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/places/docs", []byte(`{"query":"SELECT TOP 2 c.id, ST_DISTANCE(c.location, {type:'Point', coordinates:[0,0]}) AS meters FROM c ORDER BY ST_DISTANCE(c.location, {type:'Point', coordinates:[0,0]}) ASC"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("spatial distance query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if len(docs) != 2 {
		t.Fatalf("expected two spatial distance results, got %v", docs)
	}
	first := docs[0].(map[string]any)
	second := docs[1].(map[string]any)
	if first["id"] != "exact" || first["meters"] != float64(0) {
		t.Fatalf("unexpected closest spatial result: %v", first)
	}
	nearMeters, ok := second["meters"].(float64)
	if second["id"] != "near" || !ok || nearMeters < 111 || nearMeters > 112 {
		t.Fatalf("unexpected near spatial result: %v", second)
	}
}

func TestCosmosSQLDataPlaneQuerySpatialDistanceToLineAndPolygons(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"places","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	for _, payload := range []string{
		`{"id":"on-line","category":"geo","location":{"type":"Point","coordinates":[0.5,0]}}`,
		`{"id":"near-line","category":"geo","location":{"type":"Point","coordinates":[0.5,0.001]}}`,
		`{"id":"far-line","category":"geo","location":{"type":"Point","coordinates":[2,0]}}`,
		`{"id":"inside-polygon","category":"geo","location":{"type":"Point","coordinates":[10.5,10.5]}}`,
		`{"id":"inside-multipolygon","category":"geo","location":{"type":"Point","coordinates":[20.5,20.5]}}`,
	} {
		_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/places/docs", []byte(payload), nil))
	}

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/places/docs", []byte(`{"query":"SELECT TOP 3 c.id, ST_DISTANCE(c.location, {type:'LineString', coordinates:[[0,0],[1,0]]}) AS meters FROM c WHERE STARTSWITH(c.id, 'on-line') OR STARTSWITH(c.id, 'near-line') OR STARTSWITH(c.id, 'far-line') ORDER BY ST_DISTANCE(c.location, {type:'LineString', coordinates:[[0,0],[1,0]]}) ASC"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("spatial line distance query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if len(docs) != 3 {
		t.Fatalf("expected three spatial line distance results, got %v", docs)
	}
	onLine := docs[0].(map[string]any)
	nearLine := docs[1].(map[string]any)
	farLine := docs[2].(map[string]any)
	if onLine["id"] != "on-line" || onLine["meters"] != float64(0) {
		t.Fatalf("unexpected on-line spatial distance: %v", onLine)
	}
	nearMeters, nearOK := nearLine["meters"].(float64)
	if nearLine["id"] != "near-line" || !nearOK || nearMeters < 111 || nearMeters > 112 {
		t.Fatalf("unexpected near-line spatial distance: %v", nearLine)
	}
	farMeters, farOK := farLine["meters"].(float64)
	if farLine["id"] != "far-line" || !farOK || farMeters < 111000 || farMeters > 112000 {
		t.Fatalf("unexpected far-line spatial distance: %v", farLine)
	}

	polygon := `{type:'Polygon', coordinates:[[[10,10],[11,10],[11,11],[10,11],[10,10]]]}`
	resp, err = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/places/docs", []byte(`{"query":"SELECT VALUE ST_DISTANCE(c.location, `+polygon+`) FROM c WHERE c.id = 'inside-polygon'"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("spatial polygon distance query returned error: %v", err)
	}
	values := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if !reflect.DeepEqual(values, []any{float64(0)}) {
		t.Fatalf("unexpected polygon distance result: %v", values)
	}

	multiPolygon := `{type:'MultiPolygon', coordinates:[[[[20,20],[21,20],[21,21],[20,21],[20,20]]],[[[30,30],[31,30],[31,31],[30,31],[30,30]]]]}`
	resp, err = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/places/docs", []byte(`{"query":"SELECT VALUE ST_DISTANCE(c.location, `+multiPolygon+`) FROM c WHERE c.id = 'inside-multipolygon'"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("spatial multipolygon distance query returned error: %v", err)
	}
	values = decodeCosmosResponse(t, resp)["Documents"].([]any)
	if !reflect.DeepEqual(values, []any{float64(0)}) {
		t.Fatalf("unexpected multipolygon distance result: %v", values)
	}
}

func TestCosmosSQLDataPlaneQuerySpatialPointWithinPolygon(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"places","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	for _, payload := range []string{
		`{"id":"inside","category":"geo","location":{"type":"Point","coordinates":[0.5,0.5]}}`,
		`{"id":"outside","category":"geo","location":{"type":"Point","coordinates":[2,2]}}`,
	} {
		_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/places/docs", []byte(payload), nil))
	}

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/places/docs", []byte(`{"query":"SELECT c.id, ST_WITHIN(c.location, {type:'Polygon', coordinates:[[[0,0],[1,0],[1,1],[0,1],[0,0]]]}) AS inBox FROM c WHERE ST_WITHIN(c.location, {type:'Polygon', coordinates:[[[0,0],[1,0],[1,1],[0,1],[0,0]]]}) ORDER BY c.id ASC"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("spatial within query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if len(docs) != 1 {
		t.Fatalf("expected one spatial within result, got %v", docs)
	}
	inside := docs[0].(map[string]any)
	if inside["id"] != "inside" || inside["inBox"] != true {
		t.Fatalf("unexpected spatial within result: %v", inside)
	}
}

func TestCosmosSQLDataPlaneQuerySpatialPointWithinMultiPolygon(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"places","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	for _, payload := range []string{
		`{"id":"first","category":"geo","location":{"type":"Point","coordinates":[0.25,0.25]}}`,
		`{"id":"second","category":"geo","location":{"type":"Point","coordinates":[10.25,10.25]}}`,
		`{"id":"outside","category":"geo","location":{"type":"Point","coordinates":[5,5]}}`,
	} {
		_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/places/docs", []byte(payload), nil))
	}

	multiPolygon := `{type:'MultiPolygon', coordinates:[[[[0,0],[1,0],[1,1],[0,1],[0,0]]],[[[10,10],[11,10],[11,11],[10,11],[10,10]]]]}`
	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/places/docs", []byte(`{"query":"SELECT c.id, ST_WITHIN(c.location, `+multiPolygon+`) AS inCampus FROM c WHERE ST_WITHIN(c.location, `+multiPolygon+`) ORDER BY c.id ASC"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("spatial multipolygon within query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	ids := make([]string, 0, len(docs))
	for _, doc := range docs {
		row := doc.(map[string]any)
		if row["inCampus"] != true {
			t.Fatalf("unexpected spatial multipolygon within projection: %v", row)
		}
		ids = append(ids, row["id"].(string))
	}
	if !reflect.DeepEqual(ids, []string{"first", "second"}) {
		t.Fatalf("unexpected spatial multipolygon within ids: %v", ids)
	}
}

func TestCosmosSQLDataPlaneQuerySpatialPolygonsIntersect(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"places","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	for _, payload := range []string{
		`{"id":"overlap","category":"geo","area":{"type":"Polygon","coordinates":[[[0,0],[1,0],[1,1],[0,1],[0,0]]]}}`,
		`{"id":"separate","category":"geo","area":{"type":"Polygon","coordinates":[[[2,2],[3,2],[3,3],[2,3],[2,2]]]}}`,
	} {
		_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/places/docs", []byte(payload), nil))
	}

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/places/docs", []byte(`{"query":"SELECT c.id, ST_INTERSECTS(c.area, {type:'Polygon', coordinates:[[[0.5,0.5],[1.5,0.5],[1.5,1.5],[0.5,1.5],[0.5,0.5]]]}) AS overlaps FROM c WHERE ST_INTERSECTS(c.area, {type:'Polygon', coordinates:[[[0.5,0.5],[1.5,0.5],[1.5,1.5],[0.5,1.5],[0.5,0.5]]]}) ORDER BY c.id ASC"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("spatial intersects query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if len(docs) != 1 {
		t.Fatalf("expected one spatial intersects result, got %v", docs)
	}
	overlap := docs[0].(map[string]any)
	if overlap["id"] != "overlap" || overlap["overlaps"] != true {
		t.Fatalf("unexpected spatial intersects result: %v", overlap)
	}
}

func TestCosmosSQLDataPlaneQuerySpatialMultiPolygonsIntersect(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"places","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	for _, payload := range []string{
		`{"id":"multi-overlap","category":"geo","area":{"type":"MultiPolygon","coordinates":[[[[0,0],[1,0],[1,1],[0,1],[0,0]]],[[[10,10],[11,10],[11,11],[10,11],[10,10]]]]}}`,
		`{"id":"multi-separate","category":"geo","area":{"type":"MultiPolygon","coordinates":[[[[2,2],[3,2],[3,3],[2,3],[2,2]]]]}}`,
	} {
		_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/places/docs", []byte(payload), nil))
	}

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/places/docs", []byte(`{"query":"SELECT c.id, ST_INTERSECTS(c.area, {type:'Polygon', coordinates:[[[0.5,0.5],[1.5,0.5],[1.5,1.5],[0.5,1.5],[0.5,0.5]]]}) AS overlaps FROM c WHERE ST_INTERSECTS(c.area, {type:'Polygon', coordinates:[[[0.5,0.5],[1.5,0.5],[1.5,1.5],[0.5,1.5],[0.5,0.5]]]}) ORDER BY c.id ASC"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("spatial multipolygon intersects query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	ids := make([]string, 0, len(docs))
	for _, doc := range docs {
		row := doc.(map[string]any)
		if row["overlaps"] != true {
			t.Fatalf("unexpected spatial multipolygon intersects projection: %v", row)
		}
		ids = append(ids, row["id"].(string))
	}
	if !reflect.DeepEqual(ids, []string{"multi-overlap"}) {
		t.Fatalf("unexpected spatial multipolygon intersects ids: %v", ids)
	}
}

func TestCosmosSQLDataPlaneQuerySpatialValidityFunctions(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"places","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	for _, payload := range []string{
		`{"id":"valid","category":"geo","location":{"type":"Point","coordinates":[-84.38876194345323,33.75682784306348]}}`,
		`{"id":"invalid","category":"geo","location":{"type":"Point","coordinates":[133.75682784306348,-184.38876194345323]}}`,
	} {
		_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/places/docs", []byte(payload), nil))
	}

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/places/docs", []byte(`{"query":"SELECT c.id, ST_ISVALID(c.location) AS isValid, ST_ISVALIDDETAILED(c.location) AS detail FROM c ORDER BY c.id ASC"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("spatial validity query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if len(docs) != 2 {
		t.Fatalf("expected two spatial validity results, got %v", docs)
	}
	invalid := docs[0].(map[string]any)
	if invalid["id"] != "invalid" || invalid["isValid"] != false {
		t.Fatalf("unexpected invalid spatial validity result: %v", invalid)
	}
	invalidDetail := invalid["detail"].(map[string]any)
	if invalidDetail["valid"] != false || invalidDetail["reason"] != "Latitude values must be between -90 and 90 degrees." {
		t.Fatalf("unexpected invalid spatial validity detail: %v", invalidDetail)
	}
	valid := docs[1].(map[string]any)
	if valid["id"] != "valid" || valid["isValid"] != true {
		t.Fatalf("unexpected valid spatial validity result: %v", valid)
	}
	validDetail := valid["detail"].(map[string]any)
	if validDetail["valid"] != true {
		t.Fatalf("unexpected valid spatial validity detail: %v", validDetail)
	}
	if _, exists := validDetail["reason"]; exists {
		t.Fatalf("valid spatial validity detail should not include reason: %v", validDetail)
	}
}

func TestCosmosSQLDataPlaneQuerySpatialValidityPredicate(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"places","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	for _, payload := range []string{
		`{"id":"valid","category":"geo","location":{"type":"Point","coordinates":[-84.38876194345323,33.75682784306348]}}`,
		`{"id":"invalid","category":"geo","location":{"type":"Point","coordinates":[133.75682784306348,-184.38876194345323]}}`,
	} {
		_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/places/docs", []byte(payload), nil))
	}

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/places/docs", []byte(`{"query":"SELECT c.id FROM c WHERE ST_ISVALID(c.location) ORDER BY c.id ASC"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("spatial validity predicate query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if len(docs) != 1 || docs[0].(map[string]any)["id"] != "valid" {
		t.Fatalf("unexpected spatial validity predicate results: %v", docs)
	}
}

func TestCosmosSQLDataPlaneQuerySpatialPolygonArea(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"places","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	for _, payload := range []string{
		`{"id":"learn-example","category":"geo","area":{"type":"Polygon","coordinates":[[[31.8,-5],[32,-5],[32,-4.7],[31.8,-4.7],[31.8,-5]]]}}`,
		`{"id":"tiny","category":"geo","area":{"type":"Polygon","coordinates":[[[0,0],[0.001,0],[0.001,0.001],[0,0.001],[0,0]]]}}`,
	} {
		_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/places/docs", []byte(payload), nil))
	}

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/places/docs", []byte(`{"query":"SELECT c.id, ST_AREA(c.area) AS squareMeters FROM c WHERE ST_AREA(c.area) > 700000000 ORDER BY c.id ASC"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("spatial area query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if len(docs) != 1 {
		t.Fatalf("expected one spatial area result, got %v", docs)
	}
	row := docs[0].(map[string]any)
	area, ok := row["squareMeters"].(float64)
	if row["id"] != "learn-example" || !ok || area < 735000000 || area > 737000000 {
		t.Fatalf("unexpected spatial area result: %v", row)
	}
}

func TestCosmosSQLDataPlaneQueryVectorDistance(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	for _, payload := range []string{
		`{"id":"exact","category":"vectors","vector":[1,2,3]}`,
		`{"id":"near","category":"vectors","vector":[2,2,3]}`,
		`{"id":"far","category":"vectors","vector":[5,5,5]}`,
	} {
		_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(payload), nil))
	}

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT TOP 2 c.id, VECTORDISTANCE(c.vector, [1, 2, 3], true, {distanceFunction: 'Euclidean'}) AS distance FROM c ORDER BY VECTORDISTANCE(c.vector, [1, 2, 3], true, {distanceFunction: 'Euclidean'}) ASC"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("vector distance query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if len(docs) != 2 {
		t.Fatalf("expected two vector distance results, got %v", docs)
	}
	first := docs[0].(map[string]any)
	second := docs[1].(map[string]any)
	if first["id"] != "exact" || first["distance"] != float64(0) {
		t.Fatalf("unexpected closest vector result: %v", first)
	}
	if second["id"] != "near" || second["distance"] != float64(1) {
		t.Fatalf("unexpected second vector result: %v", second)
	}
}

func TestCosmosSQLDataPlaneQueryVectorDistanceOptions(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT VALUE { cosineSame: VECTORDISTANCE([1, 0], [1, 0], true, {distanceFunction: 'Cosine'}), cosineOrthogonal: VECTORDISTANCE([1, 0], [0, 1], true, {distanceFunction: 'Cosine'}), dotProduct: VECTORDISTANCE([1, 2], [3, 4], true, {distanceFunction: 'DotProduct'}) }"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("vector distance options query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if len(docs) != 1 {
		t.Fatalf("expected one vector options result, got %v", docs)
	}
	row := docs[0].(map[string]any)
	if row["cosineSame"] != float64(0) || row["cosineOrthogonal"] != float64(1) || row["dotProduct"] != float64(11) {
		t.Fatalf("unexpected vector distance option fields: %v", row)
	}
}

func TestCosmosSQLDataPlaneQueryAdditionalStringAndArrayFunctions(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"id":"functions","category":"fn","name":"  AdventureWorksLT  ","description":"AdventureWorksLT","tags":["coats","jackets"],"moreTags":["sweatshirts"],"sku":"abC-123"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"id":"other","category":"fn","name":"Other","description":"Other","tags":["boots"],"moreTags":[],"sku":"nope"}`), nil))

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT SUBSTRING(c.description, 9, 5) AS suffix, TRIM(c.name) AS trimmed, REPLACE(c.description, 'LT', 'LT2') AS replaced, INDEX_OF(c.description, 'Works') AS worksIndex, ARRAY_CONCAT(c.tags, c.moreTags) AS allTags, ARRAY_CONTAINS(c.tags, 'coats') AS containsCoats FROM c WHERE ARRAY_CONTAINS(c.tags, 'coats') AND STRINGEQUALS(TRIM(c.name), 'adventureworkslt', true) AND REGEXMATCH(c.sku, '^ABC-[0-9]+$', 'i')"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("additional function query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if len(docs) != 1 {
		t.Fatalf("expected one additional function result, got %v", docs)
	}
	row := docs[0].(map[string]any)
	if row["suffix"] != "Works" || row["trimmed"] != "AdventureWorksLT" || row["replaced"] != "AdventureWorksLT2" || row["worksIndex"] != float64(9) || row["containsCoats"] != true {
		t.Fatalf("unexpected additional string/array function fields: %v", row)
	}
	if !reflect.DeepEqual(row["allTags"], []any{"coats", "jackets", "sweatshirts"}) {
		t.Fatalf("unexpected ARRAY_CONCAT result: %v", row)
	}
}

func TestCosmosSQLDataPlaneQueryArrayConcatReturnsUndefinedForInvalidInputs(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT VALUE { twoArrays: ARRAY_CONCAT([\"backpacks\", \"daypacks\"], [\"hippacks\"]), threeArrays: ARRAY_CONCAT([1], [2], [3]), invalidSingle: ARRAY_CONCAT([\"backpacks\"]), invalidString: ARRAY_CONCAT([\"backpacks\"], \"daypacks\"), invalidNull: ARRAY_CONCAT([\"backpacks\"], null), invalidUndefined: ARRAY_CONCAT([\"backpacks\"], undefined), invalidMissing: ARRAY_CONCAT() }"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("ARRAY_CONCAT invalid-input query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	expected := []any{map[string]any{
		"twoArrays":   []any{"backpacks", "daypacks", "hippacks"},
		"threeArrays": []any{float64(1), float64(2), float64(3)},
	}}
	if !reflect.DeepEqual(docs, expected) {
		t.Fatalf("unexpected ARRAY_CONCAT invalid-input result: %v", docs)
	}
}

func TestCosmosSQLDataPlaneQueryAdditionalStringFunctions(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT VALUE { firstZero: LEFT(\"AdventureWorks\", 0), firstFive: LEFT(\"AdventureWorks\", 5), leftBeyond: LEFT(\"AdventureWorks\", 100), lastZero: RIGHT(\"AdventureWorks\", 0), lastFive: RIGHT(\"AdventureWorks\", 5), rightBeyond: RIGHT(\"AdventureWorks\", 100), ltrimWhitespace: LTRIM(\"  AdventureWorks  \"), ltrimPrefix: LTRIM(\"AdventureWorks\", \"Adventure\"), ltrimSuffixNoop: LTRIM(\"AdventureWorks\", \"Works\"), rtrimWhitespace: RTRIM(\"  AdventureWorks  \"), rtrimSuffix: RTRIM(\"AdventureWorks\", \"Works\"), rtrimPrefixNoop: RTRIM(\"AdventureWorks\", \"Adventure\"), replicateThree: REPLICATE(\"Cosmic\", 3), replicateZero: REPLICATE(\"Cosmic\", 0), replicateNegative: REPLICATE(\"Cosmic\", -1), reverseAdventureWorks: REVERSE(\"AdventureWorks\"), doubleReverseAdventureWorks: REVERSE(REVERSE(\"AdventureWorks\")) }"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("additional string function query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if len(docs) != 1 {
		t.Fatalf("expected one additional string function result, got %v", docs)
	}
	row := docs[0].(map[string]any)
	expected := map[string]any{
		"firstZero":                   "",
		"firstFive":                   "Adven",
		"leftBeyond":                  "AdventureWorks",
		"lastZero":                    "",
		"lastFive":                    "Works",
		"rightBeyond":                 "AdventureWorks",
		"ltrimWhitespace":             "AdventureWorks  ",
		"ltrimPrefix":                 "Works",
		"ltrimSuffixNoop":             "AdventureWorks",
		"rtrimWhitespace":             "  AdventureWorks",
		"rtrimSuffix":                 "Adventure",
		"rtrimPrefixNoop":             "AdventureWorks",
		"replicateThree":              "CosmicCosmicCosmic",
		"replicateZero":               "",
		"reverseAdventureWorks":       "skroWerutnevdA",
		"doubleReverseAdventureWorks": "AdventureWorks",
	}
	if !reflect.DeepEqual(row, expected) {
		for key, want := range expected {
			if got := row[key]; got != want {
				t.Fatalf("unexpected additional string function field %s: got %q, want %q; row=%v", key, got, want, row)
			}
		}
		t.Fatalf("unexpected additional string function row: %v", row)
	}
}

func TestCosmosSQLDataPlaneQueryStringFunctionsReturnUndefinedForInvalidInputs(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT VALUE { lowercase: LOWER(\"AdventureWorks\"), uppercase: UPPER(\"AdventureWorks\"), stringLength: LENGTH(\"AdventureWorks\"), substringSuffix: SUBSTRING(\"AdventureWorks\", 9, 5), trimmed: TRIM(\"---AdventureWorks---\", \"-\"), replaced: REPLACE(\"AdventureWorksLT\", \"LT\", \"LT2\"), worksIndex: INDEX_OF(\"AdventureWorks\", \"Works\"), lowerNull: LOWER(null), upperNumber: UPPER(123), lengthArray: LENGTH([1, 2, 3]), leftNumber: LEFT(123, 2), rightBool: RIGHT(true, 2), substringBool: SUBSTRING(true, 0, 2), ltrimNull: LTRIM(null), rtrimNumber: RTRIM(123), trimNumber: TRIM(123), replaceObject: REPLACE({name: \"AdventureWorks\"}, \"Adventure\", \"Contoso\"), indexArray: INDEX_OF([\"Adventure\"], \"Adventure\"), replicateArray: REPLICATE([\"Cosmic\"], 2), reverseNumber: REVERSE(123) }"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("invalid string function query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	expected := []any{map[string]any{
		"lowercase":       "adventureworks",
		"uppercase":       "ADVENTUREWORKS",
		"stringLength":    float64(14),
		"substringSuffix": "Works",
		"trimmed":         "AdventureWorks",
		"replaced":        "AdventureWorksLT2",
		"worksIndex":      float64(9),
	}}
	if !reflect.DeepEqual(docs, expected) {
		t.Fatalf("unexpected invalid string function result: %v", docs)
	}
}

func TestCosmosSQLDataPlaneQueryStringJoinAndSplitFunctions(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT VALUE { joinUsingSpaces: STRINGJOIN([\"Iropa\", \"Mountain\", \"Bike\"], \" \"), joinUsingEmptyString: STRINGJOIN([\"Iropa\", \"Mountain\", \"Bike\"], \"\"), joinUsingUndefined: STRINGJOIN([\"Iropa\", \"Mountain\", \"Bike\"], undefined), splitOnSymbol: STRINGSPLIT(\"CARBON_STEEL_BIKE_WHEEL\", \"_\"), splitOnPhrase: STRINGSPLIT(\"xenmoun mountain bike\", \"moun\"), splitEmptySeparator: STRINGSPLIT(\"Helmet\", \"\"), splitEmptySource: STRINGSPLIT(\"\", \"\") }"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("STRINGJOIN/STRINGSPLIT query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if len(docs) != 1 {
		t.Fatalf("expected one STRINGJOIN/STRINGSPLIT result, got %v", docs)
	}
	row := docs[0].(map[string]any)
	expected := map[string]any{
		"joinUsingSpaces":      "Iropa Mountain Bike",
		"joinUsingEmptyString": "IropaMountainBike",
		"splitOnSymbol":        []any{"CARBON", "STEEL", "BIKE", "WHEEL"},
		"splitOnPhrase":        []any{"xen", " ", "tain bike"},
		"splitEmptySeparator":  []any{"Helmet"},
		"splitEmptySource":     []any{""},
	}
	if !reflect.DeepEqual(row, expected) {
		t.Fatalf("unexpected STRINGJOIN/STRINGSPLIT row: %v", row)
	}
}

func TestCosmosSQLDataPlaneQueryConcatReturnsUndefinedForInvalidInputs(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT VALUE { valid: CONCAT(\"Adventure\", \"Works\", \"LT\"), invalidSingle: CONCAT(\"Adventure\"), invalidNumber: CONCAT(\"Adventure\", 123), invalidBoolean: CONCAT(\"Adventure\", true), invalidArray: CONCAT(\"Adventure\", [\"Works\"]), invalidObject: CONCAT(\"Adventure\", { suffix: \"Works\" }), invalidUndefined: CONCAT(\"Adventure\", undefined) }"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("CONCAT invalid-input query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	expected := []any{map[string]any{
		"valid": "AdventureWorksLT",
	}}
	if !reflect.DeepEqual(docs, expected) {
		t.Fatalf("unexpected CONCAT invalid-input result: %v", docs)
	}
}

func TestCosmosSQLDataPlaneQueryAdditionalTypeAndMathFunctions(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"id":"typed","category":"type","name":"value","active":true,"tags":["a"],"profile":{"kind":"object"},"integer":4,"score":100,"negative":-2.7}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"id":"other","category":"type","name":42,"active":"yes","tags":"not-array","profile":["not-object"],"integer":4.5,"score":10,"negative":3}`), nil))

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT IS_STRING(c.name) AS nameIsString, IS_BOOL(c.active) AS activeIsBool, IS_ARRAY(c.tags) AS tagsIsArray, IS_OBJECT(c.profile) AS profileIsObject, IS_INTEGER(c.integer) AS integerIsInteger, IS_FINITE_NUMBER(c.score) AS scoreIsFinite, IS_FINITE_NUMBER(8.9 / 0.0) AS divisionByZeroIsFinite, IS_FINITE_NUMBER(SQRT(-1.0)) AS nanIsFinite, IS_PRIMITIVE(c.name) AS nameIsPrimitive, POWER(c.integer, 3) AS cubed, LOG10(c.score) AS logScore, SIGN(c.negative) AS sign, TRUNC(c.negative) AS trunc, EXP(0) AS expZero FROM c WHERE IS_BOOL(c.active) AND IS_OBJECT(c.profile) AND IS_INTEGER(c.integer) AND IS_FINITE_NUMBER(c.score)"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("additional type and math query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if len(docs) != 1 {
		t.Fatalf("expected one additional type and math result, got %v", docs)
	}
	row := docs[0].(map[string]any)
	for _, field := range []string{"nameIsString", "activeIsBool", "tagsIsArray", "profileIsObject", "integerIsInteger", "scoreIsFinite", "nameIsPrimitive"} {
		if row[field] != true {
			t.Fatalf("expected %s to be true in row: %v", field, row)
		}
	}
	if row["divisionByZeroIsFinite"] != false || row["nanIsFinite"] != false {
		t.Fatalf("expected infinite and NaN expressions to be non-finite in row: %v", row)
	}
	if row["cubed"] != float64(64) || row["logScore"] != float64(2) || row["sign"] != float64(-1) || row["trunc"] != float64(-2) || row["expZero"] != float64(1) {
		t.Fatalf("unexpected additional math function fields: %v", row)
	}
}

func TestCosmosSQLDataPlaneQueryTrigonometricMathFunctions(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT VALUE { acos: ACOS(-1), asin: ASIN(1), atan: ATAN(1), atn2: ATN2(35.175643, 129.44), cosZero: COS(0), sinPiHalf: SIN(PI() / 2), tanZero: TAN(0), cotPiFourth: COT(PI() / 4), degreesPi: DEGREES(PI()), radians180: RADIANS(180), squareThree: SQUARE(3), squareNull: SQUARE(null) }"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("trigonometric math function query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if len(docs) != 1 {
		t.Fatalf("expected one trigonometric math result, got %v", docs)
	}
	row := docs[0].(map[string]any)
	assertCosmosFloatNear(t, row, "acos", math.Pi, 1e-12)
	assertCosmosFloatNear(t, row, "asin", math.Pi/2, 1e-12)
	assertCosmosFloatNear(t, row, "atan", math.Pi/4, 1e-12)
	assertCosmosFloatNear(t, row, "atn2", 0.265344532064832, 1e-12)
	assertCosmosFloatNear(t, row, "cosZero", 1, 1e-12)
	assertCosmosFloatNear(t, row, "sinPiHalf", 1, 1e-12)
	assertCosmosFloatNear(t, row, "tanZero", 0, 1e-12)
	assertCosmosFloatNear(t, row, "cotPiFourth", 1, 1e-12)
	assertCosmosFloatNear(t, row, "degreesPi", 180, 1e-12)
	assertCosmosFloatNear(t, row, "radians180", math.Pi, 1e-12)
	assertCosmosFloatNear(t, row, "squareThree", 9, 1e-12)
	if _, exists := row["squareNull"]; exists {
		t.Fatalf("expected SQUARE(null) to be omitted as undefined, got row: %v", row)
	}
}

func TestCosmosSQLDataPlaneQueryRandFunction(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT VALUE { first: RAND(), second: RAND() }"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("RAND function query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if len(docs) != 1 {
		t.Fatalf("expected one RAND function result, got %v", docs)
	}
	row := docs[0].(map[string]any)
	for _, field := range []string{"first", "second"} {
		value, ok := row[field].(float64)
		if !ok {
			t.Fatalf("expected RAND field %s to be numeric, got %T: %v", field, row[field], row)
		}
		if value < 0 || value >= 1 {
			t.Fatalf("expected RAND field %s in [0,1), got %v in row: %v", field, value, row)
		}
	}
}

func assertCosmosFloatNear(t *testing.T, row map[string]any, field string, want, tolerance float64) {
	t.Helper()
	got, ok := row[field].(float64)
	if !ok {
		t.Fatalf("expected %s to be a number, got %T: %v", field, row[field], row)
	}
	if math.Abs(got-want) > tolerance {
		t.Fatalf("expected %s near %v, got %v in row: %v", field, want, got, row)
	}
}

func assertCosmosObjectArrayPairs(t *testing.T, value any, keyField, valueField string, expected map[string]any) {
	t.Helper()
	pairs, ok := value.([]any)
	if !ok {
		t.Fatalf("expected OBJECTTOARRAY result to be an array, got %T: %v", value, value)
	}
	got := make(map[string]any, len(pairs))
	for _, pair := range pairs {
		pairMap, ok := pair.(map[string]any)
		if !ok {
			t.Fatalf("expected OBJECTTOARRAY pair to be an object, got %T: %v", pair, pairs)
		}
		key, ok := pairMap[keyField].(string)
		if !ok {
			t.Fatalf("expected OBJECTTOARRAY pair key field %q to be a string, got pair: %v", keyField, pairMap)
		}
		got[key] = pairMap[valueField]
	}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("unexpected OBJECTTOARRAY pairs: got %v, want %v", got, expected)
	}
}

func TestCosmosSQLDataPlaneQueryNumberBinAndIntegerMathFunctions(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"id":"ints","category":"math","left":22,"right":5,"bits":6,"mask":3}`), nil))

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT VALUE { roundToNegativeHundreds: NUMBERBIN(37.752, -100), roundToTens: NUMBERBIN(37.752, 10), roundToOnes: NUMBERBIN(37.752), roundToOneTenths: NUMBERBIN(37.752, 0.1), roundToOneHundreds: NUMBERBIN(37.752, 0.01), add: INTADD(c.left, c.right), sub: INTSUB(c.left, c.right), mul: INTMUL(c.left, c.right), div: INTDIV(c.left, c.right), mod: INTMOD(c.left, c.right), bitAnd: INTBITAND(c.bits, c.mask), bitOr: INTBITOR(c.bits, c.mask), bitXor: INTBITXOR(c.bits, c.mask), bitNot: INTBITNOT(c.bits), shiftLeft: INTBITLEFTSHIFT(c.mask, 2), shiftRight: INTBITRIGHTSHIFT(16, 2) } FROM c WHERE c.id = 'ints' AND INTADD(c.left, c.right) = 27"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("number bin and integer math query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if len(docs) != 1 {
		t.Fatalf("expected one number bin and integer math result, got %v", docs)
	}
	row := docs[0].(map[string]any)
	expected := map[string]any{
		"roundToNegativeHundreds": float64(100),
		"roundToTens":             float64(30),
		"roundToOnes":             float64(37),
		"add":                     float64(27),
		"sub":                     float64(17),
		"mul":                     float64(110),
		"div":                     float64(4),
		"mod":                     float64(2),
		"bitAnd":                  float64(2),
		"bitOr":                   float64(7),
		"bitXor":                  float64(5),
		"bitNot":                  float64(-7),
		"shiftLeft":               float64(12),
		"shiftRight":              float64(4),
	}
	for field, want := range expected {
		if row[field] != want {
			t.Fatalf("unexpected %s: got %v, want %v; row=%v", field, row[field], want, row)
		}
	}
	assertCosmosFloatNear(t, row, "roundToOneTenths", 37.7, 1e-12)
	assertCosmosFloatNear(t, row, "roundToOneHundreds", 37.75, 1e-12)
}

func TestCosmosSQLDataPlaneQueryMultipleOrderByFields(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	for _, payload := range []string{
		`{"id":"b-high","category":"gear","rank":2}`,
		`{"id":"a-high","category":"gear","rank":2}`,
		`{"id":"low","category":"gear","rank":1}`,
		`{"id":"food","category":"food","rank":9}`,
	} {
		_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(payload), nil))
	}

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT c.id FROM c WHERE c.category = 'gear' ORDER BY c.rank DESC, c.id ASC"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("multiple ORDER BY query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	ids := make([]string, 0, len(docs))
	for _, doc := range docs {
		ids = append(ids, doc.(map[string]any)["id"].(string))
	}
	if !reflect.DeepEqual(ids, []string{"a-high", "b-high", "low"}) {
		t.Fatalf("unexpected multiple ORDER BY results: %v", docs)
	}
}

func TestCosmosSQLDataPlaneQueryJoinNestedArrays(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"products","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	for _, payload := range []string{
		`{"id":"jacket","category":"gear","name":"Raiot Jacket","sizes":[{"key":"s","description":"Small"},{"key":"l","description":"Large"},{"key":"xl","description":"Extra Large"}]}`,
		`{"id":"fins","category":"gear","name":"Gremon Fins","sizes":[{"key":"m","description":"Medium"}]}`,
		`{"id":"pack","category":"gear","name":"Tresko Pack","sizes":[{"key":"xl","description":"Extra Large"}]}`,
	} {
		_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/products/docs", []byte(payload), nil))
	}

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/products/docs", []byte(`{"query":"SELECT p.name, s.key AS size FROM p JOIN s IN p.sizes WHERE s.description LIKE '%Large' ORDER BY p.name ASC, s.key ASC"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("JOIN query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if len(docs) != 3 {
		t.Fatalf("expected three JOIN query results, got %v", docs)
	}
	rows := make([]string, 0, len(docs))
	for _, doc := range docs {
		row := doc.(map[string]any)
		name, nameOK := row["name"].(string)
		size, sizeOK := row["size"].(string)
		if !nameOK || !sizeOK {
			t.Fatalf("unexpected JOIN row shape: %v", row)
		}
		rows = append(rows, name+":"+size)
	}
	if !reflect.DeepEqual(rows, []string{"Raiot Jacket:l", "Raiot Jacket:xl", "Tresko Pack:xl"}) {
		t.Fatalf("unexpected JOIN query results: %v", docs)
	}
}

func TestCosmosSQLDataPlaneQueryJSONExpressions(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"products","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	for _, payload := range []string{
		`{"id":"watch-a","category":"apparel","detailCategory":"apparel-accessories-watches","slug":"diannis-watch","sku":"64801","name":"Diannis Watch","quantity":159,"price":98,"metadata":{"link":"https://www.adventure-works.com/diannis-watch.p"}}`,
		`{"id":"pack-a","category":"gear","detailCategory":"gear-packs","slug":"pack","sku":"90001","name":"Trail Pack","quantity":12,"price":42,"metadata":{"link":"https://www.adventure-works.com/trail-pack.p"}}`,
	} {
		_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/products/docs", []byte(payload), nil))
	}

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/products/docs", []byte(`{"query":"SELECT VALUE {\"id\": p.sku, \"name\": p.name, \"category\": {\"department\": p.category, \"section\": p.detailCategory}, \"metadata\": [p.category, p.slug, p.metadata.link], \"financial\": {\"listPrice\": p.price}, \"stocked\": IIF(p.quantity > 0, true, false)} FROM p WHERE p.detailCategory = 'apparel-accessories-watches'"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("JSON expression query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if len(docs) != 1 {
		t.Fatalf("expected one JSON expression result, got %v", docs)
	}
	row := docs[0].(map[string]any)
	if row["id"] != "64801" || row["name"] != "Diannis Watch" || row["stocked"] != true {
		t.Fatalf("unexpected JSON expression top-level fields: %v", row)
	}
	if !reflect.DeepEqual(row["category"], map[string]any{"department": "apparel", "section": "apparel-accessories-watches"}) {
		t.Fatalf("unexpected JSON expression category: %v", row)
	}
	if !reflect.DeepEqual(row["metadata"], []any{"apparel", "diannis-watch", "https://www.adventure-works.com/diannis-watch.p"}) {
		t.Fatalf("unexpected JSON expression metadata: %v", row)
	}
	if !reflect.DeepEqual(row["financial"], map[string]any{"listPrice": float64(98)}) {
		t.Fatalf("unexpected JSON expression financial: %v", row)
	}
}

func TestCosmosSQLDataPlaneQueryUnnamedProjectionUsesGeneratedAliases(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"products","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/products/docs", []byte(`{"id":"shoe-a","category":"footwear","name":"Remdriel Shoes","sku":"61506"}`), nil))

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/products/docs", []byte(`{"query":"SELECT { identifier: p.name, model: p.sku } FROM p WHERE p.id = 'shoe-a'"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("unnamed projection query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if len(docs) != 1 {
		t.Fatalf("expected one unnamed projection result, got %v", docs)
	}
	row := docs[0].(map[string]any)
	if !reflect.DeepEqual(row, map[string]any{"$1": map[string]any{"identifier": "Remdriel Shoes", "model": "61506"}}) {
		t.Fatalf("unexpected generated alias projection: %v", row)
	}
}

func TestCosmosSQLDataPlaneQuerySelectWithoutFromUsesSingleSyntheticRow(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"products","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/products/docs", []byte(`{"query":"SELECT \"Cosmic\", \"Works\""}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("static SELECT query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if !reflect.DeepEqual(docs, []any{map[string]any{"$1": "Cosmic", "$2": "Works"}}) {
		t.Fatalf("unexpected static SELECT rows: %v", docs)
	}

	valueResp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/products/docs", []byte(`{"query":"SELECT VALUE \"Cosmic Works\""}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("static SELECT VALUE query returned error: %v", err)
	}
	valueDocs := decodeCosmosResponse(t, valueResp)["Documents"].([]any)
	if !reflect.DeepEqual(valueDocs, []any{"Cosmic Works"}) {
		t.Fatalf("unexpected static SELECT VALUE rows: %v", valueDocs)
	}
}

func TestCosmosSQLDataPlaneQueryArrayContainsAllAny(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"id":"functions","category":"fn","tags":["coats","jackets"],"sizes":[1,2,3]}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"id":"other","category":"fn","tags":["boots"],"sizes":[4]}`), nil))

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT VALUE { allTags: ARRAY_CONTAINS_ALL(c.tags, 'coats', 'jackets'), allTagsMissing: ARRAY_CONTAINS_ALL(c.tags, 'coats', 'boots'), anyTag: ARRAY_CONTAINS_ANY(c.tags, 'boots', 'jackets'), anyTagMissing: ARRAY_CONTAINS_ANY(c.tags, 'boots', 'hats'), nestedAll: ARRAY_CONTAINS_ALL([{ id: 'a', sizes: [1, 2] }, true, '3'], { id: 'a', sizes: [1, 2] }, true, '3'), allWithUndefinedMatch: ARRAY_CONTAINS_ALL([1, 2, 3], 1, undefined), allWithUndefinedMiss: ARRAY_CONTAINS_ALL([1, 2, 3], 4, undefined), anyWithUndefinedMatch: ARRAY_CONTAINS_ANY([1, 2, 3], 1, undefined), anyWithUndefinedMiss: ARRAY_CONTAINS_ANY([1, 2, 3], 4, undefined), emptyAny: ARRAY_CONTAINS_ANY([], 1), invalidAllArray: ARRAY_CONTAINS_ALL('not-array', 1), invalidAnyArray: ARRAY_CONTAINS_ANY('not-array', 1), invalidAllMissingNeedle: ARRAY_CONTAINS_ALL([1, 2, 3]), invalidAnyMissingNeedle: ARRAY_CONTAINS_ANY([1, 2, 3]), invalidAllUndefinedArray: ARRAY_CONTAINS_ALL(undefined, 1), invalidAnyUndefinedArray: ARRAY_CONTAINS_ANY(undefined, 1) } FROM c WHERE c.id = 'functions' AND ARRAY_CONTAINS_ALL(c.tags, 'coats', 'jackets') AND ARRAY_CONTAINS_ANY(c.tags, 'boots', 'coats')"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("ARRAY_CONTAINS_ALL/ANY query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if len(docs) != 1 {
		t.Fatalf("expected one ARRAY_CONTAINS_ALL/ANY result, got %v", docs)
	}
	row := docs[0].(map[string]any)
	expected := map[string]any{
		"allTags":               true,
		"allTagsMissing":        false,
		"anyTag":                true,
		"anyTagMissing":         false,
		"nestedAll":             true,
		"allWithUndefinedMiss":  false,
		"anyWithUndefinedMatch": true,
		"emptyAny":              false,
	}
	if !reflect.DeepEqual(row, expected) {
		t.Fatalf("unexpected ARRAY_CONTAINS_ALL/ANY row: %v", row)
	}
}

func TestCosmosSQLDataPlaneQueryArrayContainsReturnsUndefinedForInvalidInputs(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT VALUE { containsItem: ARRAY_CONTAINS([\"coats\", \"jackets\", \"sweatshirts\"], \"coats\"), missingItem: ARRAY_CONTAINS([\"coats\", \"jackets\", \"sweatshirts\"], \"hoodies\"), containsFullMatchObject: ARRAY_CONTAINS([{ category: \"shirts\", color: \"blue\" }], { category: \"shirts\", color: \"blue\" }), missingFullMatchObject: ARRAY_CONTAINS([{ category: \"shirts\", color: \"blue\" }], { category: \"shirts\" }), containsPartialMatchObject: ARRAY_CONTAINS([{ category: \"shirts\", color: \"blue\" }], { category: \"shirts\" }, true), missingPartialMatchObject: ARRAY_CONTAINS([{ category: \"shirts\", color: \"blue\" }], { category: \"shorts\", color: \"blue\" }, true), invalidArray: ARRAY_CONTAINS(\"coats\", \"coats\"), invalidMissingNeedle: ARRAY_CONTAINS([\"coats\"]), invalidPartialFlagString: ARRAY_CONTAINS([{ category: \"shirts\", color: \"blue\" }], { category: \"shirts\" }, \"true\"), invalidPartialFlagUndefined: ARRAY_CONTAINS([{ category: \"shirts\", color: \"blue\" }], { category: \"shirts\" }, undefined) }"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("ARRAY_CONTAINS invalid-input query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	expected := []any{map[string]any{
		"containsItem":               true,
		"missingItem":                false,
		"containsFullMatchObject":    true,
		"missingFullMatchObject":     false,
		"containsPartialMatchObject": true,
		"missingPartialMatchObject":  false,
	}}
	if !reflect.DeepEqual(docs, expected) {
		t.Fatalf("unexpected ARRAY_CONTAINS invalid-input result: %v", docs)
	}
}

func TestCosmosSQLDataPlaneQueryDocumentIDFunction(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"id":"alpha","category":"docid","name":"Alpha"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"id":"beta","category":"docid","name":"Beta"}`), nil))

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT c.id, c._rid, DOCUMENTID(c) AS documentId FROM c WHERE c.category = 'docid' AND DOCUMENTID(c) > 0 ORDER BY c.id"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("DOCUMENTID query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if len(docs) != 2 {
		t.Fatalf("expected two DOCUMENTID results, got %v", docs)
	}
	for i, doc := range docs {
		row := doc.(map[string]any)
		if row["id"] != []string{"alpha", "beta"}[i] {
			t.Fatalf("unexpected DOCUMENTID row order/content: %v", docs)
		}
		if row["_rid"] == "" {
			t.Fatalf("expected _rid in DOCUMENTID row: %v", row)
		}
		documentID, ok := row["documentId"].(float64)
		if !ok || documentID <= 0 || documentID != math.Trunc(documentID) {
			t.Fatalf("expected positive integer DOCUMENTID value, got %T %v in row: %v", row["documentId"], row["documentId"], row)
		}
	}
}

func TestCosmosSQLDataPlaneQueryAdditionalArraySetFunctions(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT VALUE { chooseZero: CHOOSE(0, \"Mt.\", \"Hood\", \"Hydration\", \"Pack\"), chooseOne: CHOOSE(1, \"Mt.\", \"Hood\", \"Hydration\", \"Pack\"), chooseTwo: CHOOSE(2, \"Mt.\", \"Hood\", \"Hydration\", \"Pack\"), chooseFour: CHOOSE(4, \"Mt.\", \"Hood\", \"Hydration\", \"Pack\"), chooseFive: CHOOSE(5, \"Mt.\", \"Hood\", \"Hydration\", \"Pack\"), simpleIntersect: SETINTERSECT([1, 2, 3, 4], [3, 4, 5, 6]), emptyIntersect: SETINTERSECT([1, 2, 3, 4], []), duplicatesIntersect: SETINTERSECT([1, 2, 3, 4], [1, 1, 1, 1]), noMatchesIntersect: SETINTERSECT([1, 2, 3, 4], [\"A\", \"B\"]), unorderedIntersect: SETINTERSECT([1, 2, \"A\", \"B\"], [\"A\", 1]), invalidIntersectSingle: SETINTERSECT([1, 2, 3, 4]), invalidIntersectString: SETINTERSECT([1, 2, 3, 4], \"not-array\"), invalidIntersectExtra: SETINTERSECT([1], [1], [1]), simpleUnion: SETUNION([1, 2, 3, 4], [3, 4, 5, 6]), emptyUnion: SETUNION([1, 2, 3, 4], []), duplicatesUnion: SETUNION([1, 2, 3, 4], [1, 1, 1, 1]), unorderedUnion: SETUNION([1, 2, \"A\", \"B\"], [\"A\", 1]), invalidUnionSingle: SETUNION([1, 2, 3, 4]), invalidUnionString: SETUNION([1, 2, 3, 4], \"not-array\"), invalidUnionExtra: SETUNION([1], [1], [1]) }"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("additional array set function query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if len(docs) != 1 {
		t.Fatalf("expected one additional array set function result, got %v", docs)
	}
	row := docs[0].(map[string]any)
	expected := map[string]any{
		"chooseOne":           "Mt.",
		"chooseTwo":           "Hood",
		"chooseFour":          "Pack",
		"simpleIntersect":     []any{float64(3), float64(4)},
		"emptyIntersect":      []any{},
		"duplicatesIntersect": []any{float64(1)},
		"noMatchesIntersect":  []any{},
		"unorderedIntersect":  []any{"A", float64(1)},
		"simpleUnion":         []any{float64(1), float64(2), float64(3), float64(4), float64(5), float64(6)},
		"emptyUnion":          []any{float64(1), float64(2), float64(3), float64(4)},
		"duplicatesUnion":     []any{float64(1), float64(2), float64(3), float64(4)},
		"unorderedUnion":      []any{float64(1), float64(2), "A", "B"},
	}
	if !reflect.DeepEqual(row, expected) {
		t.Fatalf("unexpected additional array set function row: %v", row)
	}
}

func TestCosmosSQLDataPlaneQueryObjectToArrayFunction(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT VALUE { defaultPairs: OBJECTTOARRAY({ \"a\": \"12345\", \"b\": \"67890\" }), customPairs: OBJECTTOARRAY({ \"color\": \"blue\", \"size\": \"small\" }, \"key\", \"value\"), invalidPairs: OBJECTTOARRAY([\"not\", \"object\"]), invalidExtra: OBJECTTOARRAY({ \"a\": \"12345\" }, \"key\", \"value\", \"extra\") }"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("OBJECTTOARRAY query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if len(docs) != 1 {
		t.Fatalf("expected one OBJECTTOARRAY result, got %v", docs)
	}
	row := docs[0].(map[string]any)
	for _, field := range []string{"invalidPairs", "invalidExtra"} {
		if _, exists := row[field]; exists {
			t.Fatalf("expected invalid OBJECTTOARRAY field %q to be omitted as undefined, got row: %v", field, row)
		}
	}
	assertCosmosObjectArrayPairs(t, row["defaultPairs"], "k", "v", map[string]any{
		"a": "12345",
		"b": "67890",
	})
	assertCosmosObjectArrayPairs(t, row["customPairs"], "key", "value", map[string]any{
		"color": "blue",
		"size":  "small",
	})
}

func TestCosmosSQLDataPlaneQueryConversionFunctions(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT VALUE { parseIntegerString: STRINGTONUMBER(\"100\"), parseDecimalString: STRINGTONUMBER(\"3.14\"), parseWithWhitespace: STRINGTONUMBER(\"   60   \"), parseScientific: STRINGTONUMBER(\"-1.79769e+308\"), parseInvalid: STRINGTONUMBER(\"Hello\"), parseUndefined: STRINGTONUMBER(undefined), parseBooleanString: STRINGTOBOOLEAN(\"true\"), parseBooleanWithWhitespace: STRINGTOBOOLEAN(\"  false  \"), parseBooleanInvalid: STRINGTOBOOLEAN(null), integerToString: TOSTRING(125), floatToString: TOSTRING(0.1234), booleanToString: TOSTRING(false), arrayToString: TOSTRING([ 1, 2, 3 ]), objectToString: TOSTRING({ \"department\": \"Bicycles\" }), stringToString: TOSTRING(\"Hello World\"), undefinedToString: TOSTRING(undefined) }"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("conversion function query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if len(docs) != 1 {
		t.Fatalf("expected one conversion function result, got %v", docs)
	}
	row := docs[0].(map[string]any)
	expected := map[string]any{
		"parseIntegerString":         float64(100),
		"parseDecimalString":         3.14,
		"parseWithWhitespace":        float64(60),
		"parseScientific":            -1.79769e+308,
		"parseBooleanString":         true,
		"parseBooleanWithWhitespace": false,
		"integerToString":            "125",
		"floatToString":              "0.1234",
		"booleanToString":            "false",
		"arrayToString":              "[1,2,3]",
		"objectToString":             `{"department":"Bicycles"}`,
		"stringToString":             "Hello World",
	}
	if !reflect.DeepEqual(row, expected) {
		t.Fatalf("unexpected conversion function row: %v", row)
	}
}

func TestCosmosSQLDataPlaneQueryJSONStringConversionFunctions(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT VALUE { parseEmptyArray: STRINGTOARRAY(\"[]\"), parseArray: STRINGTOARRAY('[ \"coats\", \"gloves\", \"hats\" ]'), complexArray: STRINGTOARRAY('[ { \"types\": [ \"coats\", \"gloves\" ] }, [ \"hats\" ], 76, false, null ]'), invalidArray: STRINGTOARRAY(\"[ 'coats', 'gloves', 'hats' ]\"), parseNullString: STRINGTONULL(\"null\"), parseWithWhitespace: STRINGTONULL(\"  null  \"), parseUppercase: STRINGTONULL(\"NULL\"), parseNullArgument: STRINGTONULL(null), parseEmptyObject: STRINGTOOBJECT(\"{}\"), parseObjectWithProperty: STRINGTOOBJECT('{\"isAvailable\": true}'), parseObjectNested: STRINGTOOBJECT('{\"division\": {\"name\": \"Sales\"}}'), parseObjectInvalidJson: STRINGTOOBJECT(\"{'price': 27.55}\") }"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("JSON string conversion function query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if len(docs) != 1 {
		t.Fatalf("expected one JSON string conversion function result, got %v", docs)
	}
	row := docs[0].(map[string]any)
	expected := map[string]any{
		"parseEmptyArray":     []any{},
		"parseArray":          []any{"coats", "gloves", "hats"},
		"complexArray":        []any{map[string]any{"types": []any{"coats", "gloves"}}, []any{"hats"}, float64(76), false, nil},
		"parseNullString":     nil,
		"parseWithWhitespace": nil,
		"parseEmptyObject":    map[string]any{},
		"parseObjectWithProperty": map[string]any{
			"isAvailable": true,
		},
		"parseObjectNested": map[string]any{
			"division": map[string]any{
				"name": "Sales",
			},
		},
	}
	if !reflect.DeepEqual(row, expected) {
		t.Fatalf("unexpected JSON string conversion function row: %v", row)
	}
}

func TestCosmosSQLDataPlaneQueryDateTimeFunctions(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT VALUE { addOneYear: DATETIMEADD(\"yyyy\", 1, \"2020-07-03T00:00:00.0000000\"), addOneMonth: DATETIMEADD(\"mm\", 1, \"2020-07-03T00:00:00.0000000\"), addOneDay: DATETIMEADD(\"dd\", 1, \"2020-07-03T00:00:00.0000000\"), addOneHour: DATETIMEADD(\"hh\", 1, \"2020-07-03T00:00:00.0000000\"), subtractOneSecond: DATETIMEADD(\"ss\", -1, \"2020-07-03T00:00:00.0000000\"), diffFutureMonths: DATETIMEDIFF(\"mm\", \"2018-03-05T05:00:00.0000000\", \"2019-02-04T16:00:00.0000000\"), diffFutureDays: DATETIMEDIFF(\"dd\", \"2018-03-05T05:00:00.0000000\", \"2019-02-04T16:00:00.0000000\"), diffFutureHours: DATETIMEDIFF(\"hh\", \"2018-03-05T05:00:00.0000000\", \"2019-02-04T16:00:00.0000000\"), getYear: DATETIMEPART(\"yyyy\", \"2016-05-29T08:30:00.1301617\"), getMonth: DATETIMEPART(\"mm\", \"2016-05-29T08:30:00.1301617\"), getDay: DATETIMEPART(\"dd\", \"2016-05-29T08:30:00.1301617\"), getHour: DATETIMEPART(\"hh\", \"2016-05-29T08:30:00.1301617\"), getMinute: DATETIMEPART(\"mi\", \"2016-05-29T08:30:00.1301617\"), getSecond: DATETIMEPART(\"ss\", \"2016-05-29T08:30:00.1301617\"), getMillisecond: DATETIMEPART(\"ms\", \"2016-05-29T08:30:00.1301617\"), getMicrosecond: DATETIMEPART(\"mcs\", \"2016-05-29T08:30:00.1301617\"), getNanosecond: DATETIMEPART(\"ns\", \"2016-05-29T08:30:00.1301617\"), invalidPart: DATETIMEPART(\"weekday\", \"2016-05-29T08:30:00.1301617\") }"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("date/time function query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if len(docs) != 1 {
		t.Fatalf("expected one date/time function result, got %v", docs)
	}
	row := docs[0].(map[string]any)
	expected := map[string]any{
		"addOneYear":        "2021-07-03T00:00:00.0000000Z",
		"addOneMonth":       "2020-08-03T00:00:00.0000000Z",
		"addOneDay":         "2020-07-04T00:00:00.0000000Z",
		"addOneHour":        "2020-07-03T01:00:00.0000000Z",
		"subtractOneSecond": "2020-07-02T23:59:59.0000000Z",
		"diffFutureMonths":  float64(11),
		"diffFutureDays":    float64(336),
		"diffFutureHours":   float64(8075),
		"getYear":           float64(2016),
		"getMonth":          float64(5),
		"getDay":            float64(29),
		"getHour":           float64(8),
		"getMinute":         float64(30),
		"getSecond":         float64(0),
		"getMillisecond":    float64(130),
		"getMicrosecond":    float64(130161),
		"getNanosecond":     float64(130161700),
	}
	if !reflect.DeepEqual(row, expected) {
		t.Fatalf("unexpected date/time function row: %v", row)
	}
}

func TestCosmosSQLDataPlaneQueryTimestampFunctions(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT VALUE { timestamp: DATETIMETOTIMESTAMP(\"2015-05-19T12:00:00.0000000\"), ticks: DATETIMETOTICKS(\"2015-05-19T12:00:00.0000000\"), parseTimestamp: TIMESTAMPTODATETIME(1597360794300), parseUnixEpoch: TIMESTAMPTODATETIME(0), parseWindowsEpoch: TIMESTAMPTODATETIME(-11644473600000), parseTicks: TICKSTODATETIME(15973607943002652), parseTickUnixEpoch: TICKSTODATETIME(0), parseTickWindowsEpoch: TICKSTODATETIME(-116444736000000000), invalidTimestamp: DATETIMETOTIMESTAMP(\"not-a-date\"), invalidTicks: TICKSTODATETIME(\"not-a-number\") }"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("timestamp function query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if len(docs) != 1 {
		t.Fatalf("expected one timestamp function result, got %v", docs)
	}
	row := docs[0].(map[string]any)
	expected := map[string]any{
		"timestamp":             float64(1432036800000),
		"ticks":                 float64(14320368000000000),
		"parseTimestamp":        "2020-08-13T23:19:54.3000000Z",
		"parseUnixEpoch":        "1970-01-01T00:00:00.0000000Z",
		"parseWindowsEpoch":     "1601-01-01T00:00:00.0000000Z",
		"parseTicks":            "2020-08-13T23:19:54.3002652Z",
		"parseTickUnixEpoch":    "1970-01-01T00:00:00.0000000Z",
		"parseTickWindowsEpoch": "1601-01-01T00:00:00.0000000Z",
	}
	if !reflect.DeepEqual(row, expected) {
		t.Fatalf("unexpected timestamp function row: %v", row)
	}
}

func TestCosmosSQLDataPlaneQueryCurrentDateTimeFunctions(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))

	before := time.Now().UTC()
	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT VALUE { currentDateTime: GETCURRENTDATETIME(), currentTimestamp: GETCURRENTTIMESTAMP(), currentTicks: GETCURRENTTICKS() }"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	after := time.Now().UTC()
	if err != nil {
		t.Fatalf("current date/time function query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if len(docs) != 1 {
		t.Fatalf("expected one current date/time function result, got %v", docs)
	}
	row := docs[0].(map[string]any)

	currentDateTimeRaw, ok := row["currentDateTime"].(string)
	if !ok {
		t.Fatalf("expected currentDateTime string, got %T: %v", row["currentDateTime"], row)
	}
	currentDateTime, ok := cosmosParseDateTime(currentDateTimeRaw)
	if !ok {
		t.Fatalf("expected currentDateTime to parse as Cosmos ISO date/time, got %q", currentDateTimeRaw)
	}
	if currentDateTime.Before(before.Add(-time.Second)) || currentDateTime.After(after.Add(time.Second)) {
		t.Fatalf("currentDateTime %s outside expected range %s..%s", currentDateTimeRaw, before, after)
	}

	currentTimestamp, ok := row["currentTimestamp"].(float64)
	if !ok {
		t.Fatalf("expected currentTimestamp number, got %T: %v", row["currentTimestamp"], row)
	}
	beforeMillis := before.Unix()*1000 + int64(before.Nanosecond()/int(time.Millisecond))
	afterMillis := after.Unix()*1000 + int64(after.Nanosecond()/int(time.Millisecond))
	if int64(currentTimestamp) < beforeMillis-1000 || int64(currentTimestamp) > afterMillis+1000 {
		t.Fatalf("currentTimestamp %v outside expected range %d..%d", currentTimestamp, beforeMillis, afterMillis)
	}

	currentTicks, ok := row["currentTicks"].(float64)
	if !ok {
		t.Fatalf("expected currentTicks number, got %T: %v", row["currentTicks"], row)
	}
	beforeTicks := before.Unix()*10000000 + int64(before.Nanosecond()/100)
	afterTicks := after.Unix()*10000000 + int64(after.Nanosecond()/100)
	if int64(currentTicks) < beforeTicks-10000000 || int64(currentTicks) > afterTicks+10000000 {
		t.Fatalf("currentTicks %v outside expected range %d..%d", currentTicks, beforeTicks, afterTicks)
	}
}

func TestCosmosSQLDataPlaneQueryStaticCurrentDateTimeFunctions(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"id":"one","category":"static-time"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"id":"two","category":"static-time"}`), nil))

	before := time.Now().UTC()
	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT c.id, GETCURRENTDATETIMESTATIC() AS staticDateTime, GETCURRENTTIMESTAMPSTATIC() AS staticTimestamp, GETCURRENTTICKSSTATIC() AS staticTicks FROM c WHERE c.category = 'static-time' ORDER BY c.id"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	after := time.Now().UTC()
	if err != nil {
		t.Fatalf("static current date/time function query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if len(docs) != 2 {
		t.Fatalf("expected two static current date/time function results, got %v", docs)
	}

	first := docs[0].(map[string]any)
	second := docs[1].(map[string]any)
	if first["id"] != "one" || second["id"] != "two" {
		t.Fatalf("unexpected static current date/time row order: %v", docs)
	}
	if first["staticDateTime"] != second["staticDateTime"] || first["staticTimestamp"] != second["staticTimestamp"] || first["staticTicks"] != second["staticTicks"] {
		t.Fatalf("expected static current date/time values to be stable across rows, got %v", docs)
	}

	staticDateTimeRaw, ok := first["staticDateTime"].(string)
	if !ok {
		t.Fatalf("expected staticDateTime string, got %T: %v", first["staticDateTime"], docs)
	}
	staticDateTime, ok := cosmosParseDateTime(staticDateTimeRaw)
	if !ok || staticDateTime.Before(before.Add(-time.Second)) || staticDateTime.After(after.Add(time.Second)) {
		t.Fatalf("staticDateTime %q outside expected range %s..%s", staticDateTimeRaw, before, after)
	}
	staticTimestamp, ok := first["staticTimestamp"].(float64)
	if !ok {
		t.Fatalf("expected staticTimestamp number, got %T: %v", first["staticTimestamp"], docs)
	}
	staticTicks, ok := first["staticTicks"].(float64)
	if !ok {
		t.Fatalf("expected staticTicks number, got %T: %v", first["staticTicks"], docs)
	}
	if staticTimestamp <= 0 || staticTicks <= 0 {
		t.Fatalf("expected positive static timestamp/ticks, got %v", docs)
	}
}

func TestCosmosSQLDataPlaneQueryDateTimeFromPartsAndBin(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT VALUE { constructMinArguments: DATETIMEFROMPARTS(2017, 4, 20), constructMinEquivalent: DATETIMEFROMPARTS(2017, 4, 20, 0, 0, 0, 0), constructAllArguments: DATETIMEFROMPARTS(2017, 4, 20, 13, 15, 20, 3456789), constructPartialArguments: DATETIMEFROMPARTS(2017, 4, 20, 13, 15), constructInvalidArguments: DATETIMEFROMPARTS(-2000, -1, -1), binDay: DATETIMEBIN(\"2021-01-08T18:35:00.0000000\", \"dd\"), binHour: DATETIMEBIN(\"2021-01-08T18:35:00.0000000\", \"hh\"), binSecond: DATETIMEBIN(\"2021-01-08T18:35:00.0000000\", \"ss\"), binFiveHours: DATETIMEBIN(\"2021-01-08T18:35:00.0000000\", \"hh\", 5), binSevenDaysUnixEpoch: DATETIMEBIN(\"2021-01-08T18:35:00.0000000\", \"dd\", 7), binSevenDaysWindowsEpoch: DATETIMEBIN(\"2021-01-08T18:35:00.0000000\", \"dd\", 7, \"1601-01-01T00:00:00.0000000\"), binInvalidSize: DATETIMEBIN(\"2021-01-08T18:35:00.0000000\", \"dd\", 0) }"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("date/time construction query returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if len(docs) != 1 {
		t.Fatalf("expected one date/time construction result, got %v", docs)
	}
	row := docs[0].(map[string]any)
	expected := map[string]any{
		"constructMinArguments":     "2017-04-20T00:00:00.0000000Z",
		"constructMinEquivalent":    "2017-04-20T00:00:00.0000000Z",
		"constructAllArguments":     "2017-04-20T13:15:20.3456789Z",
		"constructPartialArguments": "2017-04-20T13:15:00.0000000Z",
		"binDay":                    "2021-01-08T00:00:00.0000000Z",
		"binHour":                   "2021-01-08T18:00:00.0000000Z",
		"binSecond":                 "2021-01-08T18:35:00.0000000Z",
		"binFiveHours":              "2021-01-08T15:00:00.0000000Z",
		"binSevenDaysUnixEpoch":     "2021-01-07T00:00:00.0000000Z",
		"binSevenDaysWindowsEpoch":  "2021-01-04T00:00:00.0000000Z",
	}
	if !reflect.DeepEqual(row, expected) {
		t.Fatalf("unexpected date/time construction row: %v", row)
	}
}

func TestCosmosSQLDataPlaneQueryExistsArraySubquery(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"products","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	for _, payload := range []string{
		`{"id":"jacket","category":"gear","name":"Raiot Jacket","sizes":[{"key":"s","description":"Small"},{"key":"l","description":"Large"}]}`,
		`{"id":"fins","category":"gear","name":"Comet Fins","sizes":[{"key":"m","description":"Medium"}]}`,
		`{"id":"pack","category":"gear","name":"Tresko Pack","sizes":[{"key":"xl","description":"Extra Large"}]}`,
	} {
		_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/products/docs", []byte(payload), nil))
	}

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/products/docs", []byte(`{"query":"SELECT VALUE p.name FROM p WHERE EXISTS(SELECT VALUE s FROM s IN p.sizes WHERE s.description LIKE '%Large') ORDER BY p.name ASC"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("EXISTS subquery returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if !reflect.DeepEqual(docs, []any{"Raiot Jacket", "Tresko Pack"}) {
		t.Fatalf("unexpected EXISTS subquery results: %v", docs)
	}
}

func TestCosmosSQLDataPlaneQueryArraySubqueryProjection(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"products","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	for _, payload := range []string{
		`{"id":"sandals","category":"apparel","name":"Menti Sandals","sizes":[{"key":"5"},{"key":"6"},{"key":"7"},{"key":"8"},{"key":"9"}]}`,
		`{"id":"boots","category":"apparel","name":"Other Boots","sizes":[{"key":"10"}]}`,
	} {
		_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/products/docs", []byte(payload), nil))
	}

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/products/docs", []byte(`{"query":"SELECT p.name, ARRAY(SELECT VALUE s.key FROM s IN p.sizes) AS sizes FROM p WHERE p.name = 'Menti Sandals'"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("ARRAY subquery projection returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if len(docs) != 1 {
		t.Fatalf("expected one ARRAY subquery projection result, got %v", docs)
	}
	row := docs[0].(map[string]any)
	if row["name"] != "Menti Sandals" {
		t.Fatalf("unexpected ARRAY subquery row name: %v", row)
	}
	if !reflect.DeepEqual(row["sizes"], []any{"5", "6", "7", "8", "9"}) {
		t.Fatalf("unexpected ARRAY subquery projection: %v", row)
	}
}

func TestCosmosSQLDataPlaneQueryScalarAggregateSubqueryProjection(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"products","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	for _, payload := range []string{
		`{"id":"boots","category":"gear","name":"Blators Snowboard Boots","colors":["turquoise","cobalt","jam","galliano","violet"]}`,
		`{"id":"sandals","category":"gear","name":"Menti Sandals","colors":["white"]}`,
	} {
		_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/products/docs", []byte(payload), nil))
	}

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/products/docs", []byte(`{"query":"SELECT p.name, (SELECT VALUE COUNT(1) FROM c IN p.colors) AS colorsCount, (SELECT VALUE COUNT(1) FROM c IN p.colors WHERE c LIKE '%t') AS colorsEndsWithTCount FROM p WHERE p.id = 'boots'"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("scalar aggregate subquery returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if len(docs) != 1 {
		t.Fatalf("expected one scalar aggregate subquery result, got %v", docs)
	}
	row, ok := docs[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected scalar aggregate subquery result shape: %v", docs)
	}
	if row["name"] != "Blators Snowboard Boots" || row["colorsCount"] != float64(5) || row["colorsEndsWithTCount"] != float64(2) {
		t.Fatalf("unexpected scalar aggregate subquery row: %v", row)
	}
}

func TestCosmosSQLDataPlaneQueryScalarAggregateSubqueryFunctions(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"products","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	for _, payload := range []string{
		`{"id":"stocked","category":"gear","name":"Trail Pack","inventory":[{"location":"east","quantity":1},{"location":"west","quantity":5},{"location":"north","quantity":2},{"location":"south","quantity":3}]}`,
		`{"id":"empty","category":"gear","name":"Empty Pack","inventory":[]}`,
	} {
		_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/products/docs", []byte(payload), nil))
	}

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/products/docs", []byte(`{"query":"SELECT p.name, (SELECT VALUE SUM(i.quantity) FROM i IN p.inventory) AS totalQuantity, (SELECT VALUE AVG(i.quantity) FROM i IN p.inventory) AS averageQuantity, (SELECT VALUE MIN(i.quantity) FROM i IN p.inventory WHERE i.quantity > 1) AS minRestockQuantity, (SELECT VALUE MAX(i.quantity) FROM i IN p.inventory) AS maxQuantity FROM p WHERE p.id = 'stocked'"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("scalar aggregate subquery functions returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if len(docs) != 1 {
		t.Fatalf("expected one scalar aggregate function result, got %v", docs)
	}
	row := docs[0].(map[string]any)
	if row["name"] != "Trail Pack" || row["totalQuantity"] != float64(11) || row["averageQuantity"] != 2.75 || row["minRestockQuantity"] != float64(2) || row["maxQuantity"] != float64(5) {
		t.Fatalf("unexpected scalar aggregate function row: %v", row)
	}
}

func TestCosmosSQLDataPlaneQuerySubqueryJoinFiltersArrays(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"products","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	for _, payload := range []string{
		`{"id":"jacket","category":"gear","name":"Raiot Jacket","tags":[{"key":"fabric"},{"key":"style"}],"sizes":[{"key":"s","order":1},{"key":"l","order":4},{"key":"xl","order":5}],"colors":["charcoal-gray","red"]}`,
		`{"id":"pack","category":"gear","name":"Tresko Pack","tags":[{"key":"material"},{"key":"volume"}],"sizes":[{"key":"m","order":2},{"key":"xl","order":6}],"colors":["blue-gray","graystone"]}`,
		`{"id":"fins","category":"gear","name":"Comet Fins","tags":[{"key":"finish"}],"sizes":[{"key":"m","order":4}],"colors":["gray"]}`,
	} {
		_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/products/docs", []byte(payload), nil))
	}

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/products/docs", []byte(`{"query":"SELECT VALUE COUNT(1) FROM p JOIN (SELECT VALUE t FROM t IN p.tags WHERE t.key IN ('fabric', 'material')) JOIN (SELECT VALUE s FROM s IN p.sizes WHERE s[\"order\"] >= 3) JOIN (SELECT VALUE c FROM c IN p.colors WHERE c LIKE '%gray%')"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("subquery JOIN returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if !reflect.DeepEqual(docs, []any{float64(4)}) {
		t.Fatalf("unexpected subquery JOIN count: %v", docs)
	}
}

func TestCosmosSQLDataPlaneQuerySimpleScalarSubqueryProjection(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"products","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/products/docs", []byte(`{"id":"00000000-0000-0000-0000-000000004041","category":"apparel","name":"Remdriel Shoes"}`), nil))

	resp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/products/docs", []byte(`{"query":"SELECT (SELECT VALUE CONCAT('ID-', p.id)) AS internalId FROM p WHERE p.name = 'Remdriel Shoes'"}`), map[string]string{
		"x-ms-documentdb-isquery": "True",
	}))
	if err != nil {
		t.Fatalf("simple scalar subquery returned error: %v", err)
	}
	docs := decodeCosmosResponse(t, resp)["Documents"].([]any)
	if len(docs) != 1 {
		t.Fatalf("expected one simple scalar subquery result, got %v", docs)
	}
	row := docs[0].(map[string]any)
	if row["internalId"] != "ID-00000000-0000-0000-0000-000000004041" {
		t.Fatalf("unexpected simple scalar subquery row: %v", row)
	}
}

func TestCosmosSQLPredicateKeywordAndStringPredicates(t *testing.T) {
	where := `NOT (c.category IN ('food', 'archived')) AND c.price BETWEEN 10 AND 30 AND (CONTAINS(c.name, 'pro', true) OR STARTSWITH(c.name, 'Trail')) AND ENDSWITH(c.sku, '-A')`
	tests := []struct {
		name string
		doc  map[string]any
		want bool
	}{
		{name: "matches contains", doc: map[string]any{"category": "gear", "price": float64(12), "name": "Pro Jacket", "sku": "JK-A"}, want: true},
		{name: "matches startswith", doc: map[string]any{"category": "apparel", "price": float64(25), "name": "Trail Boots", "sku": "BT-A"}, want: true},
		{name: "excludes in category", doc: map[string]any{"category": "food", "price": float64(15), "name": "Pro Snacks", "sku": "SN-A"}},
		{name: "excludes between range", doc: map[string]any{"category": "gear", "price": float64(45), "name": "Trail Rope", "sku": "RP-A"}},
		{name: "excludes string predicates", doc: map[string]any{"category": "gear", "price": float64(20), "name": "Basic Hat", "sku": "HT-B"}},
		{name: "between predicate alone", doc: map[string]any{"price": float64(45)}, want: false},
		{name: "string predicates alone", doc: map[string]any{"name": "Basic Hat", "sku": "HT-B"}, want: false},
		{name: "post category chain excludes range", doc: map[string]any{"price": float64(45), "name": "Trail Rope", "sku": "RP-A"}, want: false},
		{name: "post category chain excludes strings", doc: map[string]any{"price": float64(20), "name": "Basic Hat", "sku": "HT-B"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			predicate := where
			if tt.name == "between predicate alone" {
				predicate = `c.price BETWEEN 10 AND 30`
			}
			if tt.name == "string predicates alone" {
				predicate = `(CONTAINS(c.name, 'pro', true) OR STARTSWITH(c.name, 'Trail')) AND ENDSWITH(c.sku, '-A')`
			}
			if strings.HasPrefix(tt.name, "post category chain") {
				predicate = `c.price BETWEEN 10 AND 30 AND (CONTAINS(c.name, 'pro', true) OR STARTSWITH(c.name, 'Trail')) AND ENDSWITH(c.sku, '-A')`
			}
			if got := cosmosSQLMatches(tt.doc, predicate, nil); got != tt.want {
				t.Fatalf("cosmosSQLMatches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCosmosSQLDataPlanePatchNestedPathsAndMove(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"id":"nested","category":"sort","inventory":{"quantity":15,"color":"red"},"profile":{"source":"catalog"},"retired":false}`), nil))

	patch, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPatch, "https://acct.documents.azure.com/dbs/app/colls/items/docs/nested", []byte(`{"operations":[
		{"op":"incr","path":"/inventory/quantity","value":10},
		{"op":"set","path":"/inventory/color","value":"silver"},
		{"op":"move","from":"/profile/source","path":"/inventory/source"},
		{"op":"remove","path":"/retired"}
	]}`), map[string]string{
		"x-ms-documentdb-partitionkey": `["sort"]`,
	}))
	if err != nil {
		t.Fatalf("nested patch document returned error: %v", err)
	}
	if patch.StatusCode != http.StatusOK {
		t.Fatalf("expected nested patch status 200, got %d; body=%s", patch.StatusCode, string(patch.RawBody))
	}
	patched := decodeCosmosResponse(t, patch)
	inventory := patched["inventory"].(map[string]any)
	if inventory["quantity"] != float64(25) || inventory["color"] != "silver" || inventory["source"] != "catalog" {
		t.Fatalf("unexpected nested inventory patch: %v", patched)
	}
	profile := patched["profile"].(map[string]any)
	if _, exists := profile["source"]; exists {
		t.Fatalf("expected source to move out of profile: %v", patched)
	}
	if _, exists := patched["retired"]; exists {
		t.Fatalf("expected retired to be removed: %v", patched)
	}
}

func TestCosmosSQLDataPlanePatchMissingStrictPathFailsAtomically(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"id":"strict","category":"sort","name":"Original","inventory":{"quantity":15}}`), nil))

	patch, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPatch, "https://acct.documents.azure.com/dbs/app/colls/items/docs/strict", []byte(`{"operations":[
		{"op":"set","path":"/name","value":"Changed"},
		{"op":"remove","path":"/missing"}
	]}`), map[string]string{
		"x-ms-documentdb-partitionkey": `["sort"]`,
	}))
	if err != nil {
		t.Fatalf("strict patch document returned error: %v", err)
	}
	if patch.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected missing strict path to return 400, got %d; body=%s", patch.StatusCode, string(patch.RawBody))
	}

	readResp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodGet, "https://acct.documents.azure.com/dbs/app/colls/items/docs/strict", nil, map[string]string{
		"x-ms-documentdb-partitionkey": `["sort"]`,
	}))
	if err != nil {
		t.Fatalf("read strict document after failed patch returned error: %v", err)
	}
	doc := decodeCosmosResponse(t, readResp)
	if doc["name"] != "Original" {
		t.Fatalf("expected failed patch to leave original name, got %v", doc)
	}
	if _, exists := doc["missing"]; exists {
		t.Fatalf("expected failed patch not to create missing field, got %v", doc)
	}
}

func TestCosmosSQLDataPlanePatchArrayPaths(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"id":"array-patch","category":"sort","tags":["r-series"],"sizes":["S","M","L"]}`), nil))

	patch, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPatch, "https://acct.documents.azure.com/dbs/app/colls/items/docs/array-patch", []byte(`{"operations":[
		{"op":"add","path":"/tags/-","value":"featured"},
		{"op":"add","path":"/sizes/1","value":"XS"},
		{"op":"set","path":"/sizes/2","value":"Medium"},
		{"op":"replace","path":"/tags/0","value":"road"},
		{"op":"remove","path":"/sizes/3"}
	]}`), map[string]string{
		"x-ms-documentdb-partitionkey": `["sort"]`,
	}))
	if err != nil {
		t.Fatalf("array patch document returned error: %v", err)
	}
	if patch.StatusCode != http.StatusOK {
		t.Fatalf("expected array patch status 200, got %d; body=%s", patch.StatusCode, string(patch.RawBody))
	}
	patched := decodeCosmosResponse(t, patch)
	tags := patched["tags"].([]any)
	if len(tags) != 2 || tags[0] != "road" || tags[1] != "featured" {
		t.Fatalf("unexpected patched tags: %v", patched)
	}
	sizes := patched["sizes"].([]any)
	if len(sizes) != 3 || sizes[0] != "S" || sizes[1] != "XS" || sizes[2] != "Medium" {
		t.Fatalf("unexpected patched sizes: %v", patched)
	}
}

func TestCosmosSQLDataPlanePatchArrayOutOfRangeFailsAtomically(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"id":"array-strict","category":"sort","name":"Original","tags":["r-series"]}`), nil))

	patch, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPatch, "https://acct.documents.azure.com/dbs/app/colls/items/docs/array-strict", []byte(`{"operations":[
		{"op":"set","path":"/name","value":"Changed"},
		{"op":"add","path":"/tags/5","value":"invalid"}
	]}`), map[string]string{
		"x-ms-documentdb-partitionkey": `["sort"]`,
	}))
	if err != nil {
		t.Fatalf("array strict patch document returned error: %v", err)
	}
	if patch.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected out-of-range array patch to return 400, got %d; body=%s", patch.StatusCode, string(patch.RawBody))
	}

	readResp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodGet, "https://acct.documents.azure.com/dbs/app/colls/items/docs/array-strict", nil, map[string]string{
		"x-ms-documentdb-partitionkey": `["sort"]`,
	}))
	if err != nil {
		t.Fatalf("read array strict document after failed patch returned error: %v", err)
	}
	doc := decodeCosmosResponse(t, readResp)
	if doc["name"] != "Original" {
		t.Fatalf("expected failed array patch to leave original name, got %v", doc)
	}
	tags := doc["tags"].([]any)
	if len(tags) != 1 || tags[0] != "r-series" {
		t.Fatalf("expected failed array patch to leave tags unchanged, got %v", doc)
	}
}

func TestCosmosSQLDataPlanePatchConditionPredicate(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"id":"conditional","category":"sort","name":"Original","price":10}`), nil))

	matchPatch, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPatch, "https://acct.documents.azure.com/dbs/app/colls/items/docs/conditional", []byte(`{
		"condition":"from c where c.category = 'sort'",
		"operations":[{"op":"set","path":"/name","value":"Matched"}]
	}`), map[string]string{
		"x-ms-documentdb-partitionkey": `["sort"]`,
	}))
	if err != nil {
		t.Fatalf("matching conditional patch returned error: %v", err)
	}
	if matchPatch.StatusCode != http.StatusOK {
		t.Fatalf("expected matching conditional patch status 200, got %d; body=%s", matchPatch.StatusCode, string(matchPatch.RawBody))
	}
	matched := decodeCosmosResponse(t, matchPatch)
	if matched["name"] != "Matched" {
		t.Fatalf("expected matching conditional patch to update name, got %v", matched)
	}

	failedPatch, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPatch, "https://acct.documents.azure.com/dbs/app/colls/items/docs/conditional", []byte(`{
		"condition":"from c where c.category = 'missing'",
		"operations":[{"op":"set","path":"/name","value":"ShouldNotApply"}]
	}`), map[string]string{
		"x-ms-documentdb-partitionkey": `["sort"]`,
	}))
	if err != nil {
		t.Fatalf("failing conditional patch returned error: %v", err)
	}
	if failedPatch.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("expected failing conditional patch status 412, got %d; body=%s", failedPatch.StatusCode, string(failedPatch.RawBody))
	}

	readResp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodGet, "https://acct.documents.azure.com/dbs/app/colls/items/docs/conditional", nil, map[string]string{
		"x-ms-documentdb-partitionkey": `["sort"]`,
	}))
	if err != nil {
		t.Fatalf("read conditional document after failed patch returned error: %v", err)
	}
	doc := decodeCosmosResponse(t, readResp)
	if doc["name"] != "Matched" {
		t.Fatalf("expected failing conditional patch to leave document unchanged, got %v", doc)
	}
}

func TestCosmosSQLDataPlanePatchRejectsMoveIntoDescendant(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"id":"move-descendant","category":"sort","inventory":{"quantity":15,"color":"red"}}`), nil))

	patch, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPatch, "https://acct.documents.azure.com/dbs/app/colls/items/docs/move-descendant", []byte(`{"operations":[
		{"op":"move","from":"/inventory","path":"/inventory/color"}
	]}`), map[string]string{
		"x-ms-documentdb-partitionkey": `["sort"]`,
	}))
	if err != nil {
		t.Fatalf("move descendant patch returned error: %v", err)
	}
	if patch.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected move descendant patch status 400, got %d; body=%s", patch.StatusCode, string(patch.RawBody))
	}

	readResp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodGet, "https://acct.documents.azure.com/dbs/app/colls/items/docs/move-descendant", nil, map[string]string{
		"x-ms-documentdb-partitionkey": `["sort"]`,
	}))
	if err != nil {
		t.Fatalf("read move descendant document after failed patch returned error: %v", err)
	}
	doc := decodeCosmosResponse(t, readResp)
	inventory := doc["inventory"].(map[string]any)
	if inventory["quantity"] != float64(15) || inventory["color"] != "red" {
		t.Fatalf("expected failed move descendant patch to leave inventory unchanged, got %v", doc)
	}
}

func TestCosmosSQLDataPlanePartitionKeyRangesAndQueryPlan(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))

	rangesResp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodGet, "https://acct.documents.azure.com/dbs/app/colls/items/pkranges", nil, nil))
	if err != nil {
		t.Fatalf("list partition key ranges returned error: %v", err)
	}
	if rangesResp.StatusCode != http.StatusOK {
		t.Fatalf("expected partition key ranges status 200, got %d; body=%s", rangesResp.StatusCode, string(rangesResp.RawBody))
	}
	rangesBody := decodeCosmosResponse(t, rangesResp)
	if rangesBody["_count"] != float64(1) || rangesResp.Headers["x-ms-item-count"] != "1" {
		t.Fatalf("unexpected partition key ranges response: body=%v headers=%v", rangesBody, rangesResp.Headers)
	}
	ranges := rangesBody["PartitionKeyRanges"].([]any)
	firstRange := ranges[0].(map[string]any)
	if firstRange["id"] != "0" || firstRange["minInclusive"] != "" || firstRange["maxExclusive"] != "FF" || firstRange["_self"] != "dbs/app/colls/items/pkranges/0" {
		t.Fatalf("unexpected partition key range: %v", firstRange)
	}

	planResp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"query":"SELECT VALUE COUNT(1) FROM c"}`), map[string]string{
		"x-ms-cosmos-is-query-plan-request": "True",
	}))
	if err != nil {
		t.Fatalf("query plan returned error: %v", err)
	}
	if planResp.StatusCode != http.StatusOK {
		t.Fatalf("expected query plan status 200, got %d; body=%s", planResp.StatusCode, string(planResp.RawBody))
	}
	plan := decodeCosmosResponse(t, planResp)
	if plan["partitionedQueryExecutionInfoVersion"] != float64(1) {
		t.Fatalf("unexpected query plan version: %v", plan)
	}
	queryInfo := plan["queryInfo"].(map[string]any)
	if queryInfo["hasSelectValue"] != true || queryInfo["distinctType"] != "None" {
		t.Fatalf("unexpected query plan info: %v", queryInfo)
	}
	queryRanges := plan["queryRanges"].([]any)
	firstQueryRange := queryRanges[0].(map[string]any)
	if firstQueryRange["min"] != "" || firstQueryRange["max"] != "FF" || firstQueryRange["isMinInclusive"] != true || firstQueryRange["isMaxInclusive"] != false {
		t.Fatalf("unexpected query plan ranges: %v", queryRanges)
	}
}

func TestCosmosSQLDataPlaneTransactionalBatchCommitsOperations(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))

	createBatch, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`[
		{"operationType":"Create","resourceBody":{"id":"b1","category":"test","v":1}},
		{"operationType":"Create","resourceBody":{"id":"b2","category":"test","v":2}},
		{"operationType":"Upsert","resourceBody":{"id":"b3","category":"test","v":3}}
	]`), map[string]string{
		"x-ms-cosmos-is-batch-request": "True",
		"x-ms-documentdb-partitionkey": `["test"]`,
	}))
	if err != nil {
		t.Fatalf("create batch returned error: %v", err)
	}
	if createBatch.StatusCode != http.StatusOK {
		t.Fatalf("expected create batch status 200, got %d; body=%s", createBatch.StatusCode, string(createBatch.RawBody))
	}
	createResults := decodeCosmosArrayResponse(t, createBatch)
	if len(createResults) != 3 || createResults[0]["statusCode"] != float64(201) || createResults[1]["statusCode"] != float64(201) || createResults[2]["statusCode"] != float64(201) {
		t.Fatalf("unexpected create batch results: %v", createResults)
	}
	if createBatch.Headers["x-ms-request-charge"] != "3" {
		t.Fatalf("expected batch request charge 3, got headers=%v", createBatch.Headers)
	}

	updateBatch, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`[
		{"operationType":"Read","id":"b1"},
		{"operationType":"Replace","id":"b2","resourceBody":{"id":"b2","category":"test","v":99}},
		{"operationType":"Delete","id":"b3"}
	]`), map[string]string{
		"x-ms-cosmos-is-batch-request": "True",
		"x-ms-documentdb-partitionkey": `["test"]`,
	}))
	if err != nil {
		t.Fatalf("update batch returned error: %v", err)
	}
	if updateBatch.StatusCode != http.StatusOK {
		t.Fatalf("expected update batch status 200, got %d; body=%s", updateBatch.StatusCode, string(updateBatch.RawBody))
	}
	updateResults := decodeCosmosArrayResponse(t, updateBatch)
	if len(updateResults) != 3 || updateResults[0]["statusCode"] != float64(200) || updateResults[1]["statusCode"] != float64(200) || updateResults[2]["statusCode"] != float64(204) {
		t.Fatalf("unexpected update batch results: %v", updateResults)
	}
	readResult := updateResults[0]["resourceBody"].(map[string]any)
	if readResult["id"] != "b1" || readResult["v"] != float64(1) {
		t.Fatalf("unexpected read batch resource body: %v", readResult)
	}

	replaced, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodGet, "https://acct.documents.azure.com/dbs/app/colls/items/docs/b2", nil, map[string]string{
		"x-ms-documentdb-partitionkey": `["test"]`,
	}))
	if err != nil {
		t.Fatalf("read replaced document returned error: %v", err)
	}
	replacedBody := decodeCosmosResponse(t, replaced)
	if replacedBody["v"] != float64(99) {
		t.Fatalf("expected replaced document value 99, got %v", replacedBody)
	}

	deleted, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodGet, "https://acct.documents.azure.com/dbs/app/colls/items/docs/b3", nil, map[string]string{
		"x-ms-documentdb-partitionkey": `["test"]`,
	}))
	if err != nil {
		t.Fatalf("read deleted document returned error: %v", err)
	}
	if deleted.StatusCode != http.StatusNotFound {
		t.Fatalf("expected deleted batch document to be missing, got %d; body=%s", deleted.StatusCode, string(deleted.RawBody))
	}
}

func TestCosmosSQLDataPlaneTransactionalBatchRollsBackOnFailure(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"id":"dup","category":"test","v":1}`), nil))

	failedBatch, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`[
		{"operationType":"Create","resourceBody":{"id":"before-failure","category":"test","v":1}},
		{"operationType":"Create","resourceBody":{"id":"dup","category":"test","v":2}},
		{"operationType":"Create","resourceBody":{"id":"after-failure","category":"test","v":3}}
	]`), map[string]string{
		"x-ms-cosmos-is-batch-request": "True",
		"x-ms-documentdb-partitionkey": `["test"]`,
	}))
	if err != nil {
		t.Fatalf("failed batch returned error: %v", err)
	}
	if failedBatch.StatusCode != http.StatusOK {
		t.Fatalf("expected failed batch HTTP status 200, got %d; body=%s", failedBatch.StatusCode, string(failedBatch.RawBody))
	}
	results := decodeCosmosArrayResponse(t, failedBatch)
	if len(results) != 3 || results[0]["statusCode"] != float64(424) || results[1]["statusCode"] != float64(409) || results[2]["statusCode"] != float64(424) {
		t.Fatalf("unexpected failed batch results: %v", results)
	}

	for _, docID := range []string{"before-failure", "after-failure"} {
		readResp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodGet, "https://acct.documents.azure.com/dbs/app/colls/items/docs/"+docID, nil, map[string]string{
			"x-ms-documentdb-partitionkey": `["test"]`,
		}))
		if err != nil {
			t.Fatalf("read %s after failed batch returned error: %v", docID, err)
		}
		if readResp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected %s to be rolled back, got %d; body=%s", docID, readResp.StatusCode, string(readResp.RawBody))
		}
	}

	dupResp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodGet, "https://acct.documents.azure.com/dbs/app/colls/items/docs/dup", nil, map[string]string{
		"x-ms-documentdb-partitionkey": `["test"]`,
	}))
	if err != nil {
		t.Fatalf("read duplicate document after failed batch returned error: %v", err)
	}
	dup := decodeCosmosResponse(t, dupResp)
	if dup["v"] != float64(1) {
		t.Fatalf("expected duplicate document to keep original value, got %v", dup)
	}
}

func TestCosmosSQLDataPlaneTransactionalBatchPatchesDocument(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"id":"patch-batch","category":"test","name":"Original","counter":10,"removable":true}`), nil))

	batchResp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`[
		{"operationType":"Patch","id":"patch-batch","resourceBody":{"operations":[
			{"op":"set","path":"/name","value":"Patched"},
			{"op":"incr","path":"/counter","value":5},
			{"op":"remove","path":"/removable"}
		]}},
		{"operationType":"Read","id":"patch-batch"}
	]`), map[string]string{
		"x-ms-cosmos-is-batch-request": "True",
		"x-ms-documentdb-partitionkey": `["test"]`,
	}))
	if err != nil {
		t.Fatalf("patch batch returned error: %v", err)
	}
	if batchResp.StatusCode != http.StatusOK {
		t.Fatalf("expected patch batch HTTP status 200, got %d; body=%s", batchResp.StatusCode, string(batchResp.RawBody))
	}
	results := decodeCosmosArrayResponse(t, batchResp)
	if len(results) != 2 || results[0]["statusCode"] != float64(200) || results[1]["statusCode"] != float64(200) {
		t.Fatalf("unexpected patch batch results: %v", results)
	}
	readResult := results[1]["resourceBody"].(map[string]any)
	if readResult["name"] != "Patched" || readResult["counter"] != float64(15) {
		t.Fatalf("unexpected patched document in batch read: %v", readResult)
	}
	if _, ok := readResult["removable"]; ok {
		t.Fatalf("expected removable to be removed in batch read: %v", readResult)
	}

	patched, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodGet, "https://acct.documents.azure.com/dbs/app/colls/items/docs/patch-batch", nil, map[string]string{
		"x-ms-documentdb-partitionkey": `["test"]`,
	}))
	if err != nil {
		t.Fatalf("read patched document after batch returned error: %v", err)
	}
	patchedBody := decodeCosmosResponse(t, patched)
	if patchedBody["name"] != "Patched" || patchedBody["counter"] != float64(15) {
		t.Fatalf("expected patch batch updates to persist, got %v", patchedBody)
	}
	if _, ok := patchedBody["removable"]; ok {
		t.Fatalf("expected removable to stay removed after batch, got %v", patchedBody)
	}
}

func TestCosmosSQLDataPlaneTransactionalBatchRollsBackFailedPatch(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"id":"batch-strict","category":"test","inventory":{"quantity":15,"color":"red"}}`), nil))

	batchResp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`[
		{"operationType":"Patch","id":"batch-strict","resourceBody":{"operations":[
			{"op":"set","path":"/inventory/color","value":"silver"},
			{"op":"remove","path":"/inventory/missing"}
		]}},
		{"operationType":"Read","id":"batch-strict"}
	]`), map[string]string{
		"x-ms-cosmos-is-batch-request": "True",
		"x-ms-documentdb-partitionkey": `["test"]`,
	}))
	if err != nil {
		t.Fatalf("failed patch batch returned error: %v", err)
	}
	if batchResp.StatusCode != http.StatusOK {
		t.Fatalf("expected failed patch batch HTTP status 200, got %d; body=%s", batchResp.StatusCode, string(batchResp.RawBody))
	}
	results := decodeCosmosArrayResponse(t, batchResp)
	if len(results) != 2 || results[0]["statusCode"] != float64(400) || results[1]["statusCode"] != float64(424) {
		t.Fatalf("unexpected failed patch batch results: %v", results)
	}

	readResp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodGet, "https://acct.documents.azure.com/dbs/app/colls/items/docs/batch-strict", nil, map[string]string{
		"x-ms-documentdb-partitionkey": `["test"]`,
	}))
	if err != nil {
		t.Fatalf("read batch strict document after failed patch returned error: %v", err)
	}
	doc := decodeCosmosResponse(t, readResp)
	inventory := doc["inventory"].(map[string]any)
	if inventory["color"] != "red" || inventory["quantity"] != float64(15) {
		t.Fatalf("expected failed patch batch to roll back nested document, got %v", doc)
	}
}

func TestCosmosSQLDataPlaneTransactionalBatchRollsBackFailedConditionalPatch(t *testing.T) {
	svc := New()
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs", []byte(`{"id":"app"}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls", []byte(`{"id":"items","partitionKey":{"paths":["/category"],"kind":"Hash"}}`), nil))
	_, _ = svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`{"id":"batch-condition","category":"test","name":"Original","counter":1}`), nil))

	batchResp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodPost, "https://acct.documents.azure.com/dbs/app/colls/items/docs", []byte(`[
		{"operationType":"Patch","id":"batch-condition","resourceBody":{"operations":[
			{"op":"set","path":"/counter","value":2}
		]}},
		{"operationType":"Patch","id":"batch-condition","resourceBody":{
			"condition":"from c where c.category = 'missing'",
			"operations":[{"op":"set","path":"/name","value":"ShouldNotApply"}]
		}},
		{"operationType":"Read","id":"batch-condition"}
	]`), map[string]string{
		"x-ms-cosmos-is-batch-request": "True",
		"x-ms-documentdb-partitionkey": `["test"]`,
	}))
	if err != nil {
		t.Fatalf("failed conditional patch batch returned error: %v", err)
	}
	if batchResp.StatusCode != http.StatusOK {
		t.Fatalf("expected failed conditional patch batch HTTP status 200, got %d; body=%s", batchResp.StatusCode, string(batchResp.RawBody))
	}
	results := decodeCosmosArrayResponse(t, batchResp)
	if len(results) != 3 || results[0]["statusCode"] != float64(424) || results[1]["statusCode"] != float64(412) || results[2]["statusCode"] != float64(424) {
		t.Fatalf("unexpected failed conditional patch batch results: %v", results)
	}

	readResp, err := svc.HandleRequest(cosmosDataCtx(t, http.MethodGet, "https://acct.documents.azure.com/dbs/app/colls/items/docs/batch-condition", nil, map[string]string{
		"x-ms-documentdb-partitionkey": `["test"]`,
	}))
	if err != nil {
		t.Fatalf("read batch condition document after failed patch returned error: %v", err)
	}
	doc := decodeCosmosResponse(t, readResp)
	if doc["name"] != "Original" || doc["counter"] != float64(1) {
		t.Fatalf("expected failed conditional patch batch to roll back document, got %v", doc)
	}
}

func TestCosmosServiceKeysIncludeSQLDataPlane(t *testing.T) {
	svc := New()

	seen := make(map[string]bool)
	for _, key := range svc.ServiceKeys() {
		seen[string(key.Provider)+"|"+key.Service+"|"+key.APIVersion] = true
	}

	if !seen["azure|Microsoft.DocumentDB/sqlApi|2018-12-31"] {
		t.Fatalf("expected Cosmos DB SQL API data-plane service key")
	}
}

func cosmosCtx(t *testing.T, method, rawURL string, body []byte) *service.RequestContext {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	req := &http.Request{Method: method, URL: u, Host: u.Host}
	req.Header = http.Header{"Authorization": []string{"Bearer azure-token"}}
	return &service.RequestContext{
		Region:     "eastus",
		AccountID:  "sub-1",
		Action:     method,
		RawRequest: req,
		Body:       body,
	}
}

func cosmosDataCtx(t *testing.T, method, rawURL string, body []byte, headers map[string]string) *service.RequestContext {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	req := &http.Request{Method: method, URL: u, Host: u.Host}
	req.Header = http.Header{
		"Authorization": []string{"type=master&ver=1.0&sig=fake"},
		"x-ms-version":  []string{"2018-12-31"},
		"Accept":        []string{"application/json"},
		"Content-Type":  []string{"application/json"},
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	if strings.Contains(rawURL, "-cosmos/") {
		req.Host = u.Host
	}
	return &service.RequestContext{
		Region:     "eastus",
		AccountID:  "sub-1",
		Action:     method,
		RawRequest: req,
		Body:       body,
	}
}

func decodeCosmosResponse(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	var out map[string]any
	if err := gojson.Unmarshal(resp.RawBody, &out); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, string(resp.RawBody))
	}
	return out
}

func decodeCosmosArrayResponse(t *testing.T, resp *service.Response) []map[string]any {
	t.Helper()
	var out []map[string]any
	if err := gojson.Unmarshal(resp.RawBody, &out); err != nil {
		t.Fatalf("decode array response: %v; body=%s", err, string(resp.RawBody))
	}
	return out
}
