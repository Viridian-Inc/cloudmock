package cloudformation

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ---- shared XML types ----

type xmlResponseMetadata struct {
	RequestID string `xml:"RequestId"`
}

// ---- CreateStack ----

type xmlCreateStackResponse struct {
	XMLName xml.Name               `xml:"CreateStackResponse"`
	Result  xmlCreateStackResult   `xml:"CreateStackResult"`
	Meta    xmlResponseMetadata    `xml:"ResponseMetadata"`
}

type xmlCreateStackResult struct {
	StackId string `xml:"StackId"`
}

func handleCreateStack(ctx *service.RequestContext, store *StackStore) (*service.Response, error) {
	form := parseForm(ctx)

	name := form.Get("StackName")
	if name == "" {
		return xmlErr(service.ErrValidation("StackName is required."))
	}
	templateBody := form.Get("TemplateBody")
	if templateBody == "" {
		return xmlErr(service.ErrValidation("TemplateBody is required."))
	}

	params := parseParameters(form)
	tags := parseTags(form)

	stack, err := store.CreateStack(name, templateBody, params, tags)
	if err != nil {
		return xmlErr(service.NewAWSError("AlreadyExistsException", err.Error(), http.StatusBadRequest))
	}

	resp := &xmlCreateStackResponse{
		Result: xmlCreateStackResult{StackId: stack.StackId},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	}
	return xmlOK(resp)
}

// ---- DeleteStack ----

