package cloudwatchlogs

import (
	"encoding/json"
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ---- JSON request/response types ----

type createLogGroupRequest struct {
	LogGroupName string            `json:"logGroupName"`
	Tags         map[string]string `json:"tags"`
}

type deleteLogGroupRequest struct {
	LogGroupName string `json:"logGroupName"`
}

type describeLogGroupsRequest struct {
	LogGroupNamePrefix string `json:"logGroupNamePrefix"`
}

type logGroupEntry struct {
	LogGroupName    string `json:"logGroupName"`
	Arn             string `json:"arn"`
	CreationTime    int64  `json:"creationTime"`
	StoredBytes     int64  `json:"storedBytes"`
	RetentionInDays int    `json:"retentionInDays,omitempty"`
}

type describeLogGroupsResponse struct {
	LogGroups []logGroupEntry `json:"logGroups"`
}

type createLogStreamRequest struct {
	LogGroupName  string `json:"logGroupName"`
	LogStreamName string `json:"logStreamName"`
}

type deleteLogStreamRequest struct {
	LogGroupName  string `json:"logGroupName"`
	LogStreamName string `json:"logStreamName"`
}

type describeLogStreamsRequest struct {
	LogGroupName        string `json:"logGroupName"`
	LogStreamNamePrefix string `json:"logStreamNamePrefix"`
}

type logStreamEntry struct {
	LogStreamName       string `json:"logStreamName"`
	Arn                 string `json:"arn"`
	CreationTime        int64  `json:"creationTime"`
	FirstEventTimestamp int64  `json:"firstEventTimestamp,omitempty"`
	LastEventTimestamp  int64  `json:"lastEventTimestamp,omitempty"`
	LastIngestionTime   int64  `json:"lastIngestionTime,omitempty"`
	UploadSequenceToken string `json:"uploadSequenceToken"`
}

type describeLogStreamsResponse struct {
	LogStreams []logStreamEntry `json:"logStreams"`
}

type inputLogEvent struct {
	Timestamp int64  `json:"timestamp"`
	Message   string `json:"message"`
}

type putLogEventsRequest struct {
	LogGroupName  string          `json:"logGroupName"`
	LogStreamName string          `json:"logStreamName"`
	LogEvents     []inputLogEvent `json:"logEvents"`
}

type putLogEventsResponse struct {
	NextSequenceToken string `json:"nextSequenceToken"`
}

type getLogEventsRequest struct {
	LogGroupName  string `json:"logGroupName"`
	LogStreamName string `json:"logStreamName"`
	StartTime     int64  `json:"startTime"`
	EndTime       int64  `json:"endTime"`
	Limit         int    `json:"limit"`
}

type outputLogEvent struct {
	Timestamp     int64  `json:"timestamp"`
	Message       string `json:"message"`
	IngestionTime int64  `json:"ingestionTime"`
}

type getLogEventsResponse struct {
	Events            []outputLogEvent `json:"events"`
	NextForwardToken  string           `json:"nextForwardToken"`
	NextBackwardToken string           `json:"nextBackwardToken"`
}

type filterLogEventsRequest struct {
	LogGroupName   string   `json:"logGroupName"`
	FilterPattern  string   `json:"filterPattern"`
	StartTime      int64    `json:"startTime"`
	EndTime        int64    `json:"endTime"`
	LogStreamNames []string `json:"logStreamNames"`
}

type filteredLogEvent struct {
	Timestamp     int64  `json:"timestamp"`
	Message       string `json:"message"`
	IngestionTime int64  `json:"ingestionTime"`
}

type filterLogEventsResponse struct {
	Events []filteredLogEvent `json:"events"`
}

type putRetentionPolicyRequest struct {
	LogGroupName    string `json:"logGroupName"`
	RetentionInDays int    `json:"retentionInDays"`
}

type deleteRetentionPolicyRequest struct {
	LogGroupName string `json:"logGroupName"`
}

type tagLogGroupRequest struct {
	LogGroupName string            `json:"logGroupName"`
	Tags         map[string]string `json:"tags"`
}

type untagLogGroupRequest struct {
	LogGroupName string   `json:"logGroupName"`
	Tags         []string `json:"tags"`
}

type listTagsLogGroupRequest struct {
	LogGroupName string `json:"logGroupName"`
}

type listTagsLogGroupResponse struct {
	Tags map[string]string `json:"tags"`
}

// ---- helpers ----

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       body,
		Format:     service.FormatJSON,
	}, nil
}

func jsonEmpty() (*service.Response, error) {
	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       struct{}{},
		Format:     service.FormatJSON,
	}, nil
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

// ---- handlers ----

func handleCreateLogGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createLogGroupRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.LogGroupName == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"logGroupName is required.", http.StatusBadRequest))
	}
	if awsErr := store.CreateLogGroup(req.LogGroupName); awsErr != nil {
		return jsonErr(awsErr)
	}
	// Apply tags if provided at creation time.
	if len(req.Tags) > 0 {
		if awsErr := store.TagLogGroup(req.LogGroupName, req.Tags); awsErr != nil {
			return jsonErr(awsErr)
		}
	}
	return jsonEmpty()
}

func handleDeleteLogGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteLogGroupRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.LogGroupName == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"logGroupName is required.", http.StatusBadRequest))
	}
	if awsErr := store.DeleteLogGroup(req.LogGroupName); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonEmpty()
}

