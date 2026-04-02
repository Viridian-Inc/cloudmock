package athena

import (
	"encoding/json"
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// ---- JSON helpers ----

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
		return service.NewAWSError("InvalidRequestException", "Invalid JSON in request body.", http.StatusBadRequest)
	}
	return nil
}

// ---- request/response types ----

type tag struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

type workGroupConfigJSON struct {
	ResultConfiguration struct {
		OutputLocation string `json:"OutputLocation"`
	} `json:"ResultConfiguration"`
	EnforceWorkGroupConfiguration bool  `json:"EnforceWorkGroupConfiguration"`
	BytesScannedCutoffPerQuery    int64 `json:"BytesScannedCutoffPerQuery"`
}

type workGroupJSON struct {
	Name          string              `json:"Name"`
	State         string              `json:"State"`
	Description   string              `json:"Description,omitempty"`
	CreationTime  float64             `json:"CreationTime"`
	Configuration workGroupConfigJSON `json:"Configuration"`
}

func toWorkGroupJSON(wg *WorkGroup) workGroupJSON {
	return workGroupJSON{
		Name:         wg.Name,
		State:        wg.State,
		Description:  wg.Description,
		CreationTime: float64(wg.CreationTime.Unix()),
		Configuration: workGroupConfigJSON{
			ResultConfiguration: struct {
				OutputLocation string `json:"OutputLocation"`
			}{OutputLocation: wg.Configuration.ResultOutputLocation},
			EnforceWorkGroupConfiguration: wg.Configuration.EnforceWorkGroupConfiguration,
			BytesScannedCutoffPerQuery:    wg.Configuration.BytesScannedCutoffPerQuery,
		},
	}
}

// ---- CreateWorkGroup ----

type createWorkGroupRequest struct {
	Name          string              `json:"Name"`
	Description   string              `json:"Description"`
	Configuration workGroupConfigJSON `json:"Configuration"`
	Tags          []tag               `json:"Tags"`
}

func handleCreateWorkGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createWorkGroupRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Name == "" {
		return jsonErr(service.ErrValidation("WorkGroup name is required."))
	}
	tags := make(map[string]string)
	for _, t := range req.Tags {
		tags[t.Key] = t.Value
	}
	cfg := WorkGroupConfiguration{
		ResultOutputLocation:       req.Configuration.ResultConfiguration.OutputLocation,
		EnforceWorkGroupConfiguration: req.Configuration.EnforceWorkGroupConfiguration,
		BytesScannedCutoffPerQuery: req.Configuration.BytesScannedCutoffPerQuery,
	}
	_, ok := store.CreateWorkGroup(req.Name, req.Description, tags, cfg)
	if !ok {
		return jsonErr(service.ErrAlreadyExists("WorkGroup", req.Name))
	}
	return jsonOK(struct{}{})
}

// ---- GetWorkGroup ----

type getWorkGroupRequest struct {
	WorkGroup string `json:"WorkGroup"`
}

func handleGetWorkGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getWorkGroupRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.WorkGroup == "" {
		return jsonErr(service.ErrValidation("WorkGroup name is required."))
	}
	wg, ok := store.GetWorkGroup(req.WorkGroup)
	if !ok {
		return jsonErr(service.ErrNotFound("WorkGroup", req.WorkGroup))
	}
	return jsonOK(map[string]any{"WorkGroup": toWorkGroupJSON(wg)})
}

// ---- ListWorkGroups ----

type workGroupSummary struct {
	Name         string  `json:"Name"`
	State        string  `json:"State"`
	Description  string  `json:"Description,omitempty"`
	CreationTime float64 `json:"CreationTime"`
}

func handleListWorkGroups(_ *service.RequestContext, store *Store) (*service.Response, error) {
	wgs := store.ListWorkGroups()
	summaries := make([]workGroupSummary, 0, len(wgs))
	for _, wg := range wgs {
		summaries = append(summaries, workGroupSummary{
			Name:         wg.Name,
			State:        wg.State,
			Description:  wg.Description,
			CreationTime: float64(wg.CreationTime.Unix()),
		})
	}
	return jsonOK(map[string]any{"WorkGroups": summaries})
}

// ---- UpdateWorkGroup ----

