package ec2

import (
	"crypto/rand"
	"fmt"
	"net"
	"sync"
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

// NetworkACL represents a VPC network ACL (auto-created with VPC).
type NetworkACL struct {
	NetworkAclId string
	VpcId        string
	IsDefault    bool
}

// Store manages all EC2 resources.
type Store struct {
	mu                    sync.RWMutex
	vpcs                  map[string]*VPC
	subnets               map[string]*Subnet
	routeTables           map[string]*RouteTable
	routeTableAssociations map[string]*RouteTableAssociation
	securityGroups        map[string]*SecurityGroup
	networkACLs           map[string]*NetworkACL
	internetGateways      map[string]*InternetGateway
	natGateways           map[string]*NatGateway
	accountID             string
	region                string
}

// NewStore returns a new Store for the given account and region.
func NewStore(accountID, region string) *Store {
	return &Store{
		vpcs:                  make(map[string]*VPC),
		subnets:               make(map[string]*Subnet),
		routeTables:           make(map[string]*RouteTable),
		routeTableAssociations: make(map[string]*RouteTableAssociation),
		securityGroups:        make(map[string]*SecurityGroup),
		networkACLs:           make(map[string]*NetworkACL),
		internetGateways:      make(map[string]*InternetGateway),
		natGateways:           make(map[string]*NatGateway),
		accountID:             accountID,
		region:                region,
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

	// Auto-create default network ACL.
	nacl := &NetworkACL{
		NetworkAclId: genID("acl-"),
		VpcId:        vpc.VpcId,
		IsDefault:    true,
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

// genID generates an ID with the given prefix followed by 17 random hex characters.
func genID(prefix string) string {
	b := make([]byte, 9) // 9 bytes = 18 hex chars; we take 17
	_, _ = rand.Read(b)
	hex := fmt.Sprintf("%x", b)
	return prefix + hex[:17]
}
