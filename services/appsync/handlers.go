package appsync

import (
	"encoding/json"
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: body, Format: service.FormatJSON}, nil
}

func jsonErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
}

func emptyOK() (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: struct{}{}, Format: service.FormatJSON}, nil
}

func parseJSON(body []byte, v any) *service.AWSError {
	if len(body) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, v); err != nil {
		return service.NewAWSError("BadRequestException", "Invalid JSON.", http.StatusBadRequest)
	}
	return nil
}

// ---- CreateGraphqlApi ----

type createGraphqlApiRequest struct {
	Name               string            `json:"name"`
	AuthenticationType string            `json:"authenticationType"`
	Tags               map[string]string `json:"tags"`
	XrayEnabled        bool              `json:"xrayEnabled"`
}

type graphqlApiJSON struct {
	ApiId              string            `json:"apiId"`
	Name               string            `json:"name"`
	Arn                string            `json:"arn"`
	AuthenticationType string            `json:"authenticationType"`
	Uris               map[string]string `json:"uris"`
	Tags               map[string]string `json:"tags,omitempty"`
	XrayEnabled        bool              `json:"xrayEnabled"`
}

type createGraphqlApiResponse struct {
	GraphqlApi graphqlApiJSON `json:"graphqlApi"`
}

func apiToJSON(api *GraphqlApi) graphqlApiJSON {
	return graphqlApiJSON{
		ApiId: api.ApiId, Name: api.Name, Arn: api.ARN,
		AuthenticationType: api.AuthenticationType,
		Uris: api.Uris, Tags: api.Tags, XrayEnabled: api.XrayEnabled,
	}
}

func handleCreateGraphqlApi(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createGraphqlApiRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Name == "" {
		return jsonErr(service.ErrValidation("name is required."))
	}
	api := store.CreateGraphqlApi(req.Name, req.AuthenticationType, req.Tags, req.XrayEnabled)
	return jsonOK(&createGraphqlApiResponse{GraphqlApi: apiToJSON(api)})
}

// ---- GetGraphqlApi ----

type getGraphqlApiRequest struct {
	ApiId string `json:"apiId"`
}

type getGraphqlApiResponse struct {
	GraphqlApi graphqlApiJSON `json:"graphqlApi"`
}

func handleGetGraphqlApi(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getGraphqlApiRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	apiID := req.ApiId
	if apiID == "" {
		apiID = ctx.Params["apiId"]
	}
	if apiID == "" {
		return jsonErr(service.ErrValidation("apiId is required."))
	}
	api, ok := store.GetGraphqlApi(apiID)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "API not found.", http.StatusNotFound))
	}
	return jsonOK(&getGraphqlApiResponse{GraphqlApi: apiToJSON(api)})
}

// ---- ListGraphqlApis ----

type listGraphqlApisResponse struct {
	GraphqlApis []graphqlApiJSON `json:"graphqlApis"`
}

func handleListGraphqlApis(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	apis := store.ListGraphqlApis()
	items := make([]graphqlApiJSON, 0, len(apis))
	for _, api := range apis {
		items = append(items, apiToJSON(api))
	}
	return jsonOK(&listGraphqlApisResponse{GraphqlApis: items})
}

// ---- UpdateGraphqlApi ----

type updateGraphqlApiRequest struct {
	ApiId              string `json:"apiId"`
	Name               string `json:"name"`
	AuthenticationType string `json:"authenticationType"`
	XrayEnabled        *bool  `json:"xrayEnabled"`
}

type updateGraphqlApiResponse struct {
	GraphqlApi graphqlApiJSON `json:"graphqlApi"`
}

func handleUpdateGraphqlApi(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req updateGraphqlApiRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	apiID := req.ApiId
	if apiID == "" {
		apiID = ctx.Params["apiId"]
	}
	if apiID == "" {
		return jsonErr(service.ErrValidation("apiId is required."))
	}
	api, ok := store.UpdateGraphqlApi(apiID, req.Name, req.AuthenticationType, req.XrayEnabled)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "API not found.", http.StatusNotFound))
	}
	return jsonOK(&updateGraphqlApiResponse{GraphqlApi: apiToJSON(api)})
}

