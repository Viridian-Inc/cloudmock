package ec2

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

// VPC represents an EC2 Virtual Private Cloud.
type VPC struct {
	VpcId              string
	CidrBlock          string
	State              string
	IsDefault          bool
	OwnerId            string
	EnableDnsSupport   bool
	EnableDnsHostnames bool
	DhcpOptionsId      string
	Tags               map[string]string
}

// Subnet represents an EC2 subnet within a VPC.
type Subnet struct {
	SubnetId                string
	VpcId                   string
	CidrBlock               string
	AvailabilityZone        string
	State                   string
	AvailableIpAddressCount int
	MapPublicIpOnLaunch     bool
	Tags                    map[string]string
}

// RouteTable represents a VPC route table (auto-created with VPC).
type RouteTable struct {
	RouteTableId string
	VpcId        string
	IsMain       bool
	Routes       []Route
	Tags         map[string]string
}

// Route represents a single entry in a route table.
type Route struct {
	DestinationCidrBlock string
	GatewayId            string // igw-*, local, or vpce-*
	NatGatewayId         string
	VpcEndpointId        string
	State                string // active
	Origin               string // CreateRoute or CreateRouteTable
}

// RouteTableAssociation links a subnet to a route table.
type RouteTableAssociation struct {
	AssociationId string
	RouteTableId  string
	SubnetId      string
	Main          bool
}

// InternetGateway represents an EC2 internet gateway.
type InternetGateway struct {
	IgwId       string
	Attachments []IGWAttachment
	Tags        map[string]string
}

// IGWAttachment records a VPC attached to an internet gateway.
type IGWAttachment struct {
	VpcId string
	State string // attached
}

// NatGateway represents an EC2 NAT gateway.
type NatGateway struct {
	NatGatewayId string
	SubnetId     string
	VpcId        string
	AllocationId string
	State        string // available, deleted
	Tags         map[string]string
}

// SGRule represents a single inbound or outbound security group rule.
type SGRule struct {
	IpProtocol string   // tcp, udp, icmp, -1 (all)
	FromPort   int
	ToPort     int
	CidrBlocks []string // CIDR ranges (e.g. "0.0.0.0/0")
	GroupIds   []string // referenced security group IDs
}

// SecurityGroup represents a VPC security group (auto-created with VPC).
type SecurityGroup struct {
	GroupId      string
	GroupName    string
	Description  string
	VpcId        string
	IngressRules []SGRule
	EgressRules  []SGRule
	Tags         map[string]string
}

// ElasticIP represents an allocated Elastic IP address.
type ElasticIP struct {
	AllocationId       string
	PublicIp           string
	AssociationId      string
	InstanceId         string
	NetworkInterfaceId string
	Tags               map[string]string
}

// NetworkInterface represents an EC2 elastic network interface.
type NetworkInterface struct {
	NetworkInterfaceId string
	SubnetId           string
	VpcId              string
	PrivateIpAddress   string
	SecurityGroupIds   []string
	Status             string // available, in-use
	Tags               map[string]string
}

// NetworkACL represents a VPC network ACL (auto-created with VPC).
type NetworkACL struct {
	NetworkAclId string
	VpcId        string
	Entries      []NACLEntry
	IsDefault    bool
	Tags         map[string]string
}

// NACLEntry represents a single rule in a Network ACL.
type NACLEntry struct {
	RuleNumber int
	Protocol   string
	RuleAction string // allow, deny
	Egress     bool
	CidrBlock  string
}

// VPCEndpoint represents a VPC endpoint.
type VPCEndpoint struct {
	VpcEndpointId   string
	VpcId           string
	ServiceName     string
	VpcEndpointType string
	State           string
	Tags            map[string]string
}

// VPCPeeringConnection represents a VPC peering connection.
type VPCPeeringConnection struct {
	PeeringConnectionId string
	RequesterVpcId      string
	AccepterVpcId       string
	Status              string // pending-acceptance, active, deleted
	Tags                map[string]string
}

// Instance represents an EC2 instance.
type Instance struct {
	InstanceId       string
	ImageId          string
	InstanceType     string
	SubnetId         string
	VpcId            string
	SecurityGroupIds []string
	KeyName          string
	State            string // pending, running, stopping, stopped, shutting-down, terminated
	LaunchTime       time.Time
	PrivateIpAddress string
	Tags             map[string]string
}

// Store manages all EC2 resources.
type Store struct {
	mu                      sync.RWMutex
	vpcs                    map[string]*VPC
	subnets                 map[string]*Subnet
	routeTables             map[string]*RouteTable
	routeTableAssociations  map[string]*RouteTableAssociation
	securityGroups          map[string]*SecurityGroup
	networkACLs             map[string]*NetworkACL
	internetGateways        map[string]*InternetGateway
	natGateways             map[string]*NatGateway
	elasticIPs              map[string]*ElasticIP              // keyed by AllocationId
	eipAssociations         map[string]string                  // AssociationId -> AllocationId
	networkInterfaces       map[string]*NetworkInterface
	vpcEndpoints            map[string]*VPCEndpoint
	vpcPeeringConnections   map[string]*VPCPeeringConnection
	instances               map[string]*Instance
	tags                    map[string]map[string]string // resourceId -> key -> value
	subnetIPCounters        map[string]uint32            // subnetId -> next host offset
	accountID               string
	region                  string
}

// NewStore returns a new Store for the given account and region.
func NewStore(accountID, region string) *Store {
	return &Store{
		vpcs:                    make(map[string]*VPC),
		subnets:                 make(map[string]*Subnet),
		routeTables:             make(map[string]*RouteTable),
		routeTableAssociations:  make(map[string]*RouteTableAssociation),
		securityGroups:          make(map[string]*SecurityGroup),
		networkACLs:             make(map[string]*NetworkACL),
		internetGateways:        make(map[string]*InternetGateway),
		natGateways:             make(map[string]*NatGateway),
		elasticIPs:              make(map[string]*ElasticIP),
		eipAssociations:         make(map[string]string),
		networkInterfaces:       make(map[string]*NetworkInterface),
		vpcEndpoints:            make(map[string]*VPCEndpoint),
		vpcPeeringConnections:   make(map[string]*VPCPeeringConnection),
		instances:               make(map[string]*Instance),
		tags:                    make(map[string]map[string]string),
		subnetIPCounters:        make(map[string]uint32),
		accountID:               accountID,
		region:                  region,
	}
}

// ---- VPC operations ----

