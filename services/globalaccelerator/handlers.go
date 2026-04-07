package globalaccelerator

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ── Generated request/response types ─────────────────────────────────────────

type Accelerator struct {
	AcceleratorArn *string `json:"AcceleratorArn,omitempty"`
	CreatedTime *time.Time `json:"CreatedTime,omitempty"`
	DnsName *string `json:"DnsName,omitempty"`
	DualStackDnsName *string `json:"DualStackDnsName,omitempty"`
	Enabled bool `json:"Enabled,omitempty"`
	Events []AcceleratorEvent `json:"Events,omitempty"`
	IpAddressType *string `json:"IpAddressType,omitempty"`
	IpSets []IpSet `json:"IpSets,omitempty"`
	LastModifiedTime *time.Time `json:"LastModifiedTime,omitempty"`
	Name *string `json:"Name,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type AcceleratorAttributes struct {
	FlowLogsEnabled bool `json:"FlowLogsEnabled,omitempty"`
	FlowLogsS3Bucket *string `json:"FlowLogsS3Bucket,omitempty"`
	FlowLogsS3Prefix *string `json:"FlowLogsS3Prefix,omitempty"`
}

type AcceleratorEvent struct {
	Message *string `json:"Message,omitempty"`
	Timestamp *time.Time `json:"Timestamp,omitempty"`
}

type AddCustomRoutingEndpointsRequest struct {
	EndpointConfigurations []CustomRoutingEndpointConfiguration `json:"EndpointConfigurations,omitempty"`
	EndpointGroupArn string `json:"EndpointGroupArn,omitempty"`
}

type AddCustomRoutingEndpointsResponse struct {
	EndpointDescriptions []CustomRoutingEndpointDescription `json:"EndpointDescriptions,omitempty"`
	EndpointGroupArn *string `json:"EndpointGroupArn,omitempty"`
}

type AddEndpointsRequest struct {
	EndpointConfigurations []EndpointConfiguration `json:"EndpointConfigurations,omitempty"`
	EndpointGroupArn string `json:"EndpointGroupArn,omitempty"`
}

type AddEndpointsResponse struct {
	EndpointDescriptions []EndpointDescription `json:"EndpointDescriptions,omitempty"`
	EndpointGroupArn *string `json:"EndpointGroupArn,omitempty"`
}

type AdvertiseByoipCidrRequest struct {
	Cidr string `json:"Cidr,omitempty"`
}

type AdvertiseByoipCidrResponse struct {
	ByoipCidr *ByoipCidr `json:"ByoipCidr,omitempty"`
}

type AllowCustomRoutingTrafficRequest struct {
	AllowAllTrafficToEndpoint bool `json:"AllowAllTrafficToEndpoint,omitempty"`
	DestinationAddresses []string `json:"DestinationAddresses,omitempty"`
	DestinationPorts []int `json:"DestinationPorts,omitempty"`
	EndpointGroupArn string `json:"EndpointGroupArn,omitempty"`
	EndpointId string `json:"EndpointId,omitempty"`
}

type Attachment struct {
	AttachmentArn *string `json:"AttachmentArn,omitempty"`
	CreatedTime *time.Time `json:"CreatedTime,omitempty"`
	LastModifiedTime *time.Time `json:"LastModifiedTime,omitempty"`
	Name *string `json:"Name,omitempty"`
	Principals []string `json:"Principals,omitempty"`
	Resources []Resource `json:"Resources,omitempty"`
}

type ByoipCidr struct {
	Cidr *string `json:"Cidr,omitempty"`
	Events []ByoipCidrEvent `json:"Events,omitempty"`
	State *string `json:"State,omitempty"`
}

type ByoipCidrEvent struct {
	Message *string `json:"Message,omitempty"`
	Timestamp *time.Time `json:"Timestamp,omitempty"`
}

type CidrAuthorizationContext struct {
	Message string `json:"Message,omitempty"`
	Signature string `json:"Signature,omitempty"`
}

type CreateAcceleratorRequest struct {
	Enabled bool `json:"Enabled,omitempty"`
	IdempotencyToken string `json:"IdempotencyToken,omitempty"`
	IpAddressType *string `json:"IpAddressType,omitempty"`
	IpAddresses []string `json:"IpAddresses,omitempty"`
	Name string `json:"Name,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
}

type CreateAcceleratorResponse struct {
	Accelerator *Accelerator `json:"Accelerator,omitempty"`
}

