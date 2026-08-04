package cosmosdb

import (
	"encoding/base64"
	"fmt"
	"math"
	"math/rand/v2"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/azurearm"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

const cosmosSQLDataPlaneAPIVersion = "2018-12-31"

type cosmosUndefinedType struct{}

var cosmosUndefined = cosmosUndefinedType{}

const cosmosStaticNowField = "__cloudmock_cosmos_static_now"

func cosmosSQLDataPlaneAccount(req *http.Request) string {
	host := strings.ToLower(req.Host)
	if host == "" {
		host = strings.ToLower(req.Header.Get("Host"))
	}
	if colon := strings.IndexByte(host, ':'); colon >= 0 {
		host = host[:colon]
	}
	for _, suffix := range []string{".documents.azure.com", ".documents.azure.us", ".documents.azure.cn"} {
		if strings.HasSuffix(host, suffix) && strings.TrimSuffix(host, suffix) != "" {
			return strings.TrimSuffix(host, suffix)
		}
	}
	parts := splitPath(req.URL.EscapedPath())
	if len(parts) > 0 && strings.HasSuffix(strings.ToLower(parts[0]), "-cosmos") {
		return strings.TrimSuffix(parts[0], "-cosmos")
	}
	return ""
}

func (s *CosmosDBService) handleSQLDataPlane(ctx *service.RequestContext) (*service.Response, error) {
	account := cosmosSQLDataPlaneAccount(ctx.RawRequest)
	parts := splitPath(ctx.RawRequest.URL.EscapedPath())
	if len(parts) > 0 && strings.HasSuffix(strings.ToLower(parts[0]), "-cosmos") {
		parts = parts[1:]
	}

	if account == "" {
		return cosmosSQLError(http.StatusBadRequest, "BadRequest", "Cosmos DB account could not be resolved.")
	}
	if len(parts) == 0 {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.cosmosAccountInfo(ctx.RawRequest, account)
		}
		return cosmosSQLError(http.StatusNotImplemented, "NotImplemented", "The Cosmos DB SQL API route is not implemented.")
	}
	if !strings.EqualFold(parts[0], "dbs") {
		return cosmosSQLError(http.StatusNotImplemented, "NotImplemented", "The Cosmos DB SQL API route is not implemented.")
	}
	if len(parts) == 1 {
		switch ctx.RawRequest.Method {
		case http.MethodGet:
			return s.listCosmosDatabases(account)
		case http.MethodPost:
			return s.createCosmosDatabase(account, ctx.Body)
		default:
			return cosmosSQLError(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
	}

	dbID := parts[1]
	if len(parts) == 2 {
		switch ctx.RawRequest.Method {
		case http.MethodGet:
			return s.getCosmosDatabase(account, dbID)
		case http.MethodDelete:
			return s.deleteCosmosDatabase(account, dbID)
		default:
			return cosmosSQLError(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
	}
	if len(parts) >= 3 && strings.EqualFold(parts[2], "colls") {
		if len(parts) == 3 {
			switch ctx.RawRequest.Method {
			case http.MethodGet:
				return s.listCosmosCollections(account, dbID)
			case http.MethodPost:
				return s.createCosmosCollection(account, dbID, ctx.Body)
			default:
				return cosmosSQLError(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
			}
		}
		collID := parts[3]
		if len(parts) == 4 {
			switch ctx.RawRequest.Method {
			case http.MethodGet:
				return s.getCosmosCollection(account, dbID, collID)
			case http.MethodDelete:
				return s.deleteCosmosCollection(account, dbID, collID)
			default:
				return cosmosSQLError(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
			}
		}
		if len(parts) == 5 && strings.EqualFold(parts[4], "pkranges") {
			if ctx.RawRequest.Method == http.MethodGet {
				return s.listCosmosPartitionKeyRanges(account, dbID, collID)
			}
			return cosmosSQLError(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
		if len(parts) >= 5 && strings.EqualFold(parts[4], "docs") {
			if len(parts) == 5 {
				switch ctx.RawRequest.Method {
				case http.MethodGet:
					return s.listCosmosDocuments(account, dbID, collID)
				case http.MethodPost:
					if isCosmosBatchRequest(ctx.RawRequest) {
						return s.executeCosmosTransactionalBatch(account, dbID, collID, ctx.Body, ctx.RawRequest.Header)
					}
					if isCosmosQueryPlanRequest(ctx.RawRequest) {
						return s.cosmosQueryPlan(account, dbID, collID, ctx.Body)
					}
					if isCosmosQueryRequest(ctx.RawRequest) {
						return s.queryCosmosDocuments(account, dbID, collID, ctx.Body, ctx.RawRequest.Header)
					}
					return s.createCosmosDocument(account, dbID, collID, ctx.Body, ctx.RawRequest.Header)
				default:
					return cosmosSQLError(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
				}
			}
			if len(parts) == 6 {
				docID := parts[5]
				switch ctx.RawRequest.Method {
				case http.MethodGet:
					return s.getCosmosDocument(account, dbID, collID, docID, ctx.RawRequest.Header)
				case http.MethodPut:
					return s.replaceCosmosDocument(account, dbID, collID, docID, ctx.Body, ctx.RawRequest.Header)
				case http.MethodPatch:
					return s.patchCosmosDocument(account, dbID, collID, docID, ctx.Body, ctx.RawRequest.Header)
				case http.MethodDelete:
					return s.deleteCosmosDocument(account, dbID, collID, docID, ctx.RawRequest.Header)
				default:
					return cosmosSQLError(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
				}
			}
		}
	}
	return cosmosSQLError(http.StatusNotImplemented, "NotImplemented", "The Cosmos DB SQL API route is not implemented.")
}

func (s *CosmosDBService) cosmosAccountInfo(req *http.Request, account string) (*service.Response, error) {
	host := req.Host
	if host == "" {
		host = req.Header.Get("Host")
	}
	scheme := "https"
	if req.URL.Scheme == "http" {
		scheme = "http"
	}
	endpoint := scheme + "://" + host + "/"
	body := map[string]any{
		"id":                           account,
		"_rid":                         "",
		"_self":                        "",
		"_ts":                          time.Now().UTC().Unix(),
		"databasesLink":                "dbs/",
		"mediaLink":                    "media/",
		"kind":                         "GlobalDocumentDB",
		"storageQuotaInMB":             10240,
		"currentMediaStorageUsageInMB": 0,
		"consistencyPolicy": map[string]any{
			"defaultConsistencyLevel": "Session",
			"maxStalenessPrefix":      100,
			"maxIntervalInSeconds":    5,
		},
		"writableLocations": []any{map[string]any{"name": "South Central US", "databaseAccountEndpoint": endpoint}},
		"readableLocations": []any{map[string]any{"name": "South Central US", "databaseAccountEndpoint": endpoint}},
	}
	return cosmosSQLJSONResponse(http.StatusOK, body, "")
}

func (s *CosmosDBService) createCosmosDatabase(account string, body []byte) (*service.Response, error) {
	var input map[string]any
	if err := gojson.Unmarshal(body, &input); err != nil {
		return cosmosSQLError(http.StatusBadRequest, "BadRequest", "The request body is invalid JSON.")
	}
	id := stringValue(input["id"])
	if id == "" {
		return cosmosSQLError(http.StatusBadRequest, "BadRequest", "'id' is required.")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	key := cosmosDataDatabaseKey(account, id)
	if _, exists := s.dataDatabases[key]; exists {
		return cosmosSQLError(http.StatusConflict, "Conflict", "Database '"+id+"' already exists.")
	}
	db := cosmosDataDatabase{ID: id, RID: s.nextCosmosTokenLocked("db"), ETag: s.nextCosmosTokenLocked("etag"), TS: time.Now().UTC().Unix()}
	s.dataDatabases[key] = db
	return cosmosSQLJSONResponse(http.StatusCreated, cosmosDatabaseBody(db), db.ETag)
}

func (s *CosmosDBService) getCosmosDatabase(account, id string) (*service.Response, error) {
	s.mu.RLock()
	db, ok := s.dataDatabases[cosmosDataDatabaseKey(account, id)]
	s.mu.RUnlock()
	if !ok {
		return cosmosSQLNotFound(id)
	}
	return cosmosSQLJSONResponse(http.StatusOK, cosmosDatabaseBody(db), db.ETag)
}

func (s *CosmosDBService) listCosmosDatabases(account string) (*service.Response, error) {
	prefix := strings.ToLower(account) + "|db|"
	s.mu.RLock()
	values := make([]map[string]any, 0)
	for key, db := range s.dataDatabases {
		if strings.HasPrefix(key, prefix) {
			values = append(values, cosmosDatabaseBody(db))
		}
	}
	s.mu.RUnlock()
	sort.Slice(values, func(i, j int) bool { return fmt.Sprint(values[i]["id"]) < fmt.Sprint(values[j]["id"]) })
	return cosmosSQLJSONResponse(http.StatusOK, map[string]any{"_rid": "", "_count": len(values), "Databases": values}, "")
}

func (s *CosmosDBService) deleteCosmosDatabase(account, id string) (*service.Response, error) {
	key := cosmosDataDatabaseKey(account, id)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.dataDatabases[key]; !ok {
		return cosmosSQLNotFound(id)
	}
	delete(s.dataDatabases, key)
	collectionPrefix := strings.ToLower(account) + "|coll|" + strings.ToLower(id) + "|"
	for collKey := range s.dataCollections {
		if strings.HasPrefix(collKey, collectionPrefix) {
			delete(s.dataCollections, collKey)
		}
	}
	documentPrefix := strings.ToLower(account) + "|doc|" + strings.ToLower(id) + "|"
	for docKey := range s.dataDocuments {
		if strings.HasPrefix(docKey, documentPrefix) {
			delete(s.dataDocuments, docKey)
		}
	}
	return emptyCosmosSQLResponse(http.StatusNoContent)
}

func (s *CosmosDBService) createCosmosCollection(account, dbID string, body []byte) (*service.Response, error) {
	var input map[string]any
	if err := gojson.Unmarshal(body, &input); err != nil {
		return cosmosSQLError(http.StatusBadRequest, "BadRequest", "The request body is invalid JSON.")
	}
	id := stringValue(input["id"])
	if id == "" {
		return cosmosSQLError(http.StatusBadRequest, "BadRequest", "'id' is required.")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.dataDatabases[cosmosDataDatabaseKey(account, dbID)]; !ok {
		return cosmosSQLNotFound(dbID)
	}
	key := cosmosDataCollectionKey(account, dbID, id)
	if _, exists := s.dataCollections[key]; exists {
		return cosmosSQLError(http.StatusConflict, "Conflict", "Container '"+id+"' already exists.")
	}
	partitionKey := mapFromAny(input["partitionKey"])
	if partitionKey == nil {
		partitionKey = map[string]any{"paths": []any{"/id"}, "kind": "Hash"}
	}
	partitionKey["version"] = float64(2)
	indexingPolicy := mapFromAny(input["indexingPolicy"])
	if indexingPolicy == nil {
		indexingPolicy = defaultCosmosIndexingPolicy()
	}
	coll := cosmosDataCollection{ID: id, RID: s.nextCosmosTokenLocked("coll"), ETag: s.nextCosmosTokenLocked("etag"), TS: time.Now().UTC().Unix(), PartitionKey: partitionKey, IndexingPolicy: indexingPolicy}
	s.dataCollections[key] = coll
	return cosmosSQLJSONResponseWithHeaders(http.StatusCreated, cosmosCollectionBody(dbID, coll), coll.ETag, map[string]string{"x-ms-alt-content-path": "dbs/" + dbID})
}

func (s *CosmosDBService) getCosmosCollection(account, dbID, collID string) (*service.Response, error) {
	s.mu.RLock()
	coll, ok := s.dataCollections[cosmosDataCollectionKey(account, dbID, collID)]
	s.mu.RUnlock()
	if !ok {
		return cosmosSQLNotFound(collID)
	}
	return cosmosSQLJSONResponse(http.StatusOK, cosmosCollectionBody(dbID, coll), coll.ETag)
}

func (s *CosmosDBService) listCosmosCollections(account, dbID string) (*service.Response, error) {
	s.mu.RLock()
	db, dbOK := s.dataDatabases[cosmosDataDatabaseKey(account, dbID)]
	values := make([]map[string]any, 0)
	prefix := strings.ToLower(account) + "|coll|" + strings.ToLower(dbID) + "|"
	for key, coll := range s.dataCollections {
		if strings.HasPrefix(key, prefix) {
			values = append(values, cosmosCollectionBody(dbID, coll))
		}
	}
	s.mu.RUnlock()
	if !dbOK {
		return cosmosSQLNotFound(dbID)
	}
	sort.Slice(values, func(i, j int) bool { return fmt.Sprint(values[i]["id"]) < fmt.Sprint(values[j]["id"]) })
	return cosmosSQLJSONResponse(http.StatusOK, map[string]any{"_rid": db.RID, "_count": len(values), "DocumentCollections": values}, "")
}

func (s *CosmosDBService) deleteCosmosCollection(account, dbID, collID string) (*service.Response, error) {
	key := cosmosDataCollectionKey(account, dbID, collID)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.dataCollections[key]; !ok {
		return cosmosSQLNotFound(collID)
	}
	delete(s.dataCollections, key)
	docPrefix := strings.ToLower(account) + "|doc|" + strings.ToLower(dbID) + "|" + strings.ToLower(collID) + "|"
	for docKey := range s.dataDocuments {
		if strings.HasPrefix(docKey, docPrefix) {
			delete(s.dataDocuments, docKey)
		}
	}
	return emptyCosmosSQLResponse(http.StatusNoContent)
}

func (s *CosmosDBService) listCosmosPartitionKeyRanges(account, dbID, collID string) (*service.Response, error) {
	s.mu.RLock()
	coll, ok := s.dataCollections[cosmosDataCollectionKey(account, dbID, collID)]
	s.mu.RUnlock()
	if !ok {
		return cosmosSQLNotFound(collID)
	}

	partitionKeyRange := map[string]any{
		"id":           "0",
		"_rid":         coll.RID + "-pkrange-0",
		"_self":        "dbs/" + dbID + "/colls/" + collID + "/pkranges/0",
		"_etag":        quotedCosmosETag(coll.ETag),
		"_ts":          coll.TS,
		"minInclusive": "",
		"maxExclusive": "FF",
		"ridPrefix":    0,
		"_lsn":         1,
		"parents":      []any{},
	}
	body := map[string]any{
		"_rid":               coll.RID,
		"_count":             1,
		"PartitionKeyRanges": []any{partitionKeyRange},
	}
	return cosmosSQLJSONResponseWithHeaders(http.StatusOK, body, "", map[string]string{"x-ms-item-count": "1"})
}

func (s *CosmosDBService) createCosmosDocument(account, dbID, collID string, body []byte, headers http.Header) (*service.Response, error) {
	var input map[string]any
	if err := gojson.Unmarshal(body, &input); err != nil {
		return cosmosSQLError(http.StatusBadRequest, "BadRequest", "The request body is invalid JSON.")
	}
	id := stringValue(input["id"])
	if id == "" {
		id = s.nextCosmosToken("doc")
		input["id"] = id
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	coll, ok := s.dataCollections[cosmosDataCollectionKey(account, dbID, collID)]
	if !ok {
		return cosmosSQLNotFound(collID)
	}
	pk := cosmosPartitionKeyValue(input, coll, headers)
	key := cosmosDataDocumentKey(account, dbID, collID, pk, id)
	existing, exists := s.dataDocuments[key]
	upsert := strings.EqualFold(headers.Get("x-ms-documentdb-is-upsert"), "true")
	if exists && !upsert {
		return cosmosSQLError(http.StatusConflict, "Conflict", "Document with id '"+id+"' already exists.")
	}
	doc := s.newCosmosDocumentLocked(dbID, collID, id, pk, input, existing)
	s.dataDocuments[key] = doc
	status := http.StatusCreated
	if exists && upsert {
		status = http.StatusOK
	}
	return cosmosSQLJSONResponseWithHeaders(status, cosmosDocumentBody(doc), doc.ETag, map[string]string{"x-ms-alt-content-path": "dbs/" + dbID + "/colls/" + collID})
}

func (s *CosmosDBService) getCosmosDocument(account, dbID, collID, docID string, headers http.Header) (*service.Response, error) {
	s.mu.RLock()
	doc, ok := s.findCosmosDocumentLocked(account, dbID, collID, docID, headers)
	s.mu.RUnlock()
	if !ok {
		return cosmosSQLNotFound(docID)
	}
	return cosmosSQLJSONResponse(http.StatusOK, cosmosDocumentBody(doc), doc.ETag)
}

func (s *CosmosDBService) listCosmosDocuments(account, dbID, collID string) (*service.Response, error) {
	s.mu.RLock()
	coll, collOK := s.dataCollections[cosmosDataCollectionKey(account, dbID, collID)]
	values := s.cosmosDocumentsLocked(account, dbID, collID)
	s.mu.RUnlock()
	if !collOK {
		return cosmosSQLNotFound(collID)
	}
	return cosmosSQLJSONResponse(http.StatusOK, map[string]any{"_rid": coll.RID, "_count": len(values), "Documents": values}, "")
}

func (s *CosmosDBService) replaceCosmosDocument(account, dbID, collID, docID string, body []byte, headers http.Header) (*service.Response, error) {
	var input map[string]any
	if err := gojson.Unmarshal(body, &input); err != nil {
		return cosmosSQLError(http.StatusBadRequest, "BadRequest", "The request body is invalid JSON.")
	}
	input["id"] = docID

	s.mu.Lock()
	defer s.mu.Unlock()
	coll, ok := s.dataCollections[cosmosDataCollectionKey(account, dbID, collID)]
	if !ok {
		return cosmosSQLNotFound(collID)
	}
	existing, exists := s.findCosmosDocumentLocked(account, dbID, collID, docID, headers)
	if !exists {
		return cosmosSQLNotFound(docID)
	}
	if !cosmosETagMatches(headers.Get("If-Match"), existing.ETag) {
		return cosmosSQLError(http.StatusPreconditionFailed, "PreconditionFailed", "The access condition specified in the request was not met.")
	}
	pk := cosmosPartitionKeyValue(input, coll, headers)
	oldKey := cosmosDataDocumentKey(account, dbID, collID, existing.PartitionKey, docID)
	newKey := cosmosDataDocumentKey(account, dbID, collID, pk, docID)
	delete(s.dataDocuments, oldKey)
	doc := s.newCosmosDocumentLocked(dbID, collID, docID, pk, input, existing)
	s.dataDocuments[newKey] = doc
	return cosmosSQLJSONResponse(http.StatusOK, cosmosDocumentBody(doc), doc.ETag)
}

func (s *CosmosDBService) patchCosmosDocument(account, dbID, collID, docID string, body []byte, headers http.Header) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, exists := s.findCosmosDocumentLocked(account, dbID, collID, docID, headers)
	if !exists {
		return cosmosSQLNotFound(docID)
	}
	if !cosmosETagMatches(headers.Get("If-Match"), existing.ETag) {
		return cosmosSQLError(http.StatusPreconditionFailed, "PreconditionFailed", "The access condition specified in the request was not met.")
	}
	ops, condition, err := parseCosmosPatchRequest(body)
	if err != nil {
		return cosmosSQLError(http.StatusBadRequest, "BadRequest", "The patch request body is invalid.")
	}
	if !cosmosPatchConditionMatches(existing.Body, condition) {
		return cosmosSQLError(http.StatusPreconditionFailed, "PreconditionFailed", "The specified pre-condition isn't met.")
	}
	docBody := cloneCosmosJSONMap(existing.Body)
	for _, op := range ops {
		if !applyCosmosPatchOperation(docBody, op) {
			return cosmosSQLError(http.StatusBadRequest, "BadRequest", "The patch request body is invalid.")
		}
	}
	doc := s.newCosmosDocumentLocked(dbID, collID, docID, existing.PartitionKey, docBody, existing)
	s.dataDocuments[cosmosDataDocumentKey(account, dbID, collID, existing.PartitionKey, docID)] = doc
	return cosmosSQLJSONResponseWithHeaders(http.StatusOK, cosmosDocumentBody(doc), doc.ETag, map[string]string{"x-ms-alt-content-path": "dbs/" + dbID + "/colls/" + collID})
}

func (s *CosmosDBService) deleteCosmosDocument(account, dbID, collID, docID string, headers http.Header) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	doc, exists := s.findCosmosDocumentLocked(account, dbID, collID, docID, headers)
	if !exists {
		return cosmosSQLNotFound(docID)
	}
	if !cosmosETagMatches(headers.Get("If-Match"), doc.ETag) {
		return cosmosSQLError(http.StatusPreconditionFailed, "PreconditionFailed", "The access condition specified in the request was not met.")
	}
	delete(s.dataDocuments, cosmosDataDocumentKey(account, dbID, collID, doc.PartitionKey, docID))
	return emptyCosmosSQLResponse(http.StatusNoContent)
}

type cosmosBatchOperation struct {
	OperationType string         `json:"operationType"`
	ID            string         `json:"id"`
	ResourceBody  map[string]any `json:"resourceBody"`
}

func (s *CosmosDBService) executeCosmosTransactionalBatch(account, dbID, collID string, body []byte, headers http.Header) (*service.Response, error) {
	var operations []cosmosBatchOperation
	if err := gojson.Unmarshal(body, &operations); err != nil {
		return cosmosSQLError(http.StatusBadRequest, "BadRequest", "Invalid batch request body.")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	coll, ok := s.dataCollections[cosmosDataCollectionKey(account, dbID, collID)]
	if !ok {
		return cosmosSQLNotFound(collID)
	}
	partitionKey, hasPartitionKey := cosmosPartitionKeyFromHeader(headers)
	originalDocuments := cloneCosmosDataDocuments(s.dataDocuments)

	results := make([]map[string]any, 0, len(operations))
	for index, operation := range operations {
		result := s.executeCosmosBatchOperationLocked(account, dbID, collID, coll, partitionKey, hasPartitionKey, operation)
		results = append(results, result)
		if cosmosBatchStatusCode(result) >= http.StatusBadRequest {
			s.dataDocuments = originalDocuments
			for i := range operations {
				switch {
				case i < len(results) && i == index:
					results[i] = result
				case i < len(results):
					results[i] = cosmosBatchResult(http.StatusFailedDependency, "", nil)
				default:
					results = append(results, cosmosBatchResult(http.StatusFailedDependency, "", nil))
				}
			}
			break
		}
	}

	return cosmosSQLJSONResponseWithHeaders(http.StatusOK, results, "", map[string]string{
		"x-ms-request-charge": strconv.Itoa(len(results)),
	})
}

func (s *CosmosDBService) executeCosmosBatchOperationLocked(account, dbID, collID string, coll cosmosDataCollection, partitionKey string, hasPartitionKey bool, operation cosmosBatchOperation) map[string]any {
	switch strings.ToLower(operation.OperationType) {
	case "create":
		return s.cosmosBatchCreateLocked(account, dbID, collID, coll, partitionKey, hasPartitionKey, operation.ResourceBody, false)
	case "upsert":
		return s.cosmosBatchCreateLocked(account, dbID, collID, coll, partitionKey, hasPartitionKey, operation.ResourceBody, true)
	case "read":
		if operation.ID == "" {
			return cosmosBatchResult(400, "", nil)
		}
		doc, ok := s.findCosmosDocumentLocked(account, dbID, collID, operation.ID, cosmosBatchPartitionHeaders(partitionKey, hasPartitionKey))
		if !ok {
			return cosmosBatchResult(404, "", nil)
		}
		return cosmosBatchResult(http.StatusOK, doc.ETag, cosmosDocumentBody(doc))
	case "replace":
		if operation.ID == "" || operation.ResourceBody == nil {
			return cosmosBatchResult(400, "", nil)
		}
		existing, ok := s.findCosmosDocumentLocked(account, dbID, collID, operation.ID, cosmosBatchPartitionHeaders(partitionKey, hasPartitionKey))
		if !ok {
			return cosmosBatchResult(404, "", nil)
		}
		body := cloneMap(operation.ResourceBody)
		body["id"] = operation.ID
		doc := s.newCosmosDocumentLocked(dbID, collID, operation.ID, existing.PartitionKey, body, existing)
		s.dataDocuments[cosmosDataDocumentKey(account, dbID, collID, existing.PartitionKey, operation.ID)] = doc
		return cosmosBatchResult(http.StatusOK, doc.ETag, cosmosDocumentBody(doc))
	case "patch":
		if operation.ID == "" || operation.ResourceBody == nil {
			return cosmosBatchResult(400, "", nil)
		}
		existing, ok := s.findCosmosDocumentLocked(account, dbID, collID, operation.ID, cosmosBatchPartitionHeaders(partitionKey, hasPartitionKey))
		if !ok {
			return cosmosBatchResult(404, "", nil)
		}
		patchBody, err := gojson.Marshal(operation.ResourceBody)
		if err != nil {
			return cosmosBatchResult(400, "", nil)
		}
		ops, condition, err := parseCosmosPatchRequest(patchBody)
		if err != nil {
			return cosmosBatchResult(400, "", nil)
		}
		if !cosmosPatchConditionMatches(existing.Body, condition) {
			return cosmosBatchResult(http.StatusPreconditionFailed, "", nil)
		}
		body := cloneCosmosJSONMap(existing.Body)
		for _, op := range ops {
			if !applyCosmosPatchOperation(body, op) {
				return cosmosBatchResult(400, "", nil)
			}
		}
		doc := s.newCosmosDocumentLocked(dbID, collID, operation.ID, existing.PartitionKey, body, existing)
		s.dataDocuments[cosmosDataDocumentKey(account, dbID, collID, existing.PartitionKey, operation.ID)] = doc
		return cosmosBatchResult(http.StatusOK, doc.ETag, cosmosDocumentBody(doc))
	case "delete":
		if operation.ID == "" {
			return cosmosBatchResult(400, "", nil)
		}
		doc, ok := s.findCosmosDocumentLocked(account, dbID, collID, operation.ID, cosmosBatchPartitionHeaders(partitionKey, hasPartitionKey))
		if !ok {
			return cosmosBatchResult(404, "", nil)
		}
		delete(s.dataDocuments, cosmosDataDocumentKey(account, dbID, collID, doc.PartitionKey, operation.ID))
		return cosmosBatchResult(http.StatusNoContent, "", nil)
	default:
		return cosmosBatchResult(400, "", nil)
	}
}

func (s *CosmosDBService) cosmosBatchCreateLocked(account, dbID, collID string, coll cosmosDataCollection, partitionKey string, hasPartitionKey bool, body map[string]any, upsert bool) map[string]any {
	if body == nil {
		return cosmosBatchResult(400, "", nil)
	}
	body = cloneMap(body)
	id := stringValue(body["id"])
	if id == "" {
		id = s.nextCosmosTokenLocked("doc")
		body["id"] = id
	}
	pk := partitionKey
	if !hasPartitionKey {
		pk = cosmosPartitionKeyValue(body, coll, nil)
	}
	key := cosmosDataDocumentKey(account, dbID, collID, pk, id)
	existing, exists := s.dataDocuments[key]
	if exists && !upsert {
		return cosmosBatchResult(http.StatusConflict, "", nil)
	}
	doc := s.newCosmosDocumentLocked(dbID, collID, id, pk, body, existing)
	s.dataDocuments[key] = doc
	status := http.StatusCreated
	if exists && upsert {
		status = http.StatusOK
	}
	return cosmosBatchResult(status, doc.ETag, cosmosDocumentBody(doc))
}

func cosmosBatchPartitionHeaders(partitionKey string, ok bool) http.Header {
	if !ok {
		return nil
	}
	return http.Header{"x-ms-documentdb-partitionkey": []string{`["` + partitionKey + `"]`}}
}

func cosmosBatchResult(statusCode int, etag string, resourceBody map[string]any) map[string]any {
	result := map[string]any{
		"statusCode":    statusCode,
		"subStatusCode": 0,
		"requestCharge": 1.0,
	}
	if etag != "" {
		result["eTag"] = quotedCosmosETag(etag)
	}
	if resourceBody != nil {
		result["resourceBody"] = resourceBody
	}
	return result
}

func cosmosBatchStatusCode(result map[string]any) int {
	switch value := result["statusCode"].(type) {
	case int:
		return value
	case float64:
		return int(value)
	case gojson.Number:
		out, _ := strconv.Atoi(string(value))
		return out
	default:
		return 0
	}
}

func cloneCosmosDataDocuments(input map[string]cosmosDataDocument) map[string]cosmosDataDocument {
	out := make(map[string]cosmosDataDocument, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func (s *CosmosDBService) queryCosmosDocuments(account, dbID, collID string, body []byte, headers http.Header) (*service.Response, error) {
	var input struct {
		Query      string           `json:"query"`
		Parameters []cosmosSQLParam `json:"parameters"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return cosmosSQLError(http.StatusBadRequest, "BadRequest", "The query request body is invalid.")
		}
	}
	if input.Query == "" {
		input.Query = "SELECT * FROM c"
	}

	s.mu.RLock()
	coll, collOK := s.dataCollections[cosmosDataCollectionKey(account, dbID, collID)]
	documents := s.cosmosDocumentsLocked(account, dbID, collID)
	s.mu.RUnlock()
	if !collOK {
		return cosmosSQLNotFound(collID)
	}

	results := executeCosmosSQLQuery(input.Query, input.Parameters, documents)
	skip := decodeCosmosContinuation(headers.Get("x-ms-continuation"))
	if skip > len(results) {
		skip = len(results)
	}
	results = results[skip:]
	next := ""
	if max := parseCosmosMaxItemCount(headers.Get("x-ms-max-item-count")); max > 0 && len(results) > max {
		next = encodeCosmosContinuation(skip + max)
		results = results[:max]
	}
	payload := map[string]any{"_rid": coll.RID, "_count": len(results), "Documents": results}
	extra := map[string]string{"x-ms-item-count": strconv.Itoa(len(results))}
	if next != "" {
		extra["x-ms-continuation"] = next
	}
	return cosmosSQLJSONResponseWithHeaders(http.StatusOK, payload, "", extra)
}

func (s *CosmosDBService) cosmosQueryPlan(account, dbID, collID string, body []byte) (*service.Response, error) {
	s.mu.RLock()
	_, ok := s.dataCollections[cosmosDataCollectionKey(account, dbID, collID)]
	s.mu.RUnlock()
	if !ok {
		return cosmosSQLNotFound(collID)
	}

	var input struct {
		Query string `json:"query"`
	}
	_ = gojson.Unmarshal(body, &input)
	hasSelectValue := strings.Contains(strings.ToUpper(input.Query), "SELECT VALUE")
	queryInfo := map[string]any{
		"distinctType":                  "None",
		"top":                           nil,
		"offset":                        nil,
		"limit":                         nil,
		"orderBy":                       []any{},
		"orderByExpressions":            []any{},
		"groupByExpressions":            []any{},
		"groupByAliases":                []any{},
		"aggregates":                    []any{},
		"groupByAliasToAggregateType":   map[string]any{},
		"aggregateAliasToAggregateType": map[string]any{},
		"rewrittenQuery":                "",
		"hasSelectValue":                hasSelectValue,
		"hasNonStreamingOrderBy":        false,
		"dCountInfo":                    nil,
	}
	queryRange := map[string]any{
		"min":            "",
		"max":            "FF",
		"isMinInclusive": true,
		"isMaxInclusive": false,
	}
	plan := map[string]any{
		"partitionedQueryExecutionInfoVersion": 1,
		"queryInfo":                            queryInfo,
		"queryRanges":                          []any{queryRange},
	}
	return cosmosSQLJSONResponse(http.StatusOK, plan, "")
}

type cosmosSQLParam struct {
	Name  string `json:"name"`
	Value any    `json:"value"`
}

func executeCosmosSQLQuery(sql string, params []cosmosSQLParam, documents []map[string]any) []any {
	sql = normalizeCosmosSQLWhitespace(sql)
	inputRows := cosmosSQLInputRows(sql, documents)
	if cosmosQueryUsesStaticTime(sql) {
		inputRows = cosmosRowsWithStaticNow(inputRows, time.Now().UTC())
	}
	filtered := make([]map[string]any, 0, len(documents))
	whereClause := cosmosSQLClause(sql, "WHERE", []string{"GROUP BY", "ORDER BY", "OFFSET", "LIMIT"})
	for _, doc := range inputRows {
		if cosmosSQLMatches(doc, whereClause, params) {
			filtered = append(filtered, doc)
		}
	}
	if orderClause := cosmosSQLClause(sql, "ORDER BY", []string{"OFFSET", "LIMIT"}); orderClause != "" {
		applyCosmosOrderBy(filtered, orderClause)
	}
	if top, ok := parseCosmosTop(sql); ok && top < len(filtered) {
		filtered = filtered[:top]
	}
	if offset, limit, ok := parseCosmosOffsetLimit(sql); ok {
		filtered = applyCosmosOffsetLimit(filtered, offset, limit)
	}
	if grouped, ok := executeCosmosGroupByAggregates(sql, filtered); ok {
		return grouped
	}
	if cosmosSelectValueCount(sql) {
		return []any{len(filtered)}
	}
	if agg, field, ok := parseCosmosAggregate(sql); ok {
		return []any{cosmosAggregate(agg, field, filtered)}
	}
	if projected, ok := executeCosmosAggregateValueObjectProjection(sql, filtered); ok {
		return projected
	}
	if projected, ok := executeCosmosAggregateProjection(sql, filtered); ok {
		return projected
	}
	out := make([]any, 0, len(filtered))
	if fields, selectValue, distinct := parseCosmosSelectFields(sql); len(fields) > 0 {
		for _, doc := range filtered {
			if selectValue && len(fields) == 1 {
				out = append(out, cosmosExpressionValue(doc, fields[0]))
				continue
			}
			projection := make(map[string]any, len(fields))
			for i, field := range fields {
				name := cosmosProjectionAlias(field, i+1)
				value := cosmosExpressionValue(doc, field)
				if cosmosIsUndefined(value) {
					continue
				}
				projection[name] = value
			}
			out = append(out, projection)
		}
		if distinct {
			return distinctCosmosValues(out)
		}
		return out
	}
	for _, doc := range filtered {
		out = append(out, cosmosVisibleDocument(doc))
	}
	return out
}

type cosmosSQLJoin struct {
	Alias    string
	Path     string
	Subquery string
}

func cosmosQueryUsesStaticTime(sql string) bool {
	upper := strings.ToUpper(sql)
	return strings.Contains(upper, "GETCURRENTDATETIMESTATIC()") ||
		strings.Contains(upper, "GETCURRENTTIMESTAMPSTATIC()") ||
		strings.Contains(upper, "GETCURRENTTICKSSTATIC()")
}

func cosmosRowsWithStaticNow(rows []map[string]any, now time.Time) []map[string]any {
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		next := make(map[string]any, len(row)+1)
		for key, value := range row {
			next[key] = value
		}
		next[cosmosStaticNowField] = now
		out = append(out, next)
	}
	return out
}

func cosmosVisibleDocument(doc map[string]any) map[string]any {
	if _, exists := doc[cosmosStaticNowField]; !exists {
		return doc
	}
	out := make(map[string]any, len(doc)-1)
	for key, value := range doc {
		if key == cosmosStaticNowField {
			continue
		}
		out[key] = value
	}
	return out
}

func cosmosSQLInputRows(sql string, documents []map[string]any) []map[string]any {
	if findCosmosTopLevelKeyword(sql, "FROM") < 0 {
		return []map[string]any{{}}
	}
	joins := parseCosmosJoins(sql)
	if len(joins) == 0 {
		return documents
	}
	baseAlias := parseCosmosFromAlias(sql)
	if baseAlias == "" {
		baseAlias = "c"
	}
	rows := make([]map[string]any, 0, len(documents))
	for _, doc := range documents {
		row := make(map[string]any, len(doc)+1)
		for key, value := range doc {
			row[key] = value
		}
		row[baseAlias] = doc
		rows = append(rows, row)
	}
	for _, join := range joins {
		expanded := make([]map[string]any, 0)
		for _, row := range rows {
			values := cosmosSQLJoinValues(row, join)
			for _, value := range values {
				next := make(map[string]any, len(row)+1)
				for key, existing := range row {
					next[key] = existing
				}
				next[join.Alias] = value
				expanded = append(expanded, next)
			}
		}
		rows = expanded
	}
	return rows
}

func cosmosSQLJoinValues(row map[string]any, join cosmosSQLJoin) []any {
	if join.Subquery != "" {
		return cosmosSubqueryValues(row, join.Subquery)
	}
	values, ok := cosmosExpressionValue(row, join.Path).([]any)
	if !ok {
		return nil
	}
	return values
}

func parseCosmosFromAlias(sql string) string {
	fromClause := cosmosSQLClause(sql, "FROM", []string{"WHERE", "GROUP BY", "ORDER BY", "OFFSET", "LIMIT"})
	if joinIdx := strings.Index(strings.ToUpper(fromClause), " JOIN "); joinIdx >= 0 {
		fromClause = strings.TrimSpace(fromClause[:joinIdx])
	}
	fields := strings.Fields(fromClause)
	if len(fields) == 0 {
		return ""
	}
	if len(fields) == 1 {
		return fields[0]
	}
	return fields[len(fields)-1]
}

func parseCosmosJoins(sql string) []cosmosSQLJoin {
	fromClause := cosmosSQLClause(sql, "FROM", []string{"WHERE", "GROUP BY", "ORDER BY", "OFFSET", "LIMIT"})
	joins := make([]cosmosSQLJoin, 0)
	for {
		joinIdx := findCosmosTopLevelKeyword(fromClause, "JOIN")
		if joinIdx < 0 {
			break
		}
		fromClause = strings.TrimSpace(fromClause[joinIdx+len("JOIN"):])
		nextIdx := findCosmosTopLevelKeyword(fromClause, "JOIN")
		source := fromClause
		if nextIdx >= 0 {
			source = strings.TrimSpace(fromClause[:nextIdx])
			fromClause = strings.TrimSpace(fromClause[nextIdx:])
		} else {
			fromClause = ""
		}
		if join, ok := parseCosmosJoinSource(source); ok {
			joins = append(joins, join)
		}
		if nextIdx < 0 {
			break
		}
	}
	return joins
}

func parseCosmosJoinSource(source string) (cosmosSQLJoin, bool) {
	source = strings.TrimSpace(source)
	if source == "" {
		return cosmosSQLJoin{}, false
	}
	if strings.HasPrefix(source, "(") {
		closeIdx := cosmosMatchingCloseParen(source, 0)
		if closeIdx < 0 {
			return cosmosSQLJoin{}, false
		}
		subquery := strings.TrimSpace(source[1:closeIdx])
		alias := cosmosSubqueryJoinAlias(subquery, source[closeIdx+1:])
		if alias == "" {
			return cosmosSQLJoin{}, false
		}
		return cosmosSQLJoin{Alias: alias, Subquery: subquery}, true
	}
	fields := strings.Fields(source)
	if len(fields) < 3 || !strings.EqualFold(fields[1], "IN") {
		return cosmosSQLJoin{}, false
	}
	return cosmosSQLJoin{Alias: fields[0], Path: fields[2]}, true
}

func cosmosSubqueryJoinAlias(subquery, suffix string) string {
	fields := strings.Fields(strings.TrimSpace(suffix))
	if len(fields) > 0 {
		if strings.EqualFold(fields[0], "AS") && len(fields) > 1 {
			return fields[1]
		}
		return fields[0]
	}
	selectFields, selectValue, _ := parseCosmosSelectFields(subquery)
	if !selectValue || len(selectFields) != 1 {
		return ""
	}
	return fieldAlias(selectFields[0])
}

func (s *CosmosDBService) cosmosDocumentsLocked(account, dbID, collID string) []map[string]any {
	prefix := strings.ToLower(account) + "|doc|" + strings.ToLower(dbID) + "|" + strings.ToLower(collID) + "|"
	values := make([]map[string]any, 0)
	for key, doc := range s.dataDocuments {
		if strings.HasPrefix(key, prefix) {
			values = append(values, cosmosDocumentBody(doc))
		}
	}
	sort.Slice(values, func(i, j int) bool { return fmt.Sprint(values[i]["id"]) < fmt.Sprint(values[j]["id"]) })
	return values
}

func (s *CosmosDBService) findCosmosDocumentLocked(account, dbID, collID, docID string, headers http.Header) (cosmosDataDocument, bool) {
	if pk, ok := cosmosPartitionKeyFromHeader(headers); ok {
		if doc, found := s.dataDocuments[cosmosDataDocumentKey(account, dbID, collID, pk, docID)]; found {
			return doc, true
		}
	}
	prefix := strings.ToLower(account) + "|doc|" + strings.ToLower(dbID) + "|" + strings.ToLower(collID) + "|"
	suffix := "|" + strings.ToLower(docID)
	for key, doc := range s.dataDocuments {
		if strings.HasPrefix(key, prefix) && strings.HasSuffix(key, suffix) {
			return doc, true
		}
	}
	return cosmosDataDocument{}, false
}

func (s *CosmosDBService) newCosmosDocumentLocked(dbID, collID, docID, partitionKey string, body map[string]any, existing cosmosDataDocument) cosmosDataDocument {
	now := time.Now().UTC()
	rid := existing.RID
	if rid == "" {
		rid = s.nextCosmosTokenLocked("doc")
	}
	docBody := cloneMap(body)
	delete(docBody, "_etag")
	delete(docBody, "_ts")
	delete(docBody, "_rid")
	delete(docBody, "_self")
	delete(docBody, "_attachments")
	etag := s.nextCosmosTokenLocked("etag")
	ts := now.Unix()
	docBody["id"] = docID
	docBody["_rid"] = rid
	docBody["_self"] = "dbs/" + dbID + "/colls/" + collID + "/docs/" + docID
	docBody["_etag"] = quotedCosmosETag(etag)
	docBody["_ts"] = ts
	docBody["_attachments"] = "attachments/"
	return cosmosDataDocument{ID: docID, PartitionKey: partitionKey, RID: rid, ETag: etag, TS: ts, Body: docBody, UpdatedAt: now}
}

func (s *CosmosDBService) nextCosmosToken(prefix string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.nextCosmosTokenLocked(prefix)
}

func (s *CosmosDBService) nextCosmosTokenLocked(prefix string) string {
	s.nextID++
	return fmt.Sprintf("%s-%08d", prefix, s.nextID)
}

func cosmosDatabaseBody(db cosmosDataDatabase) map[string]any {
	return map[string]any{"id": db.ID, "_rid": db.RID, "_self": "dbs/" + db.ID + "/", "_etag": quotedCosmosETag(db.ETag), "_ts": db.TS, "_colls": "colls/", "_users": "users/"}
}

func cosmosCollectionBody(dbID string, coll cosmosDataCollection) map[string]any {
	return map[string]any{"id": coll.ID, "_rid": coll.RID, "_self": "dbs/" + dbID + "/colls/" + coll.ID + "/", "_etag": quotedCosmosETag(coll.ETag), "_ts": coll.TS, "partitionKey": coll.PartitionKey, "indexingPolicy": coll.IndexingPolicy, "_docs": "docs/", "_sprocs": "sprocs/", "_triggers": "triggers/", "_udfs": "udfs/", "_conflicts": "conflicts/"}
}

func cosmosDocumentBody(doc cosmosDataDocument) map[string]any {
	return cloneMap(doc.Body)
}

func cosmosPartitionKeyValue(doc map[string]any, coll cosmosDataCollection, headers http.Header) string {
	if pk, ok := cosmosPartitionKeyFromHeader(headers); ok {
		return pk
	}
	paths, _ := coll.PartitionKey["paths"].([]any)
	if len(paths) == 0 {
		return ""
	}
	path := strings.TrimPrefix(fmt.Sprint(paths[0]), "/")
	if path == "" {
		return ""
	}
	if value, ok := doc[path]; ok && value != nil {
		return fmt.Sprint(value)
	}
	return ""
}

func cosmosPartitionKeyFromHeader(headers http.Header) (string, bool) {
	raw := strings.TrimSpace(headers.Get("x-ms-documentdb-partitionkey"))
	if raw == "" || raw == "[]" {
		return "", false
	}
	var values []any
	if err := gojson.Unmarshal([]byte(raw), &values); err == nil && len(values) > 0 && values[0] != nil {
		return fmt.Sprint(values[0]), true
	}
	return strings.Trim(raw, `"`), true
}

func parseCosmosPatchRequest(body []byte) ([]map[string]any, string, error) {
	var wrapped struct {
		Operations []map[string]any `json:"operations"`
		Condition  string           `json:"condition"`
	}
	if err := gojson.Unmarshal(body, &wrapped); err == nil && wrapped.Operations != nil {
		return wrapped.Operations, wrapped.Condition, nil
	}
	var raw []map[string]any
	if err := gojson.Unmarshal(body, &raw); err != nil {
		return nil, "", err
	}
	return raw, "", nil
}

func cosmosPatchConditionMatches(doc map[string]any, condition string) bool {
	condition = strings.TrimSpace(condition)
	if condition == "" {
		return true
	}
	upper := strings.ToUpper(condition)
	whereIndex := strings.Index(upper, "WHERE")
	if whereIndex >= 0 {
		condition = strings.TrimSpace(condition[whereIndex+len("WHERE"):])
	}
	return cosmosSQLMatches(doc, condition, nil)
}

func applyCosmosPatchOperation(doc map[string]any, op map[string]any) bool {
	path := stringValue(op["path"])
	if len(cosmosPatchPathParts(path)) == 0 {
		return false
	}
	switch strings.ToLower(stringValue(op["op"])) {
	case "add", "set":
		return cosmosPatchWrite(doc, path, op["value"], strings.ToLower(stringValue(op["op"])))
	case "replace":
		return cosmosPatchWrite(doc, path, op["value"], "replace")
	case "remove":
		return cosmosPatchRemove(doc, path)
	case "incr":
		current, _ := cosmosPatchGet(doc, path)
		return cosmosPatchWrite(doc, path, normalizeCosmosNumber(cosmosNumber(current)+cosmosNumber(op["value"])), "set")
	case "move":
		from := stringValue(op["from"])
		if cosmosPatchPathIsDescendant(from, path) {
			return false
		}
		value, ok := cosmosPatchGet(doc, from)
		if !ok {
			return false
		}
		if !cosmosPatchRemove(doc, from) {
			return false
		}
		return cosmosPatchWrite(doc, path, value, "add")
	default:
		return false
	}
}

func cosmosPatchGet(doc map[string]any, path string) (any, bool) {
	return cosmosPatchGetValue(doc, cosmosPatchPathParts(path))
}

func cosmosPatchWrite(doc map[string]any, path string, value any, mode string) bool {
	parts := cosmosPatchPathParts(path)
	_, ok := cosmosPatchWriteValue(doc, parts, value, mode)
	return ok
}

func cosmosPatchRemove(doc map[string]any, path string) bool {
	_, ok := cosmosPatchRemoveValue(doc, cosmosPatchPathParts(path))
	return ok
}

func cosmosPatchGetValue(current any, parts []string) (any, bool) {
	if len(parts) == 0 {
		return nil, false
	}
	switch typed := current.(type) {
	case map[string]any:
		value, ok := typed[parts[0]]
		if !ok {
			return nil, false
		}
		if len(parts) == 1 {
			return value, true
		}
		return cosmosPatchGetValue(value, parts[1:])
	case []any:
		index, ok := cosmosPatchArrayIndex(parts[0], len(typed), false)
		if !ok {
			return nil, false
		}
		value := typed[index]
		if len(parts) == 1 {
			return value, true
		}
		return cosmosPatchGetValue(value, parts[1:])
	default:
		return nil, false
	}
}

func cosmosPatchWriteValue(current any, parts []string, value any, mode string) (any, bool) {
	if len(parts) == 0 {
		return current, false
	}
	if len(parts) == 1 {
		switch typed := current.(type) {
		case map[string]any:
			if mode == "replace" {
				if _, ok := typed[parts[0]]; !ok {
					return current, false
				}
			}
			typed[parts[0]] = value
			return typed, true
		case []any:
			return cosmosPatchWriteArray(typed, parts[0], value, mode)
		default:
			return current, false
		}
	}

	switch typed := current.(type) {
	case map[string]any:
		child, ok := typed[parts[0]]
		if !ok || child == nil {
			if mode == "replace" {
				return current, false
			}
			child = map[string]any{}
		}
		updated, ok := cosmosPatchWriteValue(child, parts[1:], value, mode)
		if !ok {
			return current, false
		}
		typed[parts[0]] = updated
		return typed, true
	case []any:
		index, ok := cosmosPatchArrayIndex(parts[0], len(typed), false)
		if !ok {
			return current, false
		}
		updated, ok := cosmosPatchWriteValue(typed[index], parts[1:], value, mode)
		if !ok {
			return current, false
		}
		typed[index] = updated
		return typed, true
	default:
		return current, false
	}
}

func cosmosPatchWriteArray(values []any, part string, value any, mode string) ([]any, bool) {
	switch mode {
	case "add":
		index, ok := cosmosPatchArrayIndex(part, len(values), true)
		if !ok {
			return values, false
		}
		if index == len(values) {
			return append(values, value), true
		}
		values = append(values, nil)
		copy(values[index+1:], values[index:])
		values[index] = value
		return values, true
	case "set", "replace":
		index, ok := cosmosPatchArrayIndex(part, len(values), false)
		if !ok {
			return values, false
		}
		values[index] = value
		return values, true
	default:
		return values, false
	}
}

func cosmosPatchRemoveValue(current any, parts []string) (any, bool) {
	if len(parts) == 0 {
		return current, false
	}
	if len(parts) == 1 {
		switch typed := current.(type) {
		case map[string]any:
			if _, ok := typed[parts[0]]; !ok {
				return current, false
			}
			delete(typed, parts[0])
			return typed, true
		case []any:
			index, ok := cosmosPatchArrayIndex(parts[0], len(typed), false)
			if !ok {
				return current, false
			}
			return append(typed[:index], typed[index+1:]...), true
		default:
			return current, false
		}
	}

	switch typed := current.(type) {
	case map[string]any:
		child, ok := typed[parts[0]]
		if !ok {
			return current, false
		}
		updated, ok := cosmosPatchRemoveValue(child, parts[1:])
		if !ok {
			return current, false
		}
		typed[parts[0]] = updated
		return typed, true
	case []any:
		index, ok := cosmosPatchArrayIndex(parts[0], len(typed), false)
		if !ok {
			return current, false
		}
		updated, ok := cosmosPatchRemoveValue(typed[index], parts[1:])
		if !ok {
			return current, false
		}
		typed[index] = updated
		return typed, true
	default:
		return current, false
	}
}

func cosmosPatchArrayIndex(part string, length int, allowEnd bool) (int, bool) {
	if part == "-" {
		return length, allowEnd
	}
	index, err := strconv.Atoi(part)
	if err != nil || index < 0 {
		return 0, false
	}
	if index < length || allowEnd && index == length {
		return index, true
	}
	return 0, false
}

func cosmosPatchPathIsDescendant(parent, child string) bool {
	parentParts := cosmosPatchPathParts(parent)
	childParts := cosmosPatchPathParts(child)
	if len(parentParts) == 0 || len(childParts) <= len(parentParts) {
		return false
	}
	for i, part := range parentParts {
		if childParts[i] != part {
			return false
		}
	}
	return true
}

func cosmosPatchPathParts(path string) []string {
	path = strings.TrimPrefix(strings.TrimSpace(path), "/")
	if path == "" {
		return nil
	}
	parts := strings.Split(path, "/")
	for i, part := range parts {
		part = strings.ReplaceAll(part, "~1", "/")
		part = strings.ReplaceAll(part, "~0", "~")
		parts[i] = part
	}
	return parts
}

func cosmosSQLMatches(doc map[string]any, where string, params []cosmosSQLParam) bool {
	where = strings.TrimSpace(where)
	if where == "" {
		return true
	}
	return cosmosSQLPredicateMatches(doc, where, params)
}

func cosmosSQLPredicateMatches(doc map[string]any, predicate string, params []cosmosSQLParam) bool {
	predicate = strings.TrimSpace(predicate)
	if predicate == "" {
		return true
	}
	if strings.HasPrefix(predicate, "(") && cosmosMatchingCloseParen(predicate, 0) == len(predicate)-1 {
		return cosmosSQLPredicateMatches(doc, predicate[1:len(predicate)-1], params)
	}
	if idx := findCosmosTopLevelKeyword(predicate, "OR"); idx >= 0 {
		return cosmosSQLPredicateMatches(doc, predicate[:idx], params) || cosmosSQLPredicateMatches(doc, predicate[idx+2:], params)
	}
	if idx := findCosmosTopLevelKeyword(predicate, "AND"); idx >= 0 {
		return cosmosSQLPredicateMatches(doc, predicate[:idx], params) && cosmosSQLPredicateMatches(doc, predicate[idx+3:], params)
	}
	if strings.HasPrefix(strings.ToUpper(predicate), "NOT ") {
		return !cosmosSQLPredicateMatches(doc, predicate[4:], params)
	}
	if idx := findCosmosTopLevelKeyword(predicate, "IN"); idx >= 0 {
		left := strings.TrimSpace(predicate[:idx])
		right := strings.TrimSpace(predicate[idx+2:])
		if strings.HasPrefix(right, "(") && strings.HasSuffix(right, ")") {
			actual := cosmosExpressionValue(doc, left)
			for _, raw := range splitCosmosExpressionList(right[1 : len(right)-1]) {
				if cosmosValuesEqual(actual, cosmosExpressionValueWithParams(doc, raw, params)) {
					return true
				}
			}
			return false
		}
	}
	if idx := findCosmosTopLevelKeyword(predicate, "BETWEEN"); idx >= 0 {
		left := strings.TrimSpace(predicate[:idx])
		right := strings.TrimSpace(predicate[idx+len("BETWEEN"):])
		andIdx := findCosmosTopLevelKeyword(right, "AND")
		if andIdx >= 0 {
			actual := cosmosExpressionValue(doc, left)
			low := cosmosExpressionValueWithParams(doc, right[:andIdx], params)
			high := cosmosExpressionValueWithParams(doc, right[andIdx+3:], params)
			return cosmosCompareValues(actual, low) >= 0 && cosmosCompareValues(actual, high) <= 0
		}
	}
	if idx := findCosmosTopLevelKeyword(predicate, "LIKE"); idx >= 0 {
		actual := cosmosExpressionValue(doc, predicate[:idx])
		expected := cosmosExpressionValueWithParams(doc, predicate[idx+4:], params)
		return cosmosLikeMatches(fmt.Sprint(actual), fmt.Sprint(expected))
	}
	idx, op := cosmosTopLevelComparison(predicate)
	if idx < 0 {
		if value, ok := cosmosExpressionValue(doc, predicate).(bool); ok {
			return value
		}
		return true
	}
	actual := cosmosExpressionValue(doc, predicate[:idx])
	expected := cosmosExpressionValueWithParams(doc, predicate[idx+len(op):], params)
	switch op {
	case "=", "EQ":
		return fmt.Sprint(actual) == fmt.Sprint(expected)
	case "!=", "<>":
		return fmt.Sprint(actual) != fmt.Sprint(expected)
	case "LIKE":
		return cosmosLikeMatches(fmt.Sprint(actual), fmt.Sprint(expected))
	case ">", ">=", "<", "<=":
		left := cosmosNumber(actual)
		right := cosmosNumber(expected)
		switch op {
		case ">":
			return left > right
		case ">=":
			return left >= right
		case "<":
			return left < right
		case "<=":
			return left <= right
		}
	}
	return true
}

func cosmosValuesEqual(left, right any) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	if leftNumber, ok := cosmosNumericValue(left); ok {
		if rightNumber, ok := cosmosNumericValue(right); ok {
			return leftNumber == rightNumber
		}
	}
	switch left.(type) {
	case map[string]any, []any:
		return cosmosDistinctKey(left) == cosmosDistinctKey(right)
	}
	switch right.(type) {
	case map[string]any, []any:
		return cosmosDistinctKey(left) == cosmosDistinctKey(right)
	}
	return fmt.Sprint(left) == fmt.Sprint(right)
}

func cosmosCompareValues(left, right any) int {
	if leftNumber, ok := cosmosNumericValue(left); ok {
		if rightNumber, ok := cosmosNumericValue(right); ok {
			return cmpFloat64(leftNumber, rightNumber)
		}
	}
	return strings.Compare(fmt.Sprint(left), fmt.Sprint(right))
}

func cmpFloat64(left, right float64) int {
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}

func cosmosLikeMatches(value, pattern string) bool {
	quoted := regexp.QuoteMeta(pattern)
	quoted = strings.ReplaceAll(quoted, "%", ".*")
	quoted = strings.ReplaceAll(quoted, "_", ".")
	matched, err := regexp.MatchString("^"+quoted+"$", value)
	return err == nil && matched
}

func cosmosQueryValue(raw string, params []cosmosSQLParam) any {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "@") {
		for _, param := range params {
			if strings.EqualFold(param.Name, raw) {
				return param.Value
			}
		}
	}
	if len(raw) >= 2 {
		first := raw[0]
		last := raw[len(raw)-1]
		if first == last && (first == '\'' || first == '"') {
			return raw[1 : len(raw)-1]
		}
	}
	if strings.EqualFold(raw, "true") {
		return true
	}
	if strings.EqualFold(raw, "false") {
		return false
	}
	if strings.EqualFold(raw, "null") {
		return nil
	}
	if strings.EqualFold(raw, "undefined") {
		return cosmosUndefined
	}
	if strings.EqualFold(raw, "NaN") {
		return math.NaN()
	}
	if strings.EqualFold(raw, "Infinity") {
		return math.Inf(1)
	}
	if value, err := strconv.ParseFloat(raw, 64); err == nil {
		return value
	}
	return raw
}

func applyCosmosOrderBy(documents []map[string]any, clause string) {
	fields := parseCosmosOrderByFields(clause)
	if len(fields) == 0 {
		return
	}
	sort.SliceStable(documents, func(i, j int) bool {
		for _, field := range fields {
			left := cosmosExpressionValue(documents[i], field.Expression)
			right := cosmosExpressionValue(documents[j], field.Expression)
			cmp := cosmosCompareValues(left, right)
			if cmp == 0 {
				continue
			}
			if field.Desc {
				return cmp > 0
			}
			return cmp < 0
		}
		return false
	})
}

type cosmosOrderByField struct {
	Expression string
	Desc       bool
}

func parseCosmosOrderByFields(clause string) []cosmosOrderByField {
	parts := splitCosmosExpressionList(clause)
	out := make([]cosmosOrderByField, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		desc := false
		upper := strings.ToUpper(part)
		switch {
		case strings.HasSuffix(upper, " DESC"):
			desc = true
			part = strings.TrimSpace(part[:len(part)-len(" DESC")])
		case strings.HasSuffix(upper, " ASC"):
			part = strings.TrimSpace(part[:len(part)-len(" ASC")])
		}
		if strings.HasPrefix(strings.ToUpper(part), "RANK ") {
			desc = true
			part = strings.TrimSpace(part[len("RANK "):])
		}
		out = append(out, cosmosOrderByField{Expression: part, Desc: desc})
	}
	return out
}

func parseCosmosTop(sql string) (int, bool) {
	upper := strings.ToUpper(sql)
	selectIdx := strings.Index(upper, "SELECT")
	fromIdx := strings.Index(upper, "FROM")
	if selectIdx < 0 || fromIdx < 0 || fromIdx <= selectIdx {
		return 0, false
	}
	clause := strings.TrimSpace(sql[selectIdx+len("SELECT") : fromIdx])
	for {
		previous := clause
		if strings.HasPrefix(strings.ToUpper(clause), "DISTINCT ") {
			clause = strings.TrimSpace(clause[len("DISTINCT "):])
		}
		if strings.HasPrefix(strings.ToUpper(clause), "VALUE ") {
			clause = strings.TrimSpace(clause[len("VALUE "):])
		}
		if clause == previous {
			break
		}
	}
	match := regexp.MustCompile(`(?i)^TOP\s+(\d+)\b`).FindStringSubmatch(clause)
	if len(match) != 2 {
		return 0, false
	}
	top, err := strconv.Atoi(match[1])
	if err != nil || top < 0 {
		return 0, false
	}
	return top, true
}

func parseCosmosOffsetLimit(sql string) (int, int, bool) {
	fields := strings.Fields(strings.ToUpper(sql))
	for i := 0; i < len(fields); i++ {
		if fields[i] != "OFFSET" || i+3 >= len(fields) || fields[i+2] != "LIMIT" {
			continue
		}
		offset, offsetErr := strconv.Atoi(fields[i+1])
		limit, limitErr := strconv.Atoi(fields[i+3])
		if offsetErr != nil || limitErr != nil || offset < 0 || limit < 0 {
			return 0, 0, false
		}
		return offset, limit, true
	}
	return 0, 0, false
}

func applyCosmosOffsetLimit(documents []map[string]any, offset, limit int) []map[string]any {
	if offset > len(documents) {
		return nil
	}
	documents = documents[offset:]
	if limit < len(documents) {
		return documents[:limit]
	}
	return documents
}

func normalizeCosmosSQLWhitespace(sql string) string {
	var out strings.Builder
	inQuote := false
	var quote rune
	pendingSpace := false
	for _, r := range sql {
		if inQuote {
			out.WriteRune(r)
			if r == quote {
				inQuote = false
			}
			continue
		}
		switch {
		case r == '\'' || r == '"':
			if pendingSpace && out.Len() > 0 {
				out.WriteByte(' ')
			}
			pendingSpace = false
			inQuote = true
			quote = r
			out.WriteRune(r)
		case unicode.IsSpace(r):
			pendingSpace = true
		default:
			if pendingSpace && out.Len() > 0 {
				out.WriteByte(' ')
			}
			pendingSpace = false
			out.WriteRune(r)
		}
	}
	return strings.TrimSpace(out.String())
}

func executeCosmosAggregateProjection(sql string, documents []map[string]any) ([]any, bool) {
	fields, selectValue, _ := parseCosmosSelectFields(sql)
	if selectValue || len(fields) == 0 {
		return nil, false
	}
	row := make(map[string]any, len(fields))
	for i, field := range fields {
		agg, aggExpr, ok := parseCosmosAggregateExpression(field)
		if !ok {
			return nil, false
		}
		value := cosmosAggregate(agg, aggExpr, documents)
		if cosmosIsUndefined(value) {
			continue
		}
		row[cosmosProjectionAlias(field, i+1)] = value
	}
	return []any{row}, true
}

func executeCosmosAggregateValueObjectProjection(sql string, documents []map[string]any) ([]any, bool) {
	fields, selectValue, _ := parseCosmosSelectFields(sql)
	if !selectValue || len(fields) != 1 {
		return nil, false
	}
	expression := strings.TrimSpace(cosmosExpressionWithoutAlias(fields[0]))
	if !strings.HasPrefix(expression, "{") || !strings.HasSuffix(expression, "}") {
		return nil, false
	}
	body := strings.TrimSpace(expression[1 : len(expression)-1])
	if body == "" {
		return nil, false
	}
	row := make(map[string]any)
	for _, part := range splitCosmosExpressionList(body) {
		colon := cosmosTopLevelColon(part)
		if colon < 0 {
			return nil, false
		}
		key := cosmosObjectExpressionKey(part[:colon])
		if key == "" {
			return nil, false
		}
		agg, aggExpr, ok := parseCosmosAggregateExpression(part[colon+1:])
		if !ok {
			return nil, false
		}
		value := cosmosAggregate(agg, aggExpr, documents)
		if cosmosIsUndefined(value) {
			continue
		}
		row[key] = value
	}
	return []any{row}, true
}

func executeCosmosGroupByAggregates(sql string, documents []map[string]any) ([]any, bool) {
	groupExpr := cosmosSQLClause(sql, "GROUP BY", []string{"ORDER BY", "OFFSET", "LIMIT"})
	if groupExpr == "" {
		return nil, false
	}
	fields, _, _ := parseCosmosSelectFields(sql)
	if len(fields) == 0 {
		return nil, false
	}
	groupExprs := splitCosmosExpressionList(groupExpr)

	type groupBucket struct {
		values []any
		rows   []map[string]any
	}
	buckets := make(map[string]groupBucket)
	keys := make([]string, 0)
	for _, doc := range documents {
		values := make([]any, 0, len(groupExprs))
		for _, expr := range groupExprs {
			values = append(values, cosmosExpressionValue(doc, expr))
		}
		key := cosmosDistinctKey(values)
		bucket, exists := buckets[key]
		if !exists {
			bucket.values = values
			keys = append(keys, key)
		}
		bucket.rows = append(bucket.rows, doc)
		buckets[key] = bucket
	}
	sort.Slice(keys, func(i, j int) bool {
		return fmt.Sprint(buckets[keys[i]].values) < fmt.Sprint(buckets[keys[j]].values)
	})

	out := make([]any, 0, len(keys))
	for _, key := range keys {
		bucket := buckets[key]
		if len(bucket.rows) == 0 {
			continue
		}
		row := make(map[string]any, len(fields))
		for i, field := range fields {
			expression := cosmosExpressionWithoutAlias(field)
			alias := cosmosProjectionAlias(field, i+1)
			if agg, aggExpr, ok := parseCosmosAggregateExpression(expression); ok {
				row[alias] = cosmosGroupedAggregate(agg, aggExpr, bucket.rows)
				continue
			}
			value := cosmosExpressionValue(bucket.rows[0], expression)
			if cosmosIsUndefined(value) {
				continue
			}
			row[alias] = value
		}
		out = append(out, row)
	}
	return out, true
}

func cosmosSQLClause(sql, keyword string, endKeywords []string) string {
	start := findCosmosTopLevelKeyword(sql, keyword)
	if start < 0 {
		return ""
	}
	start += len(keyword)
	end := len(sql)
	for _, endKeyword := range endKeywords {
		if idx := findCosmosTopLevelKeyword(sql[start:], endKeyword); idx >= 0 && start+idx < end {
			end = start + idx
		}
	}
	return strings.TrimSpace(sql[start:end])
}

func parseCosmosSelectFields(sql string) ([]string, bool, bool) {
	selectIdx := findCosmosTopLevelKeyword(sql, "SELECT")
	fromIdx := findCosmosTopLevelKeyword(sql, "FROM")
	if selectIdx < 0 {
		return nil, false, false
	}
	endIdx := fromIdx
	if endIdx < 0 {
		endIdx = len(sql)
		for _, endKeyword := range []string{"WHERE", "GROUP BY", "ORDER BY", "OFFSET", "LIMIT"} {
			if idx := findCosmosTopLevelKeyword(sql[selectIdx+len("SELECT"):], endKeyword); idx >= 0 && selectIdx+len("SELECT")+idx < endIdx {
				endIdx = selectIdx + len("SELECT") + idx
			}
		}
	}
	if endIdx <= selectIdx {
		return nil, false, false
	}
	clause := strings.TrimSpace(sql[selectIdx+len("SELECT") : endIdx])
	selectValue := false
	distinct := false
	for {
		previous := clause
		upperClause := strings.ToUpper(clause)
		switch {
		case strings.HasPrefix(upperClause, "VALUE "):
			selectValue = true
			clause = strings.TrimSpace(clause[len("VALUE "):])
		case strings.HasPrefix(upperClause, "DISTINCT "):
			distinct = true
			clause = strings.TrimSpace(clause[len("DISTINCT "):])
		default:
			clause = stripCosmosSelectTop(clause)
		}
		if clause == previous {
			break
		}
	}
	if clause == "*" {
		return nil, selectValue, distinct
	}
	parts := splitCosmosExpressionList(clause)
	fields := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			fields = append(fields, part)
		}
	}
	return fields, selectValue, distinct
}

func stripCosmosSelectTop(clause string) string {
	return strings.TrimSpace(regexp.MustCompile(`(?i)^TOP\s+\d+\s+`).ReplaceAllString(strings.TrimSpace(clause), ""))
}

func cosmosSelectValueCount(sql string) bool {
	fields, selectValue, _ := parseCosmosSelectFields(sql)
	if !selectValue || len(fields) != 1 {
		return false
	}
	expression := strings.ToUpper(strings.TrimSpace(cosmosExpressionWithoutAlias(fields[0])))
	return expression == "COUNT(1)" || expression == "COUNT(*)"
}

func parseCosmosAggregate(sql string) (string, string, bool) {
	fields, selectValue, _ := parseCosmosSelectFields(sql)
	if !selectValue || len(fields) != 1 {
		return "", "", false
	}
	return parseCosmosAggregateExpression(fields[0])
}

func parseCosmosAggregateExpression(expression string) (string, string, bool) {
	expression = strings.TrimSpace(cosmosExpressionWithoutAlias(expression))
	upper := strings.ToUpper(expression)
	if (strings.HasPrefix(upper, "COUNT(") || strings.HasPrefix(upper, "COUNT (")) && strings.HasSuffix(expression, ")") {
		open := strings.Index(expression, "(")
		if open < 0 {
			return "", "", false
		}
		return "COUNT", strings.TrimSpace(expression[open+1 : len(expression)-1]), true
	}
	for _, agg := range []string{"SUM", "AVG", "MIN", "MAX"} {
		prefix := agg + "("
		if !strings.HasPrefix(upper, prefix) || !strings.HasSuffix(expression, ")") {
			continue
		}
		field := expression[len(prefix) : len(expression)-1]
		return agg, strings.TrimSpace(field), true
	}
	return "", "", false
}

func cosmosGroupedAggregate(agg, field string, docs []map[string]any) any {
	if agg == "COUNT" {
		field = strings.TrimSpace(field)
		if field == "" || field == "*" || field == "1" {
			return len(docs)
		}
		count := 0
		for _, doc := range docs {
			if cosmosCountExpressionDefined(doc, field) {
				count++
			}
		}
		return count
	}
	return cosmosAggregate(agg, field, docs)
}

func cosmosAggregate(agg, field string, docs []map[string]any) any {
	if len(docs) == 0 {
		return 0
	}
	if agg == "COUNT" {
		field = strings.TrimSpace(field)
		if field == "" || field == "*" || field == "1" {
			return len(docs)
		}
		count := 0
		for _, doc := range docs {
			if cosmosCountExpressionDefined(doc, field) {
				count++
			}
		}
		return count
	}
	sum := 0.0
	min := math.Inf(1)
	max := math.Inf(-1)
	count := 0
	for _, doc := range docs {
		value, state := cosmosAggregateNumericValue(doc, field)
		switch state {
		case cosmosAggregateValueInvalid:
			return cosmosUndefined
		case cosmosAggregateValueUndefined:
			continue
		}
		sum += value
		count++
		if value < min {
			min = value
		}
		if value > max {
			max = value
		}
	}
	switch agg {
	case "SUM":
		return normalizeCosmosNumber(sum)
	case "AVG":
		if count == 0 {
			return 0
		}
		return normalizeCosmosNumber(sum / float64(count))
	case "MIN":
		if count == 0 {
			return 0
		}
		return normalizeCosmosNumber(min)
	case "MAX":
		if count == 0 {
			return 0
		}
		return normalizeCosmosNumber(max)
	default:
		return 0
	}
}

type cosmosAggregateValueState int

const (
	cosmosAggregateValueValid cosmosAggregateValueState = iota
	cosmosAggregateValueUndefined
	cosmosAggregateValueInvalid
)

func cosmosAggregateNumericValue(doc map[string]any, expression string) (float64, cosmosAggregateValueState) {
	expression = strings.TrimSpace(expression)
	if value, ok := cosmosPathValue(doc, expression); ok {
		number, numberOK := cosmosNumericValue(value)
		if !numberOK {
			return 0, cosmosAggregateValueInvalid
		}
		return number, cosmosAggregateValueValid
	}
	if cosmosAggregateExpressionLooksLikeDirectPath(expression) {
		return 0, cosmosAggregateValueUndefined
	}
	value := cosmosExpressionValue(doc, expression)
	if cosmosIsUndefined(value) {
		return 0, cosmosAggregateValueUndefined
	}
	number, ok := cosmosNumericValue(value)
	if !ok {
		return 0, cosmosAggregateValueInvalid
	}
	return number, cosmosAggregateValueValid
}

func cosmosAggregateExpressionLooksLikeDirectPath(expression string) bool {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		return false
	}
	if strings.ContainsAny(expression, "+-*/%(),{}") {
		return false
	}
	return strings.ContainsAny(expression, ".[") || cosmosCountExpressionLooksLikePath(expression)
}

func cosmosCountExpressionDefined(doc map[string]any, expression string) bool {
	expression = strings.TrimSpace(expression)
	if expression == "" || expression == "*" || expression == "1" {
		return true
	}
	if _, ok := cosmosPathValue(doc, expression); ok {
		return true
	}
	if cosmosCountExpressionLooksLikePath(expression) {
		return false
	}
	return !cosmosIsUndefined(cosmosExpressionValue(doc, expression))
}

func cosmosCountExpressionLooksLikePath(expression string) bool {
	upper := strings.ToUpper(strings.TrimSpace(expression))
	switch upper {
	case "TRUE", "FALSE", "NULL":
		return false
	}
	if strings.ContainsAny(expression, ".[") {
		return true
	}
	if expression == "" {
		return false
	}
	for _, r := range expression {
		if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func cosmosFieldValue(doc map[string]any, field string) any {
	field = strings.TrimSpace(field)
	if aliasIdx := strings.Index(strings.ToUpper(field), " AS "); aliasIdx >= 0 {
		field = strings.TrimSpace(field[:aliasIdx])
	}
	value, _ := cosmosPathValue(doc, field)
	return value
}

func cosmosExpressionValue(doc map[string]any, expression string) any {
	expression = cosmosExpressionWithoutAlias(strings.TrimSpace(expression))
	upper := strings.ToUpper(expression)
	if value, ok := cosmosArithmeticExpressionValue(doc, expression); ok {
		return value
	}
	switch {
	case strings.HasPrefix(expression, "{") && strings.HasSuffix(expression, "}"):
		return cosmosObjectExpressionValue(doc, expression)
	case strings.HasPrefix(expression, "[") && strings.HasSuffix(expression, "]"):
		return cosmosArrayExpressionValue(doc, expression)
	case strings.HasPrefix(expression, "(") && cosmosMatchingCloseParen(expression, 0) == len(expression)-1:
		inner := strings.TrimSpace(expression[1 : len(expression)-1])
		if strings.HasPrefix(strings.ToUpper(inner), "SELECT") {
			return cosmosScalarSubqueryValue(doc, inner)
		}
		return cosmosExpressionValue(doc, inner)
	case strings.HasPrefix(upper, "ARRAY(") || strings.HasPrefix(upper, "ARRAY ("):
		subquery, ok := cosmosParenthesizedExpressionBody(expression, "ARRAY")
		if ok {
			return cosmosSubqueryValues(doc, subquery)
		}
		return nil
	case strings.HasPrefix(upper, "EXISTS"):
		subquery, ok := cosmosParenthesizedExpressionBody(expression, "EXISTS")
		return ok && cosmosSubqueryHasRows(doc, subquery)
	case strings.HasPrefix(upper, "IIF(") && strings.HasSuffix(expression, ")"):
		args := splitCosmosExpressionList(expression[4 : len(expression)-1])
		if len(args) != 3 {
			return nil
		}
		if cosmosBooleanExpression(doc, args[0]) {
			return cosmosExpressionValue(doc, args[1])
		}
		return cosmosExpressionValue(doc, args[2])
	case strings.HasPrefix(upper, "CHOOSE(") && strings.HasSuffix(expression, ")"):
		return cosmosChoose(doc, splitCosmosExpressionList(expression[len("CHOOSE("):len(expression)-1]))
	case strings.HasPrefix(upper, "DOCUMENTID(") && strings.HasSuffix(expression, ")"):
		return cosmosDocumentID(doc, splitCosmosExpressionList(expression[len("DOCUMENTID("):len(expression)-1]))
	case strings.HasPrefix(upper, "IS_DEFINED(") && strings.HasSuffix(expression, ")"):
		_, ok := cosmosExpressionDefined(doc, expression[11:len(expression)-1])
		return ok
	case strings.HasPrefix(upper, "IS_NULL(") && strings.HasSuffix(expression, ")"):
		value, ok := cosmosExpressionDefined(doc, expression[8:len(expression)-1])
		return ok && value == nil
	case strings.HasPrefix(upper, "IS_NUMBER(") && strings.HasSuffix(expression, ")"):
		value := cosmosExpressionValue(doc, expression[10:len(expression)-1])
		_, ok := cosmosNumericValue(value)
		return ok
	case strings.HasPrefix(upper, "IS_FINITE_NUMBER(") && strings.HasSuffix(expression, ")"):
		value := cosmosExpressionValue(doc, expression[len("IS_FINITE_NUMBER("):len(expression)-1])
		number, ok := cosmosNumericValue(value)
		return ok && !math.IsNaN(number) && !math.IsInf(number, 0)
	case strings.HasPrefix(upper, "IS_STRING(") && strings.HasSuffix(expression, ")"):
		value, ok := cosmosExpressionDefined(doc, expression[10:len(expression)-1])
		_, isString := value.(string)
		return ok && isString
	case strings.HasPrefix(upper, "IS_BOOL(") && strings.HasSuffix(expression, ")"):
		value, ok := cosmosExpressionDefined(doc, expression[8:len(expression)-1])
		_, isBool := value.(bool)
		return ok && isBool
	case strings.HasPrefix(upper, "IS_ARRAY(") && strings.HasSuffix(expression, ")"):
		value, ok := cosmosExpressionDefined(doc, expression[9:len(expression)-1])
		_, isArray := value.([]any)
		return ok && isArray
	case strings.HasPrefix(upper, "IS_OBJECT(") && strings.HasSuffix(expression, ")"):
		value, ok := cosmosExpressionDefined(doc, expression[10:len(expression)-1])
		_, isObject := value.(map[string]any)
		return ok && isObject
	case strings.HasPrefix(upper, "IS_INTEGER(") && strings.HasSuffix(expression, ")"):
		value, ok := cosmosExpressionDefined(doc, expression[11:len(expression)-1])
		if !ok {
			return false
		}
		number, ok := cosmosNumericValue(value)
		return ok && number == math.Trunc(number)
	case strings.HasPrefix(upper, "IS_PRIMITIVE(") && strings.HasSuffix(expression, ")"):
		value, ok := cosmosExpressionDefined(doc, expression[13:len(expression)-1])
		return ok && cosmosPrimitiveValue(value)
	case strings.HasPrefix(upper, "FULLTEXTCONTAINS(") && strings.HasSuffix(expression, ")"):
		args := splitCosmosExpressionList(expression[len("FULLTEXTCONTAINS(") : len(expression)-1])
		return cosmosFullTextContains(doc, args)
	case strings.HasPrefix(upper, "FULLTEXTCONTAINSALL(") && strings.HasSuffix(expression, ")"):
		args := splitCosmosExpressionList(expression[len("FULLTEXTCONTAINSALL(") : len(expression)-1])
		return cosmosFullTextContainsAllOrAny(doc, args, true)
	case strings.HasPrefix(upper, "FULLTEXTCONTAINSANY(") && strings.HasSuffix(expression, ")"):
		args := splitCosmosExpressionList(expression[len("FULLTEXTCONTAINSANY(") : len(expression)-1])
		return cosmosFullTextContainsAllOrAny(doc, args, false)
	case strings.HasPrefix(upper, "RRF(") && strings.HasSuffix(expression, ")"):
		args := splitCosmosExpressionList(expression[len("RRF(") : len(expression)-1])
		return cosmosRRFScore(doc, args)
	case strings.HasPrefix(upper, "ST_AREA(") && strings.HasSuffix(expression, ")"):
		args := splitCosmosExpressionList(expression[len("ST_AREA(") : len(expression)-1])
		return cosmosSpatialArea(doc, args)
	case strings.HasPrefix(upper, "ST_ISVALIDDETAILED(") && strings.HasSuffix(expression, ")"):
		args := splitCosmosExpressionList(expression[len("ST_ISVALIDDETAILED(") : len(expression)-1])
		return cosmosSpatialIsValidDetailed(doc, args)
	case strings.HasPrefix(upper, "ST_ISVALID(") && strings.HasSuffix(expression, ")"):
		args := splitCosmosExpressionList(expression[len("ST_ISVALID(") : len(expression)-1])
		return cosmosSpatialIsValid(doc, args)
	case strings.HasPrefix(upper, "ST_INTERSECTS(") && strings.HasSuffix(expression, ")"):
		args := splitCosmosExpressionList(expression[len("ST_INTERSECTS(") : len(expression)-1])
		return cosmosSpatialIntersects(doc, args)
	case strings.HasPrefix(upper, "ST_WITHIN(") && strings.HasSuffix(expression, ")"):
		args := splitCosmosExpressionList(expression[len("ST_WITHIN(") : len(expression)-1])
		return cosmosSpatialWithin(doc, args)
	case strings.HasPrefix(upper, "ST_DISTANCE(") && strings.HasSuffix(expression, ")"):
		args := splitCosmosExpressionList(expression[len("ST_DISTANCE(") : len(expression)-1])
		return cosmosSpatialDistance(doc, args)
	case strings.HasPrefix(upper, "FULLTEXTSCORE(") && strings.HasSuffix(expression, ")"):
		args := splitCosmosExpressionList(expression[len("FULLTEXTSCORE(") : len(expression)-1])
		return cosmosFullTextScore(doc, args)
	case strings.HasPrefix(upper, "VECTORDISTANCE(") && strings.HasSuffix(expression, ")"):
		args := splitCosmosExpressionList(expression[len("VECTORDISTANCE(") : len(expression)-1])
		return cosmosVectorDistance(doc, args)
	case strings.HasPrefix(upper, "CONTAINS(") && strings.HasSuffix(expression, ")"):
		args := splitCosmosExpressionList(expression[9 : len(expression)-1])
		if len(args) < 2 {
			return cosmosUndefined
		}
		return cosmosStringPredicate(doc, args, strings.Contains)
	case strings.HasPrefix(upper, "STARTSWITH(") && strings.HasSuffix(expression, ")"):
		args := splitCosmosExpressionList(expression[11 : len(expression)-1])
		if len(args) < 2 {
			return cosmosUndefined
		}
		return cosmosStringPredicate(doc, args, strings.HasPrefix)
	case strings.HasPrefix(upper, "ENDSWITH(") && strings.HasSuffix(expression, ")"):
		args := splitCosmosExpressionList(expression[9 : len(expression)-1])
		if len(args) < 2 {
			return cosmosUndefined
		}
		return cosmosStringPredicate(doc, args, strings.HasSuffix)
	case strings.HasPrefix(upper, "STRINGEQUALS(") && strings.HasSuffix(expression, ")"):
		args := splitCosmosExpressionList(expression[13 : len(expression)-1])
		if len(args) < 2 {
			return cosmosUndefined
		}
		return cosmosStringEqualsPredicate(doc, args)
	case strings.HasPrefix(upper, "REGEXMATCH(") && strings.HasSuffix(expression, ")"):
		args := splitCosmosExpressionList(expression[11 : len(expression)-1])
		if len(args) < 2 {
			return cosmosUndefined
		}
		return cosmosRegexMatch(doc, args)
	case strings.HasPrefix(upper, "LOWER(") && strings.HasSuffix(expression, ")"):
		value, ok := cosmosStringFunctionArgument(doc, expression[6:len(expression)-1])
		if !ok {
			return cosmosUndefined
		}
		return strings.ToLower(value)
	case strings.HasPrefix(upper, "UPPER(") && strings.HasSuffix(expression, ")"):
		value, ok := cosmosStringFunctionArgument(doc, expression[6:len(expression)-1])
		if !ok {
			return cosmosUndefined
		}
		return strings.ToUpper(value)
	case strings.HasPrefix(upper, "LENGTH(") && strings.HasSuffix(expression, ")"):
		value, ok := cosmosStringFunctionArgument(doc, expression[7:len(expression)-1])
		if !ok {
			return cosmosUndefined
		}
		return len(value)
	case strings.HasPrefix(upper, "LEFT(") && strings.HasSuffix(expression, ")"):
		args := splitCosmosExpressionList(expression[5 : len(expression)-1])
		return cosmosLeftString(doc, args)
	case strings.HasPrefix(upper, "RIGHT(") && strings.HasSuffix(expression, ")"):
		args := splitCosmosExpressionList(expression[6 : len(expression)-1])
		return cosmosRightString(doc, args)
	case strings.HasPrefix(upper, "SUBSTRING(") && strings.HasSuffix(expression, ")"):
		args := splitCosmosExpressionList(expression[10 : len(expression)-1])
		if len(args) < 3 {
			return cosmosUndefined
		}
		value, ok := cosmosStringFunctionArgument(doc, args[0])
		if !ok {
			return cosmosUndefined
		}
		start, ok := cosmosNumericFunctionArgument(doc, args[1])
		if !ok {
			return cosmosUndefined
		}
		length, ok := cosmosNumericFunctionArgument(doc, args[2])
		if !ok {
			return cosmosUndefined
		}
		return cosmosSubstring(value, int(start), int(length))
	case strings.HasPrefix(upper, "LTRIM(") && strings.HasSuffix(expression, ")"):
		args := splitCosmosExpressionList(expression[6 : len(expression)-1])
		return cosmosTrimLeftString(doc, args)
	case strings.HasPrefix(upper, "RTRIM(") && strings.HasSuffix(expression, ")"):
		args := splitCosmosExpressionList(expression[6 : len(expression)-1])
		return cosmosTrimRightString(doc, args)
	case strings.HasPrefix(upper, "TRIM(") && strings.HasSuffix(expression, ")"):
		args := splitCosmosExpressionList(expression[5 : len(expression)-1])
		return cosmosTrimString(doc, args)
	case strings.HasPrefix(upper, "REPLACE(") && strings.HasSuffix(expression, ")"):
		args := splitCosmosExpressionList(expression[8 : len(expression)-1])
		return cosmosReplaceString(doc, args)
	case strings.HasPrefix(upper, "INDEX_OF(") && strings.HasSuffix(expression, ")"):
		args := splitCosmosExpressionList(expression[9 : len(expression)-1])
		return cosmosIndexOfString(doc, args)
	case strings.HasPrefix(upper, "CONCAT(") && strings.HasSuffix(expression, ")"):
		args := splitCosmosExpressionList(expression[7 : len(expression)-1])
		if len(args) < 2 {
			return cosmosUndefined
		}
		var out strings.Builder
		for _, arg := range args {
			value, ok := cosmosStringFunctionArgument(doc, arg)
			if !ok {
				return cosmosUndefined
			}
			out.WriteString(value)
		}
		return out.String()
	case strings.HasPrefix(upper, "STRINGJOIN(") && strings.HasSuffix(expression, ")"):
		return cosmosStringJoin(doc, splitCosmosExpressionList(expression[len("STRINGJOIN("):len(expression)-1]))
	case strings.HasPrefix(upper, "STRINGSPLIT(") && strings.HasSuffix(expression, ")"):
		return cosmosStringSplit(doc, splitCosmosExpressionList(expression[len("STRINGSPLIT("):len(expression)-1]))
	case strings.HasPrefix(upper, "REPLICATE(") && strings.HasSuffix(expression, ")"):
		args := splitCosmosExpressionList(expression[10 : len(expression)-1])
		return cosmosReplicateString(doc, args)
	case strings.HasPrefix(upper, "REVERSE(") && strings.HasSuffix(expression, ")"):
		value, ok := cosmosStringFunctionArgument(doc, expression[8:len(expression)-1])
		if !ok {
			return cosmosUndefined
		}
		return cosmosReverseString(value)
	case strings.HasPrefix(upper, "STRINGTOARRAY(") && strings.HasSuffix(expression, ")"):
		return cosmosStringToArray(cosmosExpressionValue(doc, expression[len("STRINGTOARRAY("):len(expression)-1]))
	case strings.HasPrefix(upper, "STRINGTONUMBER(") && strings.HasSuffix(expression, ")"):
		return cosmosStringToNumber(cosmosExpressionValue(doc, expression[15:len(expression)-1]))
	case strings.HasPrefix(upper, "STRINGTOBOOLEAN(") && strings.HasSuffix(expression, ")"):
		return cosmosStringToBoolean(cosmosExpressionValue(doc, expression[16:len(expression)-1]))
	case strings.HasPrefix(upper, "STRINGTONULL(") && strings.HasSuffix(expression, ")"):
		return cosmosStringToNull(cosmosExpressionValue(doc, expression[len("STRINGTONULL("):len(expression)-1]))
	case strings.HasPrefix(upper, "STRINGTOOBJECT(") && strings.HasSuffix(expression, ")"):
		return cosmosStringToObject(cosmosExpressionValue(doc, expression[len("STRINGTOOBJECT("):len(expression)-1]))
	case strings.HasPrefix(upper, "TOSTRING(") && strings.HasSuffix(expression, ")"):
		return cosmosToString(cosmosExpressionValue(doc, expression[9:len(expression)-1]))
	case upper == "GETCURRENTDATETIME()":
		return cosmosFormatDateTime(time.Now().UTC())
	case upper == "GETCURRENTDATETIMESTATIC()":
		return cosmosFormatDateTime(cosmosStaticNow(doc))
	case upper == "GETCURRENTTIMESTAMP()":
		now := time.Now().UTC()
		return now.Unix()*1000 + int64(now.Nanosecond()/int(time.Millisecond))
	case upper == "GETCURRENTTIMESTAMPSTATIC()":
		now := cosmosStaticNow(doc)
		return now.Unix()*1000 + int64(now.Nanosecond()/int(time.Millisecond))
	case upper == "GETCURRENTTICKS()":
		now := time.Now().UTC()
		return now.Unix()*10000000 + int64(now.Nanosecond()/100)
	case upper == "GETCURRENTTICKSSTATIC()":
		now := cosmosStaticNow(doc)
		return now.Unix()*10000000 + int64(now.Nanosecond()/100)
	case strings.HasPrefix(upper, "DATETIMEFROMPARTS(") && strings.HasSuffix(expression, ")"):
		return cosmosDateTimeFromParts(doc, splitCosmosExpressionList(expression[len("DATETIMEFROMPARTS("):len(expression)-1]))
	case strings.HasPrefix(upper, "DATETIMEBIN(") && strings.HasSuffix(expression, ")"):
		return cosmosDateTimeBin(doc, splitCosmosExpressionList(expression[len("DATETIMEBIN("):len(expression)-1]))
	case strings.HasPrefix(upper, "DATETIMEADD(") && strings.HasSuffix(expression, ")"):
		return cosmosDateTimeAdd(doc, splitCosmosExpressionList(expression[len("DATETIMEADD("):len(expression)-1]))
	case strings.HasPrefix(upper, "DATETIMEDIFF(") && strings.HasSuffix(expression, ")"):
		return cosmosDateTimeDiff(doc, splitCosmosExpressionList(expression[len("DATETIMEDIFF("):len(expression)-1]))
	case strings.HasPrefix(upper, "DATETIMEPART(") && strings.HasSuffix(expression, ")"):
		return cosmosDateTimePart(doc, splitCosmosExpressionList(expression[len("DATETIMEPART("):len(expression)-1]))
	case strings.HasPrefix(upper, "DATETIMETOTIMESTAMP(") && strings.HasSuffix(expression, ")"):
		return cosmosDateTimeToTimestamp(doc, splitCosmosExpressionList(expression[len("DATETIMETOTIMESTAMP("):len(expression)-1]))
	case strings.HasPrefix(upper, "TIMESTAMPTODATETIME(") && strings.HasSuffix(expression, ")"):
		return cosmosTimestampToDateTime(doc, splitCosmosExpressionList(expression[len("TIMESTAMPTODATETIME("):len(expression)-1]))
	case strings.HasPrefix(upper, "DATETIMETOTICKS(") && strings.HasSuffix(expression, ")"):
		return cosmosDateTimeToTicks(doc, splitCosmosExpressionList(expression[len("DATETIMETOTICKS("):len(expression)-1]))
	case strings.HasPrefix(upper, "TICKSTODATETIME(") && strings.HasSuffix(expression, ")"):
		return cosmosTicksToDateTime(doc, splitCosmosExpressionList(expression[len("TICKSTODATETIME("):len(expression)-1]))
	case strings.HasPrefix(upper, "ACOS(") && strings.HasSuffix(expression, ")"):
		return cosmosUnaryMathFunction(doc, expression[len("ACOS("):len(expression)-1], math.Acos)
	case strings.HasPrefix(upper, "ASIN(") && strings.HasSuffix(expression, ")"):
		return cosmosUnaryMathFunction(doc, expression[len("ASIN("):len(expression)-1], math.Asin)
	case strings.HasPrefix(upper, "ATAN(") && strings.HasSuffix(expression, ")"):
		return cosmosUnaryMathFunction(doc, expression[len("ATAN("):len(expression)-1], math.Atan)
	case strings.HasPrefix(upper, "ATN2(") && strings.HasSuffix(expression, ")"):
		return cosmosBinaryMathFunction(doc, splitCosmosExpressionList(expression[len("ATN2("):len(expression)-1]), math.Atan2)
	case strings.HasPrefix(upper, "COS(") && strings.HasSuffix(expression, ")"):
		return cosmosUnaryMathFunction(doc, expression[len("COS("):len(expression)-1], math.Cos)
	case strings.HasPrefix(upper, "COT(") && strings.HasSuffix(expression, ")"):
		return cosmosUnaryMathFunction(doc, expression[len("COT("):len(expression)-1], func(value float64) float64 {
			return 1 / math.Tan(value)
		})
	case strings.HasPrefix(upper, "DEGREES(") && strings.HasSuffix(expression, ")"):
		return cosmosUnaryMathFunction(doc, expression[len("DEGREES("):len(expression)-1], func(value float64) float64 {
			return value * 180 / math.Pi
		})
	case strings.HasPrefix(upper, "RADIANS(") && strings.HasSuffix(expression, ")"):
		return cosmosUnaryMathFunction(doc, expression[len("RADIANS("):len(expression)-1], func(value float64) float64 {
			return value * math.Pi / 180
		})
	case upper == "RAND()":
		return rand.Float64()
	case strings.HasPrefix(upper, "SQUARE(") && strings.HasSuffix(expression, ")"):
		return cosmosUnaryMathFunction(doc, expression[len("SQUARE("):len(expression)-1], func(value float64) float64 {
			return value * value
		})
	case strings.HasPrefix(upper, "SIN(") && strings.HasSuffix(expression, ")"):
		return cosmosUnaryMathFunction(doc, expression[len("SIN("):len(expression)-1], math.Sin)
	case strings.HasPrefix(upper, "TAN(") && strings.HasSuffix(expression, ")"):
		return cosmosUnaryMathFunction(doc, expression[len("TAN("):len(expression)-1], math.Tan)
	case strings.HasPrefix(upper, "INTADD(") && strings.HasSuffix(expression, ")"):
		return cosmosBinaryIntegerFunction(doc, splitCosmosExpressionList(expression[len("INTADD("):len(expression)-1]), func(left, right int64) (int64, bool) {
			return left + right, true
		})
	case strings.HasPrefix(upper, "INTSUB(") && strings.HasSuffix(expression, ")"):
		return cosmosBinaryIntegerFunction(doc, splitCosmosExpressionList(expression[len("INTSUB("):len(expression)-1]), func(left, right int64) (int64, bool) {
			return left - right, true
		})
	case strings.HasPrefix(upper, "INTMUL(") && strings.HasSuffix(expression, ")"):
		return cosmosBinaryIntegerFunction(doc, splitCosmosExpressionList(expression[len("INTMUL("):len(expression)-1]), func(left, right int64) (int64, bool) {
			return left * right, true
		})
	case strings.HasPrefix(upper, "INTDIV(") && strings.HasSuffix(expression, ")"):
		return cosmosBinaryIntegerFunction(doc, splitCosmosExpressionList(expression[len("INTDIV("):len(expression)-1]), func(left, right int64) (int64, bool) {
			if right == 0 {
				return 0, false
			}
			return left / right, true
		})
	case strings.HasPrefix(upper, "INTMOD(") && strings.HasSuffix(expression, ")"):
		return cosmosBinaryIntegerFunction(doc, splitCosmosExpressionList(expression[len("INTMOD("):len(expression)-1]), func(left, right int64) (int64, bool) {
			if right == 0 {
				return 0, false
			}
			return left % right, true
		})
	case strings.HasPrefix(upper, "INTBITAND(") && strings.HasSuffix(expression, ")"):
		return cosmosBinaryIntegerFunction(doc, splitCosmosExpressionList(expression[len("INTBITAND("):len(expression)-1]), func(left, right int64) (int64, bool) {
			return left & right, true
		})
	case strings.HasPrefix(upper, "INTBITOR(") && strings.HasSuffix(expression, ")"):
		return cosmosBinaryIntegerFunction(doc, splitCosmosExpressionList(expression[len("INTBITOR("):len(expression)-1]), func(left, right int64) (int64, bool) {
			return left | right, true
		})
	case strings.HasPrefix(upper, "INTBITXOR(") && strings.HasSuffix(expression, ")"):
		return cosmosBinaryIntegerFunction(doc, splitCosmosExpressionList(expression[len("INTBITXOR("):len(expression)-1]), func(left, right int64) (int64, bool) {
			return left ^ right, true
		})
	case strings.HasPrefix(upper, "INTBITLEFTSHIFT(") && strings.HasSuffix(expression, ")"):
		return cosmosBinaryIntegerFunction(doc, splitCosmosExpressionList(expression[len("INTBITLEFTSHIFT("):len(expression)-1]), func(left, right int64) (int64, bool) {
			if right < 0 || right > 63 {
				return 0, false
			}
			return left << uint(right), true
		})
	case strings.HasPrefix(upper, "INTBITRIGHTSHIFT(") && strings.HasSuffix(expression, ")"):
		return cosmosBinaryIntegerFunction(doc, splitCosmosExpressionList(expression[len("INTBITRIGHTSHIFT("):len(expression)-1]), func(left, right int64) (int64, bool) {
			if right < 0 || right > 63 {
				return 0, false
			}
			return left >> uint(right), true
		})
	case strings.HasPrefix(upper, "INTBITNOT(") && strings.HasSuffix(expression, ")"):
		value, ok := cosmosIntegerValue(cosmosExpressionValue(doc, expression[len("INTBITNOT("):len(expression)-1]))
		if !ok {
			return cosmosUndefined
		}
		return normalizeCosmosNumber(float64(^value))
	case strings.HasPrefix(upper, "ABS(") && strings.HasSuffix(expression, ")"):
		if value, ok := cosmosNumericValue(cosmosExpressionValue(doc, expression[4:len(expression)-1])); ok {
			return normalizeCosmosNumber(math.Abs(value))
		}
		return nil
	case strings.HasPrefix(upper, "CEILING(") && strings.HasSuffix(expression, ")"):
		if value, ok := cosmosNumericValue(cosmosExpressionValue(doc, expression[8:len(expression)-1])); ok {
			return normalizeCosmosNumber(math.Ceil(value))
		}
		return nil
	case strings.HasPrefix(upper, "FLOOR(") && strings.HasSuffix(expression, ")"):
		if value, ok := cosmosNumericValue(cosmosExpressionValue(doc, expression[6:len(expression)-1])); ok {
			return normalizeCosmosNumber(math.Floor(value))
		}
		return nil
	case strings.HasPrefix(upper, "ROUND(") && strings.HasSuffix(expression, ")"):
		if value, ok := cosmosNumericValue(cosmosExpressionValue(doc, expression[6:len(expression)-1])); ok {
			return normalizeCosmosNumber(math.Round(value))
		}
		return nil
	case strings.HasPrefix(upper, "SQRT(") && strings.HasSuffix(expression, ")"):
		if value, ok := cosmosNumericValue(cosmosExpressionValue(doc, expression[5:len(expression)-1])); ok {
			return normalizeCosmosNumber(math.Sqrt(value))
		}
		return nil
	case strings.HasPrefix(upper, "POWER(") && strings.HasSuffix(expression, ")"):
		args := splitCosmosExpressionList(expression[6 : len(expression)-1])
		if len(args) < 2 {
			return nil
		}
		base, baseOK := cosmosNumericValue(cosmosExpressionValue(doc, args[0]))
		exp, expOK := cosmosNumericValue(cosmosExpressionValue(doc, args[1]))
		if !baseOK || !expOK {
			return nil
		}
		return normalizeCosmosNumber(math.Pow(base, exp))
	case strings.HasPrefix(upper, "LOG10(") && strings.HasSuffix(expression, ")"):
		if value, ok := cosmosNumericValue(cosmosExpressionValue(doc, expression[6:len(expression)-1])); ok {
			return normalizeCosmosNumber(math.Log10(value))
		}
		return nil
	case strings.HasPrefix(upper, "NUMBERBIN(") && strings.HasSuffix(expression, ")"):
		return cosmosNumberBin(doc, splitCosmosExpressionList(expression[len("NUMBERBIN("):len(expression)-1]))
	case strings.HasPrefix(upper, "LOG(") && strings.HasSuffix(expression, ")"):
		args := splitCosmosExpressionList(expression[4 : len(expression)-1])
		if len(args) == 0 {
			return nil
		}
		value, ok := cosmosNumericValue(cosmosExpressionValue(doc, args[0]))
		if !ok {
			return nil
		}
		if len(args) > 1 {
			base, ok := cosmosNumericValue(cosmosExpressionValue(doc, args[1]))
			if !ok {
				return nil
			}
			return normalizeCosmosNumber(math.Log(value) / math.Log(base))
		}
		return normalizeCosmosNumber(math.Log(value))
	case strings.HasPrefix(upper, "EXP(") && strings.HasSuffix(expression, ")"):
		if value, ok := cosmosNumericValue(cosmosExpressionValue(doc, expression[4:len(expression)-1])); ok {
			return normalizeCosmosNumber(math.Exp(value))
		}
		return nil
	case strings.HasPrefix(upper, "SIGN(") && strings.HasSuffix(expression, ")"):
		if value, ok := cosmosNumericValue(cosmosExpressionValue(doc, expression[5:len(expression)-1])); ok {
			return normalizeCosmosNumber(cosmosSign(value))
		}
		return nil
	case strings.HasPrefix(upper, "TRUNC(") && strings.HasSuffix(expression, ")"):
		if value, ok := cosmosNumericValue(cosmosExpressionValue(doc, expression[6:len(expression)-1])); ok {
			return normalizeCosmosNumber(math.Trunc(value))
		}
		return nil
	case strings.EqualFold(expression, "PI()"):
		return math.Pi
	case strings.HasPrefix(upper, "ARRAY_LENGTH(") && strings.HasSuffix(expression, ")"):
		args := splitCosmosExpressionList(expression[len("ARRAY_LENGTH(") : len(expression)-1])
		if len(args) != 1 {
			return cosmosUndefined
		}
		if values, ok := cosmosExpressionValue(doc, args[0]).([]any); ok {
			return len(values)
		}
		return cosmosUndefined
	case strings.HasPrefix(upper, "ARRAY_CONTAINS(") && strings.HasSuffix(expression, ")"):
		args := splitCosmosExpressionList(expression[15 : len(expression)-1])
		if len(args) < 2 {
			return cosmosUndefined
		}
		values, ok := cosmosExpressionValue(doc, args[0]).([]any)
		if !ok {
			return cosmosUndefined
		}
		needle := cosmosExpressionValue(doc, args[1])
		partial := false
		if len(args) > 2 {
			partial, ok = cosmosBooleanFunctionArgument(doc, args[2])
			if !ok {
				return cosmosUndefined
			}
		}
		return cosmosArrayContains(values, needle, partial)
	case strings.HasPrefix(upper, "ARRAY_CONTAINS_ALL(") && strings.HasSuffix(expression, ")"):
		return cosmosArrayContainsAllOrAny(doc, splitCosmosExpressionList(expression[len("ARRAY_CONTAINS_ALL("):len(expression)-1]), true)
	case strings.HasPrefix(upper, "ARRAY_CONTAINS_ANY(") && strings.HasSuffix(expression, ")"):
		return cosmosArrayContainsAllOrAny(doc, splitCosmosExpressionList(expression[len("ARRAY_CONTAINS_ANY("):len(expression)-1]), false)
	case strings.HasPrefix(upper, "OBJECTTOARRAY(") && strings.HasSuffix(expression, ")"):
		return cosmosObjectToArray(doc, splitCosmosExpressionList(expression[len("OBJECTTOARRAY("):len(expression)-1]))
	case strings.HasPrefix(upper, "SETINTERSECT(") && strings.HasSuffix(expression, ")"):
		return cosmosSetIntersect(doc, splitCosmosExpressionList(expression[len("SETINTERSECT("):len(expression)-1]))
	case strings.HasPrefix(upper, "SETUNION(") && strings.HasSuffix(expression, ")"):
		return cosmosSetUnion(doc, splitCosmosExpressionList(expression[len("SETUNION("):len(expression)-1]))
	case strings.HasPrefix(upper, "ARRAY_SLICE(") && strings.HasSuffix(expression, ")"):
		args := splitCosmosExpressionList(expression[12 : len(expression)-1])
		if len(args) < 2 {
			return cosmosUndefined
		}
		values, ok := cosmosExpressionValue(doc, args[0]).([]any)
		if !ok {
			return cosmosUndefined
		}
		start, ok := cosmosNumericFunctionArgument(doc, args[1])
		if !ok {
			return cosmosUndefined
		}
		length := len(values)
		if len(args) > 2 {
			lengthNumber, ok := cosmosNumericFunctionArgument(doc, args[2])
			if !ok {
				return cosmosUndefined
			}
			length = int(lengthNumber)
		}
		return cosmosArraySlice(values, int(start), length)
	case strings.HasPrefix(upper, "ARRAY_CONCAT(") && strings.HasSuffix(expression, ")"):
		args := splitCosmosExpressionList(expression[13 : len(expression)-1])
		if len(args) < 2 {
			return cosmosUndefined
		}
		out := make([]any, 0)
		for _, arg := range args {
			values, ok := cosmosExpressionValue(doc, arg).([]any)
			if !ok {
				return cosmosUndefined
			}
			out = append(out, values...)
		}
		return out
	default:
		if strings.HasPrefix(expression, "c.") {
			return cosmosFieldValue(doc, expression)
		}
		if value, ok := cosmosPathValue(doc, expression); ok {
			return value
		}
		return cosmosQueryValue(expression, nil)
	}
}

func cosmosExpressionValueWithParams(doc map[string]any, expression string, params []cosmosSQLParam) any {
	expression = strings.TrimSpace(expression)
	if strings.HasPrefix(expression, "@") {
		return cosmosQueryValue(expression, params)
	}
	return cosmosExpressionValue(doc, expression)
}

func cosmosObjectExpressionValue(doc map[string]any, expression string) map[string]any {
	body := strings.TrimSpace(expression[1 : len(expression)-1])
	out := make(map[string]any)
	if body == "" {
		return out
	}
	for _, part := range splitCosmosExpressionList(body) {
		colon := cosmosTopLevelColon(part)
		if colon < 0 {
			continue
		}
		key := cosmosObjectExpressionKey(part[:colon])
		if key == "" {
			continue
		}
		value := cosmosExpressionValue(doc, part[colon+1:])
		if cosmosIsUndefined(value) {
			continue
		}
		out[key] = value
	}
	return out
}

func cosmosArrayExpressionValue(doc map[string]any, expression string) []any {
	body := strings.TrimSpace(expression[1 : len(expression)-1])
	if body == "" {
		return []any{}
	}
	parts := splitCosmosExpressionList(body)
	out := make([]any, 0, len(parts))
	for _, part := range parts {
		value := cosmosExpressionValue(doc, part)
		if cosmosIsUndefined(value) {
			continue
		}
		out = append(out, value)
	}
	return out
}

func cosmosObjectExpressionKey(expression string) string {
	value := cosmosQueryValue(expression, nil)
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func cosmosParenthesizedExpressionBody(expression, keyword string) (string, bool) {
	expression = strings.TrimSpace(expression)
	if !strings.HasPrefix(strings.ToUpper(expression), strings.ToUpper(keyword)) {
		return "", false
	}
	body := strings.TrimSpace(expression[len(keyword):])
	if !strings.HasPrefix(body, "(") || cosmosMatchingCloseParen(body, 0) != len(body)-1 {
		return "", false
	}
	return strings.TrimSpace(body[1 : len(body)-1]), true
}

func cosmosSubqueryHasRows(doc map[string]any, subquery string) bool {
	return len(cosmosSubqueryValues(doc, subquery)) > 0
}

func cosmosScalarSubqueryValue(doc map[string]any, subquery string) any {
	if findCosmosTopLevelKeyword(subquery, "FROM") < 0 {
		return cosmosSelectOnlySubqueryValue(doc, subquery)
	}
	if cosmosSelectValueCount(subquery) {
		return len(cosmosSubqueryRows(doc, subquery))
	}
	if agg, field, ok := parseCosmosAggregate(subquery); ok {
		return cosmosAggregate(agg, field, cosmosSubqueryRows(doc, subquery))
	}
	values := cosmosSubqueryValues(doc, subquery)
	if len(values) == 0 {
		return nil
	}
	return values[0]
}

func cosmosSelectOnlySubqueryValue(doc map[string]any, subquery string) any {
	subquery = strings.Join(strings.Fields(subquery), " ")
	whereClause := cosmosSQLClause(subquery, "WHERE", []string{"GROUP BY", "ORDER BY", "OFFSET", "LIMIT"})
	if !cosmosSQLMatches(doc, whereClause, nil) {
		return nil
	}
	fields, selectValue, _ := parseCosmosSelectFields(subquery)
	if len(fields) == 0 {
		return nil
	}
	if selectValue && len(fields) == 1 {
		return cosmosExpressionValue(doc, fields[0])
	}
	projection := make(map[string]any, len(fields))
	for i, field := range fields {
		value := cosmosExpressionValue(doc, field)
		if cosmosIsUndefined(value) {
			continue
		}
		projection[cosmosProjectionAlias(field, i+1)] = value
	}
	return projection
}

func cosmosSubqueryValues(doc map[string]any, subquery string) []any {
	rows, fields, selectValue := cosmosSubqueryRowsAndSelect(doc, subquery)
	out := make([]any, 0, len(rows))
	for _, row := range rows {
		if selectValue && len(fields) == 1 {
			projected, exists := cosmosExpressionDefined(row, fields[0])
			if !exists {
				continue
			}
			out = append(out, projected)
			continue
		}
		projection := make(map[string]any, len(fields))
		for i, field := range fields {
			value := cosmosExpressionValue(row, field)
			if cosmosIsUndefined(value) {
				continue
			}
			projection[cosmosProjectionAlias(field, i+1)] = value
		}
		out = append(out, projection)
	}
	return out
}

func cosmosSubqueryRows(doc map[string]any, subquery string) []map[string]any {
	rows, _, _ := cosmosSubqueryRowsAndSelect(doc, subquery)
	return rows
}

func cosmosSubqueryRowsAndSelect(doc map[string]any, subquery string) ([]map[string]any, []string, bool) {
	subquery = strings.Join(strings.Fields(subquery), " ")
	fromClause := cosmosSQLClause(subquery, "FROM", []string{"WHERE", "GROUP BY", "ORDER BY", "OFFSET", "LIMIT"})
	alias, path, ok := parseCosmosArraySubquerySource(fromClause)
	if !ok {
		return nil, nil, false
	}
	values, ok := cosmosExpressionValue(doc, path).([]any)
	if !ok {
		return nil, nil, false
	}
	whereClause := cosmosSQLClause(subquery, "WHERE", []string{"GROUP BY", "ORDER BY", "OFFSET", "LIMIT"})
	fields, selectValue, _ := parseCosmosSelectFields(subquery)
	rows := make([]map[string]any, 0, len(values))
	for _, value := range values {
		row := cloneMap(doc)
		row[alias] = value
		if !cosmosSQLMatches(row, whereClause, nil) {
			continue
		}
		rows = append(rows, row)
	}
	return rows, fields, selectValue
}

func parseCosmosArraySubquerySource(fromClause string) (string, string, bool) {
	fields := strings.Fields(fromClause)
	if len(fields) != 3 || !strings.EqualFold(fields[1], "IN") {
		return "", "", false
	}
	return fields[0], fields[2], true
}

func cosmosExpressionDefined(doc map[string]any, expression string) (any, bool) {
	expression = strings.TrimSpace(expression)
	if value, ok := cosmosPathValue(doc, expression); ok {
		return value, true
	}
	if strings.HasPrefix(expression, "c.") || strings.Contains(expression, "[") {
		return nil, false
	}
	return cosmosExpressionValue(doc, expression), true
}

func cosmosBooleanExpression(doc map[string]any, expression string) bool {
	if value, ok := cosmosExpressionValue(doc, expression).(bool); ok {
		return value
	}
	return cosmosSQLPredicateMatches(doc, expression, nil)
}

func cosmosNumericValue(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case gojson.Number:
		out, err := strconv.ParseFloat(string(typed), 64)
		return out, err == nil
	default:
		return 0, false
	}
}

func cosmosArithmeticExpressionValue(doc map[string]any, expression string) (any, bool) {
	if idx, op := cosmosTopLevelArithmetic(expression, "+-"); idx >= 0 {
		return cosmosEvaluateArithmetic(doc, expression[:idx], expression[idx+len(op):], op), true
	}
	if idx, op := cosmosTopLevelArithmetic(expression, "*/%"); idx >= 0 {
		return cosmosEvaluateArithmetic(doc, expression[:idx], expression[idx+len(op):], op), true
	}
	return nil, false
}

func cosmosEvaluateArithmetic(doc map[string]any, leftExpression, rightExpression, op string) any {
	left, leftOK := cosmosNumericValue(cosmosExpressionValue(doc, leftExpression))
	right, rightOK := cosmosNumericValue(cosmosExpressionValue(doc, rightExpression))
	if !leftOK || !rightOK {
		return cosmosUndefined
	}
	switch op {
	case "+":
		return normalizeCosmosNumber(left + right)
	case "-":
		return normalizeCosmosNumber(left - right)
	case "*":
		return normalizeCosmosNumber(left * right)
	case "/":
		if right == 0 {
			return cosmosUndefined
		}
		return normalizeCosmosNumber(left / right)
	case "%":
		if right == 0 {
			return cosmosUndefined
		}
		return normalizeCosmosNumber(math.Mod(left, right))
	default:
		return cosmosUndefined
	}
}

func cosmosUnaryMathFunction(doc map[string]any, expression string, fn func(float64) float64) any {
	value, ok := cosmosNumericValue(cosmosExpressionValue(doc, expression))
	if !ok {
		return cosmosUndefined
	}
	return normalizeCosmosNumber(fn(value))
}

func cosmosBinaryMathFunction(doc map[string]any, args []string, fn func(float64, float64) float64) any {
	if len(args) < 2 {
		return cosmosUndefined
	}
	left, leftOK := cosmosNumericValue(cosmosExpressionValue(doc, args[0]))
	right, rightOK := cosmosNumericValue(cosmosExpressionValue(doc, args[1]))
	if !leftOK || !rightOK {
		return cosmosUndefined
	}
	return normalizeCosmosNumber(fn(left, right))
}

func cosmosIntegerValue(value any) (int64, bool) {
	const (
		minInt64Float = -9223372036854775808.0
		maxInt64Float = 9223372036854775807.0
	)
	number, ok := cosmosNumericValue(value)
	if !ok || number != math.Trunc(number) || number < minInt64Float || number > maxInt64Float {
		return 0, false
	}
	return int64(number), true
}

func cosmosBinaryIntegerFunction(doc map[string]any, args []string, fn func(int64, int64) (int64, bool)) any {
	if len(args) < 2 {
		return cosmosUndefined
	}
	left, leftOK := cosmosIntegerValue(cosmosExpressionValue(doc, args[0]))
	right, rightOK := cosmosIntegerValue(cosmosExpressionValue(doc, args[1]))
	if !leftOK || !rightOK {
		return cosmosUndefined
	}
	out, ok := fn(left, right)
	if !ok {
		return cosmosUndefined
	}
	return normalizeCosmosNumber(float64(out))
}

func cosmosNumberBin(doc map[string]any, args []string) any {
	if len(args) == 0 {
		return cosmosUndefined
	}
	value, ok := cosmosNumericValue(cosmosExpressionValue(doc, args[0]))
	if !ok {
		return cosmosUndefined
	}
	binSize := float64(1)
	if len(args) > 1 {
		if binSize, ok = cosmosNumericValue(cosmosExpressionValue(doc, args[1])); !ok {
			return cosmosUndefined
		}
	}
	if binSize == 0 {
		return cosmosUndefined
	}
	return normalizeCosmosNumber(math.Floor(value/binSize) * binSize)
}

func cosmosVectorDistance(doc map[string]any, args []string) any {
	if len(args) < 2 {
		return cosmosUndefined
	}
	left, ok := cosmosVectorNumbers(cosmosExpressionValue(doc, args[0]))
	if !ok {
		return cosmosUndefined
	}
	right, ok := cosmosVectorNumbers(cosmosExpressionValue(doc, args[1]))
	if !ok || len(left) == 0 || len(left) != len(right) {
		return cosmosUndefined
	}
	distanceFunction := cosmosVectorDistanceFunction(doc, args)
	switch distanceFunction {
	case "COSINE":
		dot, leftNorm, rightNorm := 0.0, 0.0, 0.0
		for i := range left {
			dot += left[i] * right[i]
			leftNorm += left[i] * left[i]
			rightNorm += right[i] * right[i]
		}
		if leftNorm == 0 || rightNorm == 0 {
			return cosmosUndefined
		}
		return normalizeCosmosNumber(1 - dot/(math.Sqrt(leftNorm)*math.Sqrt(rightNorm)))
	case "DOTPRODUCT":
		dot := 0.0
		for i := range left {
			dot += left[i] * right[i]
		}
		return normalizeCosmosNumber(dot)
	case "EUCLIDEAN":
	default:
		return cosmosUndefined
	}
	sum := 0.0
	for i := range left {
		delta := left[i] - right[i]
		sum += delta * delta
	}
	return normalizeCosmosNumber(math.Sqrt(sum))
}

func cosmosVectorDistanceFunction(doc map[string]any, args []string) string {
	if len(args) > 3 {
		if options, ok := cosmosExpressionValue(doc, args[3]).(map[string]any); ok {
			if raw, exists := options["distanceFunction"]; exists {
				return strings.ToUpper(fmt.Sprint(raw))
			}
		}
	}
	return "EUCLIDEAN"
}

func cosmosVectorNumbers(value any) ([]float64, bool) {
	values, ok := value.([]any)
	if !ok {
		return nil, false
	}
	out := make([]float64, 0, len(values))
	for _, value := range values {
		if _, nested := value.([]any); nested {
			return nil, false
		}
		number, ok := cosmosNumericValue(value)
		if !ok {
			return nil, false
		}
		out = append(out, number)
	}
	return out, true
}

func cosmosRRFScore(doc map[string]any, args []string) any {
	if len(args) < 2 {
		return cosmosUndefined
	}
	weights := make([]float64, 0)
	scoreArgs := args
	if parsedWeights, ok := cosmosRRFWeights(doc, args[len(args)-1]); ok {
		weights = parsedWeights
		scoreArgs = args[:len(args)-1]
	}
	total := 0.0
	matched := false
	for i, arg := range scoreArgs {
		score, ok := cosmosRRFComponentScore(doc, arg)
		if !ok {
			return cosmosUndefined
		}
		weight := 1.0
		if i < len(weights) {
			weight = weights[i]
		}
		total += weight * score
		matched = true
	}
	if !matched {
		return cosmosUndefined
	}
	return normalizeCosmosNumber(total)
}

func cosmosRRFWeights(doc map[string]any, expression string) ([]float64, bool) {
	values, ok := cosmosExpressionValue(doc, expression).([]any)
	if !ok {
		return nil, false
	}
	out := make([]float64, 0, len(values))
	for _, value := range values {
		weight, ok := cosmosNumericValue(value)
		if !ok {
			return nil, false
		}
		out = append(out, weight)
	}
	return out, true
}

func cosmosRRFComponentScore(doc map[string]any, expression string) (float64, bool) {
	expression = strings.TrimSpace(expression)
	upper := strings.ToUpper(expression)
	switch {
	case strings.HasPrefix(upper, "FULLTEXTSCORE(") && strings.HasSuffix(expression, ")"):
		score, ok := cosmosNumericValue(cosmosExpressionValue(doc, expression))
		return score, ok
	case strings.HasPrefix(upper, "VECTORDISTANCE(") && strings.HasSuffix(expression, ")"):
		args := splitCosmosExpressionList(expression[len("VECTORDISTANCE(") : len(expression)-1])
		score, ok := cosmosNumericValue(cosmosVectorDistance(doc, args))
		if !ok {
			return 0, false
		}
		if cosmosVectorDistanceFunction(doc, args) == "DOTPRODUCT" {
			return score, true
		}
		return 1 / (1 + math.Max(score, 0)), true
	default:
		score, ok := cosmosNumericValue(cosmosExpressionValue(doc, expression))
		return score, ok
	}
}

func cosmosSpatialDistance(doc map[string]any, args []string) any {
	if len(args) < 2 {
		return cosmosUndefined
	}
	distance, ok := cosmosSpatialDistanceBetween(cosmosExpressionValue(doc, args[0]), cosmosExpressionValue(doc, args[1]))
	if !ok {
		return cosmosUndefined
	}
	return normalizeCosmosNumber(distance)
}

func cosmosSpatialArea(doc map[string]any, args []string) any {
	if len(args) < 1 {
		return cosmosUndefined
	}
	shape := cosmosExpressionValue(doc, args[0])
	if geometry, ok := shape.(map[string]any); ok {
		switch strings.ToLower(fmt.Sprint(geometry["type"])) {
		case "point", "linestring":
			return float64(0)
		case "polygon":
			rings, ok := cosmosGeoJSONPolygonRings(shape)
			if !ok {
				return cosmosUndefined
			}
			return normalizeCosmosNumber(cosmosPolygonAreaMeters(rings))
		case "multipolygon":
			polygons, ok := cosmosGeoJSONMultiPolygonRings(shape)
			if !ok {
				return cosmosUndefined
			}
			area := float64(0)
			for _, polygon := range polygons {
				area += cosmosPolygonAreaMeters(polygon)
			}
			return normalizeCosmosNumber(area)
		}
	}
	return cosmosUndefined
}

func cosmosSpatialWithin(doc map[string]any, args []string) bool {
	if len(args) < 2 {
		return false
	}
	lon, lat, ok := cosmosGeoJSONPoint(cosmosExpressionValue(doc, args[0]))
	if !ok {
		return false
	}
	return cosmosPointWithinAnyPolygon(lon, lat, cosmosExpressionValue(doc, args[1]), false)
}

func cosmosSpatialIsValid(doc map[string]any, args []string) bool {
	if len(args) < 1 {
		return false
	}
	valid, _ := cosmosGeoJSONValidation(cosmosExpressionValue(doc, args[0]))
	return valid
}

func cosmosSpatialIsValidDetailed(doc map[string]any, args []string) map[string]any {
	if len(args) < 1 {
		return map[string]any{
			"valid":  false,
			"reason": "The GeoJSON value is invalid.",
		}
	}
	valid, reason := cosmosGeoJSONValidation(cosmosExpressionValue(doc, args[0]))
	result := map[string]any{"valid": valid}
	if !valid {
		result["reason"] = reason
	}
	return result
}

func cosmosSpatialIntersects(doc map[string]any, args []string) bool {
	if len(args) < 2 {
		return false
	}
	left := cosmosExpressionValue(doc, args[0])
	right := cosmosExpressionValue(doc, args[1])
	leftLon, leftLat, leftPoint := cosmosGeoJSONPoint(left)
	rightLon, rightLat, rightPoint := cosmosGeoJSONPoint(right)
	if leftPoint && rightPoint {
		return cosmosSamePoint(cosmosPoint{lon: leftLon, lat: leftLat}, cosmosPoint{lon: rightLon, lat: rightLat})
	}
	if leftPoint {
		return cosmosPointWithinAnyPolygon(leftLon, leftLat, right, true)
	}
	if rightPoint {
		return cosmosPointWithinAnyPolygon(rightLon, rightLat, left, true)
	}
	leftPolygons, leftOK := cosmosGeoJSONOuterPolygons(left)
	rightPolygons, rightOK := cosmosGeoJSONOuterPolygons(right)
	if !leftOK || !rightOK {
		return false
	}
	for _, leftPolygon := range leftPolygons {
		for _, rightPolygon := range rightPolygons {
			if cosmosPolygonsIntersect(leftPolygon, rightPolygon) {
				return true
			}
		}
	}
	return false
}

func cosmosGeoJSONPoint(value any) (float64, float64, bool) {
	shape, ok := value.(map[string]any)
	if !ok || !strings.EqualFold(fmt.Sprint(shape["type"]), "Point") {
		return 0, 0, false
	}
	coordinates, ok := shape["coordinates"].([]any)
	if !ok || len(coordinates) < 2 {
		return 0, 0, false
	}
	lon, lonOK := cosmosNumericValue(coordinates[0])
	lat, latOK := cosmosNumericValue(coordinates[1])
	if !lonOK || !latOK {
		return 0, 0, false
	}
	return lon, lat, true
}

func cosmosSpatialDistanceBetween(left, right any) (float64, bool) {
	if leftLon, leftLat, ok := cosmosGeoJSONPoint(left); ok {
		return cosmosPointDistanceToShapeMeters(cosmosPoint{lon: leftLon, lat: leftLat}, right)
	}
	if rightLon, rightLat, ok := cosmosGeoJSONPoint(right); ok {
		return cosmosPointDistanceToShapeMeters(cosmosPoint{lon: rightLon, lat: rightLat}, left)
	}
	leftLine, leftLineOK := cosmosGeoJSONLineString(left)
	rightLine, rightLineOK := cosmosGeoJSONLineString(right)
	if leftLineOK && rightLineOK {
		return cosmosLineStringDistanceMeters(leftLine, rightLine), true
	}
	leftPolygons, leftPolygonOK := cosmosGeoJSONPolygonSet(left)
	rightPolygons, rightPolygonOK := cosmosGeoJSONPolygonSet(right)
	switch {
	case leftLineOK && rightPolygonOK:
		return cosmosLineStringDistanceToPolygonsMeters(leftLine, rightPolygons), true
	case rightLineOK && leftPolygonOK:
		return cosmosLineStringDistanceToPolygonsMeters(rightLine, leftPolygons), true
	case leftPolygonOK && rightPolygonOK:
		return cosmosPolygonSetsDistanceMeters(leftPolygons, rightPolygons), true
	default:
		return 0, false
	}
}

func cosmosPointDistanceToShapeMeters(point cosmosPoint, value any) (float64, bool) {
	if lon, lat, ok := cosmosGeoJSONPoint(value); ok {
		return cosmosHaversineMeters(point.lon, point.lat, lon, lat), true
	}
	if line, ok := cosmosGeoJSONLineString(value); ok {
		return cosmosPointLineStringDistanceMeters(point, line), true
	}
	if polygons, ok := cosmosGeoJSONPolygonSet(value); ok {
		return cosmosPointPolygonSetDistanceMeters(point, polygons), true
	}
	return 0, false
}

func cosmosGeoJSONValidation(value any) (bool, string) {
	shape, ok := value.(map[string]any)
	if !ok {
		return false, "The GeoJSON value is invalid."
	}
	switch strings.ToLower(fmt.Sprint(shape["type"])) {
	case "point":
		_, ok, reason := cosmosGeoJSONCoordinate(shape["coordinates"])
		return ok, reason
	case "linestring":
		coordinates, ok := shape["coordinates"].([]any)
		if !ok || len(coordinates) < 2 {
			return false, "LineString coordinates must contain at least two positions."
		}
		return cosmosGeoJSONCoordinatesValid(coordinates)
	case "polygon":
		return cosmosGeoJSONPolygonValid(shape["coordinates"])
	case "multipolygon":
		polygons, ok := shape["coordinates"].([]any)
		if !ok || len(polygons) == 0 {
			return false, "MultiPolygon coordinates must contain at least one polygon."
		}
		for _, polygon := range polygons {
			valid, reason := cosmosGeoJSONPolygonValid(polygon)
			if !valid {
				return false, reason
			}
		}
		return true, ""
	default:
		return false, "GeoJSON type must be Point, LineString, Polygon, or MultiPolygon."
	}
}

func cosmosGeoJSONPolygonValid(value any) (bool, string) {
	rings, ok := value.([]any)
	if !ok || len(rings) == 0 {
		return false, "Polygon coordinates must contain at least one linear ring."
	}
	for _, rawRing := range rings {
		ring, ok := rawRing.([]any)
		if !ok || len(ring) < 4 {
			return false, "Polygon linear rings must contain at least four positions."
		}
		points := make([]cosmosPoint, 0, len(ring))
		for _, rawPoint := range ring {
			point, ok, reason := cosmosGeoJSONCoordinate(rawPoint)
			if !ok {
				return false, reason
			}
			points = append(points, point)
		}
		if !cosmosSamePoint(points[0], points[len(points)-1]) {
			return false, "Polygon linear rings must close with the same first and last position."
		}
		if !cosmosPolygonRingIsSimple(points) {
			return false, "Polygon linear rings must not self-intersect."
		}
	}
	return true, ""
}

func cosmosGeoJSONCoordinatesValid(values []any) (bool, string) {
	for _, raw := range values {
		if _, ok, reason := cosmosGeoJSONCoordinate(raw); !ok {
			return false, reason
		}
	}
	return true, ""
}

func cosmosGeoJSONCoordinate(value any) (cosmosPoint, bool, string) {
	coordinates, ok := value.([]any)
	if !ok || len(coordinates) < 2 {
		return cosmosPoint{}, false, "GeoJSON positions must contain longitude and latitude values."
	}
	lon, lonOK := cosmosNumericValue(coordinates[0])
	lat, latOK := cosmosNumericValue(coordinates[1])
	if !lonOK || !latOK {
		return cosmosPoint{}, false, "GeoJSON positions must contain numeric longitude and latitude values."
	}
	if lon < -180 || lon > 180 {
		return cosmosPoint{}, false, "Longitude values must be between -180 and 180 degrees."
	}
	if lat < -90 || lat > 90 {
		return cosmosPoint{}, false, "Latitude values must be between -90 and 90 degrees."
	}
	return cosmosPoint{lon: lon, lat: lat}, true, ""
}

type cosmosPoint struct {
	lon float64
	lat float64
}

func cosmosGeoJSONPolygon(value any) ([]cosmosPoint, bool) {
	rings, ok := cosmosGeoJSONPolygonRings(value)
	if !ok || len(rings) == 0 {
		return nil, false
	}
	return rings[0], true
}

func cosmosGeoJSONOuterPolygons(value any) ([][]cosmosPoint, bool) {
	if polygon, ok := cosmosGeoJSONPolygon(value); ok {
		return [][]cosmosPoint{polygon}, true
	}
	multiPolygon, ok := cosmosGeoJSONMultiPolygonRings(value)
	if !ok {
		return nil, false
	}
	out := make([][]cosmosPoint, 0, len(multiPolygon))
	for _, rings := range multiPolygon {
		if len(rings) == 0 {
			return nil, false
		}
		out = append(out, rings[0])
	}
	return out, true
}

func cosmosGeoJSONPolygonSet(value any) ([][][]cosmosPoint, bool) {
	if rings, ok := cosmosGeoJSONPolygonRings(value); ok {
		return [][][]cosmosPoint{rings}, true
	}
	return cosmosGeoJSONMultiPolygonRings(value)
}

func cosmosPointWithinAnyPolygon(lon, lat float64, value any, includeBoundary bool) bool {
	polygons, ok := cosmosGeoJSONOuterPolygons(value)
	if !ok {
		return false
	}
	for _, polygon := range polygons {
		if includeBoundary && cosmosPointIntersectsPolygon(lon, lat, polygon) || !includeBoundary && cosmosPointInPolygon(lon, lat, polygon) {
			return true
		}
	}
	return false
}

func cosmosGeoJSONPolygonRings(value any) ([][]cosmosPoint, bool) {
	shape, ok := value.(map[string]any)
	if !ok || !strings.EqualFold(fmt.Sprint(shape["type"]), "Polygon") {
		return nil, false
	}
	return cosmosGeoJSONPolygonCoordinateRings(shape["coordinates"])
}

func cosmosGeoJSONMultiPolygonRings(value any) ([][][]cosmosPoint, bool) {
	shape, ok := value.(map[string]any)
	if !ok || !strings.EqualFold(fmt.Sprint(shape["type"]), "MultiPolygon") {
		return nil, false
	}
	polygonValues, ok := shape["coordinates"].([]any)
	if !ok || len(polygonValues) == 0 {
		return nil, false
	}
	polygons := make([][][]cosmosPoint, 0, len(polygonValues))
	for _, rawPolygon := range polygonValues {
		rings, ok := cosmosGeoJSONPolygonCoordinateRings(rawPolygon)
		if !ok {
			return nil, false
		}
		polygons = append(polygons, rings)
	}
	return polygons, true
}

func cosmosGeoJSONLineString(value any) ([]cosmosPoint, bool) {
	shape, ok := value.(map[string]any)
	if !ok || !strings.EqualFold(fmt.Sprint(shape["type"]), "LineString") {
		return nil, false
	}
	coordinateValues, ok := shape["coordinates"].([]any)
	if !ok || len(coordinateValues) < 2 {
		return nil, false
	}
	points := make([]cosmosPoint, 0, len(coordinateValues))
	for _, raw := range coordinateValues {
		coordinate, ok := raw.([]any)
		if !ok || len(coordinate) < 2 {
			return nil, false
		}
		lon, lonOK := cosmosNumericValue(coordinate[0])
		lat, latOK := cosmosNumericValue(coordinate[1])
		if !lonOK || !latOK {
			return nil, false
		}
		points = append(points, cosmosPoint{lon: lon, lat: lat})
	}
	return points, true
}

func cosmosGeoJSONPolygonCoordinateRings(value any) ([][]cosmosPoint, bool) {
	rings, ok := value.([]any)
	if !ok || len(rings) == 0 {
		return nil, false
	}
	out := make([][]cosmosPoint, 0, len(rings))
	for _, rawRing := range rings {
		ring, ok := rawRing.([]any)
		if !ok || len(ring) < 4 {
			return nil, false
		}
		points := make([]cosmosPoint, 0, len(ring))
		for _, raw := range ring {
			coordinate, ok := raw.([]any)
			if !ok || len(coordinate) < 2 {
				return nil, false
			}
			lon, lonOK := cosmosNumericValue(coordinate[0])
			lat, latOK := cosmosNumericValue(coordinate[1])
			if !lonOK || !latOK {
				return nil, false
			}
			points = append(points, cosmosPoint{lon: lon, lat: lat})
		}
		out = append(out, points)
	}
	return out, true
}

func cosmosPointIntersectsPolygon(lon, lat float64, polygon []cosmosPoint) bool {
	return cosmosPointInPolygon(lon, lat, polygon) || cosmosPointOnPolygonBoundary(cosmosPoint{lon: lon, lat: lat}, polygon)
}

func cosmosPointInPolygonRings(point cosmosPoint, rings [][]cosmosPoint) bool {
	if len(rings) == 0 || !cosmosPointIntersectsPolygon(point.lon, point.lat, rings[0]) {
		return false
	}
	for _, hole := range rings[1:] {
		if cosmosPointOnPolygonBoundary(point, hole) {
			return true
		}
		if cosmosPointInPolygon(point.lon, point.lat, hole) {
			return false
		}
	}
	return true
}

func cosmosPointPolygonSetDistanceMeters(point cosmosPoint, polygons [][][]cosmosPoint) float64 {
	distance := math.Inf(1)
	for _, rings := range polygons {
		if cosmosPointInPolygonRings(point, rings) {
			return 0
		}
		for _, ring := range rings {
			distance = math.Min(distance, cosmosPointLineStringDistanceMeters(point, ring))
		}
	}
	if math.IsInf(distance, 1) {
		return 0
	}
	return distance
}

func cosmosLineStringDistanceToPolygonsMeters(line []cosmosPoint, polygons [][][]cosmosPoint) float64 {
	distance := math.Inf(1)
	for _, rings := range polygons {
		for _, point := range line {
			if cosmosPointInPolygonRings(point, rings) {
				return 0
			}
		}
		for _, ring := range rings {
			if cosmosLineStringsIntersect(line, ring) {
				return 0
			}
			distance = math.Min(distance, cosmosLineStringDistanceMeters(line, ring))
		}
	}
	if math.IsInf(distance, 1) {
		return 0
	}
	return distance
}

func cosmosPolygonSetsDistanceMeters(left, right [][][]cosmosPoint) float64 {
	distance := math.Inf(1)
	for _, leftRings := range left {
		for _, rightRings := range right {
			if len(leftRings) > 0 && len(leftRings[0]) > 0 && cosmosPointInPolygonRings(leftRings[0][0], rightRings) {
				return 0
			}
			if len(rightRings) > 0 && len(rightRings[0]) > 0 && cosmosPointInPolygonRings(rightRings[0][0], leftRings) {
				return 0
			}
			for _, leftRing := range leftRings {
				for _, rightRing := range rightRings {
					if cosmosLineStringsIntersect(leftRing, rightRing) {
						return 0
					}
					distance = math.Min(distance, cosmosLineStringDistanceMeters(leftRing, rightRing))
				}
			}
		}
	}
	if math.IsInf(distance, 1) {
		return 0
	}
	return distance
}

func cosmosPolygonsIntersect(left, right []cosmosPoint) bool {
	for i := range left {
		leftStart, leftEnd, ok := cosmosPolygonSegment(left, i)
		if !ok {
			continue
		}
		for j := range right {
			rightStart, rightEnd, ok := cosmosPolygonSegment(right, j)
			if !ok {
				continue
			}
			if cosmosSegmentsIntersect(leftStart, leftEnd, rightStart, rightEnd) {
				return true
			}
		}
	}
	return len(left) > 0 && cosmosPointIntersectsPolygon(left[0].lon, left[0].lat, right) ||
		len(right) > 0 && cosmosPointIntersectsPolygon(right[0].lon, right[0].lat, left)
}

func cosmosPolygonSegment(polygon []cosmosPoint, index int) (cosmosPoint, cosmosPoint, bool) {
	if len(polygon) < 2 {
		return cosmosPoint{}, cosmosPoint{}, false
	}
	start := polygon[index]
	end := polygon[(index+1)%len(polygon)]
	if cosmosSamePoint(start, end) {
		return cosmosPoint{}, cosmosPoint{}, false
	}
	return start, end, true
}

func cosmosSegmentsIntersect(a, b, c, d cosmosPoint) bool {
	const epsilon = 1e-9
	if cosmosPointOnSegment(c, a, b) || cosmosPointOnSegment(d, a, b) ||
		cosmosPointOnSegment(a, c, d) || cosmosPointOnSegment(b, c, d) {
		return true
	}
	leftC := cosmosOrientation(a, b, c)
	leftD := cosmosOrientation(a, b, d)
	rightA := cosmosOrientation(c, d, a)
	rightB := cosmosOrientation(c, d, b)
	return (leftC > epsilon && leftD < -epsilon || leftC < -epsilon && leftD > epsilon) &&
		(rightA > epsilon && rightB < -epsilon || rightA < -epsilon && rightB > epsilon)
}

func cosmosPointOnPolygonBoundary(point cosmosPoint, polygon []cosmosPoint) bool {
	for i := range polygon {
		start, end, ok := cosmosPolygonSegment(polygon, i)
		if ok && cosmosPointOnSegment(point, start, end) {
			return true
		}
	}
	return false
}

func cosmosPolygonRingIsSimple(polygon []cosmosPoint) bool {
	for i := 0; i < len(polygon)-1; i++ {
		startA, endA, ok := cosmosPolygonSegment(polygon, i)
		if !ok {
			continue
		}
		for j := i + 1; j < len(polygon)-1; j++ {
			if j == i+1 || i == 0 && j == len(polygon)-2 {
				continue
			}
			startB, endB, ok := cosmosPolygonSegment(polygon, j)
			if ok && cosmosSegmentsIntersect(startA, endA, startB, endB) {
				return false
			}
		}
	}
	return true
}

func cosmosPointOnSegment(point, start, end cosmosPoint) bool {
	const epsilon = 1e-9
	if math.Abs(cosmosOrientation(start, end, point)) > epsilon {
		return false
	}
	return point.lon >= math.Min(start.lon, end.lon)-epsilon &&
		point.lon <= math.Max(start.lon, end.lon)+epsilon &&
		point.lat >= math.Min(start.lat, end.lat)-epsilon &&
		point.lat <= math.Max(start.lat, end.lat)+epsilon
}

func cosmosOrientation(a, b, c cosmosPoint) float64 {
	return (b.lon-a.lon)*(c.lat-a.lat) - (b.lat-a.lat)*(c.lon-a.lon)
}

func cosmosSamePoint(left, right cosmosPoint) bool {
	const epsilon = 1e-9
	return math.Abs(left.lon-right.lon) <= epsilon && math.Abs(left.lat-right.lat) <= epsilon
}

func cosmosPointLineStringDistanceMeters(point cosmosPoint, line []cosmosPoint) float64 {
	if len(line) == 0 {
		return 0
	}
	distance := math.Inf(1)
	for i := 0; i < len(line)-1; i++ {
		start, end := line[i], line[i+1]
		if cosmosSamePoint(start, end) {
			distance = math.Min(distance, cosmosHaversineMeters(point.lon, point.lat, start.lon, start.lat))
			continue
		}
		distance = math.Min(distance, cosmosPointSegmentDistanceMeters(point, start, end))
	}
	if math.IsInf(distance, 1) {
		return cosmosHaversineMeters(point.lon, point.lat, line[0].lon, line[0].lat)
	}
	return distance
}

func cosmosLineStringDistanceMeters(left, right []cosmosPoint) float64 {
	if len(left) == 0 || len(right) == 0 {
		return 0
	}
	distance := math.Inf(1)
	for i := 0; i < len(left)-1; i++ {
		leftStart, leftEnd := left[i], left[i+1]
		for j := 0; j < len(right)-1; j++ {
			rightStart, rightEnd := right[j], right[j+1]
			if cosmosSegmentsIntersect(leftStart, leftEnd, rightStart, rightEnd) {
				return 0
			}
			distance = math.Min(distance, cosmosSegmentDistanceMeters(leftStart, leftEnd, rightStart, rightEnd))
		}
	}
	if math.IsInf(distance, 1) {
		return cosmosHaversineMeters(left[0].lon, left[0].lat, right[0].lon, right[0].lat)
	}
	return distance
}

func cosmosLineStringsIntersect(left, right []cosmosPoint) bool {
	for i := 0; i < len(left)-1; i++ {
		leftStart, leftEnd := left[i], left[i+1]
		for j := 0; j < len(right)-1; j++ {
			if cosmosSegmentsIntersect(leftStart, leftEnd, right[j], right[j+1]) {
				return true
			}
		}
	}
	return false
}

func cosmosSegmentDistanceMeters(a, b, c, d cosmosPoint) float64 {
	return math.Min(
		math.Min(cosmosPointSegmentDistanceMeters(a, c, d), cosmosPointSegmentDistanceMeters(b, c, d)),
		math.Min(cosmosPointSegmentDistanceMeters(c, a, b), cosmosPointSegmentDistanceMeters(d, a, b)),
	)
}

func cosmosPointSegmentDistanceMeters(point, start, end cosmosPoint) float64 {
	const earthRadiusMeters = 6371008.8
	const degreesToRadians = math.Pi / 180
	referenceLatitude := (point.lat + start.lat + end.lat) / 3 * degreesToRadians
	cosLatitude := math.Cos(referenceLatitude)
	px, py := point.lon*degreesToRadians*earthRadiusMeters*cosLatitude, point.lat*degreesToRadians*earthRadiusMeters
	sx, sy := start.lon*degreesToRadians*earthRadiusMeters*cosLatitude, start.lat*degreesToRadians*earthRadiusMeters
	ex, ey := end.lon*degreesToRadians*earthRadiusMeters*cosLatitude, end.lat*degreesToRadians*earthRadiusMeters
	dx, dy := ex-sx, ey-sy
	lengthSquared := dx*dx + dy*dy
	if lengthSquared == 0 {
		return math.Hypot(px-sx, py-sy)
	}
	t := ((px-sx)*dx + (py-sy)*dy) / lengthSquared
	t = math.Max(0, math.Min(1, t))
	closestX, closestY := sx+t*dx, sy+t*dy
	return math.Hypot(px-closestX, py-closestY)
}

func cosmosPolygonAreaMeters(rings [][]cosmosPoint) float64 {
	if len(rings) == 0 {
		return 0
	}
	area := cosmosRingAreaMeters(rings[0])
	for _, ring := range rings[1:] {
		area -= cosmosRingAreaMeters(ring)
	}
	return math.Abs(area)
}

func cosmosRingAreaMeters(ring []cosmosPoint) float64 {
	if len(ring) < 4 {
		return 0
	}
	const wgs84SemiMajorMeters = 6378137.0
	const wgs84Flattening = 1 / 298.257223563
	eccentricitySquared := wgs84Flattening * (2 - wgs84Flattening)
	eccentricity := math.Sqrt(eccentricitySquared)
	sum := float64(0)
	for i := 0; i < len(ring); i++ {
		current := ring[i]
		next := ring[(i+1)%len(ring)]
		lonDelta := (next.lon - current.lon) * math.Pi / 180
		if lonDelta > math.Pi {
			lonDelta -= 2 * math.Pi
		} else if lonDelta < -math.Pi {
			lonDelta += 2 * math.Pi
		}
		currentTerm := cosmosEllipsoidAreaTerm(current.lat*math.Pi/180, eccentricity, eccentricitySquared)
		nextTerm := cosmosEllipsoidAreaTerm(next.lat*math.Pi/180, eccentricity, eccentricitySquared)
		sum += lonDelta * (currentTerm + nextTerm)
	}
	return math.Abs(sum) * wgs84SemiMajorMeters * wgs84SemiMajorMeters * (1 - eccentricitySquared) / 4
}

func cosmosEllipsoidAreaTerm(latitudeRadians, eccentricity, eccentricitySquared float64) float64 {
	sinLat := math.Sin(latitudeRadians)
	return sinLat/(1-eccentricitySquared*sinLat*sinLat) + math.Atanh(eccentricity*sinLat)/eccentricity
}

func cosmosPointInPolygon(lon, lat float64, polygon []cosmosPoint) bool {
	inside := false
	for i, j := 0, len(polygon)-1; i < len(polygon); j, i = i, i+1 {
		xi, yi := polygon[i].lon, polygon[i].lat
		xj, yj := polygon[j].lon, polygon[j].lat
		if (yi > lat) != (yj > lat) {
			xIntersection := (xj-xi)*(lat-yi)/(yj-yi) + xi
			if lon < xIntersection {
				inside = !inside
			}
		}
	}
	return inside
}

func cosmosHaversineMeters(leftLon, leftLat, rightLon, rightLat float64) float64 {
	const earthRadiusMeters = 6371008.8
	leftLatRad := leftLat * math.Pi / 180
	rightLatRad := rightLat * math.Pi / 180
	deltaLat := (rightLat - leftLat) * math.Pi / 180
	deltaLon := (rightLon - leftLon) * math.Pi / 180
	sinLat := math.Sin(deltaLat / 2)
	sinLon := math.Sin(deltaLon / 2)
	a := sinLat*sinLat + math.Cos(leftLatRad)*math.Cos(rightLatRad)*sinLon*sinLon
	return earthRadiusMeters * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

func cosmosPrimitiveValue(value any) bool {
	if value == nil {
		return true
	}
	if _, ok := value.(string); ok {
		return true
	}
	if _, ok := value.(bool); ok {
		return true
	}
	_, ok := cosmosNumericValue(value)
	return ok
}

func cosmosSign(value float64) float64 {
	switch {
	case value < 0:
		return -1
	case value > 0:
		return 1
	default:
		return 0
	}
}

func cosmosStringPredicate(doc map[string]any, args []string, match func(string, string) bool) any {
	left, ok := cosmosStringFunctionArgument(doc, args[0])
	if !ok {
		return cosmosUndefined
	}
	right, ok := cosmosStringFunctionArgument(doc, args[1])
	if !ok {
		return cosmosUndefined
	}
	ignoreCase := false
	if len(args) > 2 {
		ignoreCase, ok = cosmosBooleanFunctionArgument(doc, args[2])
		if !ok {
			return cosmosUndefined
		}
	}
	if ignoreCase {
		left = strings.ToLower(left)
		right = strings.ToLower(right)
	}
	return match(left, right)
}

func cosmosStringEqualsPredicate(doc map[string]any, args []string) any {
	left, ok := cosmosStringFunctionArgument(doc, args[0])
	if !ok {
		return cosmosUndefined
	}
	right, ok := cosmosStringFunctionArgument(doc, args[1])
	if !ok {
		return cosmosUndefined
	}
	ignoreCase := false
	if len(args) > 2 {
		ignoreCase, ok = cosmosBooleanFunctionArgument(doc, args[2])
		if !ok {
			return cosmosUndefined
		}
	}
	if ignoreCase {
		return strings.EqualFold(left, right)
	}
	return left == right
}

func cosmosStringFunctionArgument(doc map[string]any, expression string) (string, bool) {
	value := cosmosExpressionValue(doc, expression)
	if cosmosIsUndefined(value) {
		return "", false
	}
	stringValue, ok := value.(string)
	return stringValue, ok
}

func cosmosNumericFunctionArgument(doc map[string]any, expression string) (float64, bool) {
	value := cosmosExpressionValue(doc, expression)
	if cosmosIsUndefined(value) {
		return 0, false
	}
	number, ok := cosmosNumericValue(value)
	return number, ok && !math.IsNaN(number) && !math.IsInf(number, 0)
}

func cosmosBooleanFunctionArgument(doc map[string]any, expression string) (bool, bool) {
	value := cosmosExpressionValue(doc, expression)
	if cosmosIsUndefined(value) {
		return false, false
	}
	boolValue, ok := value.(bool)
	return boolValue, ok
}

func cosmosFullTextContains(doc map[string]any, args []string) bool {
	if len(args) < 2 {
		return false
	}
	value, ok := cosmosExpressionValue(doc, args[0]).(string)
	if !ok {
		return false
	}
	term := fmt.Sprint(cosmosExpressionValue(doc, args[1]))
	return cosmosFullTextContainsTerm(value, term)
}

func cosmosFullTextContainsAllOrAny(doc map[string]any, args []string, requireAll bool) bool {
	if len(args) < 2 {
		return false
	}
	value, ok := cosmosExpressionValue(doc, args[0]).(string)
	if !ok {
		return false
	}
	matchedAny := false
	for _, arg := range args[1:] {
		term := fmt.Sprint(cosmosExpressionValue(doc, arg))
		matched := cosmosFullTextContainsTerm(value, term)
		if requireAll && !matched {
			return false
		}
		if !requireAll && matched {
			return true
		}
		matchedAny = matchedAny || matched
	}
	return requireAll && matchedAny
}

func cosmosFullTextScore(doc map[string]any, args []string) any {
	if len(args) < 2 {
		return cosmosUndefined
	}
	value, ok := cosmosExpressionValue(doc, args[0]).(string)
	if !ok {
		return cosmosUndefined
	}
	score := 0.0
	for _, arg := range args[1:] {
		term := strings.TrimSpace(fmt.Sprint(cosmosExpressionValue(doc, arg)))
		if term == "" {
			continue
		}
		score += float64(strings.Count(strings.ToLower(value), strings.ToLower(term)))
	}
	return normalizeCosmosNumber(score)
}

func cosmosFullTextContainsTerm(value, term string) bool {
	term = strings.TrimSpace(term)
	if term == "" {
		return false
	}
	return strings.Contains(strings.ToLower(value), strings.ToLower(term))
}

func cosmosRegexMatch(doc map[string]any, args []string) any {
	value, ok := cosmosStringFunctionArgument(doc, args[0])
	if !ok {
		return cosmosUndefined
	}
	pattern, ok := cosmosStringFunctionArgument(doc, args[1])
	if !ok {
		return cosmosUndefined
	}
	flags := ""
	if len(args) > 2 {
		flags, ok = cosmosStringFunctionArgument(doc, args[2])
		if !ok {
			return cosmosUndefined
		}
	}
	if strings.Contains(flags, "i") {
		pattern = "(?i)" + pattern
	}
	if strings.Contains(flags, "m") {
		pattern = "(?m)" + pattern
	}
	if strings.Contains(flags, "s") {
		pattern = "(?s)" + pattern
	}
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}
	return compiled.MatchString(value)
}

func cosmosSubstring(value string, start, length int) string {
	if start < 0 {
		start = 0
	}
	if length <= 0 || start >= len(value) {
		return ""
	}
	end := start + length
	if end > len(value) {
		end = len(value)
	}
	return value[start:end]
}

func cosmosLeftString(doc map[string]any, args []string) any {
	value, count, ok := cosmosStringCountArgs(doc, args)
	if !ok {
		return cosmosUndefined
	}
	if count <= 0 {
		return ""
	}
	if count >= len(value) {
		return value
	}
	return value[:count]
}

func cosmosRightString(doc map[string]any, args []string) any {
	value, count, ok := cosmosStringCountArgs(doc, args)
	if !ok {
		return cosmosUndefined
	}
	if count <= 0 {
		return ""
	}
	if count >= len(value) {
		return value
	}
	return value[len(value)-count:]
}

func cosmosStringCountArgs(doc map[string]any, args []string) (string, int, bool) {
	if len(args) < 2 {
		return "", 0, false
	}
	value, ok := cosmosStringFunctionArgument(doc, args[0])
	if !ok {
		return "", 0, false
	}
	countNumber, ok := cosmosNumericFunctionArgument(doc, args[1])
	if !ok {
		return "", 0, false
	}
	return value, int(countNumber), true
}

func cosmosTrimLeftString(doc map[string]any, args []string) any {
	if len(args) == 0 {
		return cosmosUndefined
	}
	value, ok := cosmosStringFunctionArgument(doc, args[0])
	if !ok {
		return cosmosUndefined
	}
	if len(args) == 1 {
		return strings.TrimLeftFunc(value, unicode.IsSpace)
	}
	prefix, ok := cosmosStringFunctionArgument(doc, args[1])
	if !ok {
		return cosmosUndefined
	}
	if prefix == "" {
		return value
	}
	return strings.TrimPrefix(value, prefix)
}

func cosmosTrimRightString(doc map[string]any, args []string) any {
	if len(args) == 0 {
		return cosmosUndefined
	}
	value, ok := cosmosStringFunctionArgument(doc, args[0])
	if !ok {
		return cosmosUndefined
	}
	if len(args) == 1 {
		return strings.TrimRightFunc(value, unicode.IsSpace)
	}
	suffix, ok := cosmosStringFunctionArgument(doc, args[1])
	if !ok {
		return cosmosUndefined
	}
	if suffix == "" {
		return value
	}
	return strings.TrimSuffix(value, suffix)
}

func cosmosTrimString(doc map[string]any, args []string) any {
	if len(args) == 0 {
		return cosmosUndefined
	}
	value, ok := cosmosStringFunctionArgument(doc, args[0])
	if !ok {
		return cosmosUndefined
	}
	if len(args) == 1 {
		return strings.TrimSpace(value)
	}
	cutset, ok := cosmosStringFunctionArgument(doc, args[1])
	if !ok {
		return cosmosUndefined
	}
	return strings.Trim(value, cutset)
}

func cosmosReplaceString(doc map[string]any, args []string) any {
	if len(args) < 3 {
		return cosmosUndefined
	}
	value, ok := cosmosStringFunctionArgument(doc, args[0])
	if !ok {
		return cosmosUndefined
	}
	oldValue, ok := cosmosStringFunctionArgument(doc, args[1])
	if !ok {
		return cosmosUndefined
	}
	newValue, ok := cosmosStringFunctionArgument(doc, args[2])
	if !ok {
		return cosmosUndefined
	}
	return strings.ReplaceAll(value, oldValue, newValue)
}

func cosmosIndexOfString(doc map[string]any, args []string) any {
	if len(args) < 2 {
		return cosmosUndefined
	}
	value, ok := cosmosStringFunctionArgument(doc, args[0])
	if !ok {
		return cosmosUndefined
	}
	search, ok := cosmosStringFunctionArgument(doc, args[1])
	if !ok {
		return cosmosUndefined
	}
	start := 0
	if len(args) > 2 {
		startNumber, ok := cosmosNumericFunctionArgument(doc, args[2])
		if !ok {
			return cosmosUndefined
		}
		start = int(startNumber)
	}
	return int64(cosmosIndexOf(value, search, start))
}

func cosmosReplicateString(doc map[string]any, args []string) any {
	value, count, ok := cosmosStringCountArgs(doc, args)
	if !ok || count < 0 || len(value)*count > 10000 {
		return cosmosUndefined
	}
	return strings.Repeat(value, count)
}

func cosmosReverseString(value string) string {
	runes := []rune(value)
	for left, right := 0, len(runes)-1; left < right; left, right = left+1, right-1 {
		runes[left], runes[right] = runes[right], runes[left]
	}
	return string(runes)
}

func cosmosIndexOf(value, search string, start int) int {
	if start < 0 {
		start = 0
	}
	if start > len(value) {
		return -1
	}
	idx := strings.Index(value[start:], search)
	if idx < 0 {
		return -1
	}
	return start + idx
}

func cosmosStringJoin(doc map[string]any, args []string) any {
	if len(args) < 2 {
		return cosmosUndefined
	}
	rawValues := cosmosExpressionValue(doc, args[0])
	if cosmosIsUndefined(rawValues) {
		return cosmosUndefined
	}
	values, ok := rawValues.([]any)
	if !ok {
		return cosmosUndefined
	}
	separator, ok := cosmosExpressionValue(doc, args[1]).(string)
	if !ok {
		return cosmosUndefined
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		part, ok := value.(string)
		if !ok {
			return cosmosUndefined
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, separator)
}

func cosmosStringSplit(doc map[string]any, args []string) any {
	if len(args) < 2 {
		return cosmosUndefined
	}
	value, ok := cosmosExpressionValue(doc, args[0]).(string)
	if !ok {
		return cosmosUndefined
	}
	separator, ok := cosmosExpressionValue(doc, args[1]).(string)
	if !ok {
		return cosmosUndefined
	}
	if separator == "" {
		return []any{value}
	}
	parts := strings.Split(value, separator)
	out := make([]any, len(parts))
	for i, part := range parts {
		out[i] = part
	}
	return out
}

func cosmosStringToArray(value any) any {
	if cosmosIsUndefined(value) {
		return cosmosUndefined
	}
	raw, ok := value.(string)
	if !ok {
		return cosmosUndefined
	}
	var out []any
	if err := gojson.Unmarshal([]byte(strings.TrimSpace(raw)), &out); err != nil {
		return cosmosUndefined
	}
	return out
}

func cosmosStringToNumber(value any) any {
	if cosmosIsUndefined(value) {
		return cosmosUndefined
	}
	raw, ok := value.(string)
	if !ok {
		return cosmosUndefined
	}
	number, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || math.IsNaN(number) || math.IsInf(number, 0) {
		return cosmosUndefined
	}
	return normalizeCosmosNumber(number)
}

func cosmosStringToBoolean(value any) any {
	if cosmosIsUndefined(value) {
		return cosmosUndefined
	}
	raw, ok := value.(string)
	if !ok {
		return cosmosUndefined
	}
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "true":
		return true
	case "false":
		return false
	default:
		return cosmosUndefined
	}
}

func cosmosStringToNull(value any) any {
	if cosmosIsUndefined(value) {
		return cosmosUndefined
	}
	raw, ok := value.(string)
	if !ok {
		return cosmosUndefined
	}
	if strings.TrimSpace(raw) == "null" {
		return nil
	}
	return cosmosUndefined
}

func cosmosStringToObject(value any) any {
	if cosmosIsUndefined(value) {
		return cosmosUndefined
	}
	raw, ok := value.(string)
	if !ok {
		return cosmosUndefined
	}
	var out map[string]any
	if err := gojson.Unmarshal([]byte(strings.TrimSpace(raw)), &out); err != nil {
		return cosmosUndefined
	}
	return out
}

func cosmosToString(value any) any {
	if cosmosIsUndefined(value) {
		return cosmosUndefined
	}
	switch typed := value.(type) {
	case string:
		return typed
	case bool:
		if typed {
			return "true"
		}
		return "false"
	case nil:
		return "null"
	case float64:
		if math.IsNaN(typed) {
			return "NaN"
		}
		if math.IsInf(typed, 1) {
			return "Infinity"
		}
		if math.IsInf(typed, -1) {
			return "-Infinity"
		}
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case float32:
		value := float64(typed)
		if math.IsNaN(value) {
			return "NaN"
		}
		if math.IsInf(value, 1) {
			return "Infinity"
		}
		if math.IsInf(value, -1) {
			return "-Infinity"
		}
		return strconv.FormatFloat(value, 'f', -1, 32)
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case int32:
		return strconv.FormatInt(int64(typed), 10)
	case gojson.Number:
		return string(typed)
	default:
		data, err := gojson.Marshal(typed)
		if err != nil {
			return cosmosUndefined
		}
		return string(data)
	}
}

func cosmosIsUndefined(value any) bool {
	_, ok := value.(cosmosUndefinedType)
	return ok
}

func cosmosDateTimeFromParts(doc map[string]any, args []string) any {
	if len(args) < 3 || len(args) > 7 {
		return cosmosUndefined
	}
	values := []int64{0, 0, 0, 0, 0, 0, 0}
	for i, arg := range args {
		value, ok := cosmosIntegerExpressionValue(doc, arg)
		if !ok {
			return cosmosUndefined
		}
		values[i] = value
	}
	year, month, day := values[0], values[1], values[2]
	hour, minute, second, fraction := values[3], values[4], values[5], values[6]
	if year <= 0 || month < 1 || month > 12 || day < 1 || hour < 0 || hour > 23 || minute < 0 || minute > 59 || second < 0 || second > 59 || fraction < 0 || fraction > 9999999 {
		return cosmosUndefined
	}
	value := time.Date(int(year), time.Month(month), int(day), int(hour), int(minute), int(second), int(fraction*100), time.UTC)
	if value.Year() != int(year) || int(value.Month()) != int(month) || value.Day() != int(day) {
		return cosmosUndefined
	}
	return cosmosFormatDateTime(value)
}

func cosmosDateTimeBin(doc map[string]any, args []string) any {
	if len(args) < 2 || len(args) > 4 {
		return cosmosUndefined
	}
	rawDate, ok := cosmosExpressionValue(doc, args[0]).(string)
	if !ok {
		return cosmosUndefined
	}
	value, ok := cosmosParseDateTime(rawDate)
	if !ok {
		return cosmosUndefined
	}
	part, ok := cosmosExpressionValue(doc, args[1]).(string)
	if !ok {
		return cosmosUndefined
	}
	binSize := int64(1)
	if len(args) >= 3 {
		binSize, ok = cosmosIntegerExpressionValue(doc, args[2])
		if !ok || binSize <= 0 {
			return cosmosUndefined
		}
	}
	start := time.Unix(0, 0).UTC()
	if len(args) == 4 {
		rawStart, ok := cosmosExpressionValue(doc, args[3]).(string)
		if !ok {
			return cosmosUndefined
		}
		start, ok = cosmosParseDateTime(rawStart)
		if !ok || start.Before(time.Date(1601, 1, 1, 0, 0, 0, 0, time.UTC)) {
			return cosmosUndefined
		}
	}
	switch cosmosDateTimePartKind(part) {
	case "year":
		return cosmosFormatDateTime(cosmosDateTimeBinMonths(value, start, binSize*12))
	case "month":
		return cosmosFormatDateTime(cosmosDateTimeBinMonths(value, start, binSize))
	case "day":
		return cosmosFormatDateTime(cosmosDateTimeBinFixed(value, start, binSize, 24*60*60*1000000000))
	case "hour":
		return cosmosFormatDateTime(cosmosDateTimeBinFixed(value, start, binSize, 60*60*1000000000))
	case "minute":
		return cosmosFormatDateTime(cosmosDateTimeBinFixed(value, start, binSize, 60*1000000000))
	case "second":
		return cosmosFormatDateTime(cosmosDateTimeBinFixed(value, start, binSize, 1000000000))
	case "millisecond":
		return cosmosFormatDateTime(cosmosDateTimeBinFixed(value, start, binSize, 1000000))
	case "microsecond":
		return cosmosFormatDateTime(cosmosDateTimeBinFixed(value, start, binSize, 1000))
	case "nanosecond":
		return cosmosFormatDateTime(cosmosDateTimeBinFixed(value, start, binSize, 1))
	default:
		return cosmosUndefined
	}
}

func cosmosDateTimeAdd(doc map[string]any, args []string) any {
	if len(args) != 3 {
		return cosmosUndefined
	}
	part, ok := cosmosExpressionValue(doc, args[0]).(string)
	if !ok {
		return cosmosUndefined
	}
	amountNumber, ok := cosmosNumericValue(cosmosExpressionValue(doc, args[1]))
	if !ok || amountNumber != math.Trunc(amountNumber) {
		return cosmosUndefined
	}
	rawDate, ok := cosmosExpressionValue(doc, args[2]).(string)
	if !ok {
		return cosmosUndefined
	}
	value, ok := cosmosParseDateTime(rawDate)
	if !ok {
		return cosmosUndefined
	}
	amount := int(amountNumber)
	switch cosmosDateTimePartKind(part) {
	case "year":
		value = value.AddDate(amount, 0, 0)
	case "month":
		value = value.AddDate(0, amount, 0)
	case "day":
		value = value.AddDate(0, 0, amount)
	case "hour":
		value = value.Add(time.Duration(amount) * time.Hour)
	case "minute":
		value = value.Add(time.Duration(amount) * time.Minute)
	case "second":
		value = value.Add(time.Duration(amount) * time.Second)
	case "millisecond":
		value = value.Add(time.Duration(amount) * time.Millisecond)
	case "microsecond":
		value = value.Add(time.Duration(amount) * time.Microsecond)
	case "nanosecond":
		value = value.Add(time.Duration(amount) * time.Nanosecond)
	default:
		return cosmosUndefined
	}
	return cosmosFormatDateTime(value)
}

func cosmosDateTimeDiff(doc map[string]any, args []string) any {
	if len(args) != 3 {
		return cosmosUndefined
	}
	part, ok := cosmosExpressionValue(doc, args[0]).(string)
	if !ok {
		return cosmosUndefined
	}
	startRaw, ok := cosmosExpressionValue(doc, args[1]).(string)
	if !ok {
		return cosmosUndefined
	}
	endRaw, ok := cosmosExpressionValue(doc, args[2]).(string)
	if !ok {
		return cosmosUndefined
	}
	start, ok := cosmosParseDateTime(startRaw)
	if !ok {
		return cosmosUndefined
	}
	end, ok := cosmosParseDateTime(endRaw)
	if !ok {
		return cosmosUndefined
	}
	switch cosmosDateTimePartKind(part) {
	case "year":
		return int64(end.Year() - start.Year())
	case "month":
		return int64((end.Year()-start.Year())*12 + int(end.Month()) - int(start.Month()))
	case "day":
		return int64(end.Sub(start).Hours() / 24)
	case "hour":
		return int64(end.Sub(start).Hours())
	case "minute":
		return int64(end.Sub(start).Minutes())
	case "second":
		return int64(end.Sub(start).Seconds())
	case "millisecond":
		return int64(end.Sub(start) / time.Millisecond)
	case "microsecond":
		return int64(end.Sub(start) / time.Microsecond)
	case "nanosecond":
		return int64(end.Sub(start) / time.Nanosecond)
	default:
		return cosmosUndefined
	}
}

func cosmosDateTimePart(doc map[string]any, args []string) any {
	if len(args) != 2 {
		return cosmosUndefined
	}
	part, ok := cosmosExpressionValue(doc, args[0]).(string)
	if !ok {
		return cosmosUndefined
	}
	rawDate, ok := cosmosExpressionValue(doc, args[1]).(string)
	if !ok {
		return cosmosUndefined
	}
	value, ok := cosmosParseDateTime(rawDate)
	if !ok {
		return cosmosUndefined
	}
	switch cosmosDateTimePartKind(part) {
	case "year":
		return int64(value.Year())
	case "month":
		return int64(value.Month())
	case "day":
		return int64(value.Day())
	case "hour":
		return int64(value.Hour())
	case "minute":
		return int64(value.Minute())
	case "second":
		return int64(value.Second())
	case "millisecond":
		return int64(value.Nanosecond() / int(time.Millisecond))
	case "microsecond":
		return int64(value.Nanosecond() / int(time.Microsecond))
	case "nanosecond":
		return int64(value.Nanosecond())
	default:
		return cosmosUndefined
	}
}

func cosmosDateTimeToTimestamp(doc map[string]any, args []string) any {
	if len(args) != 1 {
		return cosmosUndefined
	}
	rawDate, ok := cosmosExpressionValue(doc, args[0]).(string)
	if !ok {
		return cosmosUndefined
	}
	value, ok := cosmosParseDateTime(rawDate)
	if !ok {
		return cosmosUndefined
	}
	return value.Unix()*1000 + int64(value.Nanosecond()/int(time.Millisecond))
}

func cosmosTimestampToDateTime(doc map[string]any, args []string) any {
	if len(args) != 1 {
		return cosmosUndefined
	}
	millis, ok := cosmosIntegerExpressionValue(doc, args[0])
	if !ok {
		return cosmosUndefined
	}
	seconds := millis / 1000
	remainderMillis := millis % 1000
	return cosmosFormatDateTime(time.Unix(seconds, remainderMillis*int64(time.Millisecond)).UTC())
}

func cosmosDateTimeToTicks(doc map[string]any, args []string) any {
	if len(args) != 1 {
		return cosmosUndefined
	}
	rawDate, ok := cosmosExpressionValue(doc, args[0]).(string)
	if !ok {
		return cosmosUndefined
	}
	value, ok := cosmosParseDateTime(rawDate)
	if !ok {
		return cosmosUndefined
	}
	return value.Unix()*10000000 + int64(value.Nanosecond()/100)
}

func cosmosTicksToDateTime(doc map[string]any, args []string) any {
	if len(args) != 1 {
		return cosmosUndefined
	}
	ticks, ok := cosmosIntegerExpressionValue(doc, args[0])
	if !ok {
		return cosmosUndefined
	}
	seconds := ticks / 10000000
	remainderTicks := ticks % 10000000
	return cosmosFormatDateTime(time.Unix(seconds, remainderTicks*100).UTC())
}

func cosmosIntegerExpressionValue(doc map[string]any, expression string) (int64, bool) {
	if parsed, ok := cosmosIntegerLiteral(expression); ok {
		return parsed, true
	}
	number, ok := cosmosNumericValue(cosmosExpressionValue(doc, expression))
	if !ok || number != math.Trunc(number) || number < float64(math.MinInt64) || number > float64(math.MaxInt64) {
		return 0, false
	}
	return int64(number), true
}

func cosmosIntegerLiteral(expression string) (int64, bool) {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		return 0, false
	}
	if strings.ContainsAny(expression, ".eE") {
		return 0, false
	}
	value, err := strconv.ParseInt(expression, 10, 64)
	return value, err == nil
}

func cosmosDateTimeBinFixed(value, start time.Time, binSize, unitNanos int64) time.Time {
	const nanosPerSecond int64 = 1000000000
	if unitNanos >= nanosPerSecond && unitNanos%nanosPerSecond == 0 {
		unitSeconds := unitNanos / nanosPerSecond
		deltaSeconds := value.Unix() - start.Unix()
		if value.Nanosecond() < start.Nanosecond() {
			deltaSeconds--
		}
		deltaUnits := cosmosFloorDiv(deltaSeconds, unitSeconds)
		binnedUnits := cosmosFloorDiv(deltaUnits, binSize) * binSize
		return time.Unix(start.Unix()+binnedUnits*unitSeconds, int64(start.Nanosecond())).UTC()
	}

	deltaSeconds := value.Unix() - start.Unix()
	if deltaSeconds > math.MaxInt64/nanosPerSecond || deltaSeconds < math.MinInt64/nanosPerSecond {
		return value
	}
	deltaNanos := deltaSeconds*nanosPerSecond + int64(value.Nanosecond()-start.Nanosecond())
	deltaUnits := cosmosFloorDiv(deltaNanos, unitNanos)
	binnedUnits := cosmosFloorDiv(deltaUnits, binSize) * binSize
	binnedNanos := binnedUnits * unitNanos
	seconds := cosmosFloorDiv(binnedNanos, nanosPerSecond)
	nanos := binnedNanos - seconds*nanosPerSecond
	return time.Unix(start.Unix()+seconds, int64(start.Nanosecond())+nanos).UTC()
}

func cosmosDateTimeBinMonths(value, start time.Time, binMonths int64) time.Time {
	startIndex := int64(start.Year()*12 + int(start.Month()) - 1)
	valueIndex := int64(value.Year()*12 + int(value.Month()) - 1)
	deltaMonths := valueIndex - startIndex
	binnedMonths := cosmosFloorDiv(deltaMonths, binMonths) * binMonths
	targetIndex := startIndex + binnedMonths
	year := int(targetIndex / 12)
	month := time.Month(targetIndex%12 + 1)
	return time.Date(year, month, start.Day(), start.Hour(), start.Minute(), start.Second(), start.Nanosecond(), time.UTC)
}

func cosmosFloorDiv(value, divisor int64) int64 {
	if divisor == 0 {
		return 0
	}
	quotient := value / divisor
	remainder := value % divisor
	if remainder != 0 && ((remainder < 0) != (divisor < 0)) {
		quotient--
	}
	return quotient
}

func cosmosDateTimePartKind(part string) string {
	switch strings.ToLower(strings.TrimSpace(part)) {
	case "year", "yyyy", "yy":
		return "year"
	case "month", "mm", "m":
		return "month"
	case "day", "dd", "d":
		return "day"
	case "hour", "hh":
		return "hour"
	case "minute", "mi", "n":
		return "minute"
	case "second", "ss", "s":
		return "second"
	case "millisecond", "ms":
		return "millisecond"
	case "microsecond", "mcs":
		return "microsecond"
	case "nanosecond", "ns":
		return "nanosecond"
	default:
		return ""
	}
}

func cosmosParseDateTime(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	layouts := []string{
		"2006-01-02T15:04:05.999999999Z07:00",
		"2006-01-02T15:04:05.999999999",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
	}
	for _, layout := range layouts {
		value, err := time.Parse(layout, raw)
		if err == nil {
			return value.UTC(), true
		}
	}
	return time.Time{}, false
}

func cosmosFormatDateTime(value time.Time) string {
	value = value.UTC()
	return fmt.Sprintf("%s.%07dZ", value.Format("2006-01-02T15:04:05"), value.Nanosecond()/100)
}

func cosmosStaticNow(doc map[string]any) time.Time {
	if value, ok := doc[cosmosStaticNowField].(time.Time); ok {
		return value.UTC()
	}
	return time.Now().UTC()
}

func cosmosArrayContains(values []any, needle any, partial bool) bool {
	for _, value := range values {
		if partial {
			if cosmosPartialObjectMatch(value, needle) {
				return true
			}
			continue
		}
		if cosmosValuesEqual(value, needle) {
			return true
		}
	}
	return false
}

func cosmosArrayContainsAllOrAny(doc map[string]any, args []string, requireAll bool) any {
	if len(args) < 2 {
		return cosmosUndefined
	}
	values, ok := cosmosExpressionValue(doc, args[0]).([]any)
	if !ok {
		return cosmosUndefined
	}
	seenUndefined := false
	for _, arg := range args[1:] {
		needle := cosmosExpressionValue(doc, arg)
		if cosmosIsUndefined(needle) {
			seenUndefined = true
			continue
		}
		contains := cosmosArrayContains(values, needle, false)
		if requireAll && !contains {
			return false
		}
		if !requireAll && contains {
			return true
		}
	}
	if seenUndefined {
		return cosmosUndefined
	}
	return requireAll
}

func cosmosChoose(doc map[string]any, args []string) any {
	if len(args) < 2 {
		return cosmosUndefined
	}
	number, ok := cosmosNumericValue(cosmosExpressionValue(doc, args[0]))
	if !ok || number != math.Trunc(number) {
		return cosmosUndefined
	}
	index := int(number)
	if index < 1 || index >= len(args) {
		return cosmosUndefined
	}
	return cosmosExpressionValue(doc, args[index])
}

func cosmosDocumentID(doc map[string]any, args []string) any {
	if len(args) == 0 {
		return cosmosUndefined
	}
	target, ok := cosmosDocumentIDTarget(doc, args[0])
	if !ok {
		return cosmosUndefined
	}
	rid, ok := target["_rid"].(string)
	if !ok || rid == "" {
		return cosmosUndefined
	}
	idx := strings.LastIndex(rid, "-")
	if idx < 0 || idx+1 >= len(rid) {
		return cosmosUndefined
	}
	value, err := strconv.ParseInt(rid[idx+1:], 10, 64)
	if err != nil {
		return cosmosUndefined
	}
	return value
}

func cosmosDocumentIDTarget(doc map[string]any, expression string) (map[string]any, bool) {
	expression = strings.TrimSpace(expression)
	if value, ok := cosmosExpressionValue(doc, expression).(map[string]any); ok {
		return value, true
	}
	if expression != "" && cosmosSimpleIdentifier(expression) {
		return doc, true
	}
	return nil, false
}

func cosmosSimpleIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for i, r := range value {
		if r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || i > 0 && r >= '0' && r <= '9' {
			continue
		}
		return false
	}
	return true
}

func cosmosObjectToArray(doc map[string]any, args []string) any {
	if len(args) == 0 || len(args) > 3 {
		return cosmosUndefined
	}
	object, ok := cosmosExpressionValue(doc, args[0]).(map[string]any)
	if !ok {
		return cosmosUndefined
	}
	keyField := "k"
	valueField := "v"
	if len(args) > 1 {
		keyField, ok = cosmosExpressionValue(doc, args[1]).(string)
		if !ok || keyField == "" {
			return cosmosUndefined
		}
	}
	if len(args) > 2 {
		valueField, ok = cosmosExpressionValue(doc, args[2]).(string)
		if !ok || valueField == "" {
			return cosmosUndefined
		}
	}
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, map[string]any{
			keyField:   key,
			valueField: object[key],
		})
	}
	return out
}

func cosmosSetIntersect(doc map[string]any, args []string) any {
	left, right, ok := cosmosArrayPairArgs(doc, args)
	if !ok {
		return cosmosUndefined
	}
	seen := make(map[string]bool, len(right))
	out := make([]any, 0)
	for _, value := range right {
		key := cosmosDistinctKey(value)
		if seen[key] || !cosmosArrayContains(left, value, false) {
			continue
		}
		seen[key] = true
		out = append(out, value)
	}
	return out
}

func cosmosSetUnion(doc map[string]any, args []string) any {
	left, right, ok := cosmosArrayPairArgs(doc, args)
	if !ok {
		return cosmosUndefined
	}
	return distinctCosmosValues(append(append([]any{}, left...), right...))
}

func cosmosArrayPairArgs(doc map[string]any, args []string) ([]any, []any, bool) {
	if len(args) != 2 {
		return nil, nil, false
	}
	left, leftOK := cosmosExpressionValue(doc, args[0]).([]any)
	right, rightOK := cosmosExpressionValue(doc, args[1]).([]any)
	return left, right, leftOK && rightOK
}

func cosmosPartialObjectMatch(value, needle any) bool {
	needleMap, ok := needle.(map[string]any)
	if !ok {
		return cosmosValuesEqual(value, needle)
	}
	valueMap, ok := value.(map[string]any)
	if !ok {
		return false
	}
	for key, needleValue := range needleMap {
		valueValue, exists := valueMap[key]
		if !exists || !cosmosValuesEqual(valueValue, needleValue) {
			return false
		}
	}
	return true
}

func cosmosPathValue(doc map[string]any, path string) (any, bool) {
	path = cosmosExpressionWithoutAlias(strings.TrimSpace(path))
	if path == "" {
		return nil, false
	}
	path = cosmosStripPathAlias(doc, path)
	current := any(doc)
	for _, segment := range cosmosPathSegments(path) {
		next, ok := cosmosPathSegmentValue(current, segment)
		if !ok {
			return nil, false
		}
		current = next
	}
	return current, true
}

func cosmosStripPathAlias(doc map[string]any, path string) string {
	identEnd := 0
	for identEnd < len(path) {
		c := path[identEnd]
		if !(c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || identEnd > 0 && c >= '0' && c <= '9') {
			break
		}
		identEnd++
	}
	if identEnd == 0 || identEnd >= len(path) {
		return path
	}
	alias := path[:identEnd]
	if _, exists := doc[alias]; exists {
		return path
	}
	switch path[identEnd] {
	case '.':
		return path[identEnd+1:]
	case '[':
		return path[identEnd:]
	default:
		return path
	}
}

func cosmosPathSegments(path string) []string {
	segments := make([]string, 0)
	start := 0
	depth := 0
	inQuote := false
	var quote rune
	for i, r := range path {
		if inQuote {
			if r == quote {
				inQuote = false
			}
			continue
		}
		switch r {
		case '\'', '"':
			if depth > 0 {
				inQuote = true
				quote = r
			}
		case '[':
			depth++
		case ']':
			if depth > 0 {
				depth--
			}
		case '.':
			if depth == 0 {
				segments = append(segments, strings.TrimSpace(path[start:i]))
				start = i + 1
			}
		}
	}
	segments = append(segments, strings.TrimSpace(path[start:]))
	return segments
}

func cosmosPathSegmentValue(current any, segment string) (any, bool) {
	for segment != "" {
		bracket := strings.Index(segment, "[")
		property := segment
		if bracket >= 0 {
			property = segment[:bracket]
		}
		if property != "" {
			object, ok := current.(map[string]any)
			if !ok {
				return nil, false
			}
			current, ok = object[property]
			if !ok {
				return nil, false
			}
		}
		if bracket < 0 {
			return current, true
		}
		close := strings.Index(segment[bracket:], "]")
		if close < 0 {
			return nil, false
		}
		close += bracket
		key := strings.TrimSpace(segment[bracket+1 : close])
		key = strings.Trim(key, `"'`)
		switch typed := current.(type) {
		case []any:
			index, err := strconv.Atoi(key)
			if err != nil || index < 0 || index >= len(typed) {
				return nil, false
			}
			current = typed[index]
		case map[string]any:
			value, ok := typed[key]
			if !ok {
				return nil, false
			}
			current = value
		default:
			return nil, false
		}
		segment = segment[close+1:]
	}
	return current, true
}

func cosmosMatchingCloseParen(input string, start int) int {
	depth := 0
	inQuote := false
	var quote rune
	for i, r := range input {
		if i < start {
			continue
		}
		if inQuote {
			if r == quote {
				inQuote = false
			}
			continue
		}
		switch r {
		case '\'', '"':
			inQuote = true
			quote = r
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func findCosmosTopLevelKeyword(input, keyword string) int {
	depth := 0
	bracketDepth := 0
	braceDepth := 0
	inQuote := false
	var quote rune
	upper := strings.ToUpper(input)
	keyword = strings.ToUpper(keyword)
	betweenPending := false
	for i, r := range input {
		if inQuote {
			if r == quote {
				inQuote = false
			}
			continue
		}
		switch r {
		case '\'', '"':
			inQuote = true
			quote = r
			continue
		case '(':
			depth++
			continue
		case ')':
			if depth > 0 {
				depth--
			}
			continue
		case '[':
			bracketDepth++
			continue
		case ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
			continue
		case '{':
			braceDepth++
			continue
		case '}':
			if braceDepth > 0 {
				braceDepth--
			}
			continue
		}
		if depth != 0 || bracketDepth != 0 || braceDepth != 0 || !strings.HasPrefix(upper[i:], keyword) {
			if depth == 0 && bracketDepth == 0 && braceDepth == 0 && keyword == "AND" && cosmosKeywordAt(input, upper, i, "BETWEEN") {
				betweenPending = true
			}
			continue
		}
		end := i + len(keyword)
		beforeOK := i == 0 || !cosmosIdentifierRune(rune(input[i-1]))
		afterOK := end >= len(input) || !cosmosIdentifierRune(rune(input[end]))
		if beforeOK && afterOK && keyword == "AND" && betweenPending {
			betweenPending = false
			continue
		}
		if beforeOK && afterOK {
			return i
		}
	}
	return -1
}

func cosmosAndBelongsToBetween(input string, andIdx int) bool {
	depth := 0
	bracketDepth := 0
	inQuote := false
	var quote rune
	lastKeyword := ""
	upper := strings.ToUpper(input)
	for i, r := range input {
		if i >= andIdx {
			break
		}
		if inQuote {
			if r == quote {
				inQuote = false
			}
			continue
		}
		switch r {
		case '\'', '"':
			inQuote = true
			quote = r
			continue
		case '(':
			depth++
			continue
		case ')':
			if depth > 0 {
				depth--
			}
			continue
		case '[':
			bracketDepth++
			continue
		case ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
			continue
		}
		if depth != 0 || bracketDepth != 0 {
			continue
		}
		for _, keyword := range []string{"BETWEEN", "AND", "OR"} {
			if cosmosKeywordAt(input, upper, i, keyword) {
				lastKeyword = keyword
			}
		}
	}
	return lastKeyword == "BETWEEN"
}

func cosmosKeywordAt(input, upper string, idx int, keyword string) bool {
	if !strings.HasPrefix(upper[idx:], keyword) {
		return false
	}
	end := idx + len(keyword)
	beforeOK := idx == 0 || !cosmosIdentifierRune(rune(input[idx-1]))
	afterOK := end >= len(input) || !cosmosIdentifierRune(rune(input[end]))
	return beforeOK && afterOK
}

func cosmosTopLevelComparison(input string) (int, string) {
	depth := 0
	bracketDepth := 0
	braceDepth := 0
	inQuote := false
	var quote rune
	for i, r := range input {
		if inQuote {
			if r == quote {
				inQuote = false
			}
			continue
		}
		switch r {
		case '\'', '"':
			inQuote = true
			quote = r
			continue
		case '(':
			depth++
			continue
		case ')':
			if depth > 0 {
				depth--
			}
			continue
		case '[':
			bracketDepth++
			continue
		case ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
			continue
		case '{':
			braceDepth++
			continue
		case '}':
			if braceDepth > 0 {
				braceDepth--
			}
			continue
		}
		if depth != 0 || bracketDepth != 0 || braceDepth != 0 {
			continue
		}
		for _, op := range []string{">=", "<=", "!=", "<>", "=", ">", "<"} {
			if strings.HasPrefix(input[i:], op) {
				return i, op
			}
		}
	}
	if idx := findCosmosTopLevelKeyword(input, "EQ"); idx >= 0 {
		return idx, "EQ"
	}
	return -1, ""
}

func cosmosTopLevelArithmetic(input, operators string) (int, string) {
	depth := 0
	bracketDepth := 0
	braceDepth := 0
	inQuote := false
	var quote byte
	for i := len(input) - 1; i >= 0; i-- {
		ch := input[i]
		if inQuote {
			if ch == quote {
				inQuote = false
			}
			continue
		}
		switch ch {
		case '\'', '"':
			inQuote = true
			quote = ch
			continue
		case ')':
			depth++
			continue
		case '(':
			if depth > 0 {
				depth--
			}
			continue
		case ']':
			bracketDepth++
			continue
		case '[':
			if bracketDepth > 0 {
				bracketDepth--
			}
			continue
		case '}':
			braceDepth++
			continue
		case '{':
			if braceDepth > 0 {
				braceDepth--
			}
			continue
		}
		if depth != 0 || bracketDepth != 0 || braceDepth != 0 || !strings.ContainsRune(operators, rune(ch)) {
			continue
		}
		if (ch == '+' || ch == '-') && cosmosArithmeticSignIsUnary(input, i) {
			continue
		}
		return i, string(ch)
	}
	return -1, ""
}

func cosmosArithmeticSignIsUnary(input string, idx int) bool {
	for i := idx - 1; i >= 0; i-- {
		ch := input[i]
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			continue
		}
		return strings.ContainsRune("+-*/%(,[{:", rune(ch)) || ch == 'e' || ch == 'E'
	}
	return true
}

func cosmosTopLevelColon(input string) int {
	depth := 0
	bracketDepth := 0
	braceDepth := 0
	inQuote := false
	var quote rune
	for i, r := range input {
		if inQuote {
			if r == quote {
				inQuote = false
			}
			continue
		}
		switch r {
		case '\'', '"':
			inQuote = true
			quote = r
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case '[':
			bracketDepth++
		case ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
		case '{':
			braceDepth++
		case '}':
			if braceDepth > 0 {
				braceDepth--
			}
		case ':':
			if depth == 0 && bracketDepth == 0 && braceDepth == 0 {
				return i
			}
		}
	}
	return -1
}

func cosmosIdentifierRune(r rune) bool {
	return r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9'
}

func cosmosArraySlice(values []any, start, length int) []any {
	if start < 0 {
		start = len(values) + start
	}
	if start < 0 {
		start = 0
	}
	if start > len(values) || length <= 0 {
		return []any{}
	}
	end := start + length
	if end > len(values) {
		end = len(values)
	}
	return values[start:end]
}

func cosmosExpressionWithoutAlias(expression string) string {
	if aliasIdx := strings.Index(strings.ToUpper(expression), " AS "); aliasIdx >= 0 {
		return strings.TrimSpace(expression[:aliasIdx])
	}
	return expression
}

func cosmosProjectionAlias(field string, position int) string {
	if aliasIdx := strings.Index(strings.ToUpper(field), " AS "); aliasIdx >= 0 {
		return strings.TrimSpace(field[aliasIdx+4:])
	}
	if alias, ok := cosmosPathProjectionAlias(field); ok {
		return alias
	}
	if position < 1 {
		position = 1
	}
	return fmt.Sprintf("$%d", position)
}

func fieldAlias(field string) string {
	if aliasIdx := strings.Index(strings.ToUpper(field), " AS "); aliasIdx >= 0 {
		return strings.TrimSpace(field[aliasIdx+4:])
	}
	if alias, ok := cosmosPathProjectionAlias(field); ok {
		return alias
	}
	return "$1"
}

func cosmosPathProjectionAlias(field string) (string, bool) {
	if aliasIdx := strings.Index(strings.ToUpper(field), " AS "); aliasIdx >= 0 {
		field = strings.TrimSpace(field[:aliasIdx])
	}
	field = cosmosExpressionWithoutAlias(strings.TrimSpace(field))
	if !cosmosSimplePathExpression(field) {
		return "", false
	}
	field = strings.TrimPrefix(field, "c.")
	if !strings.Contains(field, "(") {
		field = strings.TrimSuffix(field, "]")
		if dot := strings.LastIndex(field, "."); dot >= 0 && dot+1 < len(field) {
			field = field[dot+1:]
		}
		if bracket := strings.LastIndex(field, "["); bracket >= 0 && bracket+1 < len(field) {
			field = field[bracket+1:]
		}
	}
	field = strings.Trim(field, `"'`)
	if field == "" {
		return "", false
	}
	return field, true
}

func cosmosSimplePathExpression(expression string) bool {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		return false
	}
	first := expression[0]
	if first == '{' || first == '[' || first == '\'' || first == '"' || first == '-' || (first >= '0' && first <= '9') {
		return false
	}
	if strings.Contains(expression, "(") || strings.ContainsAny(expression, " +-*/%<>=!&|?,") {
		return false
	}
	return true
}

func splitCosmosExpressionList(input string) []string {
	parts := make([]string, 0)
	start := 0
	depth := 0
	bracketDepth := 0
	braceDepth := 0
	inQuote := false
	var quote rune
	for i, r := range input {
		if inQuote {
			if r == quote {
				inQuote = false
			}
			continue
		}
		switch r {
		case '\'', '"':
			inQuote = true
			quote = r
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case '[':
			bracketDepth++
		case ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
		case '{':
			braceDepth++
		case '}':
			if braceDepth > 0 {
				braceDepth--
			}
		case ',':
			if depth == 0 && bracketDepth == 0 && braceDepth == 0 {
				parts = append(parts, strings.TrimSpace(input[start:i]))
				start = i + 1
			}
		}
	}
	parts = append(parts, strings.TrimSpace(input[start:]))
	return parts
}

func distinctCosmosValues(values []any) []any {
	seen := make(map[string]bool, len(values))
	out := make([]any, 0, len(values))
	for _, value := range values {
		key := cosmosDistinctKey(value)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, value)
	}
	return out
}

func cosmosDistinctKey(value any) string {
	data, err := gojson.Marshal(value)
	if err != nil {
		return fmt.Sprint(value)
	}
	return string(data)
}

func cosmosNumber(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case gojson.Number:
		out, _ := strconv.ParseFloat(string(typed), 64)
		return out
	default:
		out, _ := strconv.ParseFloat(fmt.Sprint(value), 64)
		return out
	}
}

func normalizeCosmosNumber(value float64) any {
	const (
		minInt64Float = -9223372036854775808.0
		maxInt64Float = 9223372036854775807.0
	)
	if value == math.Trunc(value) && value >= minInt64Float && value <= maxInt64Float {
		return int64(value)
	}
	return value
}

func parseCosmosMaxItemCount(raw string) int {
	if raw == "" {
		return -1
	}
	out, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return -1
	}
	return out
}

func decodeCosmosContinuation(raw string) int {
	if raw == "" {
		return 0
	}
	data, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return 0
	}
	out, err := strconv.Atoi(string(data))
	if err != nil {
		return 0
	}
	return out
}

func encodeCosmosContinuation(skip int) string {
	return base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(skip)))
}

func mapFromAny(value any) map[string]any {
	if value == nil {
		return nil
	}
	out, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	return cloneMap(out)
}

func cloneMap(input map[string]any) map[string]any {
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func cloneCosmosJSONMap(input map[string]any) map[string]any {
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = cloneCosmosJSONValue(value)
	}
	return out
}

func cloneCosmosJSONValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneCosmosJSONMap(typed)
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = cloneCosmosJSONValue(item)
		}
		return out
	default:
		return value
	}
}

