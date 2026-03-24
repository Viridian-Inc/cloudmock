package ecs

import (
	"encoding/json"
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// ---- JSON request/response types ----

type tagEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// ---- helpers ----

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       body,
		Format:     service.FormatJSON,
	}, nil
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

func tagsFromList(entries []tagEntry) map[string]string {
	m := make(map[string]string, len(entries))
	for _, t := range entries {
		m[t.Key] = t.Value
	}
	return m
}

func tagsToList(m map[string]string) []tagEntry {
	out := make([]tagEntry, 0, len(m))
	for k, v := range m {
		out = append(out, tagEntry{Key: k, Value: v})
	}
	return out
}

// ---- Cluster JSON types ----

type createClusterRequest struct {
	ClusterName string     `json:"clusterName"`
	Tags        []tagEntry `json:"tags"`
}

type clusterJSON struct {
	ClusterArn                       string `json:"clusterArn"`
	ClusterName                      string `json:"clusterName"`
	Status                           string `json:"status"`
	RegisteredContainerInstancesCount int   `json:"registeredContainerInstancesCount"`
	RunningTasksCount                int    `json:"runningTasksCount"`
	PendingTasksCount                int    `json:"pendingTasksCount"`
}

type createClusterResponse struct {
	Cluster clusterJSON `json:"cluster"`
}

type deleteClusterRequest struct {
	Cluster string `json:"cluster"`
}

type deleteClusterResponse struct {
	Cluster clusterJSON `json:"cluster"`
}

type describeClustersRequest struct {
	Clusters []string `json:"clusters"`
}

type clusterFailureJSON struct {
	ARN    string `json:"arn"`
	Reason string `json:"reason"`
}

type describeClustersResponse struct {
	Clusters  []clusterJSON        `json:"clusters"`
	Failures  []clusterFailureJSON `json:"failures"`
}

type listClustersResponse struct {
	ClusterArns []string `json:"clusterArns"`
	NextToken   string   `json:"nextToken,omitempty"`
}

func clusterToJSON(c *Cluster) clusterJSON {
	return clusterJSON{
		ClusterArn:                       c.ARN,
		ClusterName:                      c.Name,
		Status:                           c.Status,
		RegisteredContainerInstancesCount: c.RegisteredContainerInstancesCount,
		RunningTasksCount:                c.RunningTasksCount,
		PendingTasksCount:                c.PendingTasksCount,
	}
}

// ---- Cluster handlers ----

func handleCreateCluster(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createClusterRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	c, awsErr := store.CreateCluster(req.ClusterName, tagsFromList(req.Tags))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(createClusterResponse{Cluster: clusterToJSON(c)})
}

func handleDeleteCluster(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteClusterRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Cluster == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"cluster is required.", http.StatusBadRequest))
	}

	c, awsErr := store.DeleteCluster(req.Cluster)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(deleteClusterResponse{Cluster: clusterToJSON(c)})
}

func handleDescribeClusters(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeClustersRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	clusters, failures := store.DescribeClusters(req.Clusters)

	out := make([]clusterJSON, len(clusters))
	for i, c := range clusters {
		out[i] = clusterToJSON(c)
	}

	failsJSON := make([]clusterFailureJSON, len(failures))
	for i, f := range failures {
		failsJSON[i] = clusterFailureJSON{ARN: f.ARN, Reason: f.Reason}
	}

	return jsonOK(describeClustersResponse{Clusters: out, Failures: failsJSON})
}

func handleListClusters(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	arns := store.ListClusters()
	if arns == nil {
		arns = []string{}
	}
	return jsonOK(listClustersResponse{ClusterArns: arns})
}

// ---- Task Definition JSON types ----

type portMappingJSON struct {
	ContainerPort int    `json:"containerPort"`
	HostPort      int    `json:"hostPort"`
	Protocol      string `json:"protocol"`
}

