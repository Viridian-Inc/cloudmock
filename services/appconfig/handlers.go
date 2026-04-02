package appconfig

import (
	"encoding/json"
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// ---- helpers ----

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: body, Format: service.FormatJSON}, nil
}

func jsonCreated(body any) (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusCreated, Body: body, Format: service.FormatJSON}, nil
}

func jsonErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
}

func parseJSON(body []byte, v any) *service.AWSError {
	if len(body) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, v); err != nil {
		return service.NewAWSError("BadRequestException", "Invalid JSON in request body.", http.StatusBadRequest)
	}
	return nil
}

func emptyOK() (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusNoContent, Body: struct{}{}, Format: service.FormatJSON}, nil
}

// ---- CreateApplication ----

type createApplicationRequest struct {
	Name        string            `json:"Name"`
	Description string            `json:"Description"`
	Tags        map[string]string `json:"Tags"`
}

type applicationResponse struct {
	ID          string `json:"Id"`
	Name        string `json:"Name"`
	Description string `json:"Description,omitempty"`
}

func handleCreateApplication(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createApplicationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}
	app := store.CreateApplication(req.Name, req.Description, req.Tags)
	return jsonCreated(&applicationResponse{ID: app.ID, Name: app.Name, Description: app.Description})
}

// ---- GetApplication ----

type getApplicationRequest struct {
	ApplicationId string `json:"ApplicationId"`
}

func handleGetApplication(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getApplicationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ApplicationId == "" {
		return jsonErr(service.ErrValidation("ApplicationId is required."))
	}
	app, ok := store.GetApplication(req.ApplicationId)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Application "+req.ApplicationId+" not found.", http.StatusNotFound))
	}
	return jsonOK(&applicationResponse{ID: app.ID, Name: app.Name, Description: app.Description})
}

// ---- ListApplications ----

type listApplicationsResponse struct {
	Items []applicationResponse `json:"Items"`
}

func handleListApplications(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	apps := store.ListApplications()
	items := make([]applicationResponse, 0, len(apps))
	for _, app := range apps {
		items = append(items, applicationResponse{ID: app.ID, Name: app.Name, Description: app.Description})
	}
	return jsonOK(&listApplicationsResponse{Items: items})
}

// ---- UpdateApplication ----

type updateApplicationRequest struct {
	ApplicationId string `json:"ApplicationId"`
	Name          string `json:"Name"`
	Description   string `json:"Description"`
}

func handleUpdateApplication(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req updateApplicationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ApplicationId == "" {
		return jsonErr(service.ErrValidation("ApplicationId is required."))
	}
	app, ok := store.UpdateApplication(req.ApplicationId, req.Name, req.Description)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Application "+req.ApplicationId+" not found.", http.StatusNotFound))
	}
	return jsonOK(&applicationResponse{ID: app.ID, Name: app.Name, Description: app.Description})
}

// ---- DeleteApplication ----

type deleteApplicationRequest struct {
	ApplicationId string `json:"ApplicationId"`
}

func handleDeleteApplication(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteApplicationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ApplicationId == "" {
		return jsonErr(service.ErrValidation("ApplicationId is required."))
	}
	if !store.DeleteApplication(req.ApplicationId) {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Application "+req.ApplicationId+" not found.", http.StatusNotFound))
	}
	return emptyOK()
}

// ---- CreateEnvironment ----

type createEnvironmentRequest struct {
	ApplicationId string            `json:"ApplicationId"`
	Name          string            `json:"Name"`
	Description   string            `json:"Description"`
	Tags          map[string]string `json:"Tags"`
}

type environmentResponse struct {
	ApplicationId string `json:"ApplicationId"`
	ID            string `json:"Id"`
	Name          string `json:"Name"`
	Description   string `json:"Description,omitempty"`
	State         string `json:"State"`
}

func handleCreateEnvironment(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createEnvironmentRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ApplicationId == "" || req.Name == "" {
		return jsonErr(service.ErrValidation("ApplicationId and Name are required."))
	}
	env, ok := store.CreateEnvironment(req.ApplicationId, req.Name, req.Description, req.Tags)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Application "+req.ApplicationId+" not found.", http.StatusNotFound))
	}
	return jsonCreated(&environmentResponse{ApplicationId: env.ApplicationID, ID: env.ID, Name: env.Name, Description: env.Description, State: env.State})
}