type CreateCrossAccountAttachmentRequest struct {
	IdempotencyToken string `json:"IdempotencyToken,omitempty"`
	Name string `json:"Name,omitempty"`
	Principals []string `json:"Principals,omitempty"`
	Resources []Resource `json:"Resources,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
}

type CreateCrossAccountAttachmentResponse struct {
	CrossAccountAttachment *Attachment `json:"CrossAccountAttachment,omitempty"`
}

type CreateCustomRoutingAcceleratorRequest struct {
	Enabled bool `json:"Enabled,omitempty"`
	IdempotencyToken string `json:"IdempotencyToken,omitempty"`
	IpAddressType *string `json:"IpAddressType,omitempty"`
	IpAddresses []string `json:"IpAddresses,omitempty"`
	Name string `json:"Name,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
}

type CreateCustomRoutingAcceleratorResponse struct {
	Accelerator *CustomRoutingAccelerator `json:"Accelerator,omitempty"`
}

type CreateCustomRoutingEndpointGroupRequest struct {
	DestinationConfigurations []CustomRoutingDestinationConfiguration `json:"DestinationConfigurations,omitempty"`
	EndpointGroupRegion string `json:"EndpointGroupRegion,omitempty"`
	IdempotencyToken string `json:"IdempotencyToken,omitempty"`
	ListenerArn string `json:"ListenerArn,omitempty"`
}

type CreateCustomRoutingEndpointGroupResponse struct {
	EndpointGroup *CustomRoutingEndpointGroup `json:"EndpointGroup,omitempty"`
}

type CreateCustomRoutingListenerRequest struct {
	AcceleratorArn string `json:"AcceleratorArn,omitempty"`
	IdempotencyToken string `json:"IdempotencyToken,omitempty"`
	PortRanges []PortRange `json:"PortRanges,omitempty"`
}

type CreateCustomRoutingListenerResponse struct {
	Listener *CustomRoutingListener `json:"Listener,omitempty"`
}

type CreateEndpointGroupRequest struct {
	EndpointConfigurations []EndpointConfiguration `json:"EndpointConfigurations,omitempty"`
	EndpointGroupRegion string `json:"EndpointGroupRegion,omitempty"`
	HealthCheckIntervalSeconds int `json:"HealthCheckIntervalSeconds,omitempty"`
	HealthCheckPath *string `json:"HealthCheckPath,omitempty"`
	HealthCheckPort int `json:"HealthCheckPort,omitempty"`
	HealthCheckProtocol *string `json:"HealthCheckProtocol,omitempty"`
	IdempotencyToken string `json:"IdempotencyToken,omitempty"`
	ListenerArn string `json:"ListenerArn,omitempty"`
	PortOverrides []PortOverride `json:"PortOverrides,omitempty"`
	ThresholdCount int `json:"ThresholdCount,omitempty"`
	TrafficDialPercentage float64 `json:"TrafficDialPercentage,omitempty"`
}

type CreateEndpointGroupResponse struct {
	EndpointGroup *EndpointGroup `json:"EndpointGroup,omitempty"`
}

type CreateListenerRequest struct {
	AcceleratorArn string `json:"AcceleratorArn,omitempty"`
	ClientAffinity *string `json:"ClientAffinity,omitempty"`
	IdempotencyToken string `json:"IdempotencyToken,omitempty"`
	PortRanges []PortRange `json:"PortRanges,omitempty"`
	Protocol string `json:"Protocol,omitempty"`
}

type CreateListenerResponse struct {
	Listener *Listener `json:"Listener,omitempty"`
}

type CrossAccountResource struct {
	AttachmentArn *string `json:"AttachmentArn,omitempty"`
	Cidr *string `json:"Cidr,omitempty"`
	EndpointId *string `json:"EndpointId,omitempty"`
}

type CustomRoutingAccelerator struct {
	AcceleratorArn *string `json:"AcceleratorArn,omitempty"`
	CreatedTime *time.Time `json:"CreatedTime,omitempty"`
	DnsName *string `json:"DnsName,omitempty"`
	Enabled bool `json:"Enabled,omitempty"`
	IpAddressType *string `json:"IpAddressType,omitempty"`
	IpSets []IpSet `json:"IpSets,omitempty"`
	LastModifiedTime *time.Time `json:"LastModifiedTime,omitempty"`
	Name *string `json:"Name,omitempty"`
	Status *string `json:"Status,omitempty"`
}

type CustomRoutingAcceleratorAttributes struct {
	FlowLogsEnabled bool `json:"FlowLogsEnabled,omitempty"`
	FlowLogsS3Bucket *string `json:"FlowLogsS3Bucket,omitempty"`
	FlowLogsS3Prefix *string `json:"FlowLogsS3Prefix,omitempty"`
}

type CustomRoutingDestinationConfiguration struct {
	FromPort int `json:"FromPort,omitempty"`
	Protocols []string `json:"Protocols,omitempty"`
	ToPort int `json:"ToPort,omitempty"`
}

