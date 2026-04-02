package dax

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
)

// ---- helpers ----

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
		return service.NewAWSError("InvalidParameterValueException", "Invalid JSON in request body.", http.StatusBadRequest)
	}
	return nil
}

func strSlice(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func clusterToJSON(c *Cluster) map[string]any {
	nodes := make([]map[string]any, 0, len(c.Nodes))
	for _, n := range c.Nodes {
		nodeMap := map[string]any{
			"NodeId":               n.NodeId,
			"NodeCreateTime":       n.NodeCreateTime.Format(time.RFC3339),
			"AvailabilityZone":     n.AvailabilityZone,
			"NodeStatus":           n.NodeStatus,
			"ParameterGroupStatus": n.ParameterGroupStatus,
		}
		if n.Endpoint != nil {
			nodeMap["Endpoint"] = map[string]any{
				"Address": n.Endpoint.Address,
				"Port":    n.Endpoint.Port,
			}
		}
		nodes = append(nodes, nodeMap)
	}
	result := map[string]any{
		"ClusterName":       c.ClusterName,
		"ClusterArn":        c.ClusterArn,
		"Description":       c.Description,
		"NodeType":          c.NodeType,
		"ReplicationFactor": c.ReplicationFactor,
		"Status":            c.Status,
		"Nodes":             nodes,
		"ActiveNodes":       len(nodes),
		"TotalNodes":        len(nodes),
		"IamRoleArn":        c.IamRoleArn,
	}
	if c.Endpoint != nil {
		result["ClusterDiscoveryEndpoint"] = map[string]any{
			"Address": c.Endpoint.Address,
			"Port":    c.Endpoint.Port,
			"URL":     c.Endpoint.URL,
		}
	}
	if c.SSEDescription != nil {
		result["SSEDescription"] = map[string]any{"Status": c.SSEDescription.Status}
	}
	if c.SubnetGroupName != "" {
		result["SubnetGroup"] = c.SubnetGroupName
	}
	if c.ParameterGroupName != "" {
		result["ParameterGroup"] = map[string]any{"ParameterGroupName": c.ParameterGroupName}
	}
	return result
}

// ---- CreateCluster ----

type createClusterRequest struct {
	ClusterName       string            `json:"ClusterName"`
	Description       string            `json:"Description"`
	NodeType          string            `json:"NodeType"`
	ReplicationFactor int               `json:"ReplicationFactor"`
	SubnetGroupName   string            `json:"SubnetGroupName"`
	ParameterGroupName string           `json:"ParameterGroupName"`
	IamRoleArn        string            `json:"IamRoleArn"`
	AvailabilityZones []string          `json:"AvailabilityZones"`
	SecurityGroupIds  []string          `json:"SecurityGroupIds"`
	SSESpecification  *sseSpec          `json:"SSESpecification"`
	Tags              []tagPair         `json:"Tags"`
}

type sseSpec struct {
	Enabled bool `json:"Enabled"`
}

type tagPair struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

func handleCreateCluster(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createClusterRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ClusterName == "" {
		return jsonErr(service.ErrValidation("ClusterName is required."))
	}
	if req.IamRoleArn == "" {
		return jsonErr(service.ErrValidation("IamRoleArn is required."))
	}

	tags := make(map[string]string)
	for _, t := range req.Tags {
		tags[t.Key] = t.Value
	}

	sseEnabled := req.SSESpecification != nil && req.SSESpecification.Enabled

	cluster, err := store.CreateCluster(
		req.ClusterName, req.Description, req.NodeType,
		req.ReplicationFactor, req.SubnetGroupName, req.ParameterGroupName,
		req.IamRoleArn, req.AvailabilityZones, req.SecurityGroupIds,
		sseEnabled, tags,
	)
	if err != nil {
		return jsonErr(service.NewAWSError("ClusterAlreadyExistsFault", err.Error(), http.StatusConflict))
	}
	return jsonOK(map[string]any{"Cluster": clusterToJSON(cluster)})
}

// ---- DescribeClusters ----

type describeClustersRequest struct {
	ClusterNames []string `json:"ClusterNames"`
}

func handleDescribeClusters(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeClustersRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	clusters := store.DescribeClusters(req.ClusterNames)
	out := make([]map[string]any, 0, len(clusters))
	for _, c := range clusters {
		out = append(out, clusterToJSON(c))
	}
	return jsonOK(map[string]any{"Clusters": out})
}

// ---- UpdateCluster ----

type updateClusterRequest struct {
	ClusterName                string `json:"ClusterName"`
	Description                string `json:"Description"`
	PreferredMaintenanceWindow string `json:"PreferredMaintenanceWindow"`
	NotificationTopicArn       string `json:"NotificationTopicArn"`
}

func handleUpdateCluster(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req updateClusterRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ClusterName == "" {
		return jsonErr(service.ErrValidation("ClusterName is required."))
	}
	cluster, ok := store.UpdateCluster(req.ClusterName, req.Description, req.PreferredMaintenanceWindow, req.NotificationTopicArn)
	if !ok {
		return jsonErr(service.NewAWSError("ClusterNotFoundFault", "Cluster not found: "+req.ClusterName, http.StatusNotFound))
	}
	return jsonOK(map[string]any{"Cluster": clusterToJSON(cluster)})
}

// ---- DeleteCluster ----

type deleteClusterRequest struct {
	ClusterName string `json:"ClusterName"`
}

func handleDeleteCluster(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteClusterRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ClusterName == "" {
		return jsonErr(service.ErrValidation("ClusterName is required."))
	}
	cluster, ok := store.DeleteCluster(req.ClusterName)
	if !ok {
		return jsonErr(service.NewAWSError("ClusterNotFoundFault", "Cluster not found: "+req.ClusterName, http.StatusNotFound))
	}
	return jsonOK(map[string]any{"Cluster": clusterToJSON(cluster)})
}

// ---- IncreaseReplicationFactor ----

type increaseReplicationFactorRequest struct {
	ClusterName              string   `json:"ClusterName"`
	NewReplicationFactor     int      `json:"NewReplicationFactor"`
	AvailabilityZones        []string `json:"AvailabilityZones"`
}

func handleIncreaseReplicationFactor(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req increaseReplicationFactorRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ClusterName == "" || req.NewReplicationFactor == 0 {
		return jsonErr(service.ErrValidation("ClusterName and NewReplicationFactor are required."))
	}
	cluster, err := store.IncreaseReplicationFactor(req.ClusterName, req.NewReplicationFactor, req.AvailabilityZones)
	if err != nil {
		return jsonErr(service.NewAWSError("InvalidParameterValueException", err.Error(), http.StatusBadRequest))
	}
	return jsonOK(map[string]any{"Cluster": clusterToJSON(cluster)})
}

// ---- DecreaseReplicationFactor ----

type decreaseReplicationFactorRequest struct {
	ClusterName          string   `json:"ClusterName"`
	NewReplicationFactor int      `json:"NewReplicationFactor"`
	NodeIdsToRemove      []string `json:"NodeIdsToRemove"`
}

func handleDecreaseReplicationFactor(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req decreaseReplicationFactorRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ClusterName == "" || req.NewReplicationFactor == 0 {
		return jsonErr(service.ErrValidation("ClusterName and NewReplicationFactor are required."))
	}
	cluster, err := store.DecreaseReplicationFactor(req.ClusterName, req.NewReplicationFactor, req.NodeIdsToRemove)
	if err != nil {
		return jsonErr(service.NewAWSError("InvalidParameterValueException", err.Error(), http.StatusBadRequest))
	}
	return jsonOK(map[string]any{"Cluster": clusterToJSON(cluster)})
}

// ---- CreateSubnetGroup ----

type createSubnetGroupRequest struct {
	SubnetGroupName string   `json:"SubnetGroupName"`
	Description     string   `json:"Description"`
	SubnetIds       []string `json:"SubnetIds"`
}

func handleCreateSubnetGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createSubnetGroupRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.SubnetGroupName == "" {
		return jsonErr(service.ErrValidation("SubnetGroupName is required."))
	}
	sg, err := store.CreateSubnetGroup(req.SubnetGroupName, req.Description, "", req.SubnetIds)
	if err != nil {
		return jsonErr(service.NewAWSError("SubnetGroupAlreadyExistsFault", err.Error(), http.StatusConflict))
	}
	return jsonOK(map[string]any{
		"SubnetGroup": map[string]any{
			"SubnetGroupName": sg.SubnetGroupName,
			"Description":     sg.Description,
			"VpcId":           sg.VpcId,
			"Subnets":         sg.Subnets,
		},
	})
}

