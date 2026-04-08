package ec2

import (
	"crypto/rand"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

const ec2Xmlns = "http://ec2.amazonaws.com/doc/2016-11-15/"

// ---- VPC XML types ----

type xmlVpc struct {
	VpcId              string `xml:"vpcId"`
	CidrBlock          string `xml:"cidrBlock"`
	State              string `xml:"state"`
	IsDefault          bool   `xml:"isDefault"`
	OwnerId            string `xml:"ownerId"`
	DhcpOptionsId      string `xml:"dhcpOptionsId"`
	EnableDnsSupport   bool   `xml:"enableDnsSupport"`
	EnableDnsHostnames bool   `xml:"enableDnsHostnames"`
}

func toXMLVpc(vpc *VPC) xmlVpc {
	return xmlVpc{
		VpcId:              vpc.VpcId,
		CidrBlock:          vpc.CidrBlock,
		State:              vpc.State,
		IsDefault:          vpc.IsDefault,
		OwnerId:            vpc.OwnerId,
		DhcpOptionsId:      vpc.DhcpOptionsId,
		EnableDnsSupport:   vpc.EnableDnsSupport,
		EnableDnsHostnames: vpc.EnableDnsHostnames,
	}
}

// ---- Subnet XML types ----

type xmlSubnet struct {
	SubnetId                string `xml:"subnetId"`
	VpcId                   string `xml:"vpcId"`
	CidrBlock               string `xml:"cidrBlock"`
	AvailabilityZone        string `xml:"availabilityZone"`
	State                   string `xml:"state"`
	AvailableIpAddressCount int    `xml:"availableIpAddressCount"`
	MapPublicIpOnLaunch     bool   `xml:"mapPublicIpOnLaunch"`
}

func toXMLSubnet(sub *Subnet) xmlSubnet {
	return xmlSubnet{
		SubnetId:                sub.SubnetId,
		VpcId:                   sub.VpcId,
		CidrBlock:               sub.CidrBlock,
		AvailabilityZone:        sub.AvailabilityZone,
		State:                   sub.State,
		AvailableIpAddressCount: sub.AvailableIpAddressCount,
		MapPublicIpOnLaunch:     sub.MapPublicIpOnLaunch,
	}
}

// ---- CreateVpc ----

type xmlCreateVpcResponse struct {
	XMLName   xml.Name `xml:"CreateVpcResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
	Vpc       xmlVpc   `xml:"vpc"`
}

func handleCreateVpc(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	cidrBlock := form.Get("CidrBlock")
	if cidrBlock == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter CidrBlock.",
			http.StatusBadRequest))
	}

	enableDnsSupport := true
	if v := form.Get("EnableDnsSupport"); v != "" {
		enableDnsSupport, _ = strconv.ParseBool(v)
	}
	enableDnsHostnames := false
	if v := form.Get("EnableDnsHostnames"); v != "" {
		enableDnsHostnames, _ = strconv.ParseBool(v)
	}

	vpc, err := store.CreateVPC(cidrBlock, enableDnsSupport, enableDnsHostnames)
	if err != nil {
		return xmlErr(service.NewAWSError("InvalidParameterValue",
			err.Error(), http.StatusBadRequest))
	}

	return xmlOK(&xmlCreateVpcResponse{
		Xmlns:     ec2Xmlns,
		RequestID: newUUID(),
		Vpc:       toXMLVpc(vpc),
	})
}

// ---- DescribeVpcs ----

type xmlDescribeVpcsResponse struct {
	XMLName   xml.Name `xml:"DescribeVpcsResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
	VpcSet    []xmlVpc `xml:"vpcSet>item"`
}

func handleDescribeVpcs(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)

	ids := parseIndexedParam(form, "VpcId")

	vpcs := store.ListVPCs(ids)

	xmlVpcs := make([]xmlVpc, 0, len(vpcs))
	for _, vpc := range vpcs {
		xmlVpcs = append(xmlVpcs, toXMLVpc(vpc))
	}

	return xmlOK(&xmlDescribeVpcsResponse{
		Xmlns:     ec2Xmlns,
		RequestID: newUUID(),
		VpcSet:    xmlVpcs,
	})
}

// ---- DeleteVpc ----

type xmlDeleteVpcResponse struct {
	XMLName   xml.Name `xml:"DeleteVpcResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
	Return    bool     `xml:"return"`
}

func handleDeleteVpc(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	vpcId := form.Get("VpcId")
	if vpcId == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter VpcId.",
			http.StatusBadRequest))
	}

	reason, ok := store.DeleteVPC(vpcId)
	if !ok {
		switch reason {
		case "dependency":
			return xmlErr(service.NewAWSError("DependencyViolation",
				"The vpc '"+vpcId+"' has dependencies and cannot be deleted.",
				http.StatusBadRequest))
		default:
			return xmlErr(service.NewAWSError("InvalidVpcID.NotFound",
				"The vpc ID '"+vpcId+"' does not exist.",
				http.StatusBadRequest))
		}
	}

	return xmlOK(&xmlDeleteVpcResponse{
		Xmlns:     ec2Xmlns,
		RequestID: newUUID(),
		Return:    true,
	})
}

// ---- ModifyVpcAttribute ----

type xmlModifyVpcAttributeResponse struct {
	XMLName   xml.Name `xml:"ModifyVpcAttributeResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
	Return    bool     `xml:"return"`
}

