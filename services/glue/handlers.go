package glue

import (
	"encoding/json"
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// ---- JSON helpers ----

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
		return service.NewAWSError("InvalidInputException", "Invalid JSON in request body.", http.StatusBadRequest)
	}
	return nil
}

// ---- Database handlers ----

type createDatabaseRequest struct {
	DatabaseInput struct {
		Name        string `json:"Name"`
		Description string `json:"Description"`
		LocationUri string `json:"LocationUri"`
	} `json:"DatabaseInput"`
}

func handleCreateDatabase(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createDatabaseRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.DatabaseInput.Name == "" {
		return jsonErr(service.ErrValidation("Database name is required."))
	}
	_, ok := store.CreateDatabase(req.DatabaseInput.Name, req.DatabaseInput.Description, req.DatabaseInput.LocationUri)
	if !ok {
		return jsonErr(service.ErrAlreadyExists("Database", req.DatabaseInput.Name))
	}
	return jsonOK(struct{}{})
}

type getDatabaseRequest struct {
	Name string `json:"Name"`
}

type databaseJSON struct {
	Name        string  `json:"Name"`
	Description string  `json:"Description,omitempty"`
	LocationUri string  `json:"LocationUri,omitempty"`
	CreateTime  float64 `json:"CreateTime"`
}

func handleGetDatabase(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getDatabaseRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	db, ok := store.GetDatabase(req.Name)
	if !ok {
		return jsonErr(service.NewAWSError("EntityNotFoundException", "Database "+req.Name+" not found.", http.StatusNotFound))
	}
	return jsonOK(map[string]any{
		"Database": databaseJSON{Name: db.Name, Description: db.Description, LocationUri: db.LocationURI, CreateTime: float64(db.CreateTime.Unix())},
	})
}

func handleGetDatabases(_ *service.RequestContext, store *Store) (*service.Response, error) {
	dbs := store.ListDatabases()
	list := make([]databaseJSON, 0, len(dbs))
	for _, db := range dbs {
		list = append(list, databaseJSON{Name: db.Name, Description: db.Description, LocationUri: db.LocationURI, CreateTime: float64(db.CreateTime.Unix())})
	}
	return jsonOK(map[string]any{"DatabaseList": list})
}

type deleteDatabaseRequest struct {
	Name string `json:"Name"`
}

func handleDeleteDatabase(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteDatabaseRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if !store.DeleteDatabase(req.Name) {
		return jsonErr(service.NewAWSError("EntityNotFoundException", "Database "+req.Name+" not found.", http.StatusNotFound))
	}
	return jsonOK(struct{}{})
}

// ---- Table handlers ----

type columnJSON struct {
	Name    string `json:"Name"`
	Type    string `json:"Type"`
	Comment string `json:"Comment,omitempty"`
}

type storageDescJSON struct {
	Location     string       `json:"Location,omitempty"`
	InputFormat  string       `json:"InputFormat,omitempty"`
	OutputFormat string       `json:"OutputFormat,omitempty"`
	Columns      []columnJSON `json:"Columns,omitempty"`
}

type tableInputJSON struct {
	Name              string          `json:"Name"`
	Description       string          `json:"Description"`
	StorageDescriptor storageDescJSON `json:"StorageDescriptor"`
	Parameters        map[string]string `json:"Parameters"`
}

type createTableRequest struct {
	DatabaseName string         `json:"DatabaseName"`
	TableInput   tableInputJSON `json:"TableInput"`
}

func toTable(input tableInputJSON) *Table {
	cols := make([]Column, len(input.StorageDescriptor.Columns))
	for i, c := range input.StorageDescriptor.Columns {
		cols[i] = Column{Name: c.Name, Type: c.Type, Comment: c.Comment}
	}
	params := input.Parameters
	if params == nil {
		params = make(map[string]string)
	}
	return &Table{
		Name:        input.Name,
		Description: input.Description,
		StorageDesc: StorageDescriptor{
			Location:     input.StorageDescriptor.Location,
			InputFormat:  input.StorageDescriptor.InputFormat,
			OutputFormat: input.StorageDescriptor.OutputFormat,
			Columns:      cols,
		},
		Parameters: params,
	}
}