type xmlDeleteStackResponse struct {
	XMLName xml.Name            `xml:"DeleteStackResponse"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleDeleteStack(ctx *service.RequestContext, store *StackStore) (*service.Response, error) {
	form := parseForm(ctx)
	name := form.Get("StackName")
	if name == "" {
		return xmlErr(service.ErrValidation("StackName is required."))
	}

	// DeleteStack is idempotent in real AWS — don't fail on missing stack.
	store.DeleteStack(name)

	return xmlOK(&xmlDeleteStackResponse{Meta: xmlResponseMetadata{RequestID: newUUID()}})
}

// ---- DescribeStacks ----

type xmlDescribeStacksResponse struct {
	XMLName xml.Name                  `xml:"DescribeStacksResponse"`
	Result  xmlDescribeStacksResult   `xml:"DescribeStacksResult"`
	Meta    xmlResponseMetadata       `xml:"ResponseMetadata"`
}

type xmlDescribeStacksResult struct {
	Stacks []xmlStack `xml:"Stacks>member"`
}

type xmlStack struct {
	StackId      string           `xml:"StackId"`
	StackName    string           `xml:"StackName"`
	StackStatus  string           `xml:"StackStatus"`
	Description  string           `xml:"Description,omitempty"`
	CreationTime string           `xml:"CreationTime"`
	Parameters   []xmlParameter   `xml:"Parameters>member,omitempty"`
	Tags         []xmlTag         `xml:"Tags>member,omitempty"`
	Outputs      []xmlOutput      `xml:"Outputs>member,omitempty"`
}

type xmlParameter struct {
	ParameterKey   string `xml:"ParameterKey"`
	ParameterValue string `xml:"ParameterValue"`
}

type xmlTag struct {
	Key   string `xml:"Key"`
	Value string `xml:"Value"`
}

type xmlOutput struct {
	OutputKey   string `xml:"OutputKey"`
	OutputValue string `xml:"OutputValue"`
	Description string `xml:"Description,omitempty"`
	ExportName  string `xml:"ExportName,omitempty"`
}

func handleDescribeStacks(ctx *service.RequestContext, store *StackStore) (*service.Response, error) {
	form := parseForm(ctx)
	name := form.Get("StackName")

	var stacks []*Stack
	if name != "" {
		st, ok := store.GetStack(name)
		if !ok {
			return xmlErr(service.NewAWSError("ValidationError",
				fmt.Sprintf("Stack with id %s does not exist", name),
				http.StatusBadRequest))
		}
		stacks = []*Stack{st}
	} else {
		stacks = store.AllStacks()
	}

	xmlStacks := make([]xmlStack, 0, len(stacks))
	for _, st := range stacks {
		if st.StackStatus == "DELETE_COMPLETE" && name == "" {
			continue // omit deleted stacks from the all-stacks listing
		}
		xmlStacks = append(xmlStacks, stackToXML(st))
	}

	resp := &xmlDescribeStacksResponse{
		Result: xmlDescribeStacksResult{Stacks: xmlStacks},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	}
	return xmlOK(resp)
}

func stackToXML(st *Stack) xmlStack {
	params := make([]xmlParameter, 0, len(st.Parameters))
	for _, p := range st.Parameters {
		params = append(params, xmlParameter{
			ParameterKey:   p.ParameterKey,
			ParameterValue: p.ParameterValue,
		})
	}

	tags := make([]xmlTag, 0, len(st.Tags))
	for _, t := range st.Tags {
		tags = append(tags, xmlTag{Key: t.Key, Value: t.Value})
	}

	outputs := make([]xmlOutput, 0, len(st.Outputs))
	for _, o := range st.Outputs {
		outputs = append(outputs, xmlOutput{
			OutputKey:   o.OutputKey,
			OutputValue: o.OutputValue,
			Description: o.Description,
			ExportName:  o.ExportName,
		})
	}

	return xmlStack{
		StackId:      st.StackId,
		StackName:    st.StackName,
		StackStatus:  st.StackStatus,
		Description:  st.Description,
		CreationTime: st.CreationTime.Format("2006-01-02T15:04:05Z"),
		Parameters:   params,
		Tags:         tags,
		Outputs:      outputs,
	}
}

// ---- ListStacks ----

type xmlListStacksResponse struct {
	XMLName xml.Name               `xml:"ListStacksResponse"`
	Result  xmlListStacksResult    `xml:"ListStacksResult"`
	Meta    xmlResponseMetadata    `xml:"ResponseMetadata"`
}

type xmlListStacksResult struct {
	StackSummaries []xmlStackSummary `xml:"StackSummaries>member"`
}

type xmlStackSummary struct {
	StackId      string `xml:"StackId"`
	StackName    string `xml:"StackName"`
	StackStatus  string `xml:"StackStatus"`
	CreationTime string `xml:"CreationTime"`
	Description  string `xml:"Description,omitempty"`
}

func handleListStacks(ctx *service.RequestContext, store *StackStore) (*service.Response, error) {
	form := parseForm(ctx)
	filters := parseStackStatusFilters(form)

	stacks := store.ListStacks(filters)

	summaries := make([]xmlStackSummary, 0, len(stacks))
	for _, st := range stacks {
		summaries = append(summaries, xmlStackSummary{
			StackId:      st.StackId,
			StackName:    st.StackName,
			StackStatus:  st.StackStatus,
			CreationTime: st.CreationTime.Format("2006-01-02T15:04:05Z"),
			Description:  st.Description,
		})
	}

	resp := &xmlListStacksResponse{
		Result: xmlListStacksResult{StackSummaries: summaries},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	}
	return xmlOK(resp)
}

// ---- DescribeStackResources ----

type xmlDescribeStackResourcesResponse struct {
	XMLName xml.Name                          `xml:"DescribeStackResourcesResponse"`
	Result  xmlDescribeStackResourcesResult   `xml:"DescribeStackResourcesResult"`
	Meta    xmlResponseMetadata               `xml:"ResponseMetadata"`
}

type xmlDescribeStackResourcesResult struct {
	StackResources []xmlStackResource `xml:"StackResources>member"`
}

type xmlStackResource struct {
	StackId            string `xml:"StackId"`
	StackName          string `xml:"StackName"`
	LogicalResourceId  string `xml:"LogicalResourceId"`
	PhysicalResourceId string `xml:"PhysicalResourceId,omitempty"`
	ResourceType       string `xml:"ResourceType"`
	ResourceStatus     string `xml:"ResourceStatus"`
	Timestamp          string `xml:"Timestamp"`
}

func handleDescribeStackResources(ctx *service.RequestContext, store *StackStore) (*service.Response, error) {
	form := parseForm(ctx)
	name := form.Get("StackName")
	if name == "" {
		return xmlErr(service.ErrValidation("StackName is required."))
	}

	st, ok := store.GetStack(name)
	if !ok {
		return xmlErr(service.NewAWSError("ValidationError",
			fmt.Sprintf("Stack with id %s does not exist", name),
			http.StatusBadRequest))
	}

	resources := make([]xmlStackResource, 0, len(st.Resources))
	for _, r := range st.Resources {
		resources = append(resources, xmlStackResource{
			StackId:            st.StackId,
			StackName:          st.StackName,
			LogicalResourceId:  r.LogicalResourceId,
			PhysicalResourceId: r.PhysicalResourceId,
			ResourceType:       r.ResourceType,
			ResourceStatus:     r.ResourceStatus,
			Timestamp:          r.Timestamp.Format("2006-01-02T15:04:05Z"),
		})
	}

	resp := &xmlDescribeStackResourcesResponse{
		Result: xmlDescribeStackResourcesResult{StackResources: resources},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	}
	return xmlOK(resp)
}

// ---- DescribeStackEvents ----

type xmlDescribeStackEventsResponse struct {
	XMLName xml.Name                        `xml:"DescribeStackEventsResponse"`
	Result  xmlDescribeStackEventsResult    `xml:"DescribeStackEventsResult"`
	Meta    xmlResponseMetadata             `xml:"ResponseMetadata"`
}

type xmlDescribeStackEventsResult struct {
	StackEvents []xmlStackEvent `xml:"StackEvents>member"`
}

type xmlStackEvent struct {
	EventId           string `xml:"EventId"`
	StackId           string `xml:"StackId"`
	StackName         string `xml:"StackName"`
	LogicalResourceId string `xml:"LogicalResourceId"`
	ResourceType      string `xml:"ResourceType"`
	ResourceStatus    string `xml:"ResourceStatus"`
	Timestamp         string `xml:"Timestamp"`
}

func handleDescribeStackEvents(ctx *service.RequestContext, store *StackStore) (*service.Response, error) {
	form := parseForm(ctx)
	name := form.Get("StackName")
	if name == "" {
		return xmlErr(service.ErrValidation("StackName is required."))
	}

	st, ok := store.GetStack(name)
	if !ok {
		return xmlErr(service.NewAWSError("ValidationError",
			fmt.Sprintf("Stack with id %s does not exist", name),
			http.StatusBadRequest))
	}

	events := make([]xmlStackEvent, 0, len(st.Events))
	for _, e := range st.Events {
		events = append(events, xmlStackEvent{
			EventId:           e.EventId,
			StackId:           e.StackId,
			StackName:         e.StackName,
			LogicalResourceId: e.LogicalResourceId,
			ResourceType:      e.ResourceType,
			ResourceStatus:    e.ResourceStatus,
			Timestamp:         e.Timestamp.Format("2006-01-02T15:04:05Z"),
		})
	}

	resp := &xmlDescribeStackEventsResponse{
		Result: xmlDescribeStackEventsResult{StackEvents: events},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	}
	return xmlOK(resp)
}

// ---- GetTemplate ----

type xmlGetTemplateResponse struct {
	XMLName xml.Name             `xml:"GetTemplateResponse"`
	Result  xmlGetTemplateResult `xml:"GetTemplateResult"`
	Meta    xmlResponseMetadata  `xml:"ResponseMetadata"`
}

type xmlGetTemplateResult struct {
	TemplateBody string `xml:"TemplateBody"`
}

func handleGetTemplate(ctx *service.RequestContext, store *StackStore) (*service.Response, error) {
	form := parseForm(ctx)
	name := form.Get("StackName")
	if name == "" {
		return xmlErr(service.ErrValidation("StackName is required."))
	}

	st, ok := store.GetStack(name)
	if !ok {
		return xmlErr(service.NewAWSError("ValidationError",
			fmt.Sprintf("Stack with id %s does not exist", name),
			http.StatusBadRequest))
	}

	resp := &xmlGetTemplateResponse{
		Result: xmlGetTemplateResult{TemplateBody: st.TemplateBody},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	}
	return xmlOK(resp)
}

// ---- ValidateTemplate ----

type xmlValidateTemplateResponse struct {
	XMLName    xml.Name                  `xml:"ValidateTemplateResponse"`
	Result     xmlValidateTemplateResult `xml:"ValidateTemplateResult"`
	Meta       xmlResponseMetadata       `xml:"ResponseMetadata"`
}

type xmlValidateTemplateResult struct {
	Description    string                    `xml:"Description,omitempty"`
	Parameters     []xmlTemplateParameter    `xml:"Parameters>member,omitempty"`
	Capabilities   []string                  `xml:"Capabilities>member,omitempty"`
}

type xmlTemplateParameter struct {
	ParameterKey string `xml:"ParameterKey"`
	ParameterType string `xml:"ParameterType,omitempty"`
	DefaultValue  string `xml:"DefaultValue,omitempty"`
	Description   string `xml:"Description,omitempty"`
}

func handleValidateTemplate(ctx *service.RequestContext, _ *StackStore) (*service.Response, error) {
	form := parseForm(ctx)
	templateBody := form.Get("TemplateBody")
	if templateBody == "" {
		return xmlErr(service.ErrValidation("TemplateBody is required."))
	}

	description, _, _ := parseTemplate(templateBody, nil)

	// Re-parse to extract parameter definitions.
	tmplParams := extractTemplateParameters(templateBody)

	resp := &xmlValidateTemplateResponse{
		Result: xmlValidateTemplateResult{
			Description: description,
			Parameters:  tmplParams,
		},
		Meta: xmlResponseMetadata{RequestID: newUUID()},
	}
	return xmlOK(resp)
}

// ---- ListExports ----

type xmlListExportsResponse struct {
	XMLName xml.Name              `xml:"ListExportsResponse"`
	Result  xmlListExportsResult  `xml:"ListExportsResult"`
	Meta    xmlResponseMetadata   `xml:"ResponseMetadata"`
}

type xmlListExportsResult struct {
	Exports []xmlExport `xml:"Exports>member,omitempty"`
}

type xmlExport struct {
	ExportingStackId string `xml:"ExportingStackId"`
	Name             string `xml:"Name"`
	Value            string `xml:"Value"`
}

func handleListExports(ctx *service.RequestContext, store *StackStore) (*service.Response, error) {
	entries := store.ListExports()

	exports := make([]xmlExport, 0, len(entries))
	for _, e := range entries {
		exports = append(exports, xmlExport{
			ExportingStackId: e.ExportingStackId,
			Name:             e.Name,
			Value:            e.Value,
		})
	}

	resp := &xmlListExportsResponse{
		Result: xmlListExportsResult{Exports: exports},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	}
	return xmlOK(resp)
}

// ---- CreateChangeSet ----

type xmlCreateChangeSetResponse struct {
	XMLName xml.Name                   `xml:"CreateChangeSetResponse"`
	Result  xmlCreateChangeSetResult   `xml:"CreateChangeSetResult"`
	Meta    xmlResponseMetadata        `xml:"ResponseMetadata"`
}

type xmlCreateChangeSetResult struct {
	Id string `xml:"Id"`
}

func handleCreateChangeSet(ctx *service.RequestContext, store *StackStore) (*service.Response, error) {
	form := parseForm(ctx)
	stackName := form.Get("StackName")
	if stackName == "" {
		return xmlErr(service.ErrValidation("StackName is required."))
	}
	changeSetName := form.Get("ChangeSetName")
	if changeSetName == "" {
		return xmlErr(service.ErrValidation("ChangeSetName is required."))
	}
	description := form.Get("Description")

	cs, err := store.CreateChangeSet(stackName, changeSetName, description)
	if err != nil {
		return xmlErr(service.NewAWSError("ValidationError", err.Error(), http.StatusBadRequest))
	}

	resp := &xmlCreateChangeSetResponse{
		Result: xmlCreateChangeSetResult{Id: cs.ChangeSetId},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	}
	return xmlOK(resp)
}

// ---- DescribeChangeSet ----

type xmlDescribeChangeSetResponse struct {
	XMLName         xml.Name            `xml:"DescribeChangeSetResponse"`
	Result          xmlChangeSet        `xml:"DescribeChangeSetResult"`
	Meta            xmlResponseMetadata `xml:"ResponseMetadata"`
}

type xmlChangeSet struct {
	ChangeSetId     string `xml:"ChangeSetId"`
	ChangeSetName   string `xml:"ChangeSetName"`
	StackId         string `xml:"StackId"`
	StackName       string `xml:"StackName"`
	Status          string `xml:"Status"`
	ExecutionStatus string `xml:"ExecutionStatus"`
	Description     string `xml:"Description,omitempty"`
	CreationTime    string `xml:"CreationTime"`
}

func handleDescribeChangeSet(ctx *service.RequestContext, store *StackStore) (*service.Response, error) {
	form := parseForm(ctx)
	stackName := form.Get("StackName")
	if stackName == "" {
		return xmlErr(service.ErrValidation("StackName is required."))
	}
	changeSetName := form.Get("ChangeSetName")
	if changeSetName == "" {
		return xmlErr(service.ErrValidation("ChangeSetName is required."))
	}

	cs, ok := store.GetChangeSet(stackName, changeSetName)
	if !ok {
		return xmlErr(service.NewAWSError("ChangeSetNotFoundException",
			fmt.Sprintf("ChangeSet [%s] does not exist", changeSetName),
			http.StatusBadRequest))
	}

	resp := &xmlDescribeChangeSetResponse{
		Result: xmlChangeSet{
			ChangeSetId:     cs.ChangeSetId,
			ChangeSetName:   cs.ChangeSetName,
			StackId:         cs.StackId,
			StackName:       cs.StackName,
			Status:          cs.Status,
			ExecutionStatus: cs.ExecutionStatus,
			Description:     cs.Description,
			CreationTime:    cs.CreationTime.Format("2006-01-02T15:04:05Z"),
		},
		Meta: xmlResponseMetadata{RequestID: newUUID()},
	}
	return xmlOK(resp)
}

// ---- ExecuteChangeSet ----

type xmlExecuteChangeSetResponse struct {
	XMLName xml.Name            `xml:"ExecuteChangeSetResponse"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleExecuteChangeSet(ctx *service.RequestContext, store *StackStore) (*service.Response, error) {
	form := parseForm(ctx)
	stackName := form.Get("StackName")
	changeSetName := form.Get("ChangeSetName")
	if stackName == "" || changeSetName == "" {
		return xmlErr(service.ErrValidation("StackName and ChangeSetName are required."))
	}

	if !store.ExecuteChangeSet(stackName, changeSetName) {
		return xmlErr(service.NewAWSError("ChangeSetNotFoundException",
			fmt.Sprintf("ChangeSet [%s] does not exist", changeSetName),
			http.StatusBadRequest))
	}

	return xmlOK(&xmlExecuteChangeSetResponse{Meta: xmlResponseMetadata{RequestID: newUUID()}})
}

// ---- DeleteChangeSet ----

type xmlDeleteChangeSetResponse struct {
	XMLName xml.Name            `xml:"DeleteChangeSetResponse"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleDeleteChangeSet(ctx *service.RequestContext, store *StackStore) (*service.Response, error) {
	form := parseForm(ctx)
	stackName := form.Get("StackName")
	changeSetName := form.Get("ChangeSetName")
	if stackName == "" || changeSetName == "" {
		return xmlErr(service.ErrValidation("StackName and ChangeSetName are required."))
	}

	// Idempotent — don't error if not found.
	store.DeleteChangeSet(stackName, changeSetName)

	return xmlOK(&xmlDeleteChangeSetResponse{Meta: xmlResponseMetadata{RequestID: newUUID()}})
}

// ---- helper functions ----

// parseForm merges query-string params and form-encoded body into url.Values.
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

// parseParameters extracts Parameter.N.ParameterKey / ParameterValue pairs.
func parseParameters(form url.Values) []Parameter {
	var params []Parameter
	for i := 1; ; i++ {
		key := form.Get(fmt.Sprintf("Parameters.member.%d.ParameterKey", i))
		if key == "" {
			break
		}
		val := form.Get(fmt.Sprintf("Parameters.member.%d.ParameterValue", i))
		params = append(params, Parameter{ParameterKey: key, ParameterValue: val})
	}
	return params
}

// parseTags extracts Tags.member.N.Key / Value pairs.
func parseTags(form url.Values) []Tag {
	var tags []Tag
	for i := 1; ; i++ {
		key := form.Get(fmt.Sprintf("Tags.member.%d.Key", i))
		if key == "" {
			break
		}
		val := form.Get(fmt.Sprintf("Tags.member.%d.Value", i))
		tags = append(tags, Tag{Key: key, Value: val})
	}
	return tags
}

// parseStackStatusFilters extracts StackStatusFilter.member.N values.
func parseStackStatusFilters(form url.Values) []string {
	var filters []string
	for i := 1; ; i++ {
		f := form.Get(fmt.Sprintf("StackStatusFilter.member.%d", i))
		if f == "" {
			break
		}
		filters = append(filters, f)
	}
	return filters
}

// extractTemplateParameters reads parameter definitions from a raw template body.
func extractTemplateParameters(templateBody string) []xmlTemplateParameter {
	var tmpl cfnTemplate
	if err := json.Unmarshal([]byte(templateBody), &tmpl); err != nil {
		return nil
	}
	params := make([]xmlTemplateParameter, 0, len(tmpl.Parameters))
	for key, defn := range tmpl.Parameters {
		var defVal string
		if defn.Default != nil {
			var s string
			if err := json.Unmarshal(defn.Default, &s); err == nil {
				defVal = s
			} else {
				defVal = string(defn.Default)
			}
		}
		params = append(params, xmlTemplateParameter{
			ParameterKey:  key,
			ParameterType: defn.Type,
			DefaultValue:  defVal,
		})
	}
	return params
}

// xmlOK wraps a response body in a 200 XML response.
func xmlOK(body any) (*service.Response, error) {
	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       body,
		Format:     service.FormatXML,
	}, nil
}

// xmlErr returns an XML-formatted AWS error response.
func xmlErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatXML}, awsErr
}
