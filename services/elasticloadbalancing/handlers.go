package elasticloadbalancing

import (
	"crypto/rand"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/neureaux/cloudmock/pkg/service"
)

const elbXmlns = "http://elasticloadbalancing.amazonaws.com/doc/2015-12-01/"

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

// ---- LoadBalancer XML types ----

type xmlLoadBalancer struct {
	LoadBalancerArn    string                `xml:"LoadBalancerArn"`
	DNSName            string                `xml:"DNSName"`
	LoadBalancerName   string                `xml:"LoadBalancerName"`
	Scheme             string                `xml:"Scheme"`
	Type               string                `xml:"Type"`
	State              xmlLBState            `xml:"State"`
	VpcId              string                `xml:"VpcId"`
	SecurityGroups     []string              `xml:"SecurityGroups>member"`
	AvailabilityZones  []xmlAvailabilityZone `xml:"AvailabilityZones>member"`
	IpAddressType      string                `xml:"IpAddressType"`
	CreatedTime        string                `xml:"CreatedTime"`
}

type xmlLBState struct {
	Code string `xml:"Code"`
}

func toXMLLoadBalancer(lb *LoadBalancer) xmlLoadBalancer {
	azs := make([]xmlAvailabilityZone, 0, len(lb.AvailabilityZones))
	for _, az := range lb.AvailabilityZones {
		azs = append(azs, xmlAvailabilityZone{ZoneName: az.ZoneName, SubnetId: az.SubnetID})
	}
	return xmlLoadBalancer{
		LoadBalancerArn:   lb.ARN,
		DNSName:           lb.DNSName,
		LoadBalancerName:  lb.Name,
		Scheme:            lb.Scheme,
		Type:              lb.Type,
		State:             xmlLBState{Code: lb.State},
		VpcId:             lb.VpcID,
		SecurityGroups:    lb.SecurityGroups,
		AvailabilityZones: azs,
		IpAddressType:     lb.IpAddressType,
		CreatedTime:       lb.CreatedTime.Format("2006-01-02T15:04:05Z"),
	}
}

// ---- CreateLoadBalancer ----

