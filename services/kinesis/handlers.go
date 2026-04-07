package kinesis

import (
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ---- helpers ----

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       body,
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

func emptyOK() (*service.Response, error) {
	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       struct{}{},
		Format:     service.FormatJSON,
	}, nil
}

// ---- request / response types ----

type createStreamRequest struct {
	StreamName string `json:"StreamName"`
	ShardCount int    `json:"ShardCount"`
}

type deleteStreamRequest struct {
	StreamName string `json:"StreamName"`
}

type describeStreamRequest struct {
	StreamName string `json:"StreamName"`
}

type shardJSON struct {
	ShardId           string           `json:"ShardId"`
	HashKeyRange      hashKeyRangeJSON `json:"HashKeyRange"`
	SequenceNumberRange seqNumRangeJSON `json:"SequenceNumberRange"`
}

type hashKeyRangeJSON struct {
	StartingHashKey string `json:"StartingHashKey"`
	EndingHashKey   string `json:"EndingHashKey"`
}

type seqNumRangeJSON struct {
	StartingSequenceNumber string `json:"StartingSequenceNumber"`
}

type streamDescriptionJSON struct {
	StreamName           string      `json:"StreamName"`
	StreamARN            string      `json:"StreamARN"`
	StreamStatus         string      `json:"StreamStatus"`
	Shards               []shardJSON `json:"Shards"`
	HasMoreShards        bool        `json:"HasMoreShards"`
	RetentionPeriodHours int         `json:"RetentionPeriodHours"`
}

type describeStreamResponse struct {
	StreamDescription streamDescriptionJSON `json:"StreamDescription"`
}

type listStreamsResponse struct {
	StreamNames    []string `json:"StreamNames"`
	HasMoreStreams  bool     `json:"HasMoreStreams"`
}

type putRecordRequest struct {
	StreamName   string `json:"StreamName"`
	Data         string `json:"Data"` // base64-encoded
	PartitionKey string `json:"PartitionKey"`
}

type putRecordResponse struct {
	ShardId        string `json:"ShardId"`
	SequenceNumber string `json:"SequenceNumber"`
}

type putRecordsRequestRecord struct {
	Data         string `json:"Data"` // base64-encoded
	PartitionKey string `json:"PartitionKey"`
}

type putRecordsRequest struct {
	StreamName string                    `json:"StreamName"`
	Records    []putRecordsRequestRecord `json:"Records"`
}

type putRecordsResultRecord struct {
	ShardId        string `json:"ShardId"`
	SequenceNumber string `json:"SequenceNumber"`
}

type putRecordsResponse struct {
	Records           []putRecordsResultRecord `json:"Records"`
	FailedRecordCount int                      `json:"FailedRecordCount"`
}

type getShardIteratorRequest struct {
	StreamName             string `json:"StreamName"`
	ShardId                string `json:"ShardId"`
	ShardIteratorType      string `json:"ShardIteratorType"`
	StartingSequenceNumber string `json:"StartingSequenceNumber"`
}

type getShardIteratorResponse struct {
	ShardIterator string `json:"ShardIterator"`
}

type getRecordsRequest struct {
	ShardIterator string `json:"ShardIterator"`
	Limit         int    `json:"Limit"`
}

type getRecordsRecord struct {
	Data           string `json:"Data"` // base64-encoded
	PartitionKey   string `json:"PartitionKey"`
	SequenceNumber string `json:"SequenceNumber"`
}

type getRecordsResponse struct {
	Records            []getRecordsRecord `json:"Records"`
	NextShardIterator  string             `json:"NextShardIterator"`
	MillisBehindLatest int64              `json:"MillisBehindLatest"`
}

type retentionPeriodRequest struct {
	StreamName                  string `json:"StreamName"`
	RetentionPeriodHours        int    `json:"RetentionPeriodHours"`
}

type addTagsRequest struct {
	StreamName string            `json:"StreamName"`
	Tags       map[string]string `json:"Tags"`
}

type removeTagsRequest struct {
	StreamName string   `json:"StreamName"`
	TagKeys    []string `json:"TagKeys"`
}

type listTagsRequest struct {
	StreamName string `json:"StreamName"`
}

type tagJSON struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

type listTagsResponse struct {
	Tags        []tagJSON `json:"Tags"`
	HasMoreTags bool      `json:"HasMoreTags"`
}

// ---- handler functions ----

func handleCreateStream(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createStreamRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.StreamName == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"StreamName is required.", http.StatusBadRequest))
	}
	if req.ShardCount == 0 {
		req.ShardCount = 1
	}

	if awsErr := store.CreateStream(req.StreamName, req.ShardCount); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleDeleteStream(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteStreamRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.StreamName == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"StreamName is required.", http.StatusBadRequest))
	}

	if awsErr := store.DeleteStream(req.StreamName); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleDescribeStream(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeStreamRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.StreamName == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"StreamName is required.", http.StatusBadRequest))
	}

	st, awsErr := store.GetStream(req.StreamName)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	shards := make([]shardJSON, len(st.Shards))
	for i, sh := range st.Shards {
		startSeq := ""
		if len(sh.Records) > 0 {
			startSeq = sh.Records[0].SequenceNumber
		}
		shards[i] = shardJSON{
			ShardId: sh.ShardId,
			HashKeyRange: hashKeyRangeJSON{
				StartingHashKey: sh.HashKeyRange.StartingHashKey,
				EndingHashKey:   sh.HashKeyRange.EndingHashKey,
			},
			SequenceNumberRange: seqNumRangeJSON{
				StartingSequenceNumber: startSeq,
			},
		}
	}

	desc := streamDescriptionJSON{
		StreamName:           st.Name,
		StreamARN:            st.ARN,
		StreamStatus:         string(st.Status),
		Shards:               shards,
		HasMoreShards:        false,
		RetentionPeriodHours: st.RetentionPeriodHours,
	}

	return jsonOK(describeStreamResponse{StreamDescription: desc})
}