type keyValuePairJSON struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type containerDefJSON struct {
	Name         string             `json:"name"`
	Image        string             `json:"image"`
	CPU          int                `json:"cpu"`
	Memory       int                `json:"memory"`
	PortMappings []portMappingJSON  `json:"portMappings,omitempty"`
	Environment  []keyValuePairJSON `json:"environment,omitempty"`
	Essential    bool               `json:"essential"`
}

type registerTaskDefinitionRequest struct {
	Family                  string             `json:"family"`
	ContainerDefinitions    []containerDefJSON `json:"containerDefinitions"`
	NetworkMode             string             `json:"networkMode"`
	RequiresCompatibilities []string           `json:"requiresCompatibilities"`
	CPU                     string             `json:"cpu"`
	Memory                  string             `json:"memory"`
}

type taskDefinitionJSON struct {
	TaskDefinitionArn       string             `json:"taskDefinitionArn"`
	Family                  string             `json:"family"`
	Revision                int                `json:"revision"`
	Status                  string             `json:"status"`
	ContainerDefinitions    []containerDefJSON `json:"containerDefinitions"`
	NetworkMode             string             `json:"networkMode,omitempty"`
	RequiresCompatibilities []string           `json:"requiresCompatibilities,omitempty"`
	CPU                     string             `json:"cpu,omitempty"`
	Memory                  string             `json:"memory,omitempty"`
}

type registerTaskDefinitionResponse struct {
	TaskDefinition taskDefinitionJSON `json:"taskDefinition"`
}

type deregisterTaskDefinitionRequest struct {
	TaskDefinition string `json:"taskDefinition"`
}

type deregisterTaskDefinitionResponse struct {
	TaskDefinition taskDefinitionJSON `json:"taskDefinition"`
}

type describeTaskDefinitionRequest struct {
	TaskDefinition string `json:"taskDefinition"`
}

type describeTaskDefinitionResponse struct {
	TaskDefinition taskDefinitionJSON `json:"taskDefinition"`
}

type listTaskDefinitionsRequest struct {
	FamilyPrefix string `json:"familyPrefix"`
}

type listTaskDefinitionsResponse struct {
	TaskDefinitionArns []string `json:"taskDefinitionArns"`
	NextToken          string   `json:"nextToken,omitempty"`
}

func containerDefsToJSON(defs []ContainerDefinition) []containerDefJSON {
	out := make([]containerDefJSON, len(defs))
	for i, d := range defs {
		pm := make([]portMappingJSON, len(d.PortMappings))
		for j, p := range d.PortMappings {
			pm[j] = portMappingJSON{
				ContainerPort: p.ContainerPort,
				HostPort:      p.HostPort,
				Protocol:      p.Protocol,
			}
		}
		env := make([]keyValuePairJSON, len(d.Environment))
		for j, e := range d.Environment {
			env[j] = keyValuePairJSON{Name: e.Name, Value: e.Value}
		}
		out[i] = containerDefJSON{
			Name:         d.Name,
			Image:        d.Image,
			CPU:          d.CPU,
			Memory:       d.Memory,
			PortMappings: pm,
			Environment:  env,
			Essential:    d.Essential,
		}
	}
	return out
}

func taskDefToJSON(td *TaskDefinition) taskDefinitionJSON {
	return taskDefinitionJSON{
		TaskDefinitionArn:       td.ARN,
		Family:                  td.Family,
		Revision:                td.Revision,
		Status:                  td.Status,
		ContainerDefinitions:    containerDefsToJSON(td.ContainerDefinitions),
		NetworkMode:             td.NetworkMode,
		RequiresCompatibilities: td.RequiresCompatibilities,
		CPU:                     td.CPU,
		Memory:                  td.Memory,
	}
}

