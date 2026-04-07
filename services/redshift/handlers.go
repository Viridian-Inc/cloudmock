package redshift

import (
	"crypto/rand"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

const redshiftXmlns = "http://redshift.amazonaws.com/doc/2012-12-01/"

// ---- shared XML types ----

type xmlResponseMetadata struct {
	RequestID string `xml:"RequestId"`
}

type xmlTag struct {
	Key   string `xml:"Key"`
	Value string `xml:"Value"`
}

// ---- Cluster XML ----

type xmlEndpoint struct {
	Address string `xml:"Address"`
	Port    int    `xml:"Port"`
}

type xmlCluster struct {
	ClusterIdentifier  string      `xml:"ClusterIdentifier"`
	ClusterArn         string      `xml:"ClusterArn"`
	NodeType           string      `xml:"NodeType"`
	NumberOfNodes      int         `xml:"NumberOfNodes"`
	MasterUsername     string      `xml:"MasterUsername"`
	DBName             string      `xml:"DBName"`
	ClusterStatus      string      `xml:"ClusterStatus"`
	Endpoint           xmlEndpoint `xml:"Endpoint"`
	ClusterCreateTime  string      `xml:"ClusterCreateTime"`
}

func toXMLCluster(c *Cluster) xmlCluster {
	return xmlCluster{
		ClusterIdentifier:  c.Identifier,
		ClusterArn:         c.ARN,
		NodeType:           c.NodeType,
		NumberOfNodes:      c.NumberOfNodes,
		MasterUsername:     c.MasterUsername,
		DBName:             c.DBName,
		ClusterStatus:      c.Status,
		Endpoint:           xmlEndpoint{Address: c.Endpoint.Address, Port: c.Endpoint.Port},
		ClusterCreateTime:  c.CreatedTime.Format("2006-01-02T15:04:05Z"),
	}
}

// ---- helpers ----

func parseForm(ctx *service.RequestContext) url.Values {
	form := make(url.Values)
	for k, v := range ctx.Params {
		form.Set(k, v)
	}
	if len(ctx.Body) > 0 {
		if bodyVals, err := url.ParseQuery(string(ctx.Body)); err == nil {
			for k, vs := range bodyVals {
				for _, v := range vs {
					form.Add(k, v)
				}
			}
		}
	}
	return form
}

func parseTags(form url.Values) map[string]string {
	tags := make(map[string]string)
	for i := 1; ; i++ {
		key := form.Get(fmt.Sprintf("Tags.Tag.%d.Key", i))
		if key == "" {
			break
		}
		val := form.Get(fmt.Sprintf("Tags.Tag.%d.Value", i))
		tags[key] = val
	}
	return tags
}

func parseTagKeys(form url.Values) []string {
	keys := make([]string, 0)
	for i := 1; ; i++ {
		k := form.Get(fmt.Sprintf("TagKeys.TagKey.%d", i))
		if k == "" {
			break
		}
		keys = append(keys, k)
	}
	return keys
}

func parseSubnetIDs(form url.Values) []string {
	ids := make([]string, 0)
	for i := 1; ; i++ {
		id := form.Get(fmt.Sprintf("SubnetIds.SubnetIdentifier.%d", i))
		if id == "" {
			break
		}
		ids = append(ids, id)
	}
	return ids
}

func xmlOK(body any) (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: body, Format: service.FormatXML}, nil
}

func xmlErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatXML}, awsErr
}

func newRequestID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// ---- CreateCluster ----

type xmlCreateClusterResponse struct {
	XMLName xml.Name            `xml:"CreateClusterResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Result  xmlCreateClusterResult `xml:"CreateClusterResult"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

type xmlCreateClusterResult struct {
	Cluster xmlCluster `xml:"Cluster"`
}

func handleCreateCluster(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	id := form.Get("ClusterIdentifier")
	if id == "" {
		return xmlErr(service.ErrValidation("ClusterIdentifier is required."))
	}
	nodeType := form.Get("NodeType")
	if nodeType == "" {
		nodeType = "dc2.large"
	}
	numNodes := 1
	if s := form.Get("NumberOfNodes"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			numNodes = v
		}
	}
	masterUser := form.Get("MasterUsername")
	dbName := form.Get("DBName")
	subnetGroup := form.Get("ClusterSubnetGroupName")
	paramGroup := form.Get("ClusterParameterGroupName")
	tags := parseTags(form)

	c, ok := store.CreateCluster(id, nodeType, numNodes, masterUser, dbName, subnetGroup, paramGroup, tags)
	if !ok {
		return xmlErr(service.NewAWSError("ClusterAlreadyExists", "Cluster already exists: "+id, http.StatusBadRequest))
	}
	return xmlOK(&xmlCreateClusterResponse{
		Xmlns:  redshiftXmlns,
		Result: xmlCreateClusterResult{Cluster: toXMLCluster(c)},
		Meta:   xmlResponseMetadata{RequestID: newRequestID()},
	})
}

