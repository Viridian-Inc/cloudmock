package efs

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ── Generated request/response types ─────────────────────────────────────────

type AccessPointDescription struct {
	AccessPointArn *string `json:"AccessPointArn,omitempty"`
	AccessPointId *string `json:"AccessPointId,omitempty"`
	ClientToken *string `json:"ClientToken,omitempty"`
	FileSystemId *string `json:"FileSystemId,omitempty"`
	LifeCycleState *string `json:"LifeCycleState,omitempty"`
	Name *string `json:"Name,omitempty"`
	OwnerId *string `json:"OwnerId,omitempty"`
	PosixUser *PosixUser `json:"PosixUser,omitempty"`
	RootDirectory *RootDirectory `json:"RootDirectory,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
}

type BackupPolicy struct {
	Status string `json:"Status,omitempty"`
}

type BackupPolicyDescription struct {
	BackupPolicy *BackupPolicy `json:"BackupPolicy,omitempty"`
}

type CreateAccessPointRequest struct {
	ClientToken string `json:"ClientToken,omitempty"`
	FileSystemId string `json:"FileSystemId,omitempty"`
	PosixUser *PosixUser `json:"PosixUser,omitempty"`
	RootDirectory *RootDirectory `json:"RootDirectory,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
}

type CreateFileSystemRequest struct {
	AvailabilityZoneName *string `json:"AvailabilityZoneName,omitempty"`
	Backup bool `json:"Backup,omitempty"`
	CreationToken string `json:"CreationToken,omitempty"`
	Encrypted bool `json:"Encrypted,omitempty"`
	KmsKeyId *string `json:"KmsKeyId,omitempty"`
	PerformanceMode *string `json:"PerformanceMode,omitempty"`
	ProvisionedThroughputInMibps float64 `json:"ProvisionedThroughputInMibps,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
	ThroughputMode *string `json:"ThroughputMode,omitempty"`
}

type CreateMountTargetRequest struct {
	FileSystemId string `json:"FileSystemId,omitempty"`
	IpAddress *string `json:"IpAddress,omitempty"`
	IpAddressType *string `json:"IpAddressType,omitempty"`
	Ipv6Address *string `json:"Ipv6Address,omitempty"`
	SecurityGroups []string `json:"SecurityGroups,omitempty"`
	SubnetId string `json:"SubnetId,omitempty"`
}

type CreateReplicationConfigurationRequest struct {
	Destinations []DestinationToCreate `json:"Destinations,omitempty"`
	SourceFileSystemId string `json:"SourceFileSystemId,omitempty"`
}

type CreateTagsRequest struct {
	FileSystemId string `json:"FileSystemId,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
}

type CreationInfo struct {
	OwnerGid int64 `json:"OwnerGid,omitempty"`
	OwnerUid int64 `json:"OwnerUid,omitempty"`
	Permissions string `json:"Permissions,omitempty"`
}

type DeleteAccessPointRequest struct {
	AccessPointId string `json:"AccessPointId,omitempty"`
}

type DeleteFileSystemPolicyRequest struct {
	FileSystemId string `json:"FileSystemId,omitempty"`
}

type DeleteFileSystemRequest struct {
	FileSystemId string `json:"FileSystemId,omitempty"`
}

type DeleteMountTargetRequest struct {
	MountTargetId string `json:"MountTargetId,omitempty"`
}

type DeleteReplicationConfigurationRequest struct {
	DeletionMode *string `json:"deletionMode,omitempty"`
	SourceFileSystemId string `json:"SourceFileSystemId,omitempty"`
}

type DeleteTagsRequest struct {
	FileSystemId string `json:"FileSystemId,omitempty"`
	TagKeys []string `json:"TagKeys,omitempty"`
}

