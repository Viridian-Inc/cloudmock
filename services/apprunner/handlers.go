package apprunner

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
		return service.NewAWSError("InvalidRequestException", "Invalid JSON in request body.", http.StatusBadRequest)
	}
	return nil
}

func strSlice(v any) []string {
	if arr, ok := v.([]any); ok {
		out := make([]string, 0, len(arr))
		for _, item := range arr {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

// ---- service response helper ----

func serviceToJSON(svc *Service) map[string]any {
	m := map[string]any{
		"ServiceId":   svc.ServiceID,
		"ServiceName": svc.ServiceName,
		"ServiceArn":  svc.ServiceARN,
		"ServiceUrl":  svc.ServiceURL,
		"Status":      svc.Status,
		"CreatedAt":   svc.CreatedAt.Unix(),
		"UpdatedAt":   svc.UpdatedAt.Unix(),
	}
	if svc.Tags != nil {
		m["Tags"] = svc.Tags
	}
	return m
}

// ---- CreateService ----

type createServiceRequest struct {
	ServiceName             string            `json:"ServiceName"`
	SourceConfiguration     map[string]any    `json:"SourceConfiguration"`
	InstanceConfiguration   map[string]any    `json:"InstanceConfiguration"`
	Tags                    map[string]string `json:"Tags"`
}

func handleCreateService(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createServiceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ServiceName == "" {
		return jsonErr(service.ErrValidation("ServiceName is required."))
	}

	var srcCfg *SourceConfiguration
	if req.SourceConfiguration != nil {
		srcCfg = &SourceConfiguration{}
		if imgRepo, ok := req.SourceConfiguration["ImageRepository"].(map[string]any); ok {
			srcCfg.ImageRepository = &ImageRepository{}
			if id, ok := imgRepo["ImageIdentifier"].(string); ok {
				srcCfg.ImageRepository.ImageIdentifier = id
			}
			if t, ok := imgRepo["ImageRepositoryType"].(string); ok {
				srcCfg.ImageRepository.ImageRepositoryType = t
			}
		}
		if authDeployments, ok := req.SourceConfiguration["AutoDeploymentsEnabled"].(bool); ok {
			srcCfg.AutoDeploymentsEnabled = authDeployments
		}
	}

	var instCfg *InstanceConfiguration
	if req.InstanceConfiguration != nil {
		instCfg = &InstanceConfiguration{}
		if cpu, ok := req.InstanceConfiguration["Cpu"].(string); ok {
			instCfg.CPU = cpu
		}
		if mem, ok := req.InstanceConfiguration["Memory"].(string); ok {
			instCfg.Memory = mem
		}
	}

	svc, err := store.CreateService(req.ServiceName, srcCfg, instCfg, req.Tags)
	if err != nil {
		return jsonErr(service.ErrAlreadyExists("Service", req.ServiceName))
	}
	return jsonOK(map[string]any{
		"Service":         serviceToJSON(svc),
		"OperationId":     newID()[:8],
	})
}

// ---- DescribeService ----

type describeServiceRequest struct {
	ServiceArn string `json:"ServiceArn"`
}

func handleDescribeService(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeServiceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ServiceArn == "" {
		return jsonErr(service.ErrValidation("ServiceArn is required."))
	}
	svc, ok := store.GetService(req.ServiceArn)
	if !ok {
		return jsonErr(service.ErrNotFound("Service", req.ServiceArn))
	}
	return jsonOK(map[string]any{"Service": serviceToJSON(svc)})
}

// ---- ListServices ----

func handleListServices(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	services := store.ListServices()
	items := make([]map[string]any, 0, len(services))
	for _, svc := range services {
		items = append(items, serviceToJSON(svc))
	}
	return jsonOK(map[string]any{"ServiceSummaryList": items})
}

// ---- UpdateService ----

type updateServiceRequest struct {
	ServiceArn            string         `json:"ServiceArn"`
	SourceConfiguration   map[string]any `json:"SourceConfiguration"`
	InstanceConfiguration map[string]any `json:"InstanceConfiguration"`
}

func handleUpdateService(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req updateServiceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ServiceArn == "" {
		return jsonErr(service.ErrValidation("ServiceArn is required."))
	}
	svc, ok := store.UpdateService(req.ServiceArn, nil, nil)
	if !ok {
		return jsonErr(service.ErrNotFound("Service", req.ServiceArn))
	}
	return jsonOK(map[string]any{
		"Service":     serviceToJSON(svc),
		"OperationId": newID()[:8],
	})
}

// ---- DeleteService ----

type deleteServiceRequest struct {
	ServiceArn string `json:"ServiceArn"`
}

func handleDeleteService(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteServiceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ServiceArn == "" {
		return jsonErr(service.ErrValidation("ServiceArn is required."))
	}
	svc, ok := store.DeleteService(req.ServiceArn)
	if !ok {
		return jsonErr(service.ErrNotFound("Service", req.ServiceArn))
	}
	return jsonOK(map[string]any{
		"Service":     serviceToJSON(svc),
		"OperationId": newID()[:8],
	})
}

// ---- PauseService ----

type pauseServiceRequest struct {
	ServiceArn string `json:"ServiceArn"`
}

func handlePauseService(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req pauseServiceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ServiceArn == "" {
		return jsonErr(service.ErrValidation("ServiceArn is required."))
	}
	svc, ok := store.PauseService(req.ServiceArn)
	if !ok {
		return jsonErr(service.ErrNotFound("Service", req.ServiceArn))
	}
	return jsonOK(map[string]any{
		"Service":     serviceToJSON(svc),
		"OperationId": newID()[:8],
	})
}

// ---- ResumeService ----

type resumeServiceRequest struct {
	ServiceArn string `json:"ServiceArn"`
}

func handleResumeService(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req resumeServiceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ServiceArn == "" {
		return jsonErr(service.ErrValidation("ServiceArn is required."))
	}
	svc, ok := store.ResumeService(req.ServiceArn)
	if !ok {
		return jsonErr(service.ErrNotFound("Service", req.ServiceArn))
	}
	return jsonOK(map[string]any{
		"Service":     serviceToJSON(svc),
		"OperationId": newID()[:8],
	})
}

// ---- CreateConnection ----

type createConnectionRequest struct {
	ConnectionName string            `json:"ConnectionName"`
	ProviderType   string            `json:"ProviderType"`
	Tags           map[string]string `json:"Tags"`
}

func handleCreateConnection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createConnectionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ConnectionName == "" || req.ProviderType == "" {
		return jsonErr(service.ErrValidation("ConnectionName and ProviderType are required."))
	}
	conn, err := store.CreateConnection(req.ConnectionName, req.ProviderType, req.Tags)
	if err != nil {
		return jsonErr(service.ErrAlreadyExists("Connection", req.ConnectionName))
	}
	return jsonOK(map[string]any{
		"Connection": map[string]any{
			"ConnectionName": conn.ConnectionName,
			"ConnectionArn":  conn.ConnectionARN,
			"ProviderType":   conn.ProviderType,
			"Status":         conn.Status,
		},
	})
}

// ---- DescribeConnection ----

type describeConnectionRequest struct {
	ConnectionArn string `json:"ConnectionArn"`
}

func handleDescribeConnection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeConnectionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ConnectionArn == "" {
		return jsonErr(service.ErrValidation("ConnectionArn is required."))
	}
	conn, ok := store.GetConnection(req.ConnectionArn)
	if !ok {
		return jsonErr(service.ErrNotFound("Connection", req.ConnectionArn))
	}
	return jsonOK(map[string]any{
		"Connection": map[string]any{
			"ConnectionName": conn.ConnectionName,
			"ConnectionArn":  conn.ConnectionARN,
			"ProviderType":   conn.ProviderType,
			"Status":         conn.Status,
		},
	})
}

// ---- CreateAutoScalingConfiguration ----

type createASCRequest struct {
	AutoScalingConfigurationName string            `json:"AutoScalingConfigurationName"`
	MinSize                      int               `json:"MinSize"`
	MaxSize                      int               `json:"MaxSize"`
	MaxConcurrency               int               `json:"MaxConcurrency"`
	Tags                         map[string]string `json:"Tags"`
}

func handleCreateAutoScalingConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createASCRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.AutoScalingConfigurationName == "" {
		return jsonErr(service.ErrValidation("AutoScalingConfigurationName is required."))
	}
	if req.MaxSize == 0 {
		req.MaxSize = 25
	}
	if req.MaxConcurrency == 0 {
		req.MaxConcurrency = 100
	}
	if req.MinSize == 0 {
		req.MinSize = 1
	}
	asc, _ := store.CreateAutoScalingConfiguration(req.AutoScalingConfigurationName, req.MinSize, req.MaxSize, req.MaxConcurrency, req.Tags)
	return jsonOK(map[string]any{
		"AutoScalingConfiguration": map[string]any{
			"AutoScalingConfigurationArn":      asc.AutoScalingConfigurationARN,
			"AutoScalingConfigurationName":     asc.AutoScalingConfigurationName,
			"AutoScalingConfigurationRevision": asc.AutoScalingConfigurationRevision,
			"Latest":                           asc.Latest,
			"Status":                           asc.Status,
			"MinSize":                          asc.MinSize,
			"MaxSize":                          asc.MaxSize,
			"MaxConcurrency":                   asc.MaxConcurrency,
		},
	})
}

