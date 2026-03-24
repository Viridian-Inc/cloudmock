package ssm

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
)

// ---- JSON request/response types ----

type tag struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

// PutParameter

type putParameterRequest struct {
	Name        string `json:"Name"`
	Value       string `json:"Value"`
	Type        string `json:"Type"`
	Overwrite   bool   `json:"Overwrite"`
	Description string `json:"Description"`
	Tags        []tag  `json:"Tags"`
}

type putParameterResponse struct {
	Version int    `json:"Version"`
	Tier    string `json:"Tier"`
}

// GetParameter

type getParameterRequest struct {
	Name           string `json:"Name"`
	WithDecryption bool   `json:"WithDecryption"`
}

type parameterResult struct {
	Name             string  `json:"Name"`
	Type             string  `json:"Type"`
	Value            string  `json:"Value"`
	Version          int     `json:"Version"`
	ARN              string  `json:"ARN"`
	LastModifiedDate float64 `json:"LastModifiedDate"`
	DataType         string  `json:"DataType"`
}

type getParameterResponse struct {
	Parameter parameterResult `json:"Parameter"`
}

// GetParameters

type getParametersRequest struct {
	Names          []string `json:"Names"`
	WithDecryption bool     `json:"WithDecryption"`
}

type getParametersResponse struct {
	Parameters        []parameterResult `json:"Parameters"`
	InvalidParameters []string          `json:"InvalidParameters"`
}

// GetParametersByPath

type getParametersByPathRequest struct {
	Path           string `json:"Path"`
	Recursive      bool   `json:"Recursive"`
	WithDecryption bool   `json:"WithDecryption"`
}

type getParametersByPathResponse struct {
	Parameters []parameterResult `json:"Parameters"`
}

// DeleteParameter

type deleteParameterRequest struct {
	Name string `json:"Name"`
}

// DeleteParameters

type deleteParametersRequest struct {
	Names []string `json:"Names"`
}

type deleteParametersResponse struct {
	DeletedParameters []string `json:"DeletedParameters"`
	InvalidParameters []string `json:"InvalidParameters"`
}

// DescribeParameters

type parameterMetadata struct {
	Name             string  `json:"Name"`
	Type             string  `json:"Type"`
	Version          int     `json:"Version"`
	ARN              string  `json:"ARN"`
	Description      string  `json:"Description,omitempty"`
	LastModifiedDate float64 `json:"LastModifiedDate"`
	DataType         string  `json:"DataType"`
}

type describeParametersResponse struct {
	Parameters []parameterMetadata `json:"Parameters"`
}

// ---- helpers ----

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       body,
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
		return service.NewAWSError("InvalidParameterException",
			"Request body is not valid JSON.", http.StatusBadRequest)
	}
	return nil
}

func unixFloat(t time.Time) float64 {
	return float64(t.Unix())
}

func tagsToMap(tags []tag) map[string]string {
	m := make(map[string]string, len(tags))
	for _, t := range tags {
		m[t.Key] = t.Value
	}
	return m
}

func paramToResult(p *Parameter) parameterResult {
	return parameterResult{
		Name:             p.Name,
		Type:             p.Type,
		Value:            p.Value,
		Version:          p.Version,
		ARN:              p.ARN,
		LastModifiedDate: unixFloat(p.LastModifiedDate),
		DataType:         p.DataType,
	}
}

func paramToMetadata(p *Parameter) parameterMetadata {
	return parameterMetadata{
		Name:             p.Name,
		Type:             p.Type,
		Version:          p.Version,
		ARN:              p.ARN,
		Description:      p.Description,
		LastModifiedDate: unixFloat(p.LastModifiedDate),
		DataType:         p.DataType,
	}
}

// ---- handlers ----

func handlePutParameter(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req putParameterRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Name == "" {
		return jsonErr(service.NewAWSError("ValidationError", "Name is required.", http.StatusBadRequest))
	}

	version, awsErr := store.PutParameter(req.Name, req.Value, req.Type, req.Description, req.Overwrite, tagsToMap(req.Tags))
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(putParameterResponse{
		Version: version,
		Tier:    "Standard",
	})
}

func handleGetParameter(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getParameterRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Name == "" {
		return jsonErr(service.NewAWSError("ValidationError", "Name is required.", http.StatusBadRequest))
	}

	p, awsErr := store.GetParameter(req.Name)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(getParameterResponse{
		Parameter: paramToResult(p),
	})
}

func handleGetParameters(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getParametersRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	found, invalid := store.GetParameters(req.Names)

	params := make([]parameterResult, 0, len(found))
	for _, p := range found {
		params = append(params, paramToResult(p))
	}

	if invalid == nil {
		invalid = []string{}
	}

	return jsonOK(getParametersResponse{
		Parameters:        params,
		InvalidParameters: invalid,
	})
}

func handleGetParametersByPath(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getParametersByPathRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Path == "" {
		return jsonErr(service.NewAWSError("ValidationError", "Path is required.", http.StatusBadRequest))
	}

	params := store.GetParametersByPath(req.Path, req.Recursive)

	results := make([]parameterResult, 0, len(params))
	for _, p := range params {
		results = append(results, paramToResult(p))
	}

	return jsonOK(getParametersByPathResponse{
		Parameters: results,
	})
}

func handleDeleteParameter(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteParameterRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Name == "" {
		return jsonErr(service.NewAWSError("ValidationError", "Name is required.", http.StatusBadRequest))
	}

	if awsErr := store.DeleteParameter(req.Name); awsErr != nil {
		return jsonErr(awsErr)
	}

	return &service.Response{StatusCode: http.StatusOK, Body: struct{}{}, Format: service.FormatJSON}, nil
}

func handleDeleteParameters(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteParametersRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	deleted, invalid := store.DeleteParameters(req.Names)

	if deleted == nil {
		deleted = []string{}
	}
	if invalid == nil {
		invalid = []string{}
	}

	return jsonOK(deleteParametersResponse{
		DeletedParameters: deleted,
		InvalidParameters: invalid,
	})
}

func handleDescribeParameters(_ *service.RequestContext, store *Store) (*service.Response, error) {
	params := store.DescribeParameters()

	metadata := make([]parameterMetadata, 0, len(params))
	for _, p := range params {
		metadata = append(metadata, paramToMetadata(p))
	}

	return jsonOK(describeParametersResponse{
		Parameters: metadata,
	})
}
