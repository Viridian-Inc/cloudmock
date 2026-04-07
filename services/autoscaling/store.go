package autoscaling

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/eventbus"
)

// LaunchConfiguration represents an Auto Scaling launch configuration.
type LaunchConfiguration struct {
	Name              string
	ARN               string
	ImageID           string
	InstanceType      string
	KeyName           string
	SecurityGroups    []string
	UserData          string
	IAMInstanceProfile string
	CreatedTime       time.Time
}

// AutoScalingGroup represents an Auto Scaling group.
type AutoScalingGroup struct {
	Name                string
	ARN                 string
	LaunchConfigName    string
	LaunchTemplateID    string
	LaunchTemplateName  string
	MinSize             int
	MaxSize             int
	DesiredCapacity     int
	DefaultCooldown     int
	AvailabilityZones   []string
	TargetGroupARNs     []string
	HealthCheckType     string
	HealthCheckGracePeriod int
	VPCZoneIdentifier   string
	Status              string
	Instances           []*AutoScalingInstance
	Tags                map[string]string
	CreatedTime         time.Time
}

// AutoScalingInstance represents an instance in an ASG.
type AutoScalingInstance struct {
	InstanceID         string
	AutoScalingGroupName string
	AvailabilityZone   string
	LifecycleState     string
	HealthStatus       string
	LaunchConfigName   string
	ProtectedFromScaleIn bool
}

// ScalingPolicy represents an Auto Scaling scaling policy.
type ScalingPolicy struct {
	Name                string
	ARN                 string
	AutoScalingGroupName string
	PolicyType          string // SimpleScaling, StepScaling, TargetTrackingScaling
	AdjustmentType      string
	ScalingAdjustment   int
	Cooldown            int
	TargetValue         float64
	MetricName          string
	Enabled             bool
}

// Tag represents an Auto Scaling tag.
type Tag struct {
	Key               string
	Value             string
	ResourceID        string
	ResourceType      string
	PropagateAtLaunch bool
}

// ScheduledAction represents a scheduled scaling action.
type ScheduledAction struct {
	ScheduledActionName  string
	ScheduledActionARN   string
	AutoScalingGroupName string
	DesiredCapacity      int
	MinSize              int
	MaxSize              int
	Recurrence           string
	StartTime            string
	EndTime              string
	TimeZone             string
}

// LifecycleHook represents a lifecycle hook on an ASG.
type LifecycleHook struct {
	LifecycleHookName        string
	AutoScalingGroupName     string
	LifecycleTransition      string // autoscaling:EC2_INSTANCE_LAUNCHING or TERMINATING
	NotificationTargetARN    string
	RoleARN                  string
	NotificationMetadata     string
	HeartbeatTimeout         int
	DefaultResult            string
}

// Store manages all Auto Scaling resources.
type Store struct {
	mu                   sync.RWMutex
	launchConfigs        map[string]*LaunchConfiguration    // keyed by name
	autoScalingGroups    map[string]*AutoScalingGroup        // keyed by name
	scalingPolicies      map[string]*ScalingPolicy           // keyed by ARN
	policyByName         map[string]map[string]string        // asgName -> policyName -> ARN
	scheduledActions     map[string]*ScheduledAction         // key: asgName|actionName
	lifecycleHooks       map[string]*LifecycleHook           // key: asgName|hookName
	accountID            string
	region               string
	instanceSeq          int
}

// NewStore returns a new Store for the given account and region.
func NewStore(accountID, region string) *Store {
	return &Store{
		launchConfigs:     make(map[string]*LaunchConfiguration),
		autoScalingGroups: make(map[string]*AutoScalingGroup),
		scalingPolicies:   make(map[string]*ScalingPolicy),
		policyByName:      make(map[string]map[string]string),
		scheduledActions:  make(map[string]*ScheduledAction),
		lifecycleHooks:    make(map[string]*LifecycleHook),
		accountID:         accountID,
		region:            region,
	}
}

// ---- ARN helpers ----

