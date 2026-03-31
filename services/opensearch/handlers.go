package opensearch

import (
	"encoding/json"
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
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
	if err := json.Unmarshal(body, v); err != nil {
		return service.NewAWSError("ValidationException", "Invalid JSON.", http.StatusBadRequest)
	}
	return nil
}

type tag struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

// ---- Domain JSON types ----

type clusterConfigJSON struct {
	InstanceType           string `json:"InstanceType,omitempty"`
	InstanceCount          int    `json:"InstanceCount,omitempty"`
	DedicatedMasterEnabled bool   `json:"DedicatedMasterEnabled,omitempty"`
	DedicatedMasterType    string `json:"DedicatedMasterType,omitempty"`
	DedicatedMasterCount   int    `json:"DedicatedMasterCount,omitempty"`
}

type ebsOptionsJSON struct {
	EBSEnabled bool   `json:"EBSEnabled,omitempty"`
	VolumeType string `json:"VolumeType,omitempty"`
	VolumeSize int    `json:"VolumeSize,omitempty"`
}

type domainStatusJSON struct {
	DomainName    string            `json:"DomainName"`
	ARN           string            `json:"ARN"`
	DomainId      string            `json:"DomainId"`
	EngineVersion string            `json:"EngineVersion"`
	Endpoint      string            `json:"Endpoint,omitempty"`
	Processing    bool              `json:"Processing"`
	Created       bool              `json:"Created"`
	Deleted       bool              `json:"Deleted"`
	ClusterConfig clusterConfigJSON `json:"ClusterConfig"`
	EBSOptions    ebsOptionsJSON    `json:"EBSOptions"`
}

func toDomainStatusJSON(d *Domain) domainStatusJSON {
	return domainStatusJSON{
		DomainName:    d.DomainName,
		ARN:           d.ARN,
		DomainId:      d.DomainId,
		EngineVersion: d.EngineVersion,
		Endpoint:      d.Endpoint,
		Processing:    d.Processing,
		Created:       d.Created,
		Deleted:       d.Deleted,
		ClusterConfig: clusterConfigJSON{
			InstanceType: d.ClusterConfig.InstanceType, InstanceCount: d.ClusterConfig.InstanceCount,
			DedicatedMasterEnabled: d.ClusterConfig.DedicatedMasterEnabled,
			DedicatedMasterType: d.ClusterConfig.DedicatedMasterType,
			DedicatedMasterCount: d.ClusterConfig.DedicatedMasterCount,
		},
		EBSOptions: ebsOptionsJSON{
			EBSEnabled: d.EBSOptions.EBSEnabled, VolumeType: d.EBSOptions.VolumeType, VolumeSize: d.EBSOptions.VolumeSize,
		},
	}
}

// ---- CreateDomain ----

type createDomainRequest struct {
	DomainName    string            `json:"DomainName"`
	EngineVersion string            `json:"EngineVersion"`
	ClusterConfig clusterConfigJSON `json:"ClusterConfig"`
	EBSOptions    ebsOptionsJSON    `json:"EBSOptions"`
	TagList       []tag             `json:"TagList"`
}

func handleCreateDomain(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createDomainRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.DomainName == "" {
		return jsonErr(service.ErrValidation("DomainName is required."))
	}
	tags := make(map[string]string)
	for _, t := range req.TagList {
		tags[t.Key] = t.Value
	}
	cc := ClusterConfig{
		InstanceType: req.ClusterConfig.InstanceType, InstanceCount: req.ClusterConfig.InstanceCount,
		DedicatedMasterEnabled: req.ClusterConfig.DedicatedMasterEnabled,
		DedicatedMasterType: req.ClusterConfig.DedicatedMasterType,
		DedicatedMasterCount: req.ClusterConfig.DedicatedMasterCount,
	}
	ebs := EBSOptions{EBSEnabled: req.EBSOptions.EBSEnabled, VolumeType: req.EBSOptions.VolumeType, VolumeSize: req.EBSOptions.VolumeSize}
	d, ok := store.CreateDomain(req.DomainName, req.EngineVersion, cc, ebs, tags)
	if !ok {
		return jsonErr(service.ErrAlreadyExists("Domain", req.DomainName))
	}
	return jsonOK(map[string]any{"DomainStatus": toDomainStatusJSON(d)})
}

// ---- DescribeDomain ----

type describeDomainRequest struct {
	DomainName string `json:"DomainName"`
}

func handleDescribeDomain(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeDomainRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	d, ok := store.GetDomain(req.DomainName)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Domain "+req.DomainName+" not found.", http.StatusNotFound))
	}
	return jsonOK(map[string]any{"DomainStatus": toDomainStatusJSON(d)})
}

// ---- ListDomainNames ----

type domainInfo struct {
	DomainName    string `json:"DomainName"`
	EngineType    string `json:"EngineType"`
}

