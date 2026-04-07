package memorydb

import (
	"encoding/json"
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
	if len(body) == 0 { return nil }
	if err := json.Unmarshal(body, v); err != nil {
		return service.NewAWSError("InvalidParameterValueException", "Invalid JSON.", http.StatusBadRequest)
	}
	return nil
}

type tag struct { Key string `json:"Key"`; Value string `json:"Value"` }

// ---- Cluster handlers ----

type endpointJSON struct { Address string `json:"Address"`; Port int `json:"Port"` }

type clusterJSON struct {
	Name string `json:"Name"`; ARN string `json:"ARN"`; Status string `json:"Status"`
	NodeType string `json:"NodeType"`; EngineVersion string `json:"EngineVersion,omitempty"`
	NumberOfShards int `json:"NumberOfShards"`; ClusterEndpoint endpointJSON `json:"ClusterEndpoint"`
	SubnetGroupName string `json:"SubnetGroupName,omitempty"`; ACLName string `json:"ACLName,omitempty"`
	ParameterGroupName string `json:"ParameterGroupName,omitempty"`
}

func toClusterJSON(c *Cluster) clusterJSON {
	return clusterJSON{Name: c.Name, ARN: c.ARN, Status: c.Status, NodeType: c.NodeType,
		EngineVersion: c.EngineVersion, NumberOfShards: c.NumShards,
		ClusterEndpoint: endpointJSON{c.ClusterEndpoint.Address, c.ClusterEndpoint.Port},
		SubnetGroupName: c.SubnetGroupName, ACLName: c.ACLName, ParameterGroupName: c.ParameterGroupName}
}

type createClusterRequest struct {
	ClusterName string `json:"ClusterName"`; NodeType string `json:"NodeType"`
	EngineVersion string `json:"EngineVersion"`; NumShards int `json:"NumShards"`
	NumReplicasPerShard int `json:"NumReplicasPerShard"`
	SubnetGroupName string `json:"SubnetGroupName"`; ACLName string `json:"ACLName"`
	ParameterGroupName string `json:"ParameterGroupName"`; Tags []tag `json:"Tags"`
}

func handleCreateCluster(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createClusterRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil { return jsonErr(awsErr) }
	if req.ClusterName == "" { return jsonErr(service.ErrValidation("ClusterName is required.")) }
	tags := make(map[string]string); for _, t := range req.Tags { tags[t.Key] = t.Value }
	c, ok := store.CreateCluster(req.ClusterName, req.NodeType, req.EngineVersion, req.SubnetGroupName, req.ACLName, req.ParameterGroupName, req.NumShards, req.NumReplicasPerShard, tags)
	if !ok { return jsonErr(service.ErrAlreadyExists("Cluster", req.ClusterName)) }
	return jsonOK(map[string]any{"Cluster": toClusterJSON(c)})
}

type describeClustersRequest struct { ClusterName string `json:"ClusterName"` }

func handleDescribeClusters(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeClustersRequest; parseJSON(ctx.Body, &req)
	if req.ClusterName != "" {
		c, ok := store.GetCluster(req.ClusterName)
		if !ok { return jsonErr(service.NewAWSError("ClusterNotFoundFault", "Cluster not found.", http.StatusNotFound)) }
		return jsonOK(map[string]any{"Clusters": []clusterJSON{toClusterJSON(c)}})
	}
	clusters := store.ListClusters()
	list := make([]clusterJSON, 0, len(clusters)); for _, c := range clusters { list = append(list, toClusterJSON(c)) }
	return jsonOK(map[string]any{"Clusters": list})
}

type deleteClusterRequest struct { ClusterName string `json:"ClusterName"` }

func handleDeleteCluster(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteClusterRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil { return jsonErr(awsErr) }
	c, ok := store.DeleteCluster(req.ClusterName)
	if !ok { return jsonErr(service.NewAWSError("ClusterNotFoundFault", "Cluster not found.", http.StatusNotFound)) }
	return jsonOK(map[string]any{"Cluster": toClusterJSON(c)})
}

type updateClusterRequest struct {
	ClusterName string `json:"ClusterName"`; NodeType string `json:"NodeType"`
	EngineVersion string `json:"EngineVersion"`; ShardConfiguration struct{ ShardCount int `json:"ShardCount"` } `json:"ShardConfiguration"`
	ReplicaConfiguration struct{ ReplicaCount int `json:"ReplicaCount"` } `json:"ReplicaConfiguration"`
}