// ---- DescribeClusters ----

type xmlDescribeClustersResponse struct {
	XMLName xml.Name               `xml:"DescribeClustersResponse"`
	Xmlns   string                 `xml:"xmlns,attr"`
	Result  xmlDescribeClustersResult `xml:"DescribeClustersResult"`
	Meta    xmlResponseMetadata    `xml:"ResponseMetadata"`
}

type xmlDescribeClustersResult struct {
	Clusters []xmlCluster `xml:"Clusters>Cluster"`
}

func handleDescribeClusters(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	filterID := form.Get("ClusterIdentifier")
	clusters := store.ListClusters(filterID)
	if filterID != "" && len(clusters) == 0 {
		return xmlErr(service.NewAWSError("ClusterNotFound", "Cluster "+filterID+" not found.", http.StatusNotFound))
	}
	xmlClusters := make([]xmlCluster, 0, len(clusters))
	for _, c := range clusters {
		xmlClusters = append(xmlClusters, toXMLCluster(c))
	}
	return xmlOK(&xmlDescribeClustersResponse{
		Xmlns:  redshiftXmlns,
		Result: xmlDescribeClustersResult{Clusters: xmlClusters},
		Meta:   xmlResponseMetadata{RequestID: newRequestID()},
	})
}

// ---- DeleteCluster ----

type xmlDeleteClusterResponse struct {
	XMLName xml.Name              `xml:"DeleteClusterResponse"`
	Xmlns   string                `xml:"xmlns,attr"`
	Result  xmlDeleteClusterResult `xml:"DeleteClusterResult"`
	Meta    xmlResponseMetadata   `xml:"ResponseMetadata"`
}

type xmlDeleteClusterResult struct {
	Cluster xmlCluster `xml:"Cluster"`
}

func handleDeleteCluster(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	id := form.Get("ClusterIdentifier")
	if id == "" {
		return xmlErr(service.ErrValidation("ClusterIdentifier is required."))
	}
	c, ok := store.DeleteCluster(id)
	if !ok {
		return xmlErr(service.NewAWSError("ClusterNotFound", "Cluster "+id+" not found.", http.StatusNotFound))
	}
	return xmlOK(&xmlDeleteClusterResponse{
		Xmlns:  redshiftXmlns,
		Result: xmlDeleteClusterResult{Cluster: toXMLCluster(c)},
		Meta:   xmlResponseMetadata{RequestID: newRequestID()},
	})
}

// ---- ModifyCluster ----

type xmlModifyClusterResponse struct {
	XMLName xml.Name              `xml:"ModifyClusterResponse"`
	Xmlns   string                `xml:"xmlns,attr"`
	Result  xmlModifyClusterResult `xml:"ModifyClusterResult"`
	Meta    xmlResponseMetadata   `xml:"ResponseMetadata"`
}

type xmlModifyClusterResult struct {
	Cluster xmlCluster `xml:"Cluster"`
}

func handleModifyCluster(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	id := form.Get("ClusterIdentifier")
	if id == "" {
		return xmlErr(service.ErrValidation("ClusterIdentifier is required."))
	}
	nodeType := form.Get("NodeType")
	numNodes := 0
	if s := form.Get("NumberOfNodes"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			numNodes = v
		}
	}
	c, ok := store.ModifyCluster(id, nodeType, numNodes)
	if !ok {
		return xmlErr(service.NewAWSError("ClusterNotFound", "Cluster "+id+" not found.", http.StatusNotFound))
	}
	return xmlOK(&xmlModifyClusterResponse{
		Xmlns:  redshiftXmlns,
		Result: xmlModifyClusterResult{Cluster: toXMLCluster(c)},
		Meta:   xmlResponseMetadata{RequestID: newRequestID()},
	})
}

// ---- RebootCluster ----

