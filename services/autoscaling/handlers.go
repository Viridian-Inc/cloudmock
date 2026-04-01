package autoscaling

import (
	"crypto/rand"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/neureaux/cloudmock/pkg/eventbus"
	"github.com/neureaux/cloudmock/pkg/service"
)

const asXmlns = "http://autoscaling.amazonaws.com/doc/2011-01-01/"

// ---- shared XML types ----

type xmlResponseMetadata struct {
	RequestID string `xml:"RequestId"`
}

type xmlTag struct {
	Key               string `xml:"Key"`
	Value             string `xml:"Value"`
	ResourceId        string `xml:"ResourceId"`
	ResourceType      string `xml:"ResourceType"`
	PropagateAtLaunch bool   `xml:"PropagateAtLaunch"`
}

// ---- LaunchConfiguration XML types ----

type xmlLaunchConfiguration struct {
	LaunchConfigurationName string   `xml:"LaunchConfigurationName"`
	LaunchConfigurationARN  string   `xml:"LaunchConfigurationARN"`
	ImageId                 string   `xml:"ImageId"`
	InstanceType            string   `xml:"InstanceType"`
	KeyName                 string   `xml:"KeyName,omitempty"`
	SecurityGroups          []string `xml:"SecurityGroups>member"`
	CreatedTime             string   `xml:"CreatedTime"`
}

func toXMLLaunchConfiguration(lc *LaunchConfiguration) xmlLaunchConfiguration {
	return xmlLaunchConfiguration{
		LaunchConfigurationName: lc.Name,
		LaunchConfigurationARN:  lc.ARN,
		ImageId:                 lc.ImageID,
		InstanceType:            lc.InstanceType,
		KeyName:                 lc.KeyName,
		SecurityGroups:          lc.SecurityGroups,
		CreatedTime:             lc.CreatedTime.Format("2006-01-02T15:04:05Z"),
	}
}

// ---- CreateLaunchConfiguration ----

type xmlCreateLaunchConfigurationResponse struct {
	XMLName xml.Name            `xml:"CreateLaunchConfigurationResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleCreateLaunchConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	name := form.Get("LaunchConfigurationName")
	if name == "" {
		return xmlErr(service.ErrValidation("LaunchConfigurationName is required."))
	}
	imageID := form.Get("ImageId")
	if imageID == "" {
		return xmlErr(service.ErrValidation("ImageId is required."))
	}
	instanceType := form.Get("InstanceType")
	if instanceType == "" {
		return xmlErr(service.ErrValidation("InstanceType is required."))
	}
	keyName := form.Get("KeyName")
	userData := form.Get("UserData")
	iamProfile := form.Get("IamInstanceProfile")
	sgs := parseMemberList(form, "SecurityGroups")

	_, ok := store.CreateLaunchConfiguration(name, imageID, instanceType, keyName, userData, iamProfile, sgs)
	if !ok {
		return xmlErr(service.NewAWSError("AlreadyExists",
			"Launch Configuration by this name already exists - "+name, http.StatusConflict))
	}

	return xmlOK(&xmlCreateLaunchConfigurationResponse{
		Xmlns: asXmlns,
		Meta:  xmlResponseMetadata{RequestID: newUUIDHandler()},
	})
}

// ---- DescribeLaunchConfigurations ----

type xmlDescribeLaunchConfigurationsResponse struct {
	XMLName xml.Name                             `xml:"DescribeLaunchConfigurationsResponse"`
	Xmlns   string                               `xml:"xmlns,attr"`
	Result  xmlDescribeLaunchConfigurationsResult `xml:"DescribeLaunchConfigurationsResult"`
	Meta    xmlResponseMetadata                  `xml:"ResponseMetadata"`
}

type xmlDescribeLaunchConfigurationsResult struct {
	LaunchConfigurations []xmlLaunchConfiguration `xml:"LaunchConfigurations>member"`
}

func handleDescribeLaunchConfigurations(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	names := parseMemberList(form, "LaunchConfigurationNames")

	lcs := store.ListLaunchConfigurations(names)

	xmlLCs := make([]xmlLaunchConfiguration, 0, len(lcs))
	for _, lc := range lcs {
		xmlLCs = append(xmlLCs, toXMLLaunchConfiguration(lc))
	}

	return xmlOK(&xmlDescribeLaunchConfigurationsResponse{
		Xmlns:  asXmlns,
		Result: xmlDescribeLaunchConfigurationsResult{LaunchConfigurations: xmlLCs},
		Meta:   xmlResponseMetadata{RequestID: newUUIDHandler()},
	})
}

// ---- DeleteLaunchConfiguration ----

type xmlDeleteLaunchConfigurationResponse struct {
	XMLName xml.Name            `xml:"DeleteLaunchConfigurationResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleDeleteLaunchConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	name := form.Get("LaunchConfigurationName")
	if name == "" {
		return xmlErr(service.ErrValidation("LaunchConfigurationName is required."))
	}

	if !store.DeleteLaunchConfiguration(name) {
		return xmlErr(service.NewAWSError("ResourceNotFound",
			"Launch Configuration '"+name+"' not found.", http.StatusNotFound))
	}

	return xmlOK(&xmlDeleteLaunchConfigurationResponse{
		Xmlns: asXmlns,
		Meta:  xmlResponseMetadata{RequestID: newUUIDHandler()},
	})
}