// ---- DeleteGraphqlApi ----

type deleteGraphqlApiRequest struct {
	ApiId string `json:"apiId"`
}

func handleDeleteGraphqlApi(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteGraphqlApiRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	apiID := req.ApiId
	if apiID == "" {
		apiID = ctx.Params["apiId"]
	}
	if apiID == "" {
		return jsonErr(service.ErrValidation("apiId is required."))
	}
	if !store.DeleteGraphqlApi(apiID) {
		return jsonErr(service.NewAWSError("NotFoundException", "API not found.", http.StatusNotFound))
	}
	return jsonOK(struct{}{})
}

// ---- CreateDataSource ----

type createDataSourceRequest struct {
	ApiId          string         `json:"apiId"`
	Name           string         `json:"name"`
	Type           string         `json:"type"`
	Description    string         `json:"description"`
	ServiceRoleArn string         `json:"serviceRoleArn"`
	DynamodbConfig map[string]any `json:"dynamodbConfig"`
	LambdaConfig   map[string]any `json:"lambdaConfig"`
	HttpConfig     map[string]any `json:"httpConfig"`
}

type dataSourceJSON struct {
	DataSourceArn  string `json:"dataSourceArn"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	Description    string `json:"description,omitempty"`
	ServiceRoleArn string `json:"serviceRoleArn,omitempty"`
}

type createDataSourceResponse struct {
	DataSource dataSourceJSON `json:"dataSource"`
}

func handleCreateDataSource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createDataSourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	apiID := req.ApiId
	if apiID == "" {
		apiID = ctx.Params["apiId"]
	}
	if apiID == "" || req.Name == "" || req.Type == "" {
		return jsonErr(service.ErrValidation("apiId, name, and type are required."))
	}
	ds, ok := store.CreateDataSource(apiID, req.Name, req.Type, req.Description, req.ServiceRoleArn, req.DynamodbConfig, req.LambdaConfig, req.HttpConfig, nil)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "API not found or data source already exists.", http.StatusNotFound))
	}
	return jsonOK(&createDataSourceResponse{DataSource: dataSourceJSON{DataSourceArn: ds.DataSourceArn, Name: ds.Name, Type: ds.Type, Description: ds.Description, ServiceRoleArn: ds.ServiceRoleArn}})
}

// ---- GetDataSource ----

type getDataSourceRequest struct {
	ApiId string `json:"apiId"`
	Name  string `json:"name"`
}

type getDataSourceResponse struct {
	DataSource dataSourceJSON `json:"dataSource"`
}

func handleGetDataSource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getDataSourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	apiID := req.ApiId
	if apiID == "" {
		apiID = ctx.Params["apiId"]
	}
	name := req.Name
	if name == "" {
		name = ctx.Params["name"]
	}
	if apiID == "" || name == "" {
		return jsonErr(service.ErrValidation("apiId and name are required."))
	}
	ds, ok := store.GetDataSource(apiID, name)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "Data source not found.", http.StatusNotFound))
	}
	return jsonOK(&getDataSourceResponse{DataSource: dataSourceJSON{DataSourceArn: ds.DataSourceArn, Name: ds.Name, Type: ds.Type, Description: ds.Description, ServiceRoleArn: ds.ServiceRoleArn}})
}

// ---- ListDataSources ----

type listDataSourcesRequest struct {
	ApiId string `json:"apiId"`
}

type listDataSourcesResponse struct {
	DataSources []dataSourceJSON `json:"dataSources"`
}

func handleListDataSources(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listDataSourcesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	apiID := req.ApiId
	if apiID == "" {
		apiID = ctx.Params["apiId"]
	}
	if apiID == "" {
		return jsonErr(service.ErrValidation("apiId is required."))
	}
	dss, ok := store.ListDataSources(apiID)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "API not found.", http.StatusNotFound))
	}
	items := make([]dataSourceJSON, 0, len(dss))
	for _, ds := range dss {
		items = append(items, dataSourceJSON{DataSourceArn: ds.DataSourceArn, Name: ds.Name, Type: ds.Type, Description: ds.Description, ServiceRoleArn: ds.ServiceRoleArn})
	}
	return jsonOK(&listDataSourcesResponse{DataSources: items})
}

// ---- UpdateDataSource ----

type updateDataSourceRequest struct {
	ApiId          string `json:"apiId"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	Description    string `json:"description"`
	ServiceRoleArn string `json:"serviceRoleArn"`
}

