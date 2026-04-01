package iotdata

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
)

type Shadow struct {
	ThingName string
	ShadowName string // empty for classic shadow
	State     map[string]any
	Metadata  map[string]any
	Version   int64
	Timestamp time.Time
}

type PublishedMessage struct {
	Topic   string
	Payload []byte
	QoS     int
	Time    time.Time
}

type Store struct {
	mu        sync.RWMutex
	shadows   map[string]map[string]*Shadow // thingName -> shadowName -> shadow
	messages  []PublishedMessage
	accountID string
	region    string
}

func NewStore(accountID, region string) *Store {
	return &Store{
		shadows:   make(map[string]map[string]*Shadow),
		messages:  make([]PublishedMessage, 0),
		accountID: accountID,
		region:    region,
	}
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func (s *Store) GetThingShadow(thingName, shadowName string) (*Shadow, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	thingShadows, ok := s.shadows[thingName]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("No shadow exists for thing %s", thingName), http.StatusNotFound)
	}
	shadow, ok := thingShadows[shadowName]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("No shadow named %s exists for thing %s", shadowName, thingName), http.StatusNotFound)
	}
	return shadow, nil
}

func (s *Store) UpdateThingShadow(thingName, shadowName string, state map[string]any) (*Shadow, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.shadows[thingName]; !ok {
		s.shadows[thingName] = make(map[string]*Shadow)
	}

	existing, ok := s.shadows[thingName][shadowName]
	now := time.Now().UTC()
	if ok {
		// Merge desired/reported state and compute delta.
		if existing.State == nil {
			existing.State = make(map[string]any)
		}
		for k, v := range state {
			if vMap, isMap := v.(map[string]any); isMap {
				if existingMap, isExistingMap := existing.State[k].(map[string]any); isExistingMap {
					for mk, mv := range vMap {
						existingMap[mk] = mv
					}
				} else {
					existing.State[k] = v
				}
			} else {
				existing.State[k] = v
			}
		}
		// Compute delta between desired and reported
		desired, _ := existing.State["desired"].(map[string]any)
		reported, _ := existing.State["reported"].(map[string]any)
		if desired != nil && reported != nil {
			delta := make(map[string]any)
			for k, dv := range desired {
				if rv, ok := reported[k]; !ok || fmt.Sprintf("%v", dv) != fmt.Sprintf("%v", rv) {
					delta[k] = dv
				}
			}
			if len(delta) > 0 {
				existing.State["delta"] = delta
			} else {
				delete(existing.State, "delta")
			}
		}
		existing.Version++
		existing.Timestamp = now
		return existing, nil
	}

	shadow := &Shadow{
		ThingName:  thingName,
		ShadowName: shadowName,
		State:      state,
		Metadata:   map[string]any{},
		Version:    1,
		Timestamp:  now,
	}
	// Compute delta for newly created shadow
	desired, _ := state["desired"].(map[string]any)
	reported, _ := state["reported"].(map[string]any)
	if desired != nil && reported != nil {
		delta := make(map[string]any)
		for k, dv := range desired {
			if rv, ok := reported[k]; !ok || fmt.Sprintf("%v", dv) != fmt.Sprintf("%v", rv) {
				delta[k] = dv
			}
		}
		if len(delta) > 0 {
			shadow.State["delta"] = delta
		}
	}
	s.shadows[thingName][shadowName] = shadow
	return shadow, nil
}

func (s *Store) DeleteThingShadow(thingName, shadowName string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	thingShadows, ok := s.shadows[thingName]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("No shadow exists for thing %s", thingName), http.StatusNotFound)
	}
	if _, ok := thingShadows[shadowName]; !ok {
		return service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("No shadow named %s exists for thing %s", shadowName, thingName), http.StatusNotFound)
	}
	delete(thingShadows, shadowName)
	if len(thingShadows) == 0 {
		delete(s.shadows, thingName)
	}
	return nil
}

func (s *Store) ListNamedShadowsForThing(thingName string) ([]string, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	thingShadows, ok := s.shadows[thingName]
	if !ok {
		return []string{}, nil
	}
	names := make([]string, 0, len(thingShadows))
	for name := range thingShadows {
		if name != "" { // exclude classic shadow from named list
			names = append(names, name)
		}
	}
	return names, nil
}

func (s *Store) Publish(topic string, payload []byte, qos int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = append(s.messages, PublishedMessage{
		Topic:   topic,
		Payload: payload,
		QoS:     qos,
		Time:    time.Now().UTC(),
	})
}
