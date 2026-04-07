package timestreamwrite

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: body, Format: service.FormatJSON}, nil
}

func jsonErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
}

func parseJSON(body []byte, v any) *service.AWSError {
	if len(body) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, v); err != nil {
		return service.NewAWSError("ValidationException", "Invalid JSON.", http.StatusBadRequest)
	}
	return nil
}

type tag struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

// ---- Database JSON ----

type databaseJSON struct {
	Arn             string  `json:"Arn"`
	DatabaseName    string  `json:"DatabaseName"`
	KmsKeyId        string  `json:"KmsKeyId,omitempty"`
	TableCount      int     `json:"TableCount"`
	CreationTime    float64 `json:"CreationTime"`
	LastUpdatedTime float64 `json:"LastUpdatedTime"`
}

func toDBJSON(db *Database) databaseJSON {
	return databaseJSON{
		Arn: db.ARN, DatabaseName: db.Name, KmsKeyId: db.KmsKeyId,
		TableCount: db.TableCount, CreationTime: float64(db.CreationTime.Unix()),
		LastUpdatedTime: float64(db.LastUpdatedTime.Unix()),
	}
}

type createDatabaseRequest struct {
	DatabaseName string `json:"DatabaseName"`
	KmsKeyId     string `json:"KmsKeyId"`
	Tags         []tag  `json:"Tags"`
}

func handleCreateDatabase(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createDatabaseRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.DatabaseName == "" {
		return jsonErr(service.ErrValidation("DatabaseName is required."))
	}
	tags := make(map[string]string)
	for _, t := range req.Tags {
		tags[t.Key] = t.Value
	}
	db, ok := store.CreateDatabase(req.DatabaseName, req.KmsKeyId, tags)
	if !ok {
		return jsonErr(service.NewAWSError("ConflictException", "Database "+req.DatabaseName+" already exists.", http.StatusConflict))
	}
	return jsonOK(map[string]any{"Database": toDBJSON(db)})
}

type describeDatabaseRequest struct {
	DatabaseName string `json:"DatabaseName"`
}

func handleDescribeDatabase(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeDatabaseRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	db, ok := store.GetDatabase(req.DatabaseName)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Database "+req.DatabaseName+" not found.", http.StatusNotFound))
	}
	return jsonOK(map[string]any{"Database": toDBJSON(db)})
}

func handleListDatabases(_ *service.RequestContext, store *Store) (*service.Response, error) {
	dbs := store.ListDatabases()
	list := make([]databaseJSON, 0, len(dbs))
	for _, db := range dbs {
		list = append(list, toDBJSON(db))
	}
	return jsonOK(map[string]any{"Databases": list})
}

type updateDatabaseRequest struct {
	DatabaseName string `json:"DatabaseName"`
	KmsKeyId     string `json:"KmsKeyId"`
}

func handleUpdateDatabase(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req updateDatabaseRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	db, ok := store.UpdateDatabase(req.DatabaseName, req.KmsKeyId)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Database "+req.DatabaseName+" not found.", http.StatusNotFound))
	}
	return jsonOK(map[string]any{"Database": toDBJSON(db)})
}

type deleteDatabaseRequest struct {
	DatabaseName string `json:"DatabaseName"`
}

func handleDeleteDatabase(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteDatabaseRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if !store.DeleteDatabase(req.DatabaseName) {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Database "+req.DatabaseName+" not found.", http.StatusNotFound))
	}
	return jsonOK(struct{}{})
}

// ---- Table JSON ----

type retentionPropertiesJSON struct {
	MemoryStoreRetentionPeriodInHours  int64 `json:"MemoryStoreRetentionPeriodInHours"`
	MagneticStoreRetentionPeriodInDays int64 `json:"MagneticStoreRetentionPeriodInDays"`
}

type magneticStoreWritePropertiesJSON struct {
	EnableMagneticStoreWrites bool `json:"EnableMagneticStoreWrites"`
}

