package glacier

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
)

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: body, Format: service.FormatJSON}, nil
}

func jsonErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
}

func jsonNoContent() (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusNoContent, Format: service.FormatJSON}, nil
}

func handleCreateVault(name string, store *Store) (*service.Response, error) {
	_, err := store.CreateVault(name)
	if err != nil {
		return jsonErr(service.ErrAlreadyExists("Vault", name))
	}
	return &service.Response{
		StatusCode: http.StatusCreated,
		Format:     service.FormatJSON,
		Headers:    map[string]string{"Location": "/" + name},
	}, nil
}

func handleDescribeVault(name string, store *Store) (*service.Response, error) {
	vault, ok := store.GetVault(name)
	if !ok {
		return jsonErr(service.ErrNotFound("Vault", name))
	}
	return jsonOK(vaultResponse(vault))
}

func handleListVaults(store *Store) (*service.Response, error) {
	vaults := store.ListVaults()
	out := make([]map[string]any, 0, len(vaults))
	for _, v := range vaults {
		out = append(out, vaultResponse(v))
	}
	return jsonOK(map[string]any{"VaultList": out})
}

func handleDeleteVault(name string, store *Store) (*service.Response, error) {
	if !store.DeleteVault(name) {
		return jsonErr(service.ErrNotFound("Vault", name))
	}
	return jsonNoContent()
}

func handleUploadArchive(ctx *service.RequestContext, vaultName string, store *Store) (*service.Response, error) {
	r := ctx.RawRequest
	description := r.Header.Get("x-amz-archive-description")
	treeHash := r.Header.Get("x-amz-sha256-tree-hash")
	size := int64(len(ctx.Body))

	archive, err := store.UploadArchive(vaultName, description, treeHash, size)
	if err != nil {
		return jsonErr(service.ErrNotFound("Vault", vaultName))
	}
	return &service.Response{
		StatusCode: http.StatusCreated,
		Format:     service.FormatJSON,
		Headers: map[string]string{
			"x-amz-archive-id":       archive.ArchiveID,
			"x-amz-sha256-tree-hash": archive.SHA256TreeHash,
			"Location":               fmt.Sprintf("/%s/vaults/%s/archives/%s", store.accountID, vaultName, archive.ArchiveID),
		},
	}, nil
}

func handleDeleteArchive(vaultName, archiveID string, store *Store) (*service.Response, error) {
	if err := store.DeleteArchive(vaultName, archiveID); err != nil {
		return jsonErr(service.ErrNotFound("Archive", archiveID))
	}
	return jsonNoContent()
}

func handleInitiateJob(ctx *service.RequestContext, vaultName string, store *Store) (*service.Response, error) {
	var params map[string]any
	if len(ctx.Body) > 0 {
		json.Unmarshal(ctx.Body, &params)
	}

	action, _ := params["Type"].(string)
	if action == "" {
		return jsonErr(service.ErrValidation("Type is required"))
	}
	archiveID, _ := params["ArchiveId"].(string)
	snsTopic, _ := params["SNSTopic"].(string)

	job, err := store.InitiateJob(vaultName, action, archiveID, snsTopic)
	if err != nil {
		return jsonErr(service.ErrNotFound("Vault", vaultName))
	}
	return &service.Response{
		StatusCode: http.StatusAccepted,
		Format:     service.FormatJSON,
		Headers: map[string]string{
			"x-amz-job-id": job.JobID,
			"Location":     fmt.Sprintf("/%s/vaults/%s/jobs/%s", store.accountID, vaultName, job.JobID),
		},
	}, nil
}

func handleDescribeJob(vaultName, jobID string, store *Store) (*service.Response, error) {
	job, ok := store.GetJob(vaultName, jobID)
	if !ok {
		return jsonErr(service.ErrNotFound("Job", jobID))
	}
	return jsonOK(jobResponse(job, store))
}

func handleListJobs(vaultName string, store *Store) (*service.Response, error) {
	jobs := store.ListJobs(vaultName)
	out := make([]map[string]any, 0, len(jobs))
	for _, j := range jobs {
		out = append(out, jobResponse(j, store))
	}
	return jsonOK(map[string]any{"JobList": out})
}

func vaultResponse(v *Vault) map[string]any {
	resp := map[string]any{
		"VaultName":        v.VaultName,
		"VaultARN":         v.VaultARN,
		"CreationDate":     v.CreationDate.Format(time.RFC3339),
		"NumberOfArchives": v.NumberOfArchives,
		"SizeInBytes":      v.SizeInBytes,
	}
	if v.LastInventoryDate != nil {
		resp["LastInventoryDate"] = v.LastInventoryDate.Format(time.RFC3339)
	}
	return resp
}

func jobResponse(j *Job, store *Store) map[string]any {
	resp := map[string]any{
		"JobId":         j.JobID,
		"VaultARN":      fmt.Sprintf("arn:aws:glacier:%s:%s:vaults/%s", store.region, store.accountID, j.VaultName),
		"Action":        j.Action,
		"StatusCode":    j.StatusCode,
		"StatusMessage": j.StatusMessage,
		"CreationDate":  j.CreationDate.Format(time.RFC3339),
	}
	if j.ArchiveID != "" {
		resp["ArchiveId"] = j.ArchiveID
		resp["ArchiveSizeInBytes"] = j.ArchiveSizeInBytes
	}
	if j.CompletionDate != nil {
		resp["CompletionDate"] = j.CompletionDate.Format(time.RFC3339)
		resp["Completed"] = true
	} else {
		resp["Completed"] = false
	}
	return resp
}