func (s *Store) lcARN(name string) string {
	return fmt.Sprintf("arn:aws:autoscaling:%s:%s:launchConfiguration:%s:launchConfigurationName/%s", s.region, s.accountID, newUUID(), name)
}

func (s *Store) asgARN(name string) string {
	return fmt.Sprintf("arn:aws:autoscaling:%s:%s:autoScalingGroup:%s:autoScalingGroupName/%s", s.region, s.accountID, newUUID(), name)
}

func (s *Store) policyARN(asgName, policyName string) string {
	return fmt.Sprintf("arn:aws:autoscaling:%s:%s:scalingPolicy:%s:autoScalingGroupName/%s:policyName/%s", s.region, s.accountID, newUUID(), asgName, policyName)
}

// ---- LaunchConfiguration operations ----

func (s *Store) CreateLaunchConfiguration(name, imageID, instanceType, keyName, userData, iamProfile string, securityGroups []string) (*LaunchConfiguration, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.launchConfigs[name]; exists {
		return nil, false
	}

	lc := &LaunchConfiguration{
		Name:               name,
		ARN:                s.lcARN(name),
		ImageID:            imageID,
		InstanceType:       instanceType,
		KeyName:            keyName,
		SecurityGroups:     securityGroups,
		UserData:           userData,
		IAMInstanceProfile: iamProfile,
		CreatedTime:        time.Now().UTC(),
	}
	s.launchConfigs[name] = lc
	return lc, true
}

func (s *Store) GetLaunchConfiguration(name string) (*LaunchConfiguration, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	lc, ok := s.launchConfigs[name]
	return lc, ok
}

func (s *Store) ListLaunchConfigurations(names []string) []*LaunchConfiguration {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(names) == 0 {
		result := make([]*LaunchConfiguration, 0, len(s.launchConfigs))
		for _, lc := range s.launchConfigs {
			result = append(result, lc)
		}
		return result
	}

	nameSet := make(map[string]bool, len(names))
	for _, n := range names {
		nameSet[n] = true
	}
	result := make([]*LaunchConfiguration, 0)
	for _, lc := range s.launchConfigs {
		if nameSet[lc.Name] {
			result = append(result, lc)
		}
	}
	return result
}

func (s *Store) DeleteLaunchConfiguration(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.launchConfigs[name]; !ok {
		return false
	}
	delete(s.launchConfigs, name)
	return true
}

// ---- AutoScalingGroup operations ----

func (s *Store) CreateAutoScalingGroup(name, lcName, vpcZoneID, healthCheckType string, minSize, maxSize, desiredCapacity, cooldown, hcGrace int, azs, tgARNs []string, tags map[string]string) (*AutoScalingGroup, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.autoScalingGroups[name]; exists {
		return nil, false
	}

	if healthCheckType == "" {
		healthCheckType = "EC2"
	}
	if cooldown == 0 {
		cooldown = 300
	}

	asg := &AutoScalingGroup{
		Name:                   name,
		ARN:                    s.asgARN(name),
		LaunchConfigName:       lcName,
		MinSize:                minSize,
		MaxSize:                maxSize,
		DesiredCapacity:        desiredCapacity,
		DefaultCooldown:        cooldown,
		AvailabilityZones:      azs,
		TargetGroupARNs:        tgARNs,
		HealthCheckType:        healthCheckType,
		HealthCheckGracePeriod: hcGrace,
		VPCZoneIdentifier:      vpcZoneID,
		Status:                 "InService",
		Instances:              make([]*AutoScalingInstance, 0),
		Tags:                   tags,
		CreatedTime:            time.Now().UTC(),
	}

	s.autoScalingGroups[name] = asg

	// Create simulated instances to match desired capacity using stub IDs.
	s.reconcileInstances(asg)

	return asg, true
}

