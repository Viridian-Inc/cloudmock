package elasticmapreduce

import (
	"crypto/rand"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/neureaux/cloudmock/pkg/service"
)

const emrXmlns = "http://elasticmapreduce.amazonaws.com/doc/2009-03-31/"

type xmlResponseMetadata struct {
	RequestID string `xml:"RequestId"`
}

type xmlTag struct {
	Key   string `xml:"Key"`
	Value string `xml:"Value"`
}

// ---- helpers ----

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

func parseTags(form url.Values) map[string]string {
	tags := make(map[string]string)
	for i := 1; ; i++ {
		key := form.Get(fmt.Sprintf("Tags.member.%d.Key", i))
		if key == "" {
			break
		}
		val := form.Get(fmt.Sprintf("Tags.member.%d.Value", i))
		tags[key] = val
	}
	return tags
}

func parseTagKeys(form url.Values) []string {
	keys := make([]string, 0)
	for i := 1; ; i++ {
		k := form.Get(fmt.Sprintf("TagKeys.member.%d", i))
		if k == "" {
			break
		}
		keys = append(keys, k)
	}
	return keys
}

func parseStringList(form url.Values, prefix string) []string {
	result := make([]string, 0)
	for i := 1; ; i++ {
		v := form.Get(fmt.Sprintf("%s.member.%d", prefix, i))
		if v == "" {
			break
		}
		result = append(result, v)
	}
	return result
}

func xmlOK(body any) (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: body, Format: service.FormatXML}, nil
}

func xmlErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatXML}, awsErr
}

func newRequestID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// ---- RunJobFlow ----

type xmlRunJobFlowResponse struct {
	XMLName xml.Name            `xml:"RunJobFlowResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Result  struct {
		JobFlowId string `xml:"JobFlowId"`
	} `xml:"RunJobFlowResult"`
	Meta xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleRunJobFlow(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	name := form.Get("Name")
	if name == "" {
		return xmlErr(service.ErrValidation("Name is required."))
	}
	releaseLabel := form.Get("ReleaseLabel")
	logUri := form.Get("LogUri")
	serviceRole := form.Get("ServiceRole")
	jobFlowRole := form.Get("JobFlowRole")

	apps := make([]Application, 0)
	for i := 1; ; i++ {
		appName := form.Get(fmt.Sprintf("Applications.member.%d.Name", i))
		if appName == "" {
			break
		}
		apps = append(apps, Application{
			Name:    appName,
			Version: form.Get(fmt.Sprintf("Applications.member.%d.Version", i)),
		})
	}

	tags := parseTags(form)

	c := store.RunJobFlow(name, releaseLabel, logUri, serviceRole, jobFlowRole, apps, tags)

	resp := &xmlRunJobFlowResponse{Xmlns: emrXmlns, Meta: xmlResponseMetadata{RequestID: newRequestID()}}
	resp.Result.JobFlowId = c.ID
	return xmlOK(resp)
}

// ---- DescribeCluster ----

type xmlClusterDetail struct {
	Id                   string   `xml:"Id"`
	Name                 string   `xml:"Name"`
	ClusterArn           string   `xml:"ClusterArn"`
	Status               struct {
		State string `xml:"State"`
	} `xml:"Status"`
	ReleaseLabel         string   `xml:"ReleaseLabel"`
	VisibleToAllUsers    bool     `xml:"VisibleToAllUsers"`
	TerminationProtected bool     `xml:"TerminationProtected"`
	MasterPublicDnsName  string   `xml:"MasterPublicDnsName"`
	Tags                 []xmlTag `xml:"Tags>member"`
}

type xmlDescribeClusterResponse struct {
	XMLName xml.Name            `xml:"DescribeClusterResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Result  struct {
		Cluster xmlClusterDetail `xml:"Cluster"`
	} `xml:"DescribeClusterResult"`
	Meta xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleDescribeCluster(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	id := form.Get("ClusterId")
	if id == "" {
		return xmlErr(service.ErrValidation("ClusterId is required."))
	}
	c, ok := store.GetCluster(id)
	if !ok {
		return xmlErr(service.NewAWSError("InvalidRequestException", "Cluster "+id+" not found.", http.StatusBadRequest))
	}
	tags := make([]xmlTag, 0, len(c.Tags))
	for k, v := range c.Tags {
		tags = append(tags, xmlTag{Key: k, Value: v})
	}
	detail := xmlClusterDetail{
		Id: c.ID, Name: c.Name, ClusterArn: c.ARN,
		ReleaseLabel: c.ReleaseLabel, VisibleToAllUsers: c.VisibleToAllUsers,
		TerminationProtected: c.TerminationProtected, MasterPublicDnsName: c.MasterPublicDnsName,
		Tags: tags,
	}
	detail.Status.State = c.Status.State
	resp := &xmlDescribeClusterResponse{Xmlns: emrXmlns, Meta: xmlResponseMetadata{RequestID: newRequestID()}}
	resp.Result.Cluster = detail
	return xmlOK(resp)
}

