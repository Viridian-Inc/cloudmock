package kinesisanalytics

import (
	gojson "github.com/goccy/go-json"
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
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
	if err := gojson.Unmarshal(body, v); err != nil {
		return service.NewAWSError("InvalidArgumentException", "Invalid JSON in request body.", http.StatusBadRequest)
	}
	return nil
}

type tag struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

// ---- Application JSON types ----

type applicationDetailJSON struct {
	ApplicationName      string  `json:"ApplicationName"`
	ApplicationARN       string  `json:"ApplicationARN"`
	ApplicationStatus    string  `json:"ApplicationStatus"`
	ApplicationDescription string `json:"ApplicationDescription,omitempty"`
	RuntimeEnvironment   string  `json:"RuntimeEnvironment,omitempty"`
	ServiceExecutionRole string  `json:"ServiceExecutionRole,omitempty"`
	CreateTimestamp      float64 `json:"CreateTimestamp"`
	LastUpdateTimestamp  float64 `json:"LastUpdateTimestamp"`
	ApplicationVersionId int64   `json:"ApplicationVersionId"`
}

func toAppDetailJSON(app *Application) applicationDetailJSON {
	return applicationDetailJSON{
		ApplicationName:        app.Name,
		ApplicationARN:         app.ARN,
		ApplicationStatus:      app.Status,
		ApplicationDescription: app.Description,
		RuntimeEnvironment:     app.RuntimeEnvironment,
		ServiceExecutionRole:   app.ServiceExecutionRole,
		CreateTimestamp:        float64(app.CreateTimestamp.Unix()),
		LastUpdateTimestamp:    float64(app.LastUpdateTimestamp.Unix()),
		ApplicationVersionId:   app.ApplicationVersionId,
	}
}

// ---- CreateApplication ----

type createApplicationRequest struct {
	ApplicationName        string `json:"ApplicationName"`
	ApplicationDescription string `json:"ApplicationDescription"`
	RuntimeEnvironment     string `json:"RuntimeEnvironment"`
	ServiceExecutionRole   string `json:"ServiceExecutionRole"`
	Tags                   []tag  `json:"Tags"`
}

func handleCreateApplication(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createApplicationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ApplicationName == "" {
		return jsonErr(service.ErrValidation("ApplicationName is required."))
	}
	tags := make(map[string]string)
	for _, t := range req.Tags {
		tags[t.Key] = t.Value
	}
	app, ok := store.CreateApplication(req.ApplicationName, req.ApplicationDescription, req.RuntimeEnvironment, req.ServiceExecutionRole, tags)
	if !ok {
		return jsonErr(service.ErrAlreadyExists("Application", req.ApplicationName))
	}
	return jsonOK(map[string]any{"ApplicationDetail": toAppDetailJSON(app)})
}

// ---- DescribeApplication ----

type describeApplicationRequest struct {
	ApplicationName string `json:"ApplicationName"`
}

func handleDescribeApplication(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeApplicationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	app, ok := store.GetApplication(req.ApplicationName)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Application "+req.ApplicationName+" not found.", http.StatusNotFound))
	}
	return jsonOK(map[string]any{"ApplicationDetail": toAppDetailJSON(app)})
}

// ---- ListApplications ----

type applicationSummary struct {
	ApplicationName   string `json:"ApplicationName"`
	ApplicationARN    string `json:"ApplicationARN"`
	ApplicationStatus string `json:"ApplicationStatus"`
}

func handleListApplications(_ *service.RequestContext, store *Store) (*service.Response, error) {
	apps := store.ListApplications()
	summaries := make([]applicationSummary, 0, len(apps))
	for _, app := range apps {
		summaries = append(summaries, applicationSummary{
			ApplicationName: app.Name, ApplicationARN: app.ARN, ApplicationStatus: app.Status,
		})
	}
	return jsonOK(map[string]any{"ApplicationSummaries": summaries})
}

// ---- DeleteApplication ----

type deleteApplicationRequest struct {
	ApplicationName string `json:"ApplicationName"`
	CreateTimestamp float64 `json:"CreateTimestamp"`
}

