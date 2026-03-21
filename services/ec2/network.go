package ec2

import (
	"encoding/xml"
	"fmt"
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// ============================================================
// Elastic IP XML types
// ============================================================

type xmlAddress struct {
	AllocationId       string `xml:"allocationId"`
	PublicIp           string `xml:"publicIp"`
	AssociationId      string `xml:"associationId,omitempty"`
	InstanceId         string `xml:"instanceId,omitempty"`
	NetworkInterfaceId string `xml:"networkInterfaceId,omitempty"`
	Domain             string `xml:"domain"`
}

func toXMLAddress(eip *ElasticIP) xmlAddress {
	return xmlAddress{
		AllocationId:       eip.AllocationId,
		PublicIp:           eip.PublicIp,
		AssociationId:      eip.AssociationId,
		InstanceId:         eip.InstanceId,
		NetworkInterfaceId: eip.NetworkInterfaceId,
		Domain:             "vpc",
	}
}

// ---- AllocateAddress ----

type xmlAllocateAddressResponse struct {
	XMLName      xml.Name `xml:"AllocateAddressResponse"`
	Xmlns        string   `xml:"xmlns,attr"`
	RequestID    string   `xml:"requestId"`
	AllocationId string   `xml:"allocationId"`
	PublicIp     string   `xml:"publicIp"`
	Domain       string   `xml:"domain"`
}

func handleAllocateAddress(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	eip := store.AllocateAddress()
	return xmlOK(&xmlAllocateAddressResponse{
		Xmlns:        ec2Xmlns,
		RequestID:    newUUID(),
		AllocationId: eip.AllocationId,
		PublicIp:     eip.PublicIp,
		Domain:       "vpc",
	})
}

// ---- ReleaseAddress ----

type xmlReleaseAddressResponse struct {
	XMLName   xml.Name `xml:"ReleaseAddressResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
	Return    bool     `xml:"return"`
}

func handleReleaseAddress(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	allocationId := form.Get("AllocationId")
	if allocationId == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter AllocationId.",
			http.StatusBadRequest))
	}

	errCode := store.ReleaseAddress(allocationId)
	switch errCode {
	case "":
		// success
	case "not_found":
		return xmlErr(service.NewAWSError("InvalidAllocationID.NotFound",
			"The allocation ID '"+allocationId+"' does not exist.",
			http.StatusBadRequest))
	case "still_associated":
		return xmlErr(service.NewAWSError("InvalidIPAddress.InUse",
			"The allocation '"+allocationId+"' is still associated.",
			http.StatusBadRequest))
	default:
		return xmlErr(service.NewAWSError("InvalidParameterValue", errCode, http.StatusBadRequest))
	}

	return xmlOK(&xmlReleaseAddressResponse{
		Xmlns:     ec2Xmlns,
		RequestID: newUUID(),
		Return:    true,
	})
}

// ---- AssociateAddress ----

type xmlAssociateAddressResponse struct {
	XMLName       xml.Name `xml:"AssociateAddressResponse"`
	Xmlns         string   `xml:"xmlns,attr"`
	RequestID     string   `xml:"requestId"`
	AssociationId string   `xml:"associationId"`
}

func handleAssociateAddress(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	allocationId := form.Get("AllocationId")
	if allocationId == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter AllocationId.",
			http.StatusBadRequest))
	}
	instanceId := form.Get("InstanceId")
	networkInterfaceId := form.Get("NetworkInterfaceId")
	if instanceId == "" && networkInterfaceId == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain InstanceId or NetworkInterfaceId.",
			http.StatusBadRequest))
	}

	assocId, errCode := store.AssociateAddress(allocationId, instanceId, networkInterfaceId)
	switch errCode {
	case "":
		// success
	case "not_found":
		return xmlErr(service.NewAWSError("InvalidAllocationID.NotFound",
			"The allocation ID '"+allocationId+"' does not exist.",
			http.StatusBadRequest))
	case "already_associated":
		return xmlErr(service.NewAWSError("Resource.AlreadyAssociated",
			"The allocation '"+allocationId+"' is already associated.",
			http.StatusBadRequest))
	default:
		return xmlErr(service.NewAWSError("InvalidParameterValue", errCode, http.StatusBadRequest))
	}

	return xmlOK(&xmlAssociateAddressResponse{
		Xmlns:         ec2Xmlns,
		RequestID:     newUUID(),
		AssociationId: assocId,
	})
}

// ---- DisassociateAddress ----

type xmlDisassociateAddressResponse struct {
	XMLName   xml.Name `xml:"DisassociateAddressResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
	Return    bool     `xml:"return"`
}

func handleDisassociateAddress(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	assocId := form.Get("AssociationId")
	if assocId == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter AssociationId.",
			http.StatusBadRequest))
	}

	errCode := store.DisassociateAddress(assocId)
	switch errCode {
	case "":
		// success
	case "not_found":
		return xmlErr(service.NewAWSError("InvalidAssociationID.NotFound",
			"The association ID '"+assocId+"' does not exist.",
			http.StatusBadRequest))
	default:
		return xmlErr(service.NewAWSError("InvalidParameterValue", errCode, http.StatusBadRequest))
	}

	return xmlOK(&xmlDisassociateAddressResponse{
		Xmlns:     ec2Xmlns,
		RequestID: newUUID(),
		Return:    true,
	})
}