// ---- AutoScalingGroup XML types ----

type xmlAutoScalingGroup struct {
	AutoScalingGroupName string                  `xml:"AutoScalingGroupName"`
	AutoScalingGroupARN  string                  `xml:"AutoScalingGroupARN"`
	LaunchConfigurationName string               `xml:"LaunchConfigurationName,omitempty"`
	MinSize              int                     `xml:"MinSize"`
	MaxSize              int                     `xml:"MaxSize"`
	DesiredCapacity      int                     `xml:"DesiredCapacity"`
	DefaultCooldown      int                     `xml:"DefaultCooldown"`
	AvailabilityZones    []string                `xml:"AvailabilityZones>member"`
	TargetGroupARNs      []string                `xml:"TargetGroupARNs>member"`
	HealthCheckType      string                  `xml:"HealthCheckType"`
	HealthCheckGracePeriod int                   `xml:"HealthCheckGracePeriod"`
	VPCZoneIdentifier    string                  `xml:"VPCZoneIdentifier,omitempty"`
	Instances            []xmlAutoScalingInstance `xml:"Instances>member"`
	CreatedTime          string                  `xml:"CreatedTime"`
	Tags                 []xmlTag                `xml:"Tags>member"`
}

type xmlAutoScalingInstance struct {
	InstanceId         string `xml:"InstanceId"`
	AutoScalingGroupName string `xml:"AutoScalingGroupName"`
	AvailabilityZone   string `xml:"AvailabilityZone"`
	LifecycleState     string `xml:"LifecycleState"`
	HealthStatus       string `xml:"HealthStatus"`
	LaunchConfigurationName string `xml:"LaunchConfigurationName,omitempty"`
	ProtectedFromScaleIn bool  `xml:"ProtectedFromScaleIn"`
}

func toXMLAutoScalingGroup(asg *AutoScalingGroup) xmlAutoScalingGroup {
	instances := make([]xmlAutoScalingInstance, 0, len(asg.Instances))
	for _, inst := range asg.Instances {
		instances = append(instances, xmlAutoScalingInstance{
			InstanceId:              inst.InstanceID,
			AutoScalingGroupName:    inst.AutoScalingGroupName,
			AvailabilityZone:        inst.AvailabilityZone,
			LifecycleState:          inst.LifecycleState,
			HealthStatus:            inst.HealthStatus,
			LaunchConfigurationName: inst.LaunchConfigName,
			ProtectedFromScaleIn:    inst.ProtectedFromScaleIn,
		})
	}
	tags := make([]xmlTag, 0, len(asg.Tags))
	for k, v := range asg.Tags {
		tags = append(tags, xmlTag{Key: k, Value: v, ResourceId: asg.Name, ResourceType: "auto-scaling-group"})
	}
	return xmlAutoScalingGroup{
		AutoScalingGroupName:    asg.Name,
		AutoScalingGroupARN:     asg.ARN,
		LaunchConfigurationName: asg.LaunchConfigName,
		MinSize:                 asg.MinSize,
		MaxSize:                 asg.MaxSize,
		DesiredCapacity:         asg.DesiredCapacity,
		DefaultCooldown:         asg.DefaultCooldown,
		AvailabilityZones:       asg.AvailabilityZones,
		TargetGroupARNs:         asg.TargetGroupARNs,
		HealthCheckType:         asg.HealthCheckType,
		HealthCheckGracePeriod:  asg.HealthCheckGracePeriod,
		VPCZoneIdentifier:       asg.VPCZoneIdentifier,
		Instances:               instances,
		CreatedTime:             asg.CreatedTime.Format("2006-01-02T15:04:05Z"),
		Tags:                    tags,
	}
}