// ---- ListClusters ----

type xmlClusterSummary struct {
	Id     string `xml:"Id"`
	Name   string `xml:"Name"`
	Status struct {
		State string `xml:"State"`
	} `xml:"Status"`
}

type xmlListClustersResponse struct {
	XMLName xml.Name            `xml:"ListClustersResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Result  struct {
		Clusters []xmlClusterSummary `xml:"Clusters>member"`
	} `xml:"ListClustersResult"`
	Meta xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleListClusters(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	states := parseStringList(form, "ClusterStates")
	clusters := store.ListClusters(states)
	summaries := make([]xmlClusterSummary, 0, len(clusters))
	for _, c := range clusters {
		s := xmlClusterSummary{Id: c.ID, Name: c.Name}
		s.Status.State = c.Status.State
		summaries = append(summaries, s)
	}
	resp := &xmlListClustersResponse{Xmlns: emrXmlns, Meta: xmlResponseMetadata{RequestID: newRequestID()}}
	resp.Result.Clusters = summaries
	return xmlOK(resp)
}

// ---- TerminateJobFlows ----

type xmlTerminateJobFlowsResponse struct {
	XMLName xml.Name            `xml:"TerminateJobFlowsResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleTerminateJobFlows(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	ids := parseStringList(form, "JobFlowIds")
	store.TerminateJobFlows(ids)
	return xmlOK(&xmlTerminateJobFlowsResponse{Xmlns: emrXmlns, Meta: xmlResponseMetadata{RequestID: newRequestID()}})
}

// ---- AddJobFlowSteps ----

type xmlAddJobFlowStepsResponse struct {
	XMLName xml.Name            `xml:"AddJobFlowStepsResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Result  struct {
		StepIds []string `xml:"StepIds>member"`
	} `xml:"AddJobFlowStepsResult"`
	Meta xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleAddJobFlowSteps(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	clusterID := form.Get("JobFlowId")
	if clusterID == "" {
		return xmlErr(service.ErrValidation("JobFlowId is required."))
	}
	steps := make([]Step, 0)
	for i := 1; ; i++ {
		name := form.Get(fmt.Sprintf("Steps.member.%d.Name", i))
		if name == "" {
			break
		}
		steps = append(steps, Step{
			Name: name,
			Config: HadoopJarStepConfig{
				Jar:       form.Get(fmt.Sprintf("Steps.member.%d.HadoopJarStep.Jar", i)),
				MainClass: form.Get(fmt.Sprintf("Steps.member.%d.HadoopJarStep.MainClass", i)),
			},
		})
	}
	ids, ok := store.AddSteps(clusterID, steps)
	if !ok {
		return xmlErr(service.NewAWSError("InvalidRequestException", "Cluster "+clusterID+" not found.", http.StatusBadRequest))
	}
	resp := &xmlAddJobFlowStepsResponse{Xmlns: emrXmlns, Meta: xmlResponseMetadata{RequestID: newRequestID()}}
	resp.Result.StepIds = ids
	return xmlOK(resp)
}

// ---- ListSteps ----

type xmlStepSummary struct {
	Id     string `xml:"Id"`
	Name   string `xml:"Name"`
	Status struct {
		State string `xml:"State"`
	} `xml:"Status"`
}

type xmlListStepsResponse struct {
	XMLName xml.Name            `xml:"ListStepsResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Result  struct {
		Steps []xmlStepSummary `xml:"Steps>member"`
	} `xml:"ListStepsResult"`
	Meta xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleListSteps(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	clusterID := form.Get("ClusterId")
	steps := store.ListSteps(clusterID)
	xmlSteps := make([]xmlStepSummary, 0, len(steps))
	for _, st := range steps {
		s := xmlStepSummary{Id: st.ID, Name: st.Name}
		s.Status.State = st.Status.State
		xmlSteps = append(xmlSteps, s)
	}
	resp := &xmlListStepsResponse{Xmlns: emrXmlns, Meta: xmlResponseMetadata{RequestID: newRequestID()}}
	resp.Result.Steps = xmlSteps
	return xmlOK(resp)
}

