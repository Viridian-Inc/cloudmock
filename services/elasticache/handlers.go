package elasticache

import (
	"crypto/rand"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/neureaux/cloudmock/pkg/service"
)

const ecXmlns = "http://elasticache.amazonaws.com/doc/2015-02-02/"

// ---- shared XML types ----

type xmlResponseMetadata struct {
	RequestID string `xml:"RequestId"`
}

type xmlTag struct {
	Key   string `xml:"Key"`
	Value string `xml:"Value"`
}

// ---- CacheCluster XML types ----

type xmlCacheCluster struct {
	CacheClusterId      string       `xml:"CacheClusterId"`
	ARN                 string       `xml:"ARN"`
	Engine              string       `xml:"Engine"`
	EngineVersion       string       `xml:"EngineVersion"`
	CacheNodeType       string       `xml:"CacheNodeType"`
	NumCacheNodes       int          `xml:"NumCacheNodes"`
	CacheClusterStatus  string       `xml:"CacheClusterStatus"`
	PreferredAvailabilityZone string `xml:"PreferredAvailabilityZone,omitempty"`
	CacheSubnetGroupName string     `xml:"CacheSubnetGroupName,omitempty"`
	CacheParameterGroupName string  `xml:"CacheParameterGroup>CacheParameterGroupName,omitempty"`
	ConfigurationEndpoint *xmlEndpoint `xml:"ConfigurationEndpoint,omitempty"`
}

type xmlEndpoint struct {
	Address string `xml:"Address"`
	Port    int    `xml:"Port"`
}

func toXMLCacheCluster(cc *CacheCluster) xmlCacheCluster {
	x := xmlCacheCluster{
		CacheClusterId:        cc.ID,
		ARN:                   cc.ARN,
		Engine:                cc.Engine,
		EngineVersion:         cc.EngineVersion,
		CacheNodeType:         cc.CacheNodeType,
		NumCacheNodes:         cc.NumCacheNodes,
		CacheClusterStatus:    cc.Status,
		PreferredAvailabilityZone: cc.PreferredAZ,
		CacheSubnetGroupName:  cc.CacheSubnetGroupName,
		CacheParameterGroupName: cc.CacheParameterGroupName,
	}
	if cc.Endpoint != nil {
		x.ConfigurationEndpoint = &xmlEndpoint{Address: cc.Endpoint.Address, Port: cc.Endpoint.Port}
	}
	return x
}

// ---- CreateCacheCluster ----

type xmlCreateCacheClusterResponse struct {
	XMLName xml.Name                   `xml:"CreateCacheClusterResponse"`
	Xmlns   string                     `xml:"xmlns,attr"`
	Result  xmlCreateCacheClusterResult `xml:"CreateCacheClusterResult"`
	Meta    xmlResponseMetadata        `xml:"ResponseMetadata"`
}

type xmlCreateCacheClusterResult struct {
	CacheCluster xmlCacheCluster `xml:"CacheCluster"`
}

