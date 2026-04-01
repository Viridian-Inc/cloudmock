package config

import (
	"strings"
	"time"

	"github.com/neureaux/cloudmock/pkg/eventbus"
)

// ConfigurationItem represents a snapshot of resource configuration at a point in time.
type ConfigurationItem struct {
	ResourceType                string
	ResourceId                  string
	ResourceName                string
	ConfigurationItemCaptureTime time.Time
	ConfigurationItemStatus     string
	Configuration               map[string]any
	AccountId                   string
	AwsRegion                   string
}

// SetEventBus sets the event bus for cross-service configuration tracking.
func (s *ConfigService) SetEventBus(bus *eventbus.Bus) {
	s.bus = bus
}

// startBusRecording subscribes the recorder to the event bus.
func (s *ConfigService) startBusRecording(recorderName string) {
	if s.bus == nil {
		return
	}
	s.store.mu.Lock()
	defer s.store.mu.Unlock()

	rec, ok := s.store.recorders[recorderName]
	if !ok || rec.subscriptionID != "" {
		return
	}

	sub := &eventbus.Subscription{
		Types: []string{"*"},
		Handler: func(event *eventbus.Event) error {
			s.recordConfigItem(rec, event)
			return nil
		},
	}
	rec.subscriptionID = s.bus.Subscribe(sub)
}

// stopBusRecording unsubscribes the recorder from the event bus.
func (s *ConfigService) stopBusRecording(recorderName string) {
	if s.bus == nil {
		return
	}
	s.store.mu.Lock()
	defer s.store.mu.Unlock()

	rec, ok := s.store.recorders[recorderName]
	if !ok || rec.subscriptionID == "" {
		return
	}

	s.bus.Unsubscribe(rec.subscriptionID)
	rec.subscriptionID = ""
}

// recordConfigItem processes a bus event and creates a configuration item.
func (s *ConfigService) recordConfigItem(rec *ConfigurationRecorder, busEvent *eventbus.Event) {
	// Only process ApiCall events
	parts := strings.SplitN(busEvent.Type, ":", 3)
	if len(parts) < 3 || parts[1] != "ApiCall" {
		return
	}

	resourceType, _ := busEvent.Detail["resourceType"].(string)
	resourceId, _ := busEvent.Detail["resourceId"].(string)
	resourceName, _ := busEvent.Detail["resourceName"].(string)

	// Only record if we have resource info
	if resourceType == "" {
		return
	}

	// Check if this resource type is tracked by the recorder
	s.store.mu.RLock()
	if rec.RecordingGroup != nil && !rec.RecordingGroup.AllSupported {
		found := false
		for _, rt := range rec.RecordingGroup.ResourceTypes {
			if rt == resourceType {
				found = true
				break
			}
		}
		if !found {
			s.store.mu.RUnlock()
			return
		}
	}
	s.store.mu.RUnlock()

	item := ConfigurationItem{
		ResourceType:                resourceType,
		ResourceId:                  resourceId,
		ResourceName:                resourceName,
		ConfigurationItemCaptureTime: busEvent.Time,
		ConfigurationItemStatus:     "OK",
		Configuration:               busEvent.Detail,
		AccountId:                   busEvent.AccountID,
		AwsRegion:                   busEvent.Region,
	}

	key := resourceType + ":" + resourceId
	s.store.mu.Lock()
	s.store.configItems[key] = append(s.store.configItems[key], item)
	s.store.mu.Unlock()
}