func handleDeleteApplication(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteApplicationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if !store.DeleteApplication(req.ApplicationName) {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Application "+req.ApplicationName+" not found.", http.StatusNotFound))
	}
	return jsonOK(struct{}{})
}

// ---- UpdateApplication ----

type updateApplicationRequest struct {
	ApplicationName        string `json:"ApplicationName"`
	ApplicationDescription string `json:"ApplicationDescription"`
	CurrentApplicationVersionId int64 `json:"CurrentApplicationVersionId"`
}

func handleUpdateApplication(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req updateApplicationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	app, ok := store.UpdateApplication(req.ApplicationName, req.ApplicationDescription)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Application "+req.ApplicationName+" not found.", http.StatusNotFound))
	}
	return jsonOK(map[string]any{"ApplicationDetail": toAppDetailJSON(app)})
}

// ---- StartApplication / StopApplication ----

type startApplicationRequest struct {
	ApplicationName string `json:"ApplicationName"`
}

func handleStartApplication(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req startApplicationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if !store.StartApplication(req.ApplicationName) {
		return jsonErr(service.NewAWSError("ResourceInUseException", "Application is not in READY state.", http.StatusBadRequest))
	}
	return jsonOK(struct{}{})
}

type stopApplicationRequest struct {
	ApplicationName string `json:"ApplicationName"`
}

func handleStopApplication(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req stopApplicationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if !store.StopApplication(req.ApplicationName) {
		return jsonErr(service.NewAWSError("ResourceInUseException", "Application is not in RUNNING state.", http.StatusBadRequest))
	}
	return jsonOK(struct{}{})
}

// ---- AddApplicationInput ----

type addApplicationInputRequest struct {
	ApplicationName             string `json:"ApplicationName"`
	CurrentApplicationVersionId int64  `json:"CurrentApplicationVersionId"`
	Input                       struct {
		NamePrefix         string `json:"NamePrefix"`
		InputSchema        struct {
			RecordFormat struct {
				RecordFormatType string `json:"RecordFormatType"`
			} `json:"RecordFormat"`
			RecordColumns []struct {
				Name    string `json:"Name"`
				SqlType string `json:"SqlType"`
				Mapping string `json:"Mapping"`
			} `json:"RecordColumns"`
		} `json:"InputSchema"`
		KinesisStreamsInput *struct {
			ResourceARN string `json:"ResourceARN"`
		} `json:"KinesisStreamsInput"`
	} `json:"Input"`
}

func handleAddApplicationInput(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req addApplicationInputRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	input := Input{NamePrefix: req.Input.NamePrefix}
	input.InputSchema.RecordFormat.RecordFormatType = req.Input.InputSchema.RecordFormat.RecordFormatType
	for _, col := range req.Input.InputSchema.RecordColumns {
		input.InputSchema.RecordColumns = append(input.InputSchema.RecordColumns, RecordColumn{Name: col.Name, SqlType: col.SqlType, Mapping: col.Mapping})
	}
	if req.Input.KinesisStreamsInput != nil {
		input.KinesisStreamsInput = &KinesisStreamsInput{ResourceARN: req.Input.KinesisStreamsInput.ResourceARN}
	}
	_, ok := store.AddInput(req.ApplicationName, input)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Application not found.", http.StatusNotFound))
	}
	return jsonOK(struct{}{})
}

// ---- AddApplicationOutput ----

type addApplicationOutputRequest struct {
	ApplicationName             string `json:"ApplicationName"`
	CurrentApplicationVersionId int64  `json:"CurrentApplicationVersionId"`
	Output                      struct {
		Name              string `json:"Name"`
		DestinationSchema struct {
			RecordFormatType string `json:"RecordFormatType"`
		} `json:"DestinationSchema"`
		KinesisStreamsOutput *struct {
			ResourceARN string `json:"ResourceARN"`
		} `json:"KinesisStreamsOutput"`
	} `json:"Output"`
}