// ---- GetEnvironment ----

type getEnvironmentRequest struct {
	ApplicationId string `json:"ApplicationId"`
	EnvironmentId string `json:"EnvironmentId"`
}

func handleGetEnvironment(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getEnvironmentRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ApplicationId == "" || req.EnvironmentId == "" {
		return jsonErr(service.ErrValidation("ApplicationId and EnvironmentId are required."))
	}
	env, ok := store.GetEnvironment(req.ApplicationId, req.EnvironmentId)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Environment not found.", http.StatusNotFound))
	}
	return jsonOK(&environmentResponse{ApplicationId: env.ApplicationID, ID: env.ID, Name: env.Name, Description: env.Description, State: env.State})
}

// ---- ListEnvironments ----

type listEnvironmentsRequest struct {
	ApplicationId string `json:"ApplicationId"`
}

type listEnvironmentsResponse struct {
	Items []environmentResponse `json:"Items"`
}

func handleListEnvironments(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listEnvironmentsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ApplicationId == "" {
		return jsonErr(service.ErrValidation("ApplicationId is required."))
	}
	envs, ok := store.ListEnvironments(req.ApplicationId)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Application "+req.ApplicationId+" not found.", http.StatusNotFound))
	}
	items := make([]environmentResponse, 0, len(envs))
	for _, env := range envs {
		items = append(items, environmentResponse{ApplicationId: env.ApplicationID, ID: env.ID, Name: env.Name, Description: env.Description, State: env.State})
	}
	return jsonOK(&listEnvironmentsResponse{Items: items})
}

// ---- UpdateEnvironment ----

type updateEnvironmentRequest struct {
	ApplicationId string `json:"ApplicationId"`
	EnvironmentId string `json:"EnvironmentId"`
	Name          string `json:"Name"`
	Description   string `json:"Description"`
}

func handleUpdateEnvironment(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req updateEnvironmentRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ApplicationId == "" || req.EnvironmentId == "" {
		return jsonErr(service.ErrValidation("ApplicationId and EnvironmentId are required."))
	}
	env, ok := store.UpdateEnvironment(req.ApplicationId, req.EnvironmentId, req.Name, req.Description)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Environment not found.", http.StatusNotFound))
	}
	return jsonOK(&environmentResponse{ApplicationId: env.ApplicationID, ID: env.ID, Name: env.Name, Description: env.Description, State: env.State})
}

// ---- DeleteEnvironment ----

type deleteEnvironmentRequest struct {
	ApplicationId string `json:"ApplicationId"`
	EnvironmentId string `json:"EnvironmentId"`
}

func handleDeleteEnvironment(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteEnvironmentRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ApplicationId == "" || req.EnvironmentId == "" {
		return jsonErr(service.ErrValidation("ApplicationId and EnvironmentId are required."))
	}
	if !store.DeleteEnvironment(req.ApplicationId, req.EnvironmentId) {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Environment not found.", http.StatusNotFound))
	}
	return emptyOK()
}

// ---- CreateConfigurationProfile ----

type createConfigProfileRequest struct {
	ApplicationId string            `json:"ApplicationId"`
	Name          string            `json:"Name"`
	Description   string            `json:"Description"`
	LocationUri   string            `json:"LocationUri"`
	Type          string            `json:"Type"`
	Tags          map[string]string `json:"Tags"`
}

type configProfileResponse struct {
	ApplicationId string `json:"ApplicationId"`
	ID            string `json:"Id"`
	Name          string `json:"Name"`
	Description   string `json:"Description,omitempty"`
	LocationUri   string `json:"LocationUri"`
	Type          string `json:"Type"`
}

