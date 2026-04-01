package servicediscovery

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

func emptyOK() (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: struct{}{}, Format: service.FormatJSON}, nil
}

func parseJSON(body []byte, v any) *service.AWSError {
	if len(body) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, v); err != nil {
		return service.NewAWSError("InvalidInput", "Invalid JSON in request body.", http.StatusBadRequest)
	}
	return nil
}

// ---- CreateHttpNamespace ----

type createHttpNamespaceRequest struct {
	Name        string            `json:"Name"`
	Description string            `json:"Description"`
	Tags        []tagJSON         `json:"Tags"`
}

type tagJSON struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

type createNamespaceResponse struct {
	OperationId string `json:"OperationId"`
}

func tagsToMap(tags []tagJSON) map[string]string {
	m := make(map[string]string, len(tags))
	for _, t := range tags {
		m[t.Key] = t.Value
	}
	return m
}

func mapToTags(m map[string]string) []tagJSON {
	tags := make([]tagJSON, 0, len(m))
	for k, v := range m {
		tags = append(tags, tagJSON{Key: k, Value: v})
	}
	return tags
}

func handleCreateHttpNamespace(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createHttpNamespaceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}
	ns, opID := store.CreateNamespace(req.Name, req.Description, NamespaceHTTP, "", tagsToMap(req.Tags))
	if ns == nil {
		return jsonErr(service.NewAWSError("NamespaceAlreadyExists", "Namespace "+req.Name+" already exists.", http.StatusConflict))
	}
	return jsonOK(&createNamespaceResponse{OperationId: opID})
}

// ---- CreatePrivateDnsNamespace ----

type createPrivateDnsNamespaceRequest struct {
	Name        string    `json:"Name"`
	Description string    `json:"Description"`
	Vpc         string    `json:"Vpc"`
	Tags        []tagJSON `json:"Tags"`
}

func handleCreatePrivateDnsNamespace(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createPrivateDnsNamespaceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Name == "" || req.Vpc == "" {
		return jsonErr(service.ErrValidation("Name and Vpc are required."))
	}
	ns, opID := store.CreateNamespace(req.Name, req.Description, NamespacePrivateDNS, req.Vpc, tagsToMap(req.Tags))
	if ns == nil {
		return jsonErr(service.NewAWSError("NamespaceAlreadyExists", "Namespace "+req.Name+" already exists.", http.StatusConflict))
	}
	return jsonOK(&createNamespaceResponse{OperationId: opID})
}

// ---- CreatePublicDnsNamespace ----

type createPublicDnsNamespaceRequest struct {
	Name        string    `json:"Name"`
	Description string    `json:"Description"`
	Tags        []tagJSON `json:"Tags"`
}

func handleCreatePublicDnsNamespace(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createPublicDnsNamespaceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}
	ns, opID := store.CreateNamespace(req.Name, req.Description, NamespacePublicDNS, "", tagsToMap(req.Tags))
	if ns == nil {
		return jsonErr(service.NewAWSError("NamespaceAlreadyExists", "Namespace "+req.Name+" already exists.", http.StatusConflict))
	}
	return jsonOK(&createNamespaceResponse{OperationId: opID})
}

// ---- GetNamespace ----

type getNamespaceRequest struct {
	Id string `json:"Id"`
}

type namespaceJSON struct {
	Id           string            `json:"Id"`
	Arn          string            `json:"Arn"`
	Name         string            `json:"Name"`
	Type         string            `json:"Type"`
	Description  string            `json:"Description,omitempty"`
	Properties   *nsPropertiesJSON `json:"Properties,omitempty"`
}

type nsPropertiesJSON struct {
	DnsProperties *dnsPropertiesJSON `json:"DnsProperties,omitempty"`
	HttpProperties *httpPropertiesJSON `json:"HttpProperties,omitempty"`
}

type dnsPropertiesJSON struct {
	HostedZoneId string `json:"HostedZoneId"`
}

