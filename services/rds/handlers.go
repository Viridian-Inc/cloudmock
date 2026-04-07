package rds

import (
	"crypto/rand"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

const rdsXmlns = "http://rds.amazonaws.com/doc/2014-10-31/"

// ---- shared XML types ----

type xmlResponseMetadata struct {
	RequestID string `xml:"RequestId"`
}

type xmlTag struct {
	Key   string `xml:"Key"`
	Value string `xml:"Value"`
}

// ---- DBInstance XML types ----

type xmlEndpoint struct {
	Address string `xml:"Address"`
	Port    int    `xml:"Port"`
}

type xmlDBInstance struct {
	DBInstanceIdentifier string      `xml:"DBInstanceIdentifier"`
	DBInstanceArn        string      `xml:"DBInstanceArn"`
	DBInstanceClass      string      `xml:"DBInstanceClass"`
	Engine               string      `xml:"Engine"`
	EngineVersion        string      `xml:"EngineVersion"`
	DBInstanceStatus     string      `xml:"DBInstanceStatus"`
	MasterUsername       string      `xml:"MasterUsername"`
	AllocatedStorage     int         `xml:"AllocatedStorage"`
	Endpoint             xmlEndpoint `xml:"Endpoint"`
}

func toXMLDBInstance(inst *DBInstance) xmlDBInstance {
	return xmlDBInstance{
		DBInstanceIdentifier: inst.Identifier,
		DBInstanceArn:        inst.ARN,
		DBInstanceClass:      inst.Class,
		Engine:               inst.Engine,
		EngineVersion:        inst.EngineVersion,
		DBInstanceStatus:     inst.Status,
		MasterUsername:       inst.MasterUsername,
		AllocatedStorage:     inst.AllocatedStorage,
		Endpoint: xmlEndpoint{
			Address: inst.Endpoint.Address,
			Port:    inst.Endpoint.Port,
		},
	}
}

// ---- CreateDBInstance ----

type xmlCreateDBInstanceResponse struct {
	XMLName xml.Name              `xml:"CreateDBInstanceResponse"`
	Xmlns   string                `xml:"xmlns,attr"`
	Result  xmlCreateDBInstanceResult `xml:"CreateDBInstanceResult"`
	Meta    xmlResponseMetadata   `xml:"ResponseMetadata"`
}

type xmlCreateDBInstanceResult struct {
	DBInstance xmlDBInstance `xml:"DBInstance"`
}

func handleCreateDBInstance(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	id := form.Get("DBInstanceIdentifier")
	if id == "" {
		return xmlErr(service.ErrValidation("DBInstanceIdentifier is required."))
	}
	class := form.Get("DBInstanceClass")
	if class == "" {
		return xmlErr(service.ErrValidation("DBInstanceClass is required."))
	}
	engine := form.Get("Engine")
	if engine == "" {
		return xmlErr(service.ErrValidation("Engine is required."))
	}
	masterUser := form.Get("MasterUsername")
	engineVersion := form.Get("EngineVersion")

	allocStorage := 20
	if s := form.Get("AllocatedStorage"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			allocStorage = v
		}
	}

	inst, ok := store.CreateDBInstance(id, class, engine, engineVersion, masterUser, allocStorage)
	if !ok {
		return xmlErr(service.NewAWSError("DBInstanceAlreadyExists",
			"DB instance already exists.", http.StatusBadRequest))
	}

	return xmlOK(&xmlCreateDBInstanceResponse{
		Xmlns:  rdsXmlns,
		Result: xmlCreateDBInstanceResult{DBInstance: toXMLDBInstance(inst)},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- DescribeDBInstances ----

type xmlDescribeDBInstancesResponse struct {
	XMLName xml.Name                   `xml:"DescribeDBInstancesResponse"`
	Xmlns   string                     `xml:"xmlns,attr"`
	Result  xmlDescribeDBInstancesResult `xml:"DescribeDBInstancesResult"`
	Meta    xmlResponseMetadata        `xml:"ResponseMetadata"`
}

type xmlDescribeDBInstancesResult struct {
	DBInstances []xmlDBInstance `xml:"DBInstances>DBInstance"`
}

func handleDescribeDBInstances(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	filterID := form.Get("DBInstanceIdentifier")

	instances := store.ListDBInstances(filterID)

	if filterID != "" && len(instances) == 0 {
		return xmlErr(service.NewAWSError("DBInstanceNotFound",
			"DBInstance "+filterID+" not found.", http.StatusNotFound))
	}

	xmlInstances := make([]xmlDBInstance, 0, len(instances))
	for _, inst := range instances {
		xmlInstances = append(xmlInstances, toXMLDBInstance(inst))
	}

	return xmlOK(&xmlDescribeDBInstancesResponse{
		Xmlns:  rdsXmlns,
		Result: xmlDescribeDBInstancesResult{DBInstances: xmlInstances},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- ModifyDBInstance ----

type xmlModifyDBInstanceResponse struct {
	XMLName xml.Name                 `xml:"ModifyDBInstanceResponse"`
	Xmlns   string                   `xml:"xmlns,attr"`
	Result  xmlModifyDBInstanceResult `xml:"ModifyDBInstanceResult"`
	Meta    xmlResponseMetadata      `xml:"ResponseMetadata"`
}

type xmlModifyDBInstanceResult struct {
	DBInstance xmlDBInstance `xml:"DBInstance"`
}

func handleModifyDBInstance(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	id := form.Get("DBInstanceIdentifier")
	if id == "" {
		return xmlErr(service.ErrValidation("DBInstanceIdentifier is required."))
	}
	class := form.Get("DBInstanceClass")
	allocStorage := 0
	if s := form.Get("AllocatedStorage"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			allocStorage = v
		}
	}

	inst, ok := store.ModifyDBInstance(id, class, allocStorage)
	if !ok {
		return xmlErr(service.NewAWSError("DBInstanceNotFound",
			"DBInstance "+id+" not found.", http.StatusNotFound))
	}

	return xmlOK(&xmlModifyDBInstanceResponse{
		Xmlns:  rdsXmlns,
		Result: xmlModifyDBInstanceResult{DBInstance: toXMLDBInstance(inst)},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- DeleteDBInstance ----

type xmlDeleteDBInstanceResponse struct {
	XMLName xml.Name                 `xml:"DeleteDBInstanceResponse"`
	Xmlns   string                   `xml:"xmlns,attr"`
	Result  xmlDeleteDBInstanceResult `xml:"DeleteDBInstanceResult"`
	Meta    xmlResponseMetadata      `xml:"ResponseMetadata"`
}

type xmlDeleteDBInstanceResult struct {
	DBInstance xmlDBInstance `xml:"DBInstance"`
}

func handleDeleteDBInstance(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	id := form.Get("DBInstanceIdentifier")
	if id == "" {
		return xmlErr(service.ErrValidation("DBInstanceIdentifier is required."))
	}

	inst, ok := store.GetDBInstance(id)
	if !ok {
		return xmlErr(service.NewAWSError("DBInstanceNotFound",
			"DBInstance "+id+" not found.", http.StatusNotFound))
	}
	xmlInst := toXMLDBInstance(inst)
	store.DeleteDBInstance(id)

	return xmlOK(&xmlDeleteDBInstanceResponse{
		Xmlns:  rdsXmlns,
		Result: xmlDeleteDBInstanceResult{DBInstance: xmlInst},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- DBCluster XML types ----

type xmlDBCluster struct {
	DBClusterIdentifier string `xml:"DBClusterIdentifier"`
	DBClusterArn        string `xml:"DBClusterArn"`
	Engine              string `xml:"Engine"`
	EngineVersion       string `xml:"EngineVersion"`
	Status              string `xml:"Status"`
	MasterUsername      string `xml:"MasterUsername"`
	DatabaseName        string `xml:"DatabaseName"`
	Endpoint            string `xml:"Endpoint"`
	ReaderEndpoint      string `xml:"ReaderEndpoint"`
	Port                int    `xml:"Port"`
}

func toXMLDBCluster(c *DBCluster) xmlDBCluster {
	return xmlDBCluster{
		DBClusterIdentifier: c.Identifier,
		DBClusterArn:        c.ARN,
		Engine:              c.Engine,
		EngineVersion:       c.EngineVersion,
		Status:              c.Status,
		MasterUsername:      c.MasterUsername,
		DatabaseName:        c.DatabaseName,
		Endpoint:            c.Endpoint,
		ReaderEndpoint:      c.ReaderEndpoint,
		Port:                c.Port,
	}
}

// ---- CreateDBCluster ----

type xmlCreateDBClusterResponse struct {
	XMLName xml.Name               `xml:"CreateDBClusterResponse"`
	Xmlns   string                 `xml:"xmlns,attr"`
	Result  xmlCreateDBClusterResult `xml:"CreateDBClusterResult"`
	Meta    xmlResponseMetadata    `xml:"ResponseMetadata"`
}

type xmlCreateDBClusterResult struct {
	DBCluster xmlDBCluster `xml:"DBCluster"`
}

func handleCreateDBCluster(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	id := form.Get("DBClusterIdentifier")
	if id == "" {
		return xmlErr(service.ErrValidation("DBClusterIdentifier is required."))
	}
	engine := form.Get("Engine")
	if engine == "" {
		return xmlErr(service.ErrValidation("Engine is required."))
	}
	masterUser := form.Get("MasterUsername")
	engineVersion := form.Get("EngineVersion")
	dbName := form.Get("DatabaseName")

	cluster, ok := store.CreateDBCluster(id, engine, engineVersion, masterUser, dbName)
	if !ok {
		return xmlErr(service.NewAWSError("DBClusterAlreadyExistsFault",
			"DB cluster already exists.", http.StatusBadRequest))
	}

	return xmlOK(&xmlCreateDBClusterResponse{
		Xmlns:  rdsXmlns,
		Result: xmlCreateDBClusterResult{DBCluster: toXMLDBCluster(cluster)},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- DescribeDBClusters ----

type xmlDescribeDBClustersResponse struct {
	XMLName xml.Name                  `xml:"DescribeDBClustersResponse"`
	Xmlns   string                    `xml:"xmlns,attr"`
	Result  xmlDescribeDBClustersResult `xml:"DescribeDBClustersResult"`
	Meta    xmlResponseMetadata       `xml:"ResponseMetadata"`
}

type xmlDescribeDBClustersResult struct {
	DBClusters []xmlDBCluster `xml:"DBClusters>DBCluster"`
}

func handleDescribeDBClusters(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	filterID := form.Get("DBClusterIdentifier")

	clusters := store.ListDBClusters(filterID)

	if filterID != "" && len(clusters) == 0 {
		return xmlErr(service.NewAWSError("DBClusterNotFoundFault",
			"DBCluster "+filterID+" not found.", http.StatusNotFound))
	}

	xmlClusters := make([]xmlDBCluster, 0, len(clusters))
	for _, c := range clusters {
		xmlClusters = append(xmlClusters, toXMLDBCluster(c))
	}

	return xmlOK(&xmlDescribeDBClustersResponse{
		Xmlns:  rdsXmlns,
		Result: xmlDescribeDBClustersResult{DBClusters: xmlClusters},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- DeleteDBCluster ----

type xmlDeleteDBClusterResponse struct {
	XMLName xml.Name               `xml:"DeleteDBClusterResponse"`
	Xmlns   string                 `xml:"xmlns,attr"`
	Result  xmlDeleteDBClusterResult `xml:"DeleteDBClusterResult"`
	Meta    xmlResponseMetadata    `xml:"ResponseMetadata"`
}

type xmlDeleteDBClusterResult struct {
	DBCluster xmlDBCluster `xml:"DBCluster"`
}

func handleDeleteDBCluster(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	id := form.Get("DBClusterIdentifier")
	if id == "" {
		return xmlErr(service.ErrValidation("DBClusterIdentifier is required."))
	}

	cluster, ok := store.GetDBCluster(id)
	if !ok {
		return xmlErr(service.NewAWSError("DBClusterNotFoundFault",
			"DBCluster "+id+" not found.", http.StatusNotFound))
	}
	xmlCluster := toXMLDBCluster(cluster)
	store.DeleteDBCluster(id)

	return xmlOK(&xmlDeleteDBClusterResponse{
		Xmlns:  rdsXmlns,
		Result: xmlDeleteDBClusterResult{DBCluster: xmlCluster},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- DBSnapshot XML types ----

type xmlDBSnapshot struct {
	DBSnapshotIdentifier string `xml:"DBSnapshotIdentifier"`
	DBSnapshotArn        string `xml:"DBSnapshotArn"`
	DBInstanceIdentifier string `xml:"DBInstanceIdentifier"`
	Status               string `xml:"Status"`
	Engine               string `xml:"Engine"`
	EngineVersion        string `xml:"EngineVersion"`
	AllocatedStorage     int    `xml:"AllocatedStorage"`
	SnapshotCreateTime   string `xml:"SnapshotCreateTime"`
}

func toXMLDBSnapshot(snap *DBSnapshot) xmlDBSnapshot {
	return xmlDBSnapshot{
		DBSnapshotIdentifier: snap.Identifier,
		DBSnapshotArn:        snap.ARN,
		DBInstanceIdentifier: snap.DBInstanceIdentifier,
		Status:               snap.Status,
		Engine:               snap.Engine,
		EngineVersion:        snap.EngineVersion,
		AllocatedStorage:     snap.AllocatedStorage,
		SnapshotCreateTime:   snap.SnapshotCreateTime.Format("2006-01-02T15:04:05Z"),
	}
}

// ---- CreateDBSnapshot ----

type xmlCreateDBSnapshotResponse struct {
	XMLName xml.Name                `xml:"CreateDBSnapshotResponse"`
	Xmlns   string                  `xml:"xmlns,attr"`
	Result  xmlCreateDBSnapshotResult `xml:"CreateDBSnapshotResult"`
	Meta    xmlResponseMetadata     `xml:"ResponseMetadata"`
}

type xmlCreateDBSnapshotResult struct {
	DBSnapshot xmlDBSnapshot `xml:"DBSnapshot"`
}

func handleCreateDBSnapshot(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	snapshotID := form.Get("DBSnapshotIdentifier")
	if snapshotID == "" {
		return xmlErr(service.ErrValidation("DBSnapshotIdentifier is required."))
	}
	instanceID := form.Get("DBInstanceIdentifier")
	if instanceID == "" {
		return xmlErr(service.ErrValidation("DBInstanceIdentifier is required."))
	}

	snap, ok := store.CreateDBSnapshot(snapshotID, instanceID)
	if !ok {
		// Could be duplicate snapshot ID or missing instance.
		if _, instOK := store.GetDBInstance(instanceID); !instOK {
			return xmlErr(service.NewAWSError("DBInstanceNotFound",
				"DBInstance "+instanceID+" not found.", http.StatusNotFound))
		}
		return xmlErr(service.NewAWSError("DBSnapshotAlreadyExists",
			"DB snapshot already exists.", http.StatusBadRequest))
	}

	return xmlOK(&xmlCreateDBSnapshotResponse{
		Xmlns:  rdsXmlns,
		Result: xmlCreateDBSnapshotResult{DBSnapshot: toXMLDBSnapshot(snap)},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- DescribeDBSnapshots ----

type xmlDescribeDBSnapshotsResponse struct {
	XMLName xml.Name                   `xml:"DescribeDBSnapshotsResponse"`
	Xmlns   string                     `xml:"xmlns,attr"`
	Result  xmlDescribeDBSnapshotsResult `xml:"DescribeDBSnapshotsResult"`
	Meta    xmlResponseMetadata        `xml:"ResponseMetadata"`
}

type xmlDescribeDBSnapshotsResult struct {
	DBSnapshots []xmlDBSnapshot `xml:"DBSnapshots>DBSnapshot"`
}

func handleDescribeDBSnapshots(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	instanceID := form.Get("DBInstanceIdentifier")
	snapshotID := form.Get("DBSnapshotIdentifier")

	snapshots := store.ListDBSnapshots(instanceID, snapshotID)

	if snapshotID != "" && len(snapshots) == 0 {
		return xmlErr(service.NewAWSError("DBSnapshotNotFound",
			"DBSnapshot "+snapshotID+" not found.", http.StatusNotFound))
	}

	xmlSnaps := make([]xmlDBSnapshot, 0, len(snapshots))
	for _, snap := range snapshots {
		xmlSnaps = append(xmlSnaps, toXMLDBSnapshot(snap))
	}

	return xmlOK(&xmlDescribeDBSnapshotsResponse{
		Xmlns:  rdsXmlns,
		Result: xmlDescribeDBSnapshotsResult{DBSnapshots: xmlSnaps},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- DeleteDBSnapshot ----

type xmlDeleteDBSnapshotResponse struct {
	XMLName xml.Name                `xml:"DeleteDBSnapshotResponse"`
	Xmlns   string                  `xml:"xmlns,attr"`
	Result  xmlDeleteDBSnapshotResult `xml:"DeleteDBSnapshotResult"`
	Meta    xmlResponseMetadata     `xml:"ResponseMetadata"`
}

type xmlDeleteDBSnapshotResult struct {
	DBSnapshot xmlDBSnapshot `xml:"DBSnapshot"`
}

func handleDeleteDBSnapshot(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	id := form.Get("DBSnapshotIdentifier")
	if id == "" {
		return xmlErr(service.ErrValidation("DBSnapshotIdentifier is required."))
	}

	snap, ok := store.GetDBSnapshot(id)
	if !ok {
		return xmlErr(service.NewAWSError("DBSnapshotNotFound",
			"DBSnapshot "+id+" not found.", http.StatusNotFound))
	}
	xmlSnap := toXMLDBSnapshot(snap)
	store.DeleteDBSnapshot(id)

	return xmlOK(&xmlDeleteDBSnapshotResponse{
		Xmlns:  rdsXmlns,
		Result: xmlDeleteDBSnapshotResult{DBSnapshot: xmlSnap},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- DBSubnetGroup XML types ----

type xmlDBSubnetGroup struct {
	DBSubnetGroupName        string   `xml:"DBSubnetGroupName"`
	DBSubnetGroupArn         string   `xml:"DBSubnetGroupArn"`
	DBSubnetGroupDescription string   `xml:"DBSubnetGroupDescription"`
	SubnetGroupStatus        string   `xml:"SubnetGroupStatus"`
	Subnets                  []string `xml:"Subnets>SubnetIdentifier"`
}

func toXMLDBSubnetGroup(sg *DBSubnetGroup) xmlDBSubnetGroup {
	return xmlDBSubnetGroup{
		DBSubnetGroupName:        sg.Name,
		DBSubnetGroupArn:         sg.ARN,
		DBSubnetGroupDescription: sg.Description,
		SubnetGroupStatus:        sg.Status,
		Subnets:                  sg.SubnetIds,
	}
}

// ---- CreateDBSubnetGroup ----

type xmlCreateDBSubnetGroupResponse struct {
	XMLName xml.Name                   `xml:"CreateDBSubnetGroupResponse"`
	Xmlns   string                     `xml:"xmlns,attr"`
	Result  xmlCreateDBSubnetGroupResult `xml:"CreateDBSubnetGroupResult"`
	Meta    xmlResponseMetadata        `xml:"ResponseMetadata"`
}

type xmlCreateDBSubnetGroupResult struct {
	DBSubnetGroup xmlDBSubnetGroup `xml:"DBSubnetGroup"`
}

func handleCreateDBSubnetGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	name := form.Get("DBSubnetGroupName")
	if name == "" {
		return xmlErr(service.ErrValidation("DBSubnetGroupName is required."))
	}
	description := form.Get("DBSubnetGroupDescription")

	// Parse SubnetIds.member.N or SubnetIds.N
	subnetIDs := parseSubnetIDs(form)

	sg, ok := store.CreateDBSubnetGroup(name, description, subnetIDs)
	if !ok {
		return xmlErr(service.NewAWSError("DBSubnetGroupAlreadyExists",
			"DB subnet group already exists.", http.StatusBadRequest))
	}

	return xmlOK(&xmlCreateDBSubnetGroupResponse{
		Xmlns:  rdsXmlns,
		Result: xmlCreateDBSubnetGroupResult{DBSubnetGroup: toXMLDBSubnetGroup(sg)},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- DescribeDBSubnetGroups ----

type xmlDescribeDBSubnetGroupsResponse struct {
	XMLName xml.Name                     `xml:"DescribeDBSubnetGroupsResponse"`
	Xmlns   string                       `xml:"xmlns,attr"`
	Result  xmlDescribeDBSubnetGroupsResult `xml:"DescribeDBSubnetGroupsResult"`
	Meta    xmlResponseMetadata          `xml:"ResponseMetadata"`
}

type xmlDescribeDBSubnetGroupsResult struct {
	DBSubnetGroups []xmlDBSubnetGroup `xml:"DBSubnetGroups>DBSubnetGroup"`
}

func handleDescribeDBSubnetGroups(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	filterName := form.Get("DBSubnetGroupName")

	groups := store.ListDBSubnetGroups(filterName)

	if filterName != "" && len(groups) == 0 {
		return xmlErr(service.NewAWSError("DBSubnetGroupNotFoundFault",
			"DBSubnetGroup "+filterName+" not found.", http.StatusNotFound))
	}

	xmlGroups := make([]xmlDBSubnetGroup, 0, len(groups))
	for _, sg := range groups {
		xmlGroups = append(xmlGroups, toXMLDBSubnetGroup(sg))
	}

	return xmlOK(&xmlDescribeDBSubnetGroupsResponse{
		Xmlns:  rdsXmlns,
		Result: xmlDescribeDBSubnetGroupsResult{DBSubnetGroups: xmlGroups},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- DeleteDBSubnetGroup ----

type xmlDeleteDBSubnetGroupResponse struct {
	XMLName xml.Name            `xml:"DeleteDBSubnetGroupResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleDeleteDBSubnetGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	name := form.Get("DBSubnetGroupName")
	if name == "" {
		return xmlErr(service.ErrValidation("DBSubnetGroupName is required."))
	}

	if !store.DeleteDBSubnetGroup(name) {
		return xmlErr(service.NewAWSError("DBSubnetGroupNotFoundFault",
			"DBSubnetGroup "+name+" not found.", http.StatusNotFound))
	}

	return xmlOK(&xmlDeleteDBSubnetGroupResponse{
		Xmlns: rdsXmlns,
		Meta:  xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- AddTagsToResource ----

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

	if !store.AddTagsToResource(arn, tags) {
		return xmlErr(service.NewAWSError("DBInstanceNotFound",
			"Resource "+arn+" not found.", http.StatusNotFound))
	}

	return xmlOK(&xmlAddTagsToResourceResponse{
		Xmlns: rdsXmlns,
		Meta:  xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- RemoveTagsFromResource ----

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

	if !store.RemoveTagsFromResource(arn, keys) {
		return xmlErr(service.NewAWSError("DBInstanceNotFound",
			"Resource "+arn+" not found.", http.StatusNotFound))
	}

	return xmlOK(&xmlRemoveTagsFromResourceResponse{
		Xmlns: rdsXmlns,
		Meta:  xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- ListTagsForResource ----

type xmlListTagsForResourceResponse struct {
	XMLName xml.Name                   `xml:"ListTagsForResourceResponse"`
	Xmlns   string                     `xml:"xmlns,attr"`
	Result  xmlListTagsForResourceResult `xml:"ListTagsForResourceResult"`
	Meta    xmlResponseMetadata        `xml:"ResponseMetadata"`
}

type xmlListTagsForResourceResult struct {
	TagList []xmlTag `xml:"TagList>Tag"`
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	arn := form.Get("ResourceName")
	if arn == "" {
		return xmlErr(service.ErrValidation("ResourceName is required."))
	}

	tags, ok := store.ListTagsForResource(arn)
	if !ok {
		return xmlErr(service.NewAWSError("DBInstanceNotFound",
			"Resource "+arn+" not found.", http.StatusNotFound))
	}

	xmlTags := make([]xmlTag, 0, len(tags))
	for k, v := range tags {
		xmlTags = append(xmlTags, xmlTag{Key: k, Value: v})
	}

	return xmlOK(&xmlListTagsForResourceResponse{
		Xmlns:  rdsXmlns,
		Result: xmlListTagsForResourceResult{TagList: xmlTags},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- helper functions ----

// parseForm merges the query-string params and the form-encoded body into a
// single url.Values.
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

// parseTags parses Tags.member.N.Key / Tags.member.N.Value pairs.
func parseTags(form url.Values) map[string]string {
	tags := make(map[string]string)
	for i := 1; ; i++ {
		key := form.Get(fmt.Sprintf("Tags.member.%d.Key", i))
		if key == "" {
			break
		}
		val := form.Get(fmt.Sprintf("Tags.member.%d.Value", i))
		tags[key] = val
	}
	return tags
}

// parseTagKeys parses TagKeys.member.N values.
func parseTagKeys(form url.Values) []string {
	keys := make([]string, 0)
	for i := 1; ; i++ {
		k := form.Get(fmt.Sprintf("TagKeys.member.%d", i))
		if k == "" {
			break
		}
		keys = append(keys, k)
	}
	return keys
}

// parseSubnetIDs parses SubnetIds.member.N or SubnetIds.SubnetIdentifier.N values.
func parseSubnetIDs(form url.Values) []string {
	ids := make([]string, 0)
	for i := 1; ; i++ {
		id := form.Get(fmt.Sprintf("SubnetIds.member.%d", i))
		if id == "" {
			// Also try SubnetIds.SubnetIdentifier.N
			id = form.Get(fmt.Sprintf("SubnetIds.SubnetIdentifier.%d", i))
		}
		if id == "" {
			break
		}
		ids = append(ids, id)
	}
	return ids
}

// xmlOK wraps a response body in a 200 XML response.
func xmlOK(body any) (*service.Response, error) {
	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       body,
		Format:     service.FormatXML,
	}, nil
}

// xmlErr wraps an AWSError in an XML error response.
func xmlErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatXML}, awsErr
}

// newUUID returns a random UUID-shaped identifier.
func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

