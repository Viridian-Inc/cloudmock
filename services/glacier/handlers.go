package glacier

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
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
	ok, reason := store.DeleteVault(name)
	if !ok {
		switch reason {
		case "not_empty":
			return jsonErr(service.NewAWSError("InvalidParameterValueException",
				"Vault not empty: "+name, http.StatusBadRequest))
		default:
			return jsonErr(service.ErrNotFound("Vault", name))
		}
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

func handleInitiateVaultLock(ctx *service.RequestContext, vaultName string, store *Store) (*service.Response, error) {
	var params map[string]any
	if len(ctx.Body) > 0 {
		json.Unmarshal(ctx.Body, &params)
	}
	policy, _ := params["policy"].(string)
	if policy == "" {
		// Accept any structure
		policyBytes, _ := json.Marshal(params["policy"])
		policy = string(policyBytes)
	}

	lockID, err := store.InitiateVaultLock(vaultName, policy)
	if err != nil {
		if strings.Contains(err.Error(), "already locked") {
			return jsonErr(service.NewAWSError("InvalidParameterValueException", err.Error(), http.StatusBadRequest))
		}
		return jsonErr(service.ErrNotFound("Vault", vaultName))
	}
	return &service.Response{
		StatusCode: http.StatusCreated,
		Format:     service.FormatJSON,
		Headers:    map[string]string{"x-amz-lock-id": lockID},
	}, nil
}

func handleCompleteVaultLock(vaultName, lockID string, store *Store) (*service.Response, error) {
	err := store.CompleteVaultLock(vaultName, lockID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return jsonErr(service.ErrNotFound("Vault", vaultName))
		}
		return jsonErr(service.NewAWSError("InvalidParameterValueException", err.Error(), http.StatusBadRequest))
	}
	return jsonNoContent()
}

func handleSetVaultNotifications(ctx *service.RequestContext, vaultName string, store *Store) (*service.Response, error) {
	var params map[string]any
	if len(ctx.Body) > 0 {
		json.Unmarshal(ctx.Body, &params)
	}
	config, _ := params["vaultNotificationConfig"].(map[string]any)
	if config == nil {
		return jsonErr(service.ErrValidation("vaultNotificationConfig is required"))
	}
	snsTopic, _ := config["SNSTopic"].(string)
	var events []string
	if evts, ok := config["Events"].([]any); ok {
		for _, e := range evts {
			if s, ok := e.(string); ok {
				events = append(events, s)
			}
		}
	}
	if snsTopic == "" {
		return jsonErr(service.ErrValidation("SNSTopic is required in notification config"))
	}
	err := store.SetVaultNotifications(vaultName, snsTopic, events)
	if err != nil {
		return jsonErr(service.ErrNotFound("Vault", vaultName))
	}
	return jsonNoContent()
}

func handleGetVaultNotifications(vaultName string, store *Store) (*service.Response, error) {
	notif, err := store.GetVaultNotifications(vaultName)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return jsonErr(service.ErrNotFound("Vault", vaultName))
		}
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"No notification configuration for vault: "+vaultName, http.StatusNotFound))
	}
	return jsonOK(map[string]any{
		"vaultNotificationConfig": map[string]any{
			"SNSTopic": notif.SNSTopic,
			"Events":   notif.Events,
		},
	})
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

// ---- GetJobOutput ----

func handleGetJobOutput(vaultName, jobID string, store *Store) (*service.Response, error) {
	output, contentType, err := store.GetJobOutput(vaultName, jobID)
	if err != nil {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", err.Error(), http.StatusNotFound))
	}
	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       string(output),
		Headers:    map[string]string{"Content-Type": contentType},
		Format:     service.FormatJSON,
	}, nil
}

// ---- AbortVaultLock ----

func handleAbortVaultLock(vaultName string, store *Store) (*service.Response, error) {
	if err := store.AbortVaultLock(vaultName); err != nil {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", err.Error(), http.StatusNotFound))
	}
	return &service.Response{StatusCode: http.StatusNoContent, Format: service.FormatJSON}, nil
}

// ---- GetVaultLock ----

func handleGetVaultLock(vaultName string, store *Store) (*service.Response, error) {
	lock, err := store.GetVaultLock(vaultName)
	if err != nil {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", err.Error(), http.StatusNotFound))
	}
	return jsonOK(map[string]any{
		"Policy":       lock.Policy,
		"State":        lock.State,
		"LockId":       lock.LockID,
		"CreationDate": lock.CreatedAt.Format(time.RFC3339),
	})
}

// ---- Tags ----

func handleAddTagsToVault(ctx *service.RequestContext, vaultName string, store *Store) (*service.Response, error) {
	var body map[string]any
	if len(ctx.Body) > 0 {
		_ = json.Unmarshal(ctx.Body, &body)
	}
	tags := make(map[string]string)
	if body != nil {
		if t, ok := body["Tags"].(map[string]any); ok {
			for k, v := range t {
				if s, ok := v.(string); ok {
					tags[k] = s
				}
			}
		}
	}
	if err := store.AddTagsToVault(vaultName, tags); err != nil {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", err.Error(), http.StatusNotFound))
	}
	return &service.Response{StatusCode: http.StatusNoContent, Format: service.FormatJSON}, nil
}

func handleRemoveTagsFromVault(ctx *service.RequestContext, vaultName string, store *Store) (*service.Response, error) {
	var body map[string]any
	if len(ctx.Body) > 0 {
		_ = json.Unmarshal(ctx.Body, &body)
	}
	keys := make([]string, 0)
	if body != nil {
		if tagKeys, ok := body["TagKeys"].([]any); ok {
			for _, k := range tagKeys {
				if s, ok := k.(string); ok {
					keys = append(keys, s)
				}
			}
		}
	}
	if err := store.RemoveTagsFromVault(vaultName, keys); err != nil {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", err.Error(), http.StatusNotFound))
	}
	return &service.Response{StatusCode: http.StatusNoContent, Format: service.FormatJSON}, nil
}

func handleListTagsForVault(vaultName string, store *Store) (*service.Response, error) {
	tags, err := store.ListTagsForVault(vaultName)
	if err != nil {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", err.Error(), http.StatusNotFound))
	}
	return jsonOK(map[string]any{"Tags": tags})
}