type httpPropertiesJSON struct {
	HttpName string `json:"HttpName"`
}

type getNamespaceResponse struct {
	Namespace namespaceJSON `json:"Namespace"`
}

func nsToJSON(ns *Namespace) namespaceJSON {
	j := namespaceJSON{
		Id: ns.ID, Arn: ns.ARN, Name: ns.Name,
		Type: string(ns.Type), Description: ns.Description,
	}
	if ns.Type == NamespacePrivateDNS || ns.Type == NamespacePublicDNS {
		j.Properties = &nsPropertiesJSON{
			DnsProperties: &dnsPropertiesJSON{HostedZoneId: ns.HostedZoneID},
		}
	} else {
		j.Properties = &nsPropertiesJSON{
			HttpProperties: &httpPropertiesJSON{HttpName: ns.Name},
		}
	}
	return j
}

func handleGetNamespace(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getNamespaceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Id == "" {
		return jsonErr(service.ErrValidation("Id is required."))
	}
	ns, ok := store.GetNamespace(req.Id)
	if !ok {
		return jsonErr(service.NewAWSError("NamespaceNotFound", "Namespace not found.", http.StatusNotFound))
	}
	return jsonOK(&getNamespaceResponse{Namespace: nsToJSON(ns)})
}

// ---- ListNamespaces ----

type namespaceSummaryJSON struct {
	Id   string `json:"Id"`
	Arn  string `json:"Arn"`
	Name string `json:"Name"`
	Type string `json:"Type"`
}

type listNamespacesResponse struct {
	Namespaces []namespaceSummaryJSON `json:"Namespaces"`
}

func handleListNamespaces(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	namespaces := store.ListNamespaces()
	items := make([]namespaceSummaryJSON, 0, len(namespaces))
	for _, ns := range namespaces {
		items = append(items, namespaceSummaryJSON{Id: ns.ID, Arn: ns.ARN, Name: ns.Name, Type: string(ns.Type)})
	}
	return jsonOK(&listNamespacesResponse{Namespaces: items})
}

// ---- DeleteNamespace ----

type deleteNamespaceRequest struct {
	Id string `json:"Id"`
}

type deleteNamespaceResponse struct {
	OperationId string `json:"OperationId"`
}

func handleDeleteNamespace(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteNamespaceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Id == "" {
		return jsonErr(service.ErrValidation("Id is required."))
	}
	opID, ok := store.DeleteNamespace(req.Id)
	if !ok {
		return jsonErr(service.NewAWSError("NamespaceNotFound", "Namespace not found.", http.StatusNotFound))
	}
	return jsonOK(&deleteNamespaceResponse{OperationId: opID})
}

// ---- CreateService ----

type dnsRecordJSON struct {
	Type string `json:"Type"`
	TTL  int64  `json:"TTL"`
}

type dnsConfigJSON struct {
	NamespaceId   string          `json:"NamespaceId"`
	RoutingPolicy string          `json:"RoutingPolicy"`
	DnsRecords    []dnsRecordJSON `json:"DnsRecords"`
}

type healthCheckConfigJSON struct {
	Type             string `json:"Type"`
	ResourcePath     string `json:"ResourcePath"`
	FailureThreshold int    `json:"FailureThreshold"`
}

type createServiceRequest struct {
	Name            string                 `json:"Name"`
	NamespaceId     string                 `json:"NamespaceId"`
	Description     string                 `json:"Description"`
	DnsConfig       *dnsConfigJSON         `json:"DnsConfig"`
	HealthCheckConfig *healthCheckConfigJSON `json:"HealthCheckConfig"`
	Tags            []tagJSON              `json:"Tags"`
}

type serviceJSON struct {
	Id            string  `json:"Id"`
	Arn           string  `json:"Arn"`
	Name          string  `json:"Name"`
	NamespaceId   string  `json:"NamespaceId"`
	Description   string  `json:"Description,omitempty"`
	InstanceCount int     `json:"InstanceCount"`
}

