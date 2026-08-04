package storage

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"math"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/azurearm"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

const (
	tableEntitiesDefaultPageSize = 1000
	tableListDefaultPageSize     = 1000
	tableMaxEntityPropertyBytes  = 1024 * 1024
	tableMaxBinaryValueBytes     = 64 * 1024
	tableMaxCustomProperties     = 252
	tableMaxStringValueBytes     = 64 * 1024
	tableSelectMaxProperties     = 255
	tableDateTimeMinYear         = 1601
	tableDateTimeMaxYear         = 9999
)

func (s *StorageService) handleTable(ctx *service.RequestContext, account string) (*service.Response, error) {
	parts := splitPath(ctx.RawRequest.URL.EscapedPath())
	if len(parts) > 0 && strings.HasSuffix(strings.ToLower(parts[0]), "-table") {
		parts = parts[1:]
	}
	if account == "" || len(parts) == 0 {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidUri", "The table request URI is invalid.")
	}
	if len(parts) == 1 && strings.EqualFold(parts[0], "$batch") {
		if ctx.RawRequest.Method != http.MethodPost {
			return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this table route.")
		}
		return s.handleTableBatch(account, ctx)
	}

	path := parts[0]
	if strings.HasPrefix(path, "Tables") {
		switch ctx.RawRequest.Method {
		case http.MethodPost:
			if path == "Tables" {
				return s.createTable(account, ctx.Body)
			}
		case http.MethodGet:
			if path == "Tables" {
				return s.listTables(account, ctx.RawRequest)
			}
		case http.MethodDelete:
			return s.deleteTable(account, extractTableName(path))
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this table route.")
	}

	tableName, entityKey, hasEntityKey := parseTableEntityPath(path)
	if tableName == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidUri", "The table request URI is invalid.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPost:
		if hasEntityKey {
			return s.upsertTableEntity(account, tableName, entityKey.PartitionKey, entityKey.RowKey, ctx.Body, ctx.RawRequest.Header, true)
		}
		return s.insertTableEntity(account, tableName, ctx.Body, ctx.RawRequest.Header)
	case http.MethodGet:
		if hasEntityKey {
			return s.getTableEntity(account, tableName, entityKey.PartitionKey, entityKey.RowKey, ctx.RawRequest)
		}
		return s.queryTableEntities(account, tableName, ctx.RawRequest)
	case http.MethodPut, http.MethodPatch, "MERGE":
		merge := ctx.RawRequest.Method == http.MethodPatch || strings.EqualFold(ctx.RawRequest.Method, "MERGE")
		return s.upsertTableEntity(account, tableName, entityKey.PartitionKey, entityKey.RowKey, ctx.Body, ctx.RawRequest.Header, merge)
	case http.MethodDelete:
		if !hasEntityKey {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidUri", "The entity key is required.")
		}
		return s.deleteTableEntity(account, tableName, entityKey.PartitionKey, entityKey.RowKey, ctx.RawRequest.Header)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this table route.")
	}
}

type tableBatchOperation struct {
	Method    string
	Target    string
	Header    http.Header
	Body      []byte
	ContentID string
}

type tableBatchOperationResult struct {
	StatusCode int
	Headers    map[string]string
	Body       []byte
	ContentID  string
}

type tableBatchEntityRef struct {
	TableName    string
	PartitionKey string
	RowKey       string
}

func (s *StorageService) handleTableBatch(account string, ctx *service.RequestContext) (*service.Response, error) {
	if len(ctx.Body) > 4*1024*1024 {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidInput", "The batch request payload exceeds the maximum supported size.")
	}

	mediaType, params, err := mime.ParseMediaType(ctx.RawRequest.Header.Get("Content-Type"))
	if err != nil || !strings.HasPrefix(strings.ToLower(mediaType), "multipart/") || params["boundary"] == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidInput", "The batch request content type is invalid.")
	}

	operations, err := parseTableBatchOperations(ctx.Body, params["boundary"])
	if err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidInput", "The batch request content was invalid.")
	}
	if len(operations) == 0 {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidInput", "The batch request must include at least one operation.")
	}
	if len(operations) > 100 {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidInput", "The batch request may include at most 100 operations.")
	}
	if err := validateTableBatchOperations(operations); err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidInput", err.Error())
	}

	snapshot := s.cloneTablesForAccount(account)
	results := make([]tableBatchOperationResult, 0, len(operations))
	for _, op := range operations {
		resp, err := s.executeTableBatchOperation(account, op)
		if err != nil {
			s.restoreTablesForAccount(account, snapshot)
			return nil, err
		}
		results = append(results, tableBatchOperationResult{
			StatusCode: resp.StatusCode,
			Headers:    resp.Headers,
			Body:       resp.RawBody,
			ContentID:  op.ContentID,
		})
		if resp.StatusCode >= 400 {
			s.restoreTablesForAccount(account, snapshot)
			break
		}
	}
	return tableBatchResponse(results)
}

func (s *StorageService) createTable(account string, body []byte) (*service.Response, error) {
	var input struct {
		TableName string `json:"TableName"`
	}
	if err := gojson.Unmarshal(body, &input); err != nil || input.TableName == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidInput", "The request content was invalid.")
	}
	if !validTableName(input.TableName) {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidInput", "The table name is invalid.")
	}

	accountKey := strings.ToLower(account)
	tableKey := strings.ToLower(input.TableName)

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.tables[accountKey] == nil {
		s.tables[accountKey] = make(map[string]table)
	}
	if _, exists := s.tables[accountKey][tableKey]; exists {
		return azurearm.ErrorResponse(http.StatusConflict, "TableAlreadyExists", "The table specified already exists.")
	}
	s.tables[accountKey][tableKey] = table{Name: input.TableName, Entities: make(map[string]tableEntity)}
	return tableJSONResponse(http.StatusCreated, map[string]any{"TableName": input.TableName}, nil)
}