// ---- DescribeAddresses ----

type xmlDescribeAddressesResponse struct {
	XMLName     xml.Name     `xml:"DescribeAddressesResponse"`
	Xmlns       string       `xml:"xmlns,attr"`
	RequestID   string       `xml:"requestId"`
	AddressesSet []xmlAddress `xml:"addressesSet>item"`
}

func handleDescribeAddresses(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	ids := parseIndexedParam(form, "AllocationId")

	eips := store.ListAddresses(ids)
	items := make([]xmlAddress, 0, len(eips))
	for _, eip := range eips {
		items = append(items, toXMLAddress(eip))
	}

	return xmlOK(&xmlDescribeAddressesResponse{
		Xmlns:       ec2Xmlns,
		RequestID:   newUUID(),
		AddressesSet: items,
	})
}

// ============================================================
// Network Interface XML types
// ============================================================

type xmlNetworkInterface struct {
	NetworkInterfaceId string   `xml:"networkInterfaceId"`
	SubnetId           string   `xml:"subnetId"`
	VpcId              string   `xml:"vpcId"`
	PrivateIpAddress   string   `xml:"privateIpAddress"`
	Status             string   `xml:"status"`
	GroupSet           []xmlSGRef `xml:"groupSet>item,omitempty"`
}

type xmlSGRef struct {
	GroupId string `xml:"groupId"`
}

func toXMLNetworkInterface(eni *NetworkInterface) xmlNetworkInterface {
	x := xmlNetworkInterface{
		NetworkInterfaceId: eni.NetworkInterfaceId,
		SubnetId:           eni.SubnetId,
		VpcId:              eni.VpcId,
		PrivateIpAddress:   eni.PrivateIpAddress,
		Status:             eni.Status,
	}
	for _, gid := range eni.SecurityGroupIds {
		x.GroupSet = append(x.GroupSet, xmlSGRef{GroupId: gid})
	}
	return x
}

// ---- CreateNetworkInterface ----

type xmlCreateNetworkInterfaceResponse struct {
	XMLName          xml.Name            `xml:"CreateNetworkInterfaceResponse"`
	Xmlns            string              `xml:"xmlns,attr"`
	RequestID        string              `xml:"requestId"`
	NetworkInterface xmlNetworkInterface `xml:"networkInterface"`
}

func handleCreateNetworkInterface(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	subnetId := form.Get("SubnetId")
	if subnetId == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter SubnetId.",
			http.StatusBadRequest))
	}
	sgIds := parseIndexedParam(form, "SecurityGroupId")

	eni, errCode := store.CreateNetworkInterface(subnetId, sgIds)
	switch errCode {
	case "":
		// success
	case "subnet_not_found":
		return xmlErr(service.NewAWSError("InvalidSubnetID.NotFound",
			"The subnet ID '"+subnetId+"' does not exist.",
			http.StatusBadRequest))
	default:
		return xmlErr(service.NewAWSError("InvalidParameterValue", errCode, http.StatusBadRequest))
	}

	return xmlOK(&xmlCreateNetworkInterfaceResponse{
		Xmlns:            ec2Xmlns,
		RequestID:        newUUID(),
		NetworkInterface: toXMLNetworkInterface(eni),
	})
}