func handleUpdateCluster(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req updateClusterRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil { return jsonErr(awsErr) }
	c, ok := store.UpdateCluster(req.ClusterName, req.NodeType, req.EngineVersion, req.ShardConfiguration.ShardCount, req.ReplicaConfiguration.ReplicaCount)
	if !ok { return jsonErr(service.NewAWSError("ClusterNotFoundFault", "Cluster not found.", http.StatusNotFound)) }
	return jsonOK(map[string]any{"Cluster": toClusterJSON(c)})
}

// ---- ACL handlers ----

type aclJSON struct { Name string `json:"Name"`; ARN string `json:"ARN"`; Status string `json:"Status"`; UserNames []string `json:"UserNames"` }
func toACLJSON(a *ACL) aclJSON { return aclJSON{a.Name, a.ARN, a.Status, a.UserNames} }

type createACLRequest struct { ACLName string `json:"ACLName"`; UserNames []string `json:"UserNames"`; Tags []tag `json:"Tags"` }

func handleCreateACL(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createACLRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil { return jsonErr(awsErr) }
	tags := make(map[string]string); for _, t := range req.Tags { tags[t.Key] = t.Value }
	acl, ok := store.CreateACL(req.ACLName, req.UserNames, tags)
	if !ok { return jsonErr(service.ErrAlreadyExists("ACL", req.ACLName)) }
	return jsonOK(map[string]any{"ACL": toACLJSON(acl)})
}

type describeACLsRequest struct { ACLName string `json:"ACLName"` }

func handleDescribeACLs(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeACLsRequest; parseJSON(ctx.Body, &req)
	if req.ACLName != "" {
		acl, ok := store.GetACL(req.ACLName)
		if !ok { return jsonErr(service.NewAWSError("ACLNotFoundFault", "ACL not found.", http.StatusNotFound)) }
		return jsonOK(map[string]any{"ACLs": []aclJSON{toACLJSON(acl)}})
	}
	acls := store.ListACLs()
	list := make([]aclJSON, 0, len(acls)); for _, a := range acls { list = append(list, toACLJSON(a)) }
	return jsonOK(map[string]any{"ACLs": list})
}

type deleteACLRequest struct { ACLName string `json:"ACLName"` }

func handleDeleteACL(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteACLRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil { return jsonErr(awsErr) }
	if !store.DeleteACL(req.ACLName) { return jsonErr(service.NewAWSError("ACLNotFoundFault", "ACL not found.", http.StatusNotFound)) }
	return jsonOK(struct{}{})
}

type updateACLRequest struct { ACLName string `json:"ACLName"`; UserNamesToAdd []string `json:"UserNamesToAdd"`; UserNamesToRemove []string `json:"UserNamesToRemove"` }

func handleUpdateACL(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req updateACLRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil { return jsonErr(awsErr) }
	acl, ok := store.UpdateACL(req.ACLName, req.UserNamesToAdd, req.UserNamesToRemove)
	if !ok { return jsonErr(service.NewAWSError("ACLNotFoundFault", "ACL not found.", http.StatusNotFound)) }
	return jsonOK(map[string]any{"ACL": toACLJSON(acl)})
}

// ---- User handlers ----

type userJSON struct { Name string `json:"Name"`; ARN string `json:"ARN"`; Status string `json:"Status"`; AccessString string `json:"AccessString"` }
func toUserJSON(u *User) userJSON { return userJSON{u.Name, u.ARN, u.Status, u.AccessString} }

type createUserRequest struct { UserName string `json:"UserName"`; AccessString string `json:"AccessString"`; AuthenticationMode struct{ Type string `json:"Type"`; Passwords []string `json:"Passwords"` } `json:"AuthenticationMode"`; Tags []tag `json:"Tags"` }

func handleCreateUser(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createUserRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil { return jsonErr(awsErr) }
	tags := make(map[string]string); for _, t := range req.Tags { tags[t.Key] = t.Value }
	u, ok := store.CreateUser(req.UserName, req.AccessString, req.AuthenticationMode.Type, req.AuthenticationMode.Passwords, tags)
	if !ok { return jsonErr(service.ErrAlreadyExists("User", req.UserName)) }
	return jsonOK(map[string]any{"User": toUserJSON(u)})
}

type describeUsersRequest struct { UserName string `json:"UserName"` }