// ---- DescribeStep ----

type xmlStepDetail struct {
	Id     string `xml:"Id"`
	Name   string `xml:"Name"`
	Status struct {
		State string `xml:"State"`
	} `xml:"Status"`
	Config struct {
		Jar       string `xml:"Jar"`
		MainClass string `xml:"MainClass"`
	} `xml:"Config"`
}

type xmlDescribeStepResponse struct {
	XMLName xml.Name            `xml:"DescribeStepResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Result  struct {
		Step xmlStepDetail `xml:"Step"`
	} `xml:"DescribeStepResult"`
	Meta xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleDescribeStep(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	clusterID := form.Get("ClusterId")
	stepID := form.Get("StepId")
	st, ok := store.GetStep(clusterID, stepID)
	if !ok {
		return xmlErr(service.NewAWSError("InvalidRequestException", "Step "+stepID+" not found.", http.StatusBadRequest))
	}
	detail := xmlStepDetail{Id: st.ID, Name: st.Name}
	detail.Status.State = st.Status.State
	detail.Config.Jar = st.Config.Jar
	detail.Config.MainClass = st.Config.MainClass
	resp := &xmlDescribeStepResponse{Xmlns: emrXmlns, Meta: xmlResponseMetadata{RequestID: newRequestID()}}
	resp.Result.Step = detail
	return xmlOK(resp)
}

// ---- AddInstanceGroups ----

type xmlAddInstanceGroupsResponse struct {
	XMLName xml.Name            `xml:"AddInstanceGroupsResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Result  struct {
		InstanceGroupIds []string `xml:"InstanceGroupIds>member"`
	} `xml:"AddInstanceGroupsResult"`
	Meta xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleAddInstanceGroups(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	clusterID := form.Get("JobFlowId")
	groups := make([]InstanceGroup, 0)
	for i := 1; ; i++ {
		role := form.Get(fmt.Sprintf("InstanceGroups.member.%d.InstanceRole", i))
		if role == "" {
			break
		}
		count := 1
		if s := form.Get(fmt.Sprintf("InstanceGroups.member.%d.InstanceCount", i)); s != "" {
			if v, err := strconv.Atoi(s); err == nil {
				count = v
			}
		}
		groups = append(groups, InstanceGroup{
			Name:          form.Get(fmt.Sprintf("InstanceGroups.member.%d.Name", i)),
			InstanceRole:  role,
			InstanceType:  form.Get(fmt.Sprintf("InstanceGroups.member.%d.InstanceType", i)),
			InstanceCount: count,
			Market:        form.Get(fmt.Sprintf("InstanceGroups.member.%d.Market", i)),
		})
	}
	ids, ok := store.AddInstanceGroups(clusterID, groups)
	if !ok {
		return xmlErr(service.NewAWSError("InvalidRequestException", "Cluster "+clusterID+" not found.", http.StatusBadRequest))
	}
	resp := &xmlAddInstanceGroupsResponse{Xmlns: emrXmlns, Meta: xmlResponseMetadata{RequestID: newRequestID()}}
	resp.Result.InstanceGroupIds = ids
	return xmlOK(resp)
}

// ---- ListInstanceGroups ----

type xmlInstanceGroupSummary struct {
	Id            string `xml:"Id"`
	Name          string `xml:"Name"`
	InstanceRole  string `xml:"InstanceGroupType"`
	InstanceType  string `xml:"InstanceType"`
	InstanceCount int    `xml:"RequestedInstanceCount"`
	Market        string `xml:"Market"`
	Status        struct {
		State string `xml:"State"`
	} `xml:"Status"`
}