type CustomRoutingDestinationDescription struct {
	FromPort int `json:"FromPort,omitempty"`
	Protocols []string `json:"Protocols,omitempty"`
	ToPort int `json:"ToPort,omitempty"`
}

type CustomRoutingEndpointConfiguration struct {
	AttachmentArn *string `json:"AttachmentArn,omitempty"`
	EndpointId *string `json:"EndpointId,omitempty"`
}

type CustomRoutingEndpointDescription struct {
	EndpointId *string `json:"EndpointId,omitempty"`
}

type CustomRoutingEndpointGroup struct {
	DestinationDescriptions []CustomRoutingDestinationDescription `json:"DestinationDescriptions,omitempty"`
	EndpointDescriptions []CustomRoutingEndpointDescription `json:"EndpointDescriptions,omitempty"`
	EndpointGroupArn *string `json:"EndpointGroupArn,omitempty"`
	EndpointGroupRegion *string `json:"EndpointGroupRegion,omitempty"`
}

type CustomRoutingListener struct {
	ListenerArn *string `json:"ListenerArn,omitempty"`
	PortRanges []PortRange `json:"PortRanges,omitempty"`
}

type DeleteAcceleratorRequest struct {
	AcceleratorArn string `json:"AcceleratorArn,omitempty"`
}

type DeleteCrossAccountAttachmentRequest struct {
	AttachmentArn string `json:"AttachmentArn,omitempty"`
}

type DeleteCustomRoutingAcceleratorRequest struct {
	AcceleratorArn string `json:"AcceleratorArn,omitempty"`
}

type DeleteCustomRoutingEndpointGroupRequest struct {
	EndpointGroupArn string `json:"EndpointGroupArn,omitempty"`
}

type DeleteCustomRoutingListenerRequest struct {
	ListenerArn string `json:"ListenerArn,omitempty"`
}

type DeleteEndpointGroupRequest struct {
	EndpointGroupArn string `json:"EndpointGroupArn,omitempty"`
}

type DeleteListenerRequest struct {
	ListenerArn string `json:"ListenerArn,omitempty"`
}

type DenyCustomRoutingTrafficRequest struct {
	DenyAllTrafficToEndpoint bool `json:"DenyAllTrafficToEndpoint,omitempty"`
	DestinationAddresses []string `json:"DestinationAddresses,omitempty"`
	DestinationPorts []int `json:"DestinationPorts,omitempty"`
	EndpointGroupArn string `json:"EndpointGroupArn,omitempty"`
	EndpointId string `json:"EndpointId,omitempty"`
}

type DeprovisionByoipCidrRequest struct {
	Cidr string `json:"Cidr,omitempty"`
}

type DeprovisionByoipCidrResponse struct {
	ByoipCidr *ByoipCidr `json:"ByoipCidr,omitempty"`
}

type DescribeAcceleratorAttributesRequest struct {
	AcceleratorArn string `json:"AcceleratorArn,omitempty"`
}

type DescribeAcceleratorAttributesResponse struct {
	AcceleratorAttributes *AcceleratorAttributes `json:"AcceleratorAttributes,omitempty"`
}

type DescribeAcceleratorRequest struct {
	AcceleratorArn string `json:"AcceleratorArn,omitempty"`
}

type DescribeAcceleratorResponse struct {
	Accelerator *Accelerator `json:"Accelerator,omitempty"`
}

type DescribeCrossAccountAttachmentRequest struct {
	AttachmentArn string `json:"AttachmentArn,omitempty"`
}

type DescribeCrossAccountAttachmentResponse struct {
	CrossAccountAttachment *Attachment `json:"CrossAccountAttachment,omitempty"`
}

type DescribeCustomRoutingAcceleratorAttributesRequest struct {
	AcceleratorArn string `json:"AcceleratorArn,omitempty"`
}

type DescribeCustomRoutingAcceleratorAttributesResponse struct {
	AcceleratorAttributes *CustomRoutingAcceleratorAttributes `json:"AcceleratorAttributes,omitempty"`
}

type DescribeCustomRoutingAcceleratorRequest struct {
	AcceleratorArn string `json:"AcceleratorArn,omitempty"`
}

type DescribeCustomRoutingAcceleratorResponse struct {
	Accelerator *CustomRoutingAccelerator `json:"Accelerator,omitempty"`
}

type DescribeCustomRoutingEndpointGroupRequest struct {
	EndpointGroupArn string `json:"EndpointGroupArn,omitempty"`
}

type DescribeCustomRoutingEndpointGroupResponse struct {
	EndpointGroup *CustomRoutingEndpointGroup `json:"EndpointGroup,omitempty"`
}

type DescribeCustomRoutingListenerRequest struct {
	ListenerArn string `json:"ListenerArn,omitempty"`
}