func handleDescribeUsers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeUsersRequest; parseJSON(ctx.Body, &req)
	if req.UserName != "" {
		u, ok := store.GetUser(req.UserName)
		if !ok { return jsonErr(service.NewAWSError("UserNotFoundFault", "User not found.", http.StatusNotFound)) }
		return jsonOK(map[string]any{"Users": []userJSON{toUserJSON(u)}})
	}
	users := store.ListUsers()
	list := make([]userJSON, 0, len(users)); for _, u := range users { list = append(list, toUserJSON(u)) }
	return jsonOK(map[string]any{"Users": list})
}

type deleteUserRequest struct { UserName string `json:"UserName"` }

func handleDeleteUser(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteUserRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil { return jsonErr(awsErr) }
	if !store.DeleteUser(req.UserName) { return jsonErr(service.NewAWSError("UserNotFoundFault", "User not found.", http.StatusNotFound)) }
	return jsonOK(struct{}{})
}

type updateUserRequest struct { UserName string `json:"UserName"`; AccessString string `json:"AccessString"` }

func handleUpdateUser(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req updateUserRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil { return jsonErr(awsErr) }
	u, ok := store.UpdateUser(req.UserName, req.AccessString)
	if !ok { return jsonErr(service.NewAWSError("UserNotFoundFault", "User not found.", http.StatusNotFound)) }
	return jsonOK(map[string]any{"User": toUserJSON(u)})
}

// ---- SubnetGroup handlers ----

type subnetGroupJSON struct { Name string `json:"Name"`; ARN string `json:"ARN"`; Description string `json:"Description,omitempty"`; SubnetIds []string `json:"Subnets"` }
func toSGJSON(sg *SubnetGroup) subnetGroupJSON { return subnetGroupJSON{sg.Name, sg.ARN, sg.Description, sg.SubnetIds} }

type createSubnetGroupRequest struct { SubnetGroupName string `json:"SubnetGroupName"`; Description string `json:"Description"`; SubnetIds []string `json:"SubnetIds"`; Tags []tag `json:"Tags"` }

func handleCreateSubnetGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createSubnetGroupRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil { return jsonErr(awsErr) }
	tags := make(map[string]string); for _, t := range req.Tags { tags[t.Key] = t.Value }
	sg, ok := store.CreateSubnetGroup(req.SubnetGroupName, req.Description, req.SubnetIds, tags)
	if !ok { return jsonErr(service.ErrAlreadyExists("SubnetGroup", req.SubnetGroupName)) }
	return jsonOK(map[string]any{"SubnetGroup": toSGJSON(sg)})
}

func handleDescribeSubnetGroups(_ *service.RequestContext, store *Store) (*service.Response, error) {
	groups := store.ListSubnetGroups()
	list := make([]subnetGroupJSON, 0, len(groups)); for _, sg := range groups { list = append(list, toSGJSON(sg)) }
	return jsonOK(map[string]any{"SubnetGroups": list})
}

type deleteSubnetGroupRequest struct { SubnetGroupName string `json:"SubnetGroupName"` }

func handleDeleteSubnetGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteSubnetGroupRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil { return jsonErr(awsErr) }
	if !store.DeleteSubnetGroup(req.SubnetGroupName) { return jsonErr(service.NewAWSError("SubnetGroupNotFoundFault", "Subnet group not found.", http.StatusNotFound)) }
	return jsonOK(struct{}{})
}

// ---- ParameterGroup handlers ----

type parameterGroupJSON struct { Name string `json:"Name"`; ARN string `json:"ARN"`; Family string `json:"Family"`; Description string `json:"Description,omitempty"` }
func toPGJSON(pg *ParameterGroup) parameterGroupJSON { return parameterGroupJSON{pg.Name, pg.ARN, pg.Family, pg.Description} }

type createParameterGroupRequest struct { ParameterGroupName string `json:"ParameterGroupName"`; Family string `json:"Family"`; Description string `json:"Description"`; Tags []tag `json:"Tags"` }

func handleCreateParameterGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createParameterGroupRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil { return jsonErr(awsErr) }
	tags := make(map[string]string); for _, t := range req.Tags { tags[t.Key] = t.Value }
	pg, ok := store.CreateParameterGroup(req.ParameterGroupName, req.Family, req.Description, tags)
	if !ok { return jsonErr(service.ErrAlreadyExists("ParameterGroup", req.ParameterGroupName)) }
	return jsonOK(map[string]any{"ParameterGroup": toPGJSON(pg)})
}