type updateWorkGroupRequest struct {
	WorkGroup   string `json:"WorkGroup"`
	Description string `json:"Description"`
	State       string `json:"State"`
}

func handleUpdateWorkGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req updateWorkGroupRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.WorkGroup == "" {
		return jsonErr(service.ErrValidation("WorkGroup name is required."))
	}
	_, ok := store.UpdateWorkGroup(req.WorkGroup, req.Description, req.State)
	if !ok {
		return jsonErr(service.ErrNotFound("WorkGroup", req.WorkGroup))
	}
	return jsonOK(struct{}{})
}

// ---- DeleteWorkGroup ----

type deleteWorkGroupRequest struct {
	WorkGroup string `json:"WorkGroup"`
}

func handleDeleteWorkGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteWorkGroupRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.WorkGroup == "" {
		return jsonErr(service.ErrValidation("WorkGroup name is required."))
	}
	if !store.DeleteWorkGroup(req.WorkGroup) {
		return jsonErr(service.ErrNotFound("WorkGroup", req.WorkGroup))
	}
	return jsonOK(struct{}{})
}

// ---- CreateNamedQuery ----

type createNamedQueryRequest struct {
	Name        string `json:"Name"`
	Description string `json:"Description"`
	Database    string `json:"Database"`
	QueryString string `json:"QueryString"`
	WorkGroup   string `json:"WorkGroup"`
}

func handleCreateNamedQuery(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createNamedQueryRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Name == "" || req.Database == "" || req.QueryString == "" {
		return jsonErr(service.ErrValidation("Name, Database, and QueryString are required."))
	}
	nq := store.CreateNamedQuery(req.Name, req.Description, req.Database, req.QueryString, req.WorkGroup)
	return jsonOK(map[string]string{"NamedQueryId": nq.ID})
}

// ---- GetNamedQuery ----

type getNamedQueryRequest struct {
	NamedQueryId string `json:"NamedQueryId"`
}

type namedQueryJSON struct {
	NamedQueryId string `json:"NamedQueryId"`
	Name         string `json:"Name"`
	Description  string `json:"Description,omitempty"`
	Database     string `json:"Database"`
	QueryString  string `json:"QueryString"`
	WorkGroup    string `json:"WorkGroup"`
}

func handleGetNamedQuery(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getNamedQueryRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.NamedQueryId == "" {
		return jsonErr(service.ErrValidation("NamedQueryId is required."))
	}
	nq, ok := store.GetNamedQuery(req.NamedQueryId)
	if !ok {
		return jsonErr(service.ErrNotFound("NamedQuery", req.NamedQueryId))
	}
	return jsonOK(map[string]any{
		"NamedQuery": namedQueryJSON{
			NamedQueryId: nq.ID,
			Name:         nq.Name,
			Description:  nq.Description,
			Database:     nq.Database,
			QueryString:  nq.QueryString,
			WorkGroup:    nq.WorkGroup,
		},
	})
}

// ---- ListNamedQueries ----

type listNamedQueriesRequest struct {
	WorkGroup string `json:"WorkGroup"`
}

func handleListNamedQueries(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listNamedQueriesRequest
	parseJSON(ctx.Body, &req)
	ids := store.ListNamedQueries(req.WorkGroup)
	return jsonOK(map[string]any{"NamedQueryIds": ids})
}

// ---- DeleteNamedQuery ----

type deleteNamedQueryRequest struct {
	NamedQueryId string `json:"NamedQueryId"`
}

func handleDeleteNamedQuery(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteNamedQueryRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.NamedQueryId == "" {
		return jsonErr(service.ErrValidation("NamedQueryId is required."))
	}
	if !store.DeleteNamedQuery(req.NamedQueryId) {
		return jsonErr(service.ErrNotFound("NamedQuery", req.NamedQueryId))
	}
	return jsonOK(struct{}{})
}

// ---- StartQueryExecution ----

type startQueryExecutionRequest struct {
	QueryString         string `json:"QueryString"`
	QueryExecutionContext struct {
		Database string `json:"Database"`
	} `json:"QueryExecutionContext"`
	ResultConfiguration struct {
		OutputLocation string `json:"OutputLocation"`
	} `json:"ResultConfiguration"`
	WorkGroup string `json:"WorkGroup"`
}