func validTableName(name string) bool {
	if len(name) < 3 || len(name) > 63 {
		return false
	}
	if strings.EqualFold(name, "tables") {
		return false
	}
	if !isASCIILetter(name[0]) {
		return false
	}
	for i := 1; i < len(name); i++ {
		if !isASCIILetter(name[i]) && !isASCIIDigit(name[i]) {
			return false
		}
	}
	return true
}

func isASCIILetter(ch byte) bool {
	return (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z')
}

func isASCIIDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

func (s *StorageService) listTables(account string, req *http.Request) (*service.Response, error) {
	query := req.URL.Query()
	if resp, err := validateTableODataQueryOptions(query, "$filter", "$top"); resp != nil || err != nil {
		return resp, err
	}
	filter := query.Get("$filter")
	if resp, err := validateTableFilter(filter); resp != nil || err != nil {
		return resp, err
	}
	top, err := parseTableTop(query.Get("$top"))
	if err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidInput", "The $top query option is invalid.")
	}

	accountKey := strings.ToLower(account)

	s.mu.RLock()
	values := make([]map[string]any, 0, len(s.tables[accountKey]))
	for _, tbl := range s.tables[accountKey] {
		value := map[string]any{"TableName": tbl.Name}
		if tableEntityMatchesFilter(value, filter) {
			values = append(values, value)
		}
	}
	s.mu.RUnlock()

	sort.Slice(values, func(i, j int) bool { return fmt.Sprint(values[i]["TableName"]) < fmt.Sprint(values[j]["TableName"]) })
	start := tableNamePageStart(values, query.Get("NextTableName"))
	if start > 0 {
		values = values[start:]
	}

	headers := tableResponseHeaders("")
	pageLimit := tablePageLimit(top, tableListDefaultPageSize)
	if pageLimit < len(values) {
		headers["x-ms-continuation-NextTableName"] = fmt.Sprint(values[pageLimit]["TableName"])
		values = values[:pageLimit]
	}
	return tableJSONResponse(http.StatusOK, map[string]any{"value": values}, headers)
}

func (s *StorageService) deleteTable(account, tableName string) (*service.Response, error) {
	if tableName == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidInput", "The table name is required.")
	}
	s.mu.Lock()
	delete(s.tables[strings.ToLower(account)], strings.ToLower(tableName))
	s.mu.Unlock()
	return emptyResponse(http.StatusNoContent, tableResponseHeaders(""))
}

func (s *StorageService) insertTableEntity(account, tableName string, body []byte, header http.Header) (*service.Response, error) {
	entity, pk, rk, err := parseTableEntity(body)
	if err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "PropertiesNeedValue", "PartitionKey and RowKey are required.")
	}
	if !validTableEntityKey(pk) || !validTableEntityKey(rk) {
		return invalidTableEntityKeyResponse()
	}
	if !validTableEntityPropertyCount(entity) {
		return invalidTableEntityPropertyCountResponse()
	}
	if !validTableEntityPropertyNames(entity, header.Get("x-ms-version")) {
		return invalidTableEntityPropertyNameResponse()
	}
	if !validTableEntityPropertyValues(entity) {
		return invalidTableEntityPropertyValueResponse()
	}
	if !validTableEntityCombinedPropertySize(entity) {
		return invalidTableEntityCombinedSizeResponse()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	tbl, ok := s.tables[strings.ToLower(account)][strings.ToLower(tableName)]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "TableNotFound", "The specified table does not exist.")
	}
	key := tableEntityKey(pk, rk)
	if _, exists := tbl.Entities[key]; exists {
		return azurearm.ErrorResponse(http.StatusConflict, "EntityAlreadyExists", "The specified entity already exists.")
	}
	stored := s.newTableEntityLocked(entity)
	tbl.Entities[key] = stored
	s.tables[strings.ToLower(account)][strings.ToLower(tableName)] = tbl

	prefer := strings.ToLower(strings.TrimSpace(header.Get("Prefer")))
	if prefer == "return-no-content" {
		headers := tableResponseHeaders(stored.ETag)
		headers["Preference-Applied"] = "return-no-content"
		return emptyResponse(http.StatusNoContent, headers)
	}
	headers := tableResponseHeaders(stored.ETag)
	if prefer == "return-content" {
		headers["Preference-Applied"] = "return-content"
	}
	return tableJSONResponse(http.StatusCreated, tableEntityBody(stored, nil), headers)
}

func (s *StorageService) getTableEntity(account, tableName, partitionKey, rowKey string, req *http.Request) (*service.Response, error) {
	if resp, err := validateTableODataQueryOptions(req.URL.Query(), "$select"); resp != nil || err != nil {
		return resp, err
	}
	selectFields, resp, err := tableSelectFields(req)
	if resp != nil || err != nil {
		return resp, err
	}

	s.mu.RLock()
	entity, ok := s.tables[strings.ToLower(account)][strings.ToLower(tableName)].Entities[tableEntityKey(partitionKey, rowKey)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", "The specified resource does not exist.")
	}
	return tableJSONResponse(http.StatusOK, tableEntityBody(entity, selectFields), tableResponseHeaders(entity.ETag))
}