// ---- DescribeNetworkInterfaces ----

type xmlDescribeNetworkInterfacesResponse struct {
	XMLName              xml.Name              `xml:"DescribeNetworkInterfacesResponse"`
	Xmlns                string                `xml:"xmlns,attr"`
	RequestID            string                `xml:"requestId"`
	NetworkInterfaceSet  []xmlNetworkInterface `xml:"networkInterfaceSet>item"`
}

func handleDescribeNetworkInterfaces(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	ids := parseIndexedParam(form, "NetworkInterfaceId")
	filterSubnetId := extractFilterValue(form, "subnet-id")

	enis := store.ListNetworkInterfaces(ids, filterSubnetId)
	items := make([]xmlNetworkInterface, 0, len(enis))
	for _, eni := range enis {
		items = append(items, toXMLNetworkInterface(eni))
	}

	return xmlOK(&xmlDescribeNetworkInterfacesResponse{
		Xmlns:             ec2Xmlns,
		RequestID:         newUUID(),
		NetworkInterfaceSet: items,
	})
}

// ---- DeleteNetworkInterface ----

type xmlDeleteNetworkInterfaceResponse struct {
	XMLName   xml.Name `xml:"DeleteNetworkInterfaceResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
	Return    bool     `xml:"return"`
}

func handleDeleteNetworkInterface(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	eniId := form.Get("NetworkInterfaceId")
	if eniId == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter NetworkInterfaceId.",
			http.StatusBadRequest))
	}

	errCode := store.DeleteNetworkInterface(eniId)
	switch errCode {
	case "":
		// success
	case "not_found":
		return xmlErr(service.NewAWSError("InvalidNetworkInterfaceID.NotFound",
			"The network interface ID '"+eniId+"' does not exist.",
			http.StatusBadRequest))
	case "in_use":
		return xmlErr(service.NewAWSError("InvalidParameterValue",
			"The network interface '"+eniId+"' is currently in use.",
			http.StatusBadRequest))
	default:
		return xmlErr(service.NewAWSError("InvalidParameterValue", errCode, http.StatusBadRequest))
	}

	return xmlOK(&xmlDeleteNetworkInterfaceResponse{
		Xmlns:     ec2Xmlns,
		RequestID: newUUID(),
		Return:    true,
	})
}

// ============================================================
// Network ACL XML types
// ============================================================

type xmlNACLEntry struct {
	RuleNumber int    `xml:"ruleNumber"`
	Protocol   string `xml:"protocol"`
	RuleAction string `xml:"ruleAction"`
	Egress     bool   `xml:"egress"`
	CidrBlock  string `xml:"cidrBlock"`
}

type xmlNetworkAcl struct {
	NetworkAclId string         `xml:"networkAclId"`
	VpcId        string         `xml:"vpcId"`
	IsDefault    bool           `xml:"default"`
	EntrySet     []xmlNACLEntry `xml:"entrySet>item,omitempty"`
}

func toXMLNetworkAcl(acl *NetworkACL) xmlNetworkAcl {
	x := xmlNetworkAcl{
		NetworkAclId: acl.NetworkAclId,
		VpcId:        acl.VpcId,
		IsDefault:    acl.IsDefault,
	}
	for _, e := range acl.Entries {
		x.EntrySet = append(x.EntrySet, xmlNACLEntry{
			RuleNumber: e.RuleNumber,
			Protocol:   e.Protocol,
			RuleAction: e.RuleAction,
			Egress:     e.Egress,
			CidrBlock:  e.CidrBlock,
		})
	}
	return x
}

// ---- CreateNetworkAcl ----

type xmlCreateNetworkAclResponse struct {
	XMLName    xml.Name      `xml:"CreateNetworkAclResponse"`
	Xmlns      string        `xml:"xmlns,attr"`
	RequestID  string        `xml:"requestId"`
	NetworkAcl xmlNetworkAcl `xml:"networkAcl"`
}

func handleCreateNetworkAcl(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	vpcId := form.Get("VpcId")
	if vpcId == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter VpcId.",
			http.StatusBadRequest))
	}

	acl, errCode := store.CreateNetworkACL(vpcId)
	switch errCode {
	case "":
		// success
	case "vpc_not_found":
		return xmlErr(service.NewAWSError("InvalidVpcID.NotFound",
			"The vpc ID '"+vpcId+"' does not exist.",
			http.StatusBadRequest))
	default:
		return xmlErr(service.NewAWSError("InvalidParameterValue", errCode, http.StatusBadRequest))
	}

	return xmlOK(&xmlCreateNetworkAclResponse{
		Xmlns:      ec2Xmlns,
		RequestID:  newUUID(),
		NetworkAcl: toXMLNetworkAcl(acl),
	})
}

// ---- DescribeNetworkAcls ----

type xmlDescribeNetworkAclsResponse struct {
	XMLName       xml.Name        `xml:"DescribeNetworkAclsResponse"`
	Xmlns         string          `xml:"xmlns,attr"`
	RequestID     string          `xml:"requestId"`
	NetworkAclSet []xmlNetworkAcl `xml:"networkAclSet>item"`
}

func handleDescribeNetworkAcls(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	ids := parseIndexedParam(form, "NetworkAclId")
	filterVpcId := extractFilterValue(form, "vpc-id")

	acls := store.ListNetworkACLs(ids, filterVpcId)
	items := make([]xmlNetworkAcl, 0, len(acls))
	for _, acl := range acls {
		items = append(items, toXMLNetworkAcl(acl))
	}

	return xmlOK(&xmlDescribeNetworkAclsResponse{
		Xmlns:         ec2Xmlns,
		RequestID:     newUUID(),
		NetworkAclSet: items,
	})
}

// ---- DeleteNetworkAcl ----

type xmlDeleteNetworkAclResponse struct {
	XMLName   xml.Name `xml:"DeleteNetworkAclResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
	Return    bool     `xml:"return"`
}