func handleStartQueryExecution(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req startQueryExecutionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.QueryString == "" {
		return jsonErr(service.ErrValidation("QueryString is required."))
	}
	qe := store.StartQueryExecution(req.QueryString, req.QueryExecutionContext.Database, req.WorkGroup, req.ResultConfiguration.OutputLocation)
	return jsonOK(map[string]string{"QueryExecutionId": qe.ID})
}

// ---- GetQueryExecution ----

type getQueryExecutionRequest struct {
	QueryExecutionId string `json:"QueryExecutionId"`
}

type queryExecutionJSON struct {
	QueryExecutionId string `json:"QueryExecutionId"`
	Query            string `json:"Query"`
	Status           struct {
		State            string  `json:"State"`
		StateChangeReason string `json:"StateChangeReason,omitempty"`
		SubmissionDateTime float64 `json:"SubmissionDateTime"`
		CompletionDateTime *float64 `json:"CompletionDateTime,omitempty"`
	} `json:"Status"`
	Statistics struct {
		EngineExecutionTimeInMillis int64 `json:"EngineExecutionTimeInMillis"`
		DataScannedInBytes          int64 `json:"DataScannedInBytes"`
		TotalExecutionTimeInMillis  int64 `json:"TotalExecutionTimeInMillis"`
	} `json:"Statistics"`
	ResultConfiguration struct {
		OutputLocation string `json:"OutputLocation"`
	} `json:"ResultConfiguration"`
	QueryExecutionContext struct {
		Database string `json:"Database,omitempty"`
	} `json:"QueryExecutionContext"`
	WorkGroup string `json:"WorkGroup"`
}

func toQueryExecutionJSON(qe *QueryExecution) queryExecutionJSON {
	j := queryExecutionJSON{
		QueryExecutionId: qe.ID,
		Query:            qe.Query,
		WorkGroup:        qe.WorkGroup,
	}
	j.Status.State = qe.Status.State
	j.Status.StateChangeReason = qe.Status.StateChangeReason
	j.Status.SubmissionDateTime = float64(qe.Status.SubmissionTime.Unix())
	if qe.Status.CompletionTime != nil {
		ct := float64(qe.Status.CompletionTime.Unix())
		j.Status.CompletionDateTime = &ct
	}
	j.Statistics.EngineExecutionTimeInMillis = qe.Statistics.EngineExecutionTimeInMillis
	j.Statistics.DataScannedInBytes = qe.Statistics.DataScannedInBytes
	j.Statistics.TotalExecutionTimeInMillis = qe.Statistics.TotalExecutionTimeInMillis
	j.ResultConfiguration.OutputLocation = qe.ResultConfig.OutputLocation
	j.QueryExecutionContext.Database = qe.Database
	return j
}

func handleGetQueryExecution(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getQueryExecutionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.QueryExecutionId == "" {
		return jsonErr(service.ErrValidation("QueryExecutionId is required."))
	}
	qe, ok := store.GetQueryExecution(req.QueryExecutionId)
	if !ok {
		return jsonErr(service.ErrNotFound("QueryExecution", req.QueryExecutionId))
	}
	return jsonOK(map[string]any{"QueryExecution": toQueryExecutionJSON(qe)})
}

// ---- ListQueryExecutions ----

type listQueryExecutionsRequest struct {
	WorkGroup string `json:"WorkGroup"`
}

func handleListQueryExecutions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listQueryExecutionsRequest
	parseJSON(ctx.Body, &req)
	ids := store.ListQueryExecutions(req.WorkGroup)
	return jsonOK(map[string]any{"QueryExecutionIds": ids})
}

// ---- StopQueryExecution ----

type stopQueryExecutionRequest struct {
	QueryExecutionId string `json:"QueryExecutionId"`
}

func handleStopQueryExecution(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req stopQueryExecutionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.QueryExecutionId == "" {
		return jsonErr(service.ErrValidation("QueryExecutionId is required."))
	}
	if !store.StopQueryExecution(req.QueryExecutionId) {
		return jsonErr(service.ErrNotFound("QueryExecution", req.QueryExecutionId))
	}
	return jsonOK(struct{}{})
}

// ---- GetQueryResults ----

