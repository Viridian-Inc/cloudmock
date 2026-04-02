package glacier

import (
	"net/http"
	"strings"

	"github.com/neureaux/cloudmock/pkg/service"
)

// GlacierService is the cloudmock implementation of the Amazon Glacier API.
type GlacierService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new GlacierService for the given AWS account ID and region.
func New(accountID, region string) *GlacierService {
	return &GlacierService{store: NewStore(accountID, region), accountID: accountID, region: region}
}

// Name returns the AWS service name used for routing.
func (s *GlacierService) Name() string { return "glacier" }

// Actions returns the list of Glacier API actions supported by this service.
func (s *GlacierService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateVault", Method: http.MethodPut, IAMAction: "glacier:CreateVault"},
		{Name: "DescribeVault", Method: http.MethodGet, IAMAction: "glacier:DescribeVault"},
		{Name: "ListVaults", Method: http.MethodGet, IAMAction: "glacier:ListVaults"},
		{Name: "DeleteVault", Method: http.MethodDelete, IAMAction: "glacier:DeleteVault"},
		{Name: "UploadArchive", Method: http.MethodPost, IAMAction: "glacier:UploadArchive"},
		{Name: "DeleteArchive", Method: http.MethodDelete, IAMAction: "glacier:DeleteArchive"},
		{Name: "InitiateJob", Method: http.MethodPost, IAMAction: "glacier:InitiateJob"},
		{Name: "DescribeJob", Method: http.MethodGet, IAMAction: "glacier:DescribeJob"},
		{Name: "ListJobs", Method: http.MethodGet, IAMAction: "glacier:ListJobs"},
		{Name: "GetJobOutput", Method: http.MethodGet, IAMAction: "glacier:GetJobOutput"},
		{Name: "InitiateVaultLock", Method: http.MethodPost, IAMAction: "glacier:InitiateVaultLock"},
		{Name: "CompleteVaultLock", Method: http.MethodPost, IAMAction: "glacier:CompleteVaultLock"},
		{Name: "AbortVaultLock", Method: http.MethodDelete, IAMAction: "glacier:AbortVaultLock"},
		{Name: "GetVaultLock", Method: http.MethodGet, IAMAction: "glacier:GetVaultLock"},
		{Name: "AddTagsToVault", Method: http.MethodPost, IAMAction: "glacier:AddTagsToVault"},
		{Name: "RemoveTagsFromVault", Method: http.MethodPost, IAMAction: "glacier:RemoveTagsFromVault"},
		{Name: "ListTagsForVault", Method: http.MethodGet, IAMAction: "glacier:ListTagsForVault"},
	}
}

// HealthCheck always returns nil.
func (s *GlacierService) HealthCheck() error { return nil }

// HandleRequest routes an incoming Glacier request to the appropriate handler.
// Glacier uses REST-JSON protocol with path-based routing.
func (s *GlacierService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	r := ctx.RawRequest
	method := r.Method
	path := strings.TrimRight(r.URL.Path, "/")

	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")

	if len(parts) < 2 || parts[1] != "vaults" {
		return jsonErr(service.NewAWSError("NotImplemented", "Route not implemented", http.StatusNotImplemented))
	}

	// GET /-/vaults -> ListVaults
	if len(parts) == 2 && method == http.MethodGet {
		return handleListVaults(s.store)
	}

	if len(parts) < 3 {
		return jsonErr(service.NewAWSError("NotImplemented", "Route not implemented", http.StatusNotImplemented))
	}

	vaultName := parts[2]

	if len(parts) == 3 {
		switch method {
		case http.MethodPut:
			return handleCreateVault(vaultName, s.store)
		case http.MethodGet:
			return handleDescribeVault(vaultName, s.store)
		case http.MethodDelete:
			return handleDeleteVault(vaultName, s.store)
		}
		return jsonErr(service.NewAWSError("NotImplemented", "Route not implemented", http.StatusNotImplemented))
	}

	subResource := parts[3]

	switch subResource {
	case "archives":
		if len(parts) == 4 && method == http.MethodPost {
			return handleUploadArchive(ctx, vaultName, s.store)
		}
		if len(parts) == 5 && method == http.MethodDelete {
			return handleDeleteArchive(vaultName, parts[4], s.store)
		}
	case "jobs":
		if len(parts) == 4 {
			switch method {
			case http.MethodPost:
				return handleInitiateJob(ctx, vaultName, s.store)
			case http.MethodGet:
				return handleListJobs(vaultName, s.store)
			}
		}
		if len(parts) == 5 {
			switch method {
			case http.MethodGet:
				return handleDescribeJob(vaultName, parts[4], s.store)
			}
		}
		if len(parts) == 6 && parts[5] == "output" && method == http.MethodGet {
			return handleGetJobOutput(vaultName, parts[4], s.store)
		}
	case "lock-policy":
		if len(parts) == 4 {
			switch method {
			case http.MethodPost:
				return handleInitiateVaultLock(ctx, vaultName, s.store)
			case http.MethodGet:
				return handleGetVaultLock(vaultName, s.store)
			case http.MethodDelete:
				return handleAbortVaultLock(vaultName, s.store)
			}
		}
		if len(parts) == 5 && method == http.MethodPost {
			// Complete vault lock: POST /-/vaults/{name}/lock-policy/{lockId}
			return handleCompleteVaultLock(vaultName, parts[4], s.store)
		}
	case "tags":
		// GET /-/vaults/{name}/tags -> ListTagsForVault
		// POST /-/vaults/{name}/tags -> AddTagsToVault
		switch method {
		case http.MethodGet:
			return handleListTagsForVault(vaultName, s.store)
		case http.MethodPost:
			return handleAddTagsToVault(ctx, vaultName, s.store)
		}
		// DELETE tags handled with query param "operation=remove"
		if method == http.MethodPost && r.URL.Query().Get("operation") == "remove" {
			return handleRemoveTagsFromVault(ctx, vaultName, s.store)
		}
	case "notification-configuration":
		if method == http.MethodPut {
			return handleSetVaultNotifications(ctx, vaultName, s.store)
		}
		if method == http.MethodGet {
			return handleGetVaultNotifications(vaultName, s.store)
		}
	}

	return jsonErr(service.NewAWSError("NotImplemented", "Route not implemented", http.StatusNotImplemented))
}