func (s *StorageService) queryTableEntities(account, tableName string, req *http.Request) (*service.Response, error) {
	query := req.URL.Query()
	if resp, err := validateTableODataQueryOptions(query, "$filter", "$select", "$top"); resp != nil || err != nil {
		return resp, err
	}
	filter := query.Get("$filter")
	if resp, err := validateTableFilter(filter); resp != nil || err != nil {
		return resp, err
	}
	selectFields, resp, err := tableSelectFields(req)
	if resp != nil || err != nil {
		return resp, err
	}
	top, err := parseTableTop(query.Get("$top"))
	if err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidInput", "The $top query option is invalid.")
	}

	s.mu.RLock()
	entities := make([]tableEntity, 0)
	for _, entity := range s.tables[strings.ToLower(account)][strings.ToLower(tableName)].Entities {
		if tableEntityMatchesFilter(entity.Properties, filter) {
			entities = append(entities, entity)
		}
	}
	s.mu.RUnlock()

	sort.Slice(entities, func(i, j int) bool {
		left := fmt.Sprint(entities[i].Properties["PartitionKey"]) + "\x00" + fmt.Sprint(entities[i].Properties["RowKey"])
		right := fmt.Sprint(entities[j].Properties["PartitionKey"]) + "\x00" + fmt.Sprint(entities[j].Properties["RowKey"])
		return left < right
	})

	start := tableEntityPageStart(entities, query.Get("NextPartitionKey"), query.Get("NextRowKey"))
	if start > 0 {
		entities = entities[start:]
	}

	headers := tableResponseHeaders("")
	page := entities
	pageLimit := tablePageLimit(top, tableEntitiesDefaultPageSize)
	if pageLimit < len(page) {
		next := page[pageLimit]
		headers["x-ms-continuation-NextPartitionKey"] = fmt.Sprint(next.Properties["PartitionKey"])
		headers["x-ms-continuation-NextRowKey"] = fmt.Sprint(next.Properties["RowKey"])
		page = page[:pageLimit]
	}

	values := make([]map[string]any, 0, len(page))
	for _, entity := range page {
		values = append(values, tableEntityBody(entity, selectFields))
	}
	return tableJSONResponse(http.StatusOK, map[string]any{"value": values}, headers)
}

func (s *StorageService) upsertTableEntity(account, tableName, partitionKey, rowKey string, body []byte, header http.Header, merge bool) (*service.Response, error) {
	incoming, bodyPK, bodyRK, err := parseTableEntityUpdate(body)
	if err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "PropertiesNeedValue", "PartitionKey and RowKey are required.")
	}
	if partitionKey == "" {
		partitionKey = bodyPK
	}
	if rowKey == "" {
		rowKey = bodyRK
	}
	if partitionKey == "" || rowKey == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "PropertiesNeedValue", "PartitionKey and RowKey are required.")
	}
	if !validTableEntityKey(partitionKey) || !validTableEntityKey(rowKey) {
		return invalidTableEntityKeyResponse()
	}
	incoming["PartitionKey"] = partitionKey
	incoming["RowKey"] = rowKey
	if !validTableEntityPropertyNames(incoming, header.Get("x-ms-version")) {
		return invalidTableEntityPropertyNameResponse()
	}
	if !validTableEntityPropertyValues(incoming) {
		return invalidTableEntityPropertyValueResponse()
	}

	accountKey := strings.ToLower(account)
	tableKey := strings.ToLower(tableName)
	entityKey := tableEntityKey(partitionKey, rowKey)

	s.mu.Lock()
	defer s.mu.Unlock()

	tbl, ok := s.tables[accountKey][tableKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "TableNotFound", "The specified table does not exist.")
	}
	if tableRequiresIfMatchForUpdate(header) {
		return tableMissingIfMatchResponse()
	}
	existing, exists := tbl.Entities[entityKey]
	if !tableETagMatches(header.Get("If-Match"), existing.ETag, exists) {
		return azurearm.ErrorResponse(http.StatusPreconditionFailed, "UpdateConditionNotSatisfied", "The update condition specified in the request was not satisfied.")
	}
	if merge && exists {
		merged := copyTableEntityProperties(existing.Properties)
		for key, value := range incoming {
			merged[key] = value
		}
		incoming = merged
	}
	if !validTableEntityPropertyCount(incoming) {
		return invalidTableEntityPropertyCountResponse()
	}
	if !validTableEntityCombinedPropertySize(incoming) {
		return invalidTableEntityCombinedSizeResponse()
	}
	stored := s.newTableEntityLocked(incoming)
	tbl.Entities[entityKey] = stored
	s.tables[accountKey][tableKey] = tbl
	return emptyResponse(http.StatusNoContent, tableResponseHeaders(stored.ETag))
}

func (s *StorageService) deleteTableEntity(account, tableName, partitionKey, rowKey string, header http.Header) (*service.Response, error) {
	accountKey := strings.ToLower(account)
	tableKey := strings.ToLower(tableName)
	entityKey := tableEntityKey(partitionKey, rowKey)

	s.mu.Lock()
	defer s.mu.Unlock()

	tbl, ok := s.tables[accountKey][tableKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "TableNotFound", "The specified table does not exist.")
	}
	existing, exists := tbl.Entities[entityKey]
	if !exists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", "The specified resource does not exist.")
	}
	if strings.TrimSpace(header.Get("If-Match")) == "" {
		return tableMissingIfMatchResponse()
	}
	if !tableETagMatches(header.Get("If-Match"), existing.ETag, exists) {
		return azurearm.ErrorResponse(http.StatusPreconditionFailed, "UpdateConditionNotSatisfied", "The update condition specified in the request was not satisfied.")
	}
	delete(tbl.Entities, entityKey)
	s.tables[accountKey][tableKey] = tbl
	return emptyResponse(http.StatusNoContent, tableResponseHeaders(""))
}

func tableRequiresIfMatchForUpdate(header http.Header) bool {
	return strings.TrimSpace(header.Get("If-Match")) == "" && queueAPIVersionBefore(header.Get("x-ms-version"), "2011-08-18")
}

func tableMissingIfMatchResponse() (*service.Response, error) {
	return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidInput", "The If-Match header is required for this operation.")
}

func invalidTableEntityKeyResponse() (*service.Response, error) {
	return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidInput", "The PartitionKey or RowKey value is invalid.")
}

