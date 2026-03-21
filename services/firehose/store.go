package firehose

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
)

// deliveryStreamStatus represents the lifecycle state of a Firehose delivery stream.
type deliveryStreamStatus string

const (
	streamStatusActive   deliveryStreamStatus = "ACTIVE"
	streamStatusCreating deliveryStreamStatus = "CREATING"
	streamStatusDeleting deliveryStreamStatus = "DELETING"
)

// BufferingHints holds S3 buffering configuration for a destination.
type BufferingHints struct {
	IntervalInSeconds int
	SizeInMBs         int
}

// Destination holds a single delivery destination for a stream.
type Destination struct {
	DestinationId string
	S3BucketARN   string
	S3Prefix      string
	RoleARN       string
	BufferingHints BufferingHints
}

// Record holds a single in-memory Firehose record.
type Record struct {
	RecordId  string
	Data      []byte
	Timestamp time.Time
}

// DeliveryStream represents an in-memory Firehose delivery stream.
type DeliveryStream struct {
	Name         string
	ARN          string
	Status       deliveryStreamStatus
	Type         string
	Destinations []Destination
	Tags         map[string]string
	Records      []Record
	CreatedAt    time.Time
}

// Store is the in-memory store for Firehose delivery streams.
type Store struct {
	mu        sync.RWMutex
	streams   map[string]*DeliveryStream
	accountID string
	region    string
}

// NewStore creates a new empty Firehose Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		streams:   make(map[string]*DeliveryStream),
		accountID: accountID,
		region:    region,
	}
}

// newUUID returns a random UUID v4 string.
func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// buildStreamARN constructs a Firehose delivery stream ARN.
func (s *Store) buildStreamARN(name string) string {
	return fmt.Sprintf("arn:aws:firehose:%s:%s:deliverystream/%s", s.region, s.accountID, name)
}

// CreateDeliveryStream creates a new delivery stream.
func (s *Store) CreateDeliveryStream(
	name, streamType string,
	dest Destination,
) (string, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.streams[name]; exists {
		return "", service.NewAWSError("ResourceInUseException",
			fmt.Sprintf("Delivery stream %s already exists under account %s.", name, s.accountID),
			http.StatusBadRequest)
	}

	if streamType == "" {
		streamType = "DirectPut"
	}

	dest.DestinationId = "destinationId-000000000001"
	arn := s.buildStreamARN(name)

	s.streams[name] = &DeliveryStream{
		Name:         name,
		ARN:          arn,
		Status:       streamStatusActive,
		Type:         streamType,
		Destinations: []Destination{dest},
		Tags:         make(map[string]string),
		Records:      []Record{},
		CreatedAt:    time.Now().UTC(),
	}
	return arn, nil
}

// DeleteDeliveryStream removes a delivery stream by name.
func (s *Store) DeleteDeliveryStream(name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.streams[name]; !exists {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Delivery stream %s under account %s not found.", name, s.accountID),
			http.StatusBadRequest)
	}
	delete(s.streams, name)
	return nil
}

// GetDeliveryStream returns the delivery stream by name.
func (s *Store) GetDeliveryStream(name string) (*DeliveryStream, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.getStreamLocked(name)
}

// getStreamLocked returns the stream; caller must hold at least a read lock.
func (s *Store) getStreamLocked(name string) (*DeliveryStream, *service.AWSError) {
	st, ok := s.streams[name]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Delivery stream %s under account %s not found.", name, s.accountID),
			http.StatusBadRequest)
	}
	return st, nil
}

// ListDeliveryStreams returns a sorted list of delivery stream names.
func (s *Store) ListDeliveryStreams() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	names := make([]string, 0, len(s.streams))
	for n := range s.streams {
		names = append(names, n)
	}
	return names
}

// PutRecord appends a single record to the delivery stream and returns its RecordId.
func (s *Store) PutRecord(streamName string, data []byte) (string, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	st, awsErr := s.getStreamLocked(streamName)
	if awsErr != nil {
		return "", awsErr
	}

	id := newUUID()
	st.Records = append(st.Records, Record{
		RecordId:  id,
		Data:      data,
		Timestamp: time.Now().UTC(),
	})
	return id, nil
}

// PutRecordBatch appends multiple records and returns a slice of RecordIds in the
// same order as the input. All records are appended; no partial failures occur in
// this in-memory implementation.
func (s *Store) PutRecordBatch(streamName string, records [][]byte) ([]string, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	st, awsErr := s.getStreamLocked(streamName)
	if awsErr != nil {
		return nil, awsErr
	}

	ids := make([]string, 0, len(records))
	now := time.Now().UTC()
	for _, data := range records {
		id := newUUID()
		st.Records = append(st.Records, Record{
			RecordId:  id,
			Data:      data,
			Timestamp: now,
		})
		ids = append(ids, id)
	}
	return ids, nil
}

// UpdateDestination replaces the S3 configuration for an existing destination.
func (s *Store) UpdateDestination(streamName, destinationID string, update Destination) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	st, awsErr := s.getStreamLocked(streamName)
	if awsErr != nil {
		return awsErr
	}

	for i, d := range st.Destinations {
		if d.DestinationId == destinationID {
			if update.S3BucketARN != "" {
				st.Destinations[i].S3BucketARN = update.S3BucketARN
			}
			if update.S3Prefix != "" {
				st.Destinations[i].S3Prefix = update.S3Prefix
			}
			if update.RoleARN != "" {
				st.Destinations[i].RoleARN = update.RoleARN
			}
			if update.BufferingHints.IntervalInSeconds > 0 {
				st.Destinations[i].BufferingHints.IntervalInSeconds = update.BufferingHints.IntervalInSeconds
			}
			if update.BufferingHints.SizeInMBs > 0 {
				st.Destinations[i].BufferingHints.SizeInMBs = update.BufferingHints.SizeInMBs
			}
			return nil
		}
	}

	return service.NewAWSError("InvalidArgumentException",
		fmt.Sprintf("Destination %s not found in delivery stream %s.", destinationID, streamName),
		http.StatusBadRequest)
}

// TagStream adds or updates tags on a delivery stream.
func (s *Store) TagStream(streamName string, tags map[string]string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	st, awsErr := s.getStreamLocked(streamName)
	if awsErr != nil {
		return awsErr
	}
	for k, v := range tags {
		st.Tags[k] = v
	}
	return nil
}

// UntagStream removes tags from a delivery stream.
func (s *Store) UntagStream(streamName string, tagKeys []string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	st, awsErr := s.getStreamLocked(streamName)
	if awsErr != nil {
		return awsErr
	}
	for _, k := range tagKeys {
		delete(st.Tags, k)
	}
	return nil
}

// ListTags returns a copy of the tags for a delivery stream.
func (s *Store) ListTags(streamName string) (map[string]string, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	st, awsErr := s.getStreamLocked(streamName)
	if awsErr != nil {
		return nil, awsErr
	}
	out := make(map[string]string, len(st.Tags))
	for k, v := range st.Tags {
		out[k] = v
	}
	return out, nil
}