type DescribeCustomRoutingListenerResponse struct {
	Listener *CustomRoutingListener `json:"Listener,omitempty"`
}

type DescribeEndpointGroupRequest struct {
	EndpointGroupArn string `json:"EndpointGroupArn,omitempty"`
}

type DescribeEndpointGroupResponse struct {
	EndpointGroup *EndpointGroup `json:"EndpointGroup,omitempty"`
}

type DescribeListenerRequest struct {
	ListenerArn string `json:"ListenerArn,omitempty"`
}

type DescribeListenerResponse struct {
	Listener *Listener `json:"Listener,omitempty"`
}

type DestinationPortMapping struct {
	AcceleratorArn *string `json:"AcceleratorArn,omitempty"`
	AcceleratorSocketAddresses []SocketAddress `json:"AcceleratorSocketAddresses,omitempty"`
	DestinationSocketAddress *SocketAddress `json:"DestinationSocketAddress,omitempty"`
	DestinationTrafficState *string `json:"DestinationTrafficState,omitempty"`
	EndpointGroupArn *string `json:"EndpointGroupArn,omitempty"`
	EndpointGroupRegion *string `json:"EndpointGroupRegion,omitempty"`
	EndpointId *string `json:"EndpointId,omitempty"`
	IpAddressType *string `json:"IpAddressType,omitempty"`
}

type EndpointConfiguration struct {
	AttachmentArn *string `json:"AttachmentArn,omitempty"`
	ClientIPPreservationEnabled bool `json:"ClientIPPreservationEnabled,omitempty"`
	EndpointId *string `json:"EndpointId,omitempty"`
	Weight int `json:"Weight,omitempty"`
}

type EndpointDescription struct {
	ClientIPPreservationEnabled bool `json:"ClientIPPreservationEnabled,omitempty"`
	EndpointId *string `json:"EndpointId,omitempty"`
	HealthReason *string `json:"HealthReason,omitempty"`
	HealthState *string `json:"HealthState,omitempty"`
	Weight int `json:"Weight,omitempty"`
}

type EndpointGroup struct {
	EndpointDescriptions []EndpointDescription `json:"EndpointDescriptions,omitempty"`
	EndpointGroupArn *string `json:"EndpointGroupArn,omitempty"`
	EndpointGroupRegion *string `json:"EndpointGroupRegion,omitempty"`
	HealthCheckIntervalSeconds int `json:"HealthCheckIntervalSeconds,omitempty"`
	HealthCheckPath *string `json:"HealthCheckPath,omitempty"`
	HealthCheckPort int `json:"HealthCheckPort,omitempty"`
	HealthCheckProtocol *string `json:"HealthCheckProtocol,omitempty"`
	PortOverrides []PortOverride `json:"PortOverrides,omitempty"`
	ThresholdCount int `json:"ThresholdCount,omitempty"`
	TrafficDialPercentage float64 `json:"TrafficDialPercentage,omitempty"`
}

type EndpointIdentifier struct {
	ClientIPPreservationEnabled bool `json:"ClientIPPreservationEnabled,omitempty"`
	EndpointId string `json:"EndpointId,omitempty"`
}

type IpSet struct {
	IpAddressFamily *string `json:"IpAddressFamily,omitempty"`
	IpAddresses []string `json:"IpAddresses,omitempty"`
	IpFamily *string `json:"IpFamily,omitempty"`
}

