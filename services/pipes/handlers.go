package pipes

import (
	gojson "github.com/goccy/go-json"
	"net/http"

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
	if err := gojson.Unmarshal(body, v); err != nil {
		return service.NewAWSError("ValidationException", "Invalid JSON in request body.", http.StatusBadRequest)
	}
	return nil
}

// ---- CreatePipe ----

type createPipeRequest struct {
	Name             string         `json:"Name"`
	Description      string         `json:"Description"`
	Source           string         `json:"Source"`
	SourceParameters map[string]any `json:"SourceParameters"`
	Target           string         `json:"Target"`
	TargetParameters map[string]any `json:"TargetParameters"`
	RoleArn          string         `json:"RoleArn"`
	Enrichment       string         `json:"Enrichment"`
	DesiredState     string         `json:"DesiredState"`
	Tags             map[string]string `json:"Tags"`
}

type pipeResponse struct {
	Arn              string `json:"Arn"`
	Name             string `json:"Name"`
	Description      string `json:"Description,omitempty"`
	DesiredState     string `json:"DesiredState"`
	CurrentState     string `json:"CurrentState"`
	Source           string `json:"Source"`
	Target           string `json:"Target"`
	RoleArn          string `json:"RoleArn,omitempty"`
	Enrichment       string `json:"Enrichment,omitempty"`
	CreationTime     string `json:"CreationTime"`
	LastModifiedTime string `json:"LastModifiedTime"`
}

func pipeToResponse(p *Pipe) *pipeResponse {
	return &pipeResponse{
		Arn: p.ARN, Name: p.Name, Description: p.Description,
		DesiredState: p.DesiredState, CurrentState: p.CurrentState,
		Source: p.Source, Target: p.Target,
		RoleArn: p.RoleArn, Enrichment: p.Enrichment,
		CreationTime:     p.CreationTime.Format("2006-01-02T15:04:05Z"),
		LastModifiedTime: p.LastModifiedTime.Format("2006-01-02T15:04:05Z"),
	}
}

func handleCreatePipe(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createPipeRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := req.Name
	if name == "" {
		name = ctx.Params["Name"]
	}
	if name == "" || req.Source == "" || req.Target == "" || req.RoleArn == "" {
		return jsonErr(service.ErrValidation("Name, Source, Target, and RoleArn are required."))
	}
	pipe, ok := store.CreatePipe(name, req.Description, req.Source, req.Target, req.RoleArn, req.Enrichment, req.DesiredState, req.SourceParameters, req.TargetParameters, req.Tags)
	if !ok {
		return jsonErr(service.NewAWSError("ConflictException", "Pipe "+name+" already exists.", http.StatusConflict))
	}
	return jsonOK(pipeToResponse(pipe))
}

// ---- DescribePipe ----

type describePipeRequest struct {
	Name string `json:"Name"`
}

func handleDescribePipe(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describePipeRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := req.Name
	if name == "" {
		name = ctx.Params["Name"]
	}
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}
	pipe, ok := store.DescribePipe(name)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "Pipe "+name+" not found.", http.StatusNotFound))
	}
	return jsonOK(pipeToResponse(pipe))
}

// ---- ListPipes ----

type listPipesRequest struct {
	NamePrefix   string `json:"NamePrefix"`
	CurrentState string `json:"CurrentState"`
	DesiredState string `json:"DesiredState"`
	SourcePrefix string `json:"SourcePrefix"`
}

type pipeSummaryJSON struct {
	Arn          string `json:"Arn"`
	Name         string `json:"Name"`
	DesiredState string `json:"DesiredState"`
	CurrentState string `json:"CurrentState"`
	Source       string `json:"Source"`
	Target       string `json:"Target"`
	CreationTime string `json:"CreationTime"`
}

type listPipesResponse struct {
	Pipes []pipeSummaryJSON `json:"Pipes"`
}

func handleListPipes(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listPipesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	pipes := store.ListPipes(req.NamePrefix, req.CurrentState, req.DesiredState, req.SourcePrefix)
	items := make([]pipeSummaryJSON, 0, len(pipes))
	for _, p := range pipes {
		items = append(items, pipeSummaryJSON{
			Arn: p.ARN, Name: p.Name, DesiredState: p.DesiredState,
			CurrentState: p.CurrentState, Source: p.Source, Target: p.Target,
			CreationTime: p.CreationTime.Format("2006-01-02T15:04:05Z"),
		})
	}
	return jsonOK(&listPipesResponse{Pipes: items})
}

