package neptune

import (
	"crypto/rand"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"

	"github.com/neureaux/cloudmock/pkg/service"
)

const neptuneXmlns = "http://rds.amazonaws.com/doc/2014-10-31/"

type xmlResponseMetadata struct{ RequestID string `xml:"RequestId"` }
type xmlTag struct{ Key string `xml:"Key"`; Value string `xml:"Value"` }

func parseForm(ctx *service.RequestContext) url.Values {
	form := make(url.Values)
	for k, v := range ctx.Params { form.Set(k, v) }
	if len(ctx.Body) > 0 {
		if bv, err := url.ParseQuery(string(ctx.Body)); err == nil {
			for k, vs := range bv { for _, v := range vs { form.Add(k, v) } }
		}
	}
	return form
}

func parseTags(form url.Values) map[string]string {
	tags := make(map[string]string)
	for i := 1; ; i++ {
		k := form.Get(fmt.Sprintf("Tags.member.%d.Key", i))
		if k == "" { break }
		tags[k] = form.Get(fmt.Sprintf("Tags.member.%d.Value", i))
	}
	return tags
}

func parseTagKeys(form url.Values) []string {
	keys := make([]string, 0)
	for i := 1; ; i++ {
		k := form.Get(fmt.Sprintf("TagKeys.member.%d", i))
		if k == "" { break }
		keys = append(keys, k)
	}
	return keys
}

func parseSubnetIDs(form url.Values) []string {
	ids := make([]string, 0)
	for i := 1; ; i++ {
		id := form.Get(fmt.Sprintf("SubnetIds.member.%d", i))
		if id == "" { break }
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
func reqID() string {
	b := make([]byte, 16); _, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// ---- Cluster XML ----
type xmlCluster struct {
	DBClusterIdentifier string `xml:"DBClusterIdentifier"`
	DBClusterArn        string `xml:"DBClusterArn"`
	Engine              string `xml:"Engine"`
	EngineVersion       string `xml:"EngineVersion"`
	Status              string `xml:"Status"`
	Endpoint            string `xml:"Endpoint"`
	ReaderEndpoint      string `xml:"ReaderEndpoint"`
	Port                int    `xml:"Port"`
}
func toXC(c *DBCluster) xmlCluster {
	return xmlCluster{c.Identifier, c.ARN, c.Engine, c.EngineVersion, c.Status, c.Endpoint, c.ReaderEndpoint, c.Port}
}

type xmlCreateDBClusterResp struct { XMLName xml.Name `xml:"CreateDBClusterResponse"`; Xmlns string `xml:"xmlns,attr"`; Result struct{ DBCluster xmlCluster } `xml:"CreateDBClusterResult"`; Meta xmlResponseMetadata `xml:"ResponseMetadata"` }

func handleCreateDBCluster(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	id := form.Get("DBClusterIdentifier")
	if id == "" { return xmlErr(service.ErrValidation("DBClusterIdentifier is required.")) }
	tags := parseTags(form)
	c, ok := store.CreateDBCluster(id, form.Get("Engine"), form.Get("EngineVersion"), form.Get("DatabaseName"), tags)
	if !ok { return xmlErr(service.NewAWSError("DBClusterAlreadyExistsFault", "Cluster already exists.", http.StatusBadRequest)) }
	resp := &xmlCreateDBClusterResp{Xmlns: neptuneXmlns, Meta: xmlResponseMetadata{reqID()}}; resp.Result.DBCluster = toXC(c)
	return xmlOK(resp)
}

type xmlDescribeDBClustersResp struct { XMLName xml.Name `xml:"DescribeDBClustersResponse"`; Xmlns string `xml:"xmlns,attr"`; Result struct{ DBClusters []xmlCluster `xml:"DBClusters>DBCluster"` } `xml:"DescribeDBClustersResult"`; Meta xmlResponseMetadata `xml:"ResponseMetadata"` }

func handleDescribeDBClusters(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx); fid := form.Get("DBClusterIdentifier")
	clusters := store.ListDBClusters(fid)
	if fid != "" && len(clusters) == 0 { return xmlErr(service.NewAWSError("DBClusterNotFoundFault", "Cluster "+fid+" not found.", http.StatusNotFound)) }
	xc := make([]xmlCluster, len(clusters)); for i, c := range clusters { xc[i] = toXC(c) }
	resp := &xmlDescribeDBClustersResp{Xmlns: neptuneXmlns, Meta: xmlResponseMetadata{reqID()}}; resp.Result.DBClusters = xc
	return xmlOK(resp)
}

type xmlDeleteDBClusterResp struct { XMLName xml.Name `xml:"DeleteDBClusterResponse"`; Xmlns string `xml:"xmlns,attr"`; Result struct{ DBCluster xmlCluster } `xml:"DeleteDBClusterResult"`; Meta xmlResponseMetadata `xml:"ResponseMetadata"` }

func handleDeleteDBCluster(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx); id := form.Get("DBClusterIdentifier")
	if id == "" { return xmlErr(service.ErrValidation("DBClusterIdentifier is required.")) }
	c, ok := store.DeleteDBCluster(id)
	if !ok { return xmlErr(service.NewAWSError("DBClusterNotFoundFault", "Cluster "+id+" not found.", http.StatusNotFound)) }
	resp := &xmlDeleteDBClusterResp{Xmlns: neptuneXmlns, Meta: xmlResponseMetadata{reqID()}}; resp.Result.DBCluster = toXC(c)
	return xmlOK(resp)
}

type xmlModifyDBClusterResp struct { XMLName xml.Name `xml:"ModifyDBClusterResponse"`; Xmlns string `xml:"xmlns,attr"`; Result struct{ DBCluster xmlCluster } `xml:"ModifyDBClusterResult"`; Meta xmlResponseMetadata `xml:"ResponseMetadata"` }

func handleModifyDBCluster(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx); id := form.Get("DBClusterIdentifier")
	c, ok := store.ModifyDBCluster(id, form.Get("EngineVersion"))
	if !ok { return xmlErr(service.NewAWSError("DBClusterNotFoundFault", "Cluster "+id+" not found.", http.StatusNotFound)) }
	resp := &xmlModifyDBClusterResp{Xmlns: neptuneXmlns, Meta: xmlResponseMetadata{reqID()}}; resp.Result.DBCluster = toXC(c)
	return xmlOK(resp)
}