type xmlRebootClusterResponse struct {
	XMLName xml.Name              `xml:"RebootClusterResponse"`
	Xmlns   string                `xml:"xmlns,attr"`
	Result  xmlRebootClusterResult `xml:"RebootClusterResult"`
	Meta    xmlResponseMetadata   `xml:"ResponseMetadata"`
}

type xmlRebootClusterResult struct {
	Cluster xmlCluster `xml:"Cluster"`
}

func handleRebootCluster(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	id := form.Get("ClusterIdentifier")
	if id == "" {
		return xmlErr(service.ErrValidation("ClusterIdentifier is required."))
	}
	c, ok := store.RebootCluster(id)
	if !ok {
		return xmlErr(service.NewAWSError("ClusterNotFound", "Cluster "+id+" not found.", http.StatusNotFound))
	}
	return xmlOK(&xmlRebootClusterResponse{
		Xmlns:  redshiftXmlns,
		Result: xmlRebootClusterResult{Cluster: toXMLCluster(c)},
		Meta:   xmlResponseMetadata{RequestID: newRequestID()},
	})
}

// ---- Snapshot handlers ----

type xmlSnapshot struct {
	SnapshotIdentifier string `xml:"SnapshotIdentifier"`
	ClusterIdentifier  string `xml:"ClusterIdentifier"`
	Status             string `xml:"Status"`
	NodeType           string `xml:"NodeType"`
	NumberOfNodes      int    `xml:"NumberOfNodes"`
	DBName             string `xml:"DBName"`
	MasterUsername     string `xml:"MasterUsername"`
	SnapshotCreateTime string `xml:"SnapshotCreateTime"`
}

func toXMLSnapshot(s *ClusterSnapshot) xmlSnapshot {
	return xmlSnapshot{
		SnapshotIdentifier: s.Identifier, ClusterIdentifier: s.ClusterIdentifier,
		Status: s.Status, NodeType: s.NodeType, NumberOfNodes: s.NumberOfNodes,
		DBName: s.DBName, MasterUsername: s.MasterUsername,
		SnapshotCreateTime: s.SnapshotCreateTime.Format("2006-01-02T15:04:05Z"),
	}
}