func containerDefsFromJSON(defs []containerDefJSON) []ContainerDefinition {
	out := make([]ContainerDefinition, len(defs))
	for i, d := range defs {
		pm := make([]PortMapping, len(d.PortMappings))
		for j, p := range d.PortMappings {
			pm[j] = PortMapping{
				ContainerPort: p.ContainerPort,
				HostPort:      p.HostPort,
				Protocol:      p.Protocol,
			}
		}
		env := make([]KeyValuePair, len(d.Environment))
		for j, e := range d.Environment {
			env[j] = KeyValuePair{Name: e.Name, Value: e.Value}
		}
		out[i] = ContainerDefinition{
			Name:         d.Name,
			Image:        d.Image,
			CPU:          d.CPU,
			Memory:       d.Memory,
			PortMappings: pm,
			Environment:  env,
			Essential:    d.Essential,
		}
	}
	return out
}

// ---- Task Definition handlers ----

func handleRegisterTaskDefinition(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req registerTaskDefinitionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Family == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"family is required.", http.StatusBadRequest))
	}

	containerDefs := containerDefsFromJSON(req.ContainerDefinitions)
	td, awsErr := store.RegisterTaskDefinition(
		req.Family,
		containerDefs,
		req.NetworkMode,
		req.RequiresCompatibilities,
		req.CPU,
		req.Memory,
	)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(registerTaskDefinitionResponse{TaskDefinition: taskDefToJSON(td)})
}

func handleDeregisterTaskDefinition(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deregisterTaskDefinitionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.TaskDefinition == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"taskDefinition is required.", http.StatusBadRequest))
	}

	td, awsErr := store.DeregisterTaskDefinition(req.TaskDefinition)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(deregisterTaskDefinitionResponse{TaskDefinition: taskDefToJSON(td)})
}

func handleDescribeTaskDefinition(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeTaskDefinitionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.TaskDefinition == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"taskDefinition is required.", http.StatusBadRequest))
	}

	td, awsErr := store.DescribeTaskDefinition(req.TaskDefinition)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(describeTaskDefinitionResponse{TaskDefinition: taskDefToJSON(td)})
}

func handleListTaskDefinitions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listTaskDefinitionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	arns := store.ListTaskDefinitions(req.FamilyPrefix)
	if arns == nil {
		arns = []string{}
	}
	return jsonOK(listTaskDefinitionsResponse{TaskDefinitionArns: arns})
}

// ---- Service JSON types ----

type createServiceRequest struct {
	Cluster        string `json:"cluster"`
	ServiceName    string `json:"serviceName"`
	TaskDefinition string `json:"taskDefinition"`
	DesiredCount   int    `json:"desiredCount"`
	LaunchType     string `json:"launchType"`
}

type serviceJSON struct {
	ServiceArn     string `json:"serviceArn"`
	ServiceName    string `json:"serviceName"`
	ClusterArn     string `json:"clusterArn"`
	TaskDefinition string `json:"taskDefinition"`
	DesiredCount   int    `json:"desiredCount"`
	RunningCount   int    `json:"runningCount"`
	Status         string `json:"status"`
	LaunchType     string `json:"launchType"`
}

type createServiceResponse struct {
	Service serviceJSON `json:"service"`
}

type deleteServiceRequest struct {
	Cluster string `json:"cluster"`
	Service string `json:"service"`
	Force   bool   `json:"force"`
}

type deleteServiceResponse struct {
	Service serviceJSON `json:"service"`
}

type describeServicesRequest struct {
	Cluster  string   `json:"cluster"`
	Services []string `json:"services"`
}

type serviceFailureJSON struct {
	ARN    string `json:"arn"`
	Reason string `json:"reason"`
}

type describeServicesResponse struct {
	Services []serviceJSON        `json:"services"`
	Failures []serviceFailureJSON `json:"failures"`
}

type listServicesRequest struct {
	Cluster string `json:"cluster"`
}

type listServicesResponse struct {
	ServiceArns []string `json:"serviceArns"`
	NextToken   string   `json:"nextToken,omitempty"`
}

type updateServiceRequest struct {
	Cluster        string `json:"cluster"`
	Service        string `json:"service"`
	DesiredCount   *int   `json:"desiredCount"`
	TaskDefinition string `json:"taskDefinition"`
}