func handleCreateConfigurationProfile(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createConfigProfileRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ApplicationId == "" || req.Name == "" || req.LocationUri == "" {
		return jsonErr(service.ErrValidation("ApplicationId, Name, and LocationUri are required."))
	}
	profile, ok := store.CreateConfigurationProfile(req.ApplicationId, req.Name, req.Description, req.LocationUri, req.Type, req.Tags)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Application "+req.ApplicationId+" not found.", http.StatusNotFound))
	}
	return jsonCreated(&configProfileResponse{ApplicationId: profile.ApplicationID, ID: profile.ID, Name: profile.Name, Description: profile.Description, LocationUri: profile.LocationURI, Type: profile.Type})
}

// ---- GetConfigurationProfile ----

type getConfigProfileRequest struct {
	ApplicationId          string `json:"ApplicationId"`
	ConfigurationProfileId string `json:"ConfigurationProfileId"`
}

func handleGetConfigurationProfile(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getConfigProfileRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ApplicationId == "" || req.ConfigurationProfileId == "" {
		return jsonErr(service.ErrValidation("ApplicationId and ConfigurationProfileId are required."))
	}
	profile, ok := store.GetConfigurationProfile(req.ApplicationId, req.ConfigurationProfileId)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Configuration profile not found.", http.StatusNotFound))
	}
	return jsonOK(&configProfileResponse{ApplicationId: profile.ApplicationID, ID: profile.ID, Name: profile.Name, Description: profile.Description, LocationUri: profile.LocationURI, Type: profile.Type})
}

// ---- ListConfigurationProfiles ----

type listConfigProfilesRequest struct {
	ApplicationId string `json:"ApplicationId"`
}

type listConfigProfilesResponse struct {
	Items []configProfileResponse `json:"Items"`
}

func handleListConfigurationProfiles(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listConfigProfilesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ApplicationId == "" {
		return jsonErr(service.ErrValidation("ApplicationId is required."))
	}
	profiles, ok := store.ListConfigurationProfiles(req.ApplicationId)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Application "+req.ApplicationId+" not found.", http.StatusNotFound))
	}
	items := make([]configProfileResponse, 0, len(profiles))
	for _, p := range profiles {
		items = append(items, configProfileResponse{ApplicationId: p.ApplicationID, ID: p.ID, Name: p.Name, Description: p.Description, LocationUri: p.LocationURI, Type: p.Type})
	}
	return jsonOK(&listConfigProfilesResponse{Items: items})
}

// ---- UpdateConfigurationProfile ----

type updateConfigProfileRequest struct {
	ApplicationId          string `json:"ApplicationId"`
	ConfigurationProfileId string `json:"ConfigurationProfileId"`
	Name                   string `json:"Name"`
	Description            string `json:"Description"`
}

func handleUpdateConfigurationProfile(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req updateConfigProfileRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ApplicationId == "" || req.ConfigurationProfileId == "" {
		return jsonErr(service.ErrValidation("ApplicationId and ConfigurationProfileId are required."))
	}
	profile, ok := store.UpdateConfigurationProfile(req.ApplicationId, req.ConfigurationProfileId, req.Name, req.Description)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Configuration profile not found.", http.StatusNotFound))
	}
	return jsonOK(&configProfileResponse{ApplicationId: profile.ApplicationID, ID: profile.ID, Name: profile.Name, Description: profile.Description, LocationUri: profile.LocationURI, Type: profile.Type})
}

// ---- DeleteConfigurationProfile ----

type deleteConfigProfileRequest struct {
	ApplicationId          string `json:"ApplicationId"`
	ConfigurationProfileId string `json:"ConfigurationProfileId"`
}

func handleDeleteConfigurationProfile(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteConfigProfileRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ApplicationId == "" || req.ConfigurationProfileId == "" {
		return jsonErr(service.ErrValidation("ApplicationId and ConfigurationProfileId are required."))
	}
	if !store.DeleteConfigurationProfile(req.ApplicationId, req.ConfigurationProfileId) {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Configuration profile not found.", http.StatusNotFound))
	}
	return emptyOK()
}

// ---- CreateDeploymentStrategy ----