// ---- DescribeSubnetGroups ----

type describeSubnetGroupsRequest struct {
	SubnetGroupNames []string `json:"SubnetGroupNames"`
}

func handleDescribeSubnetGroups(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeSubnetGroupsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	groups := store.DescribeSubnetGroups(req.SubnetGroupNames)
	out := make([]map[string]any, 0, len(groups))
	for _, sg := range groups {
		out = append(out, map[string]any{
			"SubnetGroupName": sg.SubnetGroupName,
			"Description":     sg.Description,
			"VpcId":           sg.VpcId,
			"Subnets":         sg.Subnets,
		})
	}
	return jsonOK(map[string]any{"SubnetGroups": out})
}

// ---- DeleteSubnetGroup ----

type deleteSubnetGroupRequest struct {
	SubnetGroupName string `json:"SubnetGroupName"`
}

func handleDeleteSubnetGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteSubnetGroupRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.SubnetGroupName == "" {
		return jsonErr(service.ErrValidation("SubnetGroupName is required."))
	}
	if !store.DeleteSubnetGroup(req.SubnetGroupName) {
		return jsonErr(service.NewAWSError("SubnetGroupNotFoundFault", "Subnet group not found: "+req.SubnetGroupName, http.StatusNotFound))
	}
	return jsonOK(map[string]any{"DeletionMessage": "SubnetGroup " + req.SubnetGroupName + " deleted."})
}