type updateServiceResponse struct {
	Service serviceJSON `json:"service"`
}

func serviceToJSON(svc *Service) serviceJSON {
	return serviceJSON{
		ServiceArn:     svc.ARN,
		ServiceName:    svc.Name,
		ClusterArn:     svc.ClusterARN,
		TaskDefinition: svc.TaskDefinition,
		DesiredCount:   svc.DesiredCount,
		RunningCount:   svc.RunningCount,
		Status:         svc.Status,
		LaunchType:     svc.LaunchType,
	}
}

// ---- Service handlers ----

func handleCreateService(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createServiceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ServiceName == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"serviceName is required.", http.StatusBadRequest))
	}

	svc, awsErr := store.CreateService(req.Cluster, req.ServiceName, req.TaskDefinition, req.DesiredCount, req.LaunchType)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(createServiceResponse{Service: serviceToJSON(svc)})
}

func handleDeleteService(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteServiceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Service == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"service is required.", http.StatusBadRequest))
	}

	svc, awsErr := store.DeleteService(req.Cluster, req.Service, req.Force)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(deleteServiceResponse{Service: serviceToJSON(svc)})
}

func handleDescribeServices(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeServicesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	services, failures := store.DescribeServices(req.Cluster, req.Services)

	out := make([]serviceJSON, len(services))
	for i, svc := range services {
		out[i] = serviceToJSON(svc)
	}

	failsJSON := make([]serviceFailureJSON, len(failures))
	for i, f := range failures {
		failsJSON[i] = serviceFailureJSON{ARN: f.ARN, Reason: f.Reason}
	}

	return jsonOK(describeServicesResponse{Services: out, Failures: failsJSON})
}

func handleListServices(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listServicesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	arns := store.ListServices(req.Cluster)
	if arns == nil {
		arns = []string{}
	}
	return jsonOK(listServicesResponse{ServiceArns: arns})
}

func handleUpdateService(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req updateServiceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Service == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"service is required.", http.StatusBadRequest))
	}

	svc, awsErr := store.UpdateService(req.Cluster, req.Service, req.DesiredCount, req.TaskDefinition)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(updateServiceResponse{Service: serviceToJSON(svc)})
}

// ---- Task JSON types ----

type runTaskRequest struct {
	Cluster        string `json:"cluster"`
	TaskDefinition string `json:"taskDefinition"`
	Count          int    `json:"count"`
}

type taskJSON struct {
	TaskArn           string   `json:"taskArn"`
	TaskDefinitionArn string   `json:"taskDefinitionArn"`
	ClusterArn        string   `json:"clusterArn"`
	LastStatus        string   `json:"lastStatus"`
	DesiredStatus     string   `json:"desiredStatus"`
	StartedAt         *float64 `json:"startedAt,omitempty"`
	StoppedAt         *float64 `json:"stoppedAt,omitempty"`
	StopCode          string   `json:"stopCode,omitempty"`
	StoppedReason     string   `json:"stoppedReason,omitempty"`
}

type runTaskResponse struct {
	Tasks    []taskJSON        `json:"tasks"`
	Failures []taskFailureJSON `json:"failures"`
}

type stopTaskRequest struct {
	Cluster string `json:"cluster"`
	Task    string `json:"task"`
	Reason  string `json:"reason"`
}

type stopTaskResponse struct {
	Task taskJSON `json:"task"`
}

type describeTasksRequest struct {
	Cluster string   `json:"cluster"`
	Tasks   []string `json:"tasks"`
}

type taskFailureJSON struct {
	ARN    string `json:"arn"`
	Reason string `json:"reason"`
}

type describeTasksResponse struct {
	Tasks    []taskJSON        `json:"tasks"`
	Failures []taskFailureJSON `json:"failures"`
}

type listTasksRequest struct {
	Cluster     string `json:"cluster"`
	ServiceName string `json:"serviceName"`
}