type getQueryResultsRequest struct {
	QueryExecutionId string `json:"QueryExecutionId"`
}

func handleGetQueryResults(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getQueryResultsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.QueryExecutionId == "" {
		return jsonErr(service.ErrValidation("QueryExecutionId is required."))
	}
	qe, ok := store.GetQueryExecution(req.QueryExecutionId)
	if !ok {
		return jsonErr(service.ErrNotFound("QueryExecution", req.QueryExecutionId))
	}
	if qe.Status.State != "SUCCEEDED" {
		return jsonErr(service.NewAWSError("InvalidRequestException",
			"Query has not yet finished. Current state: "+qe.Status.State, http.StatusBadRequest))
	}

	// Return mock result set if available
	rs, hasResults := store.GetQueryResultSet(req.QueryExecutionId)
	if hasResults && rs != nil {
		// Build column info
		columnInfo := make([]map[string]string, len(rs.ColumnInfo))
		for i, ci := range rs.ColumnInfo {
			columnInfo[i] = map[string]string{"Name": ci.Name, "Type": ci.Type}
		}

		// Build rows - first row is header
		rows := make([]map[string]any, 0, len(rs.Rows)+1)
		// Header row
		headerData := make([]map[string]string, len(rs.ColumnInfo))
		for i, ci := range rs.ColumnInfo {
			headerData[i] = map[string]string{"VarCharValue": ci.Name}
		}
		rows = append(rows, map[string]any{"Data": headerData})
		// Data rows
		for _, row := range rs.Rows {
			data := make([]map[string]string, len(row))
			for i, val := range row {
				data[i] = map[string]string{"VarCharValue": val}
			}
			rows = append(rows, map[string]any{"Data": data})
		}

		return jsonOK(map[string]any{
			"ResultSet": map[string]any{
				"Rows":              rows,
				"ResultSetMetadata": map[string]any{"ColumnInfo": columnInfo},
			},
			"UpdateCount": 0,
		})
	}

	// Fallback: empty result set for non-SELECT queries
	return jsonOK(map[string]any{
		"ResultSet": map[string]any{
			"Rows":              []any{},
			"ResultSetMetadata": map[string]any{"ColumnInfo": []any{}},
		},
		"UpdateCount": 0,
	})
}

// ---- TagResource ----

type tagResourceRequest struct {
	ResourceARN string `json:"ResourceARN"`
	Tags        []tag  `json:"Tags"`
}

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req tagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ResourceARN == "" {
		return jsonErr(service.ErrValidation("ResourceARN is required."))
	}
	tags := make(map[string]string)
	for _, t := range req.Tags {
		tags[t.Key] = t.Value
	}
	if !store.TagResource(req.ResourceARN, tags) {
		return jsonErr(service.ErrNotFound("Resource", req.ResourceARN))
	}
	return jsonOK(struct{}{})
}

// ---- UntagResource ----

type untagResourceRequest struct {
	ResourceARN string   `json:"ResourceARN"`
	TagKeys     []string `json:"TagKeys"`
}

func handleUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req untagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ResourceARN == "" {
		return jsonErr(service.ErrValidation("ResourceARN is required."))
	}
	if !store.UntagResource(req.ResourceARN, req.TagKeys) {
		return jsonErr(service.ErrNotFound("Resource", req.ResourceARN))
	}
	return jsonOK(struct{}{})
}

// ---- ListTagsForResource ----

type listTagsForResourceRequest struct {
	ResourceARN string `json:"ResourceARN"`
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listTagsForResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ResourceARN == "" {
		return jsonErr(service.ErrValidation("ResourceARN is required."))
	}
	tags, ok := store.ListTagsForResource(req.ResourceARN)
	if !ok {
		return jsonErr(service.ErrNotFound("Resource", req.ResourceARN))
	}
	tagList := make([]tag, 0, len(tags))
	for k, v := range tags {
		tagList = append(tagList, tag{Key: k, Value: v})
	}
	return jsonOK(map[string]any{"Tags": tagList})
}

// ---- BatchGetNamedQuery ----

type batchGetNamedQueryRequest struct {
	NamedQueryIds []string `json:"NamedQueryIds"`
}

