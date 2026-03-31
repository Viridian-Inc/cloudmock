package applicationautoscaling

import (
	"fmt"
	"sync"
	"time"
)

// ScalableTarget represents a registered scalable target.
type ScalableTarget struct {
	ServiceNamespace  string
	ResourceID        string
	ScalableDimension string
	MinCapacity       int
	MaxCapacity       int
	RoleARN           string
	CreationTime      time.Time
	SuspendedState    *SuspendedState
	Tags              map[string]string
}

// SuspendedState represents the suspended state of a scalable target.
type SuspendedState struct {
	DynamicScalingInSuspended  bool
	DynamicScalingOutSuspended bool
	ScheduledScalingSuspended  bool
}

// ScalingPolicy represents a scaling policy.
type ScalingPolicy struct {
	PolicyARN                string
	PolicyName               string
	ServiceNamespace         string
	ResourceID               string
	ScalableDimension        string
	PolicyType               string
	TargetTrackingConfig     map[string]any
	StepScalingConfig        map[string]any
	CreationTime             time.Time
	Tags                     map[string]string
}

// ScheduledAction represents a scheduled scaling action.
type ScheduledAction struct {
	ScheduledActionARN  string
	ScheduledActionName string
	ServiceNamespace    string
	ResourceID          string
	ScalableDimension   string
	Schedule            string
	Timezone            string
	StartTime           *time.Time
	EndTime             *time.Time
	ScalableTargetAction *ScalableTargetAction
	CreationTime        time.Time
	Tags                map[string]string
}

// ScalableTargetAction defines the min/max capacity for a scheduled action.
type ScalableTargetAction struct {
	MinCapacity int
	MaxCapacity int
}

// Store manages all Application Auto Scaling state in memory.
type Store struct {
	mu               sync.RWMutex
	scalableTargets  map[string]*ScalableTarget  // key: namespace|resourceID|dimension
	scalingPolicies  map[string]*ScalingPolicy    // key: namespace|resourceID|dimension|policyName
	scheduledActions map[string]*ScheduledAction  // key: namespace|resourceID|dimension|actionName
	accountID        string
	region           string
}

// NewStore returns a new Store for the given account and region.
func NewStore(accountID, region string) *Store {
	return &Store{
		scalableTargets:  make(map[string]*ScalableTarget),
		scalingPolicies:  make(map[string]*ScalingPolicy),
		scheduledActions: make(map[string]*ScheduledAction),
		accountID:        accountID,
		region:           region,
	}
}

func targetKey(namespace, resourceID, dimension string) string {
	return namespace + "|" + resourceID + "|" + dimension
}

func policyKey(namespace, resourceID, dimension, policyName string) string {
	return namespace + "|" + resourceID + "|" + dimension + "|" + policyName
}

func actionKey(namespace, resourceID, dimension, actionName string) string {
	return namespace + "|" + resourceID + "|" + dimension + "|" + actionName
}

func (s *Store) policyARN(namespace, policyName string) string {
	return fmt.Sprintf("arn:aws:autoscaling:%s:%s:scalingPolicy:%s:resource/%s/policy/%s",
		s.region, s.accountID, newShortID(), namespace, policyName)
}

func (s *Store) scheduledActionARN(namespace, actionName string) string {
	return fmt.Sprintf("arn:aws:autoscaling:%s:%s:scheduledAction:%s:resource/%s/scheduledAction/%s",
		s.region, s.accountID, newShortID(), namespace, actionName)
}

