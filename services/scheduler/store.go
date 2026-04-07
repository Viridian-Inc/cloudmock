package scheduler

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// Schedule represents an EventBridge Scheduler schedule.
type Schedule struct {
	ARN                        string
	Name                       string
	GroupName                   string
	Description                string
	ScheduleExpression         string
	ScheduleExpressionTimezone string
	State                      string
	FlexibleTimeWindow         *FlexibleTimeWindow
	Target                     *Target
	StartDate                  *time.Time
	EndDate                    *time.Time
	KmsKeyArn                  string
	CreationDate               time.Time
	LastModificationDate       time.Time
	Tags                       map[string]string
	// Behavioral fields
	InvocationHistory []InvocationRecord
	cancelFunc        func() // cancels the scheduled worker
}

// FlexibleTimeWindow configures schedule flexibility.
type FlexibleTimeWindow struct {
	Mode                string
	MaximumWindowInMinutes int
}

// Target is the invocation target for a schedule.
type Target struct {
	Arn     string
	RoleArn string
	Input   string
}

// InvocationRecord tracks a schedule invocation.
type InvocationRecord struct {
	Time    time.Time
	Success bool
	Error   string
}

// ServiceLocator resolves other services for cross-service invocation.
type ServiceLocator interface {
	Lookup(name string) (service.Service, error)
}

// ScheduleGroup represents a schedule group.
type ScheduleGroup struct {
	ARN          string
	Name         string
	State        string
	CreationDate time.Time
	LastModificationDate time.Time
	Tags         map[string]string
}

// Store manages all Scheduler state in memory.
type Store struct {
	mu             sync.RWMutex
	schedules      map[string]map[string]*Schedule // groupName -> scheduleName -> schedule
	scheduleGroups map[string]*ScheduleGroup
	accountID      string
	region         string
	locator        ServiceLocator
}

// NewStore returns a new Store for the given account and region.
func NewStore(accountID, region string) *Store {
	s := &Store{
		schedules:      make(map[string]map[string]*Schedule),
		scheduleGroups: make(map[string]*ScheduleGroup),
		accountID:      accountID,
		region:         region,
	}
	// Create the default group.
	now := time.Now().UTC()
	s.scheduleGroups["default"] = &ScheduleGroup{
		ARN:                  s.groupARN("default"),
		Name:                 "default",
		State:                "ACTIVE",
		CreationDate:         now,
		LastModificationDate: now,
		Tags:                 make(map[string]string),
	}
	s.schedules["default"] = make(map[string]*Schedule)
	return s
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func (s *Store) scheduleARN(groupName, scheduleName string) string {
	return fmt.Sprintf("arn:aws:scheduler:%s:%s:schedule/%s/%s", s.region, s.accountID, groupName, scheduleName)
}

func (s *Store) groupARN(name string) string {
	return fmt.Sprintf("arn:aws:scheduler:%s:%s:schedule-group/%s", s.region, s.accountID, name)
}

// CreateSchedule creates a new schedule.
func (s *Store) CreateSchedule(name, groupName, description, expression, timezone, state string, flexWindow *FlexibleTimeWindow, target *Target, kmsKeyArn string, startDate, endDate *time.Time, tags map[string]string) (*Schedule, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if groupName == "" {
		groupName = "default"
	}
	if _, ok := s.scheduleGroups[groupName]; !ok {
		return nil, false
	}
	if _, ok := s.schedules[groupName][name]; ok {
		return nil, false
	}
	if state == "" {
		state = "ENABLED"
	}
	if tags == nil {
		tags = make(map[string]string)
	}
	now := time.Now().UTC()
	sched := &Schedule{
		ARN: s.scheduleARN(groupName, name), Name: name, GroupName: groupName,
		Description: description, ScheduleExpression: expression,
		ScheduleExpressionTimezone: timezone, State: state,
		FlexibleTimeWindow: flexWindow, Target: target,
		KmsKeyArn: kmsKeyArn, StartDate: startDate, EndDate: endDate,
		CreationDate: now, LastModificationDate: now, Tags: tags,
	}
	s.schedules[groupName][name] = sched
	s.scheduleExecution(sched)
	return sched, true
}

// GetSchedule returns a schedule by name and group.
func (s *Store) GetSchedule(name, groupName string) (*Schedule, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if groupName == "" {
		groupName = "default"
	}
	group, ok := s.schedules[groupName]
	if !ok {
		return nil, false
	}
	sched, ok := group[name]
	return sched, ok
}

// ListSchedules returns all schedules in a group.
func (s *Store) ListSchedules(groupName, state, namePrefix string) []*Schedule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if groupName == "" {
		groupName = "default"
	}
	group, ok := s.schedules[groupName]
	if !ok {
		return nil
	}
	result := make([]*Schedule, 0, len(group))
	for _, sched := range group {
		if state != "" && sched.State != state {
			continue
		}
		if namePrefix != "" && (len(sched.Name) < len(namePrefix) || sched.Name[:len(namePrefix)] != namePrefix) {
			continue
		}
		result = append(result, sched)
	}
	return result
}

