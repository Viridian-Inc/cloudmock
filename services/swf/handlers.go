package swf

import (
	"encoding/json"
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

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
		return service.NewAWSError("InvalidParameterException", "Invalid JSON.", http.StatusBadRequest)
	}
	return nil
}

func emptyOK() (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: struct{}{}, Format: service.FormatJSON}, nil
}

// ---- RegisterDomain ----

type registerDomainRequest struct {
	Name                                   string            `json:"name"`
	Description                            string            `json:"description"`
	WorkflowExecutionRetentionPeriodInDays string            `json:"workflowExecutionRetentionPeriodInDays"`
	Tags                                   []tagJSON         `json:"tags"`
}

type tagJSON struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func tagsToMap(tags []tagJSON) map[string]string {
	m := make(map[string]string, len(tags))
	for _, t := range tags {
		m[t.Key] = t.Value
	}
	return m
}

func mapToTags(m map[string]string) []tagJSON {
	tags := make([]tagJSON, 0, len(m))
	for k, v := range m {
		tags = append(tags, tagJSON{Key: k, Value: v})
	}
	return tags
}

func handleRegisterDomain(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req registerDomainRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Name == "" {
		return jsonErr(service.ErrValidation("name is required."))
	}
	if !store.RegisterDomain(req.Name, req.Description, req.WorkflowExecutionRetentionPeriodInDays, tagsToMap(req.Tags)) {
		return jsonErr(service.NewAWSError("DomainAlreadyExistsFault", "Domain "+req.Name+" already exists.", http.StatusBadRequest))
	}
	return emptyOK()
}

// ---- DescribeDomain ----

type describeDomainRequest struct {
	Name string `json:"name"`
}

type domainInfoJSON struct {
	Name        string `json:"name"`
	Status      string `json:"status"`
	Description string `json:"description,omitempty"`
	Arn         string `json:"arn"`
}

type domainConfigJSON struct {
	WorkflowExecutionRetentionPeriodInDays string `json:"workflowExecutionRetentionPeriodInDays"`
}

type describeDomainResponse struct {
	DomainInfo    domainInfoJSON   `json:"domainInfo"`
	Configuration domainConfigJSON `json:"configuration"`
}

func handleDescribeDomain(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeDomainRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Name == "" {
		return jsonErr(service.ErrValidation("name is required."))
	}
	d, ok := store.DescribeDomain(req.Name)
	if !ok {
		return jsonErr(service.NewAWSError("UnknownResourceFault", "Domain "+req.Name+" not found.", http.StatusBadRequest))
	}
	return jsonOK(&describeDomainResponse{
		DomainInfo:    domainInfoJSON{Name: d.Name, Status: d.Status, Description: d.Description, Arn: d.ARN},
		Configuration: domainConfigJSON{WorkflowExecutionRetentionPeriodInDays: d.WorkflowExecutionRetentionPeriodInDays},
	})
}

// ---- ListDomains ----

type listDomainsRequest struct {
	RegistrationStatus string `json:"registrationStatus"`
}

type listDomainsResponse struct {
	DomainInfos []domainInfoJSON `json:"domainInfos"`
}

func handleListDomains(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listDomainsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.RegistrationStatus == "" {
		req.RegistrationStatus = "REGISTERED"
	}
	domains := store.ListDomains(req.RegistrationStatus)
	items := make([]domainInfoJSON, 0, len(domains))
	for _, d := range domains {
		items = append(items, domainInfoJSON{Name: d.Name, Status: d.Status, Description: d.Description, Arn: d.ARN})
	}
	return jsonOK(&listDomainsResponse{DomainInfos: items})
}

// ---- DeprecateDomain ----

type deprecateDomainRequest struct {
	Name string `json:"name"`
}

func handleDeprecateDomain(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deprecateDomainRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Name == "" {
		return jsonErr(service.ErrValidation("name is required."))
	}
	if !store.DeprecateDomain(req.Name) {
		return jsonErr(service.NewAWSError("UnknownResourceFault", "Domain "+req.Name+" not found or already deprecated.", http.StatusBadRequest))
	}
	return emptyOK()
}

