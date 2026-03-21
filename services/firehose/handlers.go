package firehose

import (
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// ---- helpers ----

func jsonOK(body interface{}) (*service.Response, error) {
	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       body,
		Format:     service.FormatJSON,
	}, nil
}

func jsonErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
}

func emptyOK() (*service.Response, error) {
	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       struct{}{},
		Format:     service.FormatJSON,
	}, nil
}

func parseJSON(body []byte, v interface{}) *service.AWSError {
	if len(body) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, v); err != nil {
		return service.NewAWSError("InvalidParameterException",
			"Request body is not valid JSON.", http.StatusBadRequest)
	}
	return nil
}

// ---- request / response types ----

type s3ConfigJSON struct {
	BucketARN      string             `json:"BucketARN"`
	RoleARN        string             `json:"RoleARN"`
	Prefix         string             `json:"Prefix"`
	BufferingHints bufferingHintsJSON `json:"BufferingHints"`
}

type bufferingHintsJSON struct {
	IntervalInSeconds int `json:"IntervalInSeconds"`
	SizeInMBs         int `json:"SizeInMBs"`
}

type createDeliveryStreamRequest struct {
	DeliveryStreamName          string       `json:"DeliveryStreamName"`
	DeliveryStreamType          string       `json:"DeliveryStreamType"`
	S3DestinationConfiguration  s3ConfigJSON `json:"S3DestinationConfiguration"`
}

type createDeliveryStreamResponse struct {
	DeliveryStreamARN string `json:"DeliveryStreamARN"`
}

type deleteDeliveryStreamRequest struct {
	DeliveryStreamName string `json:"DeliveryStreamName"`
}

type describeDeliveryStreamRequest struct {
	DeliveryStreamName string `json:"DeliveryStreamName"`
}

type destinationJSON struct {
	DestinationId              string       `json:"DestinationId"`
	S3DestinationDescription   s3DescJSON   `json:"S3DestinationDescription"`
}

type s3DescJSON struct {
	BucketARN      string             `json:"BucketARN"`
	RoleARN        string             `json:"RoleARN"`
	Prefix         string             `json:"Prefix"`
	BufferingHints bufferingHintsJSON `json:"BufferingHints"`
}

type deliveryStreamDescriptionJSON struct {
	DeliveryStreamName   string            `json:"DeliveryStreamName"`
	DeliveryStreamARN    string            `json:"DeliveryStreamARN"`
	DeliveryStreamStatus string            `json:"DeliveryStreamStatus"`
	DeliveryStreamType   string            `json:"DeliveryStreamType"`
	Destinations         []destinationJSON `json:"Destinations"`
}

type describeDeliveryStreamResponse struct {
	DeliveryStreamDescription deliveryStreamDescriptionJSON `json:"DeliveryStreamDescription"`
}

type listDeliveryStreamsResponse struct {
	DeliveryStreamNames    []string `json:"DeliveryStreamNames"`
	HasMoreDeliveryStreams bool     `json:"HasMoreDeliveryStreams"`
}

type recordJSON struct {
	Data string `json:"Data"` // base64-encoded
}

type putRecordRequest struct {
	DeliveryStreamName string     `json:"DeliveryStreamName"`
	Record             recordJSON `json:"Record"`
}

type putRecordResponse struct {
	RecordId string `json:"RecordId"`
}

type putRecordBatchRequest struct {
	DeliveryStreamName string       `json:"DeliveryStreamName"`
	Records            []recordJSON `json:"Records"`
}

type requestResponseJSON struct {
	RecordId string `json:"RecordId"`
}

type putRecordBatchResponse struct {
	RequestResponses []requestResponseJSON `json:"RequestResponses"`
	FailedPutCount   int                  `json:"FailedPutCount"`
}

type updateDestinationRequest struct {
	DeliveryStreamName    string       `json:"DeliveryStreamName"`
	DestinationId         string       `json:"DestinationId"`
	S3DestinationUpdate   s3ConfigJSON `json:"S3DestinationUpdate"`
}