func handleDeleteNetworkAcl(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	aclId := form.Get("NetworkAclId")
	if aclId == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter NetworkAclId.",
			http.StatusBadRequest))
	}

	errCode := store.DeleteNetworkACL(aclId)
	switch errCode {
	case "":
		// success
	case "not_found":
		return xmlErr(service.NewAWSError("InvalidNetworkAclID.NotFound",
			"The network ACL ID '"+aclId+"' does not exist.",
			http.StatusBadRequest))
	case "default_acl":
		return xmlErr(service.NewAWSError("InvalidParameterValue",
			"The default network ACL '"+aclId+"' cannot be deleted.",
			http.StatusBadRequest))
	default:
		return xmlErr(service.NewAWSError("InvalidParameterValue", errCode, http.StatusBadRequest))
	}

	return xmlOK(&xmlDeleteNetworkAclResponse{
		Xmlns:     ec2Xmlns,
		RequestID: newUUID(),
		Return:    true,
	})
}

// ---- CreateNetworkAclEntry ----

type xmlCreateNetworkAclEntryResponse struct {
	XMLName   xml.Name `xml:"CreateNetworkAclEntryResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
	Return    bool     `xml:"return"`
}

func handleCreateNetworkAclEntry(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	aclId := form.Get("NetworkAclId")
	if aclId == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter NetworkAclId.",
			http.StatusBadRequest))
	}
	ruleNumberStr := form.Get("RuleNumber")
	if ruleNumberStr == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter RuleNumber.",
			http.StatusBadRequest))
	}
	var ruleNumber int
	if _, err := fmt.Sscanf(ruleNumberStr, "%d", &ruleNumber); err != nil {
		return xmlErr(service.NewAWSError("InvalidParameterValue",
			"Invalid RuleNumber: "+ruleNumberStr,
			http.StatusBadRequest))
	}
	protocol := form.Get("Protocol")
	if protocol == "" {
		protocol = "-1"
	}
	ruleAction := form.Get("RuleAction")
	if ruleAction == "" {
		ruleAction = "allow"
	}
	egressStr := form.Get("Egress")
	egress := egressStr == "true"
	cidrBlock := form.Get("CidrBlock")
	if cidrBlock == "" {
		cidrBlock = "0.0.0.0/0"
	}

	entry := NACLEntry{
		RuleNumber: ruleNumber,
		Protocol:   protocol,
		RuleAction: ruleAction,
		Egress:     egress,
		CidrBlock:  cidrBlock,
	}

	errCode := store.CreateNetworkACLEntry(aclId, entry)
	switch errCode {
	case "":
		// success
	case "not_found":
		return xmlErr(service.NewAWSError("InvalidNetworkAclID.NotFound",
			"The network ACL ID '"+aclId+"' does not exist.",
			http.StatusBadRequest))
	case "duplicate_rule":
		return xmlErr(service.NewAWSError("NetworkAclEntryAlreadyExists",
			fmt.Sprintf("The rule number %d already exists.", ruleNumber),
			http.StatusBadRequest))
	default:
		return xmlErr(service.NewAWSError("InvalidParameterValue", errCode, http.StatusBadRequest))
	}

	return xmlOK(&xmlCreateNetworkAclEntryResponse{
		Xmlns:     ec2Xmlns,
		RequestID: newUUID(),
		Return:    true,
	})
}

// ---- DeleteNetworkAclEntry ----

type xmlDeleteNetworkAclEntryResponse struct {
	XMLName   xml.Name `xml:"DeleteNetworkAclEntryResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
	Return    bool     `xml:"return"`
}