func handleUpdateDataSource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req updateDataSourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	apiID := req.ApiId
	if apiID == "" {
		apiID = ctx.Params["apiId"]
	}
	name := req.Name
	if name == "" {
		name = ctx.Params["name"]
	}
	if apiID == "" || name == "" {
		return jsonErr(service.ErrValidation("apiId and name are required."))
	}
	ds, ok := store.UpdateDataSource(apiID, name, req.Type, req.Description, req.ServiceRoleArn)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "Data source not found.", http.StatusNotFound))
	}
	return jsonOK(&createDataSourceResponse{DataSource: dataSourceJSON{DataSourceArn: ds.DataSourceArn, Name: ds.Name, Type: ds.Type, Description: ds.Description, ServiceRoleArn: ds.ServiceRoleArn}})
}

// ---- DeleteDataSource ----

type deleteDataSourceRequest struct {
	ApiId string `json:"apiId"`
	Name  string `json:"name"`
}

func handleDeleteDataSource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteDataSourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	apiID := req.ApiId
	if apiID == "" {
		apiID = ctx.Params["apiId"]
	}
	name := req.Name
	if name == "" {
		name = ctx.Params["name"]
	}
	if apiID == "" || name == "" {
		return jsonErr(service.ErrValidation("apiId and name are required."))
	}
	if !store.DeleteDataSource(apiID, name) {
		return jsonErr(service.NewAWSError("NotFoundException", "Data source not found.", http.StatusNotFound))
	}
	return jsonOK(struct{}{})
}

// ---- CreateResolver ----

type createResolverRequest struct {
	ApiId                   string         `json:"apiId"`
	TypeName                string         `json:"typeName"`
	FieldName               string         `json:"fieldName"`
	DataSourceName          string         `json:"dataSourceName"`
	RequestMappingTemplate  string         `json:"requestMappingTemplate"`
	ResponseMappingTemplate string         `json:"responseMappingTemplate"`
	Kind                    string         `json:"kind"`
	PipelineConfig          map[string]any `json:"pipelineConfig"`
	Code                    string         `json:"code"`
}

type resolverJSON struct {
	TypeName                string `json:"typeName"`
	FieldName               string `json:"fieldName"`
	DataSourceName          string `json:"dataSourceName,omitempty"`
	RequestMappingTemplate  string `json:"requestMappingTemplate,omitempty"`
	ResponseMappingTemplate string `json:"responseMappingTemplate,omitempty"`
	Kind                    string `json:"kind"`
}

type createResolverResponse struct {
	Resolver resolverJSON `json:"resolver"`
}

func handleCreateResolver(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createResolverRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	apiID := req.ApiId
	if apiID == "" {
		apiID = ctx.Params["apiId"]
	}
	if apiID == "" || req.TypeName == "" || req.FieldName == "" {
		return jsonErr(service.ErrValidation("apiId, typeName, and fieldName are required."))
	}
	r, ok := store.CreateResolver(apiID, req.TypeName, req.FieldName, req.DataSourceName, req.RequestMappingTemplate, req.ResponseMappingTemplate, req.Kind, req.Code, req.PipelineConfig, nil, nil)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "API not found or resolver already exists.", http.StatusNotFound))
	}
	return jsonOK(&createResolverResponse{Resolver: resolverJSON{TypeName: r.TypeName, FieldName: r.FieldName, DataSourceName: r.DataSourceName, RequestMappingTemplate: r.RequestMappingTemplate, ResponseMappingTemplate: r.ResponseMappingTemplate, Kind: r.Kind}})
}

// ---- GetResolver ----

type getResolverRequest struct {
	ApiId     string `json:"apiId"`
	TypeName  string `json:"typeName"`
	FieldName string `json:"fieldName"`
}

type getResolverResponse struct {
	Resolver resolverJSON `json:"resolver"`
}