type tableJSON struct {
	Arn                          string                            `json:"Arn"`
	TableName                    string                            `json:"TableName"`
	DatabaseName                 string                            `json:"DatabaseName"`
	TableStatus                  string                            `json:"TableStatus"`
	RetentionProperties          retentionPropertiesJSON           `json:"RetentionProperties"`
	MagneticStoreWriteProperties magneticStoreWritePropertiesJSON  `json:"MagneticStoreWriteProperties"`
	CreationTime                 float64                           `json:"CreationTime"`
	LastUpdatedTime              float64                           `json:"LastUpdatedTime"`
}

func toTableJSON(t *Table) tableJSON {
	return tableJSON{
		Arn: t.ARN, TableName: t.Name, DatabaseName: t.DatabaseName, TableStatus: t.Status,
		RetentionProperties: retentionPropertiesJSON{
			MemoryStoreRetentionPeriodInHours:  t.RetentionProperties.MemoryStoreRetentionPeriodInHours,
			MagneticStoreRetentionPeriodInDays: t.RetentionProperties.MagneticStoreRetentionPeriodInDays,
		},
		MagneticStoreWriteProperties: magneticStoreWritePropertiesJSON{
			EnableMagneticStoreWrites: t.MagneticStoreWriteProperties.EnableMagneticStoreWrites,
		},
		CreationTime: float64(t.CreationTime.Unix()), LastUpdatedTime: float64(t.LastUpdatedTime.Unix()),
	}
}

type createTableRequest struct {
	DatabaseName                 string                            `json:"DatabaseName"`
	TableName                    string                            `json:"TableName"`
	RetentionProperties          *retentionPropertiesJSON          `json:"RetentionProperties"`
	MagneticStoreWriteProperties *magneticStoreWritePropertiesJSON `json:"MagneticStoreWriteProperties"`
	Tags                         []tag                             `json:"Tags"`
}

func handleCreateTable(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createTableRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.DatabaseName == "" || req.TableName == "" {
		return jsonErr(service.ErrValidation("DatabaseName and TableName are required."))
	}
	retention := RetentionProperties{}
	if req.RetentionProperties != nil {
		retention.MemoryStoreRetentionPeriodInHours = req.RetentionProperties.MemoryStoreRetentionPeriodInHours
		retention.MagneticStoreRetentionPeriodInDays = req.RetentionProperties.MagneticStoreRetentionPeriodInDays
	}
	magnetic := MagneticStoreWriteProperties{}
	if req.MagneticStoreWriteProperties != nil {
		magnetic.EnableMagneticStoreWrites = req.MagneticStoreWriteProperties.EnableMagneticStoreWrites
	}
	tags := make(map[string]string)
	for _, t := range req.Tags {
		tags[t.Key] = t.Value
	}
	tbl, ok := store.CreateTable(req.DatabaseName, req.TableName, retention, magnetic, tags)
	if !ok {
		return jsonErr(service.NewAWSError("ConflictException", "Table already exists or database not found.", http.StatusConflict))
	}
	return jsonOK(map[string]any{"Table": toTableJSON(tbl)})
}

type describeTableRequest struct {
	DatabaseName string `json:"DatabaseName"`
	TableName    string `json:"TableName"`
}

func handleDescribeTable(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeTableRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	tbl, ok := store.GetTable(req.DatabaseName, req.TableName)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Table not found.", http.StatusNotFound))
	}
	return jsonOK(map[string]any{"Table": toTableJSON(tbl)})
}

type listTablesRequest struct {
	DatabaseName string `json:"DatabaseName"`
}

func handleListTables(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listTablesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	tables := store.ListTables(req.DatabaseName)
	list := make([]tableJSON, 0, len(tables))
	for _, t := range tables {
		list = append(list, toTableJSON(t))
	}
	return jsonOK(map[string]any{"Tables": list})
}

type updateTableRequest struct {
	DatabaseName                 string                            `json:"DatabaseName"`
	TableName                    string                            `json:"TableName"`
	RetentionProperties          *retentionPropertiesJSON          `json:"RetentionProperties"`
	MagneticStoreWriteProperties *magneticStoreWritePropertiesJSON `json:"MagneticStoreWriteProperties"`
}