// ---- DescribeAutoScalingConfiguration ----

type describeASCRequest struct {
	AutoScalingConfigurationArn string `json:"AutoScalingConfigurationArn"`
}

func handleDescribeAutoScalingConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeASCRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.AutoScalingConfigurationArn == "" {
		return jsonErr(service.ErrValidation("AutoScalingConfigurationArn is required."))
	}
	asc, ok := store.DescribeAutoScalingConfiguration(req.AutoScalingConfigurationArn)
	if !ok {
		return jsonErr(service.ErrNotFound("AutoScalingConfiguration", req.AutoScalingConfigurationArn))
	}
	return jsonOK(map[string]any{
		"AutoScalingConfiguration": map[string]any{
			"AutoScalingConfigurationArn":      asc.AutoScalingConfigurationARN,
			"AutoScalingConfigurationName":     asc.AutoScalingConfigurationName,
			"AutoScalingConfigurationRevision": asc.AutoScalingConfigurationRevision,
			"Latest":                           asc.Latest,
			"Status":                           asc.Status,
			"MinSize":                          asc.MinSize,
			"MaxSize":                          asc.MaxSize,
			"MaxConcurrency":                   asc.MaxConcurrency,
		},
	})
}

// ---- ListAutoScalingConfigurations ----

type listASCRequest struct {
	AutoScalingConfigurationName string `json:"AutoScalingConfigurationName"`
}