// CreateAutoScalingGroupWithEC2 creates an ASG and provisions instances via EC2 locator.
func (s *Store) CreateAutoScalingGroupWithEC2(name, lcName, vpcZoneID, healthCheckType string, minSize, maxSize, desiredCapacity, cooldown, hcGrace int, azs, tgARNs []string, tags map[string]string, locator ServiceLocator, bus *eventbus.Bus) (*AutoScalingGroup, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.autoScalingGroups[name]; exists {
		return nil, false
	}

	if healthCheckType == "" {
		healthCheckType = "EC2"
	}
	if cooldown == 0 {
		cooldown = 300
	}

	asg := &AutoScalingGroup{
		Name:                   name,
		ARN:                    s.asgARN(name),
		LaunchConfigName:       lcName,
		MinSize:                minSize,
		MaxSize:                maxSize,
		DesiredCapacity:        desiredCapacity,
		DefaultCooldown:        cooldown,
		AvailabilityZones:      azs,
		TargetGroupARNs:        tgARNs,
		HealthCheckType:        healthCheckType,
		HealthCheckGracePeriod: hcGrace,
		VPCZoneIdentifier:      vpcZoneID,
		Status:                 "InService",
		Instances:              make([]*AutoScalingInstance, 0),
		Tags:                   tags,
		CreatedTime:            time.Now().UTC(),
	}

	s.autoScalingGroups[name] = asg
	s.reconcileInstancesViaEC2(asg, locator, bus)

	return asg, true
}

func (s *Store) GetAutoScalingGroup(name string) (*AutoScalingGroup, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	asg, ok := s.autoScalingGroups[name]
	return asg, ok
}

func (s *Store) ListAutoScalingGroups(names []string) []*AutoScalingGroup {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(names) == 0 {
		result := make([]*AutoScalingGroup, 0, len(s.autoScalingGroups))
		for _, asg := range s.autoScalingGroups {
			result = append(result, asg)
		}
		return result
	}

	nameSet := make(map[string]bool, len(names))
	for _, n := range names {
		nameSet[n] = true
	}
	result := make([]*AutoScalingGroup, 0)
	for _, asg := range s.autoScalingGroups {
		if nameSet[asg.Name] {
			result = append(result, asg)
		}
	}
	return result
}

func (s *Store) UpdateAutoScalingGroup(name, lcName, vpcZoneID, healthCheckType string, minSize, maxSize, desiredCapacity, cooldown, hcGrace int) (*AutoScalingGroup, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	asg, ok := s.autoScalingGroups[name]
	if !ok {
		return nil, false
	}

	if lcName != "" {
		asg.LaunchConfigName = lcName
	}
	if vpcZoneID != "" {
		asg.VPCZoneIdentifier = vpcZoneID
	}
	if healthCheckType != "" {
		asg.HealthCheckType = healthCheckType
	}
	if minSize >= 0 {
		asg.MinSize = minSize
	}
	if maxSize > 0 {
		asg.MaxSize = maxSize
	}
	if desiredCapacity >= 0 {
		asg.DesiredCapacity = desiredCapacity
		s.reconcileInstances(asg)
	}
	if cooldown > 0 {
		asg.DefaultCooldown = cooldown
	}
	if hcGrace > 0 {
		asg.HealthCheckGracePeriod = hcGrace
	}
	return asg, true
}

func (s *Store) DeleteAutoScalingGroup(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.autoScalingGroups[name]; !ok {
		return false
	}
	delete(s.autoScalingGroups, name)

	// Remove associated policies.
	if pols, ok := s.policyByName[name]; ok {
		for _, arn := range pols {
			delete(s.scalingPolicies, arn)
		}
		delete(s.policyByName, name)
	}
	return true
}

func (s *Store) SetDesiredCapacity(name string, capacity int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	asg, ok := s.autoScalingGroups[name]
	if !ok {
		return false
	}
	asg.DesiredCapacity = capacity
	s.reconcileInstances(asg)
	return true
}