func handleDescribeLogGroups(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeLogGroupsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	groups := store.DescribeLogGroups(req.LogGroupNamePrefix)
	entries := make([]logGroupEntry, 0, len(groups))
	for _, g := range groups {
		entries = append(entries, logGroupEntry{
			LogGroupName:    g.Name,
			Arn:             g.ARN,
			CreationTime:    g.CreationTime,
			StoredBytes:     g.StoredBytes,
			RetentionInDays: g.RetentionDays,
		})
	}
	return jsonOK(describeLogGroupsResponse{LogGroups: entries})
}

func handleCreateLogStream(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createLogStreamRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.LogGroupName == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"logGroupName is required.", http.StatusBadRequest))
	}
	if req.LogStreamName == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"logStreamName is required.", http.StatusBadRequest))
	}
	if awsErr := store.CreateLogStream(req.LogGroupName, req.LogStreamName); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonEmpty()
}

func handleDeleteLogStream(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteLogStreamRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.LogGroupName == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"logGroupName is required.", http.StatusBadRequest))
	}
	if req.LogStreamName == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"logStreamName is required.", http.StatusBadRequest))
	}
	if awsErr := store.DeleteLogStream(req.LogGroupName, req.LogStreamName); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonEmpty()
}

func handleDescribeLogStreams(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeLogStreamsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.LogGroupName == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"logGroupName is required.", http.StatusBadRequest))
	}
	streams, awsErr := store.DescribeLogStreams(req.LogGroupName, req.LogStreamNamePrefix)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	entries := make([]logStreamEntry, 0, len(streams))
	for _, st := range streams {
		entries = append(entries, logStreamEntry{
			LogStreamName:       st.Name,
			Arn:                 st.ARN,
			CreationTime:        st.CreationTime,
			FirstEventTimestamp: st.FirstEventTimestamp,
			LastEventTimestamp:  st.LastEventTimestamp,
			LastIngestionTime:   st.LastIngestionTime,
			UploadSequenceToken: st.UploadSequenceToken,
		})
	}
	return jsonOK(describeLogStreamsResponse{LogStreams: entries})
}

func handlePutLogEvents(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req putLogEventsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.LogGroupName == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"logGroupName is required.", http.StatusBadRequest))
	}
	if req.LogStreamName == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"logStreamName is required.", http.StatusBadRequest))
	}

	events := make([]LogEvent, 0, len(req.LogEvents))
	for _, e := range req.LogEvents {
		events = append(events, LogEvent{
			Timestamp: e.Timestamp,
			Message:   e.Message,
		})
	}

	nextToken, awsErr := store.PutLogEvents(req.LogGroupName, req.LogStreamName, events)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(putLogEventsResponse{NextSequenceToken: nextToken})
}

func handleGetLogEvents(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getLogEventsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.LogGroupName == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"logGroupName is required.", http.StatusBadRequest))
	}
	if req.LogStreamName == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"logStreamName is required.", http.StatusBadRequest))
	}

	events, fwdToken, bwdToken, awsErr := store.GetLogEvents(
		req.LogGroupName, req.LogStreamName,
		req.StartTime, req.EndTime, req.Limit,
	)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	out := make([]outputLogEvent, 0, len(events))
	for _, ev := range events {
		out = append(out, outputLogEvent{
			Timestamp:     ev.Timestamp,
			Message:       ev.Message,
			IngestionTime: ev.IngestionTime,
		})
	}
	return jsonOK(getLogEventsResponse{
		Events:            out,
		NextForwardToken:  fwdToken,
		NextBackwardToken: bwdToken,
	})
}

func handleFilterLogEvents(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req filterLogEventsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.LogGroupName == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"logGroupName is required.", http.StatusBadRequest))
	}

	events, awsErr := store.FilterLogEvents(
		req.LogGroupName, req.LogStreamNames,
		req.FilterPattern, req.StartTime, req.EndTime,
	)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	out := make([]filteredLogEvent, 0, len(events))
	for _, ev := range events {
		out = append(out, filteredLogEvent{
			Timestamp:     ev.Timestamp,
			Message:       ev.Message,
			IngestionTime: ev.IngestionTime,
		})
	}
	return jsonOK(filterLogEventsResponse{Events: out})
}

func handlePutRetentionPolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req putRetentionPolicyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.LogGroupName == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"logGroupName is required.", http.StatusBadRequest))
	}
	if req.RetentionInDays <= 0 {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"retentionInDays must be a positive integer.", http.StatusBadRequest))
	}
	if awsErr := store.PutRetentionPolicy(req.LogGroupName, req.RetentionInDays); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonEmpty()
}

func handleDeleteRetentionPolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteRetentionPolicyRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.LogGroupName == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"logGroupName is required.", http.StatusBadRequest))
	}
	if awsErr := store.DeleteRetentionPolicy(req.LogGroupName); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonEmpty()
}

func handleTagLogGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req tagLogGroupRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.LogGroupName == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"logGroupName is required.", http.StatusBadRequest))
	}
	if awsErr := store.TagLogGroup(req.LogGroupName, req.Tags); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonEmpty()
}

func handleUntagLogGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req untagLogGroupRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.LogGroupName == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"logGroupName is required.", http.StatusBadRequest))
	}
	if awsErr := store.UntagLogGroup(req.LogGroupName, req.Tags); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonEmpty()
}

func handleListTagsLogGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listTagsLogGroupRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.LogGroupName == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"logGroupName is required.", http.StatusBadRequest))
	}
	tags, awsErr := store.ListTagsLogGroup(req.LogGroupName)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(listTagsLogGroupResponse{Tags: tags})
}
