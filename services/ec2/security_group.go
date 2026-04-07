package ec2

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ---- XML types ----

type xmlIpRange struct {
	CidrIp string `xml:"cidrIp"`
}

type xmlUserIdGroupPair struct {
	GroupId string `xml:"groupId"`
}

type xmlIpPermission struct {
	IpProtocol   string               `xml:"ipProtocol"`
	FromPort     *int                 `xml:"fromPort,omitempty"`
	ToPort       *int                 `xml:"toPort,omitempty"`
	IpRanges     []xmlIpRange         `xml:"ipRanges>item,omitempty"`
	Groups       []xmlUserIdGroupPair `xml:"groups>item,omitempty"`
}

type xmlSecurityGroup struct {
	GroupId              string            `xml:"groupId"`
	GroupName            string            `xml:"groupName"`
	GroupDescription     string            `xml:"groupDescription"`
	VpcId                string            `xml:"vpcId"`
	IpPermissions        []xmlIpPermission `xml:"ipPermissions>item,omitempty"`
	IpPermissionsEgress  []xmlIpPermission `xml:"ipPermissionsEgress>item,omitempty"`
}

func toXMLPermission(rule SGRule) xmlIpPermission {
	perm := xmlIpPermission{
		IpProtocol: rule.IpProtocol,
	}
	// Only include ports when protocol is not "all" (-1).
	if rule.IpProtocol != "-1" {
		from := rule.FromPort
		to := rule.ToPort
		perm.FromPort = &from
		perm.ToPort = &to
	}
	for _, cidr := range rule.CidrBlocks {
		perm.IpRanges = append(perm.IpRanges, xmlIpRange{CidrIp: cidr})
	}
	for _, gid := range rule.GroupIds {
		perm.Groups = append(perm.Groups, xmlUserIdGroupPair{GroupId: gid})
	}
	return perm
}

func toXMLSecurityGroup(sg *SecurityGroup) xmlSecurityGroup {
	xsg := xmlSecurityGroup{
		GroupId:          sg.GroupId,
		GroupName:        sg.GroupName,
		GroupDescription: sg.Description,
		VpcId:            sg.VpcId,
	}
	for _, r := range sg.IngressRules {
		xsg.IpPermissions = append(xsg.IpPermissions, toXMLPermission(r))
	}
	for _, r := range sg.EgressRules {
		xsg.IpPermissionsEgress = append(xsg.IpPermissionsEgress, toXMLPermission(r))
	}
	return xsg
}

// ---- CreateSecurityGroup ----

type xmlCreateSecurityGroupResponse struct {
	XMLName   xml.Name `xml:"CreateSecurityGroupResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
	Return    bool     `xml:"return"`
	GroupId   string   `xml:"groupId"`
}

func handleCreateSecurityGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)

	groupName := form.Get("GroupName")
	if groupName == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter GroupName.",
			http.StatusBadRequest))
	}
	description := form.Get("GroupDescription")
	if description == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter GroupDescription.",
			http.StatusBadRequest))
	}
	vpcId := form.Get("VpcId")
	if vpcId == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter VpcId.",
			http.StatusBadRequest))
	}

	sg, errCode := store.CreateSecurityGroup(groupName, description, vpcId)
	if errCode != "" {
		switch errCode {
		case "vpc_not_found":
			return xmlErr(service.NewAWSError("InvalidVpcID.NotFound",
				"The vpc ID '"+vpcId+"' does not exist.",
				http.StatusBadRequest))
		case "duplicate_name":
			return xmlErr(service.NewAWSError("InvalidGroup.Duplicate",
				"The security group '"+groupName+"' already exists for VPC '"+vpcId+"'.",
				http.StatusBadRequest))
		default:
			return xmlErr(service.NewAWSError("InvalidParameterValue",
				"Invalid parameter: "+errCode,
				http.StatusBadRequest))
		}
	}

	return xmlOK(&xmlCreateSecurityGroupResponse{
		Xmlns:     ec2Xmlns,
		RequestID: newUUID(),
		Return:    true,
		GroupId:   sg.GroupId,
	})
}

// ---- DescribeSecurityGroups ----

type xmlDescribeSecurityGroupsResponse struct {
	XMLName           xml.Name           `xml:"DescribeSecurityGroupsResponse"`
	Xmlns             string             `xml:"xmlns,attr"`
	RequestID         string             `xml:"requestId"`
	SecurityGroupInfo []xmlSecurityGroup `xml:"securityGroupInfo>item"`
}

func handleDescribeSecurityGroups(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)

	ids := parseIndexedParam(form, "GroupId")
	vpcId := extractFilterValue(form, "vpc-id")
	groupName := extractFilterValue(form, "group-name")

	sgs := store.ListSecurityGroups(ids, vpcId, groupName)

	xmlSGs := make([]xmlSecurityGroup, 0, len(sgs))
	for _, sg := range sgs {
		xmlSGs = append(xmlSGs, toXMLSecurityGroup(sg))
	}

	return xmlOK(&xmlDescribeSecurityGroupsResponse{
		Xmlns:             ec2Xmlns,
		RequestID:         newUUID(),
		SecurityGroupInfo: xmlSGs,
	})
}

// ---- DeleteSecurityGroup ----

type xmlDeleteSecurityGroupResponse struct {
	XMLName   xml.Name `xml:"DeleteSecurityGroupResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
	Return    bool     `xml:"return"`
}

func handleDeleteSecurityGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	groupId := form.Get("GroupId")
	if groupId == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter GroupId.",
			http.StatusBadRequest))
	}

	errCode := store.DeleteSecurityGroup(groupId)
	switch errCode {
	case "":
		// success
	case "not_found":
		return xmlErr(service.NewAWSError("InvalidGroup.NotFound",
			"The security group '"+groupId+"' does not exist.",
			http.StatusBadRequest))
	case "dependency":
		return xmlErr(service.NewAWSError("DependencyViolation",
			"resource sg/"+groupId+" has a dependent object.",
			http.StatusBadRequest))
	}

	return xmlOK(&xmlDeleteSecurityGroupResponse{
		Xmlns:     ec2Xmlns,
		RequestID: newUUID(),
		Return:    true,
	})
}

// ---- AuthorizeSecurityGroupIngress ----

type xmlAuthorizeSecurityGroupIngressResponse struct {
	XMLName   xml.Name `xml:"AuthorizeSecurityGroupIngressResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
	Return    bool     `xml:"return"`
}

func handleAuthorizeSecurityGroupIngress(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	groupId := form.Get("GroupId")
	if groupId == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter GroupId.",
			http.StatusBadRequest))
	}

	rules := parseIpPermissions(form)

	errCode := store.AuthorizeSecurityGroupIngress(groupId, rules)
	if errCode != "" {
		return xmlErr(service.NewAWSError("InvalidGroup.NotFound",
			"The security group '"+groupId+"' does not exist.",
			http.StatusBadRequest))
	}

	return xmlOK(&xmlAuthorizeSecurityGroupIngressResponse{
		Xmlns:     ec2Xmlns,
		RequestID: newUUID(),
		Return:    true,
	})
}

// ---- AuthorizeSecurityGroupEgress ----

type xmlAuthorizeSecurityGroupEgressResponse struct {
	XMLName   xml.Name `xml:"AuthorizeSecurityGroupEgressResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
	Return    bool     `xml:"return"`
}

func handleAuthorizeSecurityGroupEgress(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	groupId := form.Get("GroupId")
	if groupId == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter GroupId.",
			http.StatusBadRequest))
	}

	rules := parseIpPermissions(form)

	errCode := store.AuthorizeSecurityGroupEgress(groupId, rules)
	if errCode != "" {
		return xmlErr(service.NewAWSError("InvalidGroup.NotFound",
			"The security group '"+groupId+"' does not exist.",
			http.StatusBadRequest))
	}

	return xmlOK(&xmlAuthorizeSecurityGroupEgressResponse{
		Xmlns:     ec2Xmlns,
		RequestID: newUUID(),
		Return:    true,
	})
}

