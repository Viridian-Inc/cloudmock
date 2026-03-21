package ec2

import (
	"encoding/xml"
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// ============================================================
// Internet Gateway XML types
// ============================================================

type xmlIGWAttachment struct {
	VpcId string `xml:"vpcId"`
	State string `xml:"state"`
}

type xmlInternetGateway struct {
	InternetGatewayId string             `xml:"internetGatewayId"`
	AttachmentSet     []xmlIGWAttachment `xml:"attachmentSet>item,omitempty"`
}

func toXMLIGW(igw *InternetGateway) xmlInternetGateway {
	x := xmlInternetGateway{InternetGatewayId: igw.IgwId}
	for _, att := range igw.Attachments {
		x.AttachmentSet = append(x.AttachmentSet, xmlIGWAttachment{
			VpcId: att.VpcId,
			State: att.State,
		})
	}
	return x
}

// ---- CreateInternetGateway ----

type xmlCreateInternetGatewayResponse struct {
	XMLName         xml.Name           `xml:"CreateInternetGatewayResponse"`
	Xmlns           string             `xml:"xmlns,attr"`
	RequestID       string             `xml:"requestId"`
	InternetGateway xmlInternetGateway `xml:"internetGateway"`
}

func handleCreateInternetGateway(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	igw := store.CreateInternetGateway()
	return xmlOK(&xmlCreateInternetGatewayResponse{
		Xmlns:           ec2Xmlns,
		RequestID:       newUUID(),
		InternetGateway: toXMLIGW(igw),
	})
}

// ---- AttachInternetGateway ----

type xmlAttachInternetGatewayResponse struct {
	XMLName   xml.Name `xml:"AttachInternetGatewayResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
	Return    bool     `xml:"return"`
}

func handleAttachInternetGateway(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	igwId := form.Get("InternetGatewayId")
	if igwId == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter InternetGatewayId.",
			http.StatusBadRequest))
	}
	vpcId := form.Get("VpcId")
	if vpcId == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter VpcId.",
			http.StatusBadRequest))
	}

	errCode := store.AttachInternetGateway(igwId, vpcId)
	switch errCode {
	case "":
		// success
	case "igw_not_found":
		return xmlErr(service.NewAWSError("InvalidInternetGatewayID.NotFound",
			"The internet gateway '"+igwId+"' does not exist.",
			http.StatusBadRequest))
	case "vpc_not_found":
		return xmlErr(service.NewAWSError("InvalidVpcID.NotFound",
			"The vpc ID '"+vpcId+"' does not exist.",
			http.StatusBadRequest))
	case "already_attached":
		return xmlErr(service.NewAWSError("Resource.AlreadyAssociated",
			"The internet gateway '"+igwId+"' is already attached to a VPC.",
			http.StatusBadRequest))
	case "vpc_already_has_igw":
		return xmlErr(service.NewAWSError("Resource.AlreadyAssociated",
			"The vpc '"+vpcId+"' already has an internet gateway attached.",
			http.StatusBadRequest))
	default:
		return xmlErr(service.NewAWSError("InvalidParameterValue", errCode, http.StatusBadRequest))
	}

	return xmlOK(&xmlAttachInternetGatewayResponse{
		Xmlns:     ec2Xmlns,
		RequestID: newUUID(),
		Return:    true,
	})
}

// ---- DetachInternetGateway ----

type xmlDetachInternetGatewayResponse struct {
	XMLName   xml.Name `xml:"DetachInternetGatewayResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
	Return    bool     `xml:"return"`
}

func handleDetachInternetGateway(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	igwId := form.Get("InternetGatewayId")
	if igwId == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter InternetGatewayId.",
			http.StatusBadRequest))
	}
	vpcId := form.Get("VpcId")
	if vpcId == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter VpcId.",
			http.StatusBadRequest))
	}

	errCode := store.DetachInternetGateway(igwId, vpcId)
	switch errCode {
	case "":
		// success
	case "igw_not_found":
		return xmlErr(service.NewAWSError("InvalidInternetGatewayID.NotFound",
			"The internet gateway '"+igwId+"' does not exist.",
			http.StatusBadRequest))
	case "vpc_not_found":
		return xmlErr(service.NewAWSError("InvalidVpcID.NotFound",
			"The vpc ID '"+vpcId+"' does not exist.",
			http.StatusBadRequest))
	case "not_attached":
		return xmlErr(service.NewAWSError("Gateway.NotAttached",
			"The internet gateway '"+igwId+"' is not attached to vpc '"+vpcId+"'.",
			http.StatusBadRequest))
	default:
		return xmlErr(service.NewAWSError("InvalidParameterValue", errCode, http.StatusBadRequest))
	}

	return xmlOK(&xmlDetachInternetGatewayResponse{
		Xmlns:     ec2Xmlns,
		RequestID: newUUID(),
		Return:    true,
	})
}