// ---- Instance XML ----
type xmlEndpoint struct { Address string `xml:"Address"`; Port int `xml:"Port"` }
type xmlInstance struct {
	DBInstanceIdentifier string `xml:"DBInstanceIdentifier"`; DBInstanceArn string `xml:"DBInstanceArn"`
	DBInstanceClass string `xml:"DBInstanceClass"`; Engine string `xml:"Engine"`; EngineVersion string `xml:"EngineVersion"`
	DBInstanceStatus string `xml:"DBInstanceStatus"`; Endpoint xmlEndpoint `xml:"Endpoint"`
}
func toXI(i *DBInstance) xmlInstance {
	return xmlInstance{i.Identifier, i.ARN, i.Class, i.Engine, i.EngineVersion, i.Status, xmlEndpoint{i.Endpoint.Address, i.Endpoint.Port}}
}

type xmlCreateDBInstanceResp struct { XMLName xml.Name `xml:"CreateDBInstanceResponse"`; Xmlns string `xml:"xmlns,attr"`; Result struct{ DBInstance xmlInstance } `xml:"CreateDBInstanceResult"`; Meta xmlResponseMetadata `xml:"ResponseMetadata"` }

func handleCreateDBInstance(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	id := form.Get("DBInstanceIdentifier")
	if id == "" { return xmlErr(service.ErrValidation("DBInstanceIdentifier is required.")) }
	tags := parseTags(form)
	inst, ok := store.CreateDBInstance(id, form.Get("DBClusterIdentifier"), form.Get("DBInstanceClass"), form.Get("Engine"), form.Get("EngineVersion"), tags)
	if !ok { return xmlErr(service.NewAWSError("DBInstanceAlreadyExists", "Instance already exists.", http.StatusBadRequest)) }
	resp := &xmlCreateDBInstanceResp{Xmlns: neptuneXmlns, Meta: xmlResponseMetadata{reqID()}}; resp.Result.DBInstance = toXI(inst)
	return xmlOK(resp)
}

type xmlDescribeDBInstancesResp struct { XMLName xml.Name `xml:"DescribeDBInstancesResponse"`; Xmlns string `xml:"xmlns,attr"`; Result struct{ DBInstances []xmlInstance `xml:"DBInstances>DBInstance"` } `xml:"DescribeDBInstancesResult"`; Meta xmlResponseMetadata `xml:"ResponseMetadata"` }

func handleDescribeDBInstances(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx); fid := form.Get("DBInstanceIdentifier")
	instances := store.ListDBInstances(fid)
	if fid != "" && len(instances) == 0 { return xmlErr(service.NewAWSError("DBInstanceNotFound", "Instance "+fid+" not found.", http.StatusNotFound)) }
	xi := make([]xmlInstance, len(instances)); for i, inst := range instances { xi[i] = toXI(inst) }
	resp := &xmlDescribeDBInstancesResp{Xmlns: neptuneXmlns, Meta: xmlResponseMetadata{reqID()}}; resp.Result.DBInstances = xi
	return xmlOK(resp)
}