// ---- CreateParameterGroup ----

type createParameterGroupRequest struct {
	ParameterGroupName string `json:"ParameterGroupName"`
	Description        string `json:"Description"`
}

func handleCreateParameterGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createParameterGroupRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ParameterGroupName == "" {
		return jsonErr(service.ErrValidation("ParameterGroupName is required."))
	}
	pg, err := store.CreateParameterGroup(req.ParameterGroupName, req.Description)
	if err != nil {
		return jsonErr(service.NewAWSError("ParameterGroupAlreadyExistsFault", err.Error(), http.StatusConflict))
	}
	return jsonOK(map[string]any{
		"ParameterGroup": map[string]any{
			"ParameterGroupName": pg.ParameterGroupName,
			"Description":        pg.Description,
		},
	})
}

// ---- DescribeParameterGroups ----

type describeParameterGroupsRequest struct {
	ParameterGroupNames []string `json:"ParameterGroupNames"`
}

func handleDescribeParameterGroups(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeParameterGroupsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	groups := store.DescribeParameterGroups(req.ParameterGroupNames)
	out := make([]map[string]any, 0, len(groups))
	for _, pg := range groups {
		out = append(out, map[string]any{
			"ParameterGroupName": pg.ParameterGroupName,
			"Description":        pg.Description,
		})
	}
	return jsonOK(map[string]any{"ParameterGroups": out})
}

// ---- UpdateParameterGroup ----

type parameterNameValue struct {
	ParameterName  string `json:"ParameterName"`
	ParameterValue string `json:"ParameterValue"`
}

type updateParameterGroupRequest struct {
	ParameterGroupName   string               `json:"ParameterGroupName"`
	ParameterNameValues  []parameterNameValue `json:"ParameterNameValues"`
}

func handleUpdateParameterGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req updateParameterGroupRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ParameterGroupName == "" {
		return jsonErr(service.ErrValidation("ParameterGroupName is required."))
	}
	params := make(map[string]string)
	for _, pnv := range req.ParameterNameValues {
		params[pnv.ParameterName] = pnv.ParameterValue
	}
	pg, ok := store.UpdateParameterGroup(req.ParameterGroupName, params)
	if !ok {
		return jsonErr(service.NewAWSError("ParameterGroupNotFoundFault", "Parameter group not found: "+req.ParameterGroupName, http.StatusNotFound))
	}
	return jsonOK(map[string]any{
		"ParameterGroup": map[string]any{
			"ParameterGroupName": pg.ParameterGroupName,
			"Description":        pg.Description,
		},
	})
}