func handleGetResolver(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getResolverRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	apiID := req.ApiId
	if apiID == "" {
		apiID = ctx.Params["apiId"]
	}
	typeName := req.TypeName
	if typeName == "" {
		typeName = ctx.Params["typeName"]
	}
	fieldName := req.FieldName
	if fieldName == "" {
		fieldName = ctx.Params["fieldName"]
	}
	if apiID == "" || typeName == "" || fieldName == "" {
		return jsonErr(service.ErrValidation("apiId, typeName, and fieldName are required."))
	}
	r, ok := store.GetResolver(apiID, typeName, fieldName)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "Resolver not found.", http.StatusNotFound))
	}
	return jsonOK(&getResolverResponse{Resolver: resolverJSON{TypeName: r.TypeName, FieldName: r.FieldName, DataSourceName: r.DataSourceName, RequestMappingTemplate: r.RequestMappingTemplate, ResponseMappingTemplate: r.ResponseMappingTemplate, Kind: r.Kind}})
}

// ---- ListResolvers ----

type listResolversRequest struct {
	ApiId    string `json:"apiId"`
	TypeName string `json:"typeName"`
}

type listResolversResponse struct {
	Resolvers []resolverJSON `json:"resolvers"`
}

func handleListResolvers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listResolversRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	apiID := req.ApiId
	if apiID == "" {
		apiID = ctx.Params["apiId"]
	}
	typeName := req.TypeName
	if typeName == "" {
		typeName = ctx.Params["typeName"]
	}
	if apiID == "" {
		return jsonErr(service.ErrValidation("apiId is required."))
	}
	resolvers, ok := store.ListResolvers(apiID, typeName)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "API not found.", http.StatusNotFound))
	}
	items := make([]resolverJSON, 0, len(resolvers))
	for _, r := range resolvers {
		items = append(items, resolverJSON{TypeName: r.TypeName, FieldName: r.FieldName, DataSourceName: r.DataSourceName, Kind: r.Kind})
	}
	return jsonOK(&listResolversResponse{Resolvers: items})
}

// ---- UpdateResolver ----

type updateResolverRequest struct {
	ApiId                   string `json:"apiId"`
	TypeName                string `json:"typeName"`
	FieldName               string `json:"fieldName"`
	DataSourceName          string `json:"dataSourceName"`
	RequestMappingTemplate  string `json:"requestMappingTemplate"`
	ResponseMappingTemplate string `json:"responseMappingTemplate"`
	Kind                    string `json:"kind"`
	Code                    string `json:"code"`
}

func handleUpdateResolver(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req updateResolverRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	apiID := req.ApiId
	if apiID == "" {
		apiID = ctx.Params["apiId"]
	}
	typeName := req.TypeName
	if typeName == "" {
		typeName = ctx.Params["typeName"]
	}
	fieldName := req.FieldName
	if fieldName == "" {
		fieldName = ctx.Params["fieldName"]
	}
	if apiID == "" || typeName == "" || fieldName == "" {
		return jsonErr(service.ErrValidation("apiId, typeName, and fieldName are required."))
	}
	r, ok := store.UpdateResolver(apiID, typeName, fieldName, req.DataSourceName, req.RequestMappingTemplate, req.ResponseMappingTemplate, req.Kind, req.Code)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "Resolver not found.", http.StatusNotFound))
	}
	return jsonOK(&createResolverResponse{Resolver: resolverJSON{TypeName: r.TypeName, FieldName: r.FieldName, DataSourceName: r.DataSourceName, RequestMappingTemplate: r.RequestMappingTemplate, ResponseMappingTemplate: r.ResponseMappingTemplate, Kind: r.Kind}})
}

// ---- DeleteResolver ----

type deleteResolverRequest struct {
	ApiId     string `json:"apiId"`
	TypeName  string `json:"typeName"`
	FieldName string `json:"fieldName"`
}

func handleDeleteResolver(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteResolverRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	apiID := req.ApiId
	if apiID == "" {
		apiID = ctx.Params["apiId"]
	}
	typeName := req.TypeName
	if typeName == "" {
		typeName = ctx.Params["typeName"]
	}
	fieldName := req.FieldName
	if fieldName == "" {
		fieldName = ctx.Params["fieldName"]
	}
	if apiID == "" || typeName == "" || fieldName == "" {
		return jsonErr(service.ErrValidation("apiId, typeName, and fieldName are required."))
	}
	if !store.DeleteResolver(apiID, typeName, fieldName) {
		return jsonErr(service.NewAWSError("NotFoundException", "Resolver not found.", http.StatusNotFound))
	}
	return jsonOK(struct{}{})
}