// CreateVPC creates a new VPC with the given CIDR block and also creates the
// default route table, security group, and network ACL for the VPC.
func (s *Store) CreateVPC(cidrBlock string, enableDnsSupport, enableDnsHostnames bool) (*VPC, error) {
	// Validate CIDR.
	_, _, err := net.ParseCIDR(cidrBlock)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR block: %s", cidrBlock)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	vpc := &VPC{
		VpcId:              genID("vpc-"),
		CidrBlock:          cidrBlock,
		State:              "available",
		IsDefault:          false,
		OwnerId:            s.accountID,
		EnableDnsSupport:   enableDnsSupport,
		EnableDnsHostnames: enableDnsHostnames,
		DhcpOptionsId:      genID("dopt-"),
		Tags:               make(map[string]string),
	}
	s.vpcs[vpc.VpcId] = vpc

	// Auto-create default route table with local route.
	rt := &RouteTable{
		RouteTableId: genID("rtb-"),
		VpcId:        vpc.VpcId,
		IsMain:       true,
		Routes: []Route{
			{
				DestinationCidrBlock: cidrBlock,
				GatewayId:            "local",
				State:                "active",
				Origin:               "CreateRouteTable",
			},
		},
		Tags: make(map[string]string),
	}
	s.routeTables[rt.RouteTableId] = rt

	// Auto-create default security group with default egress rule.
	sg := &SecurityGroup{
		GroupId:     genID("sg-"),
		GroupName:   "default",
		Description: "default VPC security group",
		VpcId:       vpc.VpcId,
		IngressRules: []SGRule{},
		EgressRules: []SGRule{
			{IpProtocol: "-1", FromPort: 0, ToPort: 0, CidrBlocks: []string{"0.0.0.0/0"}},
		},
		Tags: make(map[string]string),
	}
	s.securityGroups[sg.GroupId] = sg

	// Auto-create default network ACL with allow-all inbound/outbound.
	nacl := &NetworkACL{
		NetworkAclId: genID("acl-"),
		VpcId:        vpc.VpcId,
		IsDefault:    true,
		Entries: []NACLEntry{
			{RuleNumber: 100, Protocol: "-1", RuleAction: "allow", Egress: false, CidrBlock: "0.0.0.0/0"},
			{RuleNumber: 32767, Protocol: "-1", RuleAction: "deny", Egress: false, CidrBlock: "0.0.0.0/0"},
			{RuleNumber: 100, Protocol: "-1", RuleAction: "allow", Egress: true, CidrBlock: "0.0.0.0/0"},
			{RuleNumber: 32767, Protocol: "-1", RuleAction: "deny", Egress: true, CidrBlock: "0.0.0.0/0"},
		},
		Tags: make(map[string]string),
	}
	s.networkACLs[nacl.NetworkAclId] = nacl

	return vpc, nil
}

// GetVPC returns a VPC by ID.
func (s *Store) GetVPC(vpcId string) (*VPC, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	vpc, ok := s.vpcs[vpcId]
	return vpc, ok
}

// ListVPCs returns all VPCs, optionally filtered by a list of IDs.
func (s *Store) ListVPCs(ids []string) []*VPC {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(ids) == 0 {
		result := make([]*VPC, 0, len(s.vpcs))
		for _, vpc := range s.vpcs {
			result = append(result, vpc)
		}
		return result
	}

	idSet := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		idSet[id] = struct{}{}
	}
	result := make([]*VPC, 0, len(ids))
	for _, vpc := range s.vpcs {
		if _, ok := idSet[vpc.VpcId]; ok {
			result = append(result, vpc)
		}
	}
	return result
}

// DeleteVPC removes a VPC and its associated default resources. Returns an error
// string if the VPC has subnets (DependencyViolation) or is not found.
func (s *Store) DeleteVPC(vpcId string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.vpcs[vpcId]; !ok {
		return "not_found", false
	}

	// Check for dependent subnets.
	for _, sub := range s.subnets {
		if sub.VpcId == vpcId {
			return "dependency", false
		}
	}

	delete(s.vpcs, vpcId)

	// Clean up associated resources.
	for id, rt := range s.routeTables {
		if rt.VpcId == vpcId {
			delete(s.routeTables, id)
		}
	}
	for id, assoc := range s.routeTableAssociations {
		rt, ok := s.routeTables[assoc.RouteTableId]
		if ok && rt.VpcId == vpcId {
			delete(s.routeTableAssociations, id)
		}
	}
	for id, sg := range s.securityGroups {
		if sg.VpcId == vpcId {
			delete(s.securityGroups, id)
		}
	}
	for id, nacl := range s.networkACLs {
		if nacl.VpcId == vpcId {
			delete(s.networkACLs, id)
		}
	}

	return "", true
}

// ModifyVPCAttribute updates DNS support or DNS hostnames on a VPC.
func (s *Store) ModifyVPCAttribute(vpcId string, dnsSupport, dnsHostnames *bool) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	vpc, ok := s.vpcs[vpcId]
	if !ok {
		return false
	}
	if dnsSupport != nil {
		vpc.EnableDnsSupport = *dnsSupport
	}
	if dnsHostnames != nil {
		vpc.EnableDnsHostnames = *dnsHostnames
	}
	return true
}

// ---- Subnet operations ----

// CreateSubnet creates a new subnet within a VPC. Validates that the subnet CIDR
// falls within the VPC CIDR.
func (s *Store) CreateSubnet(vpcId, cidrBlock, availabilityZone string) (*Subnet, string) {
	// Validate CIDR format.
	_, subnetNet, err := net.ParseCIDR(cidrBlock)
	if err != nil {
		return nil, "invalid_cidr"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	vpc, ok := s.vpcs[vpcId]
	if !ok {
		return nil, "vpc_not_found"
	}

	// Validate subnet CIDR falls within VPC CIDR.
	_, vpcNet, _ := net.ParseCIDR(vpc.CidrBlock)
	if !cidrContains(vpcNet, subnetNet) {
		return nil, "cidr_out_of_range"
	}

	if availabilityZone == "" {
		availabilityZone = s.region + "a"
	}

	// Calculate available IPs: 2^(32-prefix) - 5 (AWS reserves 5 addresses).
	ones, bits := subnetNet.Mask.Size()
	totalIPs := 1 << (bits - ones)
	availableIPs := totalIPs - 5
	if availableIPs < 0 {
		availableIPs = 0
	}

	sub := &Subnet{
		SubnetId:                genID("subnet-"),
		VpcId:                   vpcId,
		CidrBlock:               cidrBlock,
		AvailabilityZone:        availabilityZone,
		State:                   "available",
		AvailableIpAddressCount: availableIPs,
		MapPublicIpOnLaunch:     false,
		Tags:                    make(map[string]string),
	}
	s.subnets[sub.SubnetId] = sub
	return sub, ""
}

// GetSubnet returns a subnet by ID.
func (s *Store) GetSubnet(subnetId string) (*Subnet, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sub, ok := s.subnets[subnetId]
	return sub, ok
}

// ListSubnets returns subnets, optionally filtered by subnet IDs and/or VPC ID.
func (s *Store) ListSubnets(ids []string, vpcId string) []*Subnet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var idSet map[string]struct{}
	if len(ids) > 0 {
		idSet = make(map[string]struct{}, len(ids))
		for _, id := range ids {
			idSet[id] = struct{}{}
		}
	}

	result := make([]*Subnet, 0)
	for _, sub := range s.subnets {
		if idSet != nil {
			if _, ok := idSet[sub.SubnetId]; !ok {
				continue
			}
		}
		if vpcId != "" && sub.VpcId != vpcId {
			continue
		}
		result = append(result, sub)
	}
	return result
}

// DeleteSubnet removes a subnet. Returns false if not found.
func (s *Store) DeleteSubnet(subnetId string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.subnets[subnetId]; !ok {
		return false
	}
	delete(s.subnets, subnetId)
	return true
}

// ---- SecurityGroup operations ----

