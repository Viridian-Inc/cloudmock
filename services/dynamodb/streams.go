package dynamodb

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// StreamRecord represents a single change event in a DynamoDB Stream.
type StreamRecord struct {
	EventID        string `json:"eventID"`
	EventName      string `json:"eventName"` // INSERT, MODIFY, REMOVE
	NewImage       Item   `json:"NewImage,omitempty"`
	OldImage       Item   `json:"OldImage,omitempty"`
	SequenceNumber string `json:"SequenceNumber"`
	StreamViewType string `json:"StreamViewType"`
}

// StreamSpecification describes whether streams are enabled and the view type.
type StreamSpecification struct {
	StreamEnabled  bool   `json:"StreamEnabled"`
	StreamViewType string `json:"StreamViewType,omitempty"` // KEYS_ONLY, NEW_IMAGE, OLD_IMAGE, NEW_AND_OLD_IMAGES
}

// Shard represents a shard in a DynamoDB Stream.
type Shard struct {
	ShardId string `json:"ShardId"`
}

// StreamDescription holds metadata about a table's stream.
type StreamDescription struct {
	StreamARN         string  `json:"StreamArn"`
	StreamLabel       string  `json:"StreamLabel"`
	StreamStatus      string  `json:"StreamStatus"` // ENABLED, DISABLED
	StreamViewType    string  `json:"StreamViewType"`
	TableName         string  `json:"TableName"`
	Shards            []Shard `json:"Shards"`
}

// Stream holds the in-memory state for a table's DynamoDB Stream.
type Stream struct {
	mu             sync.RWMutex
	arn            string
	label          string
	viewType       string
	tableName      string
	records        []*StreamRecord
	seqCounter     atomic.Int64
	shardId        string
	iterators      map[string]int // iteratorId → position in records slice
	iteratorMu     sync.Mutex
}

// newStream creates a new Stream for the given table.
func newStream(tableARN, tableName, viewType string) *Stream {
	label := time.Now().UTC().Format("2006-01-02T15:04:05.000")
	return &Stream{
		arn:       tableARN + "/stream/" + label,
		label:     label,
		viewType:  viewType,
		tableName: tableName,
		shardId:   "shardId-" + uuid.New().String()[:8],
		iterators: make(map[string]int),
	}
}

// appendRecord adds a stream record for a write event.
func (s *Stream) appendRecord(eventName string, oldImage, newImage Item) {
	s.mu.Lock()
	defer s.mu.Unlock()

	seq := s.seqCounter.Add(1)

	rec := &StreamRecord{
		EventID:        uuid.New().String(),
		EventName:      eventName,
		SequenceNumber: fmt.Sprintf("%012d", seq),
		StreamViewType: s.viewType,
	}

	switch s.viewType {
	case "NEW_AND_OLD_IMAGES":
		rec.NewImage = newImage
		rec.OldImage = oldImage
	case "NEW_IMAGE":
		rec.NewImage = newImage
	case "OLD_IMAGE":
		rec.OldImage = oldImage
	case "KEYS_ONLY":
		// no images
	}

	s.records = append(s.records, rec)
}

// getShardIterator returns a new iterator ID starting at the given position type.
func (s *Stream) getShardIterator(shardId, iteratorType string) (string, error) {
	s.mu.RLock()
	recordLen := len(s.records)
	s.mu.RUnlock()

	var pos int
	switch iteratorType {
	case "TRIM_HORIZON":
		pos = 0
	case "LATEST":
		pos = recordLen
	default:
		pos = 0
	}

	iteratorId := uuid.New().String()
	s.iteratorMu.Lock()
	s.iterators[iteratorId] = pos
	s.iteratorMu.Unlock()

	return iteratorId, nil
}

// getRecords returns records from the given iterator position, up to limit.
func (s *Stream) getRecords(iteratorId string, limit int) ([]*StreamRecord, string, error) {
	if limit <= 0 {
		limit = 1000
	}

	s.iteratorMu.Lock()
	pos, ok := s.iterators[iteratorId]
	if !ok {
		s.iteratorMu.Unlock()
		return nil, "", fmt.Errorf("expired iterator")
	}
	delete(s.iterators, iteratorId)
	s.iteratorMu.Unlock()

	s.mu.RLock()
	end := pos + limit
	if end > len(s.records) {
		end = len(s.records)
	}
	result := s.records[pos:end]
	s.mu.RUnlock()

	// Create next iterator.
	nextIteratorId := uuid.New().String()
	s.iteratorMu.Lock()
	s.iterators[nextIteratorId] = end
	s.iteratorMu.Unlock()

	return result, nextIteratorId, nil
}

// describe returns a StreamDescription for this stream.
func (s *Stream) describe() StreamDescription {
	return StreamDescription{
		StreamARN:      s.arn,
		StreamLabel:    s.label,
		StreamStatus:   "ENABLED",
		StreamViewType: s.viewType,
		TableName:      s.tableName,
		Shards: []Shard{
			{ShardId: s.shardId},
		},
	}
}