type listTasksResponse struct {
	TaskArns  []string `json:"taskArns"`
	NextToken string   `json:"nextToken,omitempty"`
}

func taskToJSON(t *Task) taskJSON {
	tj := taskJSON{
		TaskArn:           t.ARN,
		TaskDefinitionArn: t.TaskDefinitionARN,
		ClusterArn:        t.ClusterARN,
		LastStatus:        t.LastStatus,
		DesiredStatus:     t.DesiredStatus,
		StopCode:          t.StopCode,
		StoppedReason:     t.StoppedReason,
	}
	if t.StartedAt != nil {
		v := float64(t.StartedAt.Unix())
		tj.StartedAt = &v
	}
	if t.StoppedAt != nil {
		v := float64(t.StoppedAt.Unix())
		tj.StoppedAt = &v
	}
	return tj
}

// ---- Task handlers ----

func handleRunTask(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req runTaskRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.TaskDefinition == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"taskDefinition is required.", http.StatusBadRequest))
	}
	count := req.Count
	if count == 0 {
		count = 1
	}

	tasks, awsErr := store.RunTask(req.Cluster, req.TaskDefinition, count)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	out := make([]taskJSON, len(tasks))
	for i, t := range tasks {
		out[i] = taskToJSON(t)
	}
	return jsonOK(runTaskResponse{Tasks: out, Failures: []taskFailureJSON{}})
}

func handleStopTask(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req stopTaskRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Task == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"task is required.", http.StatusBadRequest))
	}

	t, awsErr := store.StopTask(req.Cluster, req.Task, req.Reason)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(stopTaskResponse{Task: taskToJSON(t)})
}

func handleDescribeTasks(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeTasksRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	tasks, failures := store.DescribeTasks(req.Cluster, req.Tasks)

	out := make([]taskJSON, len(tasks))
	for i, t := range tasks {
		out[i] = taskToJSON(t)
	}

	failsJSON := make([]taskFailureJSON, len(failures))
	for i, f := range failures {
		failsJSON[i] = taskFailureJSON{ARN: f.ARN, Reason: f.Reason}
	}

	return jsonOK(describeTasksResponse{Tasks: out, Failures: failsJSON})
}

func handleListTasks(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listTasksRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	arns := store.ListTasks(req.Cluster, req.ServiceName)
	if arns == nil {
		arns = []string{}
	}
	return jsonOK(listTasksResponse{TaskArns: arns})
}

// ---- Tag JSON types and handlers ----

type tagResourceRequest struct {
	ResourceArn string     `json:"resourceArn"`
	Tags        []tagEntry `json:"tags"`
}

type untagResourceRequest struct {
	ResourceArn string   `json:"resourceArn"`
	TagKeys     []string `json:"tagKeys"`
}

type listTagsForResourceRequest struct {
	ResourceArn string `json:"resourceArn"`
}

type listTagsForResourceResponse struct {
	Tags []tagEntry `json:"tags"`
}

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req tagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ResourceArn == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"resourceArn is required.", http.StatusBadRequest))
	}

	if awsErr := store.TagResource(req.ResourceArn, tagsFromList(req.Tags)); awsErr != nil {
		return jsonErr(awsErr)
	}
	return &service.Response{StatusCode: http.StatusOK, Body: struct{}{}, Format: service.FormatJSON}, nil
}

func handleUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req untagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ResourceArn == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"resourceArn is required.", http.StatusBadRequest))
	}

	if awsErr := store.UntagResource(req.ResourceArn, req.TagKeys); awsErr != nil {
		return jsonErr(awsErr)
	}
	return &service.Response{StatusCode: http.StatusOK, Body: struct{}{}, Format: service.FormatJSON}, nil
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listTagsForResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ResourceArn == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"resourceArn is required.", http.StatusBadRequest))
	}

	tags, awsErr := store.ListTagsForResource(req.ResourceArn)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(listTagsForResourceResponse{Tags: tagsToList(tags)})
}