// CreateSecurityGroup creates a new security group in the given VPC. Returns the
// new group and an error code string (empty on success).
func (s *Store) CreateSecurityGroup(groupName, description, vpcId string) (*SecurityGroup, string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.vpcs[vpcId]; !ok {
		return nil, "vpc_not_found"
	}

	// Ensure unique name within VPC.
	for _, sg := range s.securityGroups {
		if sg.VpcId == vpcId && sg.GroupName == groupName {
			return nil, "duplicate_name"
		}
	}

	sg := &SecurityGroup{
		GroupId:     genID("sg-"),
		GroupName:   groupName,
		Description: description,
		VpcId:       vpcId,
		IngressRules: []SGRule{},
		EgressRules: []SGRule{
			{IpProtocol: "-1", FromPort: 0, ToPort: 0, CidrBlocks: []string{"0.0.0.0/0"}},
		},
		Tags: make(map[string]string),
	}
	s.securityGroups[sg.GroupId] = sg
	return sg, ""
}

// GetSecurityGroup returns a security group by ID.
func (s *Store) GetSecurityGroup(groupId string) (*SecurityGroup, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sg, ok := s.securityGroups[groupId]
	return sg, ok
}

// ListSecurityGroups returns security groups filtered by IDs, VPC ID, and/or
// group name. Any filter that is empty / nil is ignored.
func (s *Store) ListSecurityGroups(ids []string, vpcId, groupName string) []*SecurityGroup {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var idSet map[string]struct{}
	if len(ids) > 0 {
		idSet = make(map[string]struct{}, len(ids))
		for _, id := range ids {
			idSet[id] = struct{}{}
		}
	}

	result := make([]*SecurityGroup, 0)
	for _, sg := range s.securityGroups {
		if idSet != nil {
			if _, ok := idSet[sg.GroupId]; !ok {
				continue
			}
		}
		if vpcId != "" && sg.VpcId != vpcId {
			continue
		}
		if groupName != "" && sg.GroupName != groupName {
			continue
		}
		result = append(result, sg)
	}
	return result
}

// DeleteSecurityGroup removes a security group. Returns an error code string or
// empty on success.  Fails if any other security group references this one.
func (s *Store) DeleteSecurityGroup(groupId string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.securityGroups[groupId]; !ok {
		return "not_found"
	}

	// Check whether any SG references this one.
	for id, sg := range s.securityGroups {
		if id == groupId {
			continue
		}
		for _, rule := range sg.IngressRules {
			for _, gid := range rule.GroupIds {
				if gid == groupId {
					return "dependency"
				}
			}
		}
		for _, rule := range sg.EgressRules {
			for _, gid := range rule.GroupIds {
				if gid == groupId {
					return "dependency"
				}
			}
		}
	}

	delete(s.securityGroups, groupId)
	return ""
}

// AuthorizeSecurityGroupIngress adds ingress rules to a security group.
func (s *Store) AuthorizeSecurityGroupIngress(groupId string, rules []SGRule) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	sg, ok := s.securityGroups[groupId]
	if !ok {
		return "not_found"
	}
	sg.IngressRules = append(sg.IngressRules, rules...)
	return ""
}

// AuthorizeSecurityGroupEgress adds egress rules to a security group.
func (s *Store) AuthorizeSecurityGroupEgress(groupId string, rules []SGRule) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	sg, ok := s.securityGroups[groupId]
	if !ok {
		return "not_found"
	}
	sg.EgressRules = append(sg.EgressRules, rules...)
	return ""
}

// ruleMatches returns true when candidate matches the target rule.
func ruleMatches(target, candidate SGRule) bool {
	if target.IpProtocol != candidate.IpProtocol {
		return false
	}
	if target.FromPort != candidate.FromPort || target.ToPort != candidate.ToPort {
		return false
	}
	if len(target.CidrBlocks) != len(candidate.CidrBlocks) {
		return false
	}
	cidrSet := make(map[string]struct{}, len(target.CidrBlocks))
	for _, c := range target.CidrBlocks {
		cidrSet[c] = struct{}{}
	}
	for _, c := range candidate.CidrBlocks {
		if _, ok := cidrSet[c]; !ok {
			return false
		}
	}
	if len(target.GroupIds) != len(candidate.GroupIds) {
		return false
	}
	gidSet := make(map[string]struct{}, len(target.GroupIds))
	for _, g := range target.GroupIds {
		gidSet[g] = struct{}{}
	}
	for _, g := range candidate.GroupIds {
		if _, ok := gidSet[g]; !ok {
			return false
		}
	}
	return true
}

// removeRules removes rules from slice that match any rule in toRemove.
func removeRules(existing []SGRule, toRemove []SGRule) ([]SGRule, bool) {
	removed := false
	result := existing[:0:0]
	for _, rule := range existing {
		matched := false
		for _, rem := range toRemove {
			if ruleMatches(rem, rule) {
				matched = true
				break
			}
		}
		if matched {
			removed = true
		} else {
			result = append(result, rule)
		}
	}
	return result, removed
}

// RevokeSecurityGroupIngress removes matching ingress rules.
func (s *Store) RevokeSecurityGroupIngress(groupId string, rules []SGRule) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	sg, ok := s.securityGroups[groupId]
	if !ok {
		return "not_found"
	}
	updated, removed := removeRules(sg.IngressRules, rules)
	if !removed {
		return "rule_not_found"
	}
	sg.IngressRules = updated
	return ""
}

// RevokeSecurityGroupEgress removes matching egress rules.
func (s *Store) RevokeSecurityGroupEgress(groupId string, rules []SGRule) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	sg, ok := s.securityGroups[groupId]
	if !ok {
		return "not_found"
	}
	updated, removed := removeRules(sg.EgressRules, rules)
	if !removed {
		return "rule_not_found"
	}
	sg.EgressRules = updated
	return ""
}

// ---- InternetGateway operations ----

// CreateInternetGateway creates a new detached internet gateway.
func (s *Store) CreateInternetGateway() *InternetGateway {
	s.mu.Lock()
	defer s.mu.Unlock()

	igw := &InternetGateway{
		IgwId:       genID("igw-"),
		Attachments: []IGWAttachment{},
		Tags:        make(map[string]string),
	}
	s.internetGateways[igw.IgwId] = igw
	return igw
}

// AttachInternetGateway attaches an IGW to a VPC. Returns an error code string.
// Empty string means success.
func (s *Store) AttachInternetGateway(igwId, vpcId string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	igw, ok := s.internetGateways[igwId]
	if !ok {
		return "igw_not_found"
	}
	if _, ok := s.vpcs[vpcId]; !ok {
		return "vpc_not_found"
	}
	// IGW may only be attached to one VPC at a time.
	if len(igw.Attachments) > 0 {
		return "already_attached"
	}
	// VPC may only have one IGW.
	for _, other := range s.internetGateways {
		for _, att := range other.Attachments {
			if att.VpcId == vpcId {
				return "vpc_already_has_igw"
			}
		}
	}
	igw.Attachments = append(igw.Attachments, IGWAttachment{VpcId: vpcId, State: "attached"})
	return ""
}