// SetDesiredCapacityWithEC2 sets desired capacity and reconciles via EC2.
func (s *Store) SetDesiredCapacityWithEC2(name string, capacity int, locator ServiceLocator, bus *eventbus.Bus) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	asg, ok := s.autoScalingGroups[name]
	if !ok {
		return false
	}
	asg.DesiredCapacity = capacity
	s.reconcileInstancesViaEC2(asg, locator, bus)
	return true
}

// UpdateAutoScalingGroupWithEC2 updates an ASG and reconciles via EC2 if capacity changed.
func (s *Store) UpdateAutoScalingGroupWithEC2(name, lcName, vpcZoneID, healthCheckType string, minSize, maxSize, desiredCapacity, cooldown, hcGrace int, locator ServiceLocator, bus *eventbus.Bus) (*AutoScalingGroup, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	asg, ok := s.autoScalingGroups[name]
	if !ok {
		return nil, false
	}

	if lcName != "" {
		asg.LaunchConfigName = lcName
	}
	if vpcZoneID != "" {
		asg.VPCZoneIdentifier = vpcZoneID
	}
	if healthCheckType != "" {
		asg.HealthCheckType = healthCheckType
	}
	if minSize >= 0 {
		asg.MinSize = minSize
	}
	if maxSize > 0 {
		asg.MaxSize = maxSize
	}
	if desiredCapacity >= 0 {
		asg.DesiredCapacity = desiredCapacity
		s.reconcileInstancesViaEC2(asg, locator, bus)
	}
	if cooldown > 0 {
		asg.DefaultCooldown = cooldown
	}
	if hcGrace > 0 {
		asg.HealthCheckGracePeriod = hcGrace
	}
	return asg, true
}

// DeleteAutoScalingGroupWithEC2 deletes an ASG and terminates its EC2 instances.
func (s *Store) DeleteAutoScalingGroupWithEC2(name string, locator ServiceLocator, bus *eventbus.Bus) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	asg, ok := s.autoScalingGroups[name]
	if !ok {
		return false
	}

	// Terminate all instances
	for _, inst := range asg.Instances {
		s.terminateEC2Instance(inst.InstanceID, locator)
		if bus != nil {
			bus.Publish(&eventbus.Event{
				Source: "autoscaling",
				Type:   "autoscaling:EC2_INSTANCE_TERMINATE",
				Detail: map[string]any{
					"AutoScalingGroupName": asg.Name,
					"EC2InstanceId":        inst.InstanceID,
				},
			})
		}
	}

	delete(s.autoScalingGroups, name)

	// Remove associated policies.
	if pols, ok := s.policyByName[name]; ok {
		for _, arn := range pols {
			delete(s.scalingPolicies, arn)
		}
		delete(s.policyByName, name)
	}
	return true
}

// reconcileInstances adjusts the instance list to match desired capacity.
// Must be called with s.mu held.
func (s *Store) reconcileInstances(asg *AutoScalingGroup) {
	current := len(asg.Instances)
	desired := asg.DesiredCapacity

	if desired > current {
		for i := current; i < desired; i++ {
			az := ""
			if len(asg.AvailabilityZones) > 0 {
				az = asg.AvailabilityZones[i%len(asg.AvailabilityZones)]
			}
			s.instanceSeq++
			inst := &AutoScalingInstance{
				InstanceID:           fmt.Sprintf("i-%s", randomHex(8)),
				AutoScalingGroupName: asg.Name,
				AvailabilityZone:     az,
				LifecycleState:       "InService",
				HealthStatus:         "Healthy",
				LaunchConfigName:     asg.LaunchConfigName,
			}
			asg.Instances = append(asg.Instances, inst)
		}
	} else if desired < current {
		asg.Instances = asg.Instances[:desired]
	}
}

func (s *Store) ListAutoScalingInstances() []*AutoScalingInstance {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*AutoScalingInstance, 0)
	for _, asg := range s.autoScalingGroups {
		result = append(result, asg.Instances...)
	}
	return result
}