// ---- CreateAutoScalingGroup ----

type xmlCreateAutoScalingGroupResponse struct {
	XMLName xml.Name            `xml:"CreateAutoScalingGroupResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleCreateAutoScalingGroup(ctx *service.RequestContext, store *Store, locator ServiceLocator, bus *eventbus.Bus) (*service.Response, error) {
	form := parseForm(ctx)
	name := form.Get("AutoScalingGroupName")
	if name == "" {
		return xmlErr(service.ErrValidation("AutoScalingGroupName is required."))
	}
	lcName := form.Get("LaunchConfigurationName")
	minSize := 0
	if v := form.Get("MinSize"); v != "" {
		minSize, _ = strconv.Atoi(v)
	}
	maxSize := 1
	if v := form.Get("MaxSize"); v != "" {
		maxSize, _ = strconv.Atoi(v)
	}
	desiredCapacity := minSize
	if v := form.Get("DesiredCapacity"); v != "" {
		desiredCapacity, _ = strconv.Atoi(v)
	}
	cooldown := 0
	if v := form.Get("DefaultCooldown"); v != "" {
		cooldown, _ = strconv.Atoi(v)
	}
	hcGrace := 0
	if v := form.Get("HealthCheckGracePeriod"); v != "" {
		hcGrace, _ = strconv.Atoi(v)
	}
	healthCheckType := form.Get("HealthCheckType")
	vpcZoneID := form.Get("VPCZoneIdentifier")
	azs := parseMemberList(form, "AvailabilityZones")
	tgARNs := parseMemberList(form, "TargetGroupARNs")
	tags := parseASGTags(form)

	tagMap := make(map[string]string, len(tags))
	for _, t := range tags {
		tagMap[t.Key] = t.Value
	}

	_, ok := store.CreateAutoScalingGroupWithEC2(name, lcName, vpcZoneID, healthCheckType,
		minSize, maxSize, desiredCapacity, cooldown, hcGrace, azs, tgARNs, tagMap, locator, bus)
	if !ok {
		return xmlErr(service.NewAWSError("AlreadyExists",
			"AutoScalingGroup by this name already exists - "+name, http.StatusConflict))
	}

	return xmlOK(&xmlCreateAutoScalingGroupResponse{
		Xmlns: asXmlns,
		Meta:  xmlResponseMetadata{RequestID: newUUIDHandler()},
	})
}

// ---- DescribeAutoScalingGroups ----

type xmlDescribeAutoScalingGroupsResponse struct {
	XMLName xml.Name                          `xml:"DescribeAutoScalingGroupsResponse"`
	Xmlns   string                            `xml:"xmlns,attr"`
	Result  xmlDescribeAutoScalingGroupsResult `xml:"DescribeAutoScalingGroupsResult"`
	Meta    xmlResponseMetadata               `xml:"ResponseMetadata"`
}

type xmlDescribeAutoScalingGroupsResult struct {
	AutoScalingGroups []xmlAutoScalingGroup `xml:"AutoScalingGroups>member"`
}

func handleDescribeAutoScalingGroups(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	names := parseMemberList(form, "AutoScalingGroupNames")

	asgs := store.ListAutoScalingGroups(names)

	xmlASGs := make([]xmlAutoScalingGroup, 0, len(asgs))
	for _, asg := range asgs {
		xmlASGs = append(xmlASGs, toXMLAutoScalingGroup(asg))
	}

	return xmlOK(&xmlDescribeAutoScalingGroupsResponse{
		Xmlns:  asXmlns,
		Result: xmlDescribeAutoScalingGroupsResult{AutoScalingGroups: xmlASGs},
		Meta:   xmlResponseMetadata{RequestID: newUUIDHandler()},
	})
}

// ---- UpdateAutoScalingGroup ----

type xmlUpdateAutoScalingGroupResponse struct {
	XMLName xml.Name            `xml:"UpdateAutoScalingGroupResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleUpdateAutoScalingGroup(ctx *service.RequestContext, store *Store, locator ServiceLocator, bus *eventbus.Bus) (*service.Response, error) {
	form := parseForm(ctx)
	name := form.Get("AutoScalingGroupName")
	if name == "" {
		return xmlErr(service.ErrValidation("AutoScalingGroupName is required."))
	}
	lcName := form.Get("LaunchConfigurationName")
	vpcZoneID := form.Get("VPCZoneIdentifier")
	healthCheckType := form.Get("HealthCheckType")

	minSize := -1
	if v := form.Get("MinSize"); v != "" {
		minSize, _ = strconv.Atoi(v)
	}
	maxSize := -1
	if v := form.Get("MaxSize"); v != "" {
		maxSize, _ = strconv.Atoi(v)
	}
	desiredCapacity := -1
	if v := form.Get("DesiredCapacity"); v != "" {
		desiredCapacity, _ = strconv.Atoi(v)
	}
	cooldown := 0
	if v := form.Get("DefaultCooldown"); v != "" {
		cooldown, _ = strconv.Atoi(v)
	}
	hcGrace := 0
	if v := form.Get("HealthCheckGracePeriod"); v != "" {
		hcGrace, _ = strconv.Atoi(v)
	}

	_, ok := store.UpdateAutoScalingGroupWithEC2(name, lcName, vpcZoneID, healthCheckType,
		minSize, maxSize, desiredCapacity, cooldown, hcGrace, locator, bus)
	if !ok {
		return xmlErr(service.NewAWSError("ResourceNotFound",
			"AutoScalingGroup '"+name+"' not found.", http.StatusNotFound))
	}

	return xmlOK(&xmlUpdateAutoScalingGroupResponse{
		Xmlns: asXmlns,
		Meta:  xmlResponseMetadata{RequestID: newUUIDHandler()},
	})
}

