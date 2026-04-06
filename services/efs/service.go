package efs

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// Service is the cloudmock implementation of the AWS elasticfilesystem service.
type Service struct {
	store *Store
}

// New returns a new efs Service.
func New(accountID, region string) *Service {
	return &Service{store: NewStore(accountID, region)}
}

// Name returns the AWS service name used for request routing.
func (s *Service) Name() string { return "elasticfilesystem" }

// Actions returns all supported API actions.
func (s *Service) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateAccessPoint", Method: http.MethodPost, IAMAction: "elasticfilesystem:CreateAccessPoint"},
		{Name: "CreateFileSystem", Method: http.MethodPost, IAMAction: "elasticfilesystem:CreateFileSystem"},
		{Name: "CreateMountTarget", Method: http.MethodPost, IAMAction: "elasticfilesystem:CreateMountTarget"},
		{Name: "CreateReplicationConfiguration", Method: http.MethodPost, IAMAction: "elasticfilesystem:CreateReplicationConfiguration"},
		{Name: "CreateTags", Method: http.MethodPost, IAMAction: "elasticfilesystem:CreateTags"},
		{Name: "DeleteAccessPoint", Method: http.MethodDelete, IAMAction: "elasticfilesystem:DeleteAccessPoint"},
		{Name: "DeleteFileSystem", Method: http.MethodDelete, IAMAction: "elasticfilesystem:DeleteFileSystem"},
		{Name: "DeleteFileSystemPolicy", Method: http.MethodDelete, IAMAction: "elasticfilesystem:DeleteFileSystemPolicy"},
		{Name: "DeleteMountTarget", Method: http.MethodDelete, IAMAction: "elasticfilesystem:DeleteMountTarget"},
		{Name: "DeleteReplicationConfiguration", Method: http.MethodDelete, IAMAction: "elasticfilesystem:DeleteReplicationConfiguration"},
		{Name: "DeleteTags", Method: http.MethodPost, IAMAction: "elasticfilesystem:DeleteTags"},
		{Name: "DescribeAccessPoints", Method: http.MethodGet, IAMAction: "elasticfilesystem:DescribeAccessPoints"},
		{Name: "DescribeAccountPreferences", Method: http.MethodGet, IAMAction: "elasticfilesystem:DescribeAccountPreferences"},
		{Name: "DescribeBackupPolicy", Method: http.MethodGet, IAMAction: "elasticfilesystem:DescribeBackupPolicy"},
		{Name: "DescribeFileSystemPolicy", Method: http.MethodGet, IAMAction: "elasticfilesystem:DescribeFileSystemPolicy"},
		{Name: "DescribeFileSystems", Method: http.MethodGet, IAMAction: "elasticfilesystem:DescribeFileSystems"},
		{Name: "DescribeLifecycleConfiguration", Method: http.MethodGet, IAMAction: "elasticfilesystem:DescribeLifecycleConfiguration"},
		{Name: "DescribeMountTargetSecurityGroups", Method: http.MethodGet, IAMAction: "elasticfilesystem:DescribeMountTargetSecurityGroups"},
		{Name: "DescribeMountTargets", Method: http.MethodGet, IAMAction: "elasticfilesystem:DescribeMountTargets"},
		{Name: "DescribeReplicationConfigurations", Method: http.MethodGet, IAMAction: "elasticfilesystem:DescribeReplicationConfigurations"},
		{Name: "DescribeTags", Method: http.MethodGet, IAMAction: "elasticfilesystem:DescribeTags"},
		{Name: "ListTagsForResource", Method: http.MethodGet, IAMAction: "elasticfilesystem:ListTagsForResource"},
		{Name: "ModifyMountTargetSecurityGroups", Method: http.MethodPut, IAMAction: "elasticfilesystem:ModifyMountTargetSecurityGroups"},
		{Name: "PutAccountPreferences", Method: http.MethodPut, IAMAction: "elasticfilesystem:PutAccountPreferences"},
		{Name: "PutBackupPolicy", Method: http.MethodPut, IAMAction: "elasticfilesystem:PutBackupPolicy"},
		{Name: "PutFileSystemPolicy", Method: http.MethodPut, IAMAction: "elasticfilesystem:PutFileSystemPolicy"},
		{Name: "PutLifecycleConfiguration", Method: http.MethodPut, IAMAction: "elasticfilesystem:PutLifecycleConfiguration"},
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "elasticfilesystem:TagResource"},
		{Name: "UntagResource", Method: http.MethodDelete, IAMAction: "elasticfilesystem:UntagResource"},
		{Name: "UpdateFileSystem", Method: http.MethodPut, IAMAction: "elasticfilesystem:UpdateFileSystem"},
		{Name: "UpdateFileSystemProtection", Method: http.MethodPut, IAMAction: "elasticfilesystem:UpdateFileSystemProtection"},
	}
}