func (s *Store) AttachInstances(asgName string, instanceIDs []string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	asg, ok := s.autoScalingGroups[asgName]
	if !ok {
		return false
	}
	for _, id := range instanceIDs {
		az := ""
		if len(asg.AvailabilityZones) > 0 {
			az = asg.AvailabilityZones[0]
		}
		inst := &AutoScalingInstance{
			InstanceID:           id,
			AutoScalingGroupName: asgName,
			AvailabilityZone:     az,
			LifecycleState:       "InService",
			HealthStatus:         "Healthy",
			LaunchConfigName:     asg.LaunchConfigName,
		}
		asg.Instances = append(asg.Instances, inst)
	}
	asg.DesiredCapacity = len(asg.Instances)
	return true
}

func (s *Store) DetachInstances(asgName string, instanceIDs []string, decrementDesired bool) ([]*AutoScalingInstance, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	asg, ok := s.autoScalingGroups[asgName]
	if !ok {
		return nil, false
	}

	idSet := make(map[string]bool, len(instanceIDs))
	for _, id := range instanceIDs {
		idSet[id] = true
	}

	detached := make([]*AutoScalingInstance, 0)
	remaining := make([]*AutoScalingInstance, 0)
	for _, inst := range asg.Instances {
		if idSet[inst.InstanceID] {
			inst.LifecycleState = "Detaching"
			detached = append(detached, inst)
		} else {
			remaining = append(remaining, inst)
		}
	}
	asg.Instances = remaining
	if decrementDesired {
		asg.DesiredCapacity -= len(detached)
		if asg.DesiredCapacity < 0 {
			asg.DesiredCapacity = 0
		}
	}
	return detached, true
}

// ---- ScalingPolicy operations ----

func (s *Store) PutScalingPolicy(asgName, policyName, policyType, adjustmentType string, scalingAdjustment, cooldown int, targetValue float64, metricName string) (*ScalingPolicy, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.autoScalingGroups[asgName]; !ok {
		return nil, false
	}

	if policyType == "" {
		policyType = "SimpleScaling"
	}

	// Check for existing policy.
	if pols, ok := s.policyByName[asgName]; ok {
		if existingARN, ok := pols[policyName]; ok {
			// Update existing.
			pol := s.scalingPolicies[existingARN]
			pol.PolicyType = policyType
			pol.AdjustmentType = adjustmentType
			pol.ScalingAdjustment = scalingAdjustment
			pol.Cooldown = cooldown
			pol.TargetValue = targetValue
			pol.MetricName = metricName
			pol.Enabled = true
			return pol, true
		}
	}

	arn := s.policyARN(asgName, policyName)
	pol := &ScalingPolicy{
		Name:                 policyName,
		ARN:                  arn,
		AutoScalingGroupName: asgName,
		PolicyType:           policyType,
		AdjustmentType:       adjustmentType,
		ScalingAdjustment:    scalingAdjustment,
		Cooldown:             cooldown,
		TargetValue:          targetValue,
		MetricName:           metricName,
		Enabled:              true,
	}
	s.scalingPolicies[arn] = pol
	if _, ok := s.policyByName[asgName]; !ok {
		s.policyByName[asgName] = make(map[string]string)
	}
	s.policyByName[asgName][policyName] = arn
	return pol, true
}

func (s *Store) ListScalingPolicies(asgName string, policyNames []string) []*ScalingPolicy {
	s.mu.RLock()
	defer s.mu.RUnlock()

	nameSet := make(map[string]bool, len(policyNames))
	for _, n := range policyNames {
		nameSet[n] = true
	}

	result := make([]*ScalingPolicy, 0)
	for _, pol := range s.scalingPolicies {
		if asgName != "" && pol.AutoScalingGroupName != asgName {
			continue
		}
		if len(nameSet) > 0 && !nameSet[pol.Name] {
			continue
		}
		result = append(result, pol)
	}
	return result
}

func (s *Store) DeleteScalingPolicy(asgName, policyName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	pols, ok := s.policyByName[asgName]
	if !ok {
		return false
	}
	arn, ok := pols[policyName]
	if !ok {
		return false
	}
	delete(s.scalingPolicies, arn)
	delete(pols, policyName)
	return true
}