// ---- CreateFunction ----

type createFunctionRequest struct {
	ApiId                   string         `json:"apiId"`
	Name                    string         `json:"name"`
	Description             string         `json:"description"`
	DataSourceName          string         `json:"dataSourceName"`
	RequestMappingTemplate  string         `json:"requestMappingTemplate"`
	ResponseMappingTemplate string         `json:"responseMappingTemplate"`
	FunctionVersion         string         `json:"functionVersion"`
	Code                    string         `json:"code"`
	Runtime                 map[string]any `json:"runtime"`
}

type functionJSON struct {
	FunctionId      string `json:"functionId"`
	Name            string `json:"name"`
	Description     string `json:"description,omitempty"`
	DataSourceName  string `json:"dataSourceName"`
	FunctionVersion string `json:"functionVersion"`
}

type createFunctionResponse struct {
	FunctionConfiguration functionJSON `json:"functionConfiguration"`
}

func handleCreateFunction(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createFunctionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	apiID := req.ApiId
	if apiID == "" {
		apiID = ctx.Params["apiId"]
	}
	if apiID == "" || req.Name == "" || req.DataSourceName == "" {
		return jsonErr(service.ErrValidation("apiId, name, and dataSourceName are required."))
	}
	f, ok := store.CreateFunction(apiID, req.Name, req.Description, req.DataSourceName, req.RequestMappingTemplate, req.ResponseMappingTemplate, req.FunctionVersion, req.Code, req.Runtime)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "API not found.", http.StatusNotFound))
	}
	return jsonOK(&createFunctionResponse{FunctionConfiguration: functionJSON{FunctionId: f.FunctionId, Name: f.Name, Description: f.Description, DataSourceName: f.DataSourceName, FunctionVersion: f.FunctionVersion}})
}

// ---- GetFunction ----

type getFunctionRequest struct {
	ApiId      string `json:"apiId"`
	FunctionId string `json:"functionId"`
}

type getFunctionResponse struct {
	FunctionConfiguration functionJSON `json:"functionConfiguration"`
}

func handleGetFunction(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getFunctionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	apiID := req.ApiId
	if apiID == "" {
		apiID = ctx.Params["apiId"]
	}
	funcID := req.FunctionId
	if funcID == "" {
		funcID = ctx.Params["functionId"]
	}
	if apiID == "" || funcID == "" {
		return jsonErr(service.ErrValidation("apiId and functionId are required."))
	}
	f, ok := store.GetFunction(apiID, funcID)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "Function not found.", http.StatusNotFound))
	}
	return jsonOK(&getFunctionResponse{FunctionConfiguration: functionJSON{FunctionId: f.FunctionId, Name: f.Name, Description: f.Description, DataSourceName: f.DataSourceName, FunctionVersion: f.FunctionVersion}})
}

// ---- ListFunctions ----

type listFunctionsRequest struct {
	ApiId string `json:"apiId"`
}

type listFunctionsResponse struct {
	Functions []functionJSON `json:"functions"`
}

func handleListFunctions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listFunctionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	apiID := req.ApiId
	if apiID == "" {
		apiID = ctx.Params["apiId"]
	}
	if apiID == "" {
		return jsonErr(service.ErrValidation("apiId is required."))
	}
	funcs, ok := store.ListFunctions(apiID)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "API not found.", http.StatusNotFound))
	}
	items := make([]functionJSON, 0, len(funcs))
	for _, f := range funcs {
		items = append(items, functionJSON{FunctionId: f.FunctionId, Name: f.Name, Description: f.Description, DataSourceName: f.DataSourceName, FunctionVersion: f.FunctionVersion})
	}
	return jsonOK(&listFunctionsResponse{Functions: items})
}

// ---- UpdateFunction ----