func invalidTableEntityPropertyCountResponse() (*service.Response, error) {
	return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidInput", "An entity may include at most 252 custom properties.")
}

func invalidTableEntityPropertyNameResponse() (*service.Response, error) {
	return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidInput", "One or more property names are invalid.")
}

func invalidTableEntityPropertyValueResponse() (*service.Response, error) {
	return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidInput", "One or more property values are invalid.")
}

func invalidTableEntityCombinedSizeResponse() (*service.Response, error) {
	return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidInput", "The combined size of the entity properties exceeds the maximum allowed size.")
}

func validTableEntityKey(key string) bool {
	if key == "" || utf8.RuneCountInString(key) > 1024 {
		return false
	}
	for _, ch := range key {
		if ch == '/' || ch == '\\' || ch == '#' || ch == '?' {
			return false
		}
		if ch <= 0x1F || ch >= 0x7F && ch <= 0x9F {
			return false
		}
	}
	return true
}

func validTableEntityPropertyCount(properties map[string]any) bool {
	customProperties := 0
	for key := range properties {
		if isTableSystemProperty(key) || strings.HasSuffix(key, "@odata.type") {
			continue
		}
		customProperties++
	}
	return customProperties <= tableMaxCustomProperties
}

func validTableEntityPropertyNames(properties map[string]any, apiVersion string) bool {
	disallowDash := !queueAPIVersionBefore(apiVersion, "2009-04-14")
	for key := range properties {
		if isTableSystemProperty(key) || strings.HasSuffix(key, "@odata.type") {
			continue
		}
		if utf8.RuneCountInString(key) > 255 {
			return false
		}
		if disallowDash && strings.Contains(key, "-") {
			return false
		}
	}
	return true
}