// ---- Tag operations ----

func (s *Store) CreateOrUpdateTags(tags []Tag) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, t := range tags {
		if asg, ok := s.autoScalingGroups[t.ResourceID]; ok {
			if asg.Tags == nil {
				asg.Tags = make(map[string]string)
			}
			asg.Tags[t.Key] = t.Value
		}
	}
}

func (s *Store) ListTags(asgName string) []Tag {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Tag, 0)
	for _, asg := range s.autoScalingGroups {
		if asgName != "" && asg.Name != asgName {
			continue
		}
		for k, v := range asg.Tags {
			result = append(result, Tag{
				Key:          k,
				Value:        v,
				ResourceID:   asg.Name,
				ResourceType: "auto-scaling-group",
			})
		}
	}
	return result
}

func (s *Store) DeleteTags(tags []Tag) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, t := range tags {
		if asg, ok := s.autoScalingGroups[t.ResourceID]; ok {
			delete(asg.Tags, t.Key)
		}
	}
}

// ---- Scheduled Actions ----

func scheduledActionKey(asgName, actionName string) string {
	return asgName + "|" + actionName
}

func (s *Store) scheduledActionARN(asgName, actionName string) string {
	return fmt.Sprintf("arn:aws:autoscaling:%s:%s:scheduledUpdateGroupAction:*:autoScalingGroupName/%s:scheduledActionName/%s",
		s.region, s.accountID, asgName, actionName)
}

// PutScheduledAction creates or updates a scheduled action.
func (s *Store) PutScheduledAction(asgName, actionName, recurrence, startTime, endTime, timeZone string, desiredCapacity, minSize, maxSize int) (*ScheduledAction, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.autoScalingGroups[asgName]; !ok {
		return nil, false
	}
	key := scheduledActionKey(asgName, actionName)
	if existing, ok := s.scheduledActions[key]; ok {
		if recurrence != "" {
			existing.Recurrence = recurrence
		}
		existing.StartTime = startTime
		existing.EndTime = endTime
		existing.TimeZone = timeZone
		if desiredCapacity >= 0 {
			existing.DesiredCapacity = desiredCapacity
		}
		if minSize >= 0 {
			existing.MinSize = minSize
		}
		if maxSize >= 0 {
			existing.MaxSize = maxSize
		}
		return existing, true
	}
	sa := &ScheduledAction{
		ScheduledActionName:  actionName,
		ScheduledActionARN:   s.scheduledActionARN(asgName, actionName),
		AutoScalingGroupName: asgName,
		Recurrence:           recurrence,
		StartTime:            startTime,
		EndTime:              endTime,
		TimeZone:             timeZone,
		DesiredCapacity:      desiredCapacity,
		MinSize:              minSize,
		MaxSize:              maxSize,
	}
	s.scheduledActions[key] = sa
	return sa, true
}

// DescribeScheduledActions returns scheduled actions for an ASG.
func (s *Store) DescribeScheduledActions(asgName string, actionNames []string) []*ScheduledAction {
	s.mu.RLock()
	defer s.mu.RUnlock()
	nameSet := make(map[string]bool, len(actionNames))
	for _, n := range actionNames {
		nameSet[n] = true
	}
	result := make([]*ScheduledAction, 0)
	for _, sa := range s.scheduledActions {
		if asgName != "" && sa.AutoScalingGroupName != asgName {
			continue
		}
		if len(nameSet) > 0 && !nameSet[sa.ScheduledActionName] {
			continue
		}
		result = append(result, sa)
	}
	return result
}

// DeleteScheduledAction removes a scheduled action.
func (s *Store) DeleteScheduledAction(asgName, actionName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := scheduledActionKey(asgName, actionName)
	if _, ok := s.scheduledActions[key]; !ok {
		return false
	}
	delete(s.scheduledActions, key)
	return true
}

// ---- Lifecycle Hooks ----