func handleListAutoScalingConfigurations(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listASCRequest
	parseJSON(ctx.Body, &req)
	configs := store.ListAutoScalingConfigurations(req.AutoScalingConfigurationName)
	items := make([]map[string]any, 0, len(configs))
	for _, asc := range configs {
		items = append(items, map[string]any{
			"AutoScalingConfigurationArn":      asc.AutoScalingConfigurationARN,
			"AutoScalingConfigurationName":     asc.AutoScalingConfigurationName,
			"AutoScalingConfigurationRevision": asc.AutoScalingConfigurationRevision,
			"Latest":                           asc.Latest,
			"Status":                           asc.Status,
		})
	}
	return jsonOK(map[string]any{"AutoScalingConfigurationSummaryList": items})
}

// ---- DeleteAutoScalingConfiguration ----

type deleteASCRequest struct {
	AutoScalingConfigurationArn string `json:"AutoScalingConfigurationArn"`
}

func handleDeleteAutoScalingConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteASCRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.AutoScalingConfigurationArn == "" {
		return jsonErr(service.ErrValidation("AutoScalingConfigurationArn is required."))
	}
	if !store.DeleteAutoScalingConfiguration(req.AutoScalingConfigurationArn) {
		return jsonErr(service.ErrNotFound("AutoScalingConfiguration", req.AutoScalingConfigurationArn))
	}
	return jsonOK(map[string]any{"AutoScalingConfiguration": map[string]any{"Status": "INACTIVE"}})
}

// ---- CreateVpcConnector ----

type createVpcConnectorRequest struct {
	VpcConnectorName string            `json:"VpcConnectorName"`
	Subnets          []string          `json:"Subnets"`
	SecurityGroups   []string          `json:"SecurityGroups"`
	Tags             map[string]string `json:"Tags"`
}