type tagDeliveryStreamRequest struct {
	DeliveryStreamName string            `json:"DeliveryStreamName"`
	Tags               []tagEntryJSON    `json:"Tags"`
}

type tagEntryJSON struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

type untagDeliveryStreamRequest struct {
	DeliveryStreamName string   `json:"DeliveryStreamName"`
	TagKeys            []string `json:"TagKeys"`
}

type listTagsForDeliveryStreamRequest struct {
	DeliveryStreamName string `json:"DeliveryStreamName"`
}

type listTagsForDeliveryStreamResponse struct {
	Tags        []tagEntryJSON `json:"Tags"`
	HasMoreTags bool           `json:"HasMoreTags"`
}

// ---- handler functions ----

func handleCreateDeliveryStream(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createDeliveryStreamRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.DeliveryStreamName == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"DeliveryStreamName is required.", http.StatusBadRequest))
	}

	dest := Destination{
		S3BucketARN: req.S3DestinationConfiguration.BucketARN,
		S3Prefix:    req.S3DestinationConfiguration.Prefix,
		RoleARN:     req.S3DestinationConfiguration.RoleARN,
		BufferingHints: BufferingHints{
			IntervalInSeconds: req.S3DestinationConfiguration.BufferingHints.IntervalInSeconds,
			SizeInMBs:         req.S3DestinationConfiguration.BufferingHints.SizeInMBs,
		},
	}

	arn, awsErr := store.CreateDeliveryStream(req.DeliveryStreamName, req.DeliveryStreamType, dest)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(createDeliveryStreamResponse{DeliveryStreamARN: arn})
}

func handleDeleteDeliveryStream(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteDeliveryStreamRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.DeliveryStreamName == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"DeliveryStreamName is required.", http.StatusBadRequest))
	}

	if awsErr := store.DeleteDeliveryStream(req.DeliveryStreamName); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleDescribeDeliveryStream(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeDeliveryStreamRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.DeliveryStreamName == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"DeliveryStreamName is required.", http.StatusBadRequest))
	}

	st, awsErr := store.GetDeliveryStream(req.DeliveryStreamName)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	dests := make([]destinationJSON, len(st.Destinations))
	for i, d := range st.Destinations {
		dests[i] = destinationJSON{
			DestinationId: d.DestinationId,
			S3DestinationDescription: s3DescJSON{
				BucketARN: d.S3BucketARN,
				RoleARN:   d.RoleARN,
				Prefix:    d.S3Prefix,
				BufferingHints: bufferingHintsJSON{
					IntervalInSeconds: d.BufferingHints.IntervalInSeconds,
					SizeInMBs:         d.BufferingHints.SizeInMBs,
				},
			},
		}
	}

	desc := deliveryStreamDescriptionJSON{
		DeliveryStreamName:   st.Name,
		DeliveryStreamARN:    st.ARN,
		DeliveryStreamStatus: string(st.Status),
		DeliveryStreamType:   st.Type,
		Destinations:         dests,
	}

	return jsonOK(describeDeliveryStreamResponse{DeliveryStreamDescription: desc})
}

func handleListDeliveryStreams(_ *service.RequestContext, store *Store) (*service.Response, error) {
	names := store.ListDeliveryStreams()
	if names == nil {
		names = []string{}
	}
	return jsonOK(listDeliveryStreamsResponse{
		DeliveryStreamNames:    names,
		HasMoreDeliveryStreams: false,
	})
}

func handlePutRecord(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req putRecordRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.DeliveryStreamName == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"DeliveryStreamName is required.", http.StatusBadRequest))
	}
	if req.Record.Data == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"Record.Data is required.", http.StatusBadRequest))
	}

	data, err := base64.StdEncoding.DecodeString(req.Record.Data)
	if err != nil {
		return jsonErr(service.NewAWSError("InvalidArgumentException",
			"Record.Data must be base64-encoded.", http.StatusBadRequest))
	}

	recordId, awsErr := store.PutRecord(req.DeliveryStreamName, data)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(putRecordResponse{RecordId: recordId})
}

