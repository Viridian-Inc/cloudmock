package opensearch

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

// ---- IndexDocument ----

type indexDocumentRequest struct {
	DomainName string         `json:"DomainName"`
	Index      string         `json:"Index"`
	DocumentId string         `json:"DocumentId"`
	Document   map[string]any `json:"Document"`
}

func handleIndexDocument(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req indexDocumentRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.DomainName == "" || req.Index == "" {
		return jsonErr(service.ErrValidation("DomainName and Index are required."))
	}
	docID, ok := store.IndexDocument(req.DomainName, req.Index, req.DocumentId, req.Document)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Domain "+req.DomainName+" not found.", http.StatusNotFound))
	}
	return jsonOK(map[string]any{
		"_index":  req.Index,
		"_id":     docID,
		"result":  "created",
		"_version": 1,
	})
}

// ---- Search ----

type searchRequest struct {
	DomainName string         `json:"DomainName"`
	Index      string         `json:"Index"`
	Query      map[string]any `json:"Query"`
}

func handleSearch(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req searchRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.DomainName == "" || req.Index == "" {
		return jsonErr(service.ErrValidation("DomainName and Index are required."))
	}
	docs, ok := store.SearchDocuments(req.DomainName, req.Index, req.Query)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Domain "+req.DomainName+" not found.", http.StatusNotFound))
	}
	hits := make([]map[string]any, len(docs))
	for i, d := range docs {
		hits[i] = map[string]any{
			"_index":  d.Index,
			"_id":     d.ID,
			"_source": d.Source,
		}
	}
	return jsonOK(map[string]any{
		"hits": map[string]any{
			"total": map[string]any{"value": len(docs), "relation": "eq"},
			"hits":  hits,
		},
	})
}

// ---- ClusterHealth ----

type clusterHealthRequest struct {
	DomainName string `json:"DomainName"`
}

func handleClusterHealth(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req clusterHealthRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	health, ok := store.ClusterHealth(req.DomainName)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Domain "+req.DomainName+" not found.", http.StatusNotFound))
	}
	return jsonOK(map[string]any{
		"cluster_name":    req.DomainName,
		"status":          health,
		"number_of_nodes": 1,
	})
}

// ---- DescribeDomains ----

type describeDomainsRequest struct {
	DomainNames []string `json:"DomainNames"`
}

func handleDescribeDomains(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeDomainsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	domains := store.DescribeDomains(req.DomainNames)
	statuses := make([]domainStatusJSON, 0, len(domains))
	for _, d := range domains {
		statuses = append(statuses, toDomainStatusJSON(d))
	}
	return jsonOK(map[string]any{"DomainStatusList": statuses})
}

// ---- GetCompatibleVersions ----

type getCompatibleVersionsRequest struct {
	DomainName string `json:"DomainName"`
}

func handleGetCompatibleVersions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getCompatibleVersionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	versions := store.GetCompatibleVersions(req.DomainName)
	list := make([]map[string]any, 0, len(versions))
	for _, v := range versions {
		list = append(list, map[string]any{
			"SourceVersion":  v.SourceVersion,
			"TargetVersions": v.TargetVersions,
		})
	}
	return jsonOK(map[string]any{"CompatibleVersions": list})
}

// ---- VPC Endpoints ----

type vpcEndpointJSON struct {
	VpcEndpointId string `json:"VpcEndpointId"`
	DomainArn     string `json:"DomainArn"`
	Status        string `json:"Status"`
	Endpoint      string `json:"Endpoint,omitempty"`
	VpcOptions    map[string]any `json:"VpcOptions,omitempty"`
}

func toVpcEndpointJSON(ep *VpcEndpoint) vpcEndpointJSON {
	return vpcEndpointJSON{
		VpcEndpointId: ep.VpcEndpointID,
		DomainArn:     ep.DomainArn,
		Status:        ep.Status,
		Endpoint:      ep.Endpoint,
		VpcOptions: map[string]any{
			"VPCId":            ep.VpcOptions.VpcID,
			"SubnetIds":        ep.VpcOptions.SubnetIDs,
			"SecurityGroupIds": ep.VpcOptions.SecurityGroupIDs,
		},
	}
}

type createVpcEndpointRequest struct {
	DomainArn  string `json:"DomainArn"`
	VpcOptions struct {
		VPCId            string   `json:"VPCId"`
		SubnetIds        []string `json:"SubnetIds"`
		SecurityGroupIds []string `json:"SecurityGroupIds"`
	} `json:"VpcOptions"`
}

func handleCreateVpcEndpoint(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createVpcEndpointRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.DomainArn == "" {
		return jsonErr(service.ErrValidation("DomainArn is required."))
	}
	opts := VpcOptions{
		VpcID:            req.VpcOptions.VPCId,
		SubnetIDs:        req.VpcOptions.SubnetIds,
		SecurityGroupIDs: req.VpcOptions.SecurityGroupIds,
	}
	ep, _ := store.CreateVpcEndpoint(req.DomainArn, opts)
	return jsonOK(map[string]any{"VpcEndpoint": toVpcEndpointJSON(ep)})
}

type describeVpcEndpointsRequest struct {
	VpcEndpointIds []string `json:"VpcEndpointIds"`
}

func handleDescribeVpcEndpoints(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeVpcEndpointsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	endpoints := store.DescribeVpcEndpoints(req.VpcEndpointIds)
	list := make([]vpcEndpointJSON, 0, len(endpoints))
	for _, ep := range endpoints {
		list = append(list, toVpcEndpointJSON(ep))
	}
	return jsonOK(map[string]any{"VpcEndpoints": list})
}

type listVpcEndpointsRequest struct {
	DomainArn string `json:"DomainArn"`
}

func handleListVpcEndpoints(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listVpcEndpointsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	endpoints := store.ListVpcEndpoints(req.DomainArn)
	list := make([]vpcEndpointJSON, 0, len(endpoints))
	for _, ep := range endpoints {
		list = append(list, toVpcEndpointJSON(ep))
	}
	return jsonOK(map[string]any{"VpcEndpoints": list})
}

type deleteVpcEndpointRequest struct {
	VpcEndpointId string `json:"VpcEndpointId"`
}

func handleDeleteVpcEndpoint(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteVpcEndpointRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.VpcEndpointId == "" {
		return jsonErr(service.ErrValidation("VpcEndpointId is required."))
	}
	ep, ok := store.DeleteVpcEndpoint(req.VpcEndpointId)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"VPC endpoint "+req.VpcEndpointId+" not found.", http.StatusNotFound))
	}
	return jsonOK(map[string]any{"VpcEndpointSummary": toVpcEndpointJSON(ep)})
}