// DetachInternetGateway detaches an IGW from a VPC. Returns an error code string.
func (s *Store) DetachInternetGateway(igwId, vpcId string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	igw, ok := s.internetGateways[igwId]
	if !ok {
		return "igw_not_found"
	}
	if _, ok := s.vpcs[vpcId]; !ok {
		return "vpc_not_found"
	}
	newAtts := igw.Attachments[:0:0]
	found := false
	for _, att := range igw.Attachments {
		if att.VpcId == vpcId {
			found = true
			continue
		}
		newAtts = append(newAtts, att)
	}
	if !found {
		return "not_attached"
	}
	igw.Attachments = newAtts
	return ""
}

// DeleteInternetGateway deletes an IGW. Fails if still attached to a VPC.
func (s *Store) DeleteInternetGateway(igwId string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	igw, ok := s.internetGateways[igwId]
	if !ok {
		return "not_found"
	}
	if len(igw.Attachments) > 0 {
		return "still_attached"
	}
	delete(s.internetGateways, igwId)
	return ""
}

// ListInternetGateways returns IGWs filtered by IDs and/or attached VPC ID.
func (s *Store) ListInternetGateways(ids []string, filterVpcId string) []*InternetGateway {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var idSet map[string]struct{}
	if len(ids) > 0 {
		idSet = make(map[string]struct{}, len(ids))
		for _, id := range ids {
			idSet[id] = struct{}{}
		}
	}

	result := make([]*InternetGateway, 0)
	for _, igw := range s.internetGateways {
		if idSet != nil {
			if _, ok := idSet[igw.IgwId]; !ok {
				continue
			}
		}
		if filterVpcId != "" {
			found := false
			for _, att := range igw.Attachments {
				if att.VpcId == filterVpcId {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		result = append(result, igw)
	}
	return result
}

// ---- NatGateway operations ----

// CreateNatGateway creates a new NAT gateway in the given subnet with the given
// EIP allocation ID. Returns the gateway and an error code string.
func (s *Store) CreateNatGateway(subnetId, allocationId string) (*NatGateway, string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sub, ok := s.subnets[subnetId]
	if !ok {
		return nil, "subnet_not_found"
	}

	nat := &NatGateway{
		NatGatewayId: genID("nat-"),
		SubnetId:     subnetId,
		VpcId:        sub.VpcId,
		AllocationId: allocationId,
		State:        "available",
		Tags:         make(map[string]string),
	}
	s.natGateways[nat.NatGatewayId] = nat
	return nat, ""
}

// ListNatGateways returns NAT gateways filtered by IDs, subnet ID, VPC ID,
// and/or state. Any empty filter is ignored.
func (s *Store) ListNatGateways(ids []string, filterSubnetId, filterVpcId, filterState string) []*NatGateway {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var idSet map[string]struct{}
	if len(ids) > 0 {
		idSet = make(map[string]struct{}, len(ids))
		for _, id := range ids {
			idSet[id] = struct{}{}
		}
	}

	result := make([]*NatGateway, 0)
	for _, nat := range s.natGateways {
		if idSet != nil {
			if _, ok := idSet[nat.NatGatewayId]; !ok {
				continue
			}
		}
		if filterSubnetId != "" && nat.SubnetId != filterSubnetId {
			continue
		}
		if filterVpcId != "" && nat.VpcId != filterVpcId {
			continue
		}
		if filterState != "" && nat.State != filterState {
			continue
		}
		result = append(result, nat)
	}
	return result
}

// DeleteNatGateway sets a NAT gateway's state to deleted. Returns an error code.
func (s *Store) DeleteNatGateway(natGatewayId string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	nat, ok := s.natGateways[natGatewayId]
	if !ok {
		return "not_found"
	}
	nat.State = "deleted"
	return ""
}

// ---- RouteTable operations ----

// CreateRouteTable creates a new route table in the given VPC with a local route.
func (s *Store) CreateRouteTable(vpcId string) (*RouteTable, string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	vpc, ok := s.vpcs[vpcId]
	if !ok {
		return nil, "vpc_not_found"
	}

	rt := &RouteTable{
		RouteTableId: genID("rtb-"),
		VpcId:        vpcId,
		IsMain:       false,
		Routes: []Route{
			{
				DestinationCidrBlock: vpc.CidrBlock,
				GatewayId:            "local",
				State:                "active",
				Origin:               "CreateRouteTable",
			},
		},
		Tags: make(map[string]string),
	}
	s.routeTables[rt.RouteTableId] = rt
	return rt, ""
}

// ListRouteTables returns route tables filtered by IDs and/or VPC ID.
func (s *Store) ListRouteTables(ids []string, filterVpcId string) []*RouteTable {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var idSet map[string]struct{}
	if len(ids) > 0 {
		idSet = make(map[string]struct{}, len(ids))
		for _, id := range ids {
			idSet[id] = struct{}{}
		}
	}

	result := make([]*RouteTable, 0)
	for _, rt := range s.routeTables {
		if idSet != nil {
			if _, ok := idSet[rt.RouteTableId]; !ok {
				continue
			}
		}
		if filterVpcId != "" && rt.VpcId != filterVpcId {
			continue
		}
		result = append(result, rt)
	}
	return result
}

// DeleteRouteTable deletes a route table. Fails if it has subnet associations.
func (s *Store) DeleteRouteTable(rtbId string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	rt, ok := s.routeTables[rtbId]
	if !ok {
		return "not_found"
	}
	if rt.IsMain {
		return "main_table"
	}
	// Check for subnet associations.
	for _, assoc := range s.routeTableAssociations {
		if assoc.RouteTableId == rtbId && !assoc.Main {
			return "has_associations"
		}
	}
	delete(s.routeTables, rtbId)
	return ""
}

// CreateRoute adds a new route to a route table. Returns an error code string.
// Validates that the gateway/NAT GW target exists.
func (s *Store) CreateRoute(rtbId, destCidr, gatewayId, natGatewayId, vpcEndpointId string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	rt, ok := s.routeTables[rtbId]
	if !ok {
		return "rtb_not_found"
	}

	// Check for duplicate destination.
	for _, r := range rt.Routes {
		if r.DestinationCidrBlock == destCidr {
			return "route_already_exists"
		}
	}

	// Validate target exists.
	if gatewayId != "" && gatewayId != "local" {
		if _, ok := s.internetGateways[gatewayId]; !ok {
			return "gateway_not_found"
		}
	}
	if natGatewayId != "" {
		nat, ok := s.natGateways[natGatewayId]
		if !ok || nat.State == "deleted" {
			return "nat_not_found"
		}
	}

	route := Route{
		DestinationCidrBlock: destCidr,
		GatewayId:            gatewayId,
		NatGatewayId:         natGatewayId,
		VpcEndpointId:        vpcEndpointId,
		State:                "active",
		Origin:               "CreateRoute",
	}
	rt.Routes = append(rt.Routes, route)
	return ""
}

// DeleteRoute removes a route by destination CIDR from a route table.
func (s *Store) DeleteRoute(rtbId, destCidr string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	rt, ok := s.routeTables[rtbId]
	if !ok {
		return "rtb_not_found"
	}

	newRoutes := rt.Routes[:0:0]
	found := false
	for _, r := range rt.Routes {
		if r.DestinationCidrBlock == destCidr {
			if r.Origin == "CreateRouteTable" {
				return "local_route"
			}
			found = true
			continue
		}
		newRoutes = append(newRoutes, r)
	}
	if !found {
		return "route_not_found"
	}
	rt.Routes = newRoutes
	return ""
}

// ReplaceRoute updates the target for an existing route destination.
func (s *Store) ReplaceRoute(rtbId, destCidr, gatewayId, natGatewayId, vpcEndpointId string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	rt, ok := s.routeTables[rtbId]
	if !ok {
		return "rtb_not_found"
	}

	// Validate target exists.
	if gatewayId != "" && gatewayId != "local" {
		if _, ok := s.internetGateways[gatewayId]; !ok {
			return "gateway_not_found"
		}
	}
	if natGatewayId != "" {
		nat, ok := s.natGateways[natGatewayId]
		if !ok || nat.State == "deleted" {
			return "nat_not_found"
		}
	}

	for i, r := range rt.Routes {
		if r.DestinationCidrBlock == destCidr {
			rt.Routes[i].GatewayId = gatewayId
			rt.Routes[i].NatGatewayId = natGatewayId
			rt.Routes[i].VpcEndpointId = vpcEndpointId
			return ""
		}
	}
	return "route_not_found"
}

// AssociateRouteTable links a subnet to a route table. Returns the association
// ID and an error code string.
func (s *Store) AssociateRouteTable(rtbId, subnetId string) (string, string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.routeTables[rtbId]; !ok {
		return "", "rtb_not_found"
	}
	if _, ok := s.subnets[subnetId]; !ok {
		return "", "subnet_not_found"
	}

	// Check if subnet already associated with a route table.
	for _, assoc := range s.routeTableAssociations {
		if assoc.SubnetId == subnetId {
			return "", "already_associated"
		}
	}

	assocId := genID("rtbassoc-")
	assoc := &RouteTableAssociation{
		AssociationId: assocId,
		RouteTableId:  rtbId,
		SubnetId:      subnetId,
		Main:          false,
	}
	s.routeTableAssociations[assocId] = assoc
	return assocId, ""
}

// DisassociateRouteTable removes a subnet-to-route-table association.
func (s *Store) DisassociateRouteTable(assocId string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	assoc, ok := s.routeTableAssociations[assocId]
	if !ok {
		return "not_found"
	}
	if assoc.Main {
		return "main_association"
	}
	delete(s.routeTableAssociations, assocId)
	return ""
}

// ListRouteTableAssociations returns all associations for a given route table.
func (s *Store) ListRouteTableAssociations(rtbId string) []*RouteTableAssociation {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*RouteTableAssociation, 0)
	for _, assoc := range s.routeTableAssociations {
		if assoc.RouteTableId == rtbId {
			result = append(result, assoc)
		}
	}
	return result
}

// ---- ElasticIP operations ----

// AllocateAddress allocates a new Elastic IP in the vpc domain.
func (s *Store) AllocateAddress() *ElasticIP {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate a pseudo-random 54.x.x.x public IP.
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	publicIp := fmt.Sprintf("54.%d.%d.%d", b[0], b[1], b[2])

	eip := &ElasticIP{
		AllocationId: genID("eipalloc-"),
		PublicIp:     publicIp,
		Tags:         make(map[string]string),
	}
	s.elasticIPs[eip.AllocationId] = eip
	return eip
}

// ReleaseAddress releases an allocated Elastic IP. Returns an error code string.
func (s *Store) ReleaseAddress(allocationId string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	eip, ok := s.elasticIPs[allocationId]
	if !ok {
		return "not_found"
	}
	if eip.AssociationId != "" {
		return "still_associated"
	}
	delete(s.elasticIPs, allocationId)
	return ""
}

// AssociateAddress associates an EIP with an instance or ENI. Returns the
// association ID and an error code string.
func (s *Store) AssociateAddress(allocationId, instanceId, networkInterfaceId string) (string, string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	eip, ok := s.elasticIPs[allocationId]
	if !ok {
		return "", "not_found"
	}
	if eip.AssociationId != "" {
		return "", "already_associated"
	}

	assocId := genID("eipassoc-")
	eip.AssociationId = assocId
	eip.InstanceId = instanceId
	eip.NetworkInterfaceId = networkInterfaceId
	s.eipAssociations[assocId] = allocationId
	return assocId, ""
}

// DisassociateAddress removes an EIP association. Returns an error code string.
func (s *Store) DisassociateAddress(assocId string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	allocationId, ok := s.eipAssociations[assocId]
	if !ok {
		return "not_found"
	}
	eip, ok := s.elasticIPs[allocationId]
	if !ok {
		return "not_found"
	}
	eip.AssociationId = ""
	eip.InstanceId = ""
	eip.NetworkInterfaceId = ""
	delete(s.eipAssociations, assocId)
	return ""
}

// ListAddresses returns EIPs, optionally filtered by allocation IDs.
func (s *Store) ListAddresses(ids []string) []*ElasticIP {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var idSet map[string]struct{}
	if len(ids) > 0 {
		idSet = make(map[string]struct{}, len(ids))
		for _, id := range ids {
			idSet[id] = struct{}{}
		}
	}

	result := make([]*ElasticIP, 0)
	for _, eip := range s.elasticIPs {
		if idSet != nil {
			if _, ok := idSet[eip.AllocationId]; !ok {
				continue
			}
		}
		result = append(result, eip)
	}
	return result
}

// ---- NetworkInterface operations ----

// CreateNetworkInterface creates a new ENI in the given subnet. Assigns a
// private IP from the subnet's CIDR (10.0.x.x range).
func (s *Store) CreateNetworkInterface(subnetId string, sgIds []string) (*NetworkInterface, string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sub, ok := s.subnets[subnetId]
	if !ok {
		return nil, "subnet_not_found"
	}

	// Derive a pseudo-random private IP within the subnet CIDR.
	_, subNet, err := net.ParseCIDR(sub.CidrBlock)
	var privateIP string
	if err == nil {
		b := make([]byte, 1)
		_, _ = rand.Read(b)
		ip := make(net.IP, len(subNet.IP))
		copy(ip, subNet.IP)
		// Set the host octet to a random value in [10,250].
		ip[len(ip)-1] = 10 + b[0]%240
		privateIP = ip.String()
	} else {
		privateIP = "10.0.0.10"
	}

	eni := &NetworkInterface{
		NetworkInterfaceId: genID("eni-"),
		SubnetId:           subnetId,
		VpcId:              sub.VpcId,
		PrivateIpAddress:   privateIP,
		SecurityGroupIds:   sgIds,
		Status:             "available",
		Tags:               make(map[string]string),
	}
	s.networkInterfaces[eni.NetworkInterfaceId] = eni
	return eni, ""
}

// ListNetworkInterfaces returns ENIs filtered by IDs and/or subnet ID.
func (s *Store) ListNetworkInterfaces(ids []string, filterSubnetId string) []*NetworkInterface {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var idSet map[string]struct{}
	if len(ids) > 0 {
		idSet = make(map[string]struct{}, len(ids))
		for _, id := range ids {
			idSet[id] = struct{}{}
		}
	}

	result := make([]*NetworkInterface, 0)
	for _, eni := range s.networkInterfaces {
		if idSet != nil {
			if _, ok := idSet[eni.NetworkInterfaceId]; !ok {
				continue
			}
		}
		if filterSubnetId != "" && eni.SubnetId != filterSubnetId {
			continue
		}
		result = append(result, eni)
	}
	return result
}

// DeleteNetworkInterface removes an ENI. Returns an error code string.
func (s *Store) DeleteNetworkInterface(eniId string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	eni, ok := s.networkInterfaces[eniId]
	if !ok {
		return "not_found"
	}
	if eni.Status == "in-use" {
		return "in_use"
	}
	delete(s.networkInterfaces, eniId)
	return ""
}

// ---- NetworkACL operations ----

// CreateNetworkACL creates a new (non-default) network ACL in the given VPC.
func (s *Store) CreateNetworkACL(vpcId string) (*NetworkACL, string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.vpcs[vpcId]; !ok {
		return nil, "vpc_not_found"
	}

	acl := &NetworkACL{
		NetworkAclId: genID("acl-"),
		VpcId:        vpcId,
		IsDefault:    false,
		Entries: []NACLEntry{
			{RuleNumber: 100, Protocol: "-1", RuleAction: "allow", Egress: false, CidrBlock: "0.0.0.0/0"},
			{RuleNumber: 32767, Protocol: "-1", RuleAction: "deny", Egress: false, CidrBlock: "0.0.0.0/0"},
			{RuleNumber: 100, Protocol: "-1", RuleAction: "allow", Egress: true, CidrBlock: "0.0.0.0/0"},
			{RuleNumber: 32767, Protocol: "-1", RuleAction: "deny", Egress: true, CidrBlock: "0.0.0.0/0"},
		},
		Tags: make(map[string]string),
	}
	s.networkACLs[acl.NetworkAclId] = acl
	return acl, ""
}

// ListNetworkACLs returns NACLs filtered by IDs and/or VPC ID.
func (s *Store) ListNetworkACLs(ids []string, filterVpcId string) []*NetworkACL {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var idSet map[string]struct{}
	if len(ids) > 0 {
		idSet = make(map[string]struct{}, len(ids))
		for _, id := range ids {
			idSet[id] = struct{}{}
		}
	}

	result := make([]*NetworkACL, 0)
	for _, acl := range s.networkACLs {
		if idSet != nil {
			if _, ok := idSet[acl.NetworkAclId]; !ok {
				continue
			}
		}
		if filterVpcId != "" && acl.VpcId != filterVpcId {
			continue
		}
		result = append(result, acl)
	}
	return result
}

// DeleteNetworkACL removes a non-default NACL. Returns an error code string.
func (s *Store) DeleteNetworkACL(aclId string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	acl, ok := s.networkACLs[aclId]
	if !ok {
		return "not_found"
	}
	if acl.IsDefault {
		return "default_acl"
	}
	delete(s.networkACLs, aclId)
	return ""
}

// CreateNetworkACLEntry adds a rule to a NACL. Returns an error code string.
func (s *Store) CreateNetworkACLEntry(aclId string, entry NACLEntry) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	acl, ok := s.networkACLs[aclId]
	if !ok {
		return "not_found"
	}
	// Check for duplicate rule number + direction.
	for _, e := range acl.Entries {
		if e.RuleNumber == entry.RuleNumber && e.Egress == entry.Egress {
			return "duplicate_rule"
		}
	}
	acl.Entries = append(acl.Entries, entry)
	return ""
}

// DeleteNetworkACLEntry removes a rule from a NACL. Returns an error code string.
func (s *Store) DeleteNetworkACLEntry(aclId string, ruleNumber int, egress bool) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	acl, ok := s.networkACLs[aclId]
	if !ok {
		return "not_found"
	}
	newEntries := acl.Entries[:0:0]
	found := false
	for _, e := range acl.Entries {
		if e.RuleNumber == ruleNumber && e.Egress == egress {
			found = true
			continue
		}
		newEntries = append(newEntries, e)
	}
	if !found {
		return "entry_not_found"
	}
	acl.Entries = newEntries
	return ""
}