// ---- RegisterWorkflowType ----

type registerWorkflowTypeRequest struct {
	Domain                             string `json:"domain"`
	Name                               string `json:"name"`
	Version                            string `json:"version"`
	Description                        string `json:"description"`
	DefaultTaskList                    *struct{ Name string `json:"name"` } `json:"defaultTaskList"`
	DefaultExecutionStartToCloseTimeout string `json:"defaultExecutionStartToCloseTimeout"`
	DefaultTaskStartToCloseTimeout     string `json:"defaultTaskStartToCloseTimeout"`
	DefaultChildPolicy                 string `json:"defaultChildPolicy"`
}

func handleRegisterWorkflowType(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req registerWorkflowTypeRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Domain == "" || req.Name == "" || req.Version == "" {
		return jsonErr(service.ErrValidation("domain, name, and version are required."))
	}
	taskList := ""
	if req.DefaultTaskList != nil {
		taskList = req.DefaultTaskList.Name
	}
	if !store.RegisterWorkflowType(req.Domain, req.Name, req.Version, req.Description, taskList, req.DefaultExecutionStartToCloseTimeout, req.DefaultTaskStartToCloseTimeout, req.DefaultChildPolicy) {
		return jsonErr(service.NewAWSError("TypeAlreadyExistsFault", "Workflow type "+req.Name+":"+req.Version+" already exists.", http.StatusBadRequest))
	}
	return emptyOK()
}

// ---- DescribeWorkflowType ----

type workflowTypeJSON struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type describeWorkflowTypeRequest struct {
	Domain       string            `json:"domain"`
	WorkflowType workflowTypeJSON `json:"workflowType"`
}

type workflowTypeInfoJSON struct {
	WorkflowType workflowTypeJSON `json:"workflowType"`
	Status       string           `json:"status"`
	Description  string           `json:"description,omitempty"`
	CreationDate float64          `json:"creationDate"`
}

type workflowTypeConfigJSON struct {
	DefaultTaskList                    *taskListJSON `json:"defaultTaskList,omitempty"`
	DefaultExecutionStartToCloseTimeout string       `json:"defaultExecutionStartToCloseTimeout,omitempty"`
	DefaultTaskStartToCloseTimeout     string        `json:"defaultTaskStartToCloseTimeout,omitempty"`
	DefaultChildPolicy                 string        `json:"defaultChildPolicy,omitempty"`
}

type taskListJSON struct {
	Name string `json:"name"`
}

type describeWorkflowTypeResponse struct {
	TypeInfo      workflowTypeInfoJSON   `json:"typeInfo"`
	Configuration workflowTypeConfigJSON `json:"configuration"`
}

func handleDescribeWorkflowType(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeWorkflowTypeRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Domain == "" || req.WorkflowType.Name == "" || req.WorkflowType.Version == "" {
		return jsonErr(service.ErrValidation("domain and workflowType (name, version) are required."))
	}
	wt, ok := store.DescribeWorkflowType(req.Domain, req.WorkflowType.Name, req.WorkflowType.Version)
	if !ok {
		return jsonErr(service.NewAWSError("UnknownResourceFault", "Workflow type not found.", http.StatusBadRequest))
	}
	resp := &describeWorkflowTypeResponse{
		TypeInfo: workflowTypeInfoJSON{
			WorkflowType: workflowTypeJSON{Name: wt.Name, Version: wt.Version},
			Status: wt.Status, Description: wt.Description,
			CreationDate: float64(wt.CreationDate.Unix()),
		},
		Configuration: workflowTypeConfigJSON{
			DefaultExecutionStartToCloseTimeout: wt.DefaultExecutionTimeout,
			DefaultTaskStartToCloseTimeout:      wt.DefaultTaskTimeout,
			DefaultChildPolicy:                  wt.DefaultChildPolicy,
		},
	}
	if wt.DefaultTaskList != "" {
		resp.Configuration.DefaultTaskList = &taskListJSON{Name: wt.DefaultTaskList}
	}
	return jsonOK(resp)
}

