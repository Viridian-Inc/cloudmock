package keyspaces

import (
	gojson "github.com/goccy/go-json"
	"net/http"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ── Generated request/response types ─────────────────────────────────────────

type AutoScalingPolicy struct {
	TargetTrackingScalingPolicyConfiguration *TargetTrackingScalingPolicyConfiguration `json:"targetTrackingScalingPolicyConfiguration,omitempty"`
}

type AutoScalingSettings struct {
	AutoScalingDisabled bool `json:"autoScalingDisabled,omitempty"`
	MaximumUnits int64 `json:"maximumUnits,omitempty"`
	MinimumUnits int64 `json:"minimumUnits,omitempty"`
	ScalingPolicy *AutoScalingPolicy `json:"scalingPolicy,omitempty"`
}

type AutoScalingSpecification struct {
	ReadCapacityAutoScaling *AutoScalingSettings `json:"readCapacityAutoScaling,omitempty"`
	WriteCapacityAutoScaling *AutoScalingSettings `json:"writeCapacityAutoScaling,omitempty"`
}

type CapacitySpecification struct {
	ReadCapacityUnits int64 `json:"readCapacityUnits,omitempty"`
	ThroughputMode string `json:"throughputMode,omitempty"`
	WriteCapacityUnits int64 `json:"writeCapacityUnits,omitempty"`
}

type CapacitySpecificationSummary struct {
	LastUpdateToPayPerRequestTimestamp *time.Time `json:"lastUpdateToPayPerRequestTimestamp,omitempty"`
	ReadCapacityUnits int64 `json:"readCapacityUnits,omitempty"`
	ThroughputMode string `json:"throughputMode,omitempty"`
	WriteCapacityUnits int64 `json:"writeCapacityUnits,omitempty"`
}

type CdcSpecification struct {
	PropagateTags *string `json:"propagateTags,omitempty"`
	Status string `json:"status,omitempty"`
	Tags []Tag `json:"tags,omitempty"`
	ViewType *string `json:"viewType,omitempty"`
}

type CdcSpecificationSummary struct {
	Status string `json:"status,omitempty"`
	ViewType *string `json:"viewType,omitempty"`
}

type ClientSideTimestamps struct {
	Status string `json:"status,omitempty"`
}

type ClusteringKey struct {
	Name string `json:"name,omitempty"`
	OrderBy string `json:"orderBy,omitempty"`
}

type ColumnDefinition struct {
	Name string `json:"name,omitempty"`
	Type string `json:"type,omitempty"`
}

type Comment struct {
	Message string `json:"message,omitempty"`
}

type CreateKeyspaceRequest struct {
	KeyspaceName string `json:"keyspaceName,omitempty"`
	ReplicationSpecification *ReplicationSpecification `json:"replicationSpecification,omitempty"`
	Tags []Tag `json:"tags,omitempty"`
}

type CreateKeyspaceResponse struct {
	ResourceArn string `json:"resourceArn,omitempty"`
}

type CreateTableRequest struct {
	AutoScalingSpecification *AutoScalingSpecification `json:"autoScalingSpecification,omitempty"`
	CapacitySpecification *CapacitySpecification `json:"capacitySpecification,omitempty"`
	CdcSpecification *CdcSpecification `json:"cdcSpecification,omitempty"`
	ClientSideTimestamps *ClientSideTimestamps `json:"clientSideTimestamps,omitempty"`
	Comment *Comment `json:"comment,omitempty"`
	DefaultTimeToLive int `json:"defaultTimeToLive,omitempty"`
	EncryptionSpecification *EncryptionSpecification `json:"encryptionSpecification,omitempty"`
	KeyspaceName string `json:"keyspaceName,omitempty"`
	PointInTimeRecovery *PointInTimeRecovery `json:"pointInTimeRecovery,omitempty"`
	ReplicaSpecifications []ReplicaSpecification `json:"replicaSpecifications,omitempty"`
	SchemaDefinition SchemaDefinition `json:"schemaDefinition,omitempty"`
	TableName string `json:"tableName,omitempty"`
	Tags []Tag `json:"tags,omitempty"`
	Ttl *TimeToLive `json:"ttl,omitempty"`
	WarmThroughputSpecification *WarmThroughputSpecification `json:"warmThroughputSpecification,omitempty"`
}

type CreateTableResponse struct {
	ResourceArn string `json:"resourceArn,omitempty"`
}

type CreateTypeRequest struct {
	FieldDefinitions []FieldDefinition `json:"fieldDefinitions,omitempty"`
	KeyspaceName string `json:"keyspaceName,omitempty"`
	TypeName string `json:"typeName,omitempty"`
}

type CreateTypeResponse struct {
	KeyspaceArn string `json:"keyspaceArn,omitempty"`
	TypeName string `json:"typeName,omitempty"`
}

type DeleteKeyspaceRequest struct {
	KeyspaceName string `json:"keyspaceName,omitempty"`
}

type DeleteKeyspaceResponse struct {
}

type DeleteTableRequest struct {
	KeyspaceName string `json:"keyspaceName,omitempty"`
	TableName string `json:"tableName,omitempty"`
}

type DeleteTableResponse struct {
}

type DeleteTypeRequest struct {
	KeyspaceName string `json:"keyspaceName,omitempty"`
	TypeName string `json:"typeName,omitempty"`
}

type DeleteTypeResponse struct {
	KeyspaceArn string `json:"keyspaceArn,omitempty"`
	TypeName string `json:"typeName,omitempty"`
}

type EncryptionSpecification struct {
	KmsKeyIdentifier *string `json:"kmsKeyIdentifier,omitempty"`
	Type string `json:"type,omitempty"`
}

type FieldDefinition struct {
	Name string `json:"name,omitempty"`
	Type string `json:"type,omitempty"`
}

type GetKeyspaceRequest struct {
	KeyspaceName string `json:"keyspaceName,omitempty"`
}

type GetKeyspaceResponse struct {
	KeyspaceName string `json:"keyspaceName,omitempty"`
	ReplicationGroupStatuses []ReplicationGroupStatus `json:"replicationGroupStatuses,omitempty"`
	ReplicationRegions []string `json:"replicationRegions,omitempty"`
	ReplicationStrategy string `json:"replicationStrategy,omitempty"`
	ResourceArn string `json:"resourceArn,omitempty"`
}

type GetTableAutoScalingSettingsRequest struct {
	KeyspaceName string `json:"keyspaceName,omitempty"`
	TableName string `json:"tableName,omitempty"`
}

type GetTableAutoScalingSettingsResponse struct {
	AutoScalingSpecification *AutoScalingSpecification `json:"autoScalingSpecification,omitempty"`
	KeyspaceName string `json:"keyspaceName,omitempty"`
	ReplicaSpecifications []ReplicaAutoScalingSpecification `json:"replicaSpecifications,omitempty"`
	ResourceArn string `json:"resourceArn,omitempty"`
	TableName string `json:"tableName,omitempty"`
}

type GetTableRequest struct {
	KeyspaceName string `json:"keyspaceName,omitempty"`
	TableName string `json:"tableName,omitempty"`
}

type GetTableResponse struct {
	CapacitySpecification *CapacitySpecificationSummary `json:"capacitySpecification,omitempty"`
	CdcSpecification *CdcSpecificationSummary `json:"cdcSpecification,omitempty"`
	ClientSideTimestamps *ClientSideTimestamps `json:"clientSideTimestamps,omitempty"`
	Comment *Comment `json:"comment,omitempty"`
	CreationTimestamp *time.Time `json:"creationTimestamp,omitempty"`
	DefaultTimeToLive int `json:"defaultTimeToLive,omitempty"`
	EncryptionSpecification *EncryptionSpecification `json:"encryptionSpecification,omitempty"`
	KeyspaceName string `json:"keyspaceName,omitempty"`
	LatestStreamArn *string `json:"latestStreamArn,omitempty"`
	PointInTimeRecovery *PointInTimeRecoverySummary `json:"pointInTimeRecovery,omitempty"`
	ReplicaSpecifications []ReplicaSpecificationSummary `json:"replicaSpecifications,omitempty"`
	ResourceArn string `json:"resourceArn,omitempty"`
	SchemaDefinition *SchemaDefinition `json:"schemaDefinition,omitempty"`
	Status *string `json:"status,omitempty"`
	TableName string `json:"tableName,omitempty"`
	Ttl *TimeToLive `json:"ttl,omitempty"`
	WarmThroughputSpecification *WarmThroughputSpecificationSummary `json:"warmThroughputSpecification,omitempty"`
}

type GetTypeRequest struct {
	KeyspaceName string `json:"keyspaceName,omitempty"`
	TypeName string `json:"typeName,omitempty"`
}

type GetTypeResponse struct {
	DirectParentTypes []string `json:"directParentTypes,omitempty"`
	DirectReferringTables []string `json:"directReferringTables,omitempty"`
	FieldDefinitions []FieldDefinition `json:"fieldDefinitions,omitempty"`
	KeyspaceArn string `json:"keyspaceArn,omitempty"`
	KeyspaceName string `json:"keyspaceName,omitempty"`
	LastModifiedTimestamp *time.Time `json:"lastModifiedTimestamp,omitempty"`
	MaxNestingDepth int `json:"maxNestingDepth,omitempty"`
	Status *string `json:"status,omitempty"`
	TypeName string `json:"typeName,omitempty"`
}

type KeyspaceSummary struct {
	KeyspaceName string `json:"keyspaceName,omitempty"`
	ReplicationRegions []string `json:"replicationRegions,omitempty"`
	ReplicationStrategy string `json:"replicationStrategy,omitempty"`
	ResourceArn string `json:"resourceArn,omitempty"`
}

type ListKeyspacesRequest struct {
	MaxResults int `json:"maxResults,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type ListKeyspacesResponse struct {
	Keyspaces []KeyspaceSummary `json:"keyspaces,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type ListTablesRequest struct {
	KeyspaceName string `json:"keyspaceName,omitempty"`
	MaxResults int `json:"maxResults,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type ListTablesResponse struct {
	NextToken *string `json:"nextToken,omitempty"`
	Tables []TableSummary `json:"tables,omitempty"`
}

type ListTagsForResourceRequest struct {
	MaxResults int `json:"maxResults,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
	ResourceArn string `json:"resourceArn,omitempty"`
}

type ListTagsForResourceResponse struct {
	NextToken *string `json:"nextToken,omitempty"`
	Tags []Tag `json:"tags,omitempty"`
}

type ListTypesRequest struct {
	KeyspaceName string `json:"keyspaceName,omitempty"`
	MaxResults int `json:"maxResults,omitempty"`
	NextToken *string `json:"nextToken,omitempty"`
}

type ListTypesResponse struct {
	NextToken *string `json:"nextToken,omitempty"`
	Types []string `json:"types,omitempty"`
}

type PartitionKey struct {
	Name string `json:"name,omitempty"`
}

type PointInTimeRecovery struct {
	Status string `json:"status,omitempty"`
}

type PointInTimeRecoverySummary struct {
	EarliestRestorableTimestamp *time.Time `json:"earliestRestorableTimestamp,omitempty"`
	Status string `json:"status,omitempty"`
}

type ReplicaAutoScalingSpecification struct {
	AutoScalingSpecification *AutoScalingSpecification `json:"autoScalingSpecification,omitempty"`
	Region *string `json:"region,omitempty"`
}

type ReplicaSpecification struct {
	ReadCapacityAutoScaling *AutoScalingSettings `json:"readCapacityAutoScaling,omitempty"`
	ReadCapacityUnits int64 `json:"readCapacityUnits,omitempty"`
	Region string `json:"region,omitempty"`
}

type ReplicaSpecificationSummary struct {
	CapacitySpecification *CapacitySpecificationSummary `json:"capacitySpecification,omitempty"`
	Region *string `json:"region,omitempty"`
	Status *string `json:"status,omitempty"`
	WarmThroughputSpecification *WarmThroughputSpecificationSummary `json:"warmThroughputSpecification,omitempty"`
}

type ReplicationGroupStatus struct {
	KeyspaceStatus string `json:"keyspaceStatus,omitempty"`
	Region string `json:"region,omitempty"`
	TablesReplicationProgress *string `json:"tablesReplicationProgress,omitempty"`
}

type ReplicationSpecification struct {
	RegionList []string `json:"regionList,omitempty"`
	ReplicationStrategy string `json:"replicationStrategy,omitempty"`
}

type RestoreTableRequest struct {
	AutoScalingSpecification *AutoScalingSpecification `json:"autoScalingSpecification,omitempty"`
	CapacitySpecificationOverride *CapacitySpecification `json:"capacitySpecificationOverride,omitempty"`
	EncryptionSpecificationOverride *EncryptionSpecification `json:"encryptionSpecificationOverride,omitempty"`
	PointInTimeRecoveryOverride *PointInTimeRecovery `json:"pointInTimeRecoveryOverride,omitempty"`
	ReplicaSpecifications []ReplicaSpecification `json:"replicaSpecifications,omitempty"`
	RestoreTimestamp *time.Time `json:"restoreTimestamp,omitempty"`
	SourceKeyspaceName string `json:"sourceKeyspaceName,omitempty"`
	SourceTableName string `json:"sourceTableName,omitempty"`
	TagsOverride []Tag `json:"tagsOverride,omitempty"`
	TargetKeyspaceName string `json:"targetKeyspaceName,omitempty"`
	TargetTableName string `json:"targetTableName,omitempty"`
}

type RestoreTableResponse struct {
	RestoredTableARN string `json:"restoredTableARN,omitempty"`
}

type SchemaDefinition struct {
	AllColumns []ColumnDefinition `json:"allColumns,omitempty"`
	ClusteringKeys []ClusteringKey `json:"clusteringKeys,omitempty"`
	PartitionKeys []PartitionKey `json:"partitionKeys,omitempty"`
	StaticColumns []StaticColumn `json:"staticColumns,omitempty"`
}

type StaticColumn struct {
	Name string `json:"name,omitempty"`
}

type TableSummary struct {
	KeyspaceName string `json:"keyspaceName,omitempty"`
	ResourceArn string `json:"resourceArn,omitempty"`
	TableName string `json:"tableName,omitempty"`
}

type Tag struct {
	Key string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

type TagResourceRequest struct {
	ResourceArn string `json:"resourceArn,omitempty"`
	Tags []Tag `json:"tags,omitempty"`
}

type TagResourceResponse struct {
}

type TargetTrackingScalingPolicyConfiguration struct {
	DisableScaleIn bool `json:"disableScaleIn,omitempty"`
	ScaleInCooldown int `json:"scaleInCooldown,omitempty"`
	ScaleOutCooldown int `json:"scaleOutCooldown,omitempty"`
	TargetValue float64 `json:"targetValue,omitempty"`
}

type TimeToLive struct {
	Status string `json:"status,omitempty"`
}

type UntagResourceRequest struct {
	ResourceArn string `json:"resourceArn,omitempty"`
	Tags []Tag `json:"tags,omitempty"`
}

type UntagResourceResponse struct {
}

type UpdateKeyspaceRequest struct {
	ClientSideTimestamps *ClientSideTimestamps `json:"clientSideTimestamps,omitempty"`
	KeyspaceName string `json:"keyspaceName,omitempty"`
	ReplicationSpecification ReplicationSpecification `json:"replicationSpecification,omitempty"`
}

type UpdateKeyspaceResponse struct {
	ResourceArn string `json:"resourceArn,omitempty"`
}

type UpdateTableRequest struct {
	AddColumns []ColumnDefinition `json:"addColumns,omitempty"`
	AutoScalingSpecification *AutoScalingSpecification `json:"autoScalingSpecification,omitempty"`
	CapacitySpecification *CapacitySpecification `json:"capacitySpecification,omitempty"`
	CdcSpecification *CdcSpecification `json:"cdcSpecification,omitempty"`
	ClientSideTimestamps *ClientSideTimestamps `json:"clientSideTimestamps,omitempty"`
	DefaultTimeToLive int `json:"defaultTimeToLive,omitempty"`
	EncryptionSpecification *EncryptionSpecification `json:"encryptionSpecification,omitempty"`
	KeyspaceName string `json:"keyspaceName,omitempty"`
	PointInTimeRecovery *PointInTimeRecovery `json:"pointInTimeRecovery,omitempty"`
	ReplicaSpecifications []ReplicaSpecification `json:"replicaSpecifications,omitempty"`
	TableName string `json:"tableName,omitempty"`
	Ttl *TimeToLive `json:"ttl,omitempty"`
	WarmThroughputSpecification *WarmThroughputSpecification `json:"warmThroughputSpecification,omitempty"`
}

type UpdateTableResponse struct {
	ResourceArn string `json:"resourceArn,omitempty"`
}

type WarmThroughputSpecification struct {
	ReadUnitsPerSecond int64 `json:"readUnitsPerSecond,omitempty"`
	WriteUnitsPerSecond int64 `json:"writeUnitsPerSecond,omitempty"`
}

type WarmThroughputSpecificationSummary struct {
	ReadUnitsPerSecond int64 `json:"readUnitsPerSecond,omitempty"`
	Status string `json:"status,omitempty"`
	WriteUnitsPerSecond int64 `json:"writeUnitsPerSecond,omitempty"`
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
	if err := gojson.Unmarshal(body, v); err != nil {
		return service.NewAWSError("InvalidParameterException",
			"Request body is not valid JSON.", http.StatusBadRequest)
	}
	return nil
}

// ── Handlers ─────────────────────────────────────────────────────────────────

func handleCreateKeyspace(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateKeyspaceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateKeyspace business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateKeyspace"})
}

func handleCreateTable(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateTableRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateTable business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateTable"})
}

func handleCreateType(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateTypeRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateType business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateType"})
}

func handleDeleteKeyspace(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteKeyspaceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteKeyspace business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteKeyspace"})
}

func handleDeleteTable(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteTableRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteTable business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteTable"})
}

func handleDeleteType(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteTypeRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteType business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteType"})
}

func handleGetKeyspace(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetKeyspaceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetKeyspace business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetKeyspace"})
}

func handleGetTable(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetTableRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetTable business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetTable"})
}

func handleGetTableAutoScalingSettings(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetTableAutoScalingSettingsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetTableAutoScalingSettings business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetTableAutoScalingSettings"})
}

func handleGetType(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req GetTypeRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement GetType business logic
	return jsonOK(map[string]any{"status": "ok", "action": "GetType"})
}

func handleListKeyspaces(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListKeyspacesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListKeyspaces business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListKeyspaces"})
}

func handleListTables(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListTablesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListTables business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListTables"})
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListTagsForResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListTagsForResource business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListTagsForResource"})
}

func handleListTypes(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListTypesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListTypes business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListTypes"})
}

func handleRestoreTable(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req RestoreTableRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement RestoreTable business logic
	return jsonOK(map[string]any{"status": "ok", "action": "RestoreTable"})
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

func handleUpdateKeyspace(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateKeyspaceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateKeyspace business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateKeyspace"})
}

func handleUpdateTable(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateTableRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateTable business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateTable"})
}