// RegisterScalableTarget registers or updates a scalable target.
func (s *Store) RegisterScalableTarget(namespace, resourceID, dimension string, minCap, maxCap int, roleARN string, suspended *SuspendedState, tags map[string]string) *ScalableTarget {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := targetKey(namespace, resourceID, dimension)
	if tags == nil {
		tags = make(map[string]string)
	}
	if existing, ok := s.scalableTargets[key]; ok {
		if minCap >= 0 {
			existing.MinCapacity = minCap
		}
		if maxCap >= 0 {
			existing.MaxCapacity = maxCap
		}
		if roleARN != "" {
			existing.RoleARN = roleARN
		}
		if suspended != nil {
			existing.SuspendedState = suspended
		}
		for k, v := range tags {
			existing.Tags[k] = v
		}
		return existing
	}
	target := &ScalableTarget{
		ServiceNamespace:  namespace,
		ResourceID:        resourceID,
		ScalableDimension: dimension,
		MinCapacity:       minCap,
		MaxCapacity:       maxCap,
		RoleARN:           roleARN,
		CreationTime:      time.Now().UTC(),
		SuspendedState:    suspended,
		Tags:              tags,
	}
	s.scalableTargets[key] = target
	return target
}

// DescribeScalableTargets returns targets matching the filters.
func (s *Store) DescribeScalableTargets(namespace string, resourceIDs []string, dimension string) []*ScalableTarget {
	s.mu.RLock()
	defer s.mu.RUnlock()
	resourceSet := make(map[string]struct{}, len(resourceIDs))
	for _, id := range resourceIDs {
		resourceSet[id] = struct{}{}
	}
	result := make([]*ScalableTarget, 0)
	for _, t := range s.scalableTargets {
		if t.ServiceNamespace != namespace {
			continue
		}
		if dimension != "" && t.ScalableDimension != dimension {
			continue
		}
		if len(resourceSet) > 0 {
			if _, ok := resourceSet[t.ResourceID]; !ok {
				continue
			}
		}
		result = append(result, t)
	}
	return result
}

// DeregisterScalableTarget removes a scalable target.
func (s *Store) DeregisterScalableTarget(namespace, resourceID, dimension string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := targetKey(namespace, resourceID, dimension)
	if _, ok := s.scalableTargets[key]; !ok {
		return false
	}
	delete(s.scalableTargets, key)
	return true
}

// PutScalingPolicy creates or updates a scaling policy.
func (s *Store) PutScalingPolicy(namespace, resourceID, dimension, policyName, policyType string, targetTracking, stepScaling map[string]any, tags map[string]string) *ScalingPolicy {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := policyKey(namespace, resourceID, dimension, policyName)
	if tags == nil {
		tags = make(map[string]string)
	}
	if policyType == "" {
		policyType = "TargetTrackingScaling"
	}
	if existing, ok := s.scalingPolicies[key]; ok {
		existing.PolicyType = policyType
		existing.TargetTrackingConfig = targetTracking
		existing.StepScalingConfig = stepScaling
		for k, v := range tags {
			existing.Tags[k] = v
		}
		return existing
	}
	policy := &ScalingPolicy{
		PolicyARN:            s.policyARN(namespace, policyName),
		PolicyName:           policyName,
		ServiceNamespace:     namespace,
		ResourceID:           resourceID,
		ScalableDimension:    dimension,
		PolicyType:           policyType,
		TargetTrackingConfig: targetTracking,
		StepScalingConfig:    stepScaling,
		CreationTime:         time.Now().UTC(),
		Tags:                 tags,
	}
	s.scalingPolicies[key] = policy
	return policy
}

// DescribeScalingPolicies returns policies matching the filters.
func (s *Store) DescribeScalingPolicies(namespace, resourceID, dimension string, policyNames []string) []*ScalingPolicy {
	s.mu.RLock()
	defer s.mu.RUnlock()
	nameSet := make(map[string]struct{}, len(policyNames))
	for _, n := range policyNames {
		nameSet[n] = struct{}{}
	}
	result := make([]*ScalingPolicy, 0)
	for _, p := range s.scalingPolicies {
		if p.ServiceNamespace != namespace {
			continue
		}
		if resourceID != "" && p.ResourceID != resourceID {
			continue
		}
		if dimension != "" && p.ScalableDimension != dimension {
			continue
		}
		if len(nameSet) > 0 {
			if _, ok := nameSet[p.PolicyName]; !ok {
				continue
			}
		}
		result = append(result, p)
	}
	return result
}