type createServiceResponse struct {
	Service serviceJSON `json:"Service"`
}

func handleCreateService(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createServiceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Name == "" || req.NamespaceId == "" {
		return jsonErr(service.ErrValidation("Name and NamespaceId are required."))
	}
	var dnsConf *DnsConfig
	if req.DnsConfig != nil {
		records := make([]DnsRecord, 0, len(req.DnsConfig.DnsRecords))
		for _, r := range req.DnsConfig.DnsRecords {
			records = append(records, DnsRecord{Type: r.Type, TTL: r.TTL})
		}
		dnsConf = &DnsConfig{NamespaceID: req.DnsConfig.NamespaceId, RoutingPolicy: req.DnsConfig.RoutingPolicy, DnsRecords: records}
	}
	var hc *HealthCheckConfig
	if req.HealthCheckConfig != nil {
		hc = &HealthCheckConfig{Type: req.HealthCheckConfig.Type, ResourcePath: req.HealthCheckConfig.ResourcePath, FailureThreshold: req.HealthCheckConfig.FailureThreshold}
	}
	svc, ok := store.CreateService(req.Name, req.NamespaceId, req.Description, dnsConf, hc, tagsToMap(req.Tags))
	if !ok {
		return jsonErr(service.NewAWSError("NamespaceNotFound", "Namespace not found.", http.StatusNotFound))
	}
	return jsonOK(&createServiceResponse{Service: serviceJSON{Id: svc.ID, Arn: svc.ARN, Name: svc.Name, NamespaceId: svc.NamespaceID, Description: svc.Description, InstanceCount: svc.InstanceCount}})
}

// ---- GetService ----

type getServiceRequest struct {
	Id string `json:"Id"`
}

type getServiceResponse struct {
	Service serviceJSON `json:"Service"`
}

func handleGetService(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getServiceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Id == "" {
		return jsonErr(service.ErrValidation("Id is required."))
	}
	svc, ok := store.GetService(req.Id)
	if !ok {
		return jsonErr(service.NewAWSError("ServiceNotFound", "Service not found.", http.StatusNotFound))
	}
	return jsonOK(&getServiceResponse{Service: serviceJSON{Id: svc.ID, Arn: svc.ARN, Name: svc.Name, NamespaceId: svc.NamespaceID, Description: svc.Description, InstanceCount: svc.InstanceCount}})
}

// ---- ListServices ----

type listServicesRequest struct {
	Filters []struct {
		Name      string   `json:"Name"`
		Values    []string `json:"Values"`
		Condition string   `json:"Condition"`
	} `json:"Filters"`
}

type listServicesResponse struct {
	Services []serviceJSON `json:"Services"`
}

func handleListServices(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listServicesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	var nsFilter string
	for _, f := range req.Filters {
		if f.Name == "NAMESPACE_ID" && len(f.Values) > 0 {
			nsFilter = f.Values[0]
		}
	}
	services := store.ListServices(nsFilter)
	items := make([]serviceJSON, 0, len(services))
	for _, svc := range services {
		items = append(items, serviceJSON{Id: svc.ID, Arn: svc.ARN, Name: svc.Name, NamespaceId: svc.NamespaceID, Description: svc.Description, InstanceCount: svc.InstanceCount})
	}
	return jsonOK(&listServicesResponse{Services: items})
}

// ---- UpdateService ----

type updateServiceRequest struct {
	Id      string `json:"Id"`
	Service struct {
		Description string `json:"Description"`
	} `json:"Service"`
}

type updateServiceResponse struct {
	OperationId string `json:"OperationId"`
}

func handleUpdateService(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req updateServiceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Id == "" {
		return jsonErr(service.ErrValidation("Id is required."))
	}
	_, ok := store.UpdateService(req.Id, req.Service.Description, nil, nil)
	if !ok {
		return jsonErr(service.NewAWSError("ServiceNotFound", "Service not found.", http.StatusNotFound))
	}
	return jsonOK(&updateServiceResponse{OperationId: newOperationID()})
}