func validTableEntityPropertyValues(properties map[string]any) bool {
	for key, value := range properties {
		if strings.HasSuffix(key, "@odata.type") {
			continue
		}
		odataType := strings.ToLower(strings.TrimSpace(stringValue(properties[key+"@odata.type"])))
		text, ok := value.(string)
		switch odataType {
		case "":
			if ok && tableUTF16ByteLen(text) > tableMaxStringValueBytes {
				return false
			}
		case "edm.string":
			if !ok || tableUTF16ByteLen(text) > tableMaxStringValueBytes {
				return false
			}
		case "edm.binary":
			if !ok {
				return false
			}
			decoded, err := base64.StdEncoding.DecodeString(text)
			if err != nil || len(decoded) > tableMaxBinaryValueBytes {
				return false
			}
		case "edm.boolean":
			if _, ok := value.(bool); !ok {
				return false
			}
		case "edm.int32":
			intValue, ok := tableInt64Value(value)
			if !ok || intValue < math.MinInt32 || intValue > math.MaxInt32 {
				return false
			}
		case "edm.int64":
			if _, ok := tableInt64Value(value); !ok {
				return false
			}
		case "edm.double":
			floatValue, ok := tableFloat64Value(value)
			if !ok || math.IsInf(floatValue, 0) || math.IsNaN(floatValue) {
				return false
			}
		case "edm.guid":
			if !ok || !validTableGUID(text) {
				return false
			}
		case "edm.datetime":
			if !ok || !validTableDateTime(text) {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func tableInt64Value(value any) (int64, bool) {
	switch typed := value.(type) {
	case int:
		return int64(typed), true
	case int64:
		return typed, true
	case float64:
		if math.Trunc(typed) != typed || typed < float64(math.MinInt64) || typed > float64(math.MaxInt64) {
			return 0, false
		}
		return int64(typed), true
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func tableFloat64Value(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func validTableGUID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for i, ch := range value {
		switch i {
		case 8, 13, 18, 23:
			if ch != '-' {
				return false
			}
		default:
			if !(ch >= '0' && ch <= '9' || ch >= 'a' && ch <= 'f' || ch >= 'A' && ch <= 'F') {
				return false
			}
		}
	}
	return true
}

func validTableDateTime(value string) bool {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return false
	}
	year := parsed.UTC().Year()
	return year >= tableDateTimeMinYear && year <= tableDateTimeMaxYear
}

func validTableEntityCombinedPropertySize(properties map[string]any) bool {
	total := 0
	for key, value := range properties {
		if strings.HasSuffix(key, "@odata.type") {
			continue
		}
		total += tableUTF16ByteLen(key)
		total += tablePropertyValueByteLen(key, value, properties)
		if total > tableMaxEntityPropertyBytes {
			return false
		}
	}
	return true
}

func tablePropertyValueByteLen(key string, value any, properties map[string]any) int {
	odataType := strings.ToLower(strings.TrimSpace(stringValue(properties[key+"@odata.type"])))
	switch typed := value.(type) {
	case string:
		if odataType == "edm.binary" {
			decoded, err := base64.StdEncoding.DecodeString(typed)
			if err == nil {
				return len(decoded)
			}
			return len(typed)
		}
		if odataType == "edm.guid" {
			return 16
		}
		if odataType == "edm.datetime" {
			return 8
		}
		return tableUTF16ByteLen(typed)
	case bool:
		return 1
	case float64:
		if odataType == "edm.int32" {
			return 4
		}
		return 8
	default:
		return len(fmt.Sprint(value))
	}
}

func tableUTF16ByteLen(value string) int {
	return len(utf16.Encode([]rune(value))) * 2
}

func isTableSystemProperty(key string) bool {
	return key == "PartitionKey" || key == "RowKey" || key == "Timestamp"
}

func copyTableEntityProperties(properties map[string]any) map[string]any {
	copied := make(map[string]any, len(properties))
	for key, value := range properties {
		copied[key] = value
	}
	return copied
}

func (s *StorageService) newTableEntityLocked(properties map[string]any) tableEntity {
	now := time.Now().UTC()
	etag := s.nextToken("table")
	entity := make(map[string]any, len(properties)+1)
	for key, value := range properties {
		entity[key] = value
	}
	entity["Timestamp"] = now.Format("2006-01-02T15:04:05.000Z")
	return tableEntity{Properties: entity, ETag: etag, LastModified: now}
}

func parseTableBatchOperations(body []byte, batchBoundary string) ([]tableBatchOperation, error) {
	reader := multipart.NewReader(bytes.NewReader(body), batchBoundary)
	operations := make([]tableBatchOperation, 0)
	changesets := 0
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		mediaType, params, err := mime.ParseMediaType(part.Header.Get("Content-Type"))
		if err != nil {
			return nil, err
		}
		switch {
		case strings.HasPrefix(strings.ToLower(mediaType), "multipart/"):
			changesets++
			if changesets > 1 || params["boundary"] == "" {
				return nil, fmt.Errorf("invalid changeset")
			}
			changeReader := multipart.NewReader(part, params["boundary"])
			for {
				opPart, err := changeReader.NextPart()
				if err == io.EOF {
					break
				}
				if err != nil {
					return nil, err
				}
				op, err := parseTableBatchOperation(opPart)
				if err != nil {
					return nil, err
				}
				operations = append(operations, op)
			}
		case strings.EqualFold(mediaType, "application/http"):
			op, err := parseTableBatchOperation(part)
			if err != nil {
				return nil, err
			}
			operations = append(operations, op)
		default:
			return nil, fmt.Errorf("unsupported batch part")
		}
	}
	return operations, nil
}

func parseTableBatchOperation(part *multipart.Part) (tableBatchOperation, error) {
	rawBytes, err := io.ReadAll(part)
	if err != nil {
		return tableBatchOperation{}, err
	}
	raw := string(rawBytes)
	headerBlock, body, ok := strings.Cut(raw, "\r\n\r\n")
	if !ok {
		headerBlock, body, ok = strings.Cut(raw, "\n\n")
	}
	if !ok {
		return tableBatchOperation{}, fmt.Errorf("missing embedded request body separator")
	}

	lines := strings.Split(strings.ReplaceAll(headerBlock, "\r\n", "\n"), "\n")
	if len(lines) == 0 {
		return tableBatchOperation{}, fmt.Errorf("missing embedded request line")
	}
	requestLine := strings.Fields(strings.TrimSpace(lines[0]))
	if len(requestLine) < 3 {
		return tableBatchOperation{}, fmt.Errorf("invalid embedded request line")
	}
	header := http.Header{}
	for _, line := range lines[1:] {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		header.Add(strings.TrimSpace(key), strings.TrimSpace(value))
	}

	return tableBatchOperation{
		Method:    requestLine[0],
		Target:    requestLine[1],
		Header:    header,
		Body:      []byte(body),
		ContentID: part.Header.Get("Content-ID"),
	}, nil
}

func validateTableBatchOperations(operations []tableBatchOperation) error {
	partitionKey := ""
	seenEntities := make(map[string]struct{}, len(operations))
	for _, op := range operations {
		ref, ok := tableBatchOperationEntityRef(op)
		if !ok {
			continue
		}
		if partitionKey == "" {
			partitionKey = ref.PartitionKey
		}
		if ref.PartitionKey != partitionKey {
			return fmt.Errorf("All operations in a batch changeset must use the same PartitionKey.")
		}
		entityKey := strings.ToLower(ref.TableName) + "\x00" + ref.PartitionKey + "\x00" + ref.RowKey
		if _, exists := seenEntities[entityKey]; exists {
			return fmt.Errorf("An entity can appear only once in a batch changeset.")
		}
		seenEntities[entityKey] = struct{}{}
	}
	return nil
}

func tableBatchOperationEntityRef(op tableBatchOperation) (tableBatchEntityRef, bool) {
	path := tableBatchOperationTablePath(op)
	if path == "" || strings.HasPrefix(path, "Tables") || strings.EqualFold(path, "$batch") {
		return tableBatchEntityRef{}, false
	}
	tableName, entityKey, hasEntityKey := parseTableEntityPath(path)
	partitionKey := entityKey.PartitionKey
	rowKey := entityKey.RowKey
	if !hasEntityKey || partitionKey == "" || rowKey == "" {
		_, bodyPK, bodyRK, err := parseTableEntityUpdate(op.Body)
		if err != nil {
			return tableBatchEntityRef{}, false
		}
		if partitionKey == "" {
			partitionKey = bodyPK
		}
		if rowKey == "" {
			rowKey = bodyRK
		}
	}
	if tableName == "" || partitionKey == "" || rowKey == "" {
		return tableBatchEntityRef{}, false
	}
	return tableBatchEntityRef{TableName: tableName, PartitionKey: partitionKey, RowKey: rowKey}, true
}

func tableBatchOperationTablePath(op tableBatchOperation) string {
	target := op.Target
	if parsed, err := url.Parse(target); err == nil {
		target = parsed.EscapedPath()
	}
	parts := splitPath(target)
	if len(parts) > 0 && strings.HasSuffix(strings.ToLower(parts[0]), "-table") {
		parts = parts[1:]
	}
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func (s *StorageService) executeTableBatchOperation(account string, op tableBatchOperation) (*service.Response, error) {
	target := op.Target
	if strings.HasPrefix(target, "/") {
		target = "https://" + account + ".table.core.windows.net" + target
	}
	req, err := http.NewRequest(op.Method, target, bytes.NewReader(op.Body))
	if err != nil {
		return nil, err
	}
	req.Header = op.Header.Clone()
	return s.handleTable(&service.RequestContext{
		AccountID:  account,
		RawRequest: req,
		Body:       op.Body,
	}, account)
}

func (s *StorageService) cloneTablesForAccount(account string) map[string]table {
	accountKey := strings.ToLower(account)
	s.mu.RLock()
	defer s.mu.RUnlock()

	source := s.tables[accountKey]
	clone := make(map[string]table, len(source))
	for tableName, tbl := range source {
		entities := make(map[string]tableEntity, len(tbl.Entities))
		for key, entity := range tbl.Entities {
			properties := make(map[string]any, len(entity.Properties))
			for property, value := range entity.Properties {
				properties[property] = value
			}
			entities[key] = tableEntity{
				Properties:   properties,
				ETag:         entity.ETag,
				LastModified: entity.LastModified,
			}
		}
		clone[tableName] = table{Name: tbl.Name, Entities: entities}
	}
	return clone
}

func (s *StorageService) restoreTablesForAccount(account string, snapshot map[string]table) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tables[strings.ToLower(account)] = snapshot
}

func parseTableEntity(body []byte) (map[string]any, string, string, error) {
	var entity map[string]any
	if err := gojson.Unmarshal(body, &entity); err != nil {
		return nil, "", "", err
	}
	omitNullTableProperties(entity)
	pk := stringValue(entity["PartitionKey"])
	rk := stringValue(entity["RowKey"])
	if pk == "" || rk == "" {
		return nil, "", "", fmt.Errorf("missing key")
	}
	return entity, pk, rk, nil
}

func parseTableEntityUpdate(body []byte) (map[string]any, string, string, error) {
	var entity map[string]any
	if err := gojson.Unmarshal(body, &entity); err != nil {
		return nil, "", "", err
	}
	omitNullTableProperties(entity)
	return entity, stringValue(entity["PartitionKey"]), stringValue(entity["RowKey"]), nil
}

func omitNullTableProperties(entity map[string]any) {
	nullProperties := make([]string, 0)
	for key, value := range entity {
		if value == nil {
			nullProperties = append(nullProperties, key)
		}
	}
	for _, key := range nullProperties {
		delete(entity, key)
		delete(entity, key+"@odata.type")
	}
}

func parseTableEntityPath(path string) (string, tableEntityIdentifier, bool) {
	if !strings.Contains(path, "(") {
		return strings.TrimSuffix(path, "()"), tableEntityIdentifier{}, false
	}
	tableName := path[:strings.Index(path, "(")]
	inside := path[strings.Index(path, "(")+1 : strings.LastIndex(path, ")")]
	if strings.TrimSpace(inside) == "" {
		return tableName, tableEntityIdentifier{}, false
	}
	return tableName, tableEntityIdentifier{
		PartitionKey: extractODataKeyValue(inside, "PartitionKey"),
		RowKey:       extractODataKeyValue(inside, "RowKey"),
	}, true
}

type tableEntityIdentifier struct {
	PartitionKey string
	RowKey       string
}

func extractTableName(path string) string {
	if !strings.Contains(path, "(") {
		return ""
	}
	inside := path[strings.Index(path, "(")+1 : strings.LastIndex(path, ")")]
	return strings.Trim(strings.TrimSpace(inside), "'")
}

func extractODataKeyValue(input, name string) string {
	for _, part := range strings.Split(input, ",") {
		key, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok || !strings.EqualFold(strings.TrimSpace(key), name) {
			continue
		}
		return strings.ReplaceAll(strings.Trim(strings.TrimSpace(value), "'"), "''", "'")
	}
	return ""
}

func tableEntityKey(partitionKey, rowKey string) string {
	return partitionKey + "\x00" + rowKey
}

func tableEntityBody(entity tableEntity, selectFields []string) map[string]any {
	out := make(map[string]any)
	if len(selectFields) == 0 {
		for key, value := range entity.Properties {
			out[key] = value
		}
	} else {
		out["PartitionKey"] = entity.Properties["PartitionKey"]
		out["RowKey"] = entity.Properties["RowKey"]
		out["Timestamp"] = entity.Properties["Timestamp"]
		for _, field := range selectFields {
			if value, ok := entity.Properties[field]; ok {
				out[field] = value
			} else {
				out[field] = nil
			}
		}
	}
	out["odata.etag"] = entity.ETag
	return out
}

func parseTableSelect(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	fields := make([]string, 0, len(parts))
	for _, part := range parts {
		if field := strings.TrimSpace(part); field != "" {
			fields = append(fields, field)
		}
	}
	return fields
}

func tableSelectFields(req *http.Request) ([]string, *service.Response, error) {
	selectRaw := req.URL.Query().Get("$select")
	if strings.TrimSpace(selectRaw) != "" && queueAPIVersionBefore(req.Header.Get("x-ms-version"), "2011-08-18") {
		resp, err := azurearm.ErrorResponse(http.StatusBadRequest, "InvalidInput", "The $select query option requires x-ms-version 2011-08-18 or later.")
		return nil, resp, err
	}
	fields := parseTableSelect(selectRaw)
	if len(fields) > tableSelectMaxProperties {
		resp, err := azurearm.ErrorResponse(http.StatusBadRequest, "InvalidInput", "The $select query option may return at most 255 properties.")
		return nil, resp, err
	}
	return fields, nil, nil
}

func parseTableTop(raw string) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return 0, nil
	}
	top, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || top < 0 {
		return 0, fmt.Errorf("invalid top")
	}
	return top, nil
}

func tablePageLimit(top, defaultLimit int) int {
	if top <= 0 {
		return defaultLimit
	}
	if top > defaultLimit {
		return defaultLimit
	}
	return top
}

func tableNamePageStart(values []map[string]any, nextTableName string) int {
	if nextTableName == "" {
		return 0
	}
	for i, value := range values {
		if fmt.Sprint(value["TableName"]) >= nextTableName {
			return i
		}
	}
	return len(values)
}

func tableEntityPageStart(entities []tableEntity, nextPartitionKey, nextRowKey string) int {
	if nextPartitionKey == "" {
		return 0
	}
	for i, entity := range entities {
		partitionKey := fmt.Sprint(entity.Properties["PartitionKey"])
		rowKey := fmt.Sprint(entity.Properties["RowKey"])
		if partitionKey > nextPartitionKey || partitionKey == nextPartitionKey && (nextRowKey == "" || rowKey >= nextRowKey) {
			return i
		}
	}
	return len(entities)
}

func tableEntityMatchesFilter(properties map[string]any, filter string) bool {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return true
	}
	for _, group := range splitTableFilterOrClauses(filter) {
		if tableEntityMatchesFilterAndGroup(properties, group) {
			return true
		}
	}
	return false
}

func tableEntityMatchesFilterAndGroup(properties map[string]any, filter string) bool {
	for _, clause := range splitTableFilterAndClauses(filter) {
		if !tableEntityMatchesFilterClause(properties, clause) {
			return false
		}
	}
	return true
}

func splitTableFilterOrClauses(filter string) []string {
	parts := strings.Split(filter, " or ")
	clauses := make([]string, 0, len(parts))
	for _, part := range parts {
		clause := strings.TrimSpace(part)
		clause = strings.TrimPrefix(clause, "(")
		clause = strings.TrimSuffix(clause, ")")
		if clause != "" {
			clauses = append(clauses, clause)
		}
	}
	if len(clauses) == 0 {
		return []string{filter}
	}
	return clauses
}

func splitTableFilterAndClauses(filter string) []string {
	parts := strings.Split(filter, " and ")
	clauses := make([]string, 0, len(parts))
	for _, part := range parts {
		clause := strings.TrimSpace(part)
		clause = strings.TrimPrefix(clause, "(")
		clause = strings.TrimSuffix(clause, ")")
		if clause != "" {
			clauses = append(clauses, clause)
		}
	}
	if len(clauses) == 0 {
		return []string{filter}
	}
	return clauses
}

func tableEntityMatchesFilterClause(properties map[string]any, filter string) bool {
	filter = strings.TrimSpace(filter)
	if strings.HasPrefix(strings.ToLower(filter), "not ") {
		inner := strings.TrimSpace(filter[len("not "):])
		inner = strings.TrimPrefix(inner, "(")
		inner = strings.TrimSuffix(inner, ")")
		return !tableEntityMatchesFilterClause(properties, inner)
	}
	parts := strings.Fields(filter)
	if len(parts) == 1 {
		value, ok := booleanValue(properties[strings.Trim(parts[0], "()")])
		return ok && value
	}
	if len(parts) < 3 {
		return true
	}
	field := parts[0]
	op := strings.ToLower(parts[1])
	rawValue := strings.Join(parts[2:], " ")
	rawValue = strings.Trim(rawValue, "'")
	rawValue = tableFilterLiteralValue(rawValue)
	actual, ok := properties[field]
	if !ok {
		return false
	}
	switch op {
	case "eq":
		return fmt.Sprint(actual) == rawValue
	case "ne":
		return fmt.Sprint(actual) != rawValue
	case "gt", "ge", "lt", "le":
		actualFloat, actualOK := numericValue(actual)
		expectedFloat, err := strconv.ParseFloat(rawValue, 64)
		if actualOK && err == nil {
			switch op {
			case "gt":
				return actualFloat > expectedFloat
			case "ge":
				return actualFloat >= expectedFloat
			case "lt":
				return actualFloat < expectedFloat
			case "le":
				return actualFloat <= expectedFloat
			}
		}
		comparison := strings.Compare(fmt.Sprint(actual), rawValue)
		switch op {
		case "gt":
			return comparison > 0
		case "ge":
			return comparison >= 0
		case "lt":
			return comparison < 0
		case "le":
			return comparison <= 0
		}
	}
	return true
}

func tableFilterLiteralValue(value string) string {
	for _, prefix := range []string{"datetime", "guid"} {
		if strings.HasPrefix(value, prefix+"'") {
			typed := strings.TrimSuffix(strings.TrimPrefix(value, prefix+"'"), "'")
			return strings.ReplaceAll(typed, "''", "'")
		}
	}
	return strings.ReplaceAll(value, "''", "'")
}

func validateTableFilter(filter string) (*service.Response, error) {
	if tableFilterComparisonCount(filter) > 15 {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidInput", "The $filter query option may include at most 15 discrete comparisons.")
	}
	if tableFilterContainsNullLiteral(filter) {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidInput", "The $filter query option cannot contain null values.")
	}
	if tableFilterContainsDynamicRightSide(filter) {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidInput", "The right side of a $filter comparison must be a constant value.")
	}
	return nil, nil
}

func validateTableODataQueryOptions(query url.Values, allowed ...string) (*service.Response, error) {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, key := range allowed {
		allowedSet[strings.ToLower(key)] = struct{}{}
	}
	for key := range query {
		if !strings.HasPrefix(key, "$") {
			continue
		}
		if _, ok := allowedSet[strings.ToLower(key)]; !ok {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidInput", "The table query option is not supported.")
		}
	}
	return nil, nil
}

func tableFilterContainsNullLiteral(filter string) bool {
	for _, token := range tableFilterTokensOutsideQuotes(filter) {
		if strings.EqualFold(token, "null") {
			return true
		}
	}
	return false
}

func tableFilterContainsDynamicRightSide(filter string) bool {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return false
	}
	for _, group := range splitTableFilterOrClauses(filter) {
		for _, clause := range splitTableFilterAndClauses(group) {
			clause = strings.TrimSpace(clause)
			if strings.HasPrefix(strings.ToLower(clause), "not ") {
				clause = strings.TrimSpace(clause[len("not "):])
				clause = strings.TrimPrefix(clause, "(")
				clause = strings.TrimSuffix(clause, ")")
			}
			parts := strings.Fields(clause)
			if len(parts) < 3 || !tableFilterComparisonOperator(parts[1]) {
				continue
			}
			if !tableFilterRightSideIsConstant(strings.Join(parts[2:], " ")) {
				return true
			}
		}
	}
	return false
}