// DeleteScalingPolicy removes a scaling policy.
func (s *Store) DeleteScalingPolicy(namespace, resourceID, dimension, policyName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := policyKey(namespace, resourceID, dimension, policyName)
	if _, ok := s.scalingPolicies[key]; !ok {
		return false
	}
	delete(s.scalingPolicies, key)
	return true
}

// PutScheduledAction creates or updates a scheduled action.
func (s *Store) PutScheduledAction(namespace, resourceID, dimension, actionName, schedule, timezone string, startTime, endTime *time.Time, targetAction *ScalableTargetAction, tags map[string]string) *ScheduledAction {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := actionKey(namespace, resourceID, dimension, actionName)
	if tags == nil {
		tags = make(map[string]string)
	}
	if existing, ok := s.scheduledActions[key]; ok {
		if schedule != "" {
			existing.Schedule = schedule
		}
		if timezone != "" {
			existing.Timezone = timezone
		}
		existing.StartTime = startTime
		existing.EndTime = endTime
		if targetAction != nil {
			existing.ScalableTargetAction = targetAction
		}
		for k, v := range tags {
			existing.Tags[k] = v
		}
		return existing
	}
	action := &ScheduledAction{
		ScheduledActionARN:   s.scheduledActionARN(namespace, actionName),
		ScheduledActionName:  actionName,
		ServiceNamespace:     namespace,
		ResourceID:           resourceID,
		ScalableDimension:    dimension,
		Schedule:             schedule,
		Timezone:             timezone,
		StartTime:            startTime,
		EndTime:              endTime,
		ScalableTargetAction: targetAction,
		CreationTime:         time.Now().UTC(),
		Tags:                 tags,
	}
	s.scheduledActions[key] = action
	return action
}

// DescribeScheduledActions returns scheduled actions matching the filters.
func (s *Store) DescribeScheduledActions(namespace, resourceID, dimension string, actionNames []string) []*ScheduledAction {
	s.mu.RLock()
	defer s.mu.RUnlock()
	nameSet := make(map[string]struct{}, len(actionNames))
	for _, n := range actionNames {
		nameSet[n] = struct{}{}
	}
	result := make([]*ScheduledAction, 0)
	for _, a := range s.scheduledActions {
		if a.ServiceNamespace != namespace {
			continue
		}
		if resourceID != "" && a.ResourceID != resourceID {
			continue
		}
		if dimension != "" && a.ScalableDimension != dimension {
			continue
		}
		if len(nameSet) > 0 {
			if _, ok := nameSet[a.ScheduledActionName]; !ok {
				continue
			}
		}
		result = append(result, a)
	}
	return result
}

// DeleteScheduledAction removes a scheduled action.
func (s *Store) DeleteScheduledAction(namespace, resourceID, dimension, actionName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := actionKey(namespace, resourceID, dimension, actionName)
	if _, ok := s.scheduledActions[key]; !ok {
		return false
	}
	delete(s.scheduledActions, key)
	return true
}

// TagResource applies tags to a resource by ARN.
func (s *Store) TagResource(arn string, tags map[string]string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r := s.findTagsByARN(arn); r != nil {
		for k, v := range tags {
			r[k] = v
		}
		return true
	}
	return false
}

// UntagResource removes tags from a resource by ARN.
func (s *Store) UntagResource(arn string, keys []string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r := s.findTagsByARN(arn); r != nil {
		for _, k := range keys {
			delete(r, k)
		}
		return true
	}
	return false
}

// ListTagsForResource returns tags for a resource by ARN.
func (s *Store) ListTagsForResource(arn string) (map[string]string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if r := s.findTagsByARN(arn); r != nil {
		cp := make(map[string]string, len(r))
		for k, v := range r {
			cp[k] = v
		}
		return cp, true
	}
	return nil, false
}

func (s *Store) findTagsByARN(arn string) map[string]string {
	for _, p := range s.scalingPolicies {
		if p.PolicyARN == arn {
			return p.Tags
		}
	}
	for _, a := range s.scheduledActions {
		if a.ScheduledActionARN == arn {
			return a.Tags
		}
	}
	return nil
}

// newShortID returns a short random ID for ARN construction.
func newShortID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