// ---- VPCEndpoint operations ----

// CreateVPCEndpoint creates a new VPC endpoint. Returns the endpoint and an
// error code string.
func (s *Store) CreateVPCEndpoint(vpcId, serviceName, epType string) (*VPCEndpoint, string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.vpcs[vpcId]; !ok {
		return nil, "vpc_not_found"
	}

	ep := &VPCEndpoint{
		VpcEndpointId:   genID("vpce-"),
		VpcId:           vpcId,
		ServiceName:     serviceName,
		VpcEndpointType: epType,
		State:           "available",
		Tags:            make(map[string]string),
	}
	s.vpcEndpoints[ep.VpcEndpointId] = ep
	return ep, ""
}

// ListVPCEndpoints returns endpoints filtered by IDs and/or VPC ID.
func (s *Store) ListVPCEndpoints(ids []string, filterVpcId string) []*VPCEndpoint {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var idSet map[string]struct{}
	if len(ids) > 0 {
		idSet = make(map[string]struct{}, len(ids))
		for _, id := range ids {
			idSet[id] = struct{}{}
		}
	}

	result := make([]*VPCEndpoint, 0)
	for _, ep := range s.vpcEndpoints {
		if idSet != nil {
			if _, ok := idSet[ep.VpcEndpointId]; !ok {
				continue
			}
		}
		if filterVpcId != "" && ep.VpcId != filterVpcId {
			continue
		}
		result = append(result, ep)
	}
	return result
}