func handleDeleteNetworkAclEntry(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	aclId := form.Get("NetworkAclId")
	if aclId == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter NetworkAclId.",
			http.StatusBadRequest))
	}
	ruleNumberStr := form.Get("RuleNumber")
	if ruleNumberStr == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter RuleNumber.",
			http.StatusBadRequest))
	}
	var ruleNumber int
	if _, err := fmt.Sscanf(ruleNumberStr, "%d", &ruleNumber); err != nil {
		return xmlErr(service.NewAWSError("InvalidParameterValue",
			"Invalid RuleNumber: "+ruleNumberStr,
			http.StatusBadRequest))
	}
	egressStr := form.Get("Egress")
	egress := egressStr == "true"

	errCode := store.DeleteNetworkACLEntry(aclId, ruleNumber, egress)
	switch errCode {
	case "":
		// success
	case "not_found":
		return xmlErr(service.NewAWSError("InvalidNetworkAclID.NotFound",
			"The network ACL ID '"+aclId+"' does not exist.",
			http.StatusBadRequest))
	case "entry_not_found":
		return xmlErr(service.NewAWSError("InvalidNetworkAclEntry.NotFound",
			fmt.Sprintf("The rule %d does not exist.", ruleNumber),
			http.StatusBadRequest))
	default:
		return xmlErr(service.NewAWSError("InvalidParameterValue", errCode, http.StatusBadRequest))
	}

	return xmlOK(&xmlDeleteNetworkAclEntryResponse{
		Xmlns:     ec2Xmlns,
		RequestID: newUUID(),
		Return:    true,
	})
}

// ============================================================
// VPC Endpoint XML types
// ============================================================

type xmlVpcEndpoint struct {
	VpcEndpointId   string `xml:"vpcEndpointId"`
	VpcId           string `xml:"vpcId"`
	ServiceName     string `xml:"serviceName"`
	VpcEndpointType string `xml:"vpcEndpointType"`
	State           string `xml:"state"`
}

func toXMLVpcEndpoint(ep *VPCEndpoint) xmlVpcEndpoint {
	return xmlVpcEndpoint{
		VpcEndpointId:   ep.VpcEndpointId,
		VpcId:           ep.VpcId,
		ServiceName:     ep.ServiceName,
		VpcEndpointType: ep.VpcEndpointType,
		State:           ep.State,
	}
}

// ---- CreateVpcEndpoint ----

type xmlCreateVpcEndpointResponse struct {
	XMLName     xml.Name       `xml:"CreateVpcEndpointResponse"`
	Xmlns       string         `xml:"xmlns,attr"`
	RequestID   string         `xml:"requestId"`
	VpcEndpoint xmlVpcEndpoint `xml:"vpcEndpoint"`
}