// ---- DeleteInternetGateway ----

type xmlDeleteInternetGatewayResponse struct {
	XMLName   xml.Name `xml:"DeleteInternetGatewayResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
	Return    bool     `xml:"return"`
}

func handleDeleteInternetGateway(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	igwId := form.Get("InternetGatewayId")
	if igwId == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter InternetGatewayId.",
			http.StatusBadRequest))
	}

	errCode := store.DeleteInternetGateway(igwId)
	switch errCode {
	case "":
		// success
	case "not_found":
		return xmlErr(service.NewAWSError("InvalidInternetGatewayID.NotFound",
			"The internet gateway '"+igwId+"' does not exist.",
			http.StatusBadRequest))
	case "still_attached":
		return xmlErr(service.NewAWSError("DependencyViolation",
			"The internet gateway '"+igwId+"' has a dependency and cannot be deleted.",
			http.StatusBadRequest))
	default:
		return xmlErr(service.NewAWSError("InvalidParameterValue", errCode, http.StatusBadRequest))
	}

	return xmlOK(&xmlDeleteInternetGatewayResponse{
		Xmlns:     ec2Xmlns,
		RequestID: newUUID(),
		Return:    true,
	})
}

// ---- DescribeInternetGateways ----

type xmlDescribeInternetGatewaysResponse struct {
	XMLName            xml.Name             `xml:"DescribeInternetGatewaysResponse"`
	Xmlns              string               `xml:"xmlns,attr"`
	RequestID          string               `xml:"requestId"`
	InternetGatewaySet []xmlInternetGateway `xml:"internetGatewaySet>item"`
}

func handleDescribeInternetGateways(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	ids := parseIndexedParam(form, "InternetGatewayId")
	filterVpcId := extractFilterValue(form, "attachment.vpc-id")

	igws := store.ListInternetGateways(ids, filterVpcId)
	items := make([]xmlInternetGateway, 0, len(igws))
	for _, igw := range igws {
		items = append(items, toXMLIGW(igw))
	}

	return xmlOK(&xmlDescribeInternetGatewaysResponse{
		Xmlns:              ec2Xmlns,
		RequestID:          newUUID(),
		InternetGatewaySet: items,
	})
}

// ============================================================
// NAT Gateway XML types
// ============================================================

type xmlNatGateway struct {
	NatGatewayId string `xml:"natGatewayId"`
	SubnetId     string `xml:"subnetId"`
	VpcId        string `xml:"vpcId"`
	State        string `xml:"state"`
}

func toXMLNatGateway(nat *NatGateway) xmlNatGateway {
	return xmlNatGateway{
		NatGatewayId: nat.NatGatewayId,
		SubnetId:     nat.SubnetId,
		VpcId:        nat.VpcId,
		State:        nat.State,
	}
}

// ---- CreateNatGateway ----

type xmlCreateNatGatewayResponse struct {
	XMLName    xml.Name      `xml:"CreateNatGatewayResponse"`
	Xmlns      string        `xml:"xmlns,attr"`
	RequestID  string        `xml:"requestId"`
	NatGateway xmlNatGateway `xml:"natGateway"`
}

func handleCreateNatGateway(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	subnetId := form.Get("SubnetId")
	if subnetId == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter SubnetId.",
			http.StatusBadRequest))
	}
	allocationId := form.Get("AllocationId")
	if allocationId == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter AllocationId.",
			http.StatusBadRequest))
	}

	nat, errCode := store.CreateNatGateway(subnetId, allocationId)
	if errCode != "" {
		switch errCode {
		case "subnet_not_found":
			return xmlErr(service.NewAWSError("InvalidSubnetID.NotFound",
				"The subnet ID '"+subnetId+"' does not exist.",
				http.StatusBadRequest))
		default:
			return xmlErr(service.NewAWSError("InvalidParameterValue", errCode, http.StatusBadRequest))
		}
	}

	return xmlOK(&xmlCreateNatGatewayResponse{
		Xmlns:      ec2Xmlns,
		RequestID:  newUUID(),
		NatGateway: toXMLNatGateway(nat),
	})
}

// ---- DescribeNatGateways ----

type xmlDescribeNatGatewaysResponse struct {
	XMLName       xml.Name        `xml:"DescribeNatGatewaysResponse"`
	Xmlns         string          `xml:"xmlns,attr"`
	RequestID     string          `xml:"requestId"`
	NatGatewaySet []xmlNatGateway `xml:"natGatewaySet>item"`
}

func handleDescribeNatGateways(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	ids := parseIndexedParam(form, "NatGatewayId")
	filterSubnetId := extractFilterValue(form, "subnet-id")
	filterVpcId := extractFilterValue(form, "vpc-id")
	filterState := extractFilterValue(form, "state")

	nats := store.ListNatGateways(ids, filterSubnetId, filterVpcId, filterState)
	items := make([]xmlNatGateway, 0, len(nats))
	for _, nat := range nats {
		items = append(items, toXMLNatGateway(nat))
	}

	return xmlOK(&xmlDescribeNatGatewaysResponse{
		Xmlns:         ec2Xmlns,
		RequestID:     newUUID(),
		NatGatewaySet: items,
	})
}

// ---- DeleteNatGateway ----

type xmlDeleteNatGatewayResponse struct {
	XMLName      xml.Name      `xml:"DeleteNatGatewayResponse"`
	Xmlns        string        `xml:"xmlns,attr"`
	RequestID    string        `xml:"requestId"`
	NatGateway   xmlNatGateway `xml:"natGateway"`
}

func handleDeleteNatGateway(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	natGatewayId := form.Get("NatGatewayId")
	if natGatewayId == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter NatGatewayId.",
			http.StatusBadRequest))
	}

	errCode := store.DeleteNatGateway(natGatewayId)
	if errCode != "" {
		return xmlErr(service.NewAWSError("InvalidNatGatewayID.NotFound",
			"The nat gateway ID '"+natGatewayId+"' does not exist.",
			http.StatusBadRequest))
	}

	// Return the (now deleted) gateway state.
	nats := store.ListNatGateways([]string{natGatewayId}, "", "", "")
	var xnat xmlNatGateway
	if len(nats) > 0 {
		xnat = toXMLNatGateway(nats[0])
	}

	return xmlOK(&xmlDeleteNatGatewayResponse{
		Xmlns:      ec2Xmlns,
		RequestID:  newUUID(),
		NatGateway: xnat,
	})
}

// ============================================================
// Route Table XML types
// ============================================================

type xmlRoute struct {
	DestinationCidrBlock string `xml:"destinationCidrBlock"`
	GatewayId            string `xml:"gatewayId,omitempty"`
	NatGatewayId         string `xml:"natGatewayId,omitempty"`
	VpcEndpointId        string `xml:"vpcEndpointId,omitempty"`
	State                string `xml:"state"`
	Origin               string `xml:"origin"`
}

type xmlRTAssociation struct {
	RouteTableAssociationId string `xml:"routeTableAssociationId"`
	RouteTableId            string `xml:"routeTableId"`
	SubnetId                string `xml:"subnetId,omitempty"`
	Main                    bool   `xml:"main"`
}

type xmlRouteTable struct {
	RouteTableId   string             `xml:"routeTableId"`
	VpcId          string             `xml:"vpcId"`
	RouteSet       []xmlRoute         `xml:"routeSet>item,omitempty"`
	AssociationSet []xmlRTAssociation `xml:"associationSet>item,omitempty"`
}

func toXMLRouteTable(rt *RouteTable, assocs []*RouteTableAssociation) xmlRouteTable {
	x := xmlRouteTable{
		RouteTableId: rt.RouteTableId,
		VpcId:        rt.VpcId,
	}
	for _, r := range rt.Routes {
		x.RouteSet = append(x.RouteSet, xmlRoute{
			DestinationCidrBlock: r.DestinationCidrBlock,
			GatewayId:            r.GatewayId,
			NatGatewayId:         r.NatGatewayId,
			VpcEndpointId:        r.VpcEndpointId,
			State:                r.State,
			Origin:               r.Origin,
		})
	}
	for _, a := range assocs {
		x.AssociationSet = append(x.AssociationSet, xmlRTAssociation{
			RouteTableAssociationId: a.AssociationId,
			RouteTableId:            a.RouteTableId,
			SubnetId:                a.SubnetId,
			Main:                    a.Main,
		})
	}
	return x
}

// ---- CreateRouteTable ----

type xmlCreateRouteTableResponse struct {
	XMLName    xml.Name      `xml:"CreateRouteTableResponse"`
	Xmlns      string        `xml:"xmlns,attr"`
	RequestID  string        `xml:"requestId"`
	RouteTable xmlRouteTable `xml:"routeTable"`
}

func handleCreateRouteTable(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	vpcId := form.Get("VpcId")
	if vpcId == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter VpcId.",
			http.StatusBadRequest))
	}

	rt, errCode := store.CreateRouteTable(vpcId)
	if errCode != "" {
		return xmlErr(service.NewAWSError("InvalidVpcID.NotFound",
			"The vpc ID '"+vpcId+"' does not exist.",
			http.StatusBadRequest))
	}

	return xmlOK(&xmlCreateRouteTableResponse{
		Xmlns:      ec2Xmlns,
		RequestID:  newUUID(),
		RouteTable: toXMLRouteTable(rt, nil),
	})
}

// ---- DescribeRouteTables ----

type xmlDescribeRouteTablesResponse struct {
	XMLName       xml.Name        `xml:"DescribeRouteTablesResponse"`
	Xmlns         string          `xml:"xmlns,attr"`
	RequestID     string          `xml:"requestId"`
	RouteTableSet []xmlRouteTable `xml:"routeTableSet>item"`
}

func handleDescribeRouteTables(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	ids := parseIndexedParam(form, "RouteTableId")
	filterVpcId := extractFilterValue(form, "vpc-id")

	rts := store.ListRouteTables(ids, filterVpcId)
	items := make([]xmlRouteTable, 0, len(rts))
	for _, rt := range rts {
		assocs := store.ListRouteTableAssociations(rt.RouteTableId)
		items = append(items, toXMLRouteTable(rt, assocs))
	}

	return xmlOK(&xmlDescribeRouteTablesResponse{
		Xmlns:         ec2Xmlns,
		RequestID:     newUUID(),
		RouteTableSet: items,
	})
}

// ---- DeleteRouteTable ----

type xmlDeleteRouteTableResponse struct {
	XMLName   xml.Name `xml:"DeleteRouteTableResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
	Return    bool     `xml:"return"`
}