// DeleteVPCEndpoint removes a VPC endpoint. Returns an error code string.
func (s *Store) DeleteVPCEndpoint(epId string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.vpcEndpoints[epId]; !ok {
		return "not_found"
	}
	delete(s.vpcEndpoints, epId)
	return ""
}

// ---- VPCPeeringConnection operations ----

// CreateVPCPeeringConnection creates a new peering connection between two VPCs.
// Returns the connection and an error code string.
func (s *Store) CreateVPCPeeringConnection(requesterVpcId, accepterVpcId string) (*VPCPeeringConnection, string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.vpcs[requesterVpcId]; !ok {
		return nil, "requester_not_found"
	}
	if _, ok := s.vpcs[accepterVpcId]; !ok {
		return nil, "accepter_not_found"
	}

	pcx := &VPCPeeringConnection{
		PeeringConnectionId: genID("pcx-"),
		RequesterVpcId:      requesterVpcId,
		AccepterVpcId:       accepterVpcId,
		Status:              "pending-acceptance",
		Tags:                make(map[string]string),
	}
	s.vpcPeeringConnections[pcx.PeeringConnectionId] = pcx
	return pcx, ""
}

// AcceptVPCPeeringConnection accepts a pending peering connection. Returns the
// updated connection and an error code string.
func (s *Store) AcceptVPCPeeringConnection(pcxId string) (*VPCPeeringConnection, string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pcx, ok := s.vpcPeeringConnections[pcxId]
	if !ok {
		return nil, "not_found"
	}
	if pcx.Status != "pending-acceptance" {
		return nil, "invalid_state"
	}
	pcx.Status = "active"
	return pcx, ""
}