type ListAcceleratorsRequest struct {
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListAcceleratorsResponse struct {
	Accelerators []Accelerator `json:"Accelerators,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListByoipCidrsRequest struct {
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListByoipCidrsResponse struct {
	ByoipCidrs []ByoipCidr `json:"ByoipCidrs,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListCrossAccountAttachmentsRequest struct {
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListCrossAccountAttachmentsResponse struct {
	CrossAccountAttachments []Attachment `json:"CrossAccountAttachments,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListCrossAccountResourceAccountsRequest struct {
}

type ListCrossAccountResourceAccountsResponse struct {
	ResourceOwnerAwsAccountIds []string `json:"ResourceOwnerAwsAccountIds,omitempty"`
}

type ListCrossAccountResourcesRequest struct {
	AcceleratorArn *string `json:"AcceleratorArn,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
	ResourceOwnerAwsAccountId string `json:"ResourceOwnerAwsAccountId,omitempty"`
}

type ListCrossAccountResourcesResponse struct {
	CrossAccountResources []CrossAccountResource `json:"CrossAccountResources,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListCustomRoutingAcceleratorsRequest struct {
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListCustomRoutingAcceleratorsResponse struct {
	Accelerators []CustomRoutingAccelerator `json:"Accelerators,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListCustomRoutingEndpointGroupsRequest struct {
	ListenerArn string `json:"ListenerArn,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListCustomRoutingEndpointGroupsResponse struct {
	EndpointGroups []CustomRoutingEndpointGroup `json:"EndpointGroups,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListCustomRoutingListenersRequest struct {
	AcceleratorArn string `json:"AcceleratorArn,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListCustomRoutingListenersResponse struct {
	Listeners []CustomRoutingListener `json:"Listeners,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListCustomRoutingPortMappingsByDestinationRequest struct {
	DestinationAddress string `json:"DestinationAddress,omitempty"`
	EndpointId string `json:"EndpointId,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListCustomRoutingPortMappingsByDestinationResponse struct {
	DestinationPortMappings []DestinationPortMapping `json:"DestinationPortMappings,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListCustomRoutingPortMappingsRequest struct {
	AcceleratorArn string `json:"AcceleratorArn,omitempty"`
	EndpointGroupArn *string `json:"EndpointGroupArn,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListCustomRoutingPortMappingsResponse struct {
	NextToken *string `json:"NextToken,omitempty"`
	PortMappings []PortMapping `json:"PortMappings,omitempty"`
}

type ListEndpointGroupsRequest struct {
	ListenerArn string `json:"ListenerArn,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListEndpointGroupsResponse struct {
	EndpointGroups []EndpointGroup `json:"EndpointGroups,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListListenersRequest struct {
	AcceleratorArn string `json:"AcceleratorArn,omitempty"`
	MaxResults int `json:"MaxResults,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListListenersResponse struct {
	Listeners []Listener `json:"Listeners,omitempty"`
	NextToken *string `json:"NextToken,omitempty"`
}

type ListTagsForResourceRequest struct {
	ResourceArn string `json:"ResourceArn,omitempty"`
}

type ListTagsForResourceResponse struct {
	Tags []Tag `json:"Tags,omitempty"`
}

type Listener struct {
	ClientAffinity *string `json:"ClientAffinity,omitempty"`
	ListenerArn *string `json:"ListenerArn,omitempty"`
	PortRanges []PortRange `json:"PortRanges,omitempty"`
	Protocol *string `json:"Protocol,omitempty"`
}

type PortMapping struct {
	AcceleratorPort int `json:"AcceleratorPort,omitempty"`
	DestinationSocketAddress *SocketAddress `json:"DestinationSocketAddress,omitempty"`
	DestinationTrafficState *string `json:"DestinationTrafficState,omitempty"`
	EndpointGroupArn *string `json:"EndpointGroupArn,omitempty"`
	EndpointId *string `json:"EndpointId,omitempty"`
	Protocols []string `json:"Protocols,omitempty"`
}

type PortOverride struct {
	EndpointPort int `json:"EndpointPort,omitempty"`
	ListenerPort int `json:"ListenerPort,omitempty"`
}

type PortRange struct {
	FromPort int `json:"FromPort,omitempty"`
	ToPort int `json:"ToPort,omitempty"`
}

type ProvisionByoipCidrRequest struct {
	Cidr string `json:"Cidr,omitempty"`
	CidrAuthorizationContext CidrAuthorizationContext `json:"CidrAuthorizationContext,omitempty"`
}

type ProvisionByoipCidrResponse struct {
	ByoipCidr *ByoipCidr `json:"ByoipCidr,omitempty"`
}

type RemoveCustomRoutingEndpointsRequest struct {
	EndpointGroupArn string `json:"EndpointGroupArn,omitempty"`
	EndpointIds []string `json:"EndpointIds,omitempty"`
}

type RemoveEndpointsRequest struct {
	EndpointGroupArn string `json:"EndpointGroupArn,omitempty"`
	EndpointIdentifiers []EndpointIdentifier `json:"EndpointIdentifiers,omitempty"`
}

type Resource struct {
	Cidr *string `json:"Cidr,omitempty"`
	EndpointId *string `json:"EndpointId,omitempty"`
	Region *string `json:"Region,omitempty"`
}

type SocketAddress struct {
	IpAddress *string `json:"IpAddress,omitempty"`
	Port int `json:"Port,omitempty"`
}

type Tag struct {
	Key string `json:"Key,omitempty"`
	Value string `json:"Value,omitempty"`
}

type TagResourceRequest struct {
	ResourceArn string `json:"ResourceArn,omitempty"`
	Tags []Tag `json:"Tags,omitempty"`
}

type TagResourceResponse struct {
}

type UntagResourceRequest struct {
	ResourceArn string `json:"ResourceArn,omitempty"`
	TagKeys []string `json:"TagKeys,omitempty"`
}

type UntagResourceResponse struct {
}

type UpdateAcceleratorAttributesRequest struct {
	AcceleratorArn string `json:"AcceleratorArn,omitempty"`
	FlowLogsEnabled bool `json:"FlowLogsEnabled,omitempty"`
	FlowLogsS3Bucket *string `json:"FlowLogsS3Bucket,omitempty"`
	FlowLogsS3Prefix *string `json:"FlowLogsS3Prefix,omitempty"`
}

type UpdateAcceleratorAttributesResponse struct {
	AcceleratorAttributes *AcceleratorAttributes `json:"AcceleratorAttributes,omitempty"`
}

type UpdateAcceleratorRequest struct {
	AcceleratorArn string `json:"AcceleratorArn,omitempty"`
	Enabled bool `json:"Enabled,omitempty"`
	IpAddressType *string `json:"IpAddressType,omitempty"`
	IpAddresses []string `json:"IpAddresses,omitempty"`
	Name *string `json:"Name,omitempty"`
}

type UpdateAcceleratorResponse struct {
	Accelerator *Accelerator `json:"Accelerator,omitempty"`
}

type UpdateCrossAccountAttachmentRequest struct {
	AddPrincipals []string `json:"AddPrincipals,omitempty"`
	AddResources []Resource `json:"AddResources,omitempty"`
	AttachmentArn string `json:"AttachmentArn,omitempty"`
	Name *string `json:"Name,omitempty"`
	RemovePrincipals []string `json:"RemovePrincipals,omitempty"`
	RemoveResources []Resource `json:"RemoveResources,omitempty"`
}

type UpdateCrossAccountAttachmentResponse struct {
	CrossAccountAttachment *Attachment `json:"CrossAccountAttachment,omitempty"`
}

type UpdateCustomRoutingAcceleratorAttributesRequest struct {
	AcceleratorArn string `json:"AcceleratorArn,omitempty"`
	FlowLogsEnabled bool `json:"FlowLogsEnabled,omitempty"`
	FlowLogsS3Bucket *string `json:"FlowLogsS3Bucket,omitempty"`
	FlowLogsS3Prefix *string `json:"FlowLogsS3Prefix,omitempty"`
}

type UpdateCustomRoutingAcceleratorAttributesResponse struct {
	AcceleratorAttributes *CustomRoutingAcceleratorAttributes `json:"AcceleratorAttributes,omitempty"`
}

type UpdateCustomRoutingAcceleratorRequest struct {
	AcceleratorArn string `json:"AcceleratorArn,omitempty"`
	Enabled bool `json:"Enabled,omitempty"`
	IpAddressType *string `json:"IpAddressType,omitempty"`
	IpAddresses []string `json:"IpAddresses,omitempty"`
	Name *string `json:"Name,omitempty"`
}

type UpdateCustomRoutingAcceleratorResponse struct {
	Accelerator *CustomRoutingAccelerator `json:"Accelerator,omitempty"`
}

type UpdateCustomRoutingListenerRequest struct {
	ListenerArn string `json:"ListenerArn,omitempty"`
	PortRanges []PortRange `json:"PortRanges,omitempty"`
}

type UpdateCustomRoutingListenerResponse struct {
	Listener *CustomRoutingListener `json:"Listener,omitempty"`
}

type UpdateEndpointGroupRequest struct {
	EndpointConfigurations []EndpointConfiguration `json:"EndpointConfigurations,omitempty"`
	EndpointGroupArn string `json:"EndpointGroupArn,omitempty"`
	HealthCheckIntervalSeconds int `json:"HealthCheckIntervalSeconds,omitempty"`
	HealthCheckPath *string `json:"HealthCheckPath,omitempty"`
	HealthCheckPort int `json:"HealthCheckPort,omitempty"`
	HealthCheckProtocol *string `json:"HealthCheckProtocol,omitempty"`
	PortOverrides []PortOverride `json:"PortOverrides,omitempty"`
	ThresholdCount int `json:"ThresholdCount,omitempty"`
	TrafficDialPercentage float64 `json:"TrafficDialPercentage,omitempty"`
}

type UpdateEndpointGroupResponse struct {
	EndpointGroup *EndpointGroup `json:"EndpointGroup,omitempty"`
}

type UpdateListenerRequest struct {
	ClientAffinity *string `json:"ClientAffinity,omitempty"`
	ListenerArn string `json:"ListenerArn,omitempty"`
	PortRanges []PortRange `json:"PortRanges,omitempty"`
	Protocol *string `json:"Protocol,omitempty"`
}

type UpdateListenerResponse struct {
	Listener *Listener `json:"Listener,omitempty"`
}

type WithdrawByoipCidrRequest struct {
	Cidr string `json:"Cidr,omitempty"`
}

type WithdrawByoipCidrResponse struct {
	ByoipCidr *ByoipCidr `json:"ByoipCidr,omitempty"`
}



// ── Handler helpers ──────────────────────────────────────────────────────────

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: body, Format: service.FormatJSON}, nil
}

func jsonErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
}

func parseJSON(body []byte, v any) *service.AWSError {
	if len(body) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, v); err != nil {
		return service.NewAWSError("InvalidParameterException",
			"Request body is not valid JSON.", http.StatusBadRequest)
	}
	return nil
}

// ── Handlers ─────────────────────────────────────────────────────────────────

func handleAddCustomRoutingEndpoints(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req AddCustomRoutingEndpointsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement AddCustomRoutingEndpoints business logic
	return jsonOK(map[string]any{"status": "ok", "action": "AddCustomRoutingEndpoints"})
}

func handleAddEndpoints(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req AddEndpointsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement AddEndpoints business logic
	return jsonOK(map[string]any{"status": "ok", "action": "AddEndpoints"})
}

func handleAdvertiseByoipCidr(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req AdvertiseByoipCidrRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement AdvertiseByoipCidr business logic
	return jsonOK(map[string]any{"status": "ok", "action": "AdvertiseByoipCidr"})
}

func handleAllowCustomRoutingTraffic(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req AllowCustomRoutingTrafficRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement AllowCustomRoutingTraffic business logic
	return jsonOK(map[string]any{"status": "ok", "action": "AllowCustomRoutingTraffic"})
}

func handleCreateAccelerator(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateAcceleratorRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateAccelerator business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateAccelerator"})
}