// ---- ListWorkflowTypes ----

type listWorkflowTypesRequest struct {
	Domain             string `json:"domain"`
	RegistrationStatus string `json:"registrationStatus"`
	Name               string `json:"name"`
}

type listWorkflowTypesResponse struct {
	TypeInfos []workflowTypeInfoJSON `json:"typeInfos"`
}

func handleListWorkflowTypes(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listWorkflowTypesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Domain == "" || req.RegistrationStatus == "" {
		return jsonErr(service.ErrValidation("domain and registrationStatus are required."))
	}
	types := store.ListWorkflowTypes(req.Domain, req.RegistrationStatus, req.Name)
	items := make([]workflowTypeInfoJSON, 0, len(types))
	for _, wt := range types {
		items = append(items, workflowTypeInfoJSON{
			WorkflowType: workflowTypeJSON{Name: wt.Name, Version: wt.Version},
			Status: wt.Status, Description: wt.Description,
			CreationDate: float64(wt.CreationDate.Unix()),
		})
	}
	return jsonOK(&listWorkflowTypesResponse{TypeInfos: items})
}

// ---- DeprecateWorkflowType ----

type deprecateWorkflowTypeRequest struct {
	Domain       string           `json:"domain"`
	WorkflowType workflowTypeJSON `json:"workflowType"`
}

func handleDeprecateWorkflowType(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deprecateWorkflowTypeRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if !store.DeprecateWorkflowType(req.Domain, req.WorkflowType.Name, req.WorkflowType.Version) {
		return jsonErr(service.NewAWSError("UnknownResourceFault", "Workflow type not found or already deprecated.", http.StatusBadRequest))
	}
	return emptyOK()
}

// ---- RegisterActivityType ----

type registerActivityTypeRequest struct {
	Domain                            string `json:"domain"`
	Name                              string `json:"name"`
	Version                           string `json:"version"`
	Description                       string `json:"description"`
	DefaultTaskList                   *struct{ Name string `json:"name"` } `json:"defaultTaskList"`
	DefaultTaskStartToCloseTimeout    string `json:"defaultTaskStartToCloseTimeout"`
	DefaultTaskHeartbeatTimeout       string `json:"defaultTaskHeartbeatTimeout"`
	DefaultTaskScheduleToStartTimeout string `json:"defaultTaskScheduleToStartTimeout"`
	DefaultTaskScheduleToCloseTimeout string `json:"defaultTaskScheduleToCloseTimeout"`
}

func handleRegisterActivityType(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req registerActivityTypeRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Domain == "" || req.Name == "" || req.Version == "" {
		return jsonErr(service.ErrValidation("domain, name, and version are required."))
	}
	taskList := ""
	if req.DefaultTaskList != nil {
		taskList = req.DefaultTaskList.Name
	}
	if !store.RegisterActivityType(req.Domain, req.Name, req.Version, req.Description, taskList, req.DefaultTaskStartToCloseTimeout, req.DefaultTaskHeartbeatTimeout, req.DefaultTaskScheduleToStartTimeout, req.DefaultTaskScheduleToCloseTimeout) {
		return jsonErr(service.NewAWSError("TypeAlreadyExistsFault", "Activity type "+req.Name+":"+req.Version+" already exists.", http.StatusBadRequest))
	}
	return emptyOK()
}

// ---- DescribeActivityType ----

type activityTypeJSON struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type describeActivityTypeRequest struct {
	Domain       string           `json:"domain"`
	ActivityType activityTypeJSON `json:"activityType"`
}

type activityTypeInfoJSON struct {
	ActivityType activityTypeJSON `json:"activityType"`
	Status       string           `json:"status"`
	Description  string           `json:"description,omitempty"`
	CreationDate float64          `json:"creationDate"`
}

