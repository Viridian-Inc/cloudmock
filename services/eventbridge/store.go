package eventbridge

import (
	"fmt"
	"sync"
	"time"
)

// EventBus represents an EventBridge event bus.
type EventBus struct {
	Name   string
	ARN    string
	Policy string
	Tags   map[string]string
	Rules  map[string]*Rule // keyed by rule name
}

// Rule represents an EventBridge rule on an event bus.
type Rule struct {
	Name               string
	ARN                string
	EventBusName       string
	EventPattern       string
	ScheduleExpression string
	State              string // ENABLED | DISABLED
	Description        string
	Tags               map[string]string
	Targets            []Target
}

// Target is a destination attached to a Rule.
type Target struct {
	Id        string
	Arn       string
	Input     string
	InputPath string
}

// PutEvent is a record of a published event.
type PutEvent struct {
	EventId      string
	Source       string
	DetailType   string
	Detail       string
	EventBusName string
	Resources    []string
	Time         time.Time
}

// Store manages all EventBridge state in memory.
type Store struct {
	mu        sync.RWMutex
	buses     map[string]*EventBus // keyed by bus name
	events    []*PutEvent
	accountID string
	region    string
}

// NewStore returns a new Store for the given account and region.
// The "default" event bus is created automatically.
func NewStore(accountID, region string) *Store {
	s := &Store{
		buses:     make(map[string]*EventBus),
		events:    make([]*PutEvent, 0),
		accountID: accountID,
		region:    region,
	}
	// Always seed the default bus.
	defaultARN := s.busARN("default")
	s.buses["default"] = &EventBus{
		Name:  "default",
		ARN:   defaultARN,
		Tags:  make(map[string]string),
		Rules: make(map[string]*Rule),
	}
	return s
}

// busARN builds an ARN for an event bus.
func (s *Store) busARN(name string) string {
	if name == "default" {
		return fmt.Sprintf("arn:aws:events:%s:%s:event-bus/default", s.region, s.accountID)
	}
	return fmt.Sprintf("arn:aws:events:%s:%s:event-bus/%s", s.region, s.accountID, name)
}

// ruleARN builds an ARN for a rule.
func (s *Store) ruleARN(busName, ruleName string) string {
	return fmt.Sprintf("arn:aws:events:%s:%s:rule/%s/%s", s.region, s.accountID, busName, ruleName)
}

// CreateEventBus creates a new event bus. Returns the new bus or an error string if it already exists.
func (s *Store) CreateEventBus(name string, tags map[string]string) (*EventBus, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.buses[name]; exists {
		return nil, false
	}

	if tags == nil {
		tags = make(map[string]string)
	}

	bus := &EventBus{
		Name:  name,
		ARN:   s.busARN(name),
		Tags:  tags,
		Rules: make(map[string]*Rule),
	}
	s.buses[name] = bus
	return bus, true
}

// DeleteEventBus removes an event bus by name. Returns false if not found or if name is "default".
func (s *Store) DeleteEventBus(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if name == "default" {
		return false
	}
	if _, exists := s.buses[name]; !exists {
		return false
	}
	delete(s.buses, name)
	return true
}

// GetEventBus returns an event bus by name.
func (s *Store) GetEventBus(name string) (*EventBus, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	bus, ok := s.buses[name]
	return bus, ok
}

// ListEventBuses returns all event buses.
func (s *Store) ListEventBuses() []*EventBus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*EventBus, 0, len(s.buses))
	for _, bus := range s.buses {
		result = append(result, bus)
	}
	return result
}

// PutRule creates or replaces a rule on an event bus. Returns the rule ARN and true,
// or empty string and false if the bus is not found.
func (s *Store) PutRule(busName, name, eventPattern, scheduleExpression, state, description string, tags map[string]string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	bus, ok := s.buses[busName]
	if !ok {
		return "", false
	}

	if state == "" {
		state = "ENABLED"
	}
	if tags == nil {
		tags = make(map[string]string)
	}

	arn := s.ruleARN(busName, name)

	// Preserve existing targets if rule already exists.
	var targets []Target
	if existing, exists := bus.Rules[name]; exists {
		targets = existing.Targets
		// Merge new tags on top of existing.
		for k, v := range tags {
			existing.Tags[k] = v
		}
		tags = existing.Tags
	} else {
		targets = make([]Target, 0)
	}

	bus.Rules[name] = &Rule{
		Name:               name,
		ARN:                arn,
		EventBusName:       busName,
		EventPattern:       eventPattern,
		ScheduleExpression: scheduleExpression,
		State:              state,
		Description:        description,
		Tags:               tags,
		Targets:            targets,
	}
	return arn, true
}

// DeleteRule removes a rule from an event bus.
func (s *Store) DeleteRule(busName, name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	bus, ok := s.buses[busName]
	if !ok {
		return false
	}
	if _, exists := bus.Rules[name]; !exists {
		return false
	}
	delete(bus.Rules, name)
	return true
}

// GetRule returns a rule by name and bus.
func (s *Store) GetRule(busName, name string) (*Rule, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	bus, ok := s.buses[busName]
	if !ok {
		return nil, false
	}
	rule, ok := bus.Rules[name]
	return rule, ok
}