func handlePutRecordBatch(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req putRecordBatchRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.DeliveryStreamName == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"DeliveryStreamName is required.", http.StatusBadRequest))
	}
	if len(req.Records) == 0 {
		return jsonErr(service.NewAWSError("ValidationException",
			"Records must not be empty.", http.StatusBadRequest))
	}

	// Decode all records first; track which ones fail base64 decoding.
	decoded := make([][]byte, 0, len(req.Records))
	failedCount := 0
	failedIdx := make(map[int]bool)
	for i, r := range req.Records {
		d, err := base64.StdEncoding.DecodeString(r.Data)
		if err != nil {
			failedCount++
			failedIdx[i] = true
			decoded = append(decoded, nil)
			continue
		}
		decoded = append(decoded, d)
	}

	// Batch-put only the successfully decoded records.
	valid := make([][]byte, 0, len(decoded)-failedCount)
	validIdx := make([]int, 0, len(decoded)-failedCount)
	for i, d := range decoded {
		if !failedIdx[i] {
			valid = append(valid, d)
			validIdx = append(validIdx, i)
		}
	}

	var ids []string
	if len(valid) > 0 {
		var awsErr *service.AWSError
		ids, awsErr = store.PutRecordBatch(req.DeliveryStreamName, valid)
		if awsErr != nil {
			return jsonErr(awsErr)
		}
	}

	responses := make([]requestResponseJSON, len(req.Records))
	idCursor := 0
	for i := range req.Records {
		if failedIdx[i] {
			responses[i] = requestResponseJSON{}
		} else {
			responses[i] = requestResponseJSON{RecordId: ids[idCursor]}
			idCursor++
		}
	}

	return jsonOK(putRecordBatchResponse{
		RequestResponses: responses,
		FailedPutCount:   failedCount,
	})
}

func handleUpdateDestination(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req updateDestinationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.DeliveryStreamName == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"DeliveryStreamName is required.", http.StatusBadRequest))
	}
	if req.DestinationId == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"DestinationId is required.", http.StatusBadRequest))
	}

	update := Destination{
		S3BucketARN: req.S3DestinationUpdate.BucketARN,
		S3Prefix:    req.S3DestinationUpdate.Prefix,
		RoleARN:     req.S3DestinationUpdate.RoleARN,
		BufferingHints: BufferingHints{
			IntervalInSeconds: req.S3DestinationUpdate.BufferingHints.IntervalInSeconds,
			SizeInMBs:         req.S3DestinationUpdate.BufferingHints.SizeInMBs,
		},
	}

	if awsErr := store.UpdateDestination(req.DeliveryStreamName, req.DestinationId, update); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleTagDeliveryStream(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req tagDeliveryStreamRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.DeliveryStreamName == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"DeliveryStreamName is required.", http.StatusBadRequest))
	}

	tags := make(map[string]string, len(req.Tags))
	for _, t := range req.Tags {
		tags[t.Key] = t.Value
	}

	if awsErr := store.TagStream(req.DeliveryStreamName, tags); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleUntagDeliveryStream(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req untagDeliveryStreamRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.DeliveryStreamName == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"DeliveryStreamName is required.", http.StatusBadRequest))
	}

	if awsErr := store.UntagStream(req.DeliveryStreamName, req.TagKeys); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleListTagsForDeliveryStream(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listTagsForDeliveryStreamRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.DeliveryStreamName == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"DeliveryStreamName is required.", http.StatusBadRequest))
	}

	tags, awsErr := store.ListTags(req.DeliveryStreamName)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	tagList := make([]tagEntryJSON, 0, len(tags))
	for k, v := range tags {
		tagList = append(tagList, tagEntryJSON{Key: k, Value: v})
	}

	return jsonOK(listTagsForDeliveryStreamResponse{Tags: tagList, HasMoreTags: false})
}