type activityTypeConfigJSON struct {
	DefaultTaskList                   *taskListJSON `json:"defaultTaskList,omitempty"`
	DefaultTaskStartToCloseTimeout    string        `json:"defaultTaskStartToCloseTimeout,omitempty"`
	DefaultTaskHeartbeatTimeout       string        `json:"defaultTaskHeartbeatTimeout,omitempty"`
	DefaultTaskScheduleToStartTimeout string        `json:"defaultTaskScheduleToStartTimeout,omitempty"`
	DefaultTaskScheduleToCloseTimeout string        `json:"defaultTaskScheduleToCloseTimeout,omitempty"`
}

type describeActivityTypeResponse struct {
	TypeInfo      activityTypeInfoJSON   `json:"typeInfo"`
	Configuration activityTypeConfigJSON `json:"configuration"`
}

func handleDescribeActivityType(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeActivityTypeRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Domain == "" || req.ActivityType.Name == "" || req.ActivityType.Version == "" {
		return jsonErr(service.ErrValidation("domain and activityType (name, version) are required."))
	}
	at, ok := store.DescribeActivityType(req.Domain, req.ActivityType.Name, req.ActivityType.Version)
	if !ok {
		return jsonErr(service.NewAWSError("UnknownResourceFault", "Activity type not found.", http.StatusBadRequest))
	}
	resp := &describeActivityTypeResponse{
		TypeInfo: activityTypeInfoJSON{
			ActivityType: activityTypeJSON{Name: at.Name, Version: at.Version},
			Status: at.Status, Description: at.Description,
			CreationDate: float64(at.CreationDate.Unix()),
		},
		Configuration: activityTypeConfigJSON{
			DefaultTaskStartToCloseTimeout:    at.DefaultTaskTimeout,
			DefaultTaskHeartbeatTimeout:       at.DefaultHeartbeatTimeout,
			DefaultTaskScheduleToStartTimeout: at.DefaultScheduleToStartTimeout,
			DefaultTaskScheduleToCloseTimeout: at.DefaultScheduleToCloseTimeout,
		},
	}
	if at.DefaultTaskList != "" {
		resp.Configuration.DefaultTaskList = &taskListJSON{Name: at.DefaultTaskList}
	}
	return jsonOK(resp)
}

// ---- ListActivityTypes ----

type listActivityTypesRequest struct {
	Domain             string `json:"domain"`
	RegistrationStatus string `json:"registrationStatus"`
	Name               string `json:"name"`
}

type listActivityTypesResponse struct {
	TypeInfos []activityTypeInfoJSON `json:"typeInfos"`
}

func handleListActivityTypes(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listActivityTypesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Domain == "" || req.RegistrationStatus == "" {
		return jsonErr(service.ErrValidation("domain and registrationStatus are required."))
	}
	types := store.ListActivityTypes(req.Domain, req.RegistrationStatus, req.Name)
	items := make([]activityTypeInfoJSON, 0, len(types))
	for _, at := range types {
		items = append(items, activityTypeInfoJSON{
			ActivityType: activityTypeJSON{Name: at.Name, Version: at.Version},
			Status: at.Status, Description: at.Description,
			CreationDate: float64(at.CreationDate.Unix()),
		})
	}
	return jsonOK(&listActivityTypesResponse{TypeInfos: items})
}

// ---- DeprecateActivityType ----

type deprecateActivityTypeRequest struct {
	Domain       string           `json:"domain"`
	ActivityType activityTypeJSON `json:"activityType"`
}

func handleDeprecateActivityType(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deprecateActivityTypeRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if !store.DeprecateActivityType(req.Domain, req.ActivityType.Name, req.ActivityType.Version) {
		return jsonErr(service.NewAWSError("UnknownResourceFault", "Activity type not found or already deprecated.", http.StatusBadRequest))
	}
	return emptyOK()
}