// HealthCheck always returns nil.
func (s *Service) HealthCheck() error { return nil }

// HandleRequest routes a request to the appropriate handler.
func (s *Service) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	case "CreateAccessPoint":
		return handleCreateAccessPoint(ctx, s.store)
	case "CreateFileSystem":
		return handleCreateFileSystem(ctx, s.store)
	case "CreateMountTarget":
		return handleCreateMountTarget(ctx, s.store)
	case "CreateReplicationConfiguration":
		return handleCreateReplicationConfiguration(ctx, s.store)
	case "CreateTags":
		return handleCreateTags(ctx, s.store)
	case "DeleteAccessPoint":
		return handleDeleteAccessPoint(ctx, s.store)
	case "DeleteFileSystem":
		return handleDeleteFileSystem(ctx, s.store)
	case "DeleteFileSystemPolicy":
		return handleDeleteFileSystemPolicy(ctx, s.store)
	case "DeleteMountTarget":
		return handleDeleteMountTarget(ctx, s.store)
	case "DeleteReplicationConfiguration":
		return handleDeleteReplicationConfiguration(ctx, s.store)
	case "DeleteTags":
		return handleDeleteTags(ctx, s.store)
	case "DescribeAccessPoints":
		return handleDescribeAccessPoints(ctx, s.store)
	case "DescribeAccountPreferences":
		return handleDescribeAccountPreferences(ctx, s.store)
	case "DescribeBackupPolicy":
		return handleDescribeBackupPolicy(ctx, s.store)
	case "DescribeFileSystemPolicy":
		return handleDescribeFileSystemPolicy(ctx, s.store)
	case "DescribeFileSystems":
		return handleDescribeFileSystems(ctx, s.store)
	case "DescribeLifecycleConfiguration":
		return handleDescribeLifecycleConfiguration(ctx, s.store)
	case "DescribeMountTargetSecurityGroups":
		return handleDescribeMountTargetSecurityGroups(ctx, s.store)
	case "DescribeMountTargets":
		return handleDescribeMountTargets(ctx, s.store)
	case "DescribeReplicationConfigurations":
		return handleDescribeReplicationConfigurations(ctx, s.store)
	case "DescribeTags":
		return handleDescribeTags(ctx, s.store)
	case "ListTagsForResource":
		return handleListTagsForResource(ctx, s.store)
	case "ModifyMountTargetSecurityGroups":
		return handleModifyMountTargetSecurityGroups(ctx, s.store)
	case "PutAccountPreferences":
		return handlePutAccountPreferences(ctx, s.store)
	case "PutBackupPolicy":
		return handlePutBackupPolicy(ctx, s.store)
	case "PutFileSystemPolicy":
		return handlePutFileSystemPolicy(ctx, s.store)
	case "PutLifecycleConfiguration":
		return handlePutLifecycleConfiguration(ctx, s.store)
	case "TagResource":
		return handleTagResource(ctx, s.store)
	case "UntagResource":
		return handleUntagResource(ctx, s.store)
	case "UpdateFileSystem":
		return handleUpdateFileSystem(ctx, s.store)
	case "UpdateFileSystemProtection":
		return handleUpdateFileSystemProtection(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