func handleDescribeParameterGroups(_ *service.RequestContext, store *Store) (*service.Response, error) {
	groups := store.ListParameterGroups()
	list := make([]parameterGroupJSON, 0, len(groups)); for _, pg := range groups { list = append(list, toPGJSON(pg)) }
	return jsonOK(map[string]any{"ParameterGroups": list})
}

type deleteParameterGroupRequest struct { ParameterGroupName string `json:"ParameterGroupName"` }

func handleDeleteParameterGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteParameterGroupRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil { return jsonErr(awsErr) }
	if !store.DeleteParameterGroup(req.ParameterGroupName) { return jsonErr(service.NewAWSError("ParameterGroupNotFoundFault", "Parameter group not found.", http.StatusNotFound)) }
	return jsonOK(struct{}{})
}

// ---- Snapshot handlers ----

type snapshotJSON struct { Name string `json:"Name"`; ARN string `json:"ARN"`; ClusterName string `json:"ClusterConfiguration>Name,omitempty"`; Status string `json:"Status"`; Source string `json:"Source"` }
func toSnapJSON(s *Snapshot) snapshotJSON { return snapshotJSON{s.Name, s.ARN, s.ClusterName, s.Status, s.Source} }

type createSnapshotRequest struct { SnapshotName string `json:"SnapshotName"`; ClusterName string `json:"ClusterName"`; Tags []tag `json:"Tags"` }

func handleCreateSnapshot(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createSnapshotRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil { return jsonErr(awsErr) }
	tags := make(map[string]string); for _, t := range req.Tags { tags[t.Key] = t.Value }
	snap, ok := store.CreateSnapshot(req.SnapshotName, req.ClusterName, tags)
	if !ok { return jsonErr(service.NewAWSError("SnapshotAlreadyExistsFault", "Snapshot already exists or cluster not found.", http.StatusBadRequest)) }
	return jsonOK(map[string]any{"Snapshot": toSnapJSON(snap)})
}

type describeSnapshotsRequest struct { ClusterName string `json:"ClusterName"`; SnapshotName string `json:"SnapshotName"` }

func handleDescribeSnapshots(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeSnapshotsRequest; parseJSON(ctx.Body, &req)
	snaps := store.ListSnapshots(req.ClusterName)
	list := make([]snapshotJSON, 0, len(snaps)); for _, s := range snaps { list = append(list, toSnapJSON(s)) }
	return jsonOK(map[string]any{"Snapshots": list})
}

type deleteSnapshotRequest struct { SnapshotName string `json:"SnapshotName"` }

func handleDeleteSnapshot(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteSnapshotRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil { return jsonErr(awsErr) }
	if !store.DeleteSnapshot(req.SnapshotName) { return jsonErr(service.NewAWSError("SnapshotNotFoundFault", "Snapshot not found.", http.StatusNotFound)) }
	return jsonOK(struct{}{})
}

// ---- Tag handlers ----

type tagResourceRequest struct { ResourceArn string `json:"ResourceArn"`; Tags []tag `json:"Tags"` }

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req tagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil { return jsonErr(awsErr) }
	tags := make(map[string]string); for _, t := range req.Tags { tags[t.Key] = t.Value }
	if !store.TagResource(req.ResourceArn, tags) { return jsonErr(service.ErrNotFound("Resource", req.ResourceArn)) }
	return jsonOK(map[string]any{"TagList": req.Tags})
}

type untagResourceRequest struct { ResourceArn string `json:"ResourceArn"`; TagKeys []string `json:"TagKeys"` }

func handleUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req untagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil { return jsonErr(awsErr) }
	if !store.UntagResource(req.ResourceArn, req.TagKeys) { return jsonErr(service.ErrNotFound("Resource", req.ResourceArn)) }
	return jsonOK(map[string]any{"TagList": []tag{}})
}

type listTagsRequest struct { ResourceArn string `json:"ResourceArn"` }

func handleListTags(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listTagsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil { return jsonErr(awsErr) }
	tags, ok := store.ListTags(req.ResourceArn)
	if !ok { return jsonErr(service.ErrNotFound("Resource", req.ResourceArn)) }
	tagList := make([]tag, 0, len(tags)); for k, v := range tags { tagList = append(tagList, tag{k, v}) }
	return jsonOK(map[string]any{"TagList": tagList})
}