func tableFilterComparisonOperator(op string) bool {
	switch strings.ToLower(op) {
	case "eq", "ne", "gt", "ge", "lt", "le":
		return true
	default:
		return false
	}
}

func tableFilterRightSideIsConstant(raw string) bool {
	raw = strings.TrimSpace(raw)
	raw = strings.Trim(raw, "()")
	if raw == "" {
		return false
	}
	lower := strings.ToLower(raw)
	if lower == "true" || lower == "false" {
		return true
	}
	if _, err := strconv.ParseFloat(raw, 64); err == nil {
		return true
	}
	for _, prefix := range []string{"datetime", "guid"} {
		if strings.HasPrefix(lower, prefix+"'") && strings.HasSuffix(raw, "'") {
			return true
		}
	}
	return strings.HasPrefix(raw, "'") && strings.HasSuffix(raw, "'")
}

func tableFilterComparisonCount(filter string) int {
	count := 0
	for _, token := range tableFilterTokensOutsideQuotes(filter) {
		switch strings.ToLower(token) {
		case "eq", "ne", "gt", "ge", "lt", "le":
			count++
		}
	}
	return count
}

func tableFilterTokensOutsideQuotes(filter string) []string {
	if strings.TrimSpace(filter) == "" {
		return nil
	}
	var outside strings.Builder
	inQuote := false
	for i := 0; i < len(filter); i++ {
		ch := filter[i]
		if ch == '\'' {
			if inQuote && i+1 < len(filter) && filter[i+1] == '\'' {
				i++
				continue
			}
			inQuote = !inQuote
			outside.WriteByte(' ')
			continue
		}
		if inQuote {
			continue
		}
		switch ch {
		case '(', ')', ',':
			outside.WriteByte(' ')
		default:
			outside.WriteByte(ch)
		}
	}
	return strings.Fields(outside.String())
}

