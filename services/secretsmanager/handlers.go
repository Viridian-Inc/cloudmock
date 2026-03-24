package secretsmanager

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

type createSecretRequest struct {
	Name         string `json:"Name"`
	Description  string `json:"Description"`
	SecretString string `json:"SecretString"`
	SecretBinary []byte `json:"SecretBinary"`
	Tags         []tag  `json:"Tags"`
}

type createSecretResponse struct {
	ARN       string `json:"ARN"`
	Name      string `json:"Name"`
	VersionId string `json:"VersionId"`
}

type getSecretValueRequest struct {
	SecretId string `json:"SecretId"`
}

type getSecretValueResponse struct {
	ARN           string    `json:"ARN"`
	Name          string    `json:"Name"`
	SecretString  string    `json:"SecretString,omitempty"`
	SecretBinary  []byte    `json:"SecretBinary,omitempty"`
	VersionId     string    `json:"VersionId"`
	VersionStages []string  `json:"VersionStages"`
	CreatedDate   float64   `json:"CreatedDate"`
}

type putSecretValueRequest struct {
	SecretId     string `json:"SecretId"`
	SecretString string `json:"SecretString"`
	SecretBinary []byte `json:"SecretBinary"`
}

type putSecretValueResponse struct {
	ARN           string   `json:"ARN"`
	Name          string   `json:"Name"`
	VersionId     string   `json:"VersionId"`
	VersionStages []string `json:"VersionStages"`
}

type updateSecretRequest struct {
	SecretId     string `json:"SecretId"`
	Description  string `json:"Description"`
	SecretString string `json:"SecretString"`
}

type updateSecretResponse struct {
	ARN       string `json:"ARN"`
	Name      string `json:"Name"`
	VersionId string `json:"VersionId"`
}

type deleteSecretRequest struct {
	SecretId                   string `json:"SecretId"`
	ForceDeleteWithoutRecovery bool   `json:"ForceDeleteWithoutRecovery"`
}

type deleteSecretResponse struct {
	ARN          string  `json:"ARN"`
	Name         string  `json:"Name"`
	DeletionDate float64 `json:"DeletionDate"`
}

type restoreSecretRequest struct {
	SecretId string `json:"SecretId"`
}

type restoreSecretResponse struct {
	ARN  string `json:"ARN"`
	Name string `json:"Name"`
}

type describeSecretRequest struct {
	SecretId string `json:"SecretId"`
}

type secretMetadata struct {
	ARN             string            `json:"ARN"`
	Name            string            `json:"Name"`
	Description     string            `json:"Description,omitempty"`
	CreatedDate     float64           `json:"CreatedDate"`
	LastChangedDate float64           `json:"LastChangedDate"`
	Tags            []tag             `json:"Tags"`
	VersionIdsToStages map[string][]string `json:"VersionIdsToStages"`
}

type listSecretsResponse struct {
	SecretList []secretMetadata `json:"SecretList"`
}

type tagResourceRequest struct {
	SecretId string `json:"SecretId"`
	Tags     []tag  `json:"Tags"`
}

type untagResourceRequest struct {
	SecretId string   `json:"SecretId"`
	TagKeys  []string `json:"TagKeys"`
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

func tagsFromMap(m map[string]string) []tag {
	out := make([]tag, 0, len(m))
	for k, v := range m {
		out = append(out, tag{Key: k, Value: v})
	}
	return out
}

func tagsToMap(tags []tag) map[string]string {
	m := make(map[string]string, len(tags))
	for _, t := range tags {
		m[t.Key] = t.Value
	}
	return m
}

func secretToMetadata(sec *Secret) secretMetadata {
	return secretMetadata{
		ARN:             sec.ARN,
		Name:            sec.Name,
		Description:     sec.Description,
		CreatedDate:     unixFloat(sec.CreatedDate),
		LastChangedDate: unixFloat(sec.LastChangedDate),
		Tags:            tagsFromMap(sec.Tags),
		VersionIdsToStages: map[string][]string{
			sec.VersionId: sec.VersionStages,
		},
	}
}

// ---- handlers ----

func handleCreateSecret(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createSecretRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Name == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"Name is required.", http.StatusBadRequest))
	}

	sec, awsErr := store.CreateSecret(req.Name, req.Description, req.SecretString, req.SecretBinary, tagsToMap(req.Tags))
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(createSecretResponse{
		ARN:       sec.ARN,
		Name:      sec.Name,
		VersionId: sec.VersionId,
	})
}

