package ec2

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/neureaux/cloudmock/pkg/service"
)

// ============================================================
// Instance XML types
// ============================================================

type xmlInstanceState struct {
	Code int    `xml:"code"`
	Name string `xml:"name"`
}

type xmlInstance struct {
	InstanceId       string           `xml:"instanceId"`
	ImageId          string           `xml:"imageId"`
	InstanceState    xmlInstanceState `xml:"instanceState"`
	InstanceType     string           `xml:"instanceType"`
	SubnetId         string           `xml:"subnetId"`
	VpcId            string           `xml:"vpcId"`
	PrivateIpAddress string           `xml:"privateIpAddress"`
	KeyName          string           `xml:"keyName,omitempty"`
	LaunchTime       string           `xml:"launchTime"`
	TagSet           []xmlTag         `xml:"tagSet>item,omitempty"`
	GroupSet         []xmlGroupRef    `xml:"groupSet>item,omitempty"`
}

type xmlGroupRef struct {
	GroupId string `xml:"groupId"`
}

type xmlTag struct {
	Key   string `xml:"key"`
	Value string `xml:"value"`
}

type xmlReservation struct {
	ReservationId string        `xml:"reservationId"`
	OwnerId       string        `xml:"ownerId"`
	InstancesSet  []xmlInstance `xml:"instancesSet>item"`
}

func toXMLInstance(inst *Instance) xmlInstance {
	xi := xmlInstance{
		InstanceId: inst.InstanceId,
		ImageId:    inst.ImageId,
		InstanceState: xmlInstanceState{
			Code: instanceStateCode(inst.State),
			Name: inst.State,
		},
		InstanceType:     inst.InstanceType,
		SubnetId:         inst.SubnetId,
		VpcId:            inst.VpcId,
		PrivateIpAddress: inst.PrivateIpAddress,
		KeyName:          inst.KeyName,
		LaunchTime:       inst.LaunchTime.Format("2006-01-02T15:04:05.000Z"),
	}
	for _, sgId := range inst.SecurityGroupIds {
		xi.GroupSet = append(xi.GroupSet, xmlGroupRef{GroupId: sgId})
	}
	for k, v := range inst.Tags {
		xi.TagSet = append(xi.TagSet, xmlTag{Key: k, Value: v})
	}
	return xi
}

// ---- RunInstances ----

type xmlRunInstancesResponse struct {
	XMLName       xml.Name       `xml:"RunInstancesResponse"`
	Xmlns         string         `xml:"xmlns,attr"`
	RequestID     string         `xml:"requestId"`
	ReservationId string         `xml:"reservationId"`
	OwnerId       string         `xml:"ownerId"`
	InstancesSet  []xmlInstance  `xml:"instancesSet>item"`
}

func handleRunInstances(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)

	imageId := form.Get("ImageId")
	if imageId == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter ImageId.",
			http.StatusBadRequest))
	}

	instanceType := form.Get("InstanceType")
	if instanceType == "" {
		instanceType = "t2.micro"
	}

	subnetId := form.Get("SubnetId")
	if subnetId == "" {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter SubnetId.",
			http.StatusBadRequest))
	}

	keyName := form.Get("KeyName")

	// Parse security group IDs: SecurityGroupId.N
	sgIds := parseIndexedParam(form, "SecurityGroupId")

	// Parse MinCount / MaxCount (default 1).
	minCount := 1
	if v := form.Get("MinCount"); v != "" {
		minCount, _ = strconv.Atoi(v)
	}
	if minCount < 1 {
		minCount = 1
	}

	instances, reservationId, errCode := store.RunInstances(imageId, instanceType, subnetId, keyName, sgIds, minCount)
	if errCode != "" {
		switch errCode {
		case "subnet_not_found":
			return xmlErr(service.NewAWSError("InvalidSubnetID.NotFound",
				fmt.Sprintf("The subnet ID '%s' does not exist.", subnetId),
				http.StatusBadRequest))
		default:
			return xmlErr(service.NewAWSError("InvalidParameterValue",
				errCode, http.StatusBadRequest))
		}
	}

	xmlInstances := make([]xmlInstance, 0, len(instances))
	for _, inst := range instances {
		xmlInstances = append(xmlInstances, toXMLInstance(inst))
	}

	return xmlOK(&xmlRunInstancesResponse{
		Xmlns:         ec2Xmlns,
		RequestID:     newUUID(),
		ReservationId: reservationId,
		OwnerId:       "000000000000",
		InstancesSet:  xmlInstances,
	})
}