// ---- DeleteAutoScalingGroup ----

type xmlDeleteAutoScalingGroupResponse struct {
	XMLName xml.Name            `xml:"DeleteAutoScalingGroupResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleDeleteAutoScalingGroup(ctx *service.RequestContext, store *Store, locator ServiceLocator, bus *eventbus.Bus) (*service.Response, error) {
	form := parseForm(ctx)
	name := form.Get("AutoScalingGroupName")
	if name == "" {
		return xmlErr(service.ErrValidation("AutoScalingGroupName is required."))
	}

	if !store.DeleteAutoScalingGroupWithEC2(name, locator, bus) {
		return xmlErr(service.NewAWSError("ResourceNotFound",
			"AutoScalingGroup '"+name+"' not found.", http.StatusNotFound))
	}

	return xmlOK(&xmlDeleteAutoScalingGroupResponse{
		Xmlns: asXmlns,
		Meta:  xmlResponseMetadata{RequestID: newUUIDHandler()},
	})
}

// ---- SetDesiredCapacity ----

type xmlSetDesiredCapacityResponse struct {
	XMLName xml.Name            `xml:"SetDesiredCapacityResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleSetDesiredCapacity(ctx *service.RequestContext, store *Store, locator ServiceLocator, bus *eventbus.Bus) (*service.Response, error) {
	form := parseForm(ctx)
	name := form.Get("AutoScalingGroupName")
	if name == "" {
		return xmlErr(service.ErrValidation("AutoScalingGroupName is required."))
	}
	capacity := 0
	if v := form.Get("DesiredCapacity"); v != "" {
		capacity, _ = strconv.Atoi(v)
	}

	if !store.SetDesiredCapacityWithEC2(name, capacity, locator, bus) {
		return xmlErr(service.NewAWSError("ResourceNotFound",
			"AutoScalingGroup '"+name+"' not found.", http.StatusNotFound))
	}

	return xmlOK(&xmlSetDesiredCapacityResponse{
		Xmlns: asXmlns,
		Meta:  xmlResponseMetadata{RequestID: newUUIDHandler()},
	})
}