// ---- StartWorkflowExecution ----

type startWorkflowExecutionRequest struct {
	Domain       string           `json:"domain"`
	WorkflowId   string           `json:"workflowId"`
	WorkflowType workflowTypeJSON `json:"workflowType"`
	TaskList     *taskListJSON    `json:"taskList"`
	Input        string           `json:"input"`
	Tags         []tagJSON        `json:"tags"`
}

type startWorkflowExecutionResponse struct {
	RunId string `json:"runId"`
}

func handleStartWorkflowExecution(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req startWorkflowExecutionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Domain == "" || req.WorkflowId == "" || req.WorkflowType.Name == "" || req.WorkflowType.Version == "" {
		return jsonErr(service.ErrValidation("domain, workflowId, and workflowType (name, version) are required."))
	}
	taskList := ""
	if req.TaskList != nil {
		taskList = req.TaskList.Name
	}
	runID, errKind := store.StartWorkflowExecution(req.Domain, req.WorkflowId, req.WorkflowType.Name, req.WorkflowType.Version, taskList, req.Input, tagsToMap(req.Tags))
	if errKind != "" {
		switch errKind {
		case "duplicate":
			return jsonErr(service.NewAWSError("WorkflowExecutionAlreadyStartedFault", "Workflow execution already started.", http.StatusBadRequest))
		default:
			return jsonErr(service.NewAWSError("UnknownResourceFault", "Resource not found: "+errKind, http.StatusBadRequest))
		}
	}
	return jsonOK(&startWorkflowExecutionResponse{RunId: runID})
}

// ---- DescribeWorkflowExecution ----

type executionJSON struct {
	WorkflowId string `json:"workflowId"`
	RunId      string `json:"runId"`
}

type describeWorkflowExecutionRequest struct {
	Domain    string        `json:"domain"`
	Execution executionJSON `json:"execution"`
}

type executionInfoJSON struct {
	Execution    executionJSON    `json:"execution"`
	WorkflowType workflowTypeJSON `json:"workflowType"`
	StartTimestamp float64        `json:"startTimestamp"`
	CloseTimestamp float64        `json:"closeTimestamp,omitempty"`
	ExecutionStatus string        `json:"executionStatus"`
	CloseStatus     string        `json:"closeStatus,omitempty"`
}

type executionConfigJSON struct {
	TaskList *taskListJSON `json:"taskList,omitempty"`
}

type describeWorkflowExecutionResponse struct {
	ExecutionInfo          executionInfoJSON   `json:"executionInfo"`
	ExecutionConfiguration executionConfigJSON `json:"executionConfiguration"`
	OpenCounts             map[string]int      `json:"openCounts"`
}

func handleDescribeWorkflowExecution(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeWorkflowExecutionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Domain == "" || req.Execution.WorkflowId == "" || req.Execution.RunId == "" {
		return jsonErr(service.ErrValidation("domain and execution (workflowId, runId) are required."))
	}
	exec, ok := store.DescribeWorkflowExecution(req.Domain, req.Execution.WorkflowId, req.Execution.RunId)
	if !ok {
		return jsonErr(service.NewAWSError("UnknownResourceFault", "Workflow execution not found.", http.StatusBadRequest))
	}
	info := executionInfoJSON{
		Execution:    executionJSON{WorkflowId: exec.WorkflowID, RunId: exec.RunID},
		WorkflowType: workflowTypeJSON{Name: exec.WorkflowType.Name, Version: exec.WorkflowType.Version},
		StartTimestamp: float64(exec.StartTime.Unix()),
		ExecutionStatus: exec.Status,
		CloseStatus:     exec.CloseStatus,
	}
	if exec.CloseTime != nil {
		info.CloseTimestamp = float64(exec.CloseTime.Unix())
	}
	openCounts := map[string]int{
		"openActivityTasks":           len(exec.PendingActivities),
		"openDecisionTasks":           0,
		"openTimers":                  0,
		"openChildWorkflowExecutions": 0,
		"openLambdaFunctions":         0,
	}
	if exec.PendingDecisionTask {
		openCounts["openDecisionTasks"] = 1
	}
	return jsonOK(&describeWorkflowExecutionResponse{
		ExecutionInfo:          info,
		ExecutionConfiguration: executionConfigJSON{TaskList: &taskListJSON{Name: exec.TaskList}},
		OpenCounts:             openCounts,
	})
}