// ---- DescribeInstances ----

type xmlDescribeInstancesResponse struct {
	XMLName         xml.Name         `xml:"DescribeInstancesResponse"`
	Xmlns           string           `xml:"xmlns,attr"`
	RequestID       string           `xml:"requestId"`
	ReservationSet  []xmlReservation `xml:"reservationSet>item"`
}

func handleDescribeInstances(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)

	ids := parseIndexedParam(form, "InstanceId")
	filters := parseAllFilters(form)

	instances := store.ListInstances(ids, filters)

	// Group instances into reservations — for simplicity one reservation per instance.
	reservations := make([]xmlReservation, 0, len(instances))
	for _, inst := range instances {
		reservations = append(reservations, xmlReservation{
			ReservationId: genID("r-"),
			OwnerId:       "000000000000",
			InstancesSet:  []xmlInstance{toXMLInstance(inst)},
		})
	}

	return xmlOK(&xmlDescribeInstancesResponse{
		Xmlns:          ec2Xmlns,
		RequestID:      newUUID(),
		ReservationSet: reservations,
	})
}

// ---- TerminateInstances ----

type xmlInstanceStateChange struct {
	InstanceId    string           `xml:"instanceId"`
	CurrentState  xmlInstanceState `xml:"currentState"`
	PreviousState xmlInstanceState `xml:"previousState"`
}

type xmlTerminateInstancesResponse struct {
	XMLName     xml.Name                 `xml:"TerminateInstancesResponse"`
	Xmlns       string                   `xml:"xmlns,attr"`
	RequestID   string                   `xml:"requestId"`
	InstancesSet []xmlInstanceStateChange `xml:"instancesSet>item"`
}

func handleTerminateInstances(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	ids := parseIndexedParam(form, "InstanceId")
	if len(ids) == 0 {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter InstanceId.",
			http.StatusBadRequest))
	}

	changes := store.TerminateInstances(ids)

	items := make([]xmlInstanceStateChange, 0, len(changes))
	for id, states := range changes {
		items = append(items, xmlInstanceStateChange{
			InstanceId:    id,
			PreviousState: xmlInstanceState{Code: instanceStateCode(states[0]), Name: states[0]},
			CurrentState:  xmlInstanceState{Code: instanceStateCode(states[1]), Name: states[1]},
		})
	}

	return xmlOK(&xmlTerminateInstancesResponse{
		Xmlns:        ec2Xmlns,
		RequestID:    newUUID(),
		InstancesSet: items,
	})
}

// ---- StopInstances ----

type xmlStopInstancesResponse struct {
	XMLName      xml.Name                 `xml:"StopInstancesResponse"`
	Xmlns        string                   `xml:"xmlns,attr"`
	RequestID    string                   `xml:"requestId"`
	InstancesSet []xmlInstanceStateChange `xml:"instancesSet>item"`
}

func handleStopInstances(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	ids := parseIndexedParam(form, "InstanceId")
	if len(ids) == 0 {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter InstanceId.",
			http.StatusBadRequest))
	}

	changes := store.StopInstances(ids)

	items := make([]xmlInstanceStateChange, 0, len(changes))
	for id, states := range changes {
		items = append(items, xmlInstanceStateChange{
			InstanceId:    id,
			PreviousState: xmlInstanceState{Code: instanceStateCode(states[0]), Name: states[0]},
			CurrentState:  xmlInstanceState{Code: instanceStateCode(states[1]), Name: states[1]},
		})
	}

	return xmlOK(&xmlStopInstancesResponse{
		Xmlns:        ec2Xmlns,
		RequestID:    newUUID(),
		InstancesSet: items,
	})
}