type xmlCreateClusterSnapshotResponse struct {
	XMLName xml.Name            `xml:"CreateClusterSnapshotResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Result  struct{ Snapshot xmlSnapshot } `xml:"CreateClusterSnapshotResult"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleCreateClusterSnapshot(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	snapshotID := form.Get("SnapshotIdentifier")
	clusterID := form.Get("ClusterIdentifier")
	if snapshotID == "" || clusterID == "" {
		return xmlErr(service.ErrValidation("SnapshotIdentifier and ClusterIdentifier are required."))
	}
	tags := parseTags(form)
	snap, ok := store.CreateClusterSnapshot(snapshotID, clusterID, tags)
	if !ok {
		return xmlErr(service.NewAWSError("ClusterSnapshotAlreadyExists", "Snapshot already exists or cluster not found.", http.StatusBadRequest))
	}
	resp := &xmlCreateClusterSnapshotResponse{Xmlns: redshiftXmlns, Meta: xmlResponseMetadata{RequestID: newRequestID()}}
	resp.Result.Snapshot = toXMLSnapshot(snap)
	return xmlOK(resp)
}

type xmlDescribeClusterSnapshotsResponse struct {
	XMLName xml.Name            `xml:"DescribeClusterSnapshotsResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Result  struct{ Snapshots []xmlSnapshot `xml:"Snapshots>Snapshot"` } `xml:"DescribeClusterSnapshotsResult"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleDescribeClusterSnapshots(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	snaps := store.ListClusterSnapshots(form.Get("ClusterIdentifier"), form.Get("SnapshotIdentifier"))
	xmlSnaps := make([]xmlSnapshot, 0, len(snaps))
	for _, s := range snaps {
		xmlSnaps = append(xmlSnaps, toXMLSnapshot(s))
	}
	resp := &xmlDescribeClusterSnapshotsResponse{Xmlns: redshiftXmlns, Meta: xmlResponseMetadata{RequestID: newRequestID()}}
	resp.Result.Snapshots = xmlSnaps
	return xmlOK(resp)
}

type xmlDeleteClusterSnapshotResponse struct {
	XMLName xml.Name            `xml:"DeleteClusterSnapshotResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleDeleteClusterSnapshot(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	id := form.Get("SnapshotIdentifier")
	if id == "" {
		return xmlErr(service.ErrValidation("SnapshotIdentifier is required."))
	}
	if !store.DeleteClusterSnapshot(id) {
		return xmlErr(service.NewAWSError("ClusterSnapshotNotFound", "Snapshot "+id+" not found.", http.StatusNotFound))
	}
	return xmlOK(&xmlDeleteClusterSnapshotResponse{Xmlns: redshiftXmlns, Meta: xmlResponseMetadata{RequestID: newRequestID()}})
}

type xmlRestoreFromClusterSnapshotResponse struct {
	XMLName xml.Name            `xml:"RestoreFromClusterSnapshotResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Result  xmlCreateClusterResult `xml:"RestoreFromClusterSnapshotResult"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleRestoreFromClusterSnapshot(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	clusterID := form.Get("ClusterIdentifier")
	snapshotID := form.Get("SnapshotIdentifier")
	if clusterID == "" || snapshotID == "" {
		return xmlErr(service.ErrValidation("ClusterIdentifier and SnapshotIdentifier are required."))
	}
	c, ok := store.RestoreFromClusterSnapshot(clusterID, snapshotID)
	if !ok {
		return xmlErr(service.NewAWSError("ClusterSnapshotNotFound", "Snapshot not found or cluster already exists.", http.StatusBadRequest))
	}
	return xmlOK(&xmlRestoreFromClusterSnapshotResponse{
		Xmlns:  redshiftXmlns,
		Result: xmlCreateClusterResult{Cluster: toXMLCluster(c)},
		Meta:   xmlResponseMetadata{RequestID: newRequestID()},
	})
}

// ---- SubnetGroup handlers ----

type xmlSubnetGroup struct {
	ClusterSubnetGroupName string   `xml:"ClusterSubnetGroupName"`
	Description            string   `xml:"Description"`
	SubnetGroupStatus      string   `xml:"SubnetGroupStatus"`
	Subnets                []string `xml:"Subnets>Subnet>SubnetIdentifier"`
}

func toXMLSubnetGroup(sg *ClusterSubnetGroup) xmlSubnetGroup {
	return xmlSubnetGroup{
		ClusterSubnetGroupName: sg.Name, Description: sg.Description,
		SubnetGroupStatus: sg.Status, Subnets: sg.SubnetIds,
	}
}

type xmlCreateClusterSubnetGroupResponse struct {
	XMLName xml.Name            `xml:"CreateClusterSubnetGroupResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Result  struct{ ClusterSubnetGroup xmlSubnetGroup } `xml:"CreateClusterSubnetGroupResult"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleCreateClusterSubnetGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	name := form.Get("ClusterSubnetGroupName")
	if name == "" {
		return xmlErr(service.ErrValidation("ClusterSubnetGroupName is required."))
	}
	desc := form.Get("Description")
	subnetIDs := parseSubnetIDs(form)
	tags := parseTags(form)
	sg, ok := store.CreateClusterSubnetGroup(name, desc, subnetIDs, tags)
	if !ok {
		return xmlErr(service.ErrAlreadyExists("ClusterSubnetGroup", name))
	}
	resp := &xmlCreateClusterSubnetGroupResponse{Xmlns: redshiftXmlns, Meta: xmlResponseMetadata{RequestID: newRequestID()}}
	resp.Result.ClusterSubnetGroup = toXMLSubnetGroup(sg)
	return xmlOK(resp)
}

type xmlDescribeClusterSubnetGroupsResponse struct {
	XMLName xml.Name            `xml:"DescribeClusterSubnetGroupsResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Result  struct{ ClusterSubnetGroups []xmlSubnetGroup `xml:"ClusterSubnetGroups>ClusterSubnetGroup"` } `xml:"DescribeClusterSubnetGroupsResult"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleDescribeClusterSubnetGroups(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	groups := store.ListClusterSubnetGroups(form.Get("ClusterSubnetGroupName"))
	xmlGroups := make([]xmlSubnetGroup, 0, len(groups))
	for _, sg := range groups {
		xmlGroups = append(xmlGroups, toXMLSubnetGroup(sg))
	}
	resp := &xmlDescribeClusterSubnetGroupsResponse{Xmlns: redshiftXmlns, Meta: xmlResponseMetadata{RequestID: newRequestID()}}
	resp.Result.ClusterSubnetGroups = xmlGroups
	return xmlOK(resp)
}

type xmlDeleteClusterSubnetGroupResponse struct {
	XMLName xml.Name            `xml:"DeleteClusterSubnetGroupResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleDeleteClusterSubnetGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	name := form.Get("ClusterSubnetGroupName")
	if !store.DeleteClusterSubnetGroup(name) {
		return xmlErr(service.NewAWSError("ClusterSubnetGroupNotFoundFault", "Subnet group "+name+" not found.", http.StatusNotFound))
	}
	return xmlOK(&xmlDeleteClusterSubnetGroupResponse{Xmlns: redshiftXmlns, Meta: xmlResponseMetadata{RequestID: newRequestID()}})
}

// ---- ParameterGroup handlers ----

type xmlParameterGroup struct {
	ParameterGroupName   string `xml:"ParameterGroupName"`
	ParameterGroupFamily string `xml:"ParameterGroupFamily"`
	Description          string `xml:"Description"`
}

func toXMLParameterGroup(pg *ClusterParameterGroup) xmlParameterGroup {
	return xmlParameterGroup{ParameterGroupName: pg.Name, ParameterGroupFamily: pg.Family, Description: pg.Description}
}

type xmlCreateClusterParameterGroupResponse struct {
	XMLName xml.Name            `xml:"CreateClusterParameterGroupResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Result  struct{ ClusterParameterGroup xmlParameterGroup } `xml:"CreateClusterParameterGroupResult"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleCreateClusterParameterGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	name := form.Get("ParameterGroupName")
	family := form.Get("ParameterGroupFamily")
	desc := form.Get("Description")
	if name == "" {
		return xmlErr(service.ErrValidation("ParameterGroupName is required."))
	}
	tags := parseTags(form)
	pg, ok := store.CreateClusterParameterGroup(name, family, desc, tags)
	if !ok {
		return xmlErr(service.ErrAlreadyExists("ClusterParameterGroup", name))
	}
	resp := &xmlCreateClusterParameterGroupResponse{Xmlns: redshiftXmlns, Meta: xmlResponseMetadata{RequestID: newRequestID()}}
	resp.Result.ClusterParameterGroup = toXMLParameterGroup(pg)
	return xmlOK(resp)
}

type xmlDescribeClusterParameterGroupsResponse struct {
	XMLName xml.Name            `xml:"DescribeClusterParameterGroupsResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Result  struct{ ParameterGroups []xmlParameterGroup `xml:"ParameterGroups>ClusterParameterGroup"` } `xml:"DescribeClusterParameterGroupsResult"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleDescribeClusterParameterGroups(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	groups := store.ListClusterParameterGroups(form.Get("ParameterGroupName"))
	xmlGroups := make([]xmlParameterGroup, 0, len(groups))
	for _, pg := range groups {
		xmlGroups = append(xmlGroups, toXMLParameterGroup(pg))
	}
	resp := &xmlDescribeClusterParameterGroupsResponse{Xmlns: redshiftXmlns, Meta: xmlResponseMetadata{RequestID: newRequestID()}}
	resp.Result.ParameterGroups = xmlGroups
	return xmlOK(resp)
}

type xmlDeleteClusterParameterGroupResponse struct {
	XMLName xml.Name            `xml:"DeleteClusterParameterGroupResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleDeleteClusterParameterGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	name := form.Get("ParameterGroupName")
	if !store.DeleteClusterParameterGroup(name) {
		return xmlErr(service.NewAWSError("ClusterParameterGroupNotFoundFault", "Parameter group "+name+" not found.", http.StatusNotFound))
	}
	return xmlOK(&xmlDeleteClusterParameterGroupResponse{Xmlns: redshiftXmlns, Meta: xmlResponseMetadata{RequestID: newRequestID()}})
}

// ---- Tag handlers ----

type xmlCreateTagsResponse struct {
	XMLName xml.Name            `xml:"CreateTagsResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleCreateTags(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	arn := form.Get("ResourceName")
	if arn == "" {
		return xmlErr(service.ErrValidation("ResourceName is required."))
	}
	tags := parseTags(form)
	if !store.AddTags(arn, tags) {
		return xmlErr(service.NewAWSError("ResourceNotFoundFault", "Resource "+arn+" not found.", http.StatusNotFound))
	}
	return xmlOK(&xmlCreateTagsResponse{Xmlns: redshiftXmlns, Meta: xmlResponseMetadata{RequestID: newRequestID()}})
}

type xmlDeleteTagsResponse struct {
	XMLName xml.Name            `xml:"DeleteTagsResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleDeleteTags(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	arn := form.Get("ResourceName")
	if arn == "" {
		return xmlErr(service.ErrValidation("ResourceName is required."))
	}
	keys := parseTagKeys(form)
	if !store.RemoveTags(arn, keys) {
		return xmlErr(service.NewAWSError("ResourceNotFoundFault", "Resource "+arn+" not found.", http.StatusNotFound))
	}
	return xmlOK(&xmlDeleteTagsResponse{Xmlns: redshiftXmlns, Meta: xmlResponseMetadata{RequestID: newRequestID()}})
}

type xmlDescribeTagsResponse struct {
	XMLName xml.Name            `xml:"DescribeTagsResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Result  struct{ TaggedResources []xmlTag `xml:"TaggedResources>TaggedResource>Tag"` } `xml:"DescribeTagsResult"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleDescribeTags(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	arn := form.Get("ResourceName")
	tags, ok := store.ListTags(arn)
	if !ok {
		tags = make(map[string]string)
	}
	xmlTags := make([]xmlTag, 0, len(tags))
	for k, v := range tags {
		xmlTags = append(xmlTags, xmlTag{Key: k, Value: v})
	}
	resp := &xmlDescribeTagsResponse{Xmlns: redshiftXmlns, Meta: xmlResponseMetadata{RequestID: newRequestID()}}
	resp.Result.TaggedResources = xmlTags
	return xmlOK(resp)
}

// ---- PauseCluster ----

type xmlPauseClusterResponse struct {
	XMLName xml.Name             `xml:"PauseClusterResponse"`
	Xmlns   string               `xml:"xmlns,attr"`
	Result  xmlPauseClusterResult `xml:"PauseClusterResult"`
	Meta    xmlResponseMetadata  `xml:"ResponseMetadata"`
}

type xmlPauseClusterResult struct {
	Cluster xmlCluster `xml:"Cluster"`
}

func handlePauseCluster(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	id := form.Get("ClusterIdentifier")
	if id == "" {
		return xmlErr(service.ErrValidation("ClusterIdentifier is required."))
	}
	c, ok := store.PauseCluster(id)
	if !ok {
		return xmlErr(service.NewAWSError("ClusterNotFound", "Cluster "+id+" not found.", http.StatusNotFound))
	}
	return xmlOK(&xmlPauseClusterResponse{
		Xmlns:  redshiftXmlns,
		Result: xmlPauseClusterResult{Cluster: toXMLCluster(c)},
		Meta:   xmlResponseMetadata{RequestID: newRequestID()},
	})
}

// ---- ResumeCluster ----

type xmlResumeClusterResponse struct {
	XMLName xml.Name              `xml:"ResumeClusterResponse"`
	Xmlns   string                `xml:"xmlns,attr"`
	Result  xmlResumeClusterResult `xml:"ResumeClusterResult"`
	Meta    xmlResponseMetadata   `xml:"ResponseMetadata"`
}

type xmlResumeClusterResult struct {
	Cluster xmlCluster `xml:"Cluster"`
}

func handleResumeCluster(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	id := form.Get("ClusterIdentifier")
	if id == "" {
		return xmlErr(service.ErrValidation("ClusterIdentifier is required."))
	}
	c, ok := store.ResumeCluster(id)
	if !ok {
		return xmlErr(service.NewAWSError("ClusterNotFound", "Cluster "+id+" not found.", http.StatusNotFound))
	}
	return xmlOK(&xmlResumeClusterResponse{
		Xmlns:  redshiftXmlns,
		Result: xmlResumeClusterResult{Cluster: toXMLCluster(c)},
		Meta:   xmlResponseMetadata{RequestID: newRequestID()},
	})
}

// ---- AddTagsToResource / RemoveTagsFromResource ----

type xmlAddTagsToResourceResponse struct {
	XMLName xml.Name            `xml:"AddTagsToResourceResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleAddTagsToResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	arn := form.Get("ResourceName")
	if arn == "" {
		return xmlErr(service.ErrValidation("ResourceName is required."))
	}
	tags := parseTags(form)
	if !store.AddTags(arn, tags) {
		return xmlErr(service.NewAWSError("ResourceNotFoundFault", "Resource "+arn+" not found.", http.StatusNotFound))
	}
	return xmlOK(&xmlAddTagsToResourceResponse{Xmlns: redshiftXmlns, Meta: xmlResponseMetadata{RequestID: newRequestID()}})
}

type xmlRemoveTagsFromResourceResponse struct {
	XMLName xml.Name            `xml:"RemoveTagsFromResourceResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleRemoveTagsFromResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	arn := form.Get("ResourceName")
	if arn == "" {
		return xmlErr(service.ErrValidation("ResourceName is required."))
	}
	keys := parseTagKeys(form)
	if !store.RemoveTags(arn, keys) {
		return xmlErr(service.NewAWSError("ResourceNotFoundFault", "Resource "+arn+" not found.", http.StatusNotFound))
	}
	return xmlOK(&xmlRemoveTagsFromResourceResponse{Xmlns: redshiftXmlns, Meta: xmlResponseMetadata{RequestID: newRequestID()}})
}

// ---- Data API handlers (JSON protocol) ----

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: body, Format: service.FormatJSON}, nil
}

func jsonErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
}

func parseJSONBody(body []byte, v any) *service.AWSError {
	if len(body) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, v); err != nil {
		return service.NewAWSError("ValidationException", "Invalid JSON.", http.StatusBadRequest)
	}
	return nil
}

