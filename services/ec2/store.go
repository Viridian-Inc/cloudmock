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
}

// SecurityGroup represents a VPC security group (auto-created with VPC).
type SecurityGroup struct {
	GroupId     string
	GroupName   string
	Description string
	VpcId       string
}

// NetworkACL represents a VPC network ACL (auto-created with VPC).
type NetworkACL struct {
	NetworkAclId string
	VpcId        string
	IsDefault    bool
}

// Store manages all EC2 resources.
type Store struct {
	mu             sync.RWMutex
	vpcs           map[string]*VPC
	subnets        map[string]*Subnet
	routeTables    map[string]*RouteTable
	securityGroups map[string]*SecurityGroup
	networkACLs    map[string]*NetworkACL
	accountID      string
	region         string
}

// NewStore returns a new Store for the given account and region.
func NewStore(accountID, region string) *Store {
	return &Store{
		vpcs:           make(map[string]*VPC),
		subnets:        make(map[string]*Subnet),
		routeTables:    make(map[string]*RouteTable),
		securityGroups: make(map[string]*SecurityGroup),
		networkACLs:    make(map[string]*NetworkACL),
		accountID:      accountID,
		region:         region,
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

	// Auto-create default route table.
	rt := &RouteTable{
		RouteTableId: genID("rtb-"),
		VpcId:        vpc.VpcId,
		IsMain:       true,
	}
	s.routeTables[rt.RouteTableId] = rt

	// Auto-create default security group.
	sg := &SecurityGroup{
		GroupId:     genID("sg-"),
		GroupName:   "default",
		Description: "default VPC security group",
		VpcId:       vpc.VpcId,
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
