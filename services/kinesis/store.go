package kinesis

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// streamStatus represents the lifecycle state of a Kinesis stream.
type streamStatus string

const (
	streamStatusCreating streamStatus = "CREATING"
	streamStatusActive   streamStatus = "ACTIVE"
	streamStatusDeleting streamStatus = "DELETING"
)

// maxHashKey is 2^128 - 1, the maximum value for a shard hash key range.
var maxHashKey = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(1))

// Record holds a single Kinesis data record within a shard.
type Record struct {
	Data           []byte
	PartitionKey   string
	SequenceNumber string
	Timestamp      time.Time
}

// HashKeyRange is the inclusive range of hash keys assigned to a shard.
type HashKeyRange struct {
	StartingHashKey string
	EndingHashKey   string
}

// Shard represents one partition of a Kinesis stream.
type Shard struct {
	ShardId      string
	HashKeyRange HashKeyRange
	Records      []Record
}

// Stream represents an in-memory Kinesis stream.
type Stream struct {
	Name                 string
	ARN                  string
	Status               streamStatus
	Shards               []Shard
	RetentionPeriodHours int
	CreationTimestamp    time.Time
	Tags                 map[string]string
}

// Store is the in-memory store for Kinesis streams.
type Store struct {
	mu        sync.RWMutex
	streams   map[string]*Stream
	accountID string
	region    string
	seqGen    atomic.Uint64
}

// NewStore creates an empty Kinesis Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		streams:   make(map[string]*Stream),
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

// buildARN constructs a Kinesis stream ARN.
func (s *Store) buildARN(streamName string) string {
	return fmt.Sprintf("arn:aws:kinesis:%s:%s:stream/%s", s.region, s.accountID, streamName)
}

// nextSequenceNumber returns a monotonically increasing zero-padded sequence number.
func (s *Store) nextSequenceNumber() string {
	n := s.seqGen.Add(1)
	return fmt.Sprintf("%020d", n)
}

// shardHashRange divides the key space evenly among shardCount shards, returning
// the start and end hash keys for shard index i.
func shardHashRange(i, total int) (string, string) {
	total64 := big.NewInt(int64(total))

	// start = i * (maxHashKey+1) / total
	rangeSize := new(big.Int).Div(new(big.Int).Add(maxHashKey, big.NewInt(1)), total64)
	start := new(big.Int).Mul(big.NewInt(int64(i)), rangeSize)
	var end *big.Int
	if i == total-1 {
		end = new(big.Int).Set(maxHashKey)
	} else {
		end = new(big.Int).Sub(new(big.Int).Mul(big.NewInt(int64(i+1)), rangeSize), big.NewInt(1))
	}
	return start.String(), end.String()
}

// partitionKeyToShardIndex maps a partition key to a shard index using MD5.
func partitionKeyToShardIndex(partitionKey string, shardCount int) int {
	h := md5.Sum([]byte(partitionKey))
	// treat the 16-byte hash as a big-endian uint128
	n := new(big.Int).SetBytes(h[:])
	idx := new(big.Int).Mod(n, big.NewInt(int64(shardCount)))
	return int(idx.Int64())
}

// CreateStream creates a new stream with the given name and shard count.
func (s *Store) CreateStream(name string, shardCount int) *service.AWSError {
	if shardCount <= 0 {
		shardCount = 1
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.streams[name]; exists {
		return service.NewAWSError("ResourceInUseException",
			fmt.Sprintf("Stream %s under account %s already exists.", name, s.accountID),
			http.StatusBadRequest)
	}

	shards := make([]Shard, shardCount)
	for i := 0; i < shardCount; i++ {
		startKey, endKey := shardHashRange(i, shardCount)
		shards[i] = Shard{
			ShardId:      fmt.Sprintf("shardId-%012d", i),
			HashKeyRange: HashKeyRange{StartingHashKey: startKey, EndingHashKey: endKey},
			Records:      []Record{},
		}
	}

	s.streams[name] = &Stream{
		Name:                 name,
		ARN:                  s.buildARN(name),
		Status:               streamStatusActive,
		Shards:               shards,
		RetentionPeriodHours: 24,
		CreationTimestamp:    time.Now().UTC(),
		Tags:                 make(map[string]string),
	}
	return nil
}

// DeleteStream removes a stream by name.
func (s *Store) DeleteStream(name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.streams[name]; !exists {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Stream %s under account %s not found.", name, s.accountID),
			http.StatusBadRequest)
	}
	delete(s.streams, name)
	return nil
}