func defaultCosmosIndexingPolicy() map[string]any {
	return map[string]any{
		"automatic":     true,
		"indexingMode":  "consistent",
		"includedPaths": []any{map[string]any{"path": "/*"}},
		"excludedPaths": []any{map[string]any{"path": `/\"_etag\"/?`}},
	}
}

func isCosmosQueryRequest(req *http.Request) bool {
	return strings.EqualFold(req.Header.Get("x-ms-documentdb-isquery"), "true") || strings.EqualFold(req.Header.Get("Content-Type"), "application/query+json")
}

func isCosmosBatchRequest(req *http.Request) bool {
	return strings.EqualFold(req.Header.Get("x-ms-cosmos-is-batch-request"), "true")
}

func isCosmosQueryPlanRequest(req *http.Request) bool {
	return strings.EqualFold(req.Header.Get("x-ms-cosmos-is-query-plan-request"), "true")
}

func cosmosETagMatches(header, existing string) bool {
	header = strings.TrimSpace(header)
	if header == "" {
		return true
	}
	header = strings.Trim(header, `"`)
	if header == "*" {
		return true
	}
	return header == existing
}

func quotedCosmosETag(etag string) string {
	if etag == "" {
		return ""
	}
	return `"` + etag + `"`
}

func cosmosDataDatabaseKey(account, dbID string) string {
	return strings.ToLower(account) + "|db|" + strings.ToLower(dbID)
}