// ---- StartInstances ----

type xmlStartInstancesResponse struct {
	XMLName      xml.Name                 `xml:"StartInstancesResponse"`
	Xmlns        string                   `xml:"xmlns,attr"`
	RequestID    string                   `xml:"requestId"`
	InstancesSet []xmlInstanceStateChange `xml:"instancesSet>item"`
}

func handleStartInstances(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	ids := parseIndexedParam(form, "InstanceId")
	if len(ids) == 0 {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter InstanceId.",
			http.StatusBadRequest))
	}

	changes := store.StartInstances(ids)

	items := make([]xmlInstanceStateChange, 0, len(changes))
	for id, states := range changes {
		items = append(items, xmlInstanceStateChange{
			InstanceId:    id,
			PreviousState: xmlInstanceState{Code: instanceStateCode(states[0]), Name: states[0]},
			CurrentState:  xmlInstanceState{Code: instanceStateCode(states[1]), Name: states[1]},
		})
	}

	return xmlOK(&xmlStartInstancesResponse{
		Xmlns:        ec2Xmlns,
		RequestID:    newUUID(),
		InstancesSet: items,
	})
}

// ---- DescribeInstanceStatus ----

type xmlInstanceStatusItem struct {
	InstanceId   string          `xml:"instanceId"`
	InstanceState xmlInstanceState `xml:"instanceState"`
	SystemStatus  xmlStatusSummary `xml:"systemStatus"`
	InstanceStatus xmlStatusSummary `xml:"instanceStatus"`
}

type xmlStatusSummary struct {
	Status  string `xml:"status"`
}

type xmlDescribeInstanceStatusResponse struct {
	XMLName           xml.Name                `xml:"DescribeInstanceStatusResponse"`
	Xmlns             string                  `xml:"xmlns,attr"`
	RequestID         string                  `xml:"requestId"`
	InstanceStatusSet []xmlInstanceStatusItem `xml:"instanceStatusSet>item"`
}

func handleDescribeInstanceStatus(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	ids := parseIndexedParam(form, "InstanceId")

	instances := store.ListInstances(ids, nil)

	items := make([]xmlInstanceStatusItem, 0, len(instances))
	for _, inst := range instances {
		status := "ok"
		if inst.State != "running" {
			status = "not-applicable"
		}
		items = append(items, xmlInstanceStatusItem{
			InstanceId: inst.InstanceId,
			InstanceState: xmlInstanceState{
				Code: instanceStateCode(inst.State),
				Name: inst.State,
			},
			SystemStatus:   xmlStatusSummary{Status: status},
			InstanceStatus: xmlStatusSummary{Status: status},
		})
	}

	return xmlOK(&xmlDescribeInstanceStatusResponse{
		Xmlns:             ec2Xmlns,
		RequestID:         newUUID(),
		InstanceStatusSet: items,
	})
}

// ============================================================
// Tagging XML types and handlers
// ============================================================

type xmlCreateTagsResponse struct {
	XMLName   xml.Name `xml:"CreateTagsResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
	Return    bool     `xml:"return"`
}

func handleCreateTags(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)

	resourceIds := parseIndexedParam(form, "ResourceId")
	if len(resourceIds) == 0 {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter ResourceId.",
			http.StatusBadRequest))
	}

	tags := parseIndexedTags(form, "Tag")
	if len(tags) == 0 {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain at least one Tag.",
			http.StatusBadRequest))
	}

	store.CreateTags(resourceIds, tags)

	return xmlOK(&xmlCreateTagsResponse{
		Xmlns:     ec2Xmlns,
		RequestID: newUUID(),
		Return:    true,
	})
}

type xmlDeleteTagsResponse struct {
	XMLName   xml.Name `xml:"DeleteTagsResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
	Return    bool     `xml:"return"`
}