func handleListDomainNames(_ *service.RequestContext, store *Store) (*service.Response, error) {
	domains := store.ListDomainNames()
	list := make([]domainInfo, 0, len(domains))
	for _, d := range domains {
		list = append(list, domainInfo{DomainName: d.DomainName, EngineType: "OpenSearch"})
	}
	return jsonOK(map[string]any{"DomainNames": list})
}

// ---- DeleteDomain ----

type deleteDomainRequest struct {
	DomainName string `json:"DomainName"`
}

func handleDeleteDomain(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteDomainRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	d, ok := store.DeleteDomain(req.DomainName)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Domain "+req.DomainName+" not found.", http.StatusNotFound))
	}
	return jsonOK(map[string]any{"DomainStatus": toDomainStatusJSON(d)})
}

// ---- UpdateDomainConfig ----

type updateDomainConfigRequest struct {
	DomainName    string             `json:"DomainName"`
	EngineVersion string             `json:"EngineVersion"`
	ClusterConfig *clusterConfigJSON `json:"ClusterConfig"`
	EBSOptions    *ebsOptionsJSON    `json:"EBSOptions"`
}

func handleUpdateDomainConfig(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req updateDomainConfigRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	var cc *ClusterConfig
	if req.ClusterConfig != nil {
		cc = &ClusterConfig{
			InstanceType: req.ClusterConfig.InstanceType, InstanceCount: req.ClusterConfig.InstanceCount,
			DedicatedMasterEnabled: req.ClusterConfig.DedicatedMasterEnabled,
		}
	}
	var ebs *EBSOptions
	if req.EBSOptions != nil {
		ebs = &EBSOptions{EBSEnabled: req.EBSOptions.EBSEnabled, VolumeType: req.EBSOptions.VolumeType, VolumeSize: req.EBSOptions.VolumeSize}
	}
	d, ok := store.UpdateDomainConfig(req.DomainName, req.EngineVersion, cc, ebs)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Domain "+req.DomainName+" not found.", http.StatusNotFound))
	}
	return jsonOK(map[string]any{"DomainConfig": toDomainStatusJSON(d)})
}

// ---- DescribeDomainConfig ----

type describeDomainConfigRequest struct {
	DomainName string `json:"DomainName"`
}

func handleDescribeDomainConfig(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeDomainConfigRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	d, ok := store.GetDomain(req.DomainName)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Domain "+req.DomainName+" not found.", http.StatusNotFound))
	}
	return jsonOK(map[string]any{"DomainConfig": toDomainStatusJSON(d)})
}

// ---- Tag handlers ----

type addTagsRequest struct {
	ARN     string `json:"ARN"`
	TagList []tag  `json:"TagList"`
}

func handleAddTags(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req addTagsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	tags := make(map[string]string)
	for _, t := range req.TagList {
		tags[t.Key] = t.Value
	}
	store.AddTags(req.ARN, tags)
	return jsonOK(struct{}{})
}

type removeTagsRequest struct {
	ARN     string   `json:"ARN"`
	TagKeys []string `json:"TagKeys"`
}

func handleRemoveTags(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req removeTagsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	store.RemoveTags(req.ARN, req.TagKeys)
	return jsonOK(struct{}{})
}

type listTagsRequest struct {
	ARN string `json:"ARN"`
}

func handleListTags(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listTagsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	tags, _ := store.ListTags(req.ARN)
	if tags == nil {
		tags = make(map[string]string)
	}
	tagList := make([]tag, 0, len(tags))
	for k, v := range tags {
		tagList = append(tagList, tag{Key: k, Value: v})
	}
	return jsonOK(map[string]any{"TagList": tagList})
}

// ---- Upgrade handlers ----

type upgradeDomainRequest struct {
	DomainName    string `json:"DomainName"`
	TargetVersion string `json:"TargetVersion"`
}

func handleUpgradeDomain(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req upgradeDomainRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	status, ok := store.UpgradeDomain(req.DomainName, req.TargetVersion)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Domain not found.", http.StatusNotFound))
	}
	return jsonOK(map[string]any{
		"DomainName": req.DomainName, "TargetVersion": req.TargetVersion,
		"UpgradeId": randomHex(16), "StepStatus": status.StepStatus,
	})
}

type getUpgradeStatusRequest struct {
	DomainName string `json:"DomainName"`
}

func handleGetUpgradeStatus(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getUpgradeStatusRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	status, ok := store.GetUpgradeStatus(req.DomainName)
	if !ok {
		return jsonOK(map[string]any{"StepStatus": "NOT_ELIGIBLE"})
	}
	return jsonOK(map[string]any{
		"StepStatus": status.StepStatus, "UpgradeName": status.UpgradeName, "UpgradeStep": status.UpgradeStep,
	})
}
