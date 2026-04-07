package dynamodb

import (
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ---- Stream request/response types ----

type describeStreamRequest struct {
	StreamArn string `json:"StreamArn"`
}

type describeStreamResponse struct {
	StreamDescription StreamDescription `json:"StreamDescription"`
}

type getShardIteratorRequest struct {
	StreamArn         string `json:"StreamArn"`
	ShardId           string `json:"ShardId"`
	ShardIteratorType string `json:"ShardIteratorType"` // TRIM_HORIZON, LATEST
}

type getShardIteratorResponse struct {
	ShardIterator string `json:"ShardIterator"`
}

type getRecordsRequest struct {
	ShardIterator string `json:"ShardIterator"`
	Limit         int    `json:"Limit"`
}

type getRecordsResponse struct {
	Records           []*StreamRecord `json:"Records"`
	NextShardIterator string          `json:"NextShardIterator,omitempty"`
}

// ---- TTL request/response types ----

type updateTimeToLiveRequest struct {
	TableName               string           `json:"TableName"`
	TimeToLiveSpecification *TTLSpecification `json:"TimeToLiveSpecification"`
}

type updateTimeToLiveResponse struct {
	TimeToLiveSpecification *TTLSpecification `json:"TimeToLiveSpecification"`
}

type describeTimeToLiveRequest struct {
	TableName string `json:"TableName"`
}

type describeTimeToLiveResponse struct {
	TimeToLiveDescription *ttlDescription `json:"TimeToLiveDescription"`
}

type ttlDescription struct {
	AttributeName    string `json:"AttributeName,omitempty"`
	TimeToLiveStatus string `json:"TimeToLiveStatus"`
}

// ---- Stream handlers ----

func handleDescribeStream(ctx *service.RequestContext, store *TableStore) (*service.Response, error) {
	var req describeStreamRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.StreamArn == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"StreamArn is required.", http.StatusBadRequest))
	}

	stream := store.GetStreamByARN(req.StreamArn)
	if stream == nil {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"Requested resource not found: Stream not found", http.StatusBadRequest))
	}

	return jsonOK(describeStreamResponse{
		StreamDescription: stream.describe(),
	})
}

func handleGetShardIterator(ctx *service.RequestContext, store *TableStore) (*service.Response, error) {
	var req getShardIteratorRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.StreamArn == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"StreamArn is required.", http.StatusBadRequest))
	}

	stream := store.GetStreamByARN(req.StreamArn)
	if stream == nil {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"Requested resource not found: Stream not found", http.StatusBadRequest))
	}

	iterator, err := stream.getShardIterator(req.ShardId, req.ShardIteratorType)
	if err != nil {
		return jsonErr(service.NewAWSError("InternalServerError",
			err.Error(), http.StatusInternalServerError))
	}

	return jsonOK(getShardIteratorResponse{
		ShardIterator: iterator,
	})
}

func handleGetRecords(ctx *service.RequestContext, store *TableStore) (*service.Response, error) {
	var req getRecordsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ShardIterator == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"ShardIterator is required.", http.StatusBadRequest))
	}

	// Find the stream that owns this iterator by scanning all tables.
	var records []*StreamRecord
	var nextIterator string
	found := false

	var iterErr error
	store.tables.Range(func(key, value any) bool {
		table := value.(*Table)
		if table.Stream == nil {
			return true
		}
		table.Stream.iteratorMu.Lock()
		_, ok := table.Stream.iterators[req.ShardIterator]
		table.Stream.iteratorMu.Unlock()
		if ok {
			records, nextIterator, iterErr = table.Stream.getRecords(req.ShardIterator, req.Limit)
			found = true
			return false
		}
		return true
	})
	if iterErr != nil {
		return jsonErr(service.NewAWSError("ExpiredIteratorException",
			iterErr.Error(), http.StatusBadRequest))
	}

	if !found {
		return jsonErr(service.NewAWSError("ExpiredIteratorException",
			"Iterator is expired or not found.", http.StatusBadRequest))
	}

	if records == nil {
		records = []*StreamRecord{}
	}

	return jsonOK(getRecordsResponse{
		Records:           records,
		NextShardIterator: nextIterator,
	})
}

// ---- TTL handlers ----

func handleUpdateTimeToLive(ctx *service.RequestContext, store *TableStore) (*service.Response, error) {
	var req updateTimeToLiveRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.TableName == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"TableName is required.", http.StatusBadRequest))
	}
	if req.TimeToLiveSpecification == nil {
		return jsonErr(service.NewAWSError("ValidationException",
			"TimeToLiveSpecification is required.", http.StatusBadRequest))
	}

	if awsErr := store.UpdateTimeToLive(req.TableName, req.TimeToLiveSpecification); awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(updateTimeToLiveResponse{
		TimeToLiveSpecification: req.TimeToLiveSpecification,
	})
}

func handleDescribeTimeToLive(ctx *service.RequestContext, store *TableStore) (*service.Response, error) {
	var req describeTimeToLiveRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.TableName == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"TableName is required.", http.StatusBadRequest))
	}

	spec, awsErr := store.DescribeTimeToLive(req.TableName)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	desc := &ttlDescription{
		TimeToLiveStatus: "DISABLED",
	}
	if spec != nil && spec.Enabled {
		desc.AttributeName = spec.AttributeName
		desc.TimeToLiveStatus = "ENABLED"
	}

	return jsonOK(describeTimeToLiveResponse{
		TimeToLiveDescription: desc,
	})
}