type DescribeAccessPointsRequest struct {
	AccessPointId *string `json:"AccessPointId,omitempty"`
	FileSystemId *string `json:"FileSystemId,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type DescribeAccessPointsResponse struct {
	AccessPoints []AccessPointDescription `json:"AccessPoints,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type DescribeAccountPreferencesRequest struct {
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type DescribeAccountPreferencesResponse struct {
	NextToken *string `json:"NextToken,omitempty"`
	ResourceIdPreference *ResourceIdPreference `json:"ResourceIdPreference,omitempty"`
}

type DescribeBackupPolicyRequest struct {
	FileSystemId string `json:"FileSystemId,omitempty"`
}

type DescribeFileSystemPolicyRequest struct {
	FileSystemId string `json:"FileSystemId,omitempty"`
}

type DescribeFileSystemsRequest struct {
	CreationToken *string `json:"CreationToken,omitempty"`
	FileSystemId *string `json:"FileSystemId,omitempty"`
	Marker *string `json:"Marker,omitempty"`
	MaxItems int `json:"MaxItems,omitempty"`
}

type DescribeFileSystemsResponse struct {
	FileSystems []FileSystemDescription `json:"FileSystems,omitempty"`
	Marker *string `json:"Marker,omitempty"`
	NextMarker *string `json:"NextMarker,omitempty"`
}

type DescribeLifecycleConfigurationRequest struct {
	FileSystemId string `json:"FileSystemId,omitempty"`
}

type DescribeMountTargetSecurityGroupsRequest struct {
	MountTargetId string `json:"MountTargetId,omitempty"`
}

type DescribeMountTargetSecurityGroupsResponse struct {
	SecurityGroups []string `json:"SecurityGroups,omitempty"`
}

type DescribeMountTargetsRequest struct {
	AccessPointId *string `json:"AccessPointId,omitempty"`
	FileSystemId *string `json:"FileSystemId,omitempty"`
	Marker *string `json:"Marker,omitempty"`
	MaxItems int `json:"MaxItems,omitempty"`
	MountTargetId *string `json:"MountTargetId,omitempty"`
}

type DescribeMountTargetsResponse struct {
	Marker *string `json:"Marker,omitempty"`
	MountTargets []MountTargetDescription `json:"MountTargets,omitempty"`
	NextMarker *string `json:"NextMarker,omitempty"`
}

type DescribeReplicationConfigurationsRequest struct {
	FileSystemId *string `json:"FileSystemId,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type DescribeReplicationConfigurationsResponse struct {
	NextToken *string `json:"NextToken,omitempty"`
	Replications []ReplicationConfigurationDescription `json:"Replications,omitempty"`
}

type DescribeTagsRequest struct {
	FileSystemId string `json:"FileSystemId,omitempty"`
	Marker *string `json:"Marker,omitempty"`
	MaxItems int `json:"MaxItems,omitempty"`
}

type DescribeTagsResponse struct {
	Marker *string `json:"Marker,omitempty"`
	NextMarker *string `json:"NextMarker,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
}

type Destination struct {
	FileSystemId string `json:"FileSystemId,omitempty"`
	LastReplicatedTimestamp *time.Time `json:"LastReplicatedTimestamp,omitempty"`
	OwnerId *string `json:"OwnerId,omitempty"`
	Region string `json:"Region,omitempty"`
	RoleArn *string `json:"RoleArn,omitempty"`
	Status string `json:"Status,omitempty"`
	StatusMessage *string `json:"StatusMessage,omitempty"`
}

type DestinationToCreate struct {
	AvailabilityZoneName *string `json:"AvailabilityZoneName,omitempty"`
	FileSystemId *string `json:"FileSystemId,omitempty"`
	KmsKeyId *string `json:"KmsKeyId,omitempty"`
	Region *string `json:"Region,omitempty"`
	RoleArn *string `json:"RoleArn,omitempty"`
}

type FileSystemDescription struct {
	AvailabilityZoneId *string `json:"AvailabilityZoneId,omitempty"`
	AvailabilityZoneName *string `json:"AvailabilityZoneName,omitempty"`
	CreationTime time.Time `json:"CreationTime,omitempty"`
	CreationToken string `json:"CreationToken,omitempty"`
	Encrypted bool `json:"Encrypted,omitempty"`
	FileSystemArn *string `json:"FileSystemArn,omitempty"`
	FileSystemId string `json:"FileSystemId,omitempty"`
	FileSystemProtection *FileSystemProtectionDescription `json:"FileSystemProtection,omitempty"`
	KmsKeyId *string `json:"KmsKeyId,omitempty"`
	LifeCycleState string `json:"LifeCycleState,omitempty"`
	Name *string `json:"Name,omitempty"`
	NumberOfMountTargets int `json:"NumberOfMountTargets,omitempty"`
	OwnerId string `json:"OwnerId,omitempty"`
	PerformanceMode string `json:"PerformanceMode,omitempty"`
	ProvisionedThroughputInMibps float64 `json:"ProvisionedThroughputInMibps,omitempty"`
	SizeInBytes FileSystemSize `json:"SizeInBytes,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
	ThroughputMode *string `json:"ThroughputMode,omitempty"`
}

type FileSystemPolicyDescription struct {
	FileSystemId *string `json:"FileSystemId,omitempty"`
	Policy *string `json:"Policy,omitempty"`
}

type FileSystemProtectionDescription struct {
	ReplicationOverwriteProtection *string `json:"ReplicationOverwriteProtection,omitempty"`
}

type FileSystemSize struct {
	Timestamp *time.Time `json:"Timestamp,omitempty"`
	Value int64 `json:"Value,omitempty"`
	ValueInArchive int64 `json:"ValueInArchive,omitempty"`
	ValueInIA int64 `json:"ValueInIA,omitempty"`
	ValueInStandard int64 `json:"ValueInStandard,omitempty"`
}

type LifecycleConfigurationDescription struct {
	LifecyclePolicies []LifecyclePolicy `json:"LifecyclePolicies,omitempty"`
}

type LifecyclePolicy struct {
	TransitionToArchive *string `json:"TransitionToArchive,omitempty"`
	TransitionToIA *string `json:"TransitionToIA,omitempty"`
	TransitionToPrimaryStorageClass *string `json:"TransitionToPrimaryStorageClass,omitempty"`
}

type ListTagsForResourceRequest struct {
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	ResourceId string `json:"ResourceId,omitempty"`
}

type ListTagsForResourceResponse struct {
	NextToken *string `json:"NextToken,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
}

type ModifyMountTargetSecurityGroupsRequest struct {
	MountTargetId string `json:"MountTargetId,omitempty"`
	SecurityGroups []string `json:"SecurityGroups,omitempty"`
}

type MountTargetDescription struct {
	AvailabilityZoneId *string `json:"AvailabilityZoneId,omitempty"`
	AvailabilityZoneName *string `json:"AvailabilityZoneName,omitempty"`
	FileSystemId string `json:"FileSystemId,omitempty"`
	IpAddress *string `json:"IpAddress,omitempty"`
	Ipv6Address *string `json:"Ipv6Address,omitempty"`
	LifeCycleState string `json:"LifeCycleState,omitempty"`
	MountTargetId string `json:"MountTargetId,omitempty"`
	NetworkInterfaceId *string `json:"NetworkInterfaceId,omitempty"`
	OwnerId *string `json:"OwnerId,omitempty"`
	SubnetId string `json:"SubnetId,omitempty"`
	VpcId *string `json:"VpcId,omitempty"`
}

type PosixUser struct {
	Gid int64 `json:"Gid,omitempty"`
	SecondaryGids []int64 `json:"SecondaryGids,omitempty"`
	Uid int64 `json:"Uid,omitempty"`
}

type PutAccountPreferencesRequest struct {
	ResourceIdType string `json:"ResourceIdType,omitempty"`
}

type PutAccountPreferencesResponse struct {
	ResourceIdPreference *ResourceIdPreference `json:"ResourceIdPreference,omitempty"`
}

type PutBackupPolicyRequest struct {
	BackupPolicy BackupPolicy `json:"BackupPolicy,omitempty"`
	FileSystemId string `json:"FileSystemId,omitempty"`
}

type PutFileSystemPolicyRequest struct {
	BypassPolicyLockoutSafetyCheck bool `json:"BypassPolicyLockoutSafetyCheck,omitempty"`
	FileSystemId string `json:"FileSystemId,omitempty"`
	Policy string `json:"Policy,omitempty"`
}

type PutLifecycleConfigurationRequest struct {
	FileSystemId string `json:"FileSystemId,omitempty"`
	LifecyclePolicies []LifecyclePolicy `json:"LifecyclePolicies,omitempty"`
}

type ReplicationConfigurationDescription struct {
	CreationTime time.Time `json:"CreationTime,omitempty"`
	Destinations []Destination `json:"Destinations,omitempty"`
	OriginalSourceFileSystemArn string `json:"OriginalSourceFileSystemArn,omitempty"`
	SourceFileSystemArn string `json:"SourceFileSystemArn,omitempty"`
	SourceFileSystemId string `json:"SourceFileSystemId,omitempty"`
	SourceFileSystemOwnerId *string `json:"SourceFileSystemOwnerId,omitempty"`
	SourceFileSystemRegion string `json:"SourceFileSystemRegion,omitempty"`
}

type ResourceIdPreference struct {
	ResourceIdType *string `json:"ResourceIdType,omitempty"`
	Resources []string `json:"Resources,omitempty"`
}

type RootDirectory struct {
	CreationInfo *CreationInfo `json:"CreationInfo,omitempty"`
	Path *string `json:"Path,omitempty"`
}

type Tag struct {
	Key string `json:"Key,omitempty"`
	Value string `json:"Value,omitempty"`
}

type TagResourceRequest struct {
	ResourceId string `json:"ResourceId,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
}

type UntagResourceRequest struct {
	ResourceId string `json:"ResourceId,omitempty"`
	TagKeys []string `json:"tagKeys,omitempty"`
}

type UpdateFileSystemProtectionRequest struct {
	FileSystemId string `json:"FileSystemId,omitempty"`
	ReplicationOverwriteProtection *string `json:"ReplicationOverwriteProtection,omitempty"`
}

type UpdateFileSystemRequest struct {
	FileSystemId string `json:"FileSystemId,omitempty"`
	ProvisionedThroughputInMibps float64 `json:"ProvisionedThroughputInMibps,omitempty"`
	ThroughputMode *string `json:"ThroughputMode,omitempty"`
}



// ── Handler helpers ──────────────────────────────────────────────────────────

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
	if err := json.Unmarshal(body, v); err != nil {
		return service.NewAWSError("InvalidParameterException",
			"Request body is not valid JSON.", http.StatusBadRequest)
	}
	return nil
}

// ── Handlers ─────────────────────────────────────────────────────────────────

func handleCreateAccessPoint(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateAccessPointRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateAccessPoint business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateAccessPoint"})
}

func handleCreateFileSystem(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateFileSystemRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateFileSystem business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateFileSystem"})
}

func handleCreateMountTarget(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateMountTargetRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateMountTarget business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateMountTarget"})
}

func handleCreateReplicationConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateReplicationConfigurationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateReplicationConfiguration business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateReplicationConfiguration"})
}

func handleCreateTags(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateTagsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateTags business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateTags"})
}

func handleDeleteAccessPoint(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteAccessPointRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteAccessPoint business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteAccessPoint"})
}

func handleDeleteFileSystem(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteFileSystemRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteFileSystem business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteFileSystem"})
}

func handleDeleteFileSystemPolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteFileSystemPolicyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteFileSystemPolicy business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteFileSystemPolicy"})
}

func handleDeleteMountTarget(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteMountTargetRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteMountTarget business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteMountTarget"})
}

func handleDeleteReplicationConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteReplicationConfigurationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteReplicationConfiguration business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteReplicationConfiguration"})
}

func handleDeleteTags(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteTagsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteTags business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteTags"})
}

func handleDescribeAccessPoints(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeAccessPointsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeAccessPoints business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeAccessPoints"})
}

func handleDescribeAccountPreferences(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeAccountPreferencesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeAccountPreferences business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeAccountPreferences"})
}

func handleDescribeBackupPolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeBackupPolicyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeBackupPolicy business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeBackupPolicy"})
}

func handleDescribeFileSystemPolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeFileSystemPolicyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeFileSystemPolicy business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeFileSystemPolicy"})
}

func handleDescribeFileSystems(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeFileSystemsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeFileSystems business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeFileSystems"})
}

func handleDescribeLifecycleConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeLifecycleConfigurationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeLifecycleConfiguration business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeLifecycleConfiguration"})
}

func handleDescribeMountTargetSecurityGroups(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeMountTargetSecurityGroupsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeMountTargetSecurityGroups business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeMountTargetSecurityGroups"})
}

func handleDescribeMountTargets(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeMountTargetsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeMountTargets business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeMountTargets"})
}

func handleDescribeReplicationConfigurations(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeReplicationConfigurationsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeReplicationConfigurations business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeReplicationConfigurations"})
}

func handleDescribeTags(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeTagsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeTags business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeTags"})
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListTagsForResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListTagsForResource business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListTagsForResource"})
}

func handleModifyMountTargetSecurityGroups(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ModifyMountTargetSecurityGroupsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ModifyMountTargetSecurityGroups business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ModifyMountTargetSecurityGroups"})
}

func handlePutAccountPreferences(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req PutAccountPreferencesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement PutAccountPreferences business logic
	return jsonOK(map[string]any{"status": "ok", "action": "PutAccountPreferences"})
}

func handlePutBackupPolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req PutBackupPolicyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement PutBackupPolicy business logic
	return jsonOK(map[string]any{"status": "ok", "action": "PutBackupPolicy"})
}

func handlePutFileSystemPolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req PutFileSystemPolicyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement PutFileSystemPolicy business logic
	return jsonOK(map[string]any{"status": "ok", "action": "PutFileSystemPolicy"})
}

func handlePutLifecycleConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req PutLifecycleConfigurationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement PutLifecycleConfiguration business logic
	return jsonOK(map[string]any{"status": "ok", "action": "PutLifecycleConfiguration"})
}

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req TagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement TagResource business logic
	return jsonOK(map[string]any{"status": "ok", "action": "TagResource"})
}

func handleUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UntagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UntagResource business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UntagResource"})
}

func handleUpdateFileSystem(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateFileSystemRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateFileSystem business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateFileSystem"})
}

func handleUpdateFileSystemProtection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateFileSystemProtectionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateFileSystemProtection business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateFileSystemProtection"})
}