func handleBatchGetNamedQuery(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req batchGetNamedQueryRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	found, notFound := store.BatchGetNamedQuery(req.NamedQueryIds)
	queries := make([]namedQueryJSON, 0, len(found))
	for _, nq := range found {
		queries = append(queries, namedQueryJSON{
			NamedQueryId: nq.ID, Name: nq.Name, Description: nq.Description,
			Database: nq.Database, QueryString: nq.QueryString, WorkGroup: nq.WorkGroup,
		})
	}
	return jsonOK(map[string]any{
		"NamedQueries":    queries,
		"UnprocessedNamedQueryIds": notFound,
	})
}

// ---- BatchGetQueryExecution ----

type batchGetQueryExecutionRequest struct {
	QueryExecutionIds []string `json:"QueryExecutionIds"`
}

func handleBatchGetQueryExecution(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req batchGetQueryExecutionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	found, notFound := store.BatchGetQueryExecution(req.QueryExecutionIds)
	executions := make([]queryExecutionJSON, 0, len(found))
	for _, qe := range found {
		executions = append(executions, toQueryExecutionJSON(qe))
	}
	return jsonOK(map[string]any{
		"QueryExecutions":            executions,
		"UnprocessedQueryExecutionIds": notFound,
	})
}

// ---- CreateDataCatalog ----

type dataCatalogJSON struct {
	Name        string            `json:"Name"`
	Type        string            `json:"Type"`
	Description string            `json:"Description,omitempty"`
	Parameters  map[string]string `json:"Parameters,omitempty"`
}

type createDataCatalogRequest struct {
	Name        string            `json:"Name"`
	Type        string            `json:"Type"`
	Description string            `json:"Description"`
	Parameters  map[string]string `json:"Parameters"`
	Tags        []tag             `json:"Tags"`
}

func handleCreateDataCatalog(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createDataCatalogRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}
	if req.Type == "" {
		req.Type = "GLUE"
	}
	tags := make(map[string]string)
	for _, t := range req.Tags {
		tags[t.Key] = t.Value
	}
	params := req.Parameters
	if params == nil {
		params = make(map[string]string)
	}
	_, ok := store.CreateDataCatalog(req.Name, req.Type, req.Description, params, tags)
	if !ok {
		return jsonErr(service.ErrAlreadyExists("DataCatalog", req.Name))
	}
	return jsonOK(struct{}{})
}

// ---- GetDataCatalog ----

type getDataCatalogRequest struct {
	Name string `json:"Name"`
}

func handleGetDataCatalog(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getDataCatalogRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}
	dc, ok := store.GetDataCatalog(req.Name)
	if !ok {
		return jsonErr(service.ErrNotFound("DataCatalog", req.Name))
	}
	return jsonOK(map[string]any{
		"DataCatalog": dataCatalogJSON{Name: dc.Name, Type: dc.Type, Description: dc.Description, Parameters: dc.Parameters},
	})
}

// ---- ListDataCatalogs ----

func handleListDataCatalogs(_ *service.RequestContext, store *Store) (*service.Response, error) {
	catalogs := store.ListDataCatalogs()
	summaries := make([]dataCatalogJSON, 0, len(catalogs))
	for _, dc := range catalogs {
		summaries = append(summaries, dataCatalogJSON{Name: dc.Name, Type: dc.Type, Description: dc.Description})
	}
	return jsonOK(map[string]any{"DataCatalogsSummary": summaries})
}

// ---- UpdateDataCatalog ----

type updateDataCatalogRequest struct {
	Name        string            `json:"Name"`
	Type        string            `json:"Type"`
	Description string            `json:"Description"`
	Parameters  map[string]string `json:"Parameters"`
}

func handleUpdateDataCatalog(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req updateDataCatalogRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}
	if !store.UpdateDataCatalog(req.Name, req.Description, req.Parameters) {
		return jsonErr(service.ErrNotFound("DataCatalog", req.Name))
	}
	return jsonOK(struct{}{})
}

// ---- DeleteDataCatalog ----

type deleteDataCatalogRequest struct {
	Name string `json:"Name"`
}

func handleDeleteDataCatalog(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteDataCatalogRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}
	if !store.DeleteDataCatalog(req.Name) {
		return jsonErr(service.ErrNotFound("DataCatalog", req.Name))
	}
	return jsonOK(struct{}{})
}