type createDeploymentStrategyRequest struct {
	Name                        string            `json:"Name"`
	Description                 string            `json:"Description"`
	DeploymentDurationInMinutes int               `json:"DeploymentDurationInMinutes"`
	GrowthFactor                float64           `json:"GrowthFactor"`
	GrowthType                  string            `json:"GrowthType"`
	FinalBakeTimeInMinutes      int               `json:"FinalBakeTimeInMinutes"`
	ReplicateTo                 string            `json:"ReplicateTo"`
	Tags                        map[string]string `json:"Tags"`
}

type deploymentStrategyResponse struct {
	ID                          string  `json:"Id"`
	Name                        string  `json:"Name"`
	Description                 string  `json:"Description,omitempty"`
	DeploymentDurationInMinutes int     `json:"DeploymentDurationInMinutes"`
	GrowthFactor                float64 `json:"GrowthFactor"`
	GrowthType                  string  `json:"GrowthType"`
	FinalBakeTimeInMinutes      int     `json:"FinalBakeTimeInMinutes"`
	ReplicateTo                 string  `json:"ReplicateTo"`
}

func handleCreateDeploymentStrategy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createDeploymentStrategyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}
	ds := store.CreateDeploymentStrategy(req.Name, req.Description, req.DeploymentDurationInMinutes, req.GrowthFactor, req.GrowthType, req.FinalBakeTimeInMinutes, req.ReplicateTo, req.Tags)
	return jsonCreated(&deploymentStrategyResponse{ID: ds.ID, Name: ds.Name, Description: ds.Description, DeploymentDurationInMinutes: ds.DeploymentDurationInMinutes, GrowthFactor: ds.GrowthFactor, GrowthType: ds.GrowthType, FinalBakeTimeInMinutes: ds.FinalBakeTimeInMinutes, ReplicateTo: ds.ReplicateTo})
}

// ---- GetDeploymentStrategy ----

type getDeploymentStrategyRequest struct {
	DeploymentStrategyId string `json:"DeploymentStrategyId"`
}

func handleGetDeploymentStrategy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getDeploymentStrategyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.DeploymentStrategyId == "" {
		return jsonErr(service.ErrValidation("DeploymentStrategyId is required."))
	}
	ds, ok := store.GetDeploymentStrategy(req.DeploymentStrategyId)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Deployment strategy not found.", http.StatusNotFound))
	}
	return jsonOK(&deploymentStrategyResponse{ID: ds.ID, Name: ds.Name, Description: ds.Description, DeploymentDurationInMinutes: ds.DeploymentDurationInMinutes, GrowthFactor: ds.GrowthFactor, GrowthType: ds.GrowthType, FinalBakeTimeInMinutes: ds.FinalBakeTimeInMinutes, ReplicateTo: ds.ReplicateTo})
}

// ---- ListDeploymentStrategies ----

type listDeploymentStrategiesResponse struct {
	Items []deploymentStrategyResponse `json:"Items"`
}

func handleListDeploymentStrategies(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	strategies := store.ListDeploymentStrategies()
	items := make([]deploymentStrategyResponse, 0, len(strategies))
	for _, ds := range strategies {
		items = append(items, deploymentStrategyResponse{ID: ds.ID, Name: ds.Name, Description: ds.Description, DeploymentDurationInMinutes: ds.DeploymentDurationInMinutes, GrowthFactor: ds.GrowthFactor, GrowthType: ds.GrowthType, FinalBakeTimeInMinutes: ds.FinalBakeTimeInMinutes, ReplicateTo: ds.ReplicateTo})
	}
	return jsonOK(&listDeploymentStrategiesResponse{Items: items})
}

// ---- UpdateDeploymentStrategy ----

type updateDeploymentStrategyRequest struct {
	DeploymentStrategyId        string  `json:"DeploymentStrategyId"`
	Name                        string  `json:"Name"`
	Description                 string  `json:"Description"`
	DeploymentDurationInMinutes int     `json:"DeploymentDurationInMinutes"`
	GrowthFactor                float64 `json:"GrowthFactor"`
	GrowthType                  string  `json:"GrowthType"`
	FinalBakeTimeInMinutes      int     `json:"FinalBakeTimeInMinutes"`
}