func handleDeleteRouteTable(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	rtbId := form.Get("RouteTableId")
	if rtbId == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter RouteTableId.",
			http.StatusBadRequest))
	}

	errCode := store.DeleteRouteTable(rtbId)
	switch errCode {
	case "":
		// success
	case "not_found":
		return xmlErr(service.NewAWSError("InvalidRouteTableID.NotFound",
			"The route table ID '"+rtbId+"' does not exist.",
			http.StatusBadRequest))
	case "main_table":
		return xmlErr(service.NewAWSError("DependencyViolation",
			"The route table '"+rtbId+"' is the main route table and cannot be deleted.",
			http.StatusBadRequest))
	case "has_associations":
		return xmlErr(service.NewAWSError("DependencyViolation",
			"The route table '"+rtbId+"' has dependencies and cannot be deleted.",
			http.StatusBadRequest))
	default:
		return xmlErr(service.NewAWSError("InvalidParameterValue", errCode, http.StatusBadRequest))
	}

	return xmlOK(&xmlDeleteRouteTableResponse{
		Xmlns:     ec2Xmlns,
		RequestID: newUUID(),
		Return:    true,
	})
}

// ---- CreateRoute ----

type xmlCreateRouteResponse struct {
	XMLName   xml.Name `xml:"CreateRouteResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
	Return    bool     `xml:"return"`
}

func handleCreateRoute(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	rtbId := form.Get("RouteTableId")
	if rtbId == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter RouteTableId.",
			http.StatusBadRequest))
	}
	destCidr := form.Get("DestinationCidrBlock")
	if destCidr == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter DestinationCidrBlock.",
			http.StatusBadRequest))
	}

	gatewayId := form.Get("GatewayId")
	natGatewayId := form.Get("NatGatewayId")
	vpcEndpointId := form.Get("VpcEndpointId")

	if gatewayId == "" && natGatewayId == "" && vpcEndpointId == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"A target must be specified: GatewayId, NatGatewayId, or VpcEndpointId.",
			http.StatusBadRequest))
	}

	errCode := store.CreateRoute(rtbId, destCidr, gatewayId, natGatewayId, vpcEndpointId)
	switch errCode {
	case "":
		// success
	case "rtb_not_found":
		return xmlErr(service.NewAWSError("InvalidRouteTableID.NotFound",
			"The route table ID '"+rtbId+"' does not exist.",
			http.StatusBadRequest))
	case "route_already_exists":
		return xmlErr(service.NewAWSError("RouteAlreadyExists",
			"The route identified by "+destCidr+" already exists.",
			http.StatusBadRequest))
	case "gateway_not_found":
		return xmlErr(service.NewAWSError("InvalidGatewayID.NotFound",
			"The gateway ID '"+gatewayId+"' does not exist.",
			http.StatusBadRequest))
	case "nat_not_found":
		return xmlErr(service.NewAWSError("InvalidNatGatewayID.NotFound",
			"The nat gateway ID '"+natGatewayId+"' does not exist.",
			http.StatusBadRequest))
	default:
		return xmlErr(service.NewAWSError("InvalidParameterValue", errCode, http.StatusBadRequest))
	}

	return xmlOK(&xmlCreateRouteResponse{
		Xmlns:     ec2Xmlns,
		RequestID: newUUID(),
		Return:    true,
	})
}

// ---- DeleteRoute ----

type xmlDeleteRouteResponse struct {
	XMLName   xml.Name `xml:"DeleteRouteResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
	Return    bool     `xml:"return"`
}