type xmlDeleteDBInstanceResp struct { XMLName xml.Name `xml:"DeleteDBInstanceResponse"`; Xmlns string `xml:"xmlns,attr"`; Result struct{ DBInstance xmlInstance } `xml:"DeleteDBInstanceResult"`; Meta xmlResponseMetadata `xml:"ResponseMetadata"` }

func handleDeleteDBInstance(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx); id := form.Get("DBInstanceIdentifier")
	inst, ok := store.DeleteDBInstance(id)
	if !ok { return xmlErr(service.NewAWSError("DBInstanceNotFound", "Instance "+id+" not found.", http.StatusNotFound)) }
	resp := &xmlDeleteDBInstanceResp{Xmlns: neptuneXmlns, Meta: xmlResponseMetadata{reqID()}}; resp.Result.DBInstance = toXI(inst)
	return xmlOK(resp)
}

type xmlModifyDBInstanceResp struct { XMLName xml.Name `xml:"ModifyDBInstanceResponse"`; Xmlns string `xml:"xmlns,attr"`; Result struct{ DBInstance xmlInstance } `xml:"ModifyDBInstanceResult"`; Meta xmlResponseMetadata `xml:"ResponseMetadata"` }

func handleModifyDBInstance(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx); id := form.Get("DBInstanceIdentifier")
	inst, ok := store.ModifyDBInstance(id, form.Get("DBInstanceClass"))
	if !ok { return xmlErr(service.NewAWSError("DBInstanceNotFound", "Instance "+id+" not found.", http.StatusNotFound)) }
	resp := &xmlModifyDBInstanceResp{Xmlns: neptuneXmlns, Meta: xmlResponseMetadata{reqID()}}; resp.Result.DBInstance = toXI(inst)
	return xmlOK(resp)
}

// ---- Snapshot XML ----
type xmlSnapshot struct { DBClusterSnapshotIdentifier string `xml:"DBClusterSnapshotIdentifier"`; DBClusterIdentifier string `xml:"DBClusterIdentifier"`; Status string `xml:"Status"`; Engine string `xml:"Engine"` }
func toXS(s *DBClusterSnapshot) xmlSnapshot { return xmlSnapshot{s.Identifier, s.ClusterIdentifier, s.Status, s.Engine} }

type xmlCreateSnapshotResp struct { XMLName xml.Name `xml:"CreateDBClusterSnapshotResponse"`; Xmlns string `xml:"xmlns,attr"`; Result struct{ DBClusterSnapshot xmlSnapshot } `xml:"CreateDBClusterSnapshotResult"`; Meta xmlResponseMetadata `xml:"ResponseMetadata"` }

func handleCreateDBClusterSnapshot(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx); snapID := form.Get("DBClusterSnapshotIdentifier"); clusterID := form.Get("DBClusterIdentifier")
	tags := parseTags(form)
	snap, ok := store.CreateDBClusterSnapshot(snapID, clusterID, tags)
	if !ok { return xmlErr(service.NewAWSError("DBClusterSnapshotAlreadyExistsFault", "Snapshot already exists or cluster not found.", http.StatusBadRequest)) }
	resp := &xmlCreateSnapshotResp{Xmlns: neptuneXmlns, Meta: xmlResponseMetadata{reqID()}}; resp.Result.DBClusterSnapshot = toXS(snap)
	return xmlOK(resp)
}

type xmlDescribeSnapshotsResp struct { XMLName xml.Name `xml:"DescribeDBClusterSnapshotsResponse"`; Xmlns string `xml:"xmlns,attr"`; Result struct{ DBClusterSnapshots []xmlSnapshot `xml:"DBClusterSnapshots>DBClusterSnapshot"` } `xml:"DescribeDBClusterSnapshotsResult"`; Meta xmlResponseMetadata `xml:"ResponseMetadata"` }

func handleDescribeDBClusterSnapshots(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	snaps := store.ListDBClusterSnapshots(form.Get("DBClusterIdentifier"), form.Get("DBClusterSnapshotIdentifier"))
	xs := make([]xmlSnapshot, len(snaps)); for i, s := range snaps { xs[i] = toXS(s) }
	resp := &xmlDescribeSnapshotsResp{Xmlns: neptuneXmlns, Meta: xmlResponseMetadata{reqID()}}; resp.Result.DBClusterSnapshots = xs
	return xmlOK(resp)
}

