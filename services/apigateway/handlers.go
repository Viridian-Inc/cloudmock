package apigateway

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
)

// ---- JSON request/response types ----

type createRestApiRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type restApiResponse struct {
	Id          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedDate time.Time `json:"createdDate"`
}

type restApisListResponse struct {
	Items []*restApiResponse `json:"items"`
}

type createResourceRequest struct {
	PathPart string `json:"pathPart"`
}

type resourceResponse struct {
	Id           string `json:"id"`
	ParentId     string `json:"parentId,omitempty"`
	PathPart     string `json:"pathPart"`
	Path         string `json:"path"`
}

type resourcesListResponse struct {
	Items []*resourceResponse `json:"items"`
}

type putMethodRequest struct {
	AuthorizationType string `json:"authorizationType"`
	HttpMethod        string `json:"httpMethod"`
}

type methodResponse struct {
	HttpMethod        string       `json:"httpMethod"`
	AuthorizationType string       `json:"authorizationType"`
	MethodIntegration *Integration `json:"methodIntegration,omitempty"`
}

type putIntegrationRequest struct {
	Type       string `json:"type"`
	Uri        string `json:"uri"`
	HttpMethod string `json:"httpMethod"`
	IntegrationHttpMethod string `json:"integrationHttpMethod,omitempty"`
}

type createDeploymentRequest struct {
	StageName   string `json:"stageName"`
	Description string `json:"description"`
}

type deploymentResponse struct {
	Id          string    `json:"id"`
	CreatedDate time.Time `json:"createdDate"`
	Description string    `json:"description"`
}

type deploymentsListResponse struct {
	Items []*deploymentResponse `json:"items"`
}

type createStageRequest struct {
	StageName    string `json:"stageName"`
	DeploymentId string `json:"deploymentId"`
	Description  string `json:"description"`
}

type stageResponse struct {
	StageName    string    `json:"stageName"`
	DeploymentId string    `json:"deploymentId"`
	Description  string    `json:"description"`
	CreatedDate  time.Time `json:"createdDate"`
}

type stagesListResponse struct {
	Item []*stageResponse `json:"item"`
}

// ---- helpers ----

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       body,
		Format:     service.FormatJSON,
	}, nil
}

func jsonCreated(body any) (*service.Response, error) {
	return &service.Response{
		StatusCode: http.StatusCreated,
		Body:       body,
		Format:     service.FormatJSON,
	}, nil
}

func jsonNoContent() (*service.Response, error) {
	return &service.Response{
		StatusCode: http.StatusNoContent,
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
		return service.NewAWSError("BadRequestException",
			"Request body is not valid JSON.", http.StatusBadRequest)
	}
	return nil
}

func apiToResponse(api *RestApi) *restApiResponse {
	return &restApiResponse{
		Id:          api.Id,
		Name:        api.Name,
		Description: api.Description,
		CreatedDate: api.CreatedDate,
	}
}

func resourceToResponse(r *Resource) *resourceResponse {
	return &resourceResponse{
		Id:       r.Id,
		ParentId: r.ParentId,
		PathPart: r.PathPart,
		Path:     r.Path,
	}
}

func deploymentToResponse(d *Deployment) *deploymentResponse {
	return &deploymentResponse{
		Id:          d.Id,
		CreatedDate: d.CreatedDate,
		Description: d.Description,
	}
}

func stageToResponse(st *Stage) *stageResponse {
	return &stageResponse{
		StageName:    st.StageName,
		DeploymentId: st.DeploymentId,
		Description:  st.Description,
		CreatedDate:  st.CreatedDate,
	}
}

// ---- handler functions ----

func handleCreateRestApi(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createRestApiRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Name == "" {
		return jsonErr(service.NewAWSError("BadRequestException",
			"name is required.", http.StatusBadRequest))
	}
	api := store.CreateRestApi(req.Name, req.Description)
	return jsonCreated(apiToResponse(api))
}

func handleGetRestApis(_ *service.RequestContext, store *Store) (*service.Response, error) {
	apis := store.ListRestApis()
	items := make([]*restApiResponse, 0, len(apis))
	for _, api := range apis {
		items = append(items, apiToResponse(api))
	}
	return jsonOK(restApisListResponse{Items: items})
}

