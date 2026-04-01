package cloudtrail

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/neureaux/cloudmock/pkg/eventbus"
)

// SetEventBus sets the event bus for cross-service event recording.
func (s *CloudTrailService) SetEventBus(bus *eventbus.Bus) {
	s.bus = bus
}

// StartRecording subscribes to the event bus for all trails that have logging enabled.
// This is called externally after wiring up the bus.
func (s *CloudTrailService) StartRecording() {
	if s.bus == nil {
		return
	}
	s.store.mu.RLock()
	defer s.store.mu.RUnlock()
	for _, trail := range s.store.trails {
		if trail.IsLogging && trail.subscriptionID == "" {
			s.subscribeTrail(trail)
		}
	}
}

// subscribeTrail subscribes a single trail to the event bus.
// Caller must NOT hold store.mu.
func (s *CloudTrailService) subscribeTrail(trail *Trail) {
	if s.bus == nil {
		return
	}
	sub := &eventbus.Subscription{
		Types: []string{"*"},
		Handler: func(event *eventbus.Event) error {
			s.recordFromBusEvent(trail, event)
			return nil
		},
	}
	trail.subscriptionID = s.bus.Subscribe(sub)
}

// unsubscribeTrail removes a trail's subscription from the event bus.
func (s *CloudTrailService) unsubscribeTrail(trail *Trail) {
	if s.bus == nil || trail.subscriptionID == "" {
		return
	}
	s.bus.Unsubscribe(trail.subscriptionID)
	trail.subscriptionID = ""
}

// recordFromBusEvent processes an event bus event and stores it in the trail's event log
// if it matches the trail's event selectors.
func (s *CloudTrailService) recordFromBusEvent(trail *Trail, busEvent *eventbus.Event) {
	// Parse event type: "{service}:ApiCall:{action}"
	parts := strings.SplitN(busEvent.Type, ":", 3)
	if len(parts) < 3 {
		return
	}
	// Only process ApiCall events
	if parts[1] != "ApiCall" {
		return
	}
	eventSource := parts[0] + ".amazonaws.com"
	eventName := parts[2]

	// Check if the event matches trail's event selectors
	if !s.matchesEventSelectors(trail, eventSource, eventName, busEvent.Detail) {
		return
	}

	// Build resources from detail
	var resources []EventResource
	if resType, ok := busEvent.Detail["resourceType"].(string); ok {
		resName, _ := busEvent.Detail["resourceName"].(string)
		resources = append(resources, EventResource{
			ResourceType: resType,
			ResourceName: resName,
		})
	}

	username, _ := busEvent.Detail["username"].(string)
	readOnly := "false"
	if ro, ok := busEvent.Detail["readOnly"].(string); ok {
		readOnly = ro
	}
	accessKeyId, _ := busEvent.Detail["accessKeyId"].(string)

	// Build CloudTrailEvent JSON
	cloudTrailEvent := map[string]any{
		"eventVersion": "1.08",
		"eventSource":  eventSource,
		"eventName":    eventName,
		"awsRegion":    busEvent.Region,
		"userIdentity": map[string]any{
			"accountId": busEvent.AccountID,
			"userName":  username,
		},
		"eventTime": busEvent.Time.Format(time.RFC3339),
	}
	cteBytes, _ := json.Marshal(cloudTrailEvent)

	event := Event{
		EventId:         newUUID(),
		EventName:       eventName,
		EventTime:       busEvent.Time,
		EventSource:     eventSource,
		Username:        username,
		Resources:       resources,
		CloudTrailEvent: string(cteBytes),
		ReadOnly:        readOnly,
		AccessKeyId:     accessKeyId,
	}

	s.store.mu.Lock()
	s.store.trailEvents[trail.Name] = append(s.store.trailEvents[trail.Name], event)
	now := time.Now().UTC()
	trail.LatestDeliveryTime = &now
	s.store.mu.Unlock()
}

// matchesEventSelectors checks if an event matches the trail's event selectors.
func (s *CloudTrailService) matchesEventSelectors(trail *Trail, eventSource, eventName string, detail map[string]any) bool {
	s.store.mu.RLock()
	selectors := trail.EventSelectors
	s.store.mu.RUnlock()

	if len(selectors) == 0 {
		// Default: log all management events
		return true
	}

	isDataEvent := false
	if v, ok := detail["isDataEvent"].(bool); ok {
		isDataEvent = v
	}

	for _, sel := range selectors {
		// Check excluded management event sources
		excluded := false
		for _, excl := range sel.ExcludeManagementEventSources {
			if excl == eventSource {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}

		if isDataEvent {
			// Check data resources
			if len(sel.DataResources) > 0 {
				for _, dr := range sel.DataResources {
					if matchesReadWriteType(sel.ReadWriteType, detail) {
						_ = dr // Data events match if the type and read/write match
						return true
					}
				}
			}
		} else {
			// Management event
			if sel.IncludeManagementEvents {
				if matchesReadWriteType(sel.ReadWriteType, detail) {
					return true
				}
			}
		}
	}

	return false
}

// matchesReadWriteType checks if the event read/write type matches the selector.
func matchesReadWriteType(selectorType string, detail map[string]any) bool {
	if selectorType == "" || selectorType == "All" {
		return true
	}
	readOnly, _ := detail["readOnly"].(string)
	if selectorType == "ReadOnly" && readOnly == "true" {
		return true
	}
	if selectorType == "WriteOnly" && readOnly != "true" {
		return true
	}
	return false
}