func handleCreateCrossAccountAttachment(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateCrossAccountAttachmentRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateCrossAccountAttachment business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateCrossAccountAttachment"})
}

func handleCreateCustomRoutingAccelerator(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateCustomRoutingAcceleratorRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateCustomRoutingAccelerator business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateCustomRoutingAccelerator"})
}

func handleCreateCustomRoutingEndpointGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateCustomRoutingEndpointGroupRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateCustomRoutingEndpointGroup business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateCustomRoutingEndpointGroup"})
}

func handleCreateCustomRoutingListener(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateCustomRoutingListenerRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateCustomRoutingListener business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateCustomRoutingListener"})
}

func handleCreateEndpointGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateEndpointGroupRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateEndpointGroup business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateEndpointGroup"})
}

func handleCreateListener(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req CreateListenerRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement CreateListener business logic
	return jsonOK(map[string]any{"status": "ok", "action": "CreateListener"})
}

func handleDeleteAccelerator(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteAcceleratorRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteAccelerator business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteAccelerator"})
}

func handleDeleteCrossAccountAttachment(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteCrossAccountAttachmentRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteCrossAccountAttachment business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteCrossAccountAttachment"})
}

func handleDeleteCustomRoutingAccelerator(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteCustomRoutingAcceleratorRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteCustomRoutingAccelerator business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteCustomRoutingAccelerator"})
}

func handleDeleteCustomRoutingEndpointGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteCustomRoutingEndpointGroupRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteCustomRoutingEndpointGroup business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteCustomRoutingEndpointGroup"})
}

func handleDeleteCustomRoutingListener(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteCustomRoutingListenerRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteCustomRoutingListener business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteCustomRoutingListener"})
}

func handleDeleteEndpointGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteEndpointGroupRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteEndpointGroup business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteEndpointGroup"})
}

func handleDeleteListener(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeleteListenerRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeleteListener business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeleteListener"})
}

func handleDenyCustomRoutingTraffic(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DenyCustomRoutingTrafficRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DenyCustomRoutingTraffic business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DenyCustomRoutingTraffic"})
}

func handleDeprovisionByoipCidr(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DeprovisionByoipCidrRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DeprovisionByoipCidr business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DeprovisionByoipCidr"})
}

func handleDescribeAccelerator(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeAcceleratorRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeAccelerator business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeAccelerator"})
}

func handleDescribeAcceleratorAttributes(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeAcceleratorAttributesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeAcceleratorAttributes business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeAcceleratorAttributes"})
}

func handleDescribeCrossAccountAttachment(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeCrossAccountAttachmentRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeCrossAccountAttachment business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeCrossAccountAttachment"})
}

func handleDescribeCustomRoutingAccelerator(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeCustomRoutingAcceleratorRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeCustomRoutingAccelerator business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeCustomRoutingAccelerator"})
}

func handleDescribeCustomRoutingAcceleratorAttributes(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeCustomRoutingAcceleratorAttributesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeCustomRoutingAcceleratorAttributes business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeCustomRoutingAcceleratorAttributes"})
}

func handleDescribeCustomRoutingEndpointGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeCustomRoutingEndpointGroupRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeCustomRoutingEndpointGroup business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeCustomRoutingEndpointGroup"})
}