type xmlListInstanceGroupsResponse struct {
	XMLName xml.Name            `xml:"ListInstanceGroupsResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Result  struct {
		InstanceGroups []xmlInstanceGroupSummary `xml:"InstanceGroups>member"`
	} `xml:"ListInstanceGroupsResult"`
	Meta xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleListInstanceGroups(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	clusterID := form.Get("ClusterId")
	groups := store.ListInstanceGroups(clusterID)
	xmlGroups := make([]xmlInstanceGroupSummary, 0, len(groups))
	for _, g := range groups {
		s := xmlInstanceGroupSummary{
			Id: g.ID, Name: g.Name, InstanceRole: g.InstanceRole,
			InstanceType: g.InstanceType, InstanceCount: g.InstanceCount, Market: g.Market,
		}
		s.Status.State = g.Status.State
		xmlGroups = append(xmlGroups, s)
	}
	resp := &xmlListInstanceGroupsResponse{Xmlns: emrXmlns, Meta: xmlResponseMetadata{RequestID: newRequestID()}}
	resp.Result.InstanceGroups = xmlGroups
	return xmlOK(resp)
}

// ---- ModifyInstanceGroups ----

type xmlModifyInstanceGroupsResponse struct {
	XMLName xml.Name            `xml:"ModifyInstanceGroupsResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleModifyInstanceGroups(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	clusterID := form.Get("ClusterId")
	mods := make(map[string]int)
	for i := 1; ; i++ {
		igID := form.Get(fmt.Sprintf("InstanceGroups.member.%d.InstanceGroupId", i))
		if igID == "" {
			break
		}
		count := 1
		if s := form.Get(fmt.Sprintf("InstanceGroups.member.%d.InstanceCount", i)); s != "" {
			if v, err := strconv.Atoi(s); err == nil {
				count = v
			}
		}
		mods[igID] = count
	}
	store.ModifyInstanceGroups(clusterID, mods)
	return xmlOK(&xmlModifyInstanceGroupsResponse{Xmlns: emrXmlns, Meta: xmlResponseMetadata{RequestID: newRequestID()}})
}

// ---- SetTerminationProtection ----

type xmlSetTerminationProtectionResponse struct {
	XMLName xml.Name            `xml:"SetTerminationProtectionResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleSetTerminationProtection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	ids := parseStringList(form, "JobFlowIds")
	protected := form.Get("TerminationProtected") == "true"
	store.SetTerminationProtection(ids, protected)
	return xmlOK(&xmlSetTerminationProtectionResponse{Xmlns: emrXmlns, Meta: xmlResponseMetadata{RequestID: newRequestID()}})
}

// ---- SetVisibleToAllUsers ----

type xmlSetVisibleToAllUsersResponse struct {
	XMLName xml.Name            `xml:"SetVisibleToAllUsersResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleSetVisibleToAllUsers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	ids := parseStringList(form, "JobFlowIds")
	visible := form.Get("VisibleToAllUsers") == "true"
	store.SetVisibleToAllUsers(ids, visible)
	return xmlOK(&xmlSetVisibleToAllUsersResponse{Xmlns: emrXmlns, Meta: xmlResponseMetadata{RequestID: newRequestID()}})
}

// ---- AddTags ----

type xmlAddTagsResponse struct {
	XMLName xml.Name            `xml:"AddTagsResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleAddTags(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	resourceID := form.Get("ResourceId")
	tags := parseTags(form)
	if !store.AddTags(resourceID, tags) {
		return xmlErr(service.NewAWSError("InvalidRequestException", "Resource "+resourceID+" not found.", http.StatusBadRequest))
	}
	return xmlOK(&xmlAddTagsResponse{Xmlns: emrXmlns, Meta: xmlResponseMetadata{RequestID: newRequestID()}})
}

// ---- RemoveTags ----

type xmlRemoveTagsResponse struct {
	XMLName xml.Name            `xml:"RemoveTagsResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleRemoveTags(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	resourceID := form.Get("ResourceId")
	keys := parseTagKeys(form)
	if !store.RemoveTags(resourceID, keys) {
		return xmlErr(service.NewAWSError("InvalidRequestException", "Resource "+resourceID+" not found.", http.StatusBadRequest))
	}
	return xmlOK(&xmlRemoveTagsResponse{Xmlns: emrXmlns, Meta: xmlResponseMetadata{RequestID: newRequestID()}})
}