// DeleteVPCPeeringConnection deletes a peering connection. Returns an error code.
func (s *Store) DeleteVPCPeeringConnection(pcxId string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.vpcPeeringConnections[pcxId]; !ok {
		return "not_found"
	}
	delete(s.vpcPeeringConnections, pcxId)
	return ""
}

// ---- helpers ----

// cidrContains checks whether inner is fully contained within outer.
func cidrContains(outer, inner *net.IPNet) bool {
	// outer must contain the first IP of inner.
	if !outer.Contains(inner.IP) {
		return false
	}
	// outer must contain the last IP of inner.
	lastIP := lastAddr(inner)
	return outer.Contains(lastIP)
}

// lastAddr returns the last (broadcast) address in a CIDR range.
func lastAddr(n *net.IPNet) net.IP {
	ip := make(net.IP, len(n.IP))
	copy(ip, n.IP)
	for i := range ip {
		ip[i] |= ^n.Mask[i]
	}
	return ip
}

// ---- Instance operations ----

// instanceStateCode returns the numeric AWS state code for a state name.
func instanceStateCode(state string) int {
	switch state {
	case "pending":
		return 0
	case "running":
		return 16
	case "shutting-down":
		return 32
	case "terminated":
		return 48
	case "stopping":
		return 64
	case "stopped":
		return 80
	default:
		return 0
	}
}

// allocatePrivateIP allocates the next sequential IP from the subnet's CIDR.
// AWS reserves the first 4 and last 1 addresses; we start allocating from
// network+4 (offset 4) and increment per call.
func (s *Store) allocatePrivateIP(subnetId string) string {
	sub, ok := s.subnets[subnetId]
	if !ok {
		return ""
	}
	_, ipNet, err := net.ParseCIDR(sub.CidrBlock)
	if err != nil {
		return ""
	}

	// Current counter (starts at 0 meaning first allocation = offset 4).
	offset := s.subnetIPCounters[subnetId]
	s.subnetIPCounters[subnetId] = offset + 1

	// Convert network base IP to uint32, add 4 + offset.
	ip4 := ipNet.IP.To4()
	if ip4 == nil {
		return ""
	}
	base := binary.BigEndian.Uint32(ip4)
	addr := base + 4 + offset

	result := make(net.IP, 4)
	binary.BigEndian.PutUint32(result, addr)
	return result.String()
}

// RunInstances creates count instances.
// Returns the created instances and a reservation ID, or an error code.
func (s *Store) RunInstances(imageId, instanceType, subnetId, keyName string, sgIds []string, count int) ([]*Instance, string, string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sub, ok := s.subnets[subnetId]
	if !ok {
		return nil, "", "subnet_not_found"
	}

	reservationId := genID("r-")
	instances := make([]*Instance, 0, count)
	for i := 0; i < count; i++ {
		inst := &Instance{
			InstanceId:       genID("i-"),
			ImageId:          imageId,
			InstanceType:     instanceType,
			SubnetId:         subnetId,
			VpcId:            sub.VpcId,
			SecurityGroupIds: sgIds,
			KeyName:          keyName,
			State:            "running",
			LaunchTime:       time.Now().UTC(),
			PrivateIpAddress: s.allocatePrivateIP(subnetId),
			Tags:             make(map[string]string),
		}
		s.instances[inst.InstanceId] = inst
		instances = append(instances, inst)
	}
	return instances, reservationId, ""
}

// ListInstances returns instances filtered by IDs and/or filters.
// filters: map of filter-name -> values (any match).
func (s *Store) ListInstances(ids []string, filters map[string][]string) []*Instance {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var idSet map[string]struct{}
	if len(ids) > 0 {
		idSet = make(map[string]struct{}, len(ids))
		for _, id := range ids {
			idSet[id] = struct{}{}
		}
	}

	result := make([]*Instance, 0)
	for _, inst := range s.instances {
		if idSet != nil {
			if _, ok := idSet[inst.InstanceId]; !ok {
				continue
			}
		}
		if !instanceMatchesFilters(inst, filters) {
			continue
		}
		result = append(result, inst)
	}
	return result
}

// instanceMatchesFilters returns true if inst matches all provided filters.
func instanceMatchesFilters(inst *Instance, filters map[string][]string) bool {
	for name, vals := range filters {
		if len(vals) == 0 {
			continue
		}
		switch name {
		case "vpc-id":
			if !anyEqual(vals, inst.VpcId) {
				return false
			}
		case "subnet-id":
			if !anyEqual(vals, inst.SubnetId) {
				return false
			}
		case "instance-state-name":
			if !anyEqual(vals, inst.State) {
				return false
			}
		default:
			// tag:Name or tag:<key>
			if strings.HasPrefix(name, "tag:") {
				tagKey := name[4:]
				tagVal, exists := inst.Tags[tagKey]
				if !exists {
					return false
				}
				if !anyEqual(vals, tagVal) {
					return false
				}
			}
		}
	}
	return true
}

// anyEqual returns true if target equals any element of vals.
func anyEqual(vals []string, target string) bool {
	for _, v := range vals {
		if v == target {
			return true
		}
	}
	return false
}

// TerminateInstances sets state to terminated for the given instance IDs.
// Returns a map of instanceId -> {previousState, currentState}.
func (s *Store) TerminateInstances(ids []string) map[string][2]string {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make(map[string][2]string)
	for _, id := range ids {
		inst, ok := s.instances[id]
		if !ok {
			continue
		}
		prev := inst.State
		inst.State = "terminated"
		result[id] = [2]string{prev, "terminated"}
	}
	return result
}

// StopInstances sets state to stopped for the given instance IDs.
// Only instances in "running" state can be stopped.
func (s *Store) StopInstances(ids []string) map[string][2]string {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make(map[string][2]string)
	for _, id := range ids {
		inst, ok := s.instances[id]
		if !ok {
			continue
		}
		if inst.State != "running" {
			continue
		}
		prev := inst.State
		inst.State = "stopped"
		result[id] = [2]string{prev, "stopped"}
	}
	return result
}

// StartInstances sets state to running for the given instance IDs.
// Only instances in "stopped" state can be started.
func (s *Store) StartInstances(ids []string) map[string][2]string {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make(map[string][2]string)
	for _, id := range ids {
		inst, ok := s.instances[id]
		if !ok {
			continue
		}
		if inst.State != "stopped" {
			continue
		}
		prev := inst.State
		inst.State = "running"
		result[id] = [2]string{prev, "running"}
	}
	return result
}

// ---- Tagging operations ----

// resourceTypeFromID infers the EC2 resource type from the resource ID prefix.
func resourceTypeFromID(id string) string {
	switch {
	case strings.HasPrefix(id, "vpc-"):
		return "vpc"
	case strings.HasPrefix(id, "subnet-"):
		return "subnet"
	case strings.HasPrefix(id, "sg-"):
		return "security-group"
	case strings.HasPrefix(id, "i-"):
		return "instance"
	case strings.HasPrefix(id, "igw-"):
		return "internet-gateway"
	case strings.HasPrefix(id, "nat-"):
		return "natgateway"
	case strings.HasPrefix(id, "rtb-"):
		return "route-table"
	case strings.HasPrefix(id, "eipalloc-"):
		return "elastic-ip"
	case strings.HasPrefix(id, "eni-"):
		return "network-interface"
	case strings.HasPrefix(id, "acl-"):
		return "network-acl"
	case strings.HasPrefix(id, "vpce-"):
		return "vpc-endpoint"
	case strings.HasPrefix(id, "pcx-"):
		return "vpc-peering-connection"
	case strings.HasPrefix(id, "r-"):
		return "reservation"
	default:
		return "unknown"
	}
}

