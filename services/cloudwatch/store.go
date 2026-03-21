package cloudwatch

import (
	"fmt"
	"sync"
	"time"
)

// Dimension is a name/value pair that identifies a metric.
type Dimension struct {
	Name  string
	Value string
}

// MetricDatum is a single data point stored for a metric.
type MetricDatum struct {
	Namespace  string
	MetricName string
	Value      float64
	Unit       string
	Timestamp  time.Time
	Dimensions []Dimension
}

// Alarm represents a CloudWatch metric alarm definition.
type Alarm struct {
	AlarmName          string
	AlarmArn           string
	Namespace          string
	MetricName         string
	ComparisonOperator string
	Threshold          float64
	EvaluationPeriods  int
	Period             int
	Statistic          string
	StateValue         string
	StateReason        string
	AlarmActions       []string
	Dimensions         []Dimension
	Tags               map[string]string
}

// Store holds all CloudWatch metric data points and alarms in memory.
type Store struct {
	mu        sync.RWMutex
	dataPoints []*MetricDatum
	alarms     map[string]*Alarm // keyed by AlarmName
	accountID  string
	region     string
}

// NewStore returns a new Store for the given account and region.
func NewStore(accountID, region string) *Store {
	return &Store{
		dataPoints: make([]*MetricDatum, 0),
		alarms:     make(map[string]*Alarm),
		accountID:  accountID,
		region:     region,
	}
}

// alarmARN builds an ARN for an alarm.
func (s *Store) alarmARN(alarmName string) string {
	return fmt.Sprintf("arn:aws:cloudwatch:%s:%s:alarm:%s", s.region, s.accountID, alarmName)
}

// PutMetricData stores one or more metric data points.
func (s *Store) PutMetricData(namespace string, data []*MetricDatum) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, d := range data {
		d.Namespace = namespace
		if d.Timestamp.IsZero() {
			d.Timestamp = time.Now().UTC()
		}
		s.dataPoints = append(s.dataPoints, d)
	}
}

// ListMetrics returns unique (Namespace, MetricName, Dimensions) combinations.
// If namespace or metricName filters are set, only matching metrics are returned.
func (s *Store) ListMetrics(namespace, metricName string) []*MetricDatum {
	s.mu.RLock()
	defer s.mu.RUnlock()

	type metricKey struct {
		namespace  string
		metricName string
		dimKey     string
	}

	seen := make(map[metricKey]bool)
	result := make([]*MetricDatum, 0)

	for _, dp := range s.dataPoints {
		if namespace != "" && dp.Namespace != namespace {
			continue
		}
		if metricName != "" && dp.MetricName != metricName {
			continue
		}
		key := metricKey{
			namespace:  dp.Namespace,
			metricName: dp.MetricName,
			dimKey:     dimensionKey(dp.Dimensions),
		}
		if !seen[key] {
			seen[key] = true
			result = append(result, dp)
		}
	}
	return result
}

// dimensionKey returns a canonical string representation of a dimension set.
func dimensionKey(dims []Dimension) string {
	if len(dims) == 0 {
		return ""
	}
	key := ""
	for _, d := range dims {
		key += d.Name + "=" + d.Value + ";"
	}
	return key
}

// GetMetricData returns data points matching a query within a time window.
func (s *Store) GetMetricData(namespace, metricName string, dims []Dimension, start, end time.Time) []*MetricDatum {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*MetricDatum, 0)
	wantKey := dimensionKey(dims)

	for _, dp := range s.dataPoints {
		if dp.Namespace != namespace || dp.MetricName != metricName {
			continue
		}
		if wantKey != "" && dimensionKey(dp.Dimensions) != wantKey {
			continue
		}
		if !start.IsZero() && dp.Timestamp.Before(start) {
			continue
		}
		if !end.IsZero() && dp.Timestamp.After(end) {
			continue
		}
		result = append(result, dp)
	}
	return result
}

// PutAlarm creates or replaces an alarm. Returns the alarm.
func (s *Store) PutAlarm(a *Alarm) *Alarm {
	s.mu.Lock()
	defer s.mu.Unlock()

	a.AlarmArn = s.alarmARN(a.AlarmName)
	if a.StateValue == "" {
		a.StateValue = "INSUFFICIENT_DATA"
	}
	if a.Tags == nil {
		a.Tags = make(map[string]string)
	}
	if a.AlarmActions == nil {
		a.AlarmActions = make([]string, 0)
	}
	s.alarms[a.AlarmName] = a
	return a
}

// DescribeAlarms returns alarms, optionally filtered by name.
func (s *Store) DescribeAlarms(names []string) []*Alarm {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(names) == 0 {
		result := make([]*Alarm, 0, len(s.alarms))
		for _, a := range s.alarms {
			result = append(result, a)
		}
		return result
	}

	result := make([]*Alarm, 0, len(names))
	for _, name := range names {
		if a, ok := s.alarms[name]; ok {
			result = append(result, a)
		}
	}
	return result
}

// DeleteAlarms removes alarms by name. Silently ignores non-existent names.
func (s *Store) DeleteAlarms(names []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, name := range names {
		delete(s.alarms, name)
	}
}

// SetAlarmState updates the StateValue and StateReason on an alarm.
// Returns false if the alarm does not exist.
func (s *Store) SetAlarmState(name, stateValue, stateReason string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	a, ok := s.alarms[name]
	if !ok {
		return false
	}
	a.StateValue = stateValue
	a.StateReason = stateReason
	return true
}

// DescribeAlarmsForMetric returns alarms matching a namespace + metric name.
func (s *Store) DescribeAlarmsForMetric(namespace, metricName string) []*Alarm {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Alarm, 0)
	for _, a := range s.alarms {
		if a.Namespace == namespace && a.MetricName == metricName {
			result = append(result, a)
		}
	}
	return result
}

// TagResource sets tags on an alarm identified by its ARN. Returns false if not found.
func (s *Store) TagResource(resourceArn string, tags map[string]string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, a := range s.alarms {
		if a.AlarmArn == resourceArn {
			for k, v := range tags {
				a.Tags[k] = v
			}
			return true
		}
	}
	return false
}

// UntagResource removes tag keys from an alarm. Returns false if alarm not found.
func (s *Store) UntagResource(resourceArn string, tagKeys []string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, a := range s.alarms {
		if a.AlarmArn == resourceArn {
			for _, k := range tagKeys {
				delete(a.Tags, k)
			}
			return true
		}
	}
	return false
}

// ListTagsForResource returns tags for an alarm identified by its ARN.
// The second return value is false if not found.
func (s *Store) ListTagsForResource(resourceArn string) (map[string]string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, a := range s.alarms {
		if a.AlarmArn == resourceArn {
			// Return a copy.
			tags := make(map[string]string, len(a.Tags))
			for k, v := range a.Tags {
				tags[k] = v
			}
			return tags, true
		}
	}
	return nil, false
}