func handleDeleteTags(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)

	resourceIds := parseIndexedParam(form, "ResourceId")
	if len(resourceIds) == 0 {
		return xmlErr(service.NewAWSError("MissingParameter",
			"The request must contain the parameter ResourceId.",
			http.StatusBadRequest))
	}

	keys := parseIndexedTagKeys(form, "Tag")

	store.DeleteTags(resourceIds, keys)

	return xmlOK(&xmlDeleteTagsResponse{
		Xmlns:     ec2Xmlns,
		RequestID: newUUID(),
		Return:    true,
	})
}

type xmlTagItem struct {
	ResourceId   string `xml:"resourceId"`
	ResourceType string `xml:"resourceType"`
	Key          string `xml:"key"`
	Value        string `xml:"value"`
}

type xmlDescribeTagsResponse struct {
	XMLName   xml.Name     `xml:"DescribeTagsResponse"`
	Xmlns     string       `xml:"xmlns,attr"`
	RequestID string       `xml:"requestId"`
	TagSet    []xmlTagItem `xml:"tagSet>item"`
}

func handleDescribeTags(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	filters := parseAllFilters(form)

	entries := store.ListTags(filters)

	items := make([]xmlTagItem, 0, len(entries))
	for _, e := range entries {
		items = append(items, xmlTagItem{
			ResourceId:   e.ResourceId,
			ResourceType: e.ResourceType,
			Key:          e.Key,
			Value:        e.Value,
		})
	}

	return xmlOK(&xmlDescribeTagsResponse{
		Xmlns:     ec2Xmlns,
		RequestID: newUUID(),
		TagSet:    items,
	})
}

// ============================================================
// Helper functions
// ============================================================

// parseAllFilters parses all Filter.N.Name / Filter.N.Value.M parameters
// and returns a map of filter name -> list of values.
func parseAllFilters(form interface{ Get(string) string }) map[string][]string {
	type getter interface {
		Get(string) string
	}
	g := form.(getter)

	filters := make(map[string][]string)
	for i := 1; ; i++ {
		name := g.Get(fmt.Sprintf("Filter.%d.Name", i))
		if name == "" {
			break
		}
		var vals []string
		for j := 1; ; j++ {
			v := g.Get(fmt.Sprintf("Filter.%d.Value.%d", i, j))
			if v == "" {
				break
			}
			vals = append(vals, v)
		}
		if len(vals) > 0 {
			// Preserve case for tag:* filters; lowercase everything else.
			if strings.HasPrefix(name, "tag:") || strings.HasPrefix(name, "Tag:") {
				filters[name] = vals
			} else {
				filters[strings.ToLower(name)] = vals
			}
		}
	}
	return filters
}

// parseIndexedTags parses Tag.N.Key / Tag.N.Value pairs.
func parseIndexedTags(form interface{ Get(string) string }, prefix string) map[string]string {
	type getter interface {
		Get(string) string
	}
	g := form.(getter)

	tags := make(map[string]string)
	for i := 1; ; i++ {
		k := g.Get(fmt.Sprintf("%s.%d.Key", prefix, i))
		if k == "" {
			break
		}
		v := g.Get(fmt.Sprintf("%s.%d.Value", prefix, i))
		tags[k] = v
	}
	return tags
}

// parseIndexedTagKeys parses Tag.N.Key entries (for DeleteTags).
func parseIndexedTagKeys(form interface{ Get(string) string }, prefix string) []string {
	type getter interface {
		Get(string) string
	}
	g := form.(getter)

	var keys []string
	for i := 1; ; i++ {
		k := g.Get(fmt.Sprintf("%s.%d.Key", prefix, i))
		if k == "" {
			break
		}
		keys = append(keys, k)
	}
	return keys
}
