package s3

import (
	"encoding/base64"
	"encoding/json"
)

// s3State is the serialised form of all S3 state.
type s3State struct {
	Buckets []s3BucketState `json:"buckets"`
}

type s3BucketState struct {
	Name    string          `json:"name"`
	Objects []s3ObjectState `json:"objects"`
}

type s3ObjectState struct {
	Key         string `json:"key"`
	BodyBase64  string `json:"body_base64"`
	ContentType string `json:"content_type"`
}

// ExportState returns a JSON snapshot of all S3 buckets and objects.
func (s *S3Service) ExportState() (json.RawMessage, error) {
	buckets := s.store.ListBuckets()
	state := s3State{Buckets: make([]s3BucketState, 0, len(buckets))}
	for _, b := range buckets {
		bs := s3BucketState{Name: b.Name}
		b.Objects.mu.RLock()
		for _, obj := range b.Objects.objects {
			bs.Objects = append(bs.Objects, s3ObjectState{
				Key:         obj.Key,
				BodyBase64:  base64.StdEncoding.EncodeToString(obj.Body),
				ContentType: obj.ContentType,
			})
		}
		b.Objects.mu.RUnlock()
		state.Buckets = append(state.Buckets, bs)
	}
	return json.Marshal(state)
}

// ImportState restores S3 state from a JSON snapshot.
func (s *S3Service) ImportState(data json.RawMessage) error {
	var state s3State
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}
	for _, bs := range state.Buckets {
		// Ignore error if bucket already exists.
		_ = s.store.CreateBucket(bs.Name)
		objs, err := s.store.bucketObjects(bs.Name)
		if err != nil {
			return err
		}
		for _, o := range bs.Objects {
			body, err := base64.StdEncoding.DecodeString(o.BodyBase64)
			if err != nil {
				return err
			}
			objs.PutObject(o.Key, body, o.ContentType, nil)
		}
	}
	return nil
}