func handleCreateVpcEndpoint(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	vpcId := form.Get("VpcId")
	if vpcId == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter VpcId.",
			http.StatusBadRequest))
	}
	serviceName := form.Get("ServiceName")
	if serviceName == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter ServiceName.",
			http.StatusBadRequest))
	}
	epType := form.Get("VpcEndpointType")
	if epType == "" {
		epType = "Gateway"
	}

	ep, errCode := store.CreateVPCEndpoint(vpcId, serviceName, epType)
	switch errCode {
	case "":
		// success
	case "vpc_not_found":
		return xmlErr(service.NewAWSError("InvalidVpcID.NotFound",
			"The vpc ID '"+vpcId+"' does not exist.",
			http.StatusBadRequest))
	default:
		return xmlErr(service.NewAWSError("InvalidParameterValue", errCode, http.StatusBadRequest))
	}

	return xmlOK(&xmlCreateVpcEndpointResponse{
		Xmlns:       ec2Xmlns,
		RequestID:   newUUID(),
		VpcEndpoint: toXMLVpcEndpoint(ep),
	})
}

// ---- DescribeVpcEndpoints ----

type xmlDescribeVpcEndpointsResponse struct {
	XMLName         xml.Name         `xml:"DescribeVpcEndpointsResponse"`
	Xmlns           string           `xml:"xmlns,attr"`
	RequestID       string           `xml:"requestId"`
	VpcEndpointSet  []xmlVpcEndpoint `xml:"vpcEndpointSet>item"`
}

func handleDescribeVpcEndpoints(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	ids := parseIndexedParam(form, "VpcEndpointId")
	filterVpcId := extractFilterValue(form, "vpc-id")

	eps := store.ListVPCEndpoints(ids, filterVpcId)
	items := make([]xmlVpcEndpoint, 0, len(eps))
	for _, ep := range eps {
		items = append(items, toXMLVpcEndpoint(ep))
	}

	return xmlOK(&xmlDescribeVpcEndpointsResponse{
		Xmlns:          ec2Xmlns,
		RequestID:      newUUID(),
		VpcEndpointSet: items,
	})
}

// ---- DeleteVpcEndpoints ----