func handleGetSecretValue(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getSecretValueRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.SecretId == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"SecretId is required.", http.StatusBadRequest))
	}

	sec, awsErr := store.GetSecret(req.SecretId)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(getSecretValueResponse{
		ARN:           sec.ARN,
		Name:          sec.Name,
		SecretString:  sec.SecretString,
		SecretBinary:  sec.SecretBinary,
		VersionId:     sec.VersionId,
		VersionStages: sec.VersionStages,
		CreatedDate:   unixFloat(sec.CreatedDate),
	})
}

func handlePutSecretValue(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req putSecretValueRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.SecretId == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"SecretId is required.", http.StatusBadRequest))
	}

	sec, awsErr := store.PutSecretValue(req.SecretId, req.SecretString, req.SecretBinary)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(putSecretValueResponse{
		ARN:           sec.ARN,
		Name:          sec.Name,
		VersionId:     sec.VersionId,
		VersionStages: sec.VersionStages,
	})
}

func handleUpdateSecret(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req updateSecretRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.SecretId == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"SecretId is required.", http.StatusBadRequest))
	}

	sec, awsErr := store.UpdateSecret(req.SecretId, req.Description, req.SecretString)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(updateSecretResponse{
		ARN:       sec.ARN,
		Name:      sec.Name,
		VersionId: sec.VersionId,
	})
}

func handleDeleteSecret(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteSecretRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.SecretId == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"SecretId is required.", http.StatusBadRequest))
	}

	sec, awsErr := store.DeleteSecret(req.SecretId, req.ForceDeleteWithoutRecovery)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	var deletionDate float64
	if sec.DeletedDate != nil {
		deletionDate = unixFloat(*sec.DeletedDate)
	}

	return jsonOK(deleteSecretResponse{
		ARN:          sec.ARN,
		Name:         sec.Name,
		DeletionDate: deletionDate,
	})
}

func handleRestoreSecret(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req restoreSecretRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.SecretId == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"SecretId is required.", http.StatusBadRequest))
	}

	sec, awsErr := store.RestoreSecret(req.SecretId)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(restoreSecretResponse{
		ARN:  sec.ARN,
		Name: sec.Name,
	})
}

func handleDescribeSecret(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeSecretRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.SecretId == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"SecretId is required.", http.StatusBadRequest))
	}

	sec, awsErr := store.DescribeSecret(req.SecretId)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(secretToMetadata(sec))
}

func handleListSecrets(_ *service.RequestContext, store *Store) (*service.Response, error) {
	secrets := store.ListSecrets()
	list := make([]secretMetadata, 0, len(secrets))
	for _, sec := range secrets {
		list = append(list, secretToMetadata(sec))
	}
	return jsonOK(listSecretsResponse{SecretList: list})
}

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req tagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.SecretId == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"SecretId is required.", http.StatusBadRequest))
	}

	if awsErr := store.TagResource(req.SecretId, tagsToMap(req.Tags)); awsErr != nil {
		return jsonErr(awsErr)
	}

	return &service.Response{StatusCode: http.StatusOK, Body: struct{}{}, Format: service.FormatJSON}, nil
}

func handleUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req untagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.SecretId == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"SecretId is required.", http.StatusBadRequest))
	}

	if awsErr := store.UntagResource(req.SecretId, req.TagKeys); awsErr != nil {
		return jsonErr(awsErr)
	}

	return &service.Response{StatusCode: http.StatusOK, Body: struct{}{}, Format: service.FormatJSON}, nil
}