// ---- DeleteService ----

type deleteServiceRequest struct {
	Id string `json:"Id"`
}

func handleDeleteService(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteServiceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Id == "" {
		return jsonErr(service.ErrValidation("Id is required."))
	}
	if !store.DeleteService(req.Id) {
		return jsonErr(service.NewAWSError("ServiceNotFound", "Service not found.", http.StatusNotFound))
	}
	return jsonOK(struct{}{})
}

// ---- RegisterInstance ----

type registerInstanceRequest struct {
	ServiceId  string            `json:"ServiceId"`
	InstanceId string            `json:"InstanceId"`
	Attributes map[string]string `json:"Attributes"`
}

type registerInstanceResponse struct {
	OperationId string `json:"OperationId"`
}

func handleRegisterInstance(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req registerInstanceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ServiceId == "" || req.InstanceId == "" {
		return jsonErr(service.ErrValidation("ServiceId and InstanceId are required."))
	}
	opID, ok := store.RegisterInstance(req.ServiceId, req.InstanceId, req.Attributes)
	if !ok {
		return jsonErr(service.NewAWSError("ServiceNotFound", "Service not found.", http.StatusNotFound))
	}
	return jsonOK(&registerInstanceResponse{OperationId: opID})
}

// ---- DeregisterInstance ----

type deregisterInstanceRequest struct {
	ServiceId  string `json:"ServiceId"`
	InstanceId string `json:"InstanceId"`
}

type deregisterInstanceResponse struct {
	OperationId string `json:"OperationId"`
}

func handleDeregisterInstance(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deregisterInstanceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ServiceId == "" || req.InstanceId == "" {
		return jsonErr(service.ErrValidation("ServiceId and InstanceId are required."))
	}
	opID, ok := store.DeregisterInstance(req.ServiceId, req.InstanceId)
	if !ok {
		return jsonErr(service.NewAWSError("InstanceNotFound", "Instance not found.", http.StatusNotFound))
	}
	return jsonOK(&deregisterInstanceResponse{OperationId: opID})
}

// ---- GetInstance ----

type getInstanceRequest struct {
	ServiceId  string `json:"ServiceId"`
	InstanceId string `json:"InstanceId"`
}

type instanceJSON struct {
	Id         string            `json:"Id"`
	Attributes map[string]string `json:"Attributes"`
}

type getInstanceResponse struct {
	Instance instanceJSON `json:"Instance"`
}

func handleGetInstance(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getInstanceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ServiceId == "" || req.InstanceId == "" {
		return jsonErr(service.ErrValidation("ServiceId and InstanceId are required."))
	}
	inst, ok := store.GetInstance(req.ServiceId, req.InstanceId)
	if !ok {
		return jsonErr(service.NewAWSError("InstanceNotFound", "Instance not found.", http.StatusNotFound))
	}
	return jsonOK(&getInstanceResponse{Instance: instanceJSON{Id: inst.ID, Attributes: inst.Attributes}})
}

// ---- ListInstances ----

type listInstancesRequest struct {
	ServiceId string `json:"ServiceId"`
}

type listInstancesResponse struct {
	Instances []instanceJSON `json:"Instances"`
}

func handleListInstances(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listInstancesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ServiceId == "" {
		return jsonErr(service.ErrValidation("ServiceId is required."))
	}
	instances, ok := store.ListInstances(req.ServiceId)
	if !ok {
		return jsonErr(service.NewAWSError("ServiceNotFound", "Service not found.", http.StatusNotFound))
	}
	items := make([]instanceJSON, 0, len(instances))
	for _, inst := range instances {
		items = append(items, instanceJSON{Id: inst.ID, Attributes: inst.Attributes})
	}
	return jsonOK(&listInstancesResponse{Instances: items})
}

// ---- DiscoverInstances ----