// ---- DeleteParameterGroup ----

type deleteParameterGroupRequest struct {
	ParameterGroupName string `json:"ParameterGroupName"`
}

func handleDeleteParameterGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteParameterGroupRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ParameterGroupName == "" {
		return jsonErr(service.ErrValidation("ParameterGroupName is required."))
	}
	if !store.DeleteParameterGroup(req.ParameterGroupName) {
		return jsonErr(service.NewAWSError("ParameterGroupNotFoundFault", "Parameter group not found: "+req.ParameterGroupName, http.StatusNotFound))
	}
	return jsonOK(map[string]any{"DeletionMessage": "ParameterGroup " + req.ParameterGroupName + " deleted."})
}

// ---- DescribeParameters ----

type describeParametersRequest struct {
	ParameterGroupName string `json:"ParameterGroupName"`
}

func handleDescribeParameters(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeParametersRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ParameterGroupName == "" {
		return jsonErr(service.ErrValidation("ParameterGroupName is required."))
	}
	params, ok := store.DescribeParameters(req.ParameterGroupName)
	if !ok {
		return jsonErr(service.NewAWSError("ParameterGroupNotFoundFault", "Parameter group not found: "+req.ParameterGroupName, http.StatusNotFound))
	}
	out := make([]map[string]any, 0, len(params))
	for _, p := range params {
		out = append(out, map[string]any{
			"ParameterName":  p.ParameterName,
			"ParameterValue": p.ParameterValue,
			"DataType":       p.DataType,
			"IsModifiable":   p.IsModifiable,
		})
	}
	return jsonOK(map[string]any{"Parameters": out})
}

// ---- DescribeDefaultParameters ----

func handleDescribeDefaultParameters(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	params := store.DescribeDefaultParameters()
	out := make([]map[string]any, 0, len(params))
	for _, p := range params {
		out = append(out, map[string]any{
			"ParameterName":  p.ParameterName,
			"ParameterValue": p.ParameterValue,
			"DataType":       p.DataType,
			"IsModifiable":   p.IsModifiable,
			"Description":    p.Description,
		})
	}
	return jsonOK(map[string]any{"Parameters": out})
}

// ---- TagResource ----

type tagResourceRequest struct {
	ResourceName string    `json:"ResourceName"`
	Tags         []tagPair `json:"Tags"`
}

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req tagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ResourceName == "" {
		return jsonErr(service.ErrValidation("ResourceName is required."))
	}
	tags := make(map[string]string)
	for _, t := range req.Tags {
		tags[t.Key] = t.Value
	}
	if !store.TagResource(req.ResourceName, tags) {
		return jsonErr(service.NewAWSError("InvalidARNFault", "Resource not found: "+req.ResourceName, http.StatusNotFound))
	}
	return jsonOK(map[string]any{"Tags": req.Tags})
}

// ---- UntagResource ----

type untagResourceRequest struct {
	ResourceName string   `json:"ResourceName"`
	TagKeys      []string `json:"TagKeys"`
}

func handleUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req untagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ResourceName == "" {
		return jsonErr(service.ErrValidation("ResourceName is required."))
	}
	if !store.UntagResource(req.ResourceName, req.TagKeys) {
		return jsonErr(service.NewAWSError("InvalidARNFault", "Resource not found: "+req.ResourceName, http.StatusNotFound))
	}
	return jsonOK(map[string]any{"Tags": []map[string]string{}})
}

// ---- ListTags ----

type listTagsRequest struct {
	ResourceName string `json:"ResourceName"`
}

func handleListTags(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listTagsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ResourceName == "" {
		return jsonErr(service.ErrValidation("ResourceName is required."))
	}
	tags, ok := store.ListTags(req.ResourceName)
	if !ok {
		return jsonErr(service.NewAWSError("InvalidARNFault", "Resource not found: "+req.ResourceName, http.StatusNotFound))
	}
	pairs := make([]map[string]string, 0, len(tags))
	for k, v := range tags {
		pairs = append(pairs, map[string]string{"Key": k, "Value": v})
	}
	return jsonOK(map[string]any{"Tags": pairs})
}