func handleDeleteRoute(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	rtbId := form.Get("RouteTableId")
	if rtbId == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter RouteTableId.",
			http.StatusBadRequest))
	}
	destCidr := form.Get("DestinationCidrBlock")
	if destCidr == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter DestinationCidrBlock.",
			http.StatusBadRequest))
	}

	errCode := store.DeleteRoute(rtbId, destCidr)
	switch errCode {
	case "":
		// success
	case "rtb_not_found":
		return xmlErr(service.NewAWSError("InvalidRouteTableID.NotFound",
			"The route table ID '"+rtbId+"' does not exist.",
			http.StatusBadRequest))
	case "local_route":
		return xmlErr(service.NewAWSError("InvalidParameterValue",
			"The route with destination "+destCidr+" is a local route and cannot be deleted.",
			http.StatusBadRequest))
	case "route_not_found":
		return xmlErr(service.NewAWSError("InvalidRoute.NotFound",
			"The route identified by "+destCidr+" does not exist.",
			http.StatusBadRequest))
	default:
		return xmlErr(service.NewAWSError("InvalidParameterValue", errCode, http.StatusBadRequest))
	}

	return xmlOK(&xmlDeleteRouteResponse{
		Xmlns:     ec2Xmlns,
		RequestID: newUUID(),
		Return:    true,
	})
}