type tableJSON struct {
	Name              string          `json:"Name"`
	DatabaseName      string          `json:"DatabaseName"`
	Description       string          `json:"Description,omitempty"`
	StorageDescriptor storageDescJSON `json:"StorageDescriptor"`
	Parameters        map[string]string `json:"Parameters,omitempty"`
	CreateTime        float64         `json:"CreateTime"`
	UpdateTime        float64         `json:"UpdateTime"`
}

func toTableJSON(t *Table) tableJSON {
	cols := make([]columnJSON, len(t.StorageDesc.Columns))
	for i, c := range t.StorageDesc.Columns {
		cols[i] = columnJSON{Name: c.Name, Type: c.Type, Comment: c.Comment}
	}
	return tableJSON{
		Name:         t.Name,
		DatabaseName: t.DatabaseName,
		Description:  t.Description,
		StorageDescriptor: storageDescJSON{
			Location:     t.StorageDesc.Location,
			InputFormat:  t.StorageDesc.InputFormat,
			OutputFormat: t.StorageDesc.OutputFormat,
			Columns:      cols,
		},
		Parameters: t.Parameters,
		CreateTime: float64(t.CreateTime.Unix()),
		UpdateTime: float64(t.UpdateTime.Unix()),
	}
}

func handleCreateTable(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createTableRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.DatabaseName == "" || req.TableInput.Name == "" {
		return jsonErr(service.ErrValidation("DatabaseName and TableInput.Name are required."))
	}
	t := toTable(req.TableInput)
	if !store.CreateTable(req.DatabaseName, t) {
		return jsonErr(service.ErrAlreadyExists("Table", req.TableInput.Name))
	}
	return jsonOK(struct{}{})
}

type getTableRequest struct {
	DatabaseName string `json:"DatabaseName"`
	Name         string `json:"Name"`
}

func handleGetTable(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getTableRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	t, ok := store.GetTable(req.DatabaseName, req.Name)
	if !ok {
		return jsonErr(service.NewAWSError("EntityNotFoundException", "Table "+req.Name+" not found.", http.StatusNotFound))
	}
	return jsonOK(map[string]any{"Table": toTableJSON(t)})
}

type getTablesRequest struct {
	DatabaseName string `json:"DatabaseName"`
}

func handleGetTables(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getTablesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	tables := store.ListTables(req.DatabaseName)
	list := make([]tableJSON, 0, len(tables))
	for _, t := range tables {
		list = append(list, toTableJSON(t))
	}
	return jsonOK(map[string]any{"TableList": list})
}

type deleteTableRequest struct {
	DatabaseName string `json:"DatabaseName"`
	Name         string `json:"Name"`
}

func handleDeleteTable(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteTableRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if !store.DeleteTable(req.DatabaseName, req.Name) {
		return jsonErr(service.NewAWSError("EntityNotFoundException", "Table "+req.Name+" not found.", http.StatusNotFound))
	}
	return jsonOK(struct{}{})
}

type updateTableRequest struct {
	DatabaseName string         `json:"DatabaseName"`
	TableInput   tableInputJSON `json:"TableInput"`
}

func handleUpdateTable(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req updateTableRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	t := toTable(req.TableInput)
	if !store.UpdateTable(req.DatabaseName, t) {
		return jsonErr(service.NewAWSError("EntityNotFoundException", "Table "+req.TableInput.Name+" not found.", http.StatusNotFound))
	}
	return jsonOK(struct{}{})
}

// ---- Crawler handlers ----

type s3TargetJSON struct {
	Path       string   `json:"Path"`
	Exclusions []string `json:"Exclusions"`
}

type crawlerTargetsJSON struct {
	S3Targets []s3TargetJSON `json:"S3Targets"`
}