func booleanValue(value any) (bool, bool) {
	switch typed := value.(type) {
	case bool:
		return typed, true
	case string:
		parsed, err := strconv.ParseBool(strings.ToLower(typed))
		return parsed, err == nil
	default:
		parsed, err := strconv.ParseBool(strings.ToLower(fmt.Sprint(value)))
		return parsed, err == nil
	}
}

func numericValue(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case jsonNumber:
		out, err := strconv.ParseFloat(string(typed), 64)
		return out, err == nil
	default:
		out, err := strconv.ParseFloat(fmt.Sprint(value), 64)
		return out, err == nil
	}
}

type jsonNumber string

func tableETagMatches(header, existingETag string, exists bool) bool {
	if strings.TrimSpace(header) == "" {
		return true
	}
	clean := strings.Trim(strings.TrimSpace(header), `"`)
	if clean == "*" {
		return exists
	}
	return exists && clean == existingETag
}

func tableResponseHeaders(etag string) map[string]string {
	headers := map[string]string{
		"x-ms-version":           dataPlaneAPIVersion,
		"DataServiceVersion":     "3.0;",
		"x-ms-request-id":        "cloudmock-table",
		"Cache-Control":          "no-cache",
		"X-Content-Type-Options": "nosniff",
	}
	if etag != "" {
		headers["ETag"] = etag
	}
	return headers
}

