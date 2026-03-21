package ec2

// This file contains VPC and Subnet specific validation and helper logic.

import "net"

// ValidateCIDR checks whether a CIDR string is syntactically valid.
func ValidateCIDR(cidr string) bool {
	_, _, err := net.ParseCIDR(cidr)
	return err == nil
}

// SubnetCIDRWithinVPC returns true if the subnet CIDR block is fully contained
// within the VPC CIDR block.
func SubnetCIDRWithinVPC(vpcCIDR, subnetCIDR string) bool {
	_, vpcNet, err := net.ParseCIDR(vpcCIDR)
	if err != nil {
		return false
	}
	_, subNet, err := net.ParseCIDR(subnetCIDR)
	if err != nil {
		return false
	}
	return cidrContains(vpcNet, subNet)
}