type createCrawlerRequest struct {
	Name         string             `json:"Name"`
	Role         string             `json:"Role"`
	DatabaseName string             `json:"DatabaseName"`
	Description  string             `json:"Description"`
	Targets      crawlerTargetsJSON `json:"Targets"`
	Schedule     string             `json:"Schedule"`
	Tags         map[string]string  `json:"Tags"`
}

type crawlerJSON struct {
	Name         string             `json:"Name"`
	Role         string             `json:"Role"`
	DatabaseName string             `json:"DatabaseName"`
	Description  string             `json:"Description,omitempty"`
	Targets      crawlerTargetsJSON `json:"Targets"`
	State        string             `json:"State"`
	Schedule     string             `json:"Schedule,omitempty"`
	CreationTime float64            `json:"CreationTime"`
	LastUpdated  float64            `json:"LastUpdated"`
}

func toCrawlerJSON(c *Crawler) crawlerJSON {
	targets := make([]s3TargetJSON, len(c.Targets.S3Targets))
	for i, t := range c.Targets.S3Targets {
		targets[i] = s3TargetJSON{Path: t.Path, Exclusions: t.Exclusions}
	}
	return crawlerJSON{
		Name:         c.Name,
		Role:         c.Role,
		DatabaseName: c.DatabaseName,
		Description:  c.Description,
		Targets:      crawlerTargetsJSON{S3Targets: targets},
		State:        c.State,
		Schedule:     c.Schedule,
		CreationTime: float64(c.CreateTime.Unix()),
		LastUpdated:  float64(c.LastUpdated.Unix()),
	}
}

func handleCreateCrawler(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createCrawlerRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Name == "" {
		return jsonErr(service.ErrValidation("Crawler name is required."))
	}
	targets := CrawlerTargets{}
	for _, t := range req.Targets.S3Targets {
		targets.S3Targets = append(targets.S3Targets, S3Target{Path: t.Path, Exclusions: t.Exclusions})
	}
	tags := req.Tags
	if tags == nil {
		tags = make(map[string]string)
	}
	_, ok := store.CreateCrawler(req.Name, req.Role, req.DatabaseName, req.Description, req.Schedule, targets, tags)
	if !ok {
		return jsonErr(service.ErrAlreadyExists("Crawler", req.Name))
	}
	return jsonOK(struct{}{})
}

type getCrawlerRequest struct {
	Name string `json:"Name"`
}

func handleGetCrawler(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getCrawlerRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	c, ok := store.GetCrawler(req.Name)
	if !ok {
		return jsonErr(service.NewAWSError("EntityNotFoundException", "Crawler "+req.Name+" not found.", http.StatusNotFound))
	}
	return jsonOK(map[string]any{"Crawler": toCrawlerJSON(c)})
}

func handleGetCrawlers(_ *service.RequestContext, store *Store) (*service.Response, error) {
	crawlers := store.ListCrawlers()
	list := make([]crawlerJSON, 0, len(crawlers))
	for _, c := range crawlers {
		list = append(list, toCrawlerJSON(c))
	}
	return jsonOK(map[string]any{"Crawlers": list})
}

type deleteCrawlerRequest struct {
	Name string `json:"Name"`
}

func handleDeleteCrawler(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteCrawlerRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if !store.DeleteCrawler(req.Name) {
		return jsonErr(service.NewAWSError("EntityNotFoundException", "Crawler "+req.Name+" not found.", http.StatusNotFound))
	}
	return jsonOK(struct{}{})
}

type startCrawlerRequest struct {
	Name string `json:"Name"`
}

func handleStartCrawler(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req startCrawlerRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if !store.StartCrawler(req.Name) {
		return jsonErr(service.NewAWSError("CrawlerRunningException", "Crawler "+req.Name+" is not in READY state.", http.StatusBadRequest))
	}
	return jsonOK(struct{}{})
}

type stopCrawlerRequest struct {
	Name string `json:"Name"`
}

func handleStopCrawler(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req stopCrawlerRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if !store.StopCrawler(req.Name) {
		return jsonErr(service.NewAWSError("CrawlerNotRunningException", "Crawler "+req.Name+" is not running.", http.StatusBadRequest))
	}
	return jsonOK(struct{}{})
}