func handleModifyVpcAttribute(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	vpcId := form.Get("VpcId")
	if vpcId == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter VpcId.",
			http.StatusBadRequest))
	}

	var dnsSupport, dnsHostnames *bool
	if v := form.Get("EnableDnsSupport.Value"); v != "" {
		b, _ := strconv.ParseBool(v)
		dnsSupport = &b
	}
	if v := form.Get("EnableDnsHostnames.Value"); v != "" {
		b, _ := strconv.ParseBool(v)
		dnsHostnames = &b
	}

	if !store.ModifyVPCAttribute(vpcId, dnsSupport, dnsHostnames) {
		return xmlErr(service.NewAWSError("InvalidVpcID.NotFound",
			"The vpc ID '"+vpcId+"' does not exist.",
			http.StatusBadRequest))
	}

	return xmlOK(&xmlModifyVpcAttributeResponse{
		Xmlns:     ec2Xmlns,
		RequestID: newUUID(),
		Return:    true,
	})
}

// ---- CreateSubnet ----

type xmlCreateSubnetResponse struct {
	XMLName   xml.Name  `xml:"CreateSubnetResponse"`
	Xmlns     string    `xml:"xmlns,attr"`
	RequestID string    `xml:"requestId"`
	Subnet    xmlSubnet `xml:"subnet"`
}

func handleCreateSubnet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	vpcId := form.Get("VpcId")
	if vpcId == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter VpcId.",
			http.StatusBadRequest))
	}
	cidrBlock := form.Get("CidrBlock")
	if cidrBlock == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter CidrBlock.",
			http.StatusBadRequest))
	}
	az := form.Get("AvailabilityZone")

	sub, errCode := store.CreateSubnet(vpcId, cidrBlock, az)
	if errCode != "" {
		switch errCode {
		case "vpc_not_found":
			return xmlErr(service.NewAWSError("InvalidVpcID.NotFound",
				"The vpc ID '"+vpcId+"' does not exist.",
				http.StatusBadRequest))
		case "cidr_out_of_range":
			return xmlErr(service.NewAWSError("InvalidSubnet.Range",
				"The CIDR '"+cidrBlock+"' is invalid for VPC '"+vpcId+"'.",
				http.StatusBadRequest))
		default:
			return xmlErr(service.NewAWSError("InvalidParameterValue",
				"Invalid CIDR block: "+cidrBlock,
				http.StatusBadRequest))
		}
	}

	return xmlOK(&xmlCreateSubnetResponse{
		Xmlns:     ec2Xmlns,
		RequestID: newUUID(),
		Subnet:    toXMLSubnet(sub),
	})
}

// ---- DescribeSubnets ----

type xmlDescribeSubnetsResponse struct {
	XMLName   xml.Name    `xml:"DescribeSubnetsResponse"`
	Xmlns     string      `xml:"xmlns,attr"`
	RequestID string      `xml:"requestId"`
	SubnetSet []xmlSubnet `xml:"subnetSet>item"`
}

func handleDescribeSubnets(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)

	ids := parseIndexedParam(form, "SubnetId")
	vpcId := extractFilterValue(form, "vpc-id")

	subnets := store.ListSubnets(ids, vpcId)

	xmlSubnets := make([]xmlSubnet, 0, len(subnets))
	for _, sub := range subnets {
		xmlSubnets = append(xmlSubnets, toXMLSubnet(sub))
	}

	return xmlOK(&xmlDescribeSubnetsResponse{
		Xmlns:     ec2Xmlns,
		RequestID: newUUID(),
		SubnetSet: xmlSubnets,
	})
}

// ---- DeleteSubnet ----

type xmlDeleteSubnetResponse struct {
	XMLName   xml.Name `xml:"DeleteSubnetResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
	Return    bool     `xml:"return"`
}

func handleDeleteSubnet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	subnetId := form.Get("SubnetId")
	if subnetId == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter SubnetId.",
			http.StatusBadRequest))
	}

	if !store.DeleteSubnet(subnetId) {
		return xmlErr(service.NewAWSError("InvalidSubnetID.NotFound",
			"The subnet ID '"+subnetId+"' does not exist.",
			http.StatusBadRequest))
	}

	return xmlOK(&xmlDeleteSubnetResponse{
		Xmlns:     ec2Xmlns,
		RequestID: newUUID(),
		Return:    true,
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

// parseIndexedParam parses Param.N style indexed parameters (1-based).
func parseIndexedParam(form url.Values, prefix string) []string {
	var result []string
	for i := 1; ; i++ {
		v := form.Get(fmt.Sprintf("%s.%d", prefix, i))
		if v == "" {
			break
		}
		result = append(result, v)
	}
	return result
}

// extractFilterValue looks for EC2 Filter.N.Name / Filter.N.Value.1 style
// parameters and returns the first value for the given filter name.
func extractFilterValue(form url.Values, filterName string) string {
	for i := 1; ; i++ {
		name := form.Get(fmt.Sprintf("Filter.%d.Name", i))
		if name == "" {
			break
		}
		if strings.EqualFold(name, filterName) {
			return form.Get(fmt.Sprintf("Filter.%d.Value.1", i))
		}
	}
	return ""
}

// xmlOK wraps a response body in a 200 XML response.
func xmlOK(body any) (*service.Response, error) {
	data, err := xml.Marshal(body)
	if err != nil {
		return nil, err
	}
	return &service.Response{
		StatusCode:     http.StatusOK,
		RawBody:        data,
		RawContentType: "text/xml",
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