// GetStream returns the stream by name (caller must not hold mu).
func (s *Store) GetStream(name string) (*Stream, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.getStreamLocked(name)
}

// getStreamLocked returns the stream (caller must hold at least read lock).
func (s *Store) getStreamLocked(name string) (*Stream, *service.AWSError) {
	st, ok := s.streams[name]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Stream %s under account %s not found.", name, s.accountID),
			http.StatusBadRequest)
	}
	return st, nil
}

// ListStreams returns a sorted list of stream names.
func (s *Store) ListStreams() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	names := make([]string, 0, len(s.streams))
	for n := range s.streams {
		names = append(names, n)
	}
	return names
}

// PutRecord appends a record to the appropriate shard and returns the shard ID and sequence number.
func (s *Store) PutRecord(streamName string, data []byte, partitionKey string) (string, string, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	st, awsErr := s.getStreamLocked(streamName)
	if awsErr != nil {
		return "", "", awsErr
	}

	idx := partitionKeyToShardIndex(partitionKey, len(st.Shards))
	seq := s.nextSequenceNumber()

	st.Shards[idx].Records = append(st.Shards[idx].Records, Record{
		Data:           data,
		PartitionKey:   partitionKey,
		SequenceNumber: seq,
		Timestamp:      time.Now().UTC(),
	})

	return st.Shards[idx].ShardId, seq, nil
}

// shardIteratorEncoding holds the decoded contents of a shard iterator token.
type shardIteratorEncoding struct {
	StreamName string
	ShardId    string
	Position   int // index into the shard's Records slice (-1 means "LATEST sentinel")
}

// encodeShardIterator base64-encodes a shard iterator as "streamName\x00shardId\x00position".
func encodeShardIterator(enc shardIteratorEncoding) string {
	raw := fmt.Sprintf("%s\x00%s\x00%d", enc.StreamName, enc.ShardId, enc.Position)
	return base64.StdEncoding.EncodeToString([]byte(raw))
}

// decodeShardIterator reverses encodeShardIterator.
func decodeShardIterator(token string) (shardIteratorEncoding, *service.AWSError) {
	raw, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return shardIteratorEncoding{}, service.NewAWSError("InvalidArgumentException",
			"The shard iterator is invalid.", http.StatusBadRequest)
	}

	parts := splitOnNull(string(raw), 3)
	if len(parts) != 3 {
		return shardIteratorEncoding{}, service.NewAWSError("InvalidArgumentException",
			"The shard iterator is invalid.", http.StatusBadRequest)
	}

	pos, err := strconv.Atoi(parts[2])
	if err != nil {
		return shardIteratorEncoding{}, service.NewAWSError("InvalidArgumentException",
			"The shard iterator is invalid.", http.StatusBadRequest)
	}

	return shardIteratorEncoding{StreamName: parts[0], ShardId: parts[1], Position: pos}, nil
}

// splitOnNull splits s at NUL bytes, returning at most n parts.
func splitOnNull(s string, n int) []string {
	var parts []string
	for len(parts) < n-1 {
		idx := -1
		for i := 0; i < len(s); i++ {
			if s[i] == 0 {
				idx = i
				break
			}
		}
		if idx < 0 {
			break
		}
		parts = append(parts, s[:idx])
		s = s[idx+1:]
	}
	parts = append(parts, s)
	return parts
}

// GetShardIterator returns an opaque iterator token for the given shard and iterator type.
func (s *Store) GetShardIterator(streamName, shardID, iteratorType, startingSeqNum string) (string, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	st, awsErr := s.getStreamLocked(streamName)
	if awsErr != nil {
		return "", awsErr
	}

	shard := findShard(st, shardID)
	if shard == nil {
		return "", service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Shard %s in stream %s not found.", shardID, streamName),
			http.StatusBadRequest)
	}

	var pos int
	switch iteratorType {
	case "TRIM_HORIZON":
		pos = 0
	case "LATEST":
		pos = len(shard.Records)
	case "AT_SEQUENCE_NUMBER":
		p, err := findSeqPosition(shard, startingSeqNum, false)
		if err != nil {
			return "", err
		}
		pos = p
	case "AFTER_SEQUENCE_NUMBER":
		p, err := findSeqPosition(shard, startingSeqNum, true)
		if err != nil {
			return "", err
		}
		pos = p
	default:
		return "", service.NewAWSError("InvalidArgumentException",
			fmt.Sprintf("ShardIteratorType %s is invalid.", iteratorType),
			http.StatusBadRequest)
	}

	token := encodeShardIterator(shardIteratorEncoding{
		StreamName: streamName,
		ShardId:    shardID,
		Position:   pos,
	})
	return token, nil
}