// ---- ReplaceRoute ----

type xmlReplaceRouteResponse struct {
	XMLName   xml.Name `xml:"ReplaceRouteResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
	Return    bool     `xml:"return"`
}

func handleReplaceRoute(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	rtbId := form.Get("RouteTableId")
	if rtbId == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter RouteTableId.",
			http.StatusBadRequest))
	}
	destCidr := form.Get("DestinationCidrBlock")
	if destCidr == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter DestinationCidrBlock.",
			http.StatusBadRequest))
	}

	gatewayId := form.Get("GatewayId")
	natGatewayId := form.Get("NatGatewayId")
	vpcEndpointId := form.Get("VpcEndpointId")

	if gatewayId == "" && natGatewayId == "" && vpcEndpointId == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"A target must be specified: GatewayId, NatGatewayId, or VpcEndpointId.",
			http.StatusBadRequest))
	}

	errCode := store.ReplaceRoute(rtbId, destCidr, gatewayId, natGatewayId, vpcEndpointId)
	switch errCode {
	case "":
		// success
	case "rtb_not_found":
		return xmlErr(service.NewAWSError("InvalidRouteTableID.NotFound",
			"The route table ID '"+rtbId+"' does not exist.",
			http.StatusBadRequest))
	case "route_not_found":
		return xmlErr(service.NewAWSError("InvalidRoute.NotFound",
			"The route identified by "+destCidr+" does not exist.",
			http.StatusBadRequest))
	case "gateway_not_found":
		return xmlErr(service.NewAWSError("InvalidGatewayID.NotFound",
			"The gateway ID '"+gatewayId+"' does not exist.",
			http.StatusBadRequest))
	case "nat_not_found":
		return xmlErr(service.NewAWSError("InvalidNatGatewayID.NotFound",
			"The nat gateway ID '"+natGatewayId+"' does not exist.",
			http.StatusBadRequest))
	default:
		return xmlErr(service.NewAWSError("InvalidParameterValue", errCode, http.StatusBadRequest))
	}

	return xmlOK(&xmlReplaceRouteResponse{
		Xmlns:     ec2Xmlns,
		RequestID: newUUID(),
		Return:    true,
	})
}

// ---- AssociateRouteTable ----

type xmlAssociateRouteTableResponse struct {
	XMLName       xml.Name `xml:"AssociateRouteTableResponse"`
	Xmlns         string   `xml:"xmlns,attr"`
	RequestID     string   `xml:"requestId"`
	AssociationId string   `xml:"associationId"`
}

func handleAssociateRouteTable(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	rtbId := form.Get("RouteTableId")
	if rtbId == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter RouteTableId.",
			http.StatusBadRequest))
	}
	subnetId := form.Get("SubnetId")
	if subnetId == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter SubnetId.",
			http.StatusBadRequest))
	}

	assocId, errCode := store.AssociateRouteTable(rtbId, subnetId)
	switch errCode {
	case "":
		// success
	case "rtb_not_found":
		return xmlErr(service.NewAWSError("InvalidRouteTableID.NotFound",
			"The route table ID '"+rtbId+"' does not exist.",
			http.StatusBadRequest))
	case "subnet_not_found":
		return xmlErr(service.NewAWSError("InvalidSubnetID.NotFound",
			"The subnet ID '"+subnetId+"' does not exist.",
			http.StatusBadRequest))
	case "already_associated":
		return xmlErr(service.NewAWSError("Resource.AlreadyAssociated",
			"The subnet '"+subnetId+"' is already associated with a route table.",
			http.StatusBadRequest))
	default:
		return xmlErr(service.NewAWSError("InvalidParameterValue", errCode, http.StatusBadRequest))
	}

	return xmlOK(&xmlAssociateRouteTableResponse{
		Xmlns:         ec2Xmlns,
		RequestID:     newUUID(),
		AssociationId: assocId,
	})
}

// ---- DisassociateRouteTable ----

type xmlDisassociateRouteTableResponse struct {
	XMLName   xml.Name `xml:"DisassociateRouteTableResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
	Return    bool     `xml:"return"`
}

func handleDisassociateRouteTable(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	assocId := form.Get("AssociationId")
	if assocId == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter AssociationId.",
			http.StatusBadRequest))
	}

	errCode := store.DisassociateRouteTable(assocId)
	switch errCode {
	case "":
		// success
	case "not_found":
		return xmlErr(service.NewAWSError("InvalidAssociationID.NotFound",
			"The association ID '"+assocId+"' does not exist.",
			http.StatusBadRequest))
	case "main_association":
		return xmlErr(service.NewAWSError("InvalidParameterValue",
			"The association '"+assocId+"' is the main route table association and cannot be removed.",
			http.StatusBadRequest))
	default:
		return xmlErr(service.NewAWSError("InvalidParameterValue", errCode, http.StatusBadRequest))
	}

	return xmlOK(&xmlDisassociateRouteTableResponse{
		Xmlns:     ec2Xmlns,
		RequestID: newUUID(),
		Return:    true,
	})
}