// ---- UpdatePipe ----

type updatePipeRequest struct {
	Name             string         `json:"Name"`
	Description      string         `json:"Description"`
	Target           string         `json:"Target"`
	TargetParameters map[string]any `json:"TargetParameters"`
	RoleArn          string         `json:"RoleArn"`
	Enrichment       string         `json:"Enrichment"`
	DesiredState     string         `json:"DesiredState"`
}

type updatePipeResponse struct {
	Arn              string `json:"Arn"`
	Name             string `json:"Name"`
	DesiredState     string `json:"DesiredState"`
	CurrentState     string `json:"CurrentState"`
	CreationTime     string `json:"CreationTime"`
	LastModifiedTime string `json:"LastModifiedTime"`
}

func handleUpdatePipe(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req updatePipeRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := req.Name
	if name == "" {
		name = ctx.Params["Name"]
	}
	if name == "" || req.RoleArn == "" {
		return jsonErr(service.ErrValidation("Name and RoleArn are required."))
	}
	pipe, ok := store.UpdatePipe(name, req.Description, req.Target, req.RoleArn, req.Enrichment, req.DesiredState, req.TargetParameters)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "Pipe "+name+" not found.", http.StatusNotFound))
	}
	return jsonOK(&updatePipeResponse{
		Arn: pipe.ARN, Name: pipe.Name, DesiredState: pipe.DesiredState,
		CurrentState: pipe.CurrentState,
		CreationTime:     pipe.CreationTime.Format("2006-01-02T15:04:05Z"),
		LastModifiedTime: pipe.LastModifiedTime.Format("2006-01-02T15:04:05Z"),
	})
}

// ---- DeletePipe ----

type deletePipeRequest struct {
	Name string `json:"Name"`
}

func handleDeletePipe(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deletePipeRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := req.Name
	if name == "" {
		name = ctx.Params["Name"]
	}
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}
	pipe, ok := store.DeletePipe(name)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "Pipe "+name+" not found.", http.StatusNotFound))
	}
	return jsonOK(pipeToResponse(pipe))
}

// ---- StartPipe ----

type startPipeRequest struct {
	Name string `json:"Name"`
}

func handleStartPipe(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req startPipeRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := req.Name
	if name == "" {
		name = ctx.Params["Name"]
	}
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}
	pipe, ok := store.StartPipe(name)
	if !ok {
		return jsonErr(service.NewAWSError("ConflictException", "Pipe "+name+" cannot be started in current state.", http.StatusConflict))
	}
	return jsonOK(pipeToResponse(pipe))
}

// ---- StopPipe ----

type stopPipeRequest struct {
	Name string `json:"Name"`
}

func handleStopPipe(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req stopPipeRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := req.Name
	if name == "" {
		name = ctx.Params["Name"]
	}
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}
	pipe, ok := store.StopPipe(name)
	if !ok {
		return jsonErr(service.NewAWSError("ConflictException", "Pipe "+name+" cannot be stopped in current state.", http.StatusConflict))
	}
	return jsonOK(pipeToResponse(pipe))
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
	arn := req.ResourceArn
	if arn == "" {
		arn = ctx.Params["ResourceArn"]
	}
	if arn == "" {
		return jsonErr(service.ErrValidation("ResourceArn is required."))
	}
	if !store.TagResource(arn, req.Tags) {
		return jsonErr(service.NewAWSError("NotFoundException", "Resource not found.", http.StatusNotFound))
	}
	return jsonOK(struct{}{})
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
	arn := req.ResourceArn
	if arn == "" {
		arn = ctx.Params["ResourceArn"]
	}
	if arn == "" {
		return jsonErr(service.ErrValidation("ResourceArn is required."))
	}
	if !store.UntagResource(arn, req.TagKeys) {
		return jsonErr(service.NewAWSError("NotFoundException", "Resource not found.", http.StatusNotFound))
	}
	return jsonOK(struct{}{})
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
	arn := req.ResourceArn
	if arn == "" {
		arn = ctx.Params["ResourceArn"]
	}
	if arn == "" {
		return jsonErr(service.ErrValidation("ResourceArn is required."))
	}
	tags, ok := store.ListTagsForResource(arn)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "Resource not found.", http.StatusNotFound))
	}
	return jsonOK(&listTagsResponse{Tags: tags})
}