type xmlDeleteVpcEndpointsResponse struct {
	XMLName   xml.Name `xml:"DeleteVpcEndpointsResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
	Return    bool     `xml:"return"`
}

func handleDeleteVpcEndpoints(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	ids := parseIndexedParam(form, "VpcEndpointId")
	if len(ids) == 0 {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain at least one VpcEndpointId.",
			http.StatusBadRequest))
	}

	for _, id := range ids {
		errCode := store.DeleteVPCEndpoint(id)
		switch errCode {
		case "":
			// success
		case "not_found":
			return xmlErr(service.NewAWSError("InvalidVpcEndpointId.NotFound",
				"The vpc endpoint ID '"+id+"' does not exist.",
				http.StatusBadRequest))
		default:
			return xmlErr(service.NewAWSError("InvalidParameterValue", errCode, http.StatusBadRequest))
		}
	}

	return xmlOK(&xmlDeleteVpcEndpointsResponse{
		Xmlns:     ec2Xmlns,
		RequestID: newUUID(),
		Return:    true,
	})
}

// ============================================================
// VPC Peering Connection XML types
// ============================================================

type xmlPeeringStatus struct {
	Code    string `xml:"code"`
	Message string `xml:"message"`
}

type xmlVpcPeeringConnection struct {
	VpcPeeringConnectionId string           `xml:"vpcPeeringConnectionId"`
	RequesterVpcId         string           `xml:"requesterVpcInfo>vpcId"`
	AccepterVpcId          string           `xml:"accepterVpcInfo>vpcId"`
	Status                 xmlPeeringStatus `xml:"status"`
}

func toXMLVpcPeeringConnection(pcx *VPCPeeringConnection) xmlVpcPeeringConnection {
	return xmlVpcPeeringConnection{
		VpcPeeringConnectionId: pcx.PeeringConnectionId,
		RequesterVpcId:         pcx.RequesterVpcId,
		AccepterVpcId:          pcx.AccepterVpcId,
		Status:                 xmlPeeringStatus{Code: pcx.Status},
	}
}

// ---- CreateVpcPeeringConnection ----

type xmlCreateVpcPeeringConnectionResponse struct {
	XMLName              xml.Name                `xml:"CreateVpcPeeringConnectionResponse"`
	Xmlns                string                  `xml:"xmlns,attr"`
	RequestID            string                  `xml:"requestId"`
	VpcPeeringConnection xmlVpcPeeringConnection `xml:"vpcPeeringConnection"`
}

func handleCreateVpcPeeringConnection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	vpcId := form.Get("VpcId")
	if vpcId == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter VpcId.",
			http.StatusBadRequest))
	}
	peerVpcId := form.Get("PeerVpcId")
	if peerVpcId == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter PeerVpcId.",
			http.StatusBadRequest))
	}

	pcx, errCode := store.CreateVPCPeeringConnection(vpcId, peerVpcId)
	switch errCode {
	case "":
		// success
	case "requester_not_found":
		return xmlErr(service.NewAWSError("InvalidVpcID.NotFound",
			"The vpc ID '"+vpcId+"' does not exist.",
			http.StatusBadRequest))
	case "accepter_not_found":
		return xmlErr(service.NewAWSError("InvalidVpcID.NotFound",
			"The peer vpc ID '"+peerVpcId+"' does not exist.",
			http.StatusBadRequest))
	default:
		return xmlErr(service.NewAWSError("InvalidParameterValue", errCode, http.StatusBadRequest))
	}

	return xmlOK(&xmlCreateVpcPeeringConnectionResponse{
		Xmlns:                ec2Xmlns,
		RequestID:            newUUID(),
		VpcPeeringConnection: toXMLVpcPeeringConnection(pcx),
	})
}

// ---- AcceptVpcPeeringConnection ----

type xmlAcceptVpcPeeringConnectionResponse struct {
	XMLName              xml.Name                `xml:"AcceptVpcPeeringConnectionResponse"`
	Xmlns                string                  `xml:"xmlns,attr"`
	RequestID            string                  `xml:"requestId"`
	VpcPeeringConnection xmlVpcPeeringConnection `xml:"vpcPeeringConnection"`
}

func handleAcceptVpcPeeringConnection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	pcxId := form.Get("VpcPeeringConnectionId")
	if pcxId == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter VpcPeeringConnectionId.",
			http.StatusBadRequest))
	}

	pcx, errCode := store.AcceptVPCPeeringConnection(pcxId)
	switch errCode {
	case "":
		// success
	case "not_found":
		return xmlErr(service.NewAWSError("InvalidVpcPeeringConnectionID.NotFound",
			"The vpc peering connection ID '"+pcxId+"' does not exist.",
			http.StatusBadRequest))
	case "invalid_state":
		return xmlErr(service.NewAWSError("InvalidStateTransition",
			"The vpc peering connection '"+pcxId+"' is not in the pending-acceptance state.",
			http.StatusBadRequest))
	default:
		return xmlErr(service.NewAWSError("InvalidParameterValue", errCode, http.StatusBadRequest))
	}

	return xmlOK(&xmlAcceptVpcPeeringConnectionResponse{
		Xmlns:                ec2Xmlns,
		RequestID:            newUUID(),
		VpcPeeringConnection: toXMLVpcPeeringConnection(pcx),
	})
}

// ---- DeleteVpcPeeringConnection ----

type xmlDeleteVpcPeeringConnectionResponse struct {
	XMLName   xml.Name `xml:"DeleteVpcPeeringConnectionResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
	Return    bool     `xml:"return"`
}

func handleDeleteVpcPeeringConnection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	pcxId := form.Get("VpcPeeringConnectionId")
	if pcxId == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter VpcPeeringConnectionId.",
			http.StatusBadRequest))
	}

	errCode := store.DeleteVPCPeeringConnection(pcxId)
	switch errCode {
	case "":
		// success
	case "not_found":
		return xmlErr(service.NewAWSError("InvalidVpcPeeringConnectionID.NotFound",
			"The vpc peering connection ID '"+pcxId+"' does not exist.",
			http.StatusBadRequest))
	default:
		return xmlErr(service.NewAWSError("InvalidParameterValue", errCode, http.StatusBadRequest))
	}

	return xmlOK(&xmlDeleteVpcPeeringConnectionResponse{
		Xmlns:     ec2Xmlns,
		RequestID: newUUID(),
		Return:    true,
	})
}