// ---- ListOpenWorkflowExecutions ----

type listOpenWorkflowExecutionsRequest struct {
	Domain          string `json:"domain"`
	StartTimeFilter struct {
		OldestDate float64 `json:"oldestDate"`
		LatestDate float64 `json:"latestDate"`
	} `json:"startTimeFilter"`
}

type executionInfosResponse struct {
	ExecutionInfos []executionInfoJSON `json:"executionInfos"`
}

func handleListOpenWorkflowExecutions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listOpenWorkflowExecutionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Domain == "" {
		return jsonErr(service.ErrValidation("domain is required."))
	}
	execs := store.ListOpenWorkflowExecutions(req.Domain)
	items := make([]executionInfoJSON, 0, len(execs))
	for _, exec := range execs {
		items = append(items, executionInfoJSON{
			Execution:       executionJSON{WorkflowId: exec.WorkflowID, RunId: exec.RunID},
			WorkflowType:    workflowTypeJSON{Name: exec.WorkflowType.Name, Version: exec.WorkflowType.Version},
			StartTimestamp:  float64(exec.StartTime.Unix()),
			ExecutionStatus: exec.Status,
		})
	}
	return jsonOK(&executionInfosResponse{ExecutionInfos: items})
}

// ---- ListClosedWorkflowExecutions ----

type listClosedWorkflowExecutionsRequest struct {
	Domain          string `json:"domain"`
	StartTimeFilter struct {
		OldestDate float64 `json:"oldestDate"`
		LatestDate float64 `json:"latestDate"`
	} `json:"startTimeFilter"`
}

func handleListClosedWorkflowExecutions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listClosedWorkflowExecutionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Domain == "" {
		return jsonErr(service.ErrValidation("domain is required."))
	}
	execs := store.ListClosedWorkflowExecutions(req.Domain)
	items := make([]executionInfoJSON, 0, len(execs))
	for _, exec := range execs {
		info := executionInfoJSON{
			Execution:       executionJSON{WorkflowId: exec.WorkflowID, RunId: exec.RunID},
			WorkflowType:    workflowTypeJSON{Name: exec.WorkflowType.Name, Version: exec.WorkflowType.Version},
			StartTimestamp:  float64(exec.StartTime.Unix()),
			ExecutionStatus: exec.Status,
			CloseStatus:     exec.CloseStatus,
		}
		if exec.CloseTime != nil {
			info.CloseTimestamp = float64(exec.CloseTime.Unix())
		}
		items = append(items, info)
	}
	return jsonOK(&executionInfosResponse{ExecutionInfos: items})
}

// ---- TerminateWorkflowExecution ----

type terminateWorkflowExecutionRequest struct {
	Domain      string `json:"domain"`
	WorkflowId  string `json:"workflowId"`
	RunId       string `json:"runId"`
	Reason      string `json:"reason"`
	Details     string `json:"details"`
	ChildPolicy string `json:"childPolicy"`
}

func handleTerminateWorkflowExecution(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req terminateWorkflowExecutionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Domain == "" || req.WorkflowId == "" {
		return jsonErr(service.ErrValidation("domain and workflowId are required."))
	}
	if !store.TerminateWorkflowExecution(req.Domain, req.WorkflowId, req.RunId, req.Reason, req.Details, req.ChildPolicy) {
		return jsonErr(service.NewAWSError("UnknownResourceFault", "Workflow execution not found.", http.StatusBadRequest))
	}
	return emptyOK()
}

// ---- SignalWorkflowExecution ----