// ---- DescribeAutoScalingInstances ----

type xmlDescribeAutoScalingInstancesResponse struct {
	XMLName xml.Name                             `xml:"DescribeAutoScalingInstancesResponse"`
	Xmlns   string                               `xml:"xmlns,attr"`
	Result  xmlDescribeAutoScalingInstancesResult `xml:"DescribeAutoScalingInstancesResult"`
	Meta    xmlResponseMetadata                  `xml:"ResponseMetadata"`
}

type xmlDescribeAutoScalingInstancesResult struct {
	AutoScalingInstances []xmlAutoScalingInstance `xml:"AutoScalingInstances>member"`
}

func handleDescribeAutoScalingInstances(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	instances := store.ListAutoScalingInstances()

	xmlInstances := make([]xmlAutoScalingInstance, 0, len(instances))
	for _, inst := range instances {
		xmlInstances = append(xmlInstances, xmlAutoScalingInstance{
			InstanceId:              inst.InstanceID,
			AutoScalingGroupName:    inst.AutoScalingGroupName,
			AvailabilityZone:        inst.AvailabilityZone,
			LifecycleState:          inst.LifecycleState,
			HealthStatus:            inst.HealthStatus,
			LaunchConfigurationName: inst.LaunchConfigName,
			ProtectedFromScaleIn:    inst.ProtectedFromScaleIn,
		})
	}

	return xmlOK(&xmlDescribeAutoScalingInstancesResponse{
		Xmlns:  asXmlns,
		Result: xmlDescribeAutoScalingInstancesResult{AutoScalingInstances: xmlInstances},
		Meta:   xmlResponseMetadata{RequestID: newUUIDHandler()},
	})
}

// ---- AttachInstances ----