func cosmosDataCollectionKey(account, dbID, collID string) string {
	return strings.ToLower(account) + "|coll|" + strings.ToLower(dbID) + "|" + strings.ToLower(collID)
}

func cosmosDataDocumentKey(account, dbID, collID, partitionKey, docID string) string {
	return strings.ToLower(account) + "|doc|" + strings.ToLower(dbID) + "|" + strings.ToLower(collID) + "|" + encodeCosmosKey(partitionKey) + "|" + strings.ToLower(docID)
}

func encodeCosmosKey(value string) string {
	if value == "" {
		return "_"
	}
	return base64.RawURLEncoding.EncodeToString([]byte(value))
}

func cosmosSQLJSONResponse(statusCode int, body any, etag string) (*service.Response, error) {
	return cosmosSQLJSONResponseWithHeaders(statusCode, body, etag, nil)
}

func cosmosSQLJSONResponseWithHeaders(statusCode int, body any, etag string, extra map[string]string) (*service.Response, error) {
	data, err := gojson.Marshal(body)
	if err != nil {
		return nil, err
	}
	headers := cosmosSQLHeaders(etag)
	for key, value := range extra {
		headers[key] = value
	}
	return &service.Response{StatusCode: statusCode, RawBody: data, RawContentType: "application/json", Headers: headers}, nil
}

func emptyCosmosSQLResponse(statusCode int) (*service.Response, error) {
	return &service.Response{StatusCode: statusCode, Headers: cosmosSQLHeaders("")}, nil
}

func cosmosSQLHeaders(etag string) map[string]string {
	headers := map[string]string{
		"x-ms-request-charge": "1",
		"x-ms-session-token":  "0:0#1",
		"x-ms-activity-id":    "cloudmock-cosmos",
		"x-ms-version":        cosmosSQLDataPlaneAPIVersion,
		"Cache-Control":       "no-store, no-cache",
		"Pragma":              "no-cache",
	}
	if etag != "" {
		headers["etag"] = quotedCosmosETag(etag)
	}
	return headers
}

func cosmosSQLError(status int, code, message string) (*service.Response, error) {
	return azurearm.JSONResponse(status, map[string]any{"code": code, "message": message})
}

func cosmosSQLNotFound(id string) (*service.Response, error) {
	return cosmosSQLError(http.StatusNotFound, "NotFound", "Resource Not Found: "+id)
}