type xmlCreateLoadBalancerResponse struct {
	XMLName xml.Name                   `xml:"CreateLoadBalancerResponse"`
	Xmlns   string                     `xml:"xmlns,attr"`
	Result  xmlCreateLoadBalancerResult `xml:"CreateLoadBalancerResult"`
	Meta    xmlResponseMetadata        `xml:"ResponseMetadata"`
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

	lb, ok := store.CreateLoadBalancer(name, lbType, scheme, ipType, "", subnets, sgs)
	if !ok {
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
	XMLName xml.Name                      `xml:"DescribeLoadBalancersResponse"`
	Xmlns   string                        `xml:"xmlns,attr"`
	Result  xmlDescribeLoadBalancersResult `xml:"DescribeLoadBalancersResult"`
	Meta    xmlResponseMetadata           `xml:"ResponseMetadata"`
}

type xmlDescribeLoadBalancersResult struct {
	LoadBalancers []xmlLoadBalancer `xml:"LoadBalancers>member"`
}

func handleDescribeLoadBalancers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	names := parseMemberList(form, "Names")
	arns := parseMemberList(form, "LoadBalancerArns")

	lbs := store.ListLoadBalancers(names, arns)

	xmlLBs := make([]xmlLoadBalancer, 0, len(lbs))
	for _, lb := range lbs {
		xmlLBs = append(xmlLBs, toXMLLoadBalancer(lb))
	}

	return xmlOK(&xmlDescribeLoadBalancersResponse{
		Xmlns:  elbXmlns,
		Result: xmlDescribeLoadBalancersResult{LoadBalancers: xmlLBs},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
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

// ---- ModifyLoadBalancerAttributes ----

type xmlModifyLoadBalancerAttributesResponse struct {
	XMLName xml.Name            `xml:"ModifyLoadBalancerAttributesResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleModifyLoadBalancerAttributes(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	arn := form.Get("LoadBalancerArn")
	if arn == "" {
		return xmlErr(service.ErrValidation("LoadBalancerArn is required."))
	}

	if _, ok := store.GetLoadBalancer(arn); !ok {
		return xmlErr(service.NewAWSError("LoadBalancerNotFound",
			"Load balancer '"+arn+"' not found.", http.StatusNotFound))
	}

	return xmlOK(&xmlModifyLoadBalancerAttributesResponse{
		Xmlns: elbXmlns,
		Meta:  xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- TargetGroup XML types ----

type xmlTargetGroup struct {
	TargetGroupArn      string `xml:"TargetGroupArn"`
	TargetGroupName     string `xml:"TargetGroupName"`
	Protocol            string `xml:"Protocol"`
	Port                int    `xml:"Port"`
	VpcId               string `xml:"VpcId"`
	TargetType          string `xml:"TargetType"`
	HealthCheckEnabled  bool   `xml:"HealthCheckEnabled"`
	HealthCheckPath     string `xml:"HealthCheckPath"`
	HealthCheckProtocol string `xml:"HealthCheckProtocol"`
	HealthCheckPort     string `xml:"HealthCheckPort"`
	HealthyThreshold    int    `xml:"HealthyThresholdCount"`
	UnhealthyThreshold  int    `xml:"UnhealthyThresholdCount"`
}

func toXMLTargetGroup(tg *TargetGroup) xmlTargetGroup {
	return xmlTargetGroup{
		TargetGroupArn:      tg.ARN,
		TargetGroupName:     tg.Name,
		Protocol:            tg.Protocol,
		Port:                tg.Port,
		VpcId:               tg.VpcID,
		TargetType:          tg.TargetType,
		HealthCheckEnabled:  tg.HealthCheckEnabled,
		HealthCheckPath:     tg.HealthCheckPath,
		HealthCheckProtocol: tg.HealthCheckProtocol,
		HealthCheckPort:     tg.HealthCheckPort,
		HealthyThreshold:    tg.HealthyThreshold,
		UnhealthyThreshold:  tg.UnhealthyThreshold,
	}
}

// ---- CreateTargetGroup ----

type xmlCreateTargetGroupResponse struct {
	XMLName xml.Name                  `xml:"CreateTargetGroupResponse"`
	Xmlns   string                    `xml:"xmlns,attr"`
	Result  xmlCreateTargetGroupResult `xml:"CreateTargetGroupResult"`
	Meta    xmlResponseMetadata       `xml:"ResponseMetadata"`
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

	tg, ok := store.CreateTargetGroup(name, protocol, port, vpcID, targetType, healthPath, healthProtocol, healthPort)
	if !ok {
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
	XMLName xml.Name                     `xml:"DescribeTargetGroupsResponse"`
	Xmlns   string                       `xml:"xmlns,attr"`
	Result  xmlDescribeTargetGroupsResult `xml:"DescribeTargetGroupsResult"`
	Meta    xmlResponseMetadata          `xml:"ResponseMetadata"`
}

type xmlDescribeTargetGroupsResult struct {
	TargetGroups []xmlTargetGroup `xml:"TargetGroups>member"`
}

func handleDescribeTargetGroups(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	names := parseMemberList(form, "Names")
	arns := parseMemberList(form, "TargetGroupArns")
	lbARN := form.Get("LoadBalancerArn")

	tgs := store.ListTargetGroups(names, arns, lbARN)

	xmlTGs := make([]xmlTargetGroup, 0, len(tgs))
	for _, tg := range tgs {
		xmlTGs = append(xmlTGs, toXMLTargetGroup(tg))
	}

	return xmlOK(&xmlDescribeTargetGroupsResponse{
		Xmlns:  elbXmlns,
		Result: xmlDescribeTargetGroupsResult{TargetGroups: xmlTGs},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
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

	if !store.DeleteTargetGroup(arn) {
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
	XMLName xml.Name                  `xml:"ModifyTargetGroupResponse"`
	Xmlns   string                    `xml:"xmlns,attr"`
	Result  xmlModifyTargetGroupResult `xml:"ModifyTargetGroupResult"`
	Meta    xmlResponseMetadata       `xml:"ResponseMetadata"`
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
	healthyThresh := 0
	if v := form.Get("HealthyThresholdCount"); v != "" {
		healthyThresh, _ = strconv.Atoi(v)
	}
	unhealthyThresh := 0
	if v := form.Get("UnhealthyThresholdCount"); v != "" {
		unhealthyThresh, _ = strconv.Atoi(v)
	}

	tg, ok := store.ModifyTargetGroup(arn, healthPath, healthProtocol, healthPort, healthyThresh, unhealthyThresh)
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
	XMLName xml.Name                     `xml:"DescribeTargetHealthResponse"`
	Xmlns   string                       `xml:"xmlns,attr"`
	Result  xmlDescribeTargetHealthResult `xml:"DescribeTargetHealthResult"`
	Meta    xmlResponseMetadata          `xml:"ResponseMetadata"`
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
	State string `xml:"State"`
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
			TargetHealth: xmlTargetHealth{State: t.Health},
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
	ListenerArn     string        `xml:"ListenerArn"`
	LoadBalancerArn string        `xml:"LoadBalancerArn"`
	Protocol        string        `xml:"Protocol"`
	Port            int           `xml:"Port"`
	SslPolicy       string        `xml:"SslPolicy,omitempty"`
	DefaultActions  []xmlAction   `xml:"DefaultActions>member"`
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
	XMLName xml.Name               `xml:"CreateListenerResponse"`
	Xmlns   string                 `xml:"xmlns,attr"`
	Result  xmlCreateListenerResult `xml:"CreateListenerResult"`
	Meta    xmlResponseMetadata    `xml:"ResponseMetadata"`
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

	actions := parseActions(form)

	l, ok := store.CreateListener(lbARN, protocol, port, actions, sslPolicy, certARN)
	if !ok {
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
	XMLName xml.Name                  `xml:"DescribeListenersResponse"`
	Xmlns   string                    `xml:"xmlns,attr"`
	Result  xmlDescribeListenersResult `xml:"DescribeListenersResult"`
	Meta    xmlResponseMetadata       `xml:"ResponseMetadata"`
}

type xmlDescribeListenersResult struct {
	Listeners []xmlListener `xml:"Listeners>member"`
}

func handleDescribeListeners(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	lbARN := form.Get("LoadBalancerArn")

	listeners := store.ListListeners(lbARN)

	xmlListeners := make([]xmlListener, 0, len(listeners))
	for _, l := range listeners {
		xmlListeners = append(xmlListeners, toXMLListener(l))
	}

	return xmlOK(&xmlDescribeListenersResponse{
		Xmlns:  elbXmlns,
		Result: xmlDescribeListenersResult{Listeners: xmlListeners},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
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
	XMLName xml.Name               `xml:"ModifyListenerResponse"`
	Xmlns   string                 `xml:"xmlns,attr"`
	Result  xmlModifyListenerResult `xml:"ModifyListenerResult"`
	Meta    xmlResponseMetadata    `xml:"ResponseMetadata"`
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
	XMLName xml.Name           `xml:"CreateRuleResponse"`
	Xmlns   string             `xml:"xmlns,attr"`
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

	r, ok := store.CreateRule(listenerARN, priority, conditions, actions)
	if !ok {
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
	XMLName xml.Name              `xml:"DescribeRulesResponse"`
	Xmlns   string                `xml:"xmlns,attr"`
	Result  xmlDescribeRulesResult `xml:"DescribeRulesResult"`
	Meta    xmlResponseMetadata   `xml:"ResponseMetadata"`
}

type xmlDescribeRulesResult struct {
	Rules []xmlRule `xml:"Rules>member"`
}

func handleDescribeRules(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	listenerARN := form.Get("ListenerArn")

	rules := store.ListRules(listenerARN)

	xmlRules := make([]xmlRule, 0, len(rules))
	for _, r := range rules {
		xmlRules = append(xmlRules, toXMLRule(r))
	}

	return xmlOK(&xmlDescribeRulesResponse{
		Xmlns:  elbXmlns,
		Result: xmlDescribeRulesResult{Rules: xmlRules},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
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

	if !store.DeleteRule(arn) {
		return xmlErr(service.NewAWSError("RuleNotFound",
			"Rule '"+arn+"' not found.", http.StatusNotFound))
	}

	return xmlOK(&xmlDeleteRuleResponse{
		Xmlns: elbXmlns,
		Meta:  xmlResponseMetadata{RequestID: newUUID()},
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
		store.AddTags(arn, tags)
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
	XMLName xml.Name             `xml:"DescribeTagsResponse"`
	Xmlns   string               `xml:"xmlns,attr"`
	Result  xmlDescribeTagsResult `xml:"DescribeTagsResult"`
	Meta    xmlResponseMetadata  `xml:"ResponseMetadata"`
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
		targets = append(targets, Target{ID: id, Port: port})
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