// ListRules returns all rules on a bus, filtered by optional name prefix.
func (s *Store) ListRules(busName, namePrefix string) ([]*Rule, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	bus, ok := s.buses[busName]
	if !ok {
		return nil, false
	}

	result := make([]*Rule, 0, len(bus.Rules))
	for _, rule := range bus.Rules {
		if namePrefix == "" || len(rule.Name) >= len(namePrefix) && rule.Name[:len(namePrefix)] == namePrefix {
			result = append(result, rule)
		}
	}
	return result, true
}

// PutTargets adds or replaces targets on a rule. Returns a list of failed target IDs.
func (s *Store) PutTargets(busName, ruleName string, targets []Target) ([]string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	bus, ok := s.buses[busName]
	if !ok {
		return nil, false
	}
	rule, ok := bus.Rules[ruleName]
	if !ok {
		return nil, false
	}

	// Build a map of existing targets by ID.
	existing := make(map[string]int)
	for i, t := range rule.Targets {
		existing[t.Id] = i
	}

	for _, newT := range targets {
		if idx, found := existing[newT.Id]; found {
			rule.Targets[idx] = newT
		} else {
			rule.Targets = append(rule.Targets, newT)
			existing[newT.Id] = len(rule.Targets) - 1
		}
	}

	return []string{}, true
}

// RemoveTargets removes targets by ID from a rule. Returns ids that were not found.
func (s *Store) RemoveTargets(busName, ruleName string, ids []string) ([]string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	bus, ok := s.buses[busName]
	if !ok {
		return nil, false
	}
	rule, ok := bus.Rules[ruleName]
	if !ok {
		return nil, false
	}

	removeSet := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		removeSet[id] = struct{}{}
	}

	updated := make([]Target, 0, len(rule.Targets))
	found := make(map[string]struct{})
	for _, t := range rule.Targets {
		if _, remove := removeSet[t.Id]; remove {
			found[t.Id] = struct{}{}
		} else {
			updated = append(updated, t)
		}
	}
	rule.Targets = updated

	// Collect IDs that were not found.
	notFound := make([]string, 0)
	for _, id := range ids {
		if _, ok := found[id]; !ok {
			notFound = append(notFound, id)
		}
	}
	return notFound, true
}

// ListTargetsByRule returns targets for a rule.
func (s *Store) ListTargetsByRule(busName, ruleName string) ([]Target, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	bus, ok := s.buses[busName]
	if !ok {
		return nil, false
	}
	rule, ok := bus.Rules[ruleName]
	if !ok {
		return nil, false
	}

	result := make([]Target, len(rule.Targets))
	copy(result, rule.Targets)
	return result, true
}

// PutEvents records events in the log. Returns a slice of generated EventIds (one per entry).
func (s *Store) PutEvents(entries []PutEvent) []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	ids := make([]string, 0, len(entries))
	for i := range entries {
		entries[i].EventId = newUUID()
		if entries[i].Time.IsZero() {
			entries[i].Time = time.Now().UTC()
		}
		cp := entries[i]
		s.events = append(s.events, &cp)
		ids = append(ids, entries[i].EventId)
	}
	return ids
}

// SetRuleState enables or disables a rule.
func (s *Store) SetRuleState(busName, ruleName, state string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	bus, ok := s.buses[busName]
	if !ok {
		return false
	}
	rule, ok := bus.Rules[ruleName]
	if !ok {
		return false
	}
	rule.State = state
	return true
}

// TagResource applies tags to an event bus or rule identified by ARN.
func (s *Store) TagResource(resourceARN string, tags map[string]string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if bus := s.findBusByARN(resourceARN); bus != nil {
		for k, v := range tags {
			bus.Tags[k] = v
		}
		return true
	}
	if rule := s.findRuleByARN(resourceARN); rule != nil {
		for k, v := range tags {
			rule.Tags[k] = v
		}
		return true
	}
	return false
}

// UntagResource removes tags from a resource identified by ARN.
func (s *Store) UntagResource(resourceARN string, tagKeys []string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if bus := s.findBusByARN(resourceARN); bus != nil {
		for _, k := range tagKeys {
			delete(bus.Tags, k)
		}
		return true
	}
	if rule := s.findRuleByARN(resourceARN); rule != nil {
		for _, k := range tagKeys {
			delete(rule.Tags, k)
		}
		return true
	}
	return false
}

// ListTagsForResource returns tags for a resource identified by ARN.
func (s *Store) ListTagsForResource(resourceARN string) (map[string]string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if bus := s.findBusByARN(resourceARN); bus != nil {
		cp := make(map[string]string, len(bus.Tags))
		for k, v := range bus.Tags {
			cp[k] = v
		}
		return cp, true
	}
	if rule := s.findRuleByARN(resourceARN); rule != nil {
		cp := make(map[string]string, len(rule.Tags))
		for k, v := range rule.Tags {
			cp[k] = v
		}
		return cp, true
	}
	return nil, false
}

// findBusByARN finds an event bus by ARN. Caller must hold at least a read lock.
func (s *Store) findBusByARN(arn string) *EventBus {
	for _, bus := range s.buses {
		if bus.ARN == arn {
			return bus
		}
	}
	return nil
}

// findRuleByARN finds a rule by ARN. Caller must hold at least a read lock.
func (s *Store) findRuleByARN(arn string) *Rule {
	for _, bus := range s.buses {
		for _, rule := range bus.Rules {
			if rule.ARN == arn {
				return rule
			}
		}
	}
	return nil
}