func handleUpdateTable(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req updateTableRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	var retention *RetentionProperties
	if req.RetentionProperties != nil {
		retention = &RetentionProperties{
			MemoryStoreRetentionPeriodInHours:  req.RetentionProperties.MemoryStoreRetentionPeriodInHours,
			MagneticStoreRetentionPeriodInDays: req.RetentionProperties.MagneticStoreRetentionPeriodInDays,
		}
	}
	var magnetic *MagneticStoreWriteProperties
	if req.MagneticStoreWriteProperties != nil {
		magnetic = &MagneticStoreWriteProperties{EnableMagneticStoreWrites: req.MagneticStoreWriteProperties.EnableMagneticStoreWrites}
	}
	tbl, ok := store.UpdateTable(req.DatabaseName, req.TableName, retention, magnetic)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Table not found.", http.StatusNotFound))
	}
	return jsonOK(map[string]any{"Table": toTableJSON(tbl)})
}

type deleteTableRequest struct {
	DatabaseName string `json:"DatabaseName"`
	TableName    string `json:"TableName"`
}

func handleDeleteTable(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteTableRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if !store.DeleteTable(req.DatabaseName, req.TableName) {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Table not found.", http.StatusNotFound))
	}
	return jsonOK(struct{}{})
}

// ---- WriteRecords ----

type dimensionJSON struct {
	Name  string `json:"Name"`
	Value string `json:"Value"`
}

type recordJSON struct {
	Dimensions       []dimensionJSON `json:"Dimensions"`
	MeasureName      string          `json:"MeasureName"`
	MeasureValue     string          `json:"MeasureValue"`
	MeasureValueType string          `json:"MeasureValueType"`
	Time             string          `json:"Time"`
	TimeUnit         string          `json:"TimeUnit"`
}

type writeRecordsRequest struct {
	DatabaseName string       `json:"DatabaseName"`
	TableName    string       `json:"TableName"`
	Records      []recordJSON `json:"Records"`
}

func handleWriteRecords(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req writeRecordsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	records := make([]Record, len(req.Records))
	for i, r := range req.Records {
		dims := make([]Dimension, len(r.Dimensions))
		for j, d := range r.Dimensions {
			dims[j] = Dimension{Name: d.Name, Value: d.Value}
		}
		records[i] = Record{
			Dimensions: dims, MeasureName: r.MeasureName, MeasureValue: r.MeasureValue,
			MeasureValueType: r.MeasureValueType, Time: r.Time, TimeUnit: r.TimeUnit,
		}
	}
	if !store.WriteRecords(req.DatabaseName, req.TableName, records) {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Table not found.", http.StatusNotFound))
	}
	return jsonOK(map[string]any{"RecordsIngested": map[string]int{
		"Total": len(records), "MemoryStore": len(records), "MagneticStore": 0,
	}})
}

// ---- Query ----

type queryRequest struct {
	QueryString string `json:"QueryString"`
}

func handleQuery(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req queryRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.QueryString == "" {
		return jsonErr(service.ErrValidation("QueryString is required."))
	}

	// Parse basic SQL to extract database, table, and time range
	// Expected format: SELECT ... FROM "db"."table" WHERE time BETWEEN 'start' AND 'end'
	dbName, tableName := extractDBAndTable(req.QueryString)
	startTime, endTime := extractTimeRange(req.QueryString)

	if dbName == "" || tableName == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"Could not parse database and table from query.", http.StatusBadRequest))
	}

	records, ok := store.QueryRecords(dbName, tableName, startTime, endTime)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Table not found.", http.StatusNotFound))
	}

	// Build column info from first record's dimensions + measure
	columnInfo := []map[string]any{
		{"Name": "time", "Type": map[string]string{"ScalarType": "TIMESTAMP"}},
		{"Name": "measure_name", "Type": map[string]string{"ScalarType": "VARCHAR"}},
		{"Name": "measure_value::double", "Type": map[string]string{"ScalarType": "DOUBLE"}},
	}

	// Add dimension columns from first record
	if len(records) > 0 {
		for _, dim := range records[0].Dimensions {
			columnInfo = append(columnInfo, map[string]any{
				"Name": dim.Name, "Type": map[string]string{"ScalarType": "VARCHAR"},
			})
		}
	}

	// Build rows
	rows := make([]map[string]any, len(records))
	for i, r := range records {
		data := []map[string]any{
			{"ScalarValue": r.Time},
			{"ScalarValue": r.MeasureName},
			{"ScalarValue": r.MeasureValue},
		}
		for _, dim := range r.Dimensions {
			data = append(data, map[string]any{"ScalarValue": dim.Value})
		}
		rows[i] = map[string]any{"Data": data}
	}

	return jsonOK(map[string]any{
		"Rows":       rows,
		"ColumnInfo": columnInfo,
		"QueryId":    newUUID(),
	})
}