// ---- RevokeSecurityGroupIngress ----

type xmlRevokeSecurityGroupIngressResponse struct {
	XMLName   xml.Name `xml:"RevokeSecurityGroupIngressResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
	Return    bool     `xml:"return"`
}

func handleRevokeSecurityGroupIngress(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	groupId := form.Get("GroupId")
	if groupId == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter GroupId.",
			http.StatusBadRequest))
	}

	rules := parseIpPermissions(form)

	errCode := store.RevokeSecurityGroupIngress(groupId, rules)
	switch errCode {
	case "":
		// success
	case "not_found":
		return xmlErr(service.NewAWSError("InvalidGroup.NotFound",
			"The security group '"+groupId+"' does not exist.",
			http.StatusBadRequest))
	case "rule_not_found":
		return xmlErr(service.NewAWSError("InvalidPermission.NotFound",
			"The specified rule does not exist in this security group.",
			http.StatusBadRequest))
	}

	return xmlOK(&xmlRevokeSecurityGroupIngressResponse{
		Xmlns:     ec2Xmlns,
		RequestID: newUUID(),
		Return:    true,
	})
}

// ---- RevokeSecurityGroupEgress ----

type xmlRevokeSecurityGroupEgressResponse struct {
	XMLName   xml.Name `xml:"RevokeSecurityGroupEgressResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
	Return    bool     `xml:"return"`
}

func handleRevokeSecurityGroupEgress(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	groupId := form.Get("GroupId")
	if groupId == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter GroupId.",
			http.StatusBadRequest))
	}

	rules := parseIpPermissions(form)

	errCode := store.RevokeSecurityGroupEgress(groupId, rules)
	switch errCode {
	case "":
		// success
	case "not_found":
		return xmlErr(service.NewAWSError("InvalidGroup.NotFound",
			"The security group '"+groupId+"' does not exist.",
			http.StatusBadRequest))
	case "rule_not_found":
		return xmlErr(service.NewAWSError("InvalidPermission.NotFound",
			"The specified rule does not exist in this security group.",
			http.StatusBadRequest))
	}

	return xmlOK(&xmlRevokeSecurityGroupEgressResponse{
		Xmlns:     ec2Xmlns,
		RequestID: newUUID(),
		Return:    true,
	})
}

// ---- IpPermissions form parsing ----

// parseIpPermissions parses IpPermissions.N.* form parameters into SGRule slice.
// Supports:
//
//	IpPermissions.N.IpProtocol
//	IpPermissions.N.FromPort
//	IpPermissions.N.ToPort
//	IpPermissions.N.IpRanges.M.CidrIp
//	IpPermissions.N.UserIdGroupPairs.M.GroupId
func parseIpPermissions(form url.Values) []SGRule {
	var rules []SGRule
	for i := 1; ; i++ {
		prefix := fmt.Sprintf("IpPermissions.%d", i)
		proto := form.Get(prefix + ".IpProtocol")
		if proto == "" {
			break
		}
		rule := SGRule{
			IpProtocol: proto,
		}
		if v := form.Get(prefix + ".FromPort"); v != "" {
			rule.FromPort, _ = strconv.Atoi(v)
		}
		if v := form.Get(prefix + ".ToPort"); v != "" {
			rule.ToPort, _ = strconv.Atoi(v)
		}
		for j := 1; ; j++ {
			cidr := form.Get(fmt.Sprintf("%s.IpRanges.%d.CidrIp", prefix, j))
			if cidr == "" {
				break
			}
			rule.CidrBlocks = append(rule.CidrBlocks, cidr)
		}
		for j := 1; ; j++ {
			gid := form.Get(fmt.Sprintf("%s.UserIdGroupPairs.%d.GroupId", prefix, j))
			if gid == "" {
				break
			}
			rule.GroupIds = append(rule.GroupIds, gid)
		}
		rules = append(rules, rule)
	}
	return rules
}