func handleDescribeCustomRoutingListener(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeCustomRoutingListenerRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeCustomRoutingListener business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeCustomRoutingListener"})
}

func handleDescribeEndpointGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeEndpointGroupRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeEndpointGroup business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeEndpointGroup"})
}

func handleDescribeListener(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req DescribeListenerRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement DescribeListener business logic
	return jsonOK(map[string]any{"status": "ok", "action": "DescribeListener"})
}

func handleListAccelerators(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListAcceleratorsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListAccelerators business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListAccelerators"})
}

func handleListByoipCidrs(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListByoipCidrsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListByoipCidrs business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListByoipCidrs"})
}

func handleListCrossAccountAttachments(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListCrossAccountAttachmentsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListCrossAccountAttachments business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListCrossAccountAttachments"})
}

func handleListCrossAccountResourceAccounts(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListCrossAccountResourceAccountsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListCrossAccountResourceAccounts business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListCrossAccountResourceAccounts"})
}

func handleListCrossAccountResources(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListCrossAccountResourcesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListCrossAccountResources business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListCrossAccountResources"})
}

func handleListCustomRoutingAccelerators(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListCustomRoutingAcceleratorsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListCustomRoutingAccelerators business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListCustomRoutingAccelerators"})
}

func handleListCustomRoutingEndpointGroups(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListCustomRoutingEndpointGroupsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListCustomRoutingEndpointGroups business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListCustomRoutingEndpointGroups"})
}

func handleListCustomRoutingListeners(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListCustomRoutingListenersRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListCustomRoutingListeners business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListCustomRoutingListeners"})
}

func handleListCustomRoutingPortMappings(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListCustomRoutingPortMappingsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListCustomRoutingPortMappings business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListCustomRoutingPortMappings"})
}

func handleListCustomRoutingPortMappingsByDestination(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListCustomRoutingPortMappingsByDestinationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListCustomRoutingPortMappingsByDestination business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListCustomRoutingPortMappingsByDestination"})
}

func handleListEndpointGroups(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListEndpointGroupsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListEndpointGroups business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListEndpointGroups"})
}

func handleListListeners(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListListenersRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListListeners business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListListeners"})
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ListTagsForResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ListTagsForResource business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ListTagsForResource"})
}

func handleProvisionByoipCidr(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req ProvisionByoipCidrRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement ProvisionByoipCidr business logic
	return jsonOK(map[string]any{"status": "ok", "action": "ProvisionByoipCidr"})
}

func handleRemoveCustomRoutingEndpoints(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req RemoveCustomRoutingEndpointsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement RemoveCustomRoutingEndpoints business logic
	return jsonOK(map[string]any{"status": "ok", "action": "RemoveCustomRoutingEndpoints"})
}

func handleRemoveEndpoints(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req RemoveEndpointsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement RemoveEndpoints business logic
	return jsonOK(map[string]any{"status": "ok", "action": "RemoveEndpoints"})
}

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req TagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement TagResource business logic
	return jsonOK(map[string]any{"status": "ok", "action": "TagResource"})
}

func handleUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UntagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UntagResource business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UntagResource"})
}

func handleUpdateAccelerator(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateAcceleratorRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateAccelerator business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateAccelerator"})
}

func handleUpdateAcceleratorAttributes(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateAcceleratorAttributesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateAcceleratorAttributes business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateAcceleratorAttributes"})
}

func handleUpdateCrossAccountAttachment(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateCrossAccountAttachmentRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateCrossAccountAttachment business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateCrossAccountAttachment"})
}

func handleUpdateCustomRoutingAccelerator(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateCustomRoutingAcceleratorRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateCustomRoutingAccelerator business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateCustomRoutingAccelerator"})
}

func handleUpdateCustomRoutingAcceleratorAttributes(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateCustomRoutingAcceleratorAttributesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateCustomRoutingAcceleratorAttributes business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateCustomRoutingAcceleratorAttributes"})
}

func handleUpdateCustomRoutingListener(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateCustomRoutingListenerRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateCustomRoutingListener business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateCustomRoutingListener"})
}

func handleUpdateEndpointGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateEndpointGroupRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateEndpointGroup business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateEndpointGroup"})
}

func handleUpdateListener(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req UpdateListenerRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement UpdateListener business logic
	return jsonOK(map[string]any{"status": "ok", "action": "UpdateListener"})
}

func handleWithdrawByoipCidr(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req WithdrawByoipCidrRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	// TODO: implement WithdrawByoipCidr business logic
	return jsonOK(map[string]any{"status": "ok", "action": "WithdrawByoipCidr"})
}