type discoverInstancesRequest struct {
	NamespaceName    string            `json:"NamespaceName"`
	ServiceName      string            `json:"ServiceName"`
	QueryParameters  map[string]string `json:"QueryParameters"`
	HealthStatus     string            `json:"HealthStatus"`
}

type httpInstanceSummaryJSON struct {
	InstanceId   string            `json:"InstanceId"`
	NamespaceName string           `json:"NamespaceName"`
	ServiceName  string            `json:"ServiceName"`
	HealthStatus string            `json:"HealthStatus"`
	Attributes   map[string]string `json:"Attributes"`
}

type discoverInstancesResponse struct {
	Instances []httpInstanceSummaryJSON `json:"Instances"`
}

func handleDiscoverInstances(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req discoverInstancesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.NamespaceName == "" || req.ServiceName == "" {
		return jsonErr(service.ErrValidation("NamespaceName and ServiceName are required."))
	}
	healthFilter := req.HealthStatus
	instances := store.DiscoverInstances(req.NamespaceName, req.ServiceName, req.QueryParameters, healthFilter)
	items := make([]httpInstanceSummaryJSON, 0, len(instances))
	for _, inst := range instances {
		items = append(items, httpInstanceSummaryJSON{
			InstanceId:    inst.ID,
			NamespaceName: req.NamespaceName,
			ServiceName:   req.ServiceName,
			HealthStatus:  inst.HealthStatus,
			Attributes:    inst.Attributes,
		})
	}
	return jsonOK(&discoverInstancesResponse{Instances: items})
}

// ---- UpdateInstanceCustomHealthStatus ----

type updateInstanceCustomHealthStatusRequest struct {
	ServiceId  string `json:"ServiceId"`
	InstanceId string `json:"InstanceId"`
	Status     string `json:"Status"`
}

func handleUpdateInstanceCustomHealthStatus(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req updateInstanceCustomHealthStatusRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ServiceId == "" || req.InstanceId == "" || req.Status == "" {
		return jsonErr(service.ErrValidation("ServiceId, InstanceId, and Status are required."))
	}
	if req.Status != "HEALTHY" && req.Status != "UNHEALTHY" {
		return jsonErr(service.ErrValidation("Status must be HEALTHY or UNHEALTHY."))
	}
	if !store.UpdateInstanceCustomHealthStatus(req.ServiceId, req.InstanceId, req.Status) {
		return jsonErr(service.NewAWSError("InstanceNotFound", "Instance not found.", http.StatusNotFound))
	}
	return emptyOK()
}

// ---- TagResource ----

type tagResourceRequest struct {
	ResourceARN string    `json:"ResourceARN"`
	Tags        []tagJSON `json:"Tags"`
}

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req tagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ResourceARN == "" {
		return jsonErr(service.ErrValidation("ResourceARN is required."))
	}
	if !store.TagResource(req.ResourceARN, tagsToMap(req.Tags)) {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Resource not found.", http.StatusNotFound))
	}
	return jsonOK(struct{}{})
}

// ---- UntagResource ----

type untagResourceRequest struct {
	ResourceARN string   `json:"ResourceARN"`
	TagKeys     []string `json:"TagKeys"`
}

func handleUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req untagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ResourceARN == "" {
		return jsonErr(service.ErrValidation("ResourceARN is required."))
	}
	if !store.UntagResource(req.ResourceARN, req.TagKeys) {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Resource not found.", http.StatusNotFound))
	}
	return jsonOK(struct{}{})
}

// ---- ListTagsForResource ----

type listTagsRequest struct {
	ResourceARN string `json:"ResourceARN"`
}

type listTagsResponse struct {
	Tags []tagJSON `json:"Tags"`
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listTagsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ResourceARN == "" {
		return jsonErr(service.ErrValidation("ResourceARN is required."))
	}
	tags, ok := store.ListTagsForResource(req.ResourceARN)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Resource not found.", http.StatusNotFound))
	}
	return jsonOK(&listTagsResponse{Tags: mapToTags(tags)})
}