// applyTagsToResource propagates tags from the central tags map to the resource's own Tags map.
func (s *Store) applyTagsToResource(resourceId string, kv map[string]string) {
	for k, v := range kv {
		switch {
		case strings.HasPrefix(resourceId, "vpc-"):
			if r, ok := s.vpcs[resourceId]; ok {
				r.Tags[k] = v
			}
		case strings.HasPrefix(resourceId, "subnet-"):
			if r, ok := s.subnets[resourceId]; ok {
				r.Tags[k] = v
			}
		case strings.HasPrefix(resourceId, "sg-"):
			if r, ok := s.securityGroups[resourceId]; ok {
				r.Tags[k] = v
			}
		case strings.HasPrefix(resourceId, "i-"):
			if r, ok := s.instances[resourceId]; ok {
				r.Tags[k] = v
			}
		case strings.HasPrefix(resourceId, "igw-"):
			if r, ok := s.internetGateways[resourceId]; ok {
				r.Tags[k] = v
			}
		case strings.HasPrefix(resourceId, "nat-"):
			if r, ok := s.natGateways[resourceId]; ok {
				r.Tags[k] = v
			}
		case strings.HasPrefix(resourceId, "rtb-"):
			if r, ok := s.routeTables[resourceId]; ok {
				r.Tags[k] = v
			}
		case strings.HasPrefix(resourceId, "eipalloc-"):
			if r, ok := s.elasticIPs[resourceId]; ok {
				r.Tags[k] = v
			}
		case strings.HasPrefix(resourceId, "eni-"):
			if r, ok := s.networkInterfaces[resourceId]; ok {
				r.Tags[k] = v
			}
		case strings.HasPrefix(resourceId, "acl-"):
			if r, ok := s.networkACLs[resourceId]; ok {
				r.Tags[k] = v
			}
		case strings.HasPrefix(resourceId, "vpce-"):
			if r, ok := s.vpcEndpoints[resourceId]; ok {
				r.Tags[k] = v
			}
		case strings.HasPrefix(resourceId, "pcx-"):
			if r, ok := s.vpcPeeringConnections[resourceId]; ok {
				r.Tags[k] = v
			}
		}
	}
}

// removeTagsFromResource removes keys from the resource's own Tags map.
func (s *Store) removeTagsFromResource(resourceId string, keys []string) {
	var tags map[string]string
	switch {
	case strings.HasPrefix(resourceId, "vpc-"):
		if r, ok := s.vpcs[resourceId]; ok {
			tags = r.Tags
		}
	case strings.HasPrefix(resourceId, "subnet-"):
		if r, ok := s.subnets[resourceId]; ok {
			tags = r.Tags
		}
	case strings.HasPrefix(resourceId, "sg-"):
		if r, ok := s.securityGroups[resourceId]; ok {
			tags = r.Tags
		}
	case strings.HasPrefix(resourceId, "i-"):
		if r, ok := s.instances[resourceId]; ok {
			tags = r.Tags
		}
	case strings.HasPrefix(resourceId, "igw-"):
		if r, ok := s.internetGateways[resourceId]; ok {
			tags = r.Tags
		}
	case strings.HasPrefix(resourceId, "nat-"):
		if r, ok := s.natGateways[resourceId]; ok {
			tags = r.Tags
		}
	case strings.HasPrefix(resourceId, "rtb-"):
		if r, ok := s.routeTables[resourceId]; ok {
			tags = r.Tags
		}
	case strings.HasPrefix(resourceId, "eipalloc-"):
		if r, ok := s.elasticIPs[resourceId]; ok {
			tags = r.Tags
		}
	case strings.HasPrefix(resourceId, "eni-"):
		if r, ok := s.networkInterfaces[resourceId]; ok {
			tags = r.Tags
		}
	case strings.HasPrefix(resourceId, "acl-"):
		if r, ok := s.networkACLs[resourceId]; ok {
			tags = r.Tags
		}
	case strings.HasPrefix(resourceId, "vpce-"):
		if r, ok := s.vpcEndpoints[resourceId]; ok {
			tags = r.Tags
		}
	case strings.HasPrefix(resourceId, "pcx-"):
		if r, ok := s.vpcPeeringConnections[resourceId]; ok {
			tags = r.Tags
		}
	}
	if tags != nil {
		for _, k := range keys {
			delete(tags, k)
		}
	}
}

// CreateTags applies tags to one or more resources.
func (s *Store) CreateTags(resourceIds []string, tags map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, id := range resourceIds {
		if s.tags[id] == nil {
			s.tags[id] = make(map[string]string)
		}
		for k, v := range tags {
			s.tags[id][k] = v
		}
		s.applyTagsToResource(id, tags)
	}
}

// DeleteTags removes specific tag keys from one or more resources.
func (s *Store) DeleteTags(resourceIds []string, keys []string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, id := range resourceIds {
		if m, ok := s.tags[id]; ok {
			for _, k := range keys {
				delete(m, k)
			}
		}
		s.removeTagsFromResource(id, keys)
	}
}

// TagEntry represents a single tag associated with a resource.
type TagEntry struct {
	ResourceId   string
	ResourceType string
	Key          string
	Value        string
}

// ListTags returns all tags matching the provided filters.
// Supported filters: resource-id, resource-type, key, value.
func (s *Store) ListTags(filters map[string][]string) []TagEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []TagEntry
	for resourceId, kv := range s.tags {
		resourceType := resourceTypeFromID(resourceId)
		for k, v := range kv {
			entry := TagEntry{
				ResourceId:   resourceId,
				ResourceType: resourceType,
				Key:          k,
				Value:        v,
			}
			if !tagEntryMatchesFilters(entry, filters) {
				continue
			}
			result = append(result, entry)
		}
	}
	return result
}

// tagEntryMatchesFilters returns true if the entry satisfies all filters.
func tagEntryMatchesFilters(e TagEntry, filters map[string][]string) bool {
	for name, vals := range filters {
		if len(vals) == 0 {
			continue
		}
		switch name {
		case "resource-id":
			if !anyEqual(vals, e.ResourceId) {
				return false
			}
		case "resource-type":
			if !anyEqual(vals, e.ResourceType) {
				return false
			}
		case "key":
			if !anyEqual(vals, e.Key) {
				return false
			}
		case "value":
			if !anyEqual(vals, e.Value) {
				return false
			}
		}
	}
	return true
}

// genID generates an ID with the given prefix followed by 17 random hex characters.
func genID(prefix string) string {
	b := make([]byte, 9) // 9 bytes = 18 hex chars; we take 17
	_, _ = rand.Read(b)
	hex := fmt.Sprintf("%x", b)
	return prefix + hex[:17]
}