// extractDBAndTable parses "db"."table" from a FROM clause.
func extractDBAndTable(query string) (string, string) {
	// Look for FROM "db"."table" or FROM db.table
	upper := strings.ToUpper(query)
	fromIdx := strings.Index(upper, "FROM")
	if fromIdx < 0 {
		return "", ""
	}
	rest := strings.TrimSpace(query[fromIdx+4:])
	// Remove WHERE and everything after
	whereIdx := strings.Index(strings.ToUpper(rest), "WHERE")
	if whereIdx > 0 {
		rest = strings.TrimSpace(rest[:whereIdx])
	}
	// Remove surrounding quotes
	rest = strings.ReplaceAll(rest, "\"", "")
	parts := strings.SplitN(rest, ".", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}

// extractTimeRange parses BETWEEN 'start' AND 'end' from a WHERE clause.
func extractTimeRange(query string) (string, string) {
	upper := strings.ToUpper(query)
	betweenIdx := strings.Index(upper, "BETWEEN")
	if betweenIdx < 0 {
		return "", ""
	}
	rest := query[betweenIdx+7:]
	andIdx := strings.Index(strings.ToUpper(rest), " AND ")
	if andIdx < 0 {
		return "", ""
	}
	start := strings.TrimSpace(rest[:andIdx])
	end := strings.TrimSpace(rest[andIdx+5:])
	// Remove quotes
	start = strings.Trim(start, "'\"")
	end = strings.Trim(end, "'\"")
	return start, end
}

// ---- DescribeEndpoints ----

func handleDescribeEndpoints(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return jsonOK(map[string]any{
		"Endpoints": []map[string]any{
			{"Address": fmt.Sprintf("ingest.timestream.%s.amazonaws.com", store.region), "CachePeriodInMinutes": 1440},
			{"Address": fmt.Sprintf("query.timestream.%s.amazonaws.com", store.region), "CachePeriodInMinutes": 1440},
		},
	})
}

// ---- Tag handlers ----

type tagResourceRequest struct {
	ResourceARN string `json:"ResourceARN"`
	Tags        []tag  `json:"Tags"`
}

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req tagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	tags := make(map[string]string)
	for _, t := range req.Tags {
		tags[t.Key] = t.Value
	}
	if !store.TagResource(req.ResourceARN, tags) {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Resource not found.", http.StatusNotFound))
	}
	return jsonOK(struct{}{})
}

type untagResourceRequest struct {
	ResourceARN string   `json:"ResourceARN"`
	TagKeys     []string `json:"TagKeys"`
}

func handleUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req untagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if !store.UntagResource(req.ResourceARN, req.TagKeys) {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Resource not found.", http.StatusNotFound))
	}
	return jsonOK(struct{}{})
}

type listTagsForResourceRequest struct {
	ResourceARN string `json:"ResourceARN"`
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listTagsForResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	tags, ok := store.ListTagsForResource(req.ResourceARN)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Resource not found.", http.StatusNotFound))
	}
	tagList := make([]tag, 0, len(tags))
	for k, v := range tags {
		tagList = append(tagList, tag{Key: k, Value: v})
	}
	return jsonOK(map[string]any{"Tags": tagList})
}