// ---- Job handlers ----

type jobCommandJSON struct {
	Name           string `json:"Name"`
	ScriptLocation string `json:"ScriptLocation"`
	PythonVersion  string `json:"PythonVersion"`
}

type createJobRequest struct {
	Name            string            `json:"Name"`
	Role            string            `json:"Role"`
	Command         jobCommandJSON    `json:"Command"`
	Description     string            `json:"Description"`
	MaxRetries      int               `json:"MaxRetries"`
	MaxCapacity     float64           `json:"MaxCapacity"`
	GlueVersion     string            `json:"GlueVersion"`
	NumberOfWorkers int               `json:"NumberOfWorkers"`
	WorkerType      string            `json:"WorkerType"`
	Tags            map[string]string `json:"Tags"`
}

type jobJSON struct {
	Name            string         `json:"Name"`
	Role            string         `json:"Role"`
	Command         jobCommandJSON `json:"Command"`
	Description     string         `json:"Description,omitempty"`
	MaxRetries      int            `json:"MaxRetries"`
	MaxCapacity     float64        `json:"MaxCapacity"`
	GlueVersion     string         `json:"GlueVersion,omitempty"`
	NumberOfWorkers int            `json:"NumberOfWorkers"`
	WorkerType      string         `json:"WorkerType,omitempty"`
	CreatedOn       float64        `json:"CreatedOn"`
}

func toJobJSON(j *Job) jobJSON {
	return jobJSON{
		Name: j.Name, Role: j.Role,
		Command:         jobCommandJSON{Name: j.Command.Name, ScriptLocation: j.Command.ScriptLocation, PythonVersion: j.Command.PythonVersion},
		Description:     j.Description,
		MaxRetries:      j.MaxRetries,
		MaxCapacity:     j.MaxCapacity,
		GlueVersion:     j.GlueVersion,
		NumberOfWorkers: j.NumberOfWorkers,
		WorkerType:      j.WorkerType,
		CreatedOn:       float64(j.CreateTime.Unix()),
	}
}

func handleCreateJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createJobRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Name == "" {
		return jsonErr(service.ErrValidation("Job name is required."))
	}
	tags := req.Tags
	if tags == nil {
		tags = make(map[string]string)
	}
	j := &Job{
		Name:            req.Name,
		Role:            req.Role,
		Command:         JobCommand{Name: req.Command.Name, ScriptLocation: req.Command.ScriptLocation, PythonVersion: req.Command.PythonVersion},
		Description:     req.Description,
		MaxRetries:      req.MaxRetries,
		MaxCapacity:     req.MaxCapacity,
		GlueVersion:     req.GlueVersion,
		NumberOfWorkers: req.NumberOfWorkers,
		WorkerType:      req.WorkerType,
		Tags:            tags,
	}
	if !store.CreateJob(j) {
		return jsonErr(service.ErrAlreadyExists("Job", req.Name))
	}
	return jsonOK(map[string]string{"Name": req.Name})
}

type getJobRequest struct {
	JobName string `json:"JobName"`
}

func handleGetJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getJobRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	j, ok := store.GetJob(req.JobName)
	if !ok {
		return jsonErr(service.NewAWSError("EntityNotFoundException", "Job "+req.JobName+" not found.", http.StatusNotFound))
	}
	return jsonOK(map[string]any{"Job": toJobJSON(j)})
}

func handleGetJobs(_ *service.RequestContext, store *Store) (*service.Response, error) {
	jobs := store.ListJobs()
	list := make([]jobJSON, 0, len(jobs))
	for _, j := range jobs {
		list = append(list, toJobJSON(j))
	}
	return jsonOK(map[string]any{"Jobs": list})
}

type deleteJobRequest struct {
	JobName string `json:"JobName"`
}

func handleDeleteJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteJobRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if !store.DeleteJob(req.JobName) {
		return jsonErr(service.NewAWSError("EntityNotFoundException", "Job "+req.JobName+" not found.", http.StatusNotFound))
	}
	return jsonOK(map[string]string{"JobName": req.JobName})
}