func handleAddApplicationOutput(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req addApplicationOutputRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	output := Output{Name: req.Output.Name}
	output.DestinationSchema.RecordFormatType = req.Output.DestinationSchema.RecordFormatType
	if req.Output.KinesisStreamsOutput != nil {
		output.KinesisStreamsOutput = &KinesisStreamsOutput{ResourceARN: req.Output.KinesisStreamsOutput.ResourceARN}
	}
	_, ok := store.AddOutput(req.ApplicationName, output)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Application not found.", http.StatusNotFound))
	}
	return jsonOK(struct{}{})
}

// ---- DeleteApplicationOutput ----

type deleteApplicationOutputRequest struct {
	ApplicationName             string `json:"ApplicationName"`
	CurrentApplicationVersionId int64  `json:"CurrentApplicationVersionId"`
	OutputId                    string `json:"OutputId"`
}

func handleDeleteApplicationOutput(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteApplicationOutputRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if !store.DeleteOutput(req.ApplicationName, req.OutputId) {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Output not found.", http.StatusNotFound))
	}
	return jsonOK(struct{}{})
}

// ---- Snapshot handlers ----

type createSnapshotRequest struct {
	ApplicationName string `json:"ApplicationName"`
	SnapshotName    string `json:"SnapshotName"`
}

func handleCreateApplicationSnapshot(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createSnapshotRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	_, ok := store.CreateSnapshot(req.ApplicationName, req.SnapshotName)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Application not found or snapshot already exists.", http.StatusBadRequest))
	}
	return jsonOK(struct{}{})
}

type listSnapshotsRequest struct {
	ApplicationName string `json:"ApplicationName"`
}

type snapshotDetailJSON struct {
	SnapshotName              string  `json:"SnapshotName"`
	SnapshotStatus            string  `json:"SnapshotStatus"`
	ApplicationVersionId      int64   `json:"ApplicationVersionId"`
	SnapshotCreationTimestamp float64 `json:"SnapshotCreationTimestamp"`
}

func handleListApplicationSnapshots(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listSnapshotsRequest
	parseJSON(ctx.Body, &req)
	snaps := store.ListSnapshots(req.ApplicationName)
	details := make([]snapshotDetailJSON, 0, len(snaps))
	for _, s := range snaps {
		details = append(details, snapshotDetailJSON{
			SnapshotName: s.SnapshotName, SnapshotStatus: s.SnapshotStatus,
			ApplicationVersionId: s.ApplicationVersionId,
			SnapshotCreationTimestamp: float64(s.SnapshotCreationTimestamp.Unix()),
		})
	}
	return jsonOK(map[string]any{"SnapshotSummaries": details})
}

type deleteSnapshotRequest struct {
	ApplicationName string `json:"ApplicationName"`
	SnapshotName    string `json:"SnapshotName"`
}

func handleDeleteApplicationSnapshot(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteSnapshotRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if !store.DeleteSnapshot(req.ApplicationName, req.SnapshotName) {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Snapshot not found.", http.StatusNotFound))
	}
	return jsonOK(struct{}{})
}

// ---- Tag handlers ----

type tagResourceRequest struct {
	ResourceARN string `json:"ResourceARN"`
	Tags        []tag  `json:"Tags"`
}

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req tagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	tags := make(map[string]string)
	for _, t := range req.Tags {
		tags[t.Key] = t.Value
	}
	if !store.TagResource(req.ResourceARN, tags) {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Resource not found.", http.StatusNotFound))
	}
	return jsonOK(struct{}{})
}

type untagResourceRequest struct {
	ResourceARN string   `json:"ResourceARN"`
	TagKeys     []string `json:"TagKeys"`
}

func handleUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req untagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if !store.UntagResource(req.ResourceARN, req.TagKeys) {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Resource not found.", http.StatusNotFound))
	}
	return jsonOK(struct{}{})
}

type listTagsForResourceRequest struct {
	ResourceARN string `json:"ResourceARN"`
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listTagsForResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	tags, ok := store.ListTagsForResource(req.ResourceARN)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Resource not found.", http.StatusNotFound))
	}
	tagList := make([]tag, 0, len(tags))
	for k, v := range tags {
		tagList = append(tagList, tag{Key: k, Value: v})
	}
	return jsonOK(map[string]any{"Tags": tagList})
}