func handleCreateVpcConnector(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createVpcConnectorRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.VpcConnectorName == "" {
		return jsonErr(service.ErrValidation("VpcConnectorName is required."))
	}
	vc, _ := store.CreateVpcConnector(req.VpcConnectorName, req.Subnets, req.SecurityGroups, req.Tags)
	return jsonOK(map[string]any{
		"VpcConnector": map[string]any{
			"VpcConnectorArn":      vc.VpcConnectorARN,
			"VpcConnectorName":     vc.VpcConnectorName,
			"VpcConnectorRevision": vc.VpcConnectorRevision,
			"Subnets":              vc.Subnets,
			"SecurityGroups":       vc.SecurityGroups,
			"Status":               vc.Status,
		},
	})
}

// ---- DescribeVpcConnector ----

type describeVpcConnectorRequest struct {
	VpcConnectorArn string `json:"VpcConnectorArn"`
}

func handleDescribeVpcConnector(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeVpcConnectorRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.VpcConnectorArn == "" {
		return jsonErr(service.ErrValidation("VpcConnectorArn is required."))
	}
	vc, ok := store.DescribeVpcConnector(req.VpcConnectorArn)
	if !ok {
		return jsonErr(service.ErrNotFound("VpcConnector", req.VpcConnectorArn))
	}
	return jsonOK(map[string]any{
		"VpcConnector": map[string]any{
			"VpcConnectorArn":      vc.VpcConnectorARN,
			"VpcConnectorName":     vc.VpcConnectorName,
			"VpcConnectorRevision": vc.VpcConnectorRevision,
			"Subnets":              vc.Subnets,
			"SecurityGroups":       vc.SecurityGroups,
			"Status":               vc.Status,
		},
	})
}

// ---- ListVpcConnectors ----

func handleListVpcConnectors(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	connectors := store.ListVpcConnectors()
	items := make([]map[string]any, 0, len(connectors))
	for _, vc := range connectors {
		items = append(items, map[string]any{
			"VpcConnectorArn":      vc.VpcConnectorARN,
			"VpcConnectorName":     vc.VpcConnectorName,
			"VpcConnectorRevision": vc.VpcConnectorRevision,
			"Status":               vc.Status,
		})
	}
	return jsonOK(map[string]any{"VpcConnectors": items})
}

// ---- DeleteVpcConnector ----

type deleteVpcConnectorRequest struct {
	VpcConnectorArn string `json:"VpcConnectorArn"`
}

func handleDeleteVpcConnector(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteVpcConnectorRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.VpcConnectorArn == "" {
		return jsonErr(service.ErrValidation("VpcConnectorArn is required."))
	}
	if !store.DeleteVpcConnector(req.VpcConnectorArn) {
		return jsonErr(service.ErrNotFound("VpcConnector", req.VpcConnectorArn))
	}
	return jsonOK(map[string]any{"VpcConnector": map[string]any{"Status": "INACTIVE"}})
}

// ---- TagResource ----

type tagResourceRequest struct {
	ResourceArn string            `json:"ResourceArn"`
	Tags        map[string]string `json:"Tags"`
}

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req tagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ResourceArn == "" {
		return jsonErr(service.ErrValidation("ResourceArn is required."))
	}
	store.TagResource(req.ResourceArn, req.Tags)
	return jsonOK(map[string]any{})
}

// ---- UntagResource ----

type untagResourceRequest struct {
	ResourceArn string   `json:"ResourceArn"`
	TagKeys     []string `json:"TagKeys"`
}

func handleUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req untagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ResourceArn == "" {
		return jsonErr(service.ErrValidation("ResourceArn is required."))
	}
	store.UntagResource(req.ResourceArn, req.TagKeys)
	return jsonOK(map[string]any{})
}

// ---- ListTagsForResource ----

type listTagsRequest struct {
	ResourceArn string `json:"ResourceArn"`
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listTagsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ResourceArn == "" {
		return jsonErr(service.ErrValidation("ResourceArn is required."))
	}
	tags := store.ListTagsForResource(req.ResourceArn)
	return jsonOK(map[string]any{"Tags": tags})
}