type xmlAttachInstancesResponse struct {
	XMLName xml.Name            `xml:"AttachInstancesResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleAttachInstances(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	asgName := form.Get("AutoScalingGroupName")
	if asgName == "" {
		return xmlErr(service.ErrValidation("AutoScalingGroupName is required."))
	}
	instanceIDs := parseMemberList(form, "InstanceIds")

	if !store.AttachInstances(asgName, instanceIDs) {
		return xmlErr(service.NewAWSError("ResourceNotFound",
			"AutoScalingGroup '"+asgName+"' not found.", http.StatusNotFound))
	}

	return xmlOK(&xmlAttachInstancesResponse{
		Xmlns: asXmlns,
		Meta:  xmlResponseMetadata{RequestID: newUUIDHandler()},
	})
}

// ---- DetachInstances ----

type xmlDetachInstancesResponse struct {
	XMLName xml.Name                `xml:"DetachInstancesResponse"`
	Xmlns   string                  `xml:"xmlns,attr"`
	Result  xmlDetachInstancesResult `xml:"DetachInstancesResult"`
	Meta    xmlResponseMetadata     `xml:"ResponseMetadata"`
}

type xmlDetachInstancesResult struct {
	Activities []xmlActivity `xml:"Activities>member"`
}

type xmlActivity struct {
	ActivityId string `xml:"ActivityId"`
	StatusCode string `xml:"StatusCode"`
}

func handleDetachInstances(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	asgName := form.Get("AutoScalingGroupName")
	if asgName == "" {
		return xmlErr(service.ErrValidation("AutoScalingGroupName is required."))
	}
	instanceIDs := parseMemberList(form, "InstanceIds")
	decrement := form.Get("ShouldDecrementDesiredCapacity") == "true"

	_, ok := store.DetachInstances(asgName, instanceIDs, decrement)
	if !ok {
		return xmlErr(service.NewAWSError("ResourceNotFound",
			"AutoScalingGroup '"+asgName+"' not found.", http.StatusNotFound))
	}

	activities := make([]xmlActivity, 0, len(instanceIDs))
	for range instanceIDs {
		activities = append(activities, xmlActivity{
			ActivityId: newUUIDHandler(),
			StatusCode: "InProgress",
		})
	}

	return xmlOK(&xmlDetachInstancesResponse{
		Xmlns:  asXmlns,
		Result: xmlDetachInstancesResult{Activities: activities},
		Meta:   xmlResponseMetadata{RequestID: newUUIDHandler()},
	})
}

// ---- ScalingPolicy XML types ----

type xmlScalingPolicy struct {
	PolicyName           string  `xml:"PolicyName"`
	PolicyARN            string  `xml:"PolicyARN"`
	AutoScalingGroupName string  `xml:"AutoScalingGroupName"`
	PolicyType           string  `xml:"PolicyType"`
	AdjustmentType       string  `xml:"AdjustmentType,omitempty"`
	ScalingAdjustment    int     `xml:"ScalingAdjustment,omitempty"`
	Cooldown             int     `xml:"Cooldown,omitempty"`
	Enabled              bool    `xml:"Enabled"`
}

func toXMLScalingPolicy(p *ScalingPolicy) xmlScalingPolicy {
	return xmlScalingPolicy{
		PolicyName:           p.Name,
		PolicyARN:            p.ARN,
		AutoScalingGroupName: p.AutoScalingGroupName,
		PolicyType:           p.PolicyType,
		AdjustmentType:       p.AdjustmentType,
		ScalingAdjustment:    p.ScalingAdjustment,
		Cooldown:             p.Cooldown,
		Enabled:              p.Enabled,
	}
}

// ---- PutScalingPolicy ----

type xmlPutScalingPolicyResponse struct {
	XMLName xml.Name                 `xml:"PutScalingPolicyResponse"`
	Xmlns   string                   `xml:"xmlns,attr"`
	Result  xmlPutScalingPolicyResult `xml:"PutScalingPolicyResult"`
	Meta    xmlResponseMetadata      `xml:"ResponseMetadata"`
}

type xmlPutScalingPolicyResult struct {
	PolicyARN string `xml:"PolicyARN"`
}

func handlePutScalingPolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	asgName := form.Get("AutoScalingGroupName")
	if asgName == "" {
		return xmlErr(service.ErrValidation("AutoScalingGroupName is required."))
	}
	policyName := form.Get("PolicyName")
	if policyName == "" {
		return xmlErr(service.ErrValidation("PolicyName is required."))
	}
	policyType := form.Get("PolicyType")
	adjustmentType := form.Get("AdjustmentType")
	scalingAdj := 0
	if v := form.Get("ScalingAdjustment"); v != "" {
		scalingAdj, _ = strconv.Atoi(v)
	}
	cooldown := 0
	if v := form.Get("Cooldown"); v != "" {
		cooldown, _ = strconv.Atoi(v)
	}

	pol, ok := store.PutScalingPolicy(asgName, policyName, policyType, adjustmentType, scalingAdj, cooldown, 0, "")
	if !ok {
		return xmlErr(service.NewAWSError("ResourceNotFound",
			"AutoScalingGroup '"+asgName+"' not found.", http.StatusNotFound))
	}

	return xmlOK(&xmlPutScalingPolicyResponse{
		Xmlns:  asXmlns,
		Result: xmlPutScalingPolicyResult{PolicyARN: pol.ARN},
		Meta:   xmlResponseMetadata{RequestID: newUUIDHandler()},
	})
}

// ---- DescribePolicies ----

type xmlDescribePoliciesResponse struct {
	XMLName xml.Name                 `xml:"DescribePoliciesResponse"`
	Xmlns   string                   `xml:"xmlns,attr"`
	Result  xmlDescribePoliciesResult `xml:"DescribePoliciesResult"`
	Meta    xmlResponseMetadata      `xml:"ResponseMetadata"`
}

type xmlDescribePoliciesResult struct {
	ScalingPolicies []xmlScalingPolicy `xml:"ScalingPolicies>member"`
}

func handleDescribePolicies(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	asgName := form.Get("AutoScalingGroupName")
	policyNames := parseMemberList(form, "PolicyNames")

	policies := store.ListScalingPolicies(asgName, policyNames)

	xmlPols := make([]xmlScalingPolicy, 0, len(policies))
	for _, p := range policies {
		xmlPols = append(xmlPols, toXMLScalingPolicy(p))
	}

	return xmlOK(&xmlDescribePoliciesResponse{
		Xmlns:  asXmlns,
		Result: xmlDescribePoliciesResult{ScalingPolicies: xmlPols},
		Meta:   xmlResponseMetadata{RequestID: newUUIDHandler()},
	})
}

// ---- DeletePolicy ----

type xmlDeletePolicyResponse struct {
	XMLName xml.Name            `xml:"DeletePolicyResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleDeletePolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	asgName := form.Get("AutoScalingGroupName")
	if asgName == "" {
		return xmlErr(service.ErrValidation("AutoScalingGroupName is required."))
	}
	policyName := form.Get("PolicyName")
	if policyName == "" {
		return xmlErr(service.ErrValidation("PolicyName is required."))
	}

	if !store.DeleteScalingPolicy(asgName, policyName) {
		return xmlErr(service.NewAWSError("ResourceNotFound",
			"Scaling policy '"+policyName+"' not found.", http.StatusNotFound))
	}

	return xmlOK(&xmlDeletePolicyResponse{
		Xmlns: asXmlns,
		Meta:  xmlResponseMetadata{RequestID: newUUIDHandler()},
	})
}