type updateFunctionRequest struct {
	ApiId                   string `json:"apiId"`
	FunctionId              string `json:"functionId"`
	Name                    string `json:"name"`
	Description             string `json:"description"`
	DataSourceName          string `json:"dataSourceName"`
	RequestMappingTemplate  string `json:"requestMappingTemplate"`
	ResponseMappingTemplate string `json:"responseMappingTemplate"`
	Code                    string `json:"code"`
}

func handleUpdateFunction(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req updateFunctionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	apiID := req.ApiId
	if apiID == "" {
		apiID = ctx.Params["apiId"]
	}
	funcID := req.FunctionId
	if funcID == "" {
		funcID = ctx.Params["functionId"]
	}
	if apiID == "" || funcID == "" {
		return jsonErr(service.ErrValidation("apiId and functionId are required."))
	}
	f, ok := store.UpdateFunction(apiID, funcID, req.Name, req.Description, req.DataSourceName, req.RequestMappingTemplate, req.ResponseMappingTemplate, req.Code)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "Function not found.", http.StatusNotFound))
	}
	return jsonOK(&createFunctionResponse{FunctionConfiguration: functionJSON{FunctionId: f.FunctionId, Name: f.Name, Description: f.Description, DataSourceName: f.DataSourceName, FunctionVersion: f.FunctionVersion}})
}

// ---- DeleteFunction ----

type deleteFunctionRequest struct {
	ApiId      string `json:"apiId"`
	FunctionId string `json:"functionId"`
}

func handleDeleteFunction(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteFunctionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	apiID := req.ApiId
	if apiID == "" {
		apiID = ctx.Params["apiId"]
	}
	funcID := req.FunctionId
	if funcID == "" {
		funcID = ctx.Params["functionId"]
	}
	if apiID == "" || funcID == "" {
		return jsonErr(service.ErrValidation("apiId and functionId are required."))
	}
	if !store.DeleteFunction(apiID, funcID) {
		return jsonErr(service.NewAWSError("NotFoundException", "Function not found.", http.StatusNotFound))
	}
	return jsonOK(struct{}{})
}

// ---- CreateApiKey ----

type createApiKeyRequest struct {
	ApiId       string `json:"apiId"`
	Description string `json:"description"`
	Expires     int64  `json:"expires"`
}

type apiKeyJSON struct {
	Id          string `json:"id"`
	Description string `json:"description,omitempty"`
	Expires     int64  `json:"expires"`
	Deletes     int64  `json:"deletes"`
}

type createApiKeyResponse struct {
	ApiKey apiKeyJSON `json:"apiKey"`
}

func handleCreateApiKey(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createApiKeyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	apiID := req.ApiId
	if apiID == "" {
		apiID = ctx.Params["apiId"]
	}
	if apiID == "" {
		return jsonErr(service.ErrValidation("apiId is required."))
	}
	key, ok := store.CreateApiKey(apiID, req.Description, req.Expires)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "API not found.", http.StatusNotFound))
	}
	return jsonOK(&createApiKeyResponse{ApiKey: apiKeyJSON{Id: key.Id, Description: key.Description, Expires: key.Expires, Deletes: key.Deletes}})
}

// ---- ListApiKeys ----

type listApiKeysRequest struct {
	ApiId string `json:"apiId"`
}

type listApiKeysResponse struct {
	ApiKeys []apiKeyJSON `json:"apiKeys"`
}

func handleListApiKeys(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listApiKeysRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	apiID := req.ApiId
	if apiID == "" {
		apiID = ctx.Params["apiId"]
	}
	if apiID == "" {
		return jsonErr(service.ErrValidation("apiId is required."))
	}
	keys, ok := store.ListApiKeys(apiID)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "API not found.", http.StatusNotFound))
	}
	items := make([]apiKeyJSON, 0, len(keys))
	for _, k := range keys {
		items = append(items, apiKeyJSON{Id: k.Id, Description: k.Description, Expires: k.Expires, Deletes: k.Deletes})
	}
	return jsonOK(&listApiKeysResponse{ApiKeys: items})
}

// ---- UpdateApiKey ----

type updateApiKeyRequest struct {
	ApiId       string `json:"apiId"`
	Id          string `json:"id"`
	Description string `json:"description"`
	Expires     int64  `json:"expires"`
}