func handleListStreams(_ *service.RequestContext, store *Store) (*service.Response, error) {
	names := store.ListStreams()
	if names == nil {
		names = []string{}
	}
	return jsonOK(listStreamsResponse{StreamNames: names, HasMoreStreams: false})
}

func handlePutRecord(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req putRecordRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.StreamName == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"StreamName is required.", http.StatusBadRequest))
	}
	if req.PartitionKey == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"PartitionKey is required.", http.StatusBadRequest))
	}

	data, err := base64.StdEncoding.DecodeString(req.Data)
	if err != nil {
		return jsonErr(service.NewAWSError("InvalidArgumentException",
			"Data must be base64-encoded.", http.StatusBadRequest))
	}

	shardID, seqNum, awsErr := store.PutRecord(req.StreamName, data, req.PartitionKey)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(putRecordResponse{ShardId: shardID, SequenceNumber: seqNum})
}

func handlePutRecords(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req putRecordsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.StreamName == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"StreamName is required.", http.StatusBadRequest))
	}
	if len(req.Records) == 0 {
		return jsonErr(service.NewAWSError("ValidationException",
			"Records must not be empty.", http.StatusBadRequest))
	}

	results := make([]putRecordsResultRecord, 0, len(req.Records))
	failedCount := 0

	for _, r := range req.Records {
		data, err := base64.StdEncoding.DecodeString(r.Data)
		if err != nil {
			failedCount++
			results = append(results, putRecordsResultRecord{})
			continue
		}

		shardID, seqNum, awsErr := store.PutRecord(req.StreamName, data, r.PartitionKey)
		if awsErr != nil {
			failedCount++
			results = append(results, putRecordsResultRecord{})
			continue
		}

		results = append(results, putRecordsResultRecord{
			ShardId:        shardID,
			SequenceNumber: seqNum,
		})
	}

	return jsonOK(putRecordsResponse{
		Records:           results,
		FailedRecordCount: failedCount,
	})
}

func handleGetShardIterator(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getShardIteratorRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.StreamName == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"StreamName is required.", http.StatusBadRequest))
	}
	if req.ShardId == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"ShardId is required.", http.StatusBadRequest))
	}
	if req.ShardIteratorType == "" {
		req.ShardIteratorType = "TRIM_HORIZON"
	}

	iter, awsErr := store.GetShardIterator(req.StreamName, req.ShardId, req.ShardIteratorType, req.StartingSequenceNumber)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(getShardIteratorResponse{ShardIterator: iter})
}

func handleGetRecords(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getRecordsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ShardIterator == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"ShardIterator is required.", http.StatusBadRequest))
	}
	if req.Limit <= 0 {
		req.Limit = 10000
	}

	records, nextIter, millisBehind, awsErr := store.GetRecords(req.ShardIterator, req.Limit)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	outRecords := make([]getRecordsRecord, len(records))
	for i, r := range records {
		outRecords[i] = getRecordsRecord{
			Data:           base64.StdEncoding.EncodeToString(r.Data),
			PartitionKey:   r.PartitionKey,
			SequenceNumber: r.SequenceNumber,
		}
	}

	return jsonOK(getRecordsResponse{
		Records:            outRecords,
		NextShardIterator:  nextIter,
		MillisBehindLatest: millisBehind,
	})
}

func handleIncreaseStreamRetentionPeriod(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req retentionPeriodRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.StreamName == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"StreamName is required.", http.StatusBadRequest))
	}

	if awsErr := store.IncreaseRetentionPeriod(req.StreamName, req.RetentionPeriodHours); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleDecreaseStreamRetentionPeriod(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req retentionPeriodRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.StreamName == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"StreamName is required.", http.StatusBadRequest))
	}

	if awsErr := store.DecreaseRetentionPeriod(req.StreamName, req.RetentionPeriodHours); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleAddTagsToStream(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req addTagsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.StreamName == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"StreamName is required.", http.StatusBadRequest))
	}

	if awsErr := store.AddTags(req.StreamName, req.Tags); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleRemoveTagsFromStream(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req removeTagsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.StreamName == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"StreamName is required.", http.StatusBadRequest))
	}

	if awsErr := store.RemoveTags(req.StreamName, req.TagKeys); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleListTagsForStream(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listTagsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.StreamName == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"StreamName is required.", http.StatusBadRequest))
	}

	tags, awsErr := store.ListTags(req.StreamName)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	tagList := make([]tagJSON, 0, len(tags))
	for k, v := range tags {
		tagList = append(tagList, tagJSON{Key: k, Value: v})
	}

	return jsonOK(listTagsResponse{Tags: tagList, HasMoreTags: false})
}