// ---- JobRun handlers ----

type startJobRunRequest struct {
	JobName string `json:"JobName"`
}

type jobRunJSON struct {
	Id            string   `json:"Id"`
	JobName       string   `json:"JobName"`
	JobRunState   string   `json:"JobRunState"`
	StartedOn     float64  `json:"StartedOn"`
	CompletedOn   *float64 `json:"CompletedOn,omitempty"`
	Attempt       int      `json:"Attempt"`
	ExecutionTime int      `json:"ExecutionTime"`
	ErrorMessage  string   `json:"ErrorMessage,omitempty"`
}

func toJobRunJSON(r *JobRun) jobRunJSON {
	j := jobRunJSON{
		Id: r.ID, JobName: r.JobName, JobRunState: r.State,
		StartedOn: float64(r.StartedOn.Unix()), Attempt: r.Attempt, ExecutionTime: r.ExecutionTime,
		ErrorMessage: r.ErrorMessage,
	}
	if r.CompletedOn != nil {
		ct := float64(r.CompletedOn.Unix())
		j.CompletedOn = &ct
	}
	return j
}

func handleStartJobRun(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req startJobRunRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	run, ok := store.StartJobRun(req.JobName)
	if !ok {
		return jsonErr(service.NewAWSError("EntityNotFoundException", "Job "+req.JobName+" not found.", http.StatusNotFound))
	}
	return jsonOK(map[string]string{"JobRunId": run.ID})
}

type getJobRunRequest struct {
	JobName string `json:"JobName"`
	RunId   string `json:"RunId"`
}

func handleGetJobRun(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getJobRunRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	run, ok := store.GetJobRun(req.JobName, req.RunId)
	if !ok {
		return jsonErr(service.NewAWSError("EntityNotFoundException", "JobRun not found.", http.StatusNotFound))
	}
	return jsonOK(map[string]any{"JobRun": toJobRunJSON(run)})
}

type getJobRunsRequest struct {
	JobName string `json:"JobName"`
}

func handleGetJobRuns(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getJobRunsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	runs := store.ListJobRuns(req.JobName)
	list := make([]jobRunJSON, 0, len(runs))
	for _, r := range runs {
		list = append(list, toJobRunJSON(r))
	}
	return jsonOK(map[string]any{"JobRuns": list})
}

type batchStopJobRunRequest struct {
	JobName string   `json:"JobName"`
	JobRunIds []string `json:"JobRunIds"`
}

type batchStopJobRunSuccessItem struct {
	JobName  string `json:"JobName"`
	JobRunId string `json:"JobRunId"`
}

func handleBatchStopJobRun(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req batchStopJobRunRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	successful := make([]batchStopJobRunSuccessItem, 0)
	for _, id := range req.JobRunIds {
		if store.StopJobRun(req.JobName, id) {
			successful = append(successful, batchStopJobRunSuccessItem{JobName: req.JobName, JobRunId: id})
		}
	}
	return jsonOK(map[string]any{"SuccessfulSubmissions": successful, "Errors": []any{}})
}

// ---- Connection handlers ----

type physicalConnReqJSON struct {
	SubnetId            string   `json:"SubnetId"`
	SecurityGroupIdList []string `json:"SecurityGroupIdList"`
	AvailabilityZone    string   `json:"AvailabilityZone"`
}

type connectionInputJSON struct {
	Name                           string            `json:"Name"`
	Description                    string            `json:"Description"`
	ConnectionType                 string            `json:"ConnectionType"`
	ConnectionProperties           map[string]string `json:"ConnectionProperties"`
	PhysicalConnectionRequirements *physicalConnReqJSON `json:"PhysicalConnectionRequirements"`
}

type createConnectionRequest struct {
	ConnectionInput connectionInputJSON `json:"ConnectionInput"`
}

type connectionJSON struct {
	Name                           string            `json:"Name"`
	Description                    string            `json:"Description,omitempty"`
	ConnectionType                 string            `json:"ConnectionType"`
	ConnectionProperties           map[string]string `json:"ConnectionProperties,omitempty"`
	PhysicalConnectionRequirements *physicalConnReqJSON `json:"PhysicalConnectionRequirements,omitempty"`
	CreationTime                   float64           `json:"CreationTime"`
}