type xmlDeleteSnapshotResp struct { XMLName xml.Name `xml:"DeleteDBClusterSnapshotResponse"`; Xmlns string `xml:"xmlns,attr"`; Meta xmlResponseMetadata `xml:"ResponseMetadata"` }

func handleDeleteDBClusterSnapshot(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx); id := form.Get("DBClusterSnapshotIdentifier")
	if !store.DeleteDBClusterSnapshot(id) { return xmlErr(service.NewAWSError("DBClusterSnapshotNotFoundFault", "Snapshot "+id+" not found.", http.StatusNotFound)) }
	return xmlOK(&xmlDeleteSnapshotResp{Xmlns: neptuneXmlns, Meta: xmlResponseMetadata{reqID()}})
}

// ---- SubnetGroup XML ----
type xmlSubnetGroup struct { DBSubnetGroupName string `xml:"DBSubnetGroupName"`; DBSubnetGroupDescription string `xml:"DBSubnetGroupDescription"`; SubnetGroupStatus string `xml:"SubnetGroupStatus"` }
func toXSG(sg *DBSubnetGroup) xmlSubnetGroup { return xmlSubnetGroup{sg.Name, sg.Description, sg.Status} }

type xmlCreateSubnetGroupResp struct { XMLName xml.Name `xml:"CreateDBSubnetGroupResponse"`; Xmlns string `xml:"xmlns,attr"`; Result struct{ DBSubnetGroup xmlSubnetGroup } `xml:"CreateDBSubnetGroupResult"`; Meta xmlResponseMetadata `xml:"ResponseMetadata"` }

func handleCreateDBSubnetGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx); name := form.Get("DBSubnetGroupName"); desc := form.Get("DBSubnetGroupDescription")
	tags := parseTags(form); subnetIDs := parseSubnetIDs(form)
	sg, ok := store.CreateDBSubnetGroup(name, desc, subnetIDs, tags)
	if !ok { return xmlErr(service.ErrAlreadyExists("DBSubnetGroup", name)) }
	resp := &xmlCreateSubnetGroupResp{Xmlns: neptuneXmlns, Meta: xmlResponseMetadata{reqID()}}; resp.Result.DBSubnetGroup = toXSG(sg)
	return xmlOK(resp)
}

type xmlDescribeSubnetGroupsResp struct { XMLName xml.Name `xml:"DescribeDBSubnetGroupsResponse"`; Xmlns string `xml:"xmlns,attr"`; Result struct{ DBSubnetGroups []xmlSubnetGroup `xml:"DBSubnetGroups>DBSubnetGroup"` } `xml:"DescribeDBSubnetGroupsResult"`; Meta xmlResponseMetadata `xml:"ResponseMetadata"` }

func handleDescribeDBSubnetGroups(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx); groups := store.ListDBSubnetGroups(form.Get("DBSubnetGroupName"))
	xg := make([]xmlSubnetGroup, len(groups)); for i, sg := range groups { xg[i] = toXSG(sg) }
	resp := &xmlDescribeSubnetGroupsResp{Xmlns: neptuneXmlns, Meta: xmlResponseMetadata{reqID()}}; resp.Result.DBSubnetGroups = xg
	return xmlOK(resp)
}

type xmlDeleteSubnetGroupResp struct { XMLName xml.Name `xml:"DeleteDBSubnetGroupResponse"`; Xmlns string `xml:"xmlns,attr"`; Meta xmlResponseMetadata `xml:"ResponseMetadata"` }

func handleDeleteDBSubnetGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx); name := form.Get("DBSubnetGroupName")
	if !store.DeleteDBSubnetGroup(name) { return xmlErr(service.NewAWSError("DBSubnetGroupNotFoundFault", "Subnet group "+name+" not found.", http.StatusNotFound)) }
	return xmlOK(&xmlDeleteSubnetGroupResp{Xmlns: neptuneXmlns, Meta: xmlResponseMetadata{reqID()}})
}

// ---- ParameterGroup XML ----
type xmlParamGroup struct { DBClusterParameterGroupName string `xml:"DBClusterParameterGroupName"`; DBParameterGroupFamily string `xml:"DBParameterGroupFamily"`; Description string `xml:"Description"` }

type xmlCreateParamGroupResp struct { XMLName xml.Name `xml:"CreateDBClusterParameterGroupResponse"`; Xmlns string `xml:"xmlns,attr"`; Result struct{ DBClusterParameterGroup xmlParamGroup } `xml:"CreateDBClusterParameterGroupResult"`; Meta xmlResponseMetadata `xml:"ResponseMetadata"` }

func handleCreateDBClusterParameterGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx); name := form.Get("DBClusterParameterGroupName"); family := form.Get("DBParameterGroupFamily"); desc := form.Get("Description")
	tags := parseTags(form)
	pg, ok := store.CreateDBClusterParameterGroup(name, family, desc, tags)
	if !ok { return xmlErr(service.ErrAlreadyExists("DBClusterParameterGroup", name)) }
	resp := &xmlCreateParamGroupResp{Xmlns: neptuneXmlns, Meta: xmlResponseMetadata{reqID()}}
	resp.Result.DBClusterParameterGroup = xmlParamGroup{pg.Name, pg.Family, pg.Description}
	return xmlOK(resp)
}

type xmlDescribeParamGroupsResp struct { XMLName xml.Name `xml:"DescribeDBClusterParameterGroupsResponse"`; Xmlns string `xml:"xmlns,attr"`; Result struct{ DBClusterParameterGroups []xmlParamGroup `xml:"DBClusterParameterGroups>DBClusterParameterGroup"` } `xml:"DescribeDBClusterParameterGroupsResult"`; Meta xmlResponseMetadata `xml:"ResponseMetadata"` }

func handleDescribeDBClusterParameterGroups(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx); groups := store.ListDBClusterParameterGroups(form.Get("DBClusterParameterGroupName"))
	xg := make([]xmlParamGroup, len(groups)); for i, pg := range groups { xg[i] = xmlParamGroup{pg.Name, pg.Family, pg.Description} }
	resp := &xmlDescribeParamGroupsResp{Xmlns: neptuneXmlns, Meta: xmlResponseMetadata{reqID()}}; resp.Result.DBClusterParameterGroups = xg
	return xmlOK(resp)
}

type xmlDeleteParamGroupResp struct { XMLName xml.Name `xml:"DeleteDBClusterParameterGroupResponse"`; Xmlns string `xml:"xmlns,attr"`; Meta xmlResponseMetadata `xml:"ResponseMetadata"` }

func handleDeleteDBClusterParameterGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx); name := form.Get("DBClusterParameterGroupName")
	if !store.DeleteDBClusterParameterGroup(name) { return xmlErr(service.NewAWSError("DBParameterGroupNotFoundFault", "Parameter group "+name+" not found.", http.StatusNotFound)) }
	return xmlOK(&xmlDeleteParamGroupResp{Xmlns: neptuneXmlns, Meta: xmlResponseMetadata{reqID()}})
}

// ---- Tag handlers ----
type xmlAddTagsResp struct { XMLName xml.Name `xml:"AddTagsToResourceResponse"`; Xmlns string `xml:"xmlns,attr"`; Meta xmlResponseMetadata `xml:"ResponseMetadata"` }

func handleAddTagsToResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx); arn := form.Get("ResourceName"); tags := parseTags(form)
	if !store.AddTags(arn, tags) { return xmlErr(service.ErrNotFound("Resource", arn)) }
	return xmlOK(&xmlAddTagsResp{Xmlns: neptuneXmlns, Meta: xmlResponseMetadata{reqID()}})
}

type xmlRemoveTagsResp struct { XMLName xml.Name `xml:"RemoveTagsFromResourceResponse"`; Xmlns string `xml:"xmlns,attr"`; Meta xmlResponseMetadata `xml:"ResponseMetadata"` }

func handleRemoveTagsFromResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx); arn := form.Get("ResourceName"); keys := parseTagKeys(form)
	if !store.RemoveTags(arn, keys) { return xmlErr(service.ErrNotFound("Resource", arn)) }
	return xmlOK(&xmlRemoveTagsResp{Xmlns: neptuneXmlns, Meta: xmlResponseMetadata{reqID()}})
}

type xmlListTagsResp struct { XMLName xml.Name `xml:"ListTagsForResourceResponse"`; Xmlns string `xml:"xmlns,attr"`; Result struct{ TagList []xmlTag `xml:"TagList>Tag"` } `xml:"ListTagsForResourceResult"`; Meta xmlResponseMetadata `xml:"ResponseMetadata"` }

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx); arn := form.Get("ResourceName")
	tags, ok := store.ListTags(arn)
	if !ok { return xmlErr(service.ErrNotFound("Resource", arn)) }
	xt := make([]xmlTag, 0, len(tags)); for k, v := range tags { xt = append(xt, xmlTag{k, v}) }
	resp := &xmlListTagsResp{Xmlns: neptuneXmlns, Meta: xmlResponseMetadata{reqID()}}; resp.Result.TagList = xt
	return xmlOK(resp)
}