// ---- CreateOrUpdateTags ----

type xmlCreateOrUpdateTagsResponse struct {
	XMLName xml.Name            `xml:"CreateOrUpdateTagsResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleCreateOrUpdateTags(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	tags := parseASGTags(form)

	store.CreateOrUpdateTags(tags)

	return xmlOK(&xmlCreateOrUpdateTagsResponse{
		Xmlns: asXmlns,
		Meta:  xmlResponseMetadata{RequestID: newUUIDHandler()},
	})
}

// ---- DescribeTags ----

type xmlDescribeTagsResponse struct {
	XMLName xml.Name             `xml:"DescribeTagsResponse"`
	Xmlns   string               `xml:"xmlns,attr"`
	Result  xmlDescribeTagsResult `xml:"DescribeTagsResult"`
	Meta    xmlResponseMetadata  `xml:"ResponseMetadata"`
}

type xmlDescribeTagsResult struct {
	Tags []xmlTag `xml:"Tags>member"`
}

func handleDescribeTags(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	tags := store.ListTags("")

	xmlTags := make([]xmlTag, 0, len(tags))
	for _, t := range tags {
		xmlTags = append(xmlTags, xmlTag{
			Key:               t.Key,
			Value:             t.Value,
			ResourceId:        t.ResourceID,
			ResourceType:      t.ResourceType,
			PropagateAtLaunch: t.PropagateAtLaunch,
		})
	}

	return xmlOK(&xmlDescribeTagsResponse{
		Xmlns:  asXmlns,
		Result: xmlDescribeTagsResult{Tags: xmlTags},
		Meta:   xmlResponseMetadata{RequestID: newUUIDHandler()},
	})
}

// ---- DeleteTags ----

type xmlDeleteTagsResponse struct {
	XMLName xml.Name            `xml:"DeleteTagsResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleDeleteTags(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	tags := parseASGTags(form)

	store.DeleteTags(tags)

	return xmlOK(&xmlDeleteTagsResponse{
		Xmlns: asXmlns,
		Meta:  xmlResponseMetadata{RequestID: newUUIDHandler()},
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

func parseASGTags(form url.Values) []Tag {
	var tags []Tag
	for i := 1; ; i++ {
		key := form.Get(fmt.Sprintf("Tags.member.%d.Key", i))
		if key == "" {
			break
		}
		t := Tag{
			Key:          key,
			Value:        form.Get(fmt.Sprintf("Tags.member.%d.Value", i)),
			ResourceID:   form.Get(fmt.Sprintf("Tags.member.%d.ResourceId", i)),
			ResourceType: form.Get(fmt.Sprintf("Tags.member.%d.ResourceType", i)),
		}
		if form.Get(fmt.Sprintf("Tags.member.%d.PropagateAtLaunch", i)) == "true" {
			t.PropagateAtLaunch = true
		}
		tags = append(tags, t)
	}
	return tags
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

func newUUIDHandler() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