func tableJSONResponse(statusCode int, body any, headers map[string]string) (*service.Response, error) {
	data, err := gojson.Marshal(body)
	if err != nil {
		return nil, err
	}
	if headers == nil {
		headers = tableResponseHeaders("")
	}
	return &service.Response{
		StatusCode:     statusCode,
		RawBody:        data,
		RawContentType: "application/json",
		Headers:        headers,
	}, nil
}

func tableBatchResponse(results []tableBatchOperationResult) (*service.Response, error) {
	batchBoundary := "batchresponse_cloudmock"
	changesetBoundary := "changesetresponse_cloudmock"
	var body strings.Builder

	fmt.Fprintf(&body, "--%s\r\n", batchBoundary)
	fmt.Fprintf(&body, "Content-Type: multipart/mixed; boundary=%s\r\n\r\n", changesetBoundary)
	for index, result := range results {
		fmt.Fprintf(&body, "--%s\r\n", changesetBoundary)
		body.WriteString("Content-Type: application/http\r\n")
		body.WriteString("Content-Transfer-Encoding: binary\r\n\r\n")
		reason := http.StatusText(result.StatusCode)
		if reason == "" {
			reason = "Status"
		}
		fmt.Fprintf(&body, "HTTP/1.1 %d %s\r\n", result.StatusCode, reason)
		contentID := result.ContentID
		if contentID == "" {
			contentID = strconv.Itoa(index + 1)
		}
		fmt.Fprintf(&body, "Content-ID: %s\r\n", contentID)
		keys := make([]string, 0, len(result.Headers))
		for key := range result.Headers {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			fmt.Fprintf(&body, "%s: %s\r\n", key, result.Headers[key])
		}
		body.WriteString("\r\n")
		if len(result.Body) > 0 {
			body.Write(result.Body)
			body.WriteString("\r\n")
		}
	}
	fmt.Fprintf(&body, "--%s--\r\n", changesetBoundary)
	fmt.Fprintf(&body, "--%s--\r\n", batchBoundary)

	return &service.Response{
		StatusCode:     http.StatusAccepted,
		RawBody:        []byte(body.String()),
		RawContentType: "multipart/mixed; boundary=" + batchBoundary,
		Headers:        tableResponseHeaders(""),
	}, nil
}