func handleUpdateDeploymentStrategy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req updateDeploymentStrategyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.DeploymentStrategyId == "" {
		return jsonErr(service.ErrValidation("DeploymentStrategyId is required."))
	}
	ds, ok := store.UpdateDeploymentStrategy(req.DeploymentStrategyId, req.Name, req.Description, req.DeploymentDurationInMinutes, req.GrowthFactor, req.GrowthType, req.FinalBakeTimeInMinutes)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Deployment strategy not found.", http.StatusNotFound))
	}
	return jsonOK(&deploymentStrategyResponse{ID: ds.ID, Name: ds.Name, Description: ds.Description, DeploymentDurationInMinutes: ds.DeploymentDurationInMinutes, GrowthFactor: ds.GrowthFactor, GrowthType: ds.GrowthType, FinalBakeTimeInMinutes: ds.FinalBakeTimeInMinutes, ReplicateTo: ds.ReplicateTo})
}

// ---- DeleteDeploymentStrategy ----

type deleteDeploymentStrategyRequest struct {
	DeploymentStrategyId string `json:"DeploymentStrategyId"`
}

func handleDeleteDeploymentStrategy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteDeploymentStrategyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.DeploymentStrategyId == "" {
		return jsonErr(service.ErrValidation("DeploymentStrategyId is required."))
	}
	if !store.DeleteDeploymentStrategy(req.DeploymentStrategyId) {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Deployment strategy not found.", http.StatusNotFound))
	}
	return emptyOK()
}

// ---- StartDeployment ----

type startDeploymentRequest struct {
	ApplicationId          string            `json:"ApplicationId"`
	EnvironmentId          string            `json:"EnvironmentId"`
	DeploymentStrategyId   string            `json:"DeploymentStrategyId"`
	ConfigurationProfileId string            `json:"ConfigurationProfileId"`
	ConfigurationVersion   string            `json:"ConfigurationVersion"`
	Description            string            `json:"Description"`
	Tags                   map[string]string `json:"Tags"`
}

type deploymentResponse struct {
	ApplicationId          string  `json:"ApplicationId"`
	EnvironmentId          string  `json:"EnvironmentId"`
	DeploymentStrategyId   string  `json:"DeploymentStrategyId"`
	ConfigurationProfileId string  `json:"ConfigurationProfileId"`
	ConfigurationVersion   string  `json:"ConfigurationVersion"`
	DeploymentNumber       int     `json:"DeploymentNumber"`
	State                  string  `json:"State"`
	Description            string  `json:"Description,omitempty"`
	PercentageComplete     float64 `json:"PercentageComplete"`
	StartedAt              string  `json:"StartedAt"`
	CompletedAt            string  `json:"CompletedAt,omitempty"`
}

func deploymentToResponse(dep *Deployment) *deploymentResponse {
	r := &deploymentResponse{
		ApplicationId:          dep.ApplicationID,
		EnvironmentId:          dep.EnvironmentID,
		DeploymentStrategyId:   dep.DeploymentStrategyID,
		ConfigurationProfileId: dep.ConfigurationProfileID,
		ConfigurationVersion:   dep.ConfigurationVersion,
		DeploymentNumber:       dep.DeploymentNumber,
		State:                  dep.State,
		Description:            dep.Description,
		PercentageComplete:     dep.PercentageComplete,
		StartedAt:              dep.StartedAt.Format("2006-01-02T15:04:05Z"),
	}
	if dep.CompletedAt != nil {
		r.CompletedAt = dep.CompletedAt.Format("2006-01-02T15:04:05Z")
	}
	return r
}

func handleStartDeployment(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req startDeploymentRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ApplicationId == "" || req.EnvironmentId == "" || req.DeploymentStrategyId == "" || req.ConfigurationProfileId == "" || req.ConfigurationVersion == "" {
		return jsonErr(service.ErrValidation("ApplicationId, EnvironmentId, DeploymentStrategyId, ConfigurationProfileId, and ConfigurationVersion are required."))
	}
	dep, errKind := store.StartDeployment(req.ApplicationId, req.EnvironmentId, req.DeploymentStrategyId, req.ConfigurationProfileId, req.ConfigurationVersion, req.Description, req.Tags)
	if dep == nil {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Resource not found: "+errKind, http.StatusNotFound))
	}
	return jsonCreated(deploymentToResponse(dep))
}