func handleUpdateApiKey(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req updateApiKeyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	apiID := req.ApiId
	if apiID == "" {
		apiID = ctx.Params["apiId"]
	}
	keyID := req.Id
	if keyID == "" {
		keyID = ctx.Params["id"]
	}
	if apiID == "" || keyID == "" {
		return jsonErr(service.ErrValidation("apiId and id are required."))
	}
	key, ok := store.UpdateApiKey(apiID, keyID, req.Description, req.Expires)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "API key not found.", http.StatusNotFound))
	}
	return jsonOK(&createApiKeyResponse{ApiKey: apiKeyJSON{Id: key.Id, Description: key.Description, Expires: key.Expires, Deletes: key.Deletes}})
}

// ---- DeleteApiKey ----

type deleteApiKeyRequest struct {
	ApiId string `json:"apiId"`
	Id    string `json:"id"`
}

func handleDeleteApiKey(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteApiKeyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	apiID := req.ApiId
	if apiID == "" {
		apiID = ctx.Params["apiId"]
	}
	keyID := req.Id
	if keyID == "" {
		keyID = ctx.Params["id"]
	}
	if apiID == "" || keyID == "" {
		return jsonErr(service.ErrValidation("apiId and id are required."))
	}
	if !store.DeleteApiKey(apiID, keyID) {
		return jsonErr(service.NewAWSError("NotFoundException", "API key not found.", http.StatusNotFound))
	}
	return jsonOK(struct{}{})
}

// ---- TagResource ----

type tagResourceRequest struct {
	ResourceArn string            `json:"resourceArn"`
	Tags        map[string]string `json:"tags"`
}

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req tagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := req.ResourceArn
	if arn == "" {
		arn = ctx.Params["resourceArn"]
	}
	if arn == "" {
		return jsonErr(service.ErrValidation("resourceArn is required."))
	}
	if !store.TagResource(arn, req.Tags) {
		return jsonErr(service.NewAWSError("NotFoundException", "Resource not found.", http.StatusNotFound))
	}
	return jsonOK(struct{}{})
}

// ---- UntagResource ----

type untagResourceRequest struct {
	ResourceArn string   `json:"resourceArn"`
	TagKeys     []string `json:"tagKeys"`
}

func handleUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req untagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := req.ResourceArn
	if arn == "" {
		arn = ctx.Params["resourceArn"]
	}
	if arn == "" {
		return jsonErr(service.ErrValidation("resourceArn is required."))
	}
	if !store.UntagResource(arn, req.TagKeys) {
		return jsonErr(service.NewAWSError("NotFoundException", "Resource not found.", http.StatusNotFound))
	}
	return jsonOK(struct{}{})
}

// ---- ListTagsForResource ----

type listTagsRequest struct {
	ResourceArn string `json:"resourceArn"`
}

type listTagsResponse struct {
	Tags map[string]string `json:"tags"`
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listTagsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := req.ResourceArn
	if arn == "" {
		arn = ctx.Params["resourceArn"]
	}
	if arn == "" {
		return jsonErr(service.ErrValidation("resourceArn is required."))
	}
	tags, ok := store.ListTagsForResource(arn)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "Resource not found.", http.StatusNotFound))
	}
	return jsonOK(&listTagsResponse{Tags: tags})
}

// ---- Type handlers ----

type createTypeRequest struct {
	ApiId       string `json:"apiId"`
	TypeName    string `json:"typeName"`
	Definition  string `json:"definition"`
	Format      string `json:"format"`
	Description string `json:"description"`
}

type typeResponse struct {
	Arn         string `json:"arn"`
	Name        string `json:"name"`
	Format      string `json:"format"`
	Definition  string `json:"definition,omitempty"`
	Description string `json:"description,omitempty"`
}

func typeToResp(td *TypeDef) *typeResponse {
	return &typeResponse{Arn: td.ARN, Name: td.Name, Format: td.Format, Definition: td.Definition, Description: td.Description}
}