type signalWorkflowExecutionRequest struct {
	Domain     string `json:"domain"`
	WorkflowId string `json:"workflowId"`
	RunId      string `json:"runId"`
	SignalName string `json:"signalName"`
	Input      string `json:"input"`
}

func handleSignalWorkflowExecution(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req signalWorkflowExecutionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Domain == "" || req.WorkflowId == "" || req.SignalName == "" {
		return jsonErr(service.ErrValidation("domain, workflowId, and signalName are required."))
	}
	if !store.SignalWorkflowExecution(req.Domain, req.WorkflowId, req.RunId, req.SignalName, req.Input) {
		return jsonErr(service.NewAWSError("UnknownResourceFault", "Workflow execution not found.", http.StatusBadRequest))
	}
	return emptyOK()
}

// ---- RequestCancelWorkflowExecution ----

type requestCancelWorkflowExecutionRequest struct {
	Domain     string `json:"domain"`
	WorkflowId string `json:"workflowId"`
	RunId      string `json:"runId"`
}

func handleRequestCancelWorkflowExecution(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req requestCancelWorkflowExecutionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Domain == "" || req.WorkflowId == "" {
		return jsonErr(service.ErrValidation("domain and workflowId are required."))
	}
	if !store.RequestCancelWorkflowExecution(req.Domain, req.WorkflowId, req.RunId) {
		return jsonErr(service.NewAWSError("UnknownResourceFault", "Workflow execution not found.", http.StatusBadRequest))
	}
	return emptyOK()
}

// ---- PollForDecisionTask ----

type pollForDecisionTaskRequest struct {
	Domain   string        `json:"domain"`
	TaskList taskListJSON `json:"taskList"`
}

type decisionTaskJSON struct {
	TaskToken         string           `json:"taskToken"`
	StartedEventId    int              `json:"startedEventId"`
	WorkflowExecution executionJSON    `json:"workflowExecution"`
	WorkflowType      workflowTypeJSON `json:"workflowType"`
	Events            []any            `json:"events"`
	PreviousStartedEventId int         `json:"previousStartedEventId"`
}

func handlePollForDecisionTask(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req pollForDecisionTaskRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Domain == "" || req.TaskList.Name == "" {
		return jsonErr(service.ErrValidation("domain and taskList are required."))
	}
	exec, token := store.PollForDecisionTask(req.Domain, req.TaskList.Name)
	if exec == nil {
		// Long poll returns empty task token when no tasks available.
		return jsonOK(&decisionTaskJSON{TaskToken: "", Events: []any{}})
	}
	return jsonOK(&decisionTaskJSON{
		TaskToken:      token,
		StartedEventId: 1,
		WorkflowExecution: executionJSON{WorkflowId: exec.WorkflowID, RunId: exec.RunID},
		WorkflowType:      workflowTypeJSON{Name: exec.WorkflowType.Name, Version: exec.WorkflowType.Version},
		Events:             []any{},
	})
}

// ---- RespondDecisionTaskCompleted ----

type respondDecisionTaskCompletedRequest struct {
	TaskToken string           `json:"taskToken"`
	Decisions []map[string]any `json:"decisions"`
}

func handleRespondDecisionTaskCompleted(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req respondDecisionTaskCompletedRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.TaskToken == "" {
		return jsonErr(service.ErrValidation("taskToken is required."))
	}
	if !store.RespondDecisionTaskCompleted(req.TaskToken, req.Decisions) {
		return jsonErr(service.NewAWSError("UnknownResourceFault", "Decision task not found.", http.StatusBadRequest))
	}
	return emptyOK()
}

// ---- PollForActivityTask ----

type pollForActivityTaskRequest struct {
	Domain   string       `json:"domain"`
	TaskList taskListJSON `json:"taskList"`
}