type executeStatementRequest struct {
	ClusterIdentifier string `json:"ClusterIdentifier"`
	Database          string `json:"Database"`
	Sql               string `json:"Sql"`
}

func handleExecuteStatement(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req executeStatementRequest
	if awsErr := parseJSONBody(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ClusterIdentifier == "" || req.Sql == "" {
		return jsonErr(service.ErrValidation("ClusterIdentifier and Sql are required."))
	}
	stmt, err := store.ExecuteStatement(req.ClusterIdentifier, req.Database, req.Sql)
	if err != nil {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", err.Error(), http.StatusNotFound))
	}
	return jsonOK(map[string]any{
		"Id":                stmt.ID,
		"ClusterIdentifier": stmt.ClusterID,
		"Database":          stmt.Database,
		"CreatedAt":         float64(stmt.CreatedAt.Unix()),
		"Status":            stmt.Status,
	})
}

type describeStatementRequest struct {
	Id string `json:"Id"`
}

func handleDescribeStatement(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeStatementRequest
	if awsErr := parseJSONBody(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	stmt, ok := store.GetStatement(req.Id)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Statement "+req.Id+" not found.", http.StatusNotFound))
	}
	result := map[string]any{
		"Id":                stmt.ID,
		"ClusterIdentifier": stmt.ClusterID,
		"Database":          stmt.Database,
		"Status":            stmt.Status,
		"CreatedAt":         float64(stmt.CreatedAt.Unix()),
		"UpdatedAt":         float64(stmt.UpdatedAt.Unix()),
		"QueryString":       stmt.SQL,
		"ResultRows":        stmt.ResultRows,
		"ResultSize":        stmt.ResultSize,
	}
	if stmt.Error != "" {
		result["Error"] = stmt.Error
	}
	return jsonOK(result)
}

type getStatementResultRequest struct {
	Id string `json:"Id"`
}

func handleGetStatementResult(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getStatementResultRequest
	if awsErr := parseJSONBody(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	stmt, ok := store.GetStatement(req.Id)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Statement "+req.Id+" not found.", http.StatusNotFound))
	}
	if stmt.Status != "FINISHED" {
		return jsonErr(service.NewAWSError("ValidationException", "Statement is not finished. Current status: "+stmt.Status, http.StatusBadRequest))
	}

	// Build column metadata
	colMeta := make([]map[string]string, len(stmt.ResultColumns))
	for i, c := range stmt.ResultColumns {
		colMeta[i] = map[string]string{"name": c.Name, "typeName": c.DataType}
	}

	// Build records
	records := make([][]map[string]string, len(stmt.ResultData))
	for i, row := range stmt.ResultData {
		record := make([]map[string]string, len(row))
		for j, val := range row {
			record[j] = map[string]string{"stringValue": val}
		}
		records[i] = record
	}

	return jsonOK(map[string]any{
		"ColumnMetadata": colMeta,
		"Records":        records,
		"TotalNumRows":   stmt.ResultRows,
	})
}