func handleCreateType(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createTypeRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	apiID := req.ApiId
	if apiID == "" {
		apiID = ctx.Params["apiId"]
	}
	if apiID == "" {
		return jsonErr(service.ErrValidation("apiId is required."))
	}
	if req.Definition == "" || req.Format == "" {
		return jsonErr(service.ErrValidation("definition and format are required."))
	}
	// Prefer explicit typeName, else extract from SDL.
	name := req.TypeName
	if name == "" {
		name = extractTypeName(req.Definition)
	}
	if name == "" {
		name = "UnnamedType"
	}
	td, ok := store.CreateType(apiID, name, req.Format, req.Definition, req.Description)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "API "+apiID+" not found.", http.StatusNotFound))
	}
	return jsonOK(map[string]any{"type": typeToResp(td)})
}

func handleGetType(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createTypeRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	apiID := req.ApiId
	if apiID == "" {
		apiID = ctx.Params["apiId"]
	}
	typeName := req.TypeName
	if typeName == "" {
		typeName = ctx.Params["typeName"]
	}
	format := req.Format
	if format == "" {
		format = ctx.Params["format"]
	}
	td, ok := store.GetType(apiID, typeName, format)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "Type "+typeName+" not found.", http.StatusNotFound))
	}
	return jsonOK(map[string]any{"type": typeToResp(td)})
}

func handleListTypes(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createTypeRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	apiID := req.ApiId
	if apiID == "" {
		apiID = ctx.Params["apiId"]
	}
	format := req.Format
	if format == "" {
		format = ctx.Params["format"]
	}
	types := store.ListTypes(apiID, format)
	result := make([]*typeResponse, 0, len(types))
	for _, td := range types {
		result = append(result, typeToResp(td))
	}
	return jsonOK(map[string]any{"types": result})
}

func handleUpdateType(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createTypeRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	apiID := req.ApiId
	if apiID == "" {
		apiID = ctx.Params["apiId"]
	}
	typeName := req.TypeName
	if typeName == "" {
		typeName = ctx.Params["typeName"]
	}
	td, ok := store.UpdateType(apiID, typeName, req.Format, req.Definition, req.Description)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "Type "+typeName+" not found.", http.StatusNotFound))
	}
	return jsonOK(map[string]any{"type": typeToResp(td)})
}

func handleDeleteType(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createTypeRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	apiID := req.ApiId
	if apiID == "" {
		apiID = ctx.Params["apiId"]
	}
	typeName := req.TypeName
	if typeName == "" {
		typeName = ctx.Params["typeName"]
	}
	if !store.DeleteType(apiID, typeName) {
		return jsonErr(service.NewAWSError("NotFoundException", "Type "+typeName+" not found.", http.StatusNotFound))
	}
	return emptyOK()
}

// ---- Schema handlers ----

type startSchemaCreationRequest struct {
	ApiId      string `json:"apiId"`
	Definition string `json:"definition"`
}

func handleStartSchemaCreation(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req startSchemaCreationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	apiID := req.ApiId
	if apiID == "" {
		apiID = ctx.Params["apiId"]
	}
	if req.Definition == "" {
		return jsonErr(service.ErrValidation("definition is required."))
	}
	if !store.StartSchemaCreation(apiID, req.Definition) {
		return jsonErr(service.NewAWSError("NotFoundException", "API "+apiID+" not found.", http.StatusNotFound))
	}
	return jsonOK(map[string]any{"status": "PROCESSING"})
}

func handleGetSchemaCreationStatus(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req startSchemaCreationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	apiID := req.ApiId
	if apiID == "" {
		apiID = ctx.Params["apiId"]
	}
	sc, ok := store.GetSchemaCreationStatus(apiID)
	if !ok {
		return jsonOK(map[string]any{"status": "NOT_STARTED", "details": ""})
	}
	return jsonOK(map[string]any{"status": sc.Status, "details": sc.Details})
}

// extractTypeName parses a simple SDL definition to find the type name.
func extractTypeName(definition string) string {
	// Look for "type Name" or "input Name" or "interface Name" patterns.
	for i, ch := range definition {
		if ch == ' ' {
			start := i + 1
			for j := start; j < len(definition); j++ {
				if definition[j] == ' ' || definition[j] == '{' || definition[j] == '\n' {
					return definition[start:j]
				}
			}
		}
	}
	return ""
}