func toConnectionJSON(c *Connection) connectionJSON {
	j := connectionJSON{
		Name: c.Name, Description: c.Description, ConnectionType: c.ConnectionType,
		ConnectionProperties: c.ConnectionProperties, CreationTime: float64(c.CreateTime.Unix()),
	}
	if c.PhysicalConnectionRequirements != nil {
		j.PhysicalConnectionRequirements = &physicalConnReqJSON{
			SubnetId:            c.PhysicalConnectionRequirements.SubnetID,
			SecurityGroupIdList: c.PhysicalConnectionRequirements.SecurityGroupIDList,
			AvailabilityZone:    c.PhysicalConnectionRequirements.AvailabilityZone,
		}
	}
	return j
}

func handleCreateConnection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createConnectionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ConnectionInput.Name == "" {
		return jsonErr(service.ErrValidation("Connection name is required."))
	}
	conn := &Connection{
		Name:               req.ConnectionInput.Name,
		Description:        req.ConnectionInput.Description,
		ConnectionType:     req.ConnectionInput.ConnectionType,
		ConnectionProperties: req.ConnectionInput.ConnectionProperties,
	}
	if req.ConnectionInput.PhysicalConnectionRequirements != nil {
		conn.PhysicalConnectionRequirements = &PhysicalConnectionRequirements{
			SubnetID:            req.ConnectionInput.PhysicalConnectionRequirements.SubnetId,
			SecurityGroupIDList: req.ConnectionInput.PhysicalConnectionRequirements.SecurityGroupIdList,
			AvailabilityZone:    req.ConnectionInput.PhysicalConnectionRequirements.AvailabilityZone,
		}
	}
	if !store.CreateConnection(conn) {
		return jsonErr(service.ErrAlreadyExists("Connection", req.ConnectionInput.Name))
	}
	return jsonOK(struct{}{})
}

type getConnectionRequest struct {
	Name string `json:"Name"`
}

func handleGetConnection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getConnectionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	c, ok := store.GetConnection(req.Name)
	if !ok {
		return jsonErr(service.NewAWSError("EntityNotFoundException", "Connection "+req.Name+" not found.", http.StatusNotFound))
	}
	return jsonOK(map[string]any{"Connection": toConnectionJSON(c)})
}

func handleGetConnections(_ *service.RequestContext, store *Store) (*service.Response, error) {
	conns := store.ListConnections()
	list := make([]connectionJSON, 0, len(conns))
	for _, c := range conns {
		list = append(list, toConnectionJSON(c))
	}
	return jsonOK(map[string]any{"ConnectionList": list})
}

type deleteConnectionRequest struct {
	ConnectionName string `json:"ConnectionName"`
}

func handleDeleteConnection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteConnectionRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if !store.DeleteConnection(req.ConnectionName) {
		return jsonErr(service.NewAWSError("EntityNotFoundException", "Connection "+req.ConnectionName+" not found.", http.StatusNotFound))
	}
	return jsonOK(struct{}{})
}

// ---- Tag handlers ----

type tagResourceRequest struct {
	ResourceArn string            `json:"ResourceArn"`
	TagsToAdd   map[string]string `json:"TagsToAdd"`
}

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req tagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	store.TagResource(req.ResourceArn, req.TagsToAdd)
	return jsonOK(struct{}{})
}

type untagResourceRequest struct {
	ResourceArn  string   `json:"ResourceArn"`
	TagsToRemove []string `json:"TagsToRemove"`
}

func handleUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req untagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	store.UntagResource(req.ResourceArn, req.TagsToRemove)
	return jsonOK(struct{}{})
}

type getTagsRequest struct {
	ResourceArn string `json:"ResourceArn"`
}

func handleGetTags(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getTagsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	tags := store.GetTags(req.ResourceArn)
	return jsonOK(map[string]any{"Tags": tags})
}
