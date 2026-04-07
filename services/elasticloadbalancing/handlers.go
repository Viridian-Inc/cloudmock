package elasticloadbalancing

import (
	"crypto/rand"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"

	"github.com/Viridian-Inc/cloudmock/pkg/pagination"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

const elbXmlns = "http://elasticloadbalancing.amazonaws.com/doc/2015-12-01/"

// Default page size for ELBv2 list operations (AWS default is 400 for most).
const defaultPageSize = 400

// ---- shared XML types ----

type xmlResponseMetadata struct {
	RequestID string `xml:"RequestId"`
}

type xmlTag struct {
	Key   string `xml:"Key"`
	Value string `xml:"Value"`
}

type xmlAvailabilityZone struct {
	ZoneName string `xml:"ZoneName"`
	SubnetId string `xml:"SubnetId"`
}

type xmlAttribute struct {
	Key   string `xml:"Key"`
	Value string `xml:"Value"`
}

// ---- LoadBalancer XML types ----

type xmlLoadBalancer struct {
	LoadBalancerArn       string                `xml:"LoadBalancerArn"`
	DNSName               string                `xml:"DNSName"`
	CanonicalHostedZoneId string                `xml:"CanonicalHostedZoneId"`
	LoadBalancerName      string                `xml:"LoadBalancerName"`
	Scheme                string                `xml:"Scheme"`
	Type                  string                `xml:"Type"`
	State                 xmlLBState            `xml:"State"`
	VpcId                 string                `xml:"VpcId"`
	SecurityGroups        *xmlSecurityGroups    `xml:"SecurityGroups,omitempty"`
	AvailabilityZones     []xmlAvailabilityZone `xml:"AvailabilityZones>member"`
	IpAddressType         string                `xml:"IpAddressType"`
	CreatedTime           string                `xml:"CreatedTime"`
}

type xmlSecurityGroups struct {
	Members []string `xml:"member"`
}

type xmlLBState struct {
	Code   string `xml:"Code"`
	Reason string `xml:"Reason,omitempty"`
}

func toXMLLoadBalancer(lb *LoadBalancer) xmlLoadBalancer {
	azs := make([]xmlAvailabilityZone, 0, len(lb.AvailabilityZones))
	for _, az := range lb.AvailabilityZones {
		azs = append(azs, xmlAvailabilityZone{ZoneName: az.ZoneName, SubnetId: az.SubnetID})
	}
	var sgs *xmlSecurityGroups
	if len(lb.SecurityGroups) > 0 {
		sgs = &xmlSecurityGroups{Members: lb.SecurityGroups}
	}
	return xmlLoadBalancer{
		LoadBalancerArn:       lb.ARN,
		DNSName:               lb.DNSName,
		CanonicalHostedZoneId: lb.CanonicalHostedZoneID,
		LoadBalancerName:      lb.Name,
		Scheme:                lb.Scheme,
		Type:                  lb.Type,
		State:                 xmlLBState{Code: lb.State, Reason: lb.StateReason},
		VpcId:                 lb.VpcID,
		SecurityGroups:        sgs,
		AvailabilityZones:     azs,
		IpAddressType:         lb.IpAddressType,
		CreatedTime:           lb.CreatedTime.Format("2006-01-02T15:04:05Z"),
	}
}

// ---- CreateLoadBalancer ----

type xmlCreateLoadBalancerResponse struct {
	XMLName xml.Name                    `xml:"CreateLoadBalancerResponse"`
	Xmlns   string                      `xml:"xmlns,attr"`
	Result  xmlCreateLoadBalancerResult `xml:"CreateLoadBalancerResult"`
	Meta    xmlResponseMetadata         `xml:"ResponseMetadata"`
}

type xmlCreateLoadBalancerResult struct {
	LoadBalancers []xmlLoadBalancer `xml:"LoadBalancers>member"`
}

func handleCreateLoadBalancer(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	name := form.Get("Name")
	if name == "" {
		return xmlErr(service.ErrValidation("Name is required."))
	}

	lbType := form.Get("Type")
	scheme := form.Get("Scheme")
	ipType := form.Get("IpAddressType")
	subnets := parseMemberList(form, "Subnets")
	sgs := parseMemberList(form, "SecurityGroups")
	tags := parseTags(form)

	lb, err := store.CreateLoadBalancer(name, lbType, scheme, ipType, "", subnets, sgs, tags)
	if err != nil {
		return xmlErr(service.NewAWSError("DuplicateLoadBalancerName",
			"A load balancer with the name '"+name+"' already exists.", http.StatusBadRequest))
	}

	return xmlOK(&xmlCreateLoadBalancerResponse{
		Xmlns:  elbXmlns,
		Result: xmlCreateLoadBalancerResult{LoadBalancers: []xmlLoadBalancer{toXMLLoadBalancer(lb)}},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- DescribeLoadBalancers ----

type xmlDescribeLoadBalancersResponse struct {
	XMLName xml.Name                       `xml:"DescribeLoadBalancersResponse"`
	Xmlns   string                         `xml:"xmlns,attr"`
	Result  xmlDescribeLoadBalancersResult `xml:"DescribeLoadBalancersResult"`
	Meta    xmlResponseMetadata            `xml:"ResponseMetadata"`
}

type xmlDescribeLoadBalancersResult struct {
	LoadBalancers []xmlLoadBalancer `xml:"LoadBalancers>member"`
	NextMarker    string            `xml:"NextMarker,omitempty"`
}

func handleDescribeLoadBalancers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	names := parseMemberList(form, "Names")
	arns := parseMemberList(form, "LoadBalancerArns")
	marker := form.Get("Marker")
	pageSize := parsePageSize(form)

	lbs := store.ListLoadBalancers(names, arns)

	// If names or arns specified but no results found, return error per AWS behavior
	if (len(names) > 0 || len(arns) > 0) && len(lbs) == 0 {
		return xmlErr(service.NewAWSError("LoadBalancerNotFound",
			"One or more load balancers not found.", http.StatusBadRequest))
	}

	page := pagination.Paginate(lbs, marker, pageSize, defaultPageSize)

	xmlLBs := make([]xmlLoadBalancer, 0, len(page.Items))
	for _, lb := range page.Items {
		xmlLBs = append(xmlLBs, toXMLLoadBalancer(lb))
	}

	return xmlOK(&xmlDescribeLoadBalancersResponse{
		Xmlns: elbXmlns,
		Result: xmlDescribeLoadBalancersResult{
			LoadBalancers: xmlLBs,
			NextMarker:    page.NextToken,
		},
		Meta: xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- DeleteLoadBalancer ----

type xmlDeleteLoadBalancerResponse struct {
	XMLName xml.Name            `xml:"DeleteLoadBalancerResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleDeleteLoadBalancer(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	arn := form.Get("LoadBalancerArn")
	if arn == "" {
		return xmlErr(service.ErrValidation("LoadBalancerArn is required."))
	}

	if !store.DeleteLoadBalancer(arn) {
		return xmlErr(service.NewAWSError("LoadBalancerNotFound",
			"Load balancer '"+arn+"' not found.", http.StatusNotFound))
	}

	return xmlOK(&xmlDeleteLoadBalancerResponse{
		Xmlns: elbXmlns,
		Meta:  xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- DescribeLoadBalancerAttributes ----

type xmlDescribeLoadBalancerAttributesResponse struct {
	XMLName xml.Name                                 `xml:"DescribeLoadBalancerAttributesResponse"`
	Xmlns   string                                   `xml:"xmlns,attr"`
	Result  xmlDescribeLoadBalancerAttributesResult   `xml:"DescribeLoadBalancerAttributesResult"`
	Meta    xmlResponseMetadata                      `xml:"ResponseMetadata"`
}

type xmlDescribeLoadBalancerAttributesResult struct {
	Attributes []xmlAttribute `xml:"Attributes>member"`
}

func handleDescribeLoadBalancerAttributes(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	arn := form.Get("LoadBalancerArn")
	if arn == "" {
		return xmlErr(service.ErrValidation("LoadBalancerArn is required."))
	}

	attrs, ok := store.GetLoadBalancerAttributes(arn)
	if !ok {
		return xmlErr(service.NewAWSError("LoadBalancerNotFound",
			"Load balancer '"+arn+"' not found.", http.StatusNotFound))
	}

	xmlAttrs := attrsToXML(attrs)

	return xmlOK(&xmlDescribeLoadBalancerAttributesResponse{
		Xmlns:  elbXmlns,
		Result: xmlDescribeLoadBalancerAttributesResult{Attributes: xmlAttrs},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- ModifyLoadBalancerAttributes ----

type xmlModifyLoadBalancerAttributesResponse struct {
	XMLName xml.Name                               `xml:"ModifyLoadBalancerAttributesResponse"`
	Xmlns   string                                 `xml:"xmlns,attr"`
	Result  xmlModifyLoadBalancerAttributesResult   `xml:"ModifyLoadBalancerAttributesResult"`
	Meta    xmlResponseMetadata                    `xml:"ResponseMetadata"`
}

type xmlModifyLoadBalancerAttributesResult struct {
	Attributes []xmlAttribute `xml:"Attributes>member"`
}

func handleModifyLoadBalancerAttributes(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	arn := form.Get("LoadBalancerArn")
	if arn == "" {
		return xmlErr(service.ErrValidation("LoadBalancerArn is required."))
	}

	attrs := parseAttributes(form)

	result, ok := store.ModifyLoadBalancerAttributes(arn, attrs)
	if !ok {
		return xmlErr(service.NewAWSError("LoadBalancerNotFound",
			"Load balancer '"+arn+"' not found.", http.StatusNotFound))
	}

	return xmlOK(&xmlModifyLoadBalancerAttributesResponse{
		Xmlns:  elbXmlns,
		Result: xmlModifyLoadBalancerAttributesResult{Attributes: attrsToXML(result)},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- SetSecurityGroups ----

type xmlSetSecurityGroupsResponse struct {
	XMLName xml.Name                    `xml:"SetSecurityGroupsResponse"`
	Xmlns   string                      `xml:"xmlns,attr"`
	Result  xmlSetSecurityGroupsResult  `xml:"SetSecurityGroupsResult"`
	Meta    xmlResponseMetadata         `xml:"ResponseMetadata"`
}

type xmlSetSecurityGroupsResult struct {
	SecurityGroupIds []string `xml:"SecurityGroupIds>member"`
}

func handleSetSecurityGroups(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	arn := form.Get("LoadBalancerArn")
	if arn == "" {
		return xmlErr(service.ErrValidation("LoadBalancerArn is required."))
	}

	sgs := parseMemberList(form, "SecurityGroups")

	result, ok := store.SetSecurityGroups(arn, sgs)
	if !ok {
		return xmlErr(service.NewAWSError("LoadBalancerNotFound",
			"Load balancer '"+arn+"' not found.", http.StatusNotFound))
	}

	return xmlOK(&xmlSetSecurityGroupsResponse{
		Xmlns:  elbXmlns,
		Result: xmlSetSecurityGroupsResult{SecurityGroupIds: result},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- SetSubnets ----

type xmlSetSubnetsResponse struct {
	XMLName xml.Name             `xml:"SetSubnetsResponse"`
	Xmlns   string               `xml:"xmlns,attr"`
	Result  xmlSetSubnetsResult  `xml:"SetSubnetsResult"`
	Meta    xmlResponseMetadata  `xml:"ResponseMetadata"`
}

type xmlSetSubnetsResult struct {
	AvailabilityZones []xmlAvailabilityZone `xml:"AvailabilityZones>member"`
}

func handleSetSubnets(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	arn := form.Get("LoadBalancerArn")
	if arn == "" {
		return xmlErr(service.ErrValidation("LoadBalancerArn is required."))
	}

	subnets := parseMemberList(form, "Subnets")

	azs, ok := store.SetSubnets(arn, subnets)
	if !ok {
		return xmlErr(service.NewAWSError("LoadBalancerNotFound",
			"Load balancer '"+arn+"' not found.", http.StatusNotFound))
	}

	xmlAZs := make([]xmlAvailabilityZone, 0, len(azs))
	for _, az := range azs {
		xmlAZs = append(xmlAZs, xmlAvailabilityZone{ZoneName: az.ZoneName, SubnetId: az.SubnetID})
	}

	return xmlOK(&xmlSetSubnetsResponse{
		Xmlns:  elbXmlns,
		Result: xmlSetSubnetsResult{AvailabilityZones: xmlAZs},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- TargetGroup XML types ----

type xmlTargetGroup struct {
	TargetGroupArn             string `xml:"TargetGroupArn"`
	TargetGroupName            string `xml:"TargetGroupName"`
	Protocol                   string `xml:"Protocol"`
	ProtocolVersion            string `xml:"ProtocolVersion,omitempty"`
	Port                       int    `xml:"Port"`
	VpcId                      string `xml:"VpcId"`
	TargetType                 string `xml:"TargetType"`
	HealthCheckEnabled         bool   `xml:"HealthCheckEnabled"`
	HealthCheckPath            string `xml:"HealthCheckPath,omitempty"`
	HealthCheckProtocol        string `xml:"HealthCheckProtocol"`
	HealthCheckPort            string `xml:"HealthCheckPort"`
	HealthCheckIntervalSeconds int    `xml:"HealthCheckIntervalSeconds"`
	HealthCheckTimeoutSeconds  int    `xml:"HealthCheckTimeoutSeconds"`
	HealthyThreshold           int    `xml:"HealthyThresholdCount"`
	UnhealthyThreshold         int    `xml:"UnhealthyThresholdCount"`
	Matcher                    *xmlMatcher `xml:"Matcher,omitempty"`
}

type xmlMatcher struct {
	HttpCode string `xml:"HttpCode"`
}

func toXMLTargetGroup(tg *TargetGroup) xmlTargetGroup {
	var matcher *xmlMatcher
	if tg.Matcher != "" {
		matcher = &xmlMatcher{HttpCode: tg.Matcher}
	}
	return xmlTargetGroup{
		TargetGroupArn:             tg.ARN,
		TargetGroupName:            tg.Name,
		Protocol:                   tg.Protocol,
		ProtocolVersion:            tg.ProtocolVersion,
		Port:                       tg.Port,
		VpcId:                      tg.VpcID,
		TargetType:                 tg.TargetType,
		HealthCheckEnabled:         tg.HealthCheckEnabled,
		HealthCheckPath:            tg.HealthCheckPath,
		HealthCheckProtocol:        tg.HealthCheckProtocol,
		HealthCheckPort:            tg.HealthCheckPort,
		HealthCheckIntervalSeconds: tg.HealthCheckIntervalSeconds,
		HealthCheckTimeoutSeconds:  tg.HealthCheckTimeoutSeconds,
		HealthyThreshold:           tg.HealthyThreshold,
		UnhealthyThreshold:         tg.UnhealthyThreshold,
		Matcher:                    matcher,
	}
}

// ---- CreateTargetGroup ----

type xmlCreateTargetGroupResponse struct {
	XMLName xml.Name                   `xml:"CreateTargetGroupResponse"`
	Xmlns   string                     `xml:"xmlns,attr"`
	Result  xmlCreateTargetGroupResult `xml:"CreateTargetGroupResult"`
	Meta    xmlResponseMetadata        `xml:"ResponseMetadata"`
}

type xmlCreateTargetGroupResult struct {
	TargetGroups []xmlTargetGroup `xml:"TargetGroups>member"`
}

func handleCreateTargetGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	name := form.Get("Name")
	if name == "" {
		return xmlErr(service.ErrValidation("Name is required."))
	}
	protocol := form.Get("Protocol")
	port := 0
	if p := form.Get("Port"); p != "" {
		port, _ = strconv.Atoi(p)
	}
	vpcID := form.Get("VpcId")
	targetType := form.Get("TargetType")
	healthPath := form.Get("HealthCheckPath")
	healthProtocol := form.Get("HealthCheckProtocol")
	healthPort := form.Get("HealthCheckPort")
	tags := parseTags(form)

	tg, err := store.CreateTargetGroup(name, protocol, port, vpcID, targetType, healthPath, healthProtocol, healthPort, tags)
	if err != nil {
		return xmlErr(service.NewAWSError("DuplicateTargetGroupName",
			"A target group with the name '"+name+"' already exists.", http.StatusBadRequest))
	}

	return xmlOK(&xmlCreateTargetGroupResponse{
		Xmlns:  elbXmlns,
		Result: xmlCreateTargetGroupResult{TargetGroups: []xmlTargetGroup{toXMLTargetGroup(tg)}},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- DescribeTargetGroups ----

type xmlDescribeTargetGroupsResponse struct {
	XMLName xml.Name                      `xml:"DescribeTargetGroupsResponse"`
	Xmlns   string                        `xml:"xmlns,attr"`
	Result  xmlDescribeTargetGroupsResult `xml:"DescribeTargetGroupsResult"`
	Meta    xmlResponseMetadata           `xml:"ResponseMetadata"`
}

type xmlDescribeTargetGroupsResult struct {
	TargetGroups []xmlTargetGroup `xml:"TargetGroups>member"`
	NextMarker   string           `xml:"NextMarker,omitempty"`
}

func handleDescribeTargetGroups(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	names := parseMemberList(form, "Names")
	arns := parseMemberList(form, "TargetGroupArns")
	lbARN := form.Get("LoadBalancerArn")
	marker := form.Get("Marker")
	pageSize := parsePageSize(form)

	tgs := store.ListTargetGroups(names, arns, lbARN)

	if (len(names) > 0 || len(arns) > 0) && len(tgs) == 0 {
		return xmlErr(service.NewAWSError("TargetGroupNotFound",
			"One or more target groups not found.", http.StatusBadRequest))
	}

	page := pagination.Paginate(tgs, marker, pageSize, defaultPageSize)

	xmlTGs := make([]xmlTargetGroup, 0, len(page.Items))
	for _, tg := range page.Items {
		xmlTGs = append(xmlTGs, toXMLTargetGroup(tg))
	}

	return xmlOK(&xmlDescribeTargetGroupsResponse{
		Xmlns: elbXmlns,
		Result: xmlDescribeTargetGroupsResult{
			TargetGroups: xmlTGs,
			NextMarker:   page.NextToken,
		},
		Meta: xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- DeleteTargetGroup ----

type xmlDeleteTargetGroupResponse struct {
	XMLName xml.Name            `xml:"DeleteTargetGroupResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleDeleteTargetGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	arn := form.Get("TargetGroupArn")
	if arn == "" {
		return xmlErr(service.ErrValidation("TargetGroupArn is required."))
	}

	ok, err := store.DeleteTargetGroup(arn)
	if err != nil {
		errMsg := err.Error()
		if errMsg == "ResourceInUse" {
			return xmlErr(service.NewAWSError("ResourceInUse",
				"Target group '"+arn+"' is currently in use by a listener or rule.", http.StatusBadRequest))
		}
		return xmlErr(service.NewAWSError("TargetGroupNotFound",
			"Target group '"+arn+"' not found.", http.StatusNotFound))
	}
	if !ok {
		return xmlErr(service.NewAWSError("TargetGroupNotFound",
			"Target group '"+arn+"' not found.", http.StatusNotFound))
	}

	return xmlOK(&xmlDeleteTargetGroupResponse{
		Xmlns: elbXmlns,
		Meta:  xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- ModifyTargetGroup ----

type xmlModifyTargetGroupResponse struct {
	XMLName xml.Name                   `xml:"ModifyTargetGroupResponse"`
	Xmlns   string                     `xml:"xmlns,attr"`
	Result  xmlModifyTargetGroupResult `xml:"ModifyTargetGroupResult"`
	Meta    xmlResponseMetadata        `xml:"ResponseMetadata"`
}

type xmlModifyTargetGroupResult struct {
	TargetGroups []xmlTargetGroup `xml:"TargetGroups>member"`
}

func handleModifyTargetGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	arn := form.Get("TargetGroupArn")
	if arn == "" {
		return xmlErr(service.ErrValidation("TargetGroupArn is required."))
	}

	healthPath := form.Get("HealthCheckPath")
	healthProtocol := form.Get("HealthCheckProtocol")
	healthPort := form.Get("HealthCheckPort")
	matcher := form.Get("Matcher.HttpCode")
	healthyThresh := 0
	if v := form.Get("HealthyThresholdCount"); v != "" {
		healthyThresh, _ = strconv.Atoi(v)
	}
	unhealthyThresh := 0
	if v := form.Get("UnhealthyThresholdCount"); v != "" {
		unhealthyThresh, _ = strconv.Atoi(v)
	}
	intervalSeconds := 0
	if v := form.Get("HealthCheckIntervalSeconds"); v != "" {
		intervalSeconds, _ = strconv.Atoi(v)
	}
	timeoutSeconds := 0
	if v := form.Get("HealthCheckTimeoutSeconds"); v != "" {
		timeoutSeconds, _ = strconv.Atoi(v)
	}
	var healthCheckEnabled *bool
	if v := form.Get("HealthCheckEnabled"); v != "" {
		b := v == "true"
		healthCheckEnabled = &b
	}

	tg, ok := store.ModifyTargetGroup(arn, healthPath, healthProtocol, healthPort, healthyThresh, unhealthyThresh, intervalSeconds, timeoutSeconds, healthCheckEnabled, matcher)
	if !ok {
		return xmlErr(service.NewAWSError("TargetGroupNotFound",
			"Target group '"+arn+"' not found.", http.StatusNotFound))
	}

	return xmlOK(&xmlModifyTargetGroupResponse{
		Xmlns:  elbXmlns,
		Result: xmlModifyTargetGroupResult{TargetGroups: []xmlTargetGroup{toXMLTargetGroup(tg)}},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- DescribeTargetGroupAttributes ----

type xmlDescribeTargetGroupAttributesResponse struct {
	XMLName xml.Name                                `xml:"DescribeTargetGroupAttributesResponse"`
	Xmlns   string                                  `xml:"xmlns,attr"`
	Result  xmlDescribeTargetGroupAttributesResult   `xml:"DescribeTargetGroupAttributesResult"`
	Meta    xmlResponseMetadata                     `xml:"ResponseMetadata"`
}

type xmlDescribeTargetGroupAttributesResult struct {
	Attributes []xmlAttribute `xml:"Attributes>member"`
}

func handleDescribeTargetGroupAttributes(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	arn := form.Get("TargetGroupArn")
	if arn == "" {
		return xmlErr(service.ErrValidation("TargetGroupArn is required."))
	}

	attrs, ok := store.GetTargetGroupAttributes(arn)
	if !ok {
		return xmlErr(service.NewAWSError("TargetGroupNotFound",
			"Target group '"+arn+"' not found.", http.StatusNotFound))
	}

	return xmlOK(&xmlDescribeTargetGroupAttributesResponse{
		Xmlns:  elbXmlns,
		Result: xmlDescribeTargetGroupAttributesResult{Attributes: attrsToXML(attrs)},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- ModifyTargetGroupAttributes ----

type xmlModifyTargetGroupAttributesResponse struct {
	XMLName xml.Name                              `xml:"ModifyTargetGroupAttributesResponse"`
	Xmlns   string                                `xml:"xmlns,attr"`
	Result  xmlModifyTargetGroupAttributesResult   `xml:"ModifyTargetGroupAttributesResult"`
	Meta    xmlResponseMetadata                   `xml:"ResponseMetadata"`
}

type xmlModifyTargetGroupAttributesResult struct {
	Attributes []xmlAttribute `xml:"Attributes>member"`
}

func handleModifyTargetGroupAttributes(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	arn := form.Get("TargetGroupArn")
	if arn == "" {
		return xmlErr(service.ErrValidation("TargetGroupArn is required."))
	}

	attrs := parseAttributes(form)

	result, ok := store.ModifyTargetGroupAttributes(arn, attrs)
	if !ok {
		return xmlErr(service.NewAWSError("TargetGroupNotFound",
			"Target group '"+arn+"' not found.", http.StatusNotFound))
	}

	return xmlOK(&xmlModifyTargetGroupAttributesResponse{
		Xmlns:  elbXmlns,
		Result: xmlModifyTargetGroupAttributesResult{Attributes: attrsToXML(result)},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- RegisterTargets ----

type xmlRegisterTargetsResponse struct {
	XMLName xml.Name            `xml:"RegisterTargetsResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleRegisterTargets(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	tgARN := form.Get("TargetGroupArn")
	if tgARN == "" {
		return xmlErr(service.ErrValidation("TargetGroupArn is required."))
	}

	targets := parseTargets(form)

	if !store.RegisterTargets(tgARN, targets) {
		return xmlErr(service.NewAWSError("TargetGroupNotFound",
			"Target group '"+tgARN+"' not found.", http.StatusNotFound))
	}

	return xmlOK(&xmlRegisterTargetsResponse{
		Xmlns: elbXmlns,
		Meta:  xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- DeregisterTargets ----

type xmlDeregisterTargetsResponse struct {
	XMLName xml.Name            `xml:"DeregisterTargetsResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleDeregisterTargets(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	tgARN := form.Get("TargetGroupArn")
	if tgARN == "" {
		return xmlErr(service.ErrValidation("TargetGroupArn is required."))
	}

	var ids []string
	for i := 1; ; i++ {
		id := form.Get(fmt.Sprintf("Targets.member.%d.Id", i))
		if id == "" {
			break
		}
		ids = append(ids, id)
	}

	if !store.DeregisterTargets(tgARN, ids) {
		return xmlErr(service.NewAWSError("TargetGroupNotFound",
			"Target group '"+tgARN+"' not found.", http.StatusNotFound))
	}

	return xmlOK(&xmlDeregisterTargetsResponse{
		Xmlns: elbXmlns,
		Meta:  xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- DescribeTargetHealth ----

type xmlDescribeTargetHealthResponse struct {
	XMLName xml.Name                      `xml:"DescribeTargetHealthResponse"`
	Xmlns   string                        `xml:"xmlns,attr"`
	Result  xmlDescribeTargetHealthResult `xml:"DescribeTargetHealthResult"`
	Meta    xmlResponseMetadata           `xml:"ResponseMetadata"`
}

type xmlDescribeTargetHealthResult struct {
	TargetHealthDescriptions []xmlTargetHealthDescription `xml:"TargetHealthDescriptions>member"`
}

type xmlTargetHealthDescription struct {
	Target       xmlTargetDescription `xml:"Target"`
	TargetHealth xmlTargetHealth      `xml:"TargetHealth"`
}

type xmlTargetDescription struct {
	Id   string `xml:"Id"`
	Port int    `xml:"Port"`
}

type xmlTargetHealth struct {
	State       string `xml:"State"`
	Reason      string `xml:"Reason,omitempty"`
	Description string `xml:"Description,omitempty"`
}

func handleDescribeTargetHealth(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	tgARN := form.Get("TargetGroupArn")
	if tgARN == "" {
		return xmlErr(service.ErrValidation("TargetGroupArn is required."))
	}

	targets, ok := store.DescribeTargetHealth(tgARN)
	if !ok {
		return xmlErr(service.NewAWSError("TargetGroupNotFound",
			"Target group '"+tgARN+"' not found.", http.StatusNotFound))
	}

	xmlDescs := make([]xmlTargetHealthDescription, 0, len(targets))
	for _, t := range targets {
		xmlDescs = append(xmlDescs, xmlTargetHealthDescription{
			Target:       xmlTargetDescription{Id: t.ID, Port: t.Port},
			TargetHealth: xmlTargetHealth{State: t.Health, Reason: t.HealthReason},
		})
	}

	return xmlOK(&xmlDescribeTargetHealthResponse{
		Xmlns:  elbXmlns,
		Result: xmlDescribeTargetHealthResult{TargetHealthDescriptions: xmlDescs},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- Listener XML types ----

type xmlListener struct {
	ListenerArn     string      `xml:"ListenerArn"`
	LoadBalancerArn string      `xml:"LoadBalancerArn"`
	Protocol        string      `xml:"Protocol"`
	Port            int         `xml:"Port"`
	SslPolicy       string      `xml:"SslPolicy,omitempty"`
	DefaultActions  []xmlAction `xml:"DefaultActions>member"`
}

type xmlAction struct {
	Type           string `xml:"Type"`
	TargetGroupArn string `xml:"TargetGroupArn,omitempty"`
	Order          int    `xml:"Order"`
}

func toXMLListener(l *Listener) xmlListener {
	actions := make([]xmlAction, 0, len(l.DefaultActions))
	for _, a := range l.DefaultActions {
		actions = append(actions, xmlAction{
			Type:           a.Type,
			TargetGroupArn: a.TargetGroupARN,
			Order:          a.Order,
		})
	}
	return xmlListener{
		ListenerArn:     l.ARN,
		LoadBalancerArn: l.LoadBalancerARN,
		Protocol:        l.Protocol,
		Port:            l.Port,
		SslPolicy:       l.SslPolicy,
		DefaultActions:  actions,
	}
}

// ---- CreateListener ----

type xmlCreateListenerResponse struct {
	XMLName xml.Name                `xml:"CreateListenerResponse"`
	Xmlns   string                  `xml:"xmlns,attr"`
	Result  xmlCreateListenerResult `xml:"CreateListenerResult"`
	Meta    xmlResponseMetadata     `xml:"ResponseMetadata"`
}

type xmlCreateListenerResult struct {
	Listeners []xmlListener `xml:"Listeners>member"`
}

func handleCreateListener(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	lbARN := form.Get("LoadBalancerArn")
	if lbARN == "" {
		return xmlErr(service.ErrValidation("LoadBalancerArn is required."))
	}
	protocol := form.Get("Protocol")
	if protocol == "" {
		protocol = "HTTP"
	}
	port := 80
	if p := form.Get("Port"); p != "" {
		port, _ = strconv.Atoi(p)
	}
	sslPolicy := form.Get("SslPolicy")
	certARN := form.Get("Certificates.member.1.CertificateArn")
	tags := parseTags(form)

	actions := parseActions(form)

	l, err := store.CreateListener(lbARN, protocol, port, actions, sslPolicy, certARN, tags)
	if err != nil {
		errMsg := err.Error()
		if errMsg == "DuplicateListener" {
			return xmlErr(service.NewAWSError("DuplicateListener",
				"A listener already exists on port "+strconv.Itoa(port)+" for load balancer '"+lbARN+"'.", http.StatusBadRequest))
		}
		return xmlErr(service.NewAWSError("LoadBalancerNotFound",
			"Load balancer '"+lbARN+"' not found.", http.StatusNotFound))
	}

	return xmlOK(&xmlCreateListenerResponse{
		Xmlns:  elbXmlns,
		Result: xmlCreateListenerResult{Listeners: []xmlListener{toXMLListener(l)}},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- DescribeListeners ----

type xmlDescribeListenersResponse struct {
	XMLName xml.Name                   `xml:"DescribeListenersResponse"`
	Xmlns   string                     `xml:"xmlns,attr"`
	Result  xmlDescribeListenersResult `xml:"DescribeListenersResult"`
	Meta    xmlResponseMetadata        `xml:"ResponseMetadata"`
}

type xmlDescribeListenersResult struct {
	Listeners  []xmlListener `xml:"Listeners>member"`
	NextMarker string        `xml:"NextMarker,omitempty"`
}

func handleDescribeListeners(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	lbARN := form.Get("LoadBalancerArn")
	listenerARNs := parseMemberList(form, "ListenerArns")
	marker := form.Get("Marker")
	pageSize := parsePageSize(form)

	listeners := store.ListListeners(lbARN, listenerARNs)

	page := pagination.Paginate(listeners, marker, pageSize, defaultPageSize)

	xmlListeners := make([]xmlListener, 0, len(page.Items))
	for _, l := range page.Items {
		xmlListeners = append(xmlListeners, toXMLListener(l))
	}

	return xmlOK(&xmlDescribeListenersResponse{
		Xmlns: elbXmlns,
		Result: xmlDescribeListenersResult{
			Listeners:  xmlListeners,
			NextMarker: page.NextToken,
		},
		Meta: xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- DeleteListener ----

type xmlDeleteListenerResponse struct {
	XMLName xml.Name            `xml:"DeleteListenerResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleDeleteListener(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	arn := form.Get("ListenerArn")
	if arn == "" {
		return xmlErr(service.ErrValidation("ListenerArn is required."))
	}

	if !store.DeleteListener(arn) {
		return xmlErr(service.NewAWSError("ListenerNotFound",
			"Listener '"+arn+"' not found.", http.StatusNotFound))
	}

	return xmlOK(&xmlDeleteListenerResponse{
		Xmlns: elbXmlns,
		Meta:  xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- ModifyListener ----

type xmlModifyListenerResponse struct {
	XMLName xml.Name                `xml:"ModifyListenerResponse"`
	Xmlns   string                  `xml:"xmlns,attr"`
	Result  xmlModifyListenerResult `xml:"ModifyListenerResult"`
	Meta    xmlResponseMetadata     `xml:"ResponseMetadata"`
}

type xmlModifyListenerResult struct {
	Listeners []xmlListener `xml:"Listeners>member"`
}

func handleModifyListener(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	arn := form.Get("ListenerArn")
	if arn == "" {
		return xmlErr(service.ErrValidation("ListenerArn is required."))
	}

	protocol := form.Get("Protocol")
	port := 0
	if p := form.Get("Port"); p != "" {
		port, _ = strconv.Atoi(p)
	}
	sslPolicy := form.Get("SslPolicy")
	certARN := form.Get("Certificates.member.1.CertificateArn")
	actions := parseActions(form)

	l, ok := store.ModifyListener(arn, protocol, port, actions, sslPolicy, certARN)
	if !ok {
		return xmlErr(service.NewAWSError("ListenerNotFound",
			"Listener '"+arn+"' not found.", http.StatusNotFound))
	}

	return xmlOK(&xmlModifyListenerResponse{
		Xmlns:  elbXmlns,
		Result: xmlModifyListenerResult{Listeners: []xmlListener{toXMLListener(l)}},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- Rule XML types ----

type xmlRule struct {
	RuleArn    string         `xml:"RuleArn"`
	Priority   string         `xml:"Priority"`
	Conditions []xmlCondition `xml:"Conditions>member"`
	Actions    []xmlAction    `xml:"Actions>member"`
	IsDefault  bool           `xml:"IsDefault"`
}

type xmlCondition struct {
	Field  string   `xml:"Field"`
	Values []string `xml:"Values>member"`
}

func toXMLRule(r *Rule) xmlRule {
	conditions := make([]xmlCondition, 0, len(r.Conditions))
	for _, c := range r.Conditions {
		conditions = append(conditions, xmlCondition{Field: c.Field, Values: c.Values})
	}
	actions := make([]xmlAction, 0, len(r.Actions))
	for _, a := range r.Actions {
		actions = append(actions, xmlAction{Type: a.Type, TargetGroupArn: a.TargetGroupARN, Order: a.Order})
	}
	return xmlRule{
		RuleArn:    r.ARN,
		Priority:   r.Priority,
		Conditions: conditions,
		Actions:    actions,
		IsDefault:  r.IsDefault,
	}
}

// ---- CreateRule ----

type xmlCreateRuleResponse struct {
	XMLName xml.Name            `xml:"CreateRuleResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Result  xmlCreateRuleResult `xml:"CreateRuleResult"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

type xmlCreateRuleResult struct {
	Rules []xmlRule `xml:"Rules>member"`
}

func handleCreateRule(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	listenerARN := form.Get("ListenerArn")
	if listenerARN == "" {
		return xmlErr(service.ErrValidation("ListenerArn is required."))
	}
	priority := form.Get("Priority")
	if priority == "" {
		priority = "1"
	}

	conditions := parseConditions(form)
	actions := parseActions(form)
	tags := parseTags(form)

	r, err := store.CreateRule(listenerARN, priority, conditions, actions, tags)
	if err != nil {
		errMsg := err.Error()
		if errMsg == "PriorityInUse" {
			return xmlErr(service.NewAWSError("PriorityInUse",
				"Priority '"+priority+"' is already in use.", http.StatusBadRequest))
		}
		return xmlErr(service.NewAWSError("ListenerNotFound",
			"Listener '"+listenerARN+"' not found.", http.StatusNotFound))
	}

	return xmlOK(&xmlCreateRuleResponse{
		Xmlns:  elbXmlns,
		Result: xmlCreateRuleResult{Rules: []xmlRule{toXMLRule(r)}},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- DescribeRules ----

type xmlDescribeRulesResponse struct {
	XMLName xml.Name               `xml:"DescribeRulesResponse"`
	Xmlns   string                 `xml:"xmlns,attr"`
	Result  xmlDescribeRulesResult `xml:"DescribeRulesResult"`
	Meta    xmlResponseMetadata    `xml:"ResponseMetadata"`
}

type xmlDescribeRulesResult struct {
	Rules      []xmlRule `xml:"Rules>member"`
	NextMarker string    `xml:"NextMarker,omitempty"`
}

func handleDescribeRules(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	listenerARN := form.Get("ListenerArn")
	ruleARNs := parseMemberList(form, "RuleArns")
	marker := form.Get("Marker")
	pageSize := parsePageSize(form)

	rules := store.ListRules(listenerARN, ruleARNs)

	page := pagination.Paginate(rules, marker, pageSize, defaultPageSize)

	xmlRules := make([]xmlRule, 0, len(page.Items))
	for _, r := range page.Items {
		xmlRules = append(xmlRules, toXMLRule(r))
	}

	return xmlOK(&xmlDescribeRulesResponse{
		Xmlns: elbXmlns,
		Result: xmlDescribeRulesResult{
			Rules:      xmlRules,
			NextMarker: page.NextToken,
		},
		Meta: xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- DeleteRule ----

type xmlDeleteRuleResponse struct {
	XMLName xml.Name            `xml:"DeleteRuleResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleDeleteRule(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	arn := form.Get("RuleArn")
	if arn == "" {
		return xmlErr(service.ErrValidation("RuleArn is required."))
	}

	ok, err := store.DeleteRule(arn)
	if err != nil {
		errMsg := err.Error()
		if errMsg == "OperationNotPermitted" {
			return xmlErr(service.NewAWSError("OperationNotPermitted",
				"Default rules cannot be deleted.", http.StatusBadRequest))
		}
		return xmlErr(service.NewAWSError("RuleNotFound",
			"Rule '"+arn+"' not found.", http.StatusNotFound))
	}
	if !ok {
		return xmlErr(service.NewAWSError("RuleNotFound",
			"Rule '"+arn+"' not found.", http.StatusNotFound))
	}

	return xmlOK(&xmlDeleteRuleResponse{
		Xmlns: elbXmlns,
		Meta:  xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- ModifyRule ----

type xmlModifyRuleResponse struct {
	XMLName xml.Name            `xml:"ModifyRuleResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Result  xmlModifyRuleResult `xml:"ModifyRuleResult"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

type xmlModifyRuleResult struct {
	Rules []xmlRule `xml:"Rules>member"`
}

func handleModifyRule(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	arn := form.Get("RuleArn")
	if arn == "" {
		return xmlErr(service.ErrValidation("RuleArn is required."))
	}

	conditions := parseConditions(form)
	actions := parseActions(form)

	r, ok := store.ModifyRule(arn, conditions, actions)
	if !ok {
		return xmlErr(service.NewAWSError("RuleNotFound",
			"Rule '"+arn+"' not found.", http.StatusNotFound))
	}

	return xmlOK(&xmlModifyRuleResponse{
		Xmlns:  elbXmlns,
		Result: xmlModifyRuleResult{Rules: []xmlRule{toXMLRule(r)}},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- SetRulePriorities ----

type xmlSetRulePrioritiesResponse struct {
	XMLName xml.Name                   `xml:"SetRulePrioritiesResponse"`
	Xmlns   string                     `xml:"xmlns,attr"`
	Result  xmlSetRulePrioritiesResult `xml:"SetRulePrioritiesResult"`
	Meta    xmlResponseMetadata        `xml:"ResponseMetadata"`
}

type xmlSetRulePrioritiesResult struct {
	Rules []xmlRule `xml:"Rules>member"`
}

func handleSetRulePriorities(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)

	priorities := make(map[string]string)
	for i := 1; ; i++ {
		ruleARN := form.Get(fmt.Sprintf("RulePriorities.member.%d.RuleArn", i))
		if ruleARN == "" {
			break
		}
		priority := form.Get(fmt.Sprintf("RulePriorities.member.%d.Priority", i))
		priorities[ruleARN] = priority
	}

	rules, err := store.SetRulePriorities(priorities)
	if err != nil {
		errMsg := err.Error()
		if errMsg == "OperationNotPermitted" {
			return xmlErr(service.NewAWSError("OperationNotPermitted",
				"Cannot set priority of default rule.", http.StatusBadRequest))
		}
		return xmlErr(service.NewAWSError("RuleNotFound",
			"One or more rules not found.", http.StatusNotFound))
	}

	xmlRules := make([]xmlRule, 0, len(rules))
	for _, r := range rules {
		xmlRules = append(xmlRules, toXMLRule(r))
	}

	return xmlOK(&xmlSetRulePrioritiesResponse{
		Xmlns:  elbXmlns,
		Result: xmlSetRulePrioritiesResult{Rules: xmlRules},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- AddTags ----

type xmlAddTagsResponse struct {
	XMLName xml.Name            `xml:"AddTagsResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleAddTags(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	arns := parseMemberList(form, "ResourceArns")
	tags := parseTags(form)

	for _, arn := range arns {
		if !store.AddTags(arn, tags) {
			return xmlErr(service.NewAWSError("LoadBalancerNotFound",
				"Resource '"+arn+"' not found.", http.StatusNotFound))
		}
	}

	return xmlOK(&xmlAddTagsResponse{
		Xmlns: elbXmlns,
		Meta:  xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- RemoveTags ----

type xmlRemoveTagsResponse struct {
	XMLName xml.Name            `xml:"RemoveTagsResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleRemoveTags(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	arns := parseMemberList(form, "ResourceArns")
	keys := parseTagKeys(form)

	for _, arn := range arns {
		store.RemoveTags(arn, keys)
	}

	return xmlOK(&xmlRemoveTagsResponse{
		Xmlns: elbXmlns,
		Meta:  xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- DescribeTagsForResource ----

type xmlDescribeTagsResponse struct {
	XMLName xml.Name              `xml:"DescribeTagsResponse"`
	Xmlns   string                `xml:"xmlns,attr"`
	Result  xmlDescribeTagsResult `xml:"DescribeTagsResult"`
	Meta    xmlResponseMetadata   `xml:"ResponseMetadata"`
}

type xmlDescribeTagsResult struct {
	TagDescriptions []xmlTagDescription `xml:"TagDescriptions>member"`
}

type xmlTagDescription struct {
	ResourceArn string   `xml:"ResourceArn"`
	Tags        []xmlTag `xml:"Tags>member"`
}

func handleDescribeTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	arns := parseMemberList(form, "ResourceArns")

	descs := make([]xmlTagDescription, 0, len(arns))
	for _, arn := range arns {
		tags, ok := store.ListTags(arn)
		if !ok {
			continue
		}
		xmlTags := make([]xmlTag, 0, len(tags))
		for k, v := range tags {
			xmlTags = append(xmlTags, xmlTag{Key: k, Value: v})
		}
		descs = append(descs, xmlTagDescription{ResourceArn: arn, Tags: xmlTags})
	}

	return xmlOK(&xmlDescribeTagsResponse{
		Xmlns:  elbXmlns,
		Result: xmlDescribeTagsResult{TagDescriptions: descs},
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

func parseTargets(form url.Values) []Target {
	var targets []Target
	for i := 1; ; i++ {
		id := form.Get(fmt.Sprintf("Targets.member.%d.Id", i))
		if id == "" {
			break
		}
		port := 0
		if p := form.Get(fmt.Sprintf("Targets.member.%d.Port", i)); p != "" {
			port, _ = strconv.Atoi(p)
		}
		az := form.Get(fmt.Sprintf("Targets.member.%d.AvailabilityZone", i))
		targets = append(targets, Target{ID: id, Port: port, AvailabilityZone: az})
	}
	return targets
}

func parseActions(form url.Values) []Action {
	var actions []Action
	for i := 1; ; i++ {
		aType := form.Get(fmt.Sprintf("DefaultActions.member.%d.Type", i))
		if aType == "" {
			// Also try Actions.member.N for rules.
			aType = form.Get(fmt.Sprintf("Actions.member.%d.Type", i))
		}
		if aType == "" {
			break
		}
		tgARN := form.Get(fmt.Sprintf("DefaultActions.member.%d.TargetGroupArn", i))
		if tgARN == "" {
			tgARN = form.Get(fmt.Sprintf("Actions.member.%d.TargetGroupArn", i))
		}
		order := i
		if o := form.Get(fmt.Sprintf("DefaultActions.member.%d.Order", i)); o != "" {
			order, _ = strconv.Atoi(o)
		} else if o := form.Get(fmt.Sprintf("Actions.member.%d.Order", i)); o != "" {
			order, _ = strconv.Atoi(o)
		}
		actions = append(actions, Action{Type: aType, TargetGroupARN: tgARN, Order: order})
	}
	return actions
}

func parseConditions(form url.Values) []RuleCondition {
	var conditions []RuleCondition
	for i := 1; ; i++ {
		field := form.Get(fmt.Sprintf("Conditions.member.%d.Field", i))
		if field == "" {
			break
		}
		var values []string
		for j := 1; ; j++ {
			v := form.Get(fmt.Sprintf("Conditions.member.%d.Values.member.%d", i, j))
			if v == "" {
				break
			}
			values = append(values, v)
		}
		conditions = append(conditions, RuleCondition{Field: field, Values: values})
	}
	return conditions
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

func parseAttributes(form url.Values) map[string]string {
	attrs := make(map[string]string)
	for i := 1; ; i++ {
		key := form.Get(fmt.Sprintf("Attributes.member.%d.Key", i))
		if key == "" {
			break
		}
		val := form.Get(fmt.Sprintf("Attributes.member.%d.Value", i))
		attrs[key] = val
	}
	return attrs
}

func parsePageSize(form url.Values) int {
	if v := form.Get("PageSize"); v != "" {
		n, _ := strconv.Atoi(v)
		return n
	}
	return 0
}

func attrsToXML(attrs map[string]string) []xmlAttribute {
	result := make([]xmlAttribute, 0, len(attrs))
	for k, v := range attrs {
		result = append(result, xmlAttribute{Key: k, Value: v})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Key < result[j].Key })
	return result
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