func handleCreateCacheCluster(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	id := form.Get("CacheClusterId")
	if id == "" {
		return xmlErr(service.ErrValidation("CacheClusterId is required."))
	}
	engine := form.Get("Engine")
	engineVersion := form.Get("EngineVersion")
	nodeType := form.Get("CacheNodeType")
	az := form.Get("PreferredAvailabilityZone")
	subnetGroup := form.Get("CacheSubnetGroupName")
	paramGroup := form.Get("CacheParameterGroupName")
	numNodes := 1
	if v := form.Get("NumCacheNodes"); v != "" {
		numNodes, _ = strconv.Atoi(v)
	}
	sgIDs := parseMemberList(form, "SecurityGroupIds")

	cc, ok := store.CreateCacheCluster(id, engine, engineVersion, nodeType, az, subnetGroup, paramGroup, numNodes, sgIDs)
	if !ok {
		return xmlErr(service.NewAWSError("CacheClusterAlreadyExists",
			"Cache cluster "+id+" already exists.", http.StatusBadRequest))
	}

	return xmlOK(&xmlCreateCacheClusterResponse{
		Xmlns:  ecXmlns,
		Result: xmlCreateCacheClusterResult{CacheCluster: toXMLCacheCluster(cc)},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- DescribeCacheClusters ----

type xmlDescribeCacheClustersResponse struct {
	XMLName xml.Name                      `xml:"DescribeCacheClustersResponse"`
	Xmlns   string                        `xml:"xmlns,attr"`
	Result  xmlDescribeCacheClustersResult `xml:"DescribeCacheClustersResult"`
	Meta    xmlResponseMetadata           `xml:"ResponseMetadata"`
}

type xmlDescribeCacheClustersResult struct {
	CacheClusters []xmlCacheCluster `xml:"CacheClusters>CacheCluster"`
}

func handleDescribeCacheClusters(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	filterID := form.Get("CacheClusterId")

	clusters := store.ListCacheClusters(filterID)

	if filterID != "" && len(clusters) == 0 {
		return xmlErr(service.NewAWSError("CacheClusterNotFound",
			"CacheCluster "+filterID+" not found.", http.StatusNotFound))
	}

	xmlClusters := make([]xmlCacheCluster, 0, len(clusters))
	for _, cc := range clusters {
		xmlClusters = append(xmlClusters, toXMLCacheCluster(cc))
	}

	return xmlOK(&xmlDescribeCacheClustersResponse{
		Xmlns:  ecXmlns,
		Result: xmlDescribeCacheClustersResult{CacheClusters: xmlClusters},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- ModifyCacheCluster ----

type xmlModifyCacheClusterResponse struct {
	XMLName xml.Name                   `xml:"ModifyCacheClusterResponse"`
	Xmlns   string                     `xml:"xmlns,attr"`
	Result  xmlModifyCacheClusterResult `xml:"ModifyCacheClusterResult"`
	Meta    xmlResponseMetadata        `xml:"ResponseMetadata"`
}

type xmlModifyCacheClusterResult struct {
	CacheCluster xmlCacheCluster `xml:"CacheCluster"`
}

func handleModifyCacheCluster(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	id := form.Get("CacheClusterId")
	if id == "" {
		return xmlErr(service.ErrValidation("CacheClusterId is required."))
	}
	nodeType := form.Get("CacheNodeType")
	engineVersion := form.Get("EngineVersion")
	numNodes := 0
	if v := form.Get("NumCacheNodes"); v != "" {
		numNodes, _ = strconv.Atoi(v)
	}

	cc, ok := store.ModifyCacheCluster(id, nodeType, engineVersion, numNodes)
	if !ok {
		return xmlErr(service.NewAWSError("CacheClusterNotFound",
			"CacheCluster "+id+" not found.", http.StatusNotFound))
	}

	return xmlOK(&xmlModifyCacheClusterResponse{
		Xmlns:  ecXmlns,
		Result: xmlModifyCacheClusterResult{CacheCluster: toXMLCacheCluster(cc)},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- DeleteCacheCluster ----

type xmlDeleteCacheClusterResponse struct {
	XMLName xml.Name                   `xml:"DeleteCacheClusterResponse"`
	Xmlns   string                     `xml:"xmlns,attr"`
	Result  xmlDeleteCacheClusterResult `xml:"DeleteCacheClusterResult"`
	Meta    xmlResponseMetadata        `xml:"ResponseMetadata"`
}

type xmlDeleteCacheClusterResult struct {
	CacheCluster xmlCacheCluster `xml:"CacheCluster"`
}

func handleDeleteCacheCluster(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	id := form.Get("CacheClusterId")
	if id == "" {
		return xmlErr(service.ErrValidation("CacheClusterId is required."))
	}

	cc, ok := store.DeleteCacheCluster(id)
	if !ok {
		return xmlErr(service.NewAWSError("CacheClusterNotFound",
			"CacheCluster "+id+" not found.", http.StatusNotFound))
	}

	return xmlOK(&xmlDeleteCacheClusterResponse{
		Xmlns:  ecXmlns,
		Result: xmlDeleteCacheClusterResult{CacheCluster: toXMLCacheCluster(cc)},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- ReplicationGroup XML types ----

type xmlReplicationGroup struct {
	ReplicationGroupId  string       `xml:"ReplicationGroupId"`
	ARN                 string       `xml:"ARN"`
	Description         string       `xml:"Description"`
	Status              string       `xml:"Status"`
	MemberClusters      []string     `xml:"MemberClusters>ClusterId"`
	AutomaticFailover   string       `xml:"AutomaticFailover"`
	MultiAZ             string       `xml:"MultiAZ"`
	NodeGroups          []xmlNodeGroupMember `xml:"NodeGroups>NodeGroup,omitempty"`
}

type xmlNodeGroupMember struct {
	PrimaryEndpoint *xmlEndpoint `xml:"PrimaryEndpoint,omitempty"`
	ReaderEndpoint  *xmlEndpoint `xml:"ReaderEndpoint,omitempty"`
}

func toXMLReplicationGroup(rg *ReplicationGroup) xmlReplicationGroup {
	multiAZ := "disabled"
	if rg.MultiAZEnabled {
		multiAZ = "enabled"
	}
	x := xmlReplicationGroup{
		ReplicationGroupId: rg.ID,
		ARN:                rg.ARN,
		Description:        rg.Description,
		Status:             rg.Status,
		MemberClusters:     rg.MemberClusters,
		AutomaticFailover:  rg.AutomaticFailover,
		MultiAZ:            multiAZ,
	}
	if rg.PrimaryEndpoint != nil || rg.ReaderEndpoint != nil {
		ng := xmlNodeGroupMember{}
		if rg.PrimaryEndpoint != nil {
			ng.PrimaryEndpoint = &xmlEndpoint{Address: rg.PrimaryEndpoint.Address, Port: rg.PrimaryEndpoint.Port}
		}
		if rg.ReaderEndpoint != nil {
			ng.ReaderEndpoint = &xmlEndpoint{Address: rg.ReaderEndpoint.Address, Port: rg.ReaderEndpoint.Port}
		}
		x.NodeGroups = []xmlNodeGroupMember{ng}
	}
	return x
}

// ---- CreateReplicationGroup ----

type xmlCreateReplicationGroupResponse struct {
	XMLName xml.Name                       `xml:"CreateReplicationGroupResponse"`
	Xmlns   string                         `xml:"xmlns,attr"`
	Result  xmlCreateReplicationGroupResult `xml:"CreateReplicationGroupResult"`
	Meta    xmlResponseMetadata            `xml:"ResponseMetadata"`
}

type xmlCreateReplicationGroupResult struct {
	ReplicationGroup xmlReplicationGroup `xml:"ReplicationGroup"`
}

func handleCreateReplicationGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	id := form.Get("ReplicationGroupId")
	if id == "" {
		return xmlErr(service.ErrValidation("ReplicationGroupId is required."))
	}
	description := form.Get("ReplicationGroupDescription")
	engine := form.Get("Engine")
	engineVersion := form.Get("EngineVersion")
	nodeType := form.Get("CacheNodeType")
	subnetGroup := form.Get("CacheSubnetGroupName")
	failover := form.Get("AutomaticFailoverEnabled")
	if failover == "true" {
		failover = "enabled"
	} else if failover != "" {
		failover = "disabled"
	}
	numClusters := 1
	if v := form.Get("NumCacheClusters"); v != "" {
		numClusters, _ = strconv.Atoi(v)
	}
	multiAZ := form.Get("MultiAZEnabled") == "true"
	port := 0
	if v := form.Get("Port"); v != "" {
		port, _ = strconv.Atoi(v)
	}

	rg, ok := store.CreateReplicationGroup(id, description, engine, engineVersion, nodeType, subnetGroup, failover, numClusters, multiAZ, port)
	if !ok {
		return xmlErr(service.NewAWSError("ReplicationGroupAlreadyExistsFault",
			"ReplicationGroup "+id+" already exists.", http.StatusBadRequest))
	}

	return xmlOK(&xmlCreateReplicationGroupResponse{
		Xmlns:  ecXmlns,
		Result: xmlCreateReplicationGroupResult{ReplicationGroup: toXMLReplicationGroup(rg)},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- DescribeReplicationGroups ----

type xmlDescribeReplicationGroupsResponse struct {
	XMLName xml.Name                          `xml:"DescribeReplicationGroupsResponse"`
	Xmlns   string                            `xml:"xmlns,attr"`
	Result  xmlDescribeReplicationGroupsResult `xml:"DescribeReplicationGroupsResult"`
	Meta    xmlResponseMetadata               `xml:"ResponseMetadata"`
}

type xmlDescribeReplicationGroupsResult struct {
	ReplicationGroups []xmlReplicationGroup `xml:"ReplicationGroups>ReplicationGroup"`
}

func handleDescribeReplicationGroups(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	filterID := form.Get("ReplicationGroupId")

	rgs := store.ListReplicationGroups(filterID)

	if filterID != "" && len(rgs) == 0 {
		return xmlErr(service.NewAWSError("ReplicationGroupNotFoundFault",
			"ReplicationGroup "+filterID+" not found.", http.StatusNotFound))
	}

	xmlRGs := make([]xmlReplicationGroup, 0, len(rgs))
	for _, rg := range rgs {
		xmlRGs = append(xmlRGs, toXMLReplicationGroup(rg))
	}

	return xmlOK(&xmlDescribeReplicationGroupsResponse{
		Xmlns:  ecXmlns,
		Result: xmlDescribeReplicationGroupsResult{ReplicationGroups: xmlRGs},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- ModifyReplicationGroup ----

type xmlModifyReplicationGroupResponse struct {
	XMLName xml.Name                       `xml:"ModifyReplicationGroupResponse"`
	Xmlns   string                         `xml:"xmlns,attr"`
	Result  xmlModifyReplicationGroupResult `xml:"ModifyReplicationGroupResult"`
	Meta    xmlResponseMetadata            `xml:"ResponseMetadata"`
}

type xmlModifyReplicationGroupResult struct {
	ReplicationGroup xmlReplicationGroup `xml:"ReplicationGroup"`
}

func handleModifyReplicationGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	id := form.Get("ReplicationGroupId")
	if id == "" {
		return xmlErr(service.ErrValidation("ReplicationGroupId is required."))
	}
	description := form.Get("ReplicationGroupDescription")
	nodeType := form.Get("CacheNodeType")
	engineVersion := form.Get("EngineVersion")
	failover := form.Get("AutomaticFailoverEnabled")
	if failover == "true" {
		failover = "enabled"
	} else if failover == "false" {
		failover = "disabled"
	}

	rg, ok := store.ModifyReplicationGroup(id, description, nodeType, engineVersion, failover)
	if !ok {
		return xmlErr(service.NewAWSError("ReplicationGroupNotFoundFault",
			"ReplicationGroup "+id+" not found.", http.StatusNotFound))
	}

	return xmlOK(&xmlModifyReplicationGroupResponse{
		Xmlns:  ecXmlns,
		Result: xmlModifyReplicationGroupResult{ReplicationGroup: toXMLReplicationGroup(rg)},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- DeleteReplicationGroup ----

type xmlDeleteReplicationGroupResponse struct {
	XMLName xml.Name                       `xml:"DeleteReplicationGroupResponse"`
	Xmlns   string                         `xml:"xmlns,attr"`
	Result  xmlDeleteReplicationGroupResult `xml:"DeleteReplicationGroupResult"`
	Meta    xmlResponseMetadata            `xml:"ResponseMetadata"`
}

type xmlDeleteReplicationGroupResult struct {
	ReplicationGroup xmlReplicationGroup `xml:"ReplicationGroup"`
}

func handleDeleteReplicationGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	id := form.Get("ReplicationGroupId")
	if id == "" {
		return xmlErr(service.ErrValidation("ReplicationGroupId is required."))
	}

	rg, ok := store.DeleteReplicationGroup(id)
	if !ok {
		return xmlErr(service.NewAWSError("ReplicationGroupNotFoundFault",
			"ReplicationGroup "+id+" not found.", http.StatusNotFound))
	}

	return xmlOK(&xmlDeleteReplicationGroupResponse{
		Xmlns:  ecXmlns,
		Result: xmlDeleteReplicationGroupResult{ReplicationGroup: toXMLReplicationGroup(rg)},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- CacheSubnetGroup XML types ----

type xmlCacheSubnetGroup struct {
	CacheSubnetGroupName        string   `xml:"CacheSubnetGroupName"`
	ARN                         string   `xml:"ARN"`
	CacheSubnetGroupDescription string   `xml:"CacheSubnetGroupDescription"`
	VpcId                       string   `xml:"VpcId"`
	Subnets                     []string `xml:"Subnets>Subnet>SubnetIdentifier"`
}

func toXMLCacheSubnetGroup(sg *CacheSubnetGroup) xmlCacheSubnetGroup {
	return xmlCacheSubnetGroup{
		CacheSubnetGroupName:        sg.Name,
		ARN:                         sg.ARN,
		CacheSubnetGroupDescription: sg.Description,
		VpcId:                       sg.VpcID,
		Subnets:                     sg.SubnetIDs,
	}
}

// ---- CreateCacheSubnetGroup ----

type xmlCreateCacheSubnetGroupResponse struct {
	XMLName xml.Name                       `xml:"CreateCacheSubnetGroupResponse"`
	Xmlns   string                         `xml:"xmlns,attr"`
	Result  xmlCreateCacheSubnetGroupResult `xml:"CreateCacheSubnetGroupResult"`
	Meta    xmlResponseMetadata            `xml:"ResponseMetadata"`
}

type xmlCreateCacheSubnetGroupResult struct {
	CacheSubnetGroup xmlCacheSubnetGroup `xml:"CacheSubnetGroup"`
}

func handleCreateCacheSubnetGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	name := form.Get("CacheSubnetGroupName")
	if name == "" {
		return xmlErr(service.ErrValidation("CacheSubnetGroupName is required."))
	}
	description := form.Get("CacheSubnetGroupDescription")
	vpcID := form.Get("VpcId")
	subnetIDs := parseMemberList(form, "SubnetIds")

	sg, ok := store.CreateCacheSubnetGroup(name, description, vpcID, subnetIDs)
	if !ok {
		return xmlErr(service.NewAWSError("CacheSubnetGroupAlreadyExists",
			"Cache subnet group "+name+" already exists.", http.StatusBadRequest))
	}

	return xmlOK(&xmlCreateCacheSubnetGroupResponse{
		Xmlns:  ecXmlns,
		Result: xmlCreateCacheSubnetGroupResult{CacheSubnetGroup: toXMLCacheSubnetGroup(sg)},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- DescribeCacheSubnetGroups ----

type xmlDescribeCacheSubnetGroupsResponse struct {
	XMLName xml.Name                          `xml:"DescribeCacheSubnetGroupsResponse"`
	Xmlns   string                            `xml:"xmlns,attr"`
	Result  xmlDescribeCacheSubnetGroupsResult `xml:"DescribeCacheSubnetGroupsResult"`
	Meta    xmlResponseMetadata               `xml:"ResponseMetadata"`
}

type xmlDescribeCacheSubnetGroupsResult struct {
	CacheSubnetGroups []xmlCacheSubnetGroup `xml:"CacheSubnetGroups>CacheSubnetGroup"`
}

func handleDescribeCacheSubnetGroups(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	filterName := form.Get("CacheSubnetGroupName")

	groups := store.ListCacheSubnetGroups(filterName)

	if filterName != "" && len(groups) == 0 {
		return xmlErr(service.NewAWSError("CacheSubnetGroupNotFoundFault",
			"CacheSubnetGroup "+filterName+" not found.", http.StatusNotFound))
	}

	xmlGroups := make([]xmlCacheSubnetGroup, 0, len(groups))
	for _, sg := range groups {
		xmlGroups = append(xmlGroups, toXMLCacheSubnetGroup(sg))
	}

	return xmlOK(&xmlDescribeCacheSubnetGroupsResponse{
		Xmlns:  ecXmlns,
		Result: xmlDescribeCacheSubnetGroupsResult{CacheSubnetGroups: xmlGroups},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- DeleteCacheSubnetGroup ----

type xmlDeleteCacheSubnetGroupResponse struct {
	XMLName xml.Name            `xml:"DeleteCacheSubnetGroupResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleDeleteCacheSubnetGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	name := form.Get("CacheSubnetGroupName")
	if name == "" {
		return xmlErr(service.ErrValidation("CacheSubnetGroupName is required."))
	}

	if !store.DeleteCacheSubnetGroup(name) {
		return xmlErr(service.NewAWSError("CacheSubnetGroupNotFoundFault",
			"CacheSubnetGroup "+name+" not found.", http.StatusNotFound))
	}

	return xmlOK(&xmlDeleteCacheSubnetGroupResponse{
		Xmlns: ecXmlns,
		Meta:  xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- CacheParameterGroup XML types ----

type xmlCacheParameterGroup struct {
	CacheParameterGroupName   string `xml:"CacheParameterGroupName"`
	ARN                       string `xml:"ARN"`
	CacheParameterGroupFamily string `xml:"CacheParameterGroupFamily"`
	Description               string `xml:"Description"`
}

func toXMLCacheParameterGroup(pg *CacheParameterGroup) xmlCacheParameterGroup {
	return xmlCacheParameterGroup{
		CacheParameterGroupName:   pg.Name,
		ARN:                       pg.ARN,
		CacheParameterGroupFamily: pg.Family,
		Description:               pg.Description,
	}
}

// ---- CreateCacheParameterGroup ----

type xmlCreateCacheParameterGroupResponse struct {
	XMLName xml.Name                          `xml:"CreateCacheParameterGroupResponse"`
	Xmlns   string                            `xml:"xmlns,attr"`
	Result  xmlCreateCacheParameterGroupResult `xml:"CreateCacheParameterGroupResult"`
	Meta    xmlResponseMetadata               `xml:"ResponseMetadata"`
}

type xmlCreateCacheParameterGroupResult struct {
	CacheParameterGroup xmlCacheParameterGroup `xml:"CacheParameterGroup"`
}

func handleCreateCacheParameterGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	name := form.Get("CacheParameterGroupName")
	if name == "" {
		return xmlErr(service.ErrValidation("CacheParameterGroupName is required."))
	}
	family := form.Get("CacheParameterGroupFamily")
	if family == "" {
		return xmlErr(service.ErrValidation("CacheParameterGroupFamily is required."))
	}
	description := form.Get("Description")

	pg, ok := store.CreateCacheParameterGroup(name, family, description)
	if !ok {
		return xmlErr(service.NewAWSError("CacheParameterGroupAlreadyExists",
			"CacheParameterGroup "+name+" already exists.", http.StatusBadRequest))
	}

	return xmlOK(&xmlCreateCacheParameterGroupResponse{
		Xmlns:  ecXmlns,
		Result: xmlCreateCacheParameterGroupResult{CacheParameterGroup: toXMLCacheParameterGroup(pg)},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- DescribeCacheParameterGroups ----

type xmlDescribeCacheParameterGroupsResponse struct {
	XMLName xml.Name                             `xml:"DescribeCacheParameterGroupsResponse"`
	Xmlns   string                               `xml:"xmlns,attr"`
	Result  xmlDescribeCacheParameterGroupsResult `xml:"DescribeCacheParameterGroupsResult"`
	Meta    xmlResponseMetadata                  `xml:"ResponseMetadata"`
}

type xmlDescribeCacheParameterGroupsResult struct {
	CacheParameterGroups []xmlCacheParameterGroup `xml:"CacheParameterGroups>CacheParameterGroup"`
}

func handleDescribeCacheParameterGroups(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	filterName := form.Get("CacheParameterGroupName")

	groups := store.ListCacheParameterGroups(filterName)

	if filterName != "" && len(groups) == 0 {
		return xmlErr(service.NewAWSError("CacheParameterGroupNotFound",
			"CacheParameterGroup "+filterName+" not found.", http.StatusNotFound))
	}

	xmlGroups := make([]xmlCacheParameterGroup, 0, len(groups))
	for _, pg := range groups {
		xmlGroups = append(xmlGroups, toXMLCacheParameterGroup(pg))
	}

	return xmlOK(&xmlDescribeCacheParameterGroupsResponse{
		Xmlns:  ecXmlns,
		Result: xmlDescribeCacheParameterGroupsResult{CacheParameterGroups: xmlGroups},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- DeleteCacheParameterGroup ----

type xmlDeleteCacheParameterGroupResponse struct {
	XMLName xml.Name            `xml:"DeleteCacheParameterGroupResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleDeleteCacheParameterGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	name := form.Get("CacheParameterGroupName")
	if name == "" {
		return xmlErr(service.ErrValidation("CacheParameterGroupName is required."))
	}

	if !store.DeleteCacheParameterGroup(name) {
		return xmlErr(service.NewAWSError("CacheParameterGroupNotFound",
			"CacheParameterGroup "+name+" not found.", http.StatusNotFound))
	}

	return xmlOK(&xmlDeleteCacheParameterGroupResponse{
		Xmlns: ecXmlns,
		Meta:  xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- AddTagsToResource ----

type xmlAddTagsToResourceResponse struct {
	XMLName xml.Name                  `xml:"AddTagsToResourceResponse"`
	Xmlns   string                    `xml:"xmlns,attr"`
	Result  xmlAddTagsToResourceResult `xml:"AddTagsToResourceResult"`
	Meta    xmlResponseMetadata       `xml:"ResponseMetadata"`
}

type xmlAddTagsToResourceResult struct {
	TagList []xmlTag `xml:"TagList>Tag"`
}

func handleAddTagsToResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	arn := form.Get("ResourceName")
	if arn == "" {
		return xmlErr(service.ErrValidation("ResourceName is required."))
	}
	tags := parseTags(form)

	if !store.AddTagsToResource(arn, tags) {
		return xmlErr(service.NewAWSError("InvalidARN",
			"Resource "+arn+" not found.", http.StatusNotFound))
	}

	allTags, _ := store.ListTagsForResource(arn)
	xmlTags := make([]xmlTag, 0, len(allTags))
	for k, v := range allTags {
		xmlTags = append(xmlTags, xmlTag{Key: k, Value: v})
	}

	return xmlOK(&xmlAddTagsToResourceResponse{
		Xmlns:  ecXmlns,
		Result: xmlAddTagsToResourceResult{TagList: xmlTags},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- RemoveTagsFromResource ----

type xmlRemoveTagsFromResourceResponse struct {
	XMLName xml.Name                       `xml:"RemoveTagsFromResourceResponse"`
	Xmlns   string                         `xml:"xmlns,attr"`
	Result  xmlRemoveTagsFromResourceResult `xml:"RemoveTagsFromResourceResult"`
	Meta    xmlResponseMetadata            `xml:"ResponseMetadata"`
}

type xmlRemoveTagsFromResourceResult struct {
	TagList []xmlTag `xml:"TagList>Tag"`
}

func handleRemoveTagsFromResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	arn := form.Get("ResourceName")
	if arn == "" {
		return xmlErr(service.ErrValidation("ResourceName is required."))
	}
	keys := parseTagKeys(form)

	if !store.RemoveTagsFromResource(arn, keys) {
		return xmlErr(service.NewAWSError("InvalidARN",
			"Resource "+arn+" not found.", http.StatusNotFound))
	}

	allTags, _ := store.ListTagsForResource(arn)
	xmlTags := make([]xmlTag, 0, len(allTags))
	for k, v := range allTags {
		xmlTags = append(xmlTags, xmlTag{Key: k, Value: v})
	}

	return xmlOK(&xmlRemoveTagsFromResourceResponse{
		Xmlns:  ecXmlns,
		Result: xmlRemoveTagsFromResourceResult{TagList: xmlTags},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- ListTagsForResource ----

type xmlListTagsForResourceResponse struct {
	XMLName xml.Name                    `xml:"ListTagsForResourceResponse"`
	Xmlns   string                      `xml:"xmlns,attr"`
	Result  xmlListTagsForResourceResult `xml:"ListTagsForResourceResult"`
	Meta    xmlResponseMetadata         `xml:"ResponseMetadata"`
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
		return xmlErr(service.NewAWSError("InvalidARN",
			"Resource "+arn+" not found.", http.StatusNotFound))
	}

	xmlTags := make([]xmlTag, 0, len(tags))
	for k, v := range tags {
		xmlTags = append(xmlTags, xmlTag{Key: k, Value: v})
	}

	return xmlOK(&xmlListTagsForResourceResponse{
		Xmlns:  ecXmlns,
		Result: xmlListTagsForResourceResult{TagList: xmlTags},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- helper functions ----

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

func parseMemberList(form url.Values, prefix string) []string {
	var result []string
	for i := 1; ; i++ {
		v := form.Get(fmt.Sprintf("%s.member.%d", prefix, i))
		if v == "" {
			break
		}
		result = append(result, v)
	}
	return result
}

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

func parseTagKeys(form url.Values) []string {
	var keys []string
	for i := 1; ; i++ {
		k := form.Get(fmt.Sprintf("TagKeys.member.%d", i))
		if k == "" {
			break
		}
		keys = append(keys, k)
	}
	return keys
}

func xmlOK(body any) (*service.Response, error) {
	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       body,
		Format:     service.FormatXML,
	}, nil
}

func xmlErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatXML}, awsErr
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