func lifecycleHookKey(asgName, hookName string) string {
	return asgName + "|" + hookName
}

// PutLifecycleHook creates or updates a lifecycle hook.
func (s *Store) PutLifecycleHook(asgName, hookName, transition, targetARN, roleARN, metadata, defaultResult string, heartbeatTimeout int) (*LifecycleHook, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.autoScalingGroups[asgName]; !ok {
		return nil, false
	}
	key := lifecycleHookKey(asgName, hookName)
	if existing, ok := s.lifecycleHooks[key]; ok {
		if transition != "" {
			existing.LifecycleTransition = transition
		}
		if targetARN != "" {
			existing.NotificationTargetARN = targetARN
		}
		if roleARN != "" {
			existing.RoleARN = roleARN
		}
		if metadata != "" {
			existing.NotificationMetadata = metadata
		}
		if defaultResult != "" {
			existing.DefaultResult = defaultResult
		}
		if heartbeatTimeout > 0 {
			existing.HeartbeatTimeout = heartbeatTimeout
		}
		return existing, true
	}
	hook := &LifecycleHook{
		LifecycleHookName:     hookName,
		AutoScalingGroupName:  asgName,
		LifecycleTransition:   transition,
		NotificationTargetARN: targetARN,
		RoleARN:               roleARN,
		NotificationMetadata:  metadata,
		DefaultResult:         defaultResult,
		HeartbeatTimeout:      heartbeatTimeout,
	}
	if hook.DefaultResult == "" {
		hook.DefaultResult = "ABANDON"
	}
	if hook.HeartbeatTimeout == 0 {
		hook.HeartbeatTimeout = 3600
	}
	s.lifecycleHooks[key] = hook
	return hook, true
}

// DescribeLifecycleHooks returns lifecycle hooks for an ASG.
func (s *Store) DescribeLifecycleHooks(asgName string, hookNames []string) []*LifecycleHook {
	s.mu.RLock()
	defer s.mu.RUnlock()
	nameSet := make(map[string]bool, len(hookNames))
	for _, n := range hookNames {
		nameSet[n] = true
	}
	result := make([]*LifecycleHook, 0)
	for _, hook := range s.lifecycleHooks {
		if asgName != "" && hook.AutoScalingGroupName != asgName {
			continue
		}
		if len(nameSet) > 0 && !nameSet[hook.LifecycleHookName] {
			continue
		}
		result = append(result, hook)
	}
	return result
}

// DeleteLifecycleHook removes a lifecycle hook.
func (s *Store) DeleteLifecycleHook(asgName, hookName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := lifecycleHookKey(asgName, hookName)
	if _, ok := s.lifecycleHooks[key]; !ok {
		return false
	}
	delete(s.lifecycleHooks, key)
	return true
}

// ExecutePolicy simulates executing a scaling policy by adjusting desired capacity.
func (s *Store) ExecutePolicy(asgName, policyName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	asg, ok := s.autoScalingGroups[asgName]
	if !ok {
		return false
	}
	// Find the policy and apply its adjustment.
	pols, ok := s.policyByName[asgName]
	if !ok {
		return false
	}
	policyARN, ok := pols[policyName]
	if !ok {
		return false
	}
	pol := s.scalingPolicies[policyARN]
	if pol == nil {
		return false
	}
	newDesired := asg.DesiredCapacity + pol.ScalingAdjustment
	if newDesired < asg.MinSize {
		newDesired = asg.MinSize
	}
	if newDesired > asg.MaxSize {
		newDesired = asg.MaxSize
	}
	asg.DesiredCapacity = newDesired
	s.reconcileInstances(asg)
	return true
}

// EnableMetricsCollection enables metrics collection for an ASG.
func (s *Store) EnableMetricsCollection(asgName, granularity string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.autoScalingGroups[asgName]
	return ok
}

// DisableMetricsCollection disables metrics collection for an ASG.
func (s *Store) DisableMetricsCollection(asgName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.autoScalingGroups[asgName]
	return ok
}

// ---- utility ----

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