// ---- GetDeployment ----

type getDeploymentRequest struct {
	ApplicationId    string `json:"ApplicationId"`
	EnvironmentId    string `json:"EnvironmentId"`
	DeploymentNumber int    `json:"DeploymentNumber"`
}

func handleGetDeployment(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getDeploymentRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ApplicationId == "" || req.EnvironmentId == "" || req.DeploymentNumber == 0 {
		return jsonErr(service.ErrValidation("ApplicationId, EnvironmentId, and DeploymentNumber are required."))
	}
	dep, ok := store.GetDeployment(req.ApplicationId, req.EnvironmentId, req.DeploymentNumber)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Deployment not found.", http.StatusNotFound))
	}
	return jsonOK(deploymentToResponse(dep))
}

// ---- ListDeployments ----

type listDeploymentsRequest struct {
	ApplicationId string `json:"ApplicationId"`
	EnvironmentId string `json:"EnvironmentId"`
}

type listDeploymentsResponse struct {
	Items []deploymentResponse `json:"Items"`
}

func handleListDeployments(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listDeploymentsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ApplicationId == "" || req.EnvironmentId == "" {
		return jsonErr(service.ErrValidation("ApplicationId and EnvironmentId are required."))
	}
	deps, ok := store.ListDeployments(req.ApplicationId, req.EnvironmentId)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Environment not found.", http.StatusNotFound))
	}
	items := make([]deploymentResponse, 0, len(deps))
	for _, dep := range deps {
		items = append(items, *deploymentToResponse(dep))
	}
	return jsonOK(&listDeploymentsResponse{Items: items})
}

// ---- StopDeployment ----

type stopDeploymentRequest struct {
	ApplicationId    string `json:"ApplicationId"`
	EnvironmentId    string `json:"EnvironmentId"`
	DeploymentNumber int    `json:"DeploymentNumber"`
}

func handleStopDeployment(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req stopDeploymentRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ApplicationId == "" || req.EnvironmentId == "" || req.DeploymentNumber == 0 {
		return jsonErr(service.ErrValidation("ApplicationId, EnvironmentId, and DeploymentNumber are required."))
	}
	dep, ok := store.StopDeployment(req.ApplicationId, req.EnvironmentId, req.DeploymentNumber)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Deployment not found.", http.StatusNotFound))
	}
	return jsonOK(deploymentToResponse(dep))
}

// ---- TagResource ----

type tagResourceRequest struct {
	ResourceArn string            `json:"ResourceArn"`
	Tags        map[string]string `json:"Tags"`
}

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req tagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ResourceArn == "" {
		return jsonErr(service.ErrValidation("ResourceArn is required."))
	}
	if !store.TagResource(req.ResourceArn, req.Tags) {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Resource not found: "+req.ResourceArn, http.StatusNotFound))
	}
	return emptyOK()
}

// ---- UntagResource ----

type untagResourceRequest struct {
	ResourceArn string   `json:"ResourceArn"`
	TagKeys     []string `json:"TagKeys"`
}

func handleUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req untagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ResourceArn == "" {
		return jsonErr(service.ErrValidation("ResourceArn is required."))
	}
	if !store.UntagResource(req.ResourceArn, req.TagKeys) {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Resource not found: "+req.ResourceArn, http.StatusNotFound))
	}
	return emptyOK()
}

// ---- ListTagsForResource ----

type listTagsRequest struct {
	ResourceArn string `json:"ResourceArn"`
}

type listTagsResponse struct {
	Tags map[string]string `json:"Tags"`
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listTagsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ResourceArn == "" {
		return jsonErr(service.ErrValidation("ResourceArn is required."))
	}
	tags, ok := store.ListTagsForResource(req.ResourceArn)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Resource not found: "+req.ResourceArn, http.StatusNotFound))
	}
	return jsonOK(&listTagsResponse{Tags: tags})
}