func handleGetRestApi(_ *service.RequestContext, store *Store, apiID string) (*service.Response, error) {
	api, awsErr := store.GetRestApi(apiID)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(apiToResponse(api))
}

func handleDeleteRestApi(_ *service.RequestContext, store *Store, apiID string) (*service.Response, error) {
	if awsErr := store.DeleteRestApi(apiID); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonNoContent()
}

func handleGetResources(_ *service.RequestContext, store *Store, apiID string) (*service.Response, error) {
	resources, awsErr := store.GetResources(apiID)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	items := make([]*resourceResponse, 0, len(resources))
	for _, r := range resources {
		items = append(items, resourceToResponse(r))
	}
	return jsonOK(resourcesListResponse{Items: items})
}

func handleCreateResource(ctx *service.RequestContext, store *Store, apiID, parentID string) (*service.Response, error) {
	var req createResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.PathPart == "" {
		return jsonErr(service.NewAWSError("BadRequestException",
			"pathPart is required.", http.StatusBadRequest))
	}
	r, awsErr := store.CreateResource(apiID, parentID, req.PathPart)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonCreated(resourceToResponse(r))
}

func handleDeleteResource(_ *service.RequestContext, store *Store, apiID, resourceID string) (*service.Response, error) {
	if awsErr := store.DeleteResource(apiID, resourceID); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonNoContent()
}

func handlePutMethod(ctx *service.RequestContext, store *Store, apiID, resourceID, httpMethod string) (*service.Response, error) {
	var req putMethodRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.AuthorizationType == "" {
		req.AuthorizationType = "NONE"
	}
	m, awsErr := store.PutMethod(apiID, resourceID, httpMethod, req.AuthorizationType)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(methodResponse{
		HttpMethod:        m.HttpMethod,
		AuthorizationType: m.AuthorizationType,
	})
}

func handleGetMethod(_ *service.RequestContext, store *Store, apiID, resourceID, httpMethod string) (*service.Response, error) {
	m, awsErr := store.GetMethod(apiID, resourceID, httpMethod)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(methodResponse{
		HttpMethod:        m.HttpMethod,
		AuthorizationType: m.AuthorizationType,
		MethodIntegration: m.Integration,
	})
}

func handlePutIntegration(ctx *service.RequestContext, store *Store, apiID, resourceID, httpMethod string) (*service.Response, error) {
	var req putIntegrationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Type == "" {
		return jsonErr(service.NewAWSError("BadRequestException",
			"type is required.", http.StatusBadRequest))
	}
	integration := &Integration{
		Type:                  req.Type,
		Uri:                   req.Uri,
		HttpMethod:            req.HttpMethod,
		IntegrationHttpMethod: req.IntegrationHttpMethod,
	}
	result, awsErr := store.PutIntegration(apiID, resourceID, httpMethod, integration)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(result)
}

func handleCreateDeployment(ctx *service.RequestContext, store *Store, apiID string) (*service.Response, error) {
	var req createDeploymentRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	d, awsErr := store.CreateDeployment(apiID, req.Description)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonCreated(deploymentToResponse(d))
}

func handleGetDeployments(_ *service.RequestContext, store *Store, apiID string) (*service.Response, error) {
	deployments, awsErr := store.GetDeployments(apiID)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	items := make([]*deploymentResponse, 0, len(deployments))
	for _, d := range deployments {
		items = append(items, deploymentToResponse(d))
	}
	return jsonOK(deploymentsListResponse{Items: items})
}

func handleCreateStage(ctx *service.RequestContext, store *Store, apiID string) (*service.Response, error) {
	var req createStageRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.StageName == "" {
		return jsonErr(service.NewAWSError("BadRequestException",
			"stageName is required.", http.StatusBadRequest))
	}
	if req.DeploymentId == "" {
		return jsonErr(service.NewAWSError("BadRequestException",
			"deploymentId is required.", http.StatusBadRequest))
	}
	st, awsErr := store.CreateStage(apiID, req.StageName, req.DeploymentId, req.Description)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonCreated(stageToResponse(st))
}

func handleGetStages(_ *service.RequestContext, store *Store, apiID string) (*service.Response, error) {
	stages, awsErr := store.GetStages(apiID)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	items := make([]*stageResponse, 0, len(stages))
	for _, st := range stages {
		items = append(items, stageToResponse(st))
	}
	return jsonOK(stagesListResponse{Item: items})
}