type activityTaskJSON struct {
	TaskToken         string           `json:"taskToken"`
	ActivityId        string           `json:"activityId"`
	StartedEventId    int              `json:"startedEventId"`
	WorkflowExecution executionJSON    `json:"workflowExecution"`
	ActivityType      activityTypeJSON `json:"activityType"`
	Input             string           `json:"input"`
}

func handlePollForActivityTask(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req pollForActivityTaskRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Domain == "" || req.TaskList.Name == "" {
		return jsonErr(service.ErrValidation("domain and taskList are required."))
	}
	exec, activity := store.PollForActivityTask(req.Domain, req.TaskList.Name)
	if exec == nil || activity == nil {
		return jsonOK(&activityTaskJSON{TaskToken: ""})
	}
	return jsonOK(&activityTaskJSON{
		TaskToken:         activity.TaskToken,
		ActivityId:        activity.ActivityID,
		StartedEventId:    1,
		WorkflowExecution: executionJSON{WorkflowId: exec.WorkflowID, RunId: exec.RunID},
		ActivityType:      activityTypeJSON{Name: activity.ActivityType, Version: "1.0"},
		Input:             activity.Input,
	})
}

// ---- RespondActivityTaskCompleted ----

type respondActivityTaskCompletedRequest struct {
	TaskToken string `json:"taskToken"`
	Result    string `json:"result"`
}

func handleRespondActivityTaskCompleted(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req respondActivityTaskCompletedRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.TaskToken == "" {
		return jsonErr(service.ErrValidation("taskToken is required."))
	}
	if !store.RespondActivityTaskCompleted(req.TaskToken, req.Result) {
		return jsonErr(service.NewAWSError("UnknownResourceFault", "Activity task not found.", http.StatusBadRequest))
	}
	return emptyOK()
}

// ---- RespondActivityTaskFailed ----

type respondActivityTaskFailedRequest struct {
	TaskToken string `json:"taskToken"`
	Reason    string `json:"reason"`
	Details   string `json:"details"`
}

func handleRespondActivityTaskFailed(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req respondActivityTaskFailedRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.TaskToken == "" {
		return jsonErr(service.ErrValidation("taskToken is required."))
	}
	if !store.RespondActivityTaskFailed(req.TaskToken, req.Reason, req.Details) {
		return jsonErr(service.NewAWSError("UnknownResourceFault", "Activity task not found.", http.StatusBadRequest))
	}
	return emptyOK()
}

// ---- TagResource ----

type tagResourceRequest struct {
	ResourceArn string    `json:"resourceArn"`
	Tags        []tagJSON `json:"tags"`
}

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req tagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ResourceArn == "" {
		return jsonErr(service.ErrValidation("resourceArn is required."))
	}
	if !store.TagResource(req.ResourceArn, tagsToMap(req.Tags)) {
		return jsonErr(service.NewAWSError("UnknownResourceFault", "Resource not found.", http.StatusBadRequest))
	}
	return emptyOK()
}

// ---- UntagResource ----

type untagResourceRequest struct {
	ResourceArn string   `json:"resourceArn"`
	TagKeys     []string `json:"tagKeys"`
}

func handleUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req untagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ResourceArn == "" {
		return jsonErr(service.ErrValidation("resourceArn is required."))
	}
	if !store.UntagResource(req.ResourceArn, req.TagKeys) {
		return jsonErr(service.NewAWSError("UnknownResourceFault", "Resource not found.", http.StatusBadRequest))
	}
	return emptyOK()
}

// ---- ListTagsForResource ----

type listTagsRequest struct {
	ResourceArn string `json:"resourceArn"`
}

type listTagsResponse struct {
	Tags []tagJSON `json:"tags"`
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listTagsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ResourceArn == "" {
		return jsonErr(service.ErrValidation("resourceArn is required."))
	}
	tags, ok := store.ListTagsForResource(req.ResourceArn)
	if !ok {
		return jsonErr(service.NewAWSError("UnknownResourceFault", "Resource not found.", http.StatusBadRequest))
	}
	return jsonOK(&listTagsResponse{Tags: mapToTags(tags)})
}