// UpdateSchedule updates a schedule.
func (s *Store) UpdateSchedule(name, groupName, description, expression, timezone, state string, flexWindow *FlexibleTimeWindow, target *Target) (*Schedule, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if groupName == "" {
		groupName = "default"
	}
	group, ok := s.schedules[groupName]
	if !ok {
		return nil, false
	}
	sched, ok := group[name]
	if !ok {
		return nil, false
	}
	if description != "" {
		sched.Description = description
	}
	if expression != "" {
		sched.ScheduleExpression = expression
	}
	if timezone != "" {
		sched.ScheduleExpressionTimezone = timezone
	}
	if state != "" {
		sched.State = state
	}
	if flexWindow != nil {
		sched.FlexibleTimeWindow = flexWindow
	}
	if target != nil {
		sched.Target = target
	}
	sched.LastModificationDate = time.Now().UTC()
	return sched, true
}

// DeleteSchedule removes a schedule.
func (s *Store) DeleteSchedule(name, groupName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if groupName == "" {
		groupName = "default"
	}
	group, ok := s.schedules[groupName]
	if !ok {
		return false
	}
	if _, ok := group[name]; !ok {
		return false
	}
	delete(group, name)
	return true
}

// CreateScheduleGroup creates a new schedule group.
func (s *Store) CreateScheduleGroup(name string, tags map[string]string) (*ScheduleGroup, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.scheduleGroups[name]; ok {
		return nil, false
	}
	if tags == nil {
		tags = make(map[string]string)
	}
	now := time.Now().UTC()
	group := &ScheduleGroup{
		ARN: s.groupARN(name), Name: name, State: "ACTIVE",
		CreationDate: now, LastModificationDate: now, Tags: tags,
	}
	s.scheduleGroups[name] = group
	s.schedules[name] = make(map[string]*Schedule)
	return group, true
}

// GetScheduleGroup returns a schedule group by name.
func (s *Store) GetScheduleGroup(name string) (*ScheduleGroup, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	group, ok := s.scheduleGroups[name]
	return group, ok
}

// ListScheduleGroups returns all schedule groups.
func (s *Store) ListScheduleGroups(namePrefix string) []*ScheduleGroup {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*ScheduleGroup, 0, len(s.scheduleGroups))
	for _, g := range s.scheduleGroups {
		if namePrefix != "" && (len(g.Name) < len(namePrefix) || g.Name[:len(namePrefix)] != namePrefix) {
			continue
		}
		result = append(result, g)
	}
	return result
}

// DeleteScheduleGroup removes a schedule group and all its schedules.
func (s *Store) DeleteScheduleGroup(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if name == "default" {
		return false
	}
	if _, ok := s.scheduleGroups[name]; !ok {
		return false
	}
	delete(s.scheduleGroups, name)
	delete(s.schedules, name)
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

// SetLocator sets the service locator for cross-service invocation.
func (s *Store) SetLocator(locator ServiceLocator) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.locator = locator
}

// GetInvocationHistory returns invocation records for a schedule.
func (s *Store) GetInvocationHistory(name, groupName string) []InvocationRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if groupName == "" {
		groupName = "default"
	}
	group, ok := s.schedules[groupName]
	if !ok {
		return nil
	}
	sched, ok := group[name]
	if !ok {
		return nil
	}
	result := make([]InvocationRecord, len(sched.InvocationHistory))
	copy(result, sched.InvocationHistory)
	return result
}