// findSeqPosition finds the index in shard.Records for the given sequence number.
// If after is true, returns the index after the matching record.
func findSeqPosition(shard *Shard, seqNum string, after bool) (int, *service.AWSError) {
	for i, r := range shard.Records {
		if r.SequenceNumber == seqNum {
			if after {
				return i + 1, nil
			}
			return i, nil
		}
	}
	return 0, service.NewAWSError("InvalidArgumentException",
		fmt.Sprintf("Sequence number %s not found in shard.", seqNum),
		http.StatusBadRequest)
}

// findShard finds a shard by ID within a stream.
func findShard(st *Stream, shardID string) *Shard {
	for i := range st.Shards {
		if st.Shards[i].ShardId == shardID {
			return &st.Shards[i]
		}
	}
	return nil
}

// GetRecords reads up to limit records starting from the iterator position.
// Returns the records, next iterator token, and milliseconds behind latest.
func (s *Store) GetRecords(iteratorToken string, limit int) ([]Record, string, int64, *service.AWSError) {
	enc, awsErr := decodeShardIterator(iteratorToken)
	if awsErr != nil {
		return nil, "", 0, awsErr
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	st, awsErr := s.getStreamLocked(enc.StreamName)
	if awsErr != nil {
		return nil, "", 0, awsErr
	}

	shard := findShard(st, enc.ShardId)
	if shard == nil {
		return nil, "", 0, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Shard %s in stream %s not found.", enc.ShardId, enc.StreamName),
			http.StatusBadRequest)
	}

	pos := enc.Position
	if pos < 0 {
		pos = 0
	}
	if pos > len(shard.Records) {
		pos = len(shard.Records)
	}

	end := pos + limit
	if end > len(shard.Records) {
		end = len(shard.Records)
	}

	records := shard.Records[pos:end]

	nextPos := end
	nextToken := encodeShardIterator(shardIteratorEncoding{
		StreamName: enc.StreamName,
		ShardId:    enc.ShardId,
		Position:   nextPos,
	})

	var millisBehind int64
	if len(shard.Records) > 0 && nextPos < len(shard.Records) {
		// Estimate lag based on the timestamp of the last record vs now.
		lastRecord := shard.Records[len(shard.Records)-1]
		millisBehind = time.Since(lastRecord.Timestamp).Milliseconds()
		if millisBehind < 0 {
			millisBehind = 0
		}
	}

	return records, nextToken, millisBehind, nil
}

// IncreaseRetentionPeriod increases the stream's retention period.
func (s *Store) IncreaseRetentionPeriod(streamName string, hours int) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	st, awsErr := s.getStreamLocked(streamName)
	if awsErr != nil {
		return awsErr
	}
	if hours <= st.RetentionPeriodHours {
		return service.NewAWSError("InvalidArgumentException",
			fmt.Sprintf("Requested retention period (%d hours) must be greater than current retention period (%d hours).", hours, st.RetentionPeriodHours),
			http.StatusBadRequest)
	}
	if hours > 8760 {
		return service.NewAWSError("InvalidArgumentException",
			"Retention period must be between 24 and 8760 hours.",
			http.StatusBadRequest)
	}
	st.RetentionPeriodHours = hours
	return nil
}

// DecreaseRetentionPeriod decreases the stream's retention period.
func (s *Store) DecreaseRetentionPeriod(streamName string, hours int) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	st, awsErr := s.getStreamLocked(streamName)
	if awsErr != nil {
		return awsErr
	}
	if hours >= st.RetentionPeriodHours {
		return service.NewAWSError("InvalidArgumentException",
			fmt.Sprintf("Requested retention period (%d hours) must be less than current retention period (%d hours).", hours, st.RetentionPeriodHours),
			http.StatusBadRequest)
	}
	if hours < 24 {
		return service.NewAWSError("InvalidArgumentException",
			"Retention period must be between 24 and 8760 hours.",
			http.StatusBadRequest)
	}
	st.RetentionPeriodHours = hours
	return nil
}

// AddTags adds or updates tags on a stream.
func (s *Store) AddTags(streamName string, tags map[string]string) *service.AWSError {
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

// RemoveTags removes tags from a stream.
func (s *Store) RemoveTags(streamName string, tagKeys []string) *service.AWSError {
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

// ListTags returns a copy of the tags for a stream.
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

// randomHex32 returns 4 random bytes encoded as hex (used for shard iterator uniqueness).
func randomHex32() string {
	var b [4]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("%08x", binary.BigEndian.Uint32(b[:]))
}