// parseRateExpression parses "rate(5 minutes)" and returns the duration.
func parseRateExpression(expr string) (time.Duration, bool) {
	re := regexp.MustCompile(`^rate\((\d+)\s+(minutes?|hours?|days?)\)$`)
	matches := re.FindStringSubmatch(expr)
	if matches == nil {
		return 0, false
	}
	val, _ := strconv.Atoi(matches[1])
	unit := strings.TrimSuffix(matches[2], "s")
	switch unit {
	case "minute":
		return time.Duration(val) * time.Minute, true
	case "hour":
		return time.Duration(val) * time.Hour, true
	case "day":
		return time.Duration(val) * 24 * time.Hour, true
	default:
		return 0, false
	}
}

// isOneTimeExpression returns true if the expression is an "at()" one-time schedule.
func isOneTimeExpression(expr string) bool {
	return strings.HasPrefix(expr, "at(")
}

// invokeTarget invokes the schedule's target via the service locator.
// Must NOT be called with s.mu held.
func (s *Store) invokeTarget(sched *Schedule) {
	if sched.Target == nil || sched.Target.Arn == "" {
		s.recordInvocation(sched, false, "no target configured")
		return
	}

	s.mu.RLock()
	locator := s.locator
	s.mu.RUnlock()

	if locator == nil {
		s.recordInvocation(sched, false, "no service locator")
		return
	}

	targetARN := sched.Target.Arn
	input := sched.Target.Input

	// Determine target service and action from ARN.
	svcName, action, actionBody := resolveTarget(targetARN, input)
	if svcName == "" {
		s.recordInvocation(sched, false, "unrecognized target ARN: "+targetARN)
		return
	}

	targetSvc, err := locator.Lookup(svcName)
	if err != nil {
		s.recordInvocation(sched, false, "target service not found: "+svcName)
		return
	}

	ctx := &service.RequestContext{
		Action:     action,
		Body:       actionBody,
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
	}
	_, err = targetSvc.HandleRequest(ctx)
	if err != nil {
		s.recordInvocation(sched, false, err.Error())
		return
	}
	s.recordInvocation(sched, true, "")
}

// recordInvocation appends an invocation record to the schedule.
func (s *Store) recordInvocation(sched *Schedule, success bool, errMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sched.InvocationHistory = append(sched.InvocationHistory, InvocationRecord{
		Time:    time.Now().UTC(),
		Success: success,
		Error:   errMsg,
	})
}

// resolveTarget determines which service and action to invoke from a target ARN.
func resolveTarget(arn, input string) (svcName, action string, body []byte) {
	// Lambda: arn:aws:lambda:...:function:name
	if strings.Contains(arn, ":function:") {
		parts := strings.Split(arn, ":")
		funcName := parts[len(parts)-1]
		payload := input
		if payload == "" {
			payload = "{}"
		}
		body, _ = json.Marshal(map[string]any{
			"FunctionName": funcName,
			"Payload":      payload,
		})
		return "lambda", "Invoke", body
	}
	// SQS: arn:aws:sqs:...:queue-name
	if strings.Contains(arn, ":sqs:") || strings.Contains(arn, "sqs.") {
		body, _ = json.Marshal(map[string]any{
			"QueueUrl":    arn,
			"MessageBody": input,
		})
		return "sqs", "SendMessage", body
	}
	// SNS: arn:aws:sns:...:topic-name
	if strings.Contains(arn, ":sns:") {
		body, _ = json.Marshal(map[string]any{
			"TopicArn": arn,
			"Message":  input,
		})
		return "sns", "Publish", body
	}
	return "", "", nil
}

// scheduleExecution sets up invocation for a schedule (instant fire for one-time, or store interval info).
// Must be called with s.mu held.
func (s *Store) scheduleExecution(sched *Schedule) {
	if sched.State != "ENABLED" {
		return
	}

	expr := sched.ScheduleExpression

	if isOneTimeExpression(expr) {
		// One-time schedule: fire immediately in mock mode.
		go s.invokeTarget(sched)
		return
	}

	if _, ok := parseRateExpression(expr); ok {
		// Rate expression: fire once immediately in mock mode.
		go s.invokeTarget(sched)
		return
	}

	// Cron expression or unknown: fire once immediately.
	if strings.HasPrefix(expr, "cron(") {
		go s.invokeTarget(sched)
		return
	}
}

func (s *Store) findTagsByARN(arn string) map[string]string {
	for _, g := range s.scheduleGroups {
		if g.ARN == arn {
			return g.Tags
		}
	}
	for _, group := range s.schedules {
		for _, sched := range group {
			if sched.ARN == arn {
				return sched.Tags
			}
		}
	}
	return nil
}
