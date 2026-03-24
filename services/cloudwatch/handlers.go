package cloudwatch

import (
	"crypto/rand"
	"encoding/xml"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
)

const cwXmlns = "http://monitoring.amazonaws.com/doc/2010-08-01/"

// ---- shared XML types ----

type xmlResponseMetadata struct {
	RequestID string `xml:"RequestId"`
}

// ---- PutMetricData ----

type xmlPutMetricDataResponse struct {
	XMLName xml.Name            `xml:"PutMetricDataResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handlePutMetricData(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	namespace := form.Get("Namespace")
	if namespace == "" {
		return xmlErr(service.ErrValidation("Namespace is required."))
	}

	data := make([]*MetricDatum, 0)
	for i := 1; ; i++ {
		prefix := fmt.Sprintf("MetricData.member.%d.", i)
		metricName := form.Get(prefix + "MetricName")
		if metricName == "" {
			break
		}

		datum := &MetricDatum{
			MetricName: metricName,
			Unit:       form.Get(prefix + "Unit"),
		}

		if valStr := form.Get(prefix + "Value"); valStr != "" {
			v, err := strconv.ParseFloat(valStr, 64)
			if err != nil {
				return xmlErr(service.ErrValidation(fmt.Sprintf("Invalid Value for metric %s: %s", metricName, valStr)))
			}
			datum.Value = v
		}

		if tsStr := form.Get(prefix + "Timestamp"); tsStr != "" {
			ts, err := time.Parse(time.RFC3339, tsStr)
			if err != nil {
				// Try Unix timestamp
				if unixSec, err2 := strconv.ParseInt(tsStr, 10, 64); err2 == nil {
					ts = time.Unix(unixSec, 0).UTC()
				} else {
					ts = time.Now().UTC()
				}
			}
			datum.Timestamp = ts
		}

		// Parse dimensions: MetricData.member.N.Dimensions.member.M.Name/Value
		for j := 1; ; j++ {
			dimPrefix := fmt.Sprintf("%sDimensions.member.%d.", prefix, j)
			dimName := form.Get(dimPrefix + "Name")
			if dimName == "" {
				break
			}
			datum.Dimensions = append(datum.Dimensions, Dimension{
				Name:  dimName,
				Value: form.Get(dimPrefix + "Value"),
			})
		}

		data = append(data, datum)
	}

	if len(data) == 0 {
		return xmlErr(service.ErrValidation("MetricData is required."))
	}

	store.PutMetricData(namespace, data)

	return xmlOK(&xmlPutMetricDataResponse{
		Xmlns: cwXmlns,
		Meta:  xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- ListMetrics ----

type xmlListMetricsResponse struct {
	XMLName xml.Name              `xml:"ListMetricsResponse"`
	Xmlns   string                `xml:"xmlns,attr"`
	Result  xmlListMetricsResult  `xml:"ListMetricsResult"`
	Meta    xmlResponseMetadata   `xml:"ResponseMetadata"`
}

type xmlListMetricsResult struct {
	Metrics []xmlMetricEntry `xml:"Metrics>member"`
}

type xmlMetricEntry struct {
	Namespace  string           `xml:"Namespace"`
	MetricName string           `xml:"MetricName"`
	Dimensions []xmlDimension   `xml:"Dimensions>member"`
}

type xmlDimension struct {
	Name  string `xml:"Name"`
	Value string `xml:"Value"`
}

func handleListMetrics(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	namespace := form.Get("Namespace")
	metricName := form.Get("MetricName")

	metrics := store.ListMetrics(namespace, metricName)

	entries := make([]xmlMetricEntry, 0, len(metrics))
	for _, m := range metrics {
		dims := make([]xmlDimension, 0, len(m.Dimensions))
		for _, d := range m.Dimensions {
			dims = append(dims, xmlDimension{Name: d.Name, Value: d.Value})
		}
		entries = append(entries, xmlMetricEntry{
			Namespace:  m.Namespace,
			MetricName: m.MetricName,
			Dimensions: dims,
		})
	}

	return xmlOK(&xmlListMetricsResponse{
		Xmlns:  cwXmlns,
		Result: xmlListMetricsResult{Metrics: entries},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- GetMetricData ----

type xmlGetMetricDataResponse struct {
	XMLName xml.Name               `xml:"GetMetricDataResponse"`
	Xmlns   string                 `xml:"xmlns,attr"`
	Result  xmlGetMetricDataResult `xml:"GetMetricDataResult"`
	Meta    xmlResponseMetadata    `xml:"ResponseMetadata"`
}

type xmlGetMetricDataResult struct {
	MetricDataResults []xmlMetricDataResult `xml:"MetricDataResults>member"`
}

type xmlMetricDataResult struct {
	ID         string    `xml:"Id"`
	Label      string    `xml:"Label"`
	StatusCode string    `xml:"StatusCode"`
	Timestamps []string  `xml:"Timestamps>member"`
	Values     []float64 `xml:"Values>member"`
}

func handleGetMetricData(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)

	startTimeStr := form.Get("StartTime")
	endTimeStr := form.Get("EndTime")

	var startTime, endTime time.Time
	if startTimeStr != "" {
		t, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			if unix, err2 := strconv.ParseInt(startTimeStr, 10, 64); err2 == nil {
				t = time.Unix(unix, 0).UTC()
			}
		} else {
			startTime = t
		}
	}
	if endTimeStr != "" {
		t, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			if unix, err2 := strconv.ParseInt(endTimeStr, 10, 64); err2 == nil {
				t = time.Unix(unix, 0).UTC()
			}
		} else {
			endTime = t
		}
	}

	results := make([]xmlMetricDataResult, 0)

	for i := 1; ; i++ {
		prefix := fmt.Sprintf("MetricDataQueries.member.%d.", i)
		queryID := form.Get(prefix + "Id")
		if queryID == "" {
			break
		}

		statPrefix := prefix + "MetricStat."
		metricPrefix := statPrefix + "Metric."

		namespace := form.Get(metricPrefix + "Namespace")
		metricName := form.Get(metricPrefix + "MetricName")

		// Parse query dimensions
		dims := make([]Dimension, 0)
		for j := 1; ; j++ {
			dimPrefix := fmt.Sprintf("%sDimensions.member.%d.", metricPrefix, j)
			dimName := form.Get(dimPrefix + "Name")
			if dimName == "" {
				break
			}
			dims = append(dims, Dimension{
				Name:  dimName,
				Value: form.Get(dimPrefix + "Value"),
			})
		}

		dataPoints := store.GetMetricData(namespace, metricName, dims, startTime, endTime)

		timestamps := make([]string, 0, len(dataPoints))
		values := make([]float64, 0, len(dataPoints))
		for _, dp := range dataPoints {
			timestamps = append(timestamps, dp.Timestamp.UTC().Format(time.RFC3339))
			values = append(values, dp.Value)
		}

		statusCode := "Complete"
		if len(dataPoints) == 0 {
			statusCode = "InternalError"
		}

		results = append(results, xmlMetricDataResult{
			ID:         queryID,
			Label:      metricName,
			StatusCode: statusCode,
			Timestamps: timestamps,
			Values:     values,
		})
	}

	return xmlOK(&xmlGetMetricDataResponse{
		Xmlns:  cwXmlns,
		Result: xmlGetMetricDataResult{MetricDataResults: results},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- PutMetricAlarm ----

type xmlPutMetricAlarmResponse struct {
	XMLName xml.Name            `xml:"PutMetricAlarmResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handlePutMetricAlarm(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	alarmName := form.Get("AlarmName")
	if alarmName == "" {
		return xmlErr(service.ErrValidation("AlarmName is required."))
	}

	threshold, _ := strconv.ParseFloat(form.Get("Threshold"), 64)
	evalPeriods, _ := strconv.Atoi(form.Get("EvaluationPeriods"))
	period, _ := strconv.Atoi(form.Get("Period"))

	// Parse alarm actions: AlarmActions.member.1, AlarmActions.member.2, ...
	alarmActions := make([]string, 0)
	for i := 1; ; i++ {
		action := form.Get(fmt.Sprintf("AlarmActions.member.%d", i))
		if action == "" {
			break
		}
		alarmActions = append(alarmActions, action)
	}

	// Parse dimensions
	dims := make([]Dimension, 0)
	for i := 1; ; i++ {
		dimPrefix := fmt.Sprintf("Dimensions.member.%d.", i)
		dimName := form.Get(dimPrefix + "Name")
		if dimName == "" {
			break
		}
		dims = append(dims, Dimension{
			Name:  dimName,
			Value: form.Get(dimPrefix + "Value"),
		})
	}

	alarm := &Alarm{
		AlarmName:          alarmName,
		Namespace:          form.Get("Namespace"),
		MetricName:         form.Get("MetricName"),
		ComparisonOperator: form.Get("ComparisonOperator"),
		Threshold:          threshold,
		EvaluationPeriods:  evalPeriods,
		Period:             period,
		Statistic:          form.Get("Statistic"),
		AlarmActions:       alarmActions,
		Dimensions:         dims,
	}

	store.PutAlarm(alarm)

	return xmlOK(&xmlPutMetricAlarmResponse{
		Xmlns: cwXmlns,
		Meta:  xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- DescribeAlarms ----

type xmlDescribeAlarmsResponse struct {
	XMLName xml.Name                `xml:"DescribeAlarmsResponse"`
	Xmlns   string                  `xml:"xmlns,attr"`
	Result  xmlDescribeAlarmsResult `xml:"DescribeAlarmsResult"`
	Meta    xmlResponseMetadata     `xml:"ResponseMetadata"`
}

type xmlDescribeAlarmsResult struct {
	MetricAlarms []xmlMetricAlarm `xml:"MetricAlarms>member"`
}

type xmlMetricAlarm struct {
	AlarmName          string         `xml:"AlarmName"`
	AlarmArn           string         `xml:"AlarmArn"`
	Namespace          string         `xml:"Namespace"`
	MetricName         string         `xml:"MetricName"`
	ComparisonOperator string         `xml:"ComparisonOperator"`
	Threshold          float64        `xml:"Threshold"`
	EvaluationPeriods  int            `xml:"EvaluationPeriods"`
	Period             int            `xml:"Period"`
	Statistic          string         `xml:"Statistic"`
	StateValue         string         `xml:"StateValue"`
	StateReason        string         `xml:"StateReason"`
	AlarmActions       []string       `xml:"AlarmActions>member"`
	Dimensions         []xmlDimension `xml:"Dimensions>member"`
}

func handleDescribeAlarms(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)

	names := make([]string, 0)
	for i := 1; ; i++ {
		name := form.Get(fmt.Sprintf("AlarmNames.member.%d", i))
		if name == "" {
			break
		}
		names = append(names, name)
	}

	alarms := store.DescribeAlarms(names)
	return xmlOK(&xmlDescribeAlarmsResponse{
		Xmlns:  cwXmlns,
		Result: xmlDescribeAlarmsResult{MetricAlarms: alarmsToXML(alarms)},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// alarmsToXML converts a slice of *Alarm to xmlMetricAlarm entries.
func alarmsToXML(alarms []*Alarm) []xmlMetricAlarm {
	result := make([]xmlMetricAlarm, 0, len(alarms))
	for _, a := range alarms {
		dims := make([]xmlDimension, 0, len(a.Dimensions))
		for _, d := range a.Dimensions {
			dims = append(dims, xmlDimension{Name: d.Name, Value: d.Value})
		}
		result = append(result, xmlMetricAlarm{
			AlarmName:          a.AlarmName,
			AlarmArn:           a.AlarmArn,
			Namespace:          a.Namespace,
			MetricName:         a.MetricName,
			ComparisonOperator: a.ComparisonOperator,
			Threshold:          a.Threshold,
			EvaluationPeriods:  a.EvaluationPeriods,
			Period:             a.Period,
			Statistic:          a.Statistic,
			StateValue:         a.StateValue,
			StateReason:        a.StateReason,
			AlarmActions:       a.AlarmActions,
			Dimensions:         dims,
		})
	}
	return result
}

// ---- DeleteAlarms ----

type xmlDeleteAlarmsResponse struct {
	XMLName xml.Name            `xml:"DeleteAlarmsResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleDeleteAlarms(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)

	names := make([]string, 0)
	for i := 1; ; i++ {
		name := form.Get(fmt.Sprintf("AlarmNames.member.%d", i))
		if name == "" {
			break
		}
		names = append(names, name)
	}

	store.DeleteAlarms(names)

	return xmlOK(&xmlDeleteAlarmsResponse{
		Xmlns: cwXmlns,
		Meta:  xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- SetAlarmState ----

type xmlSetAlarmStateResponse struct {
	XMLName xml.Name            `xml:"SetAlarmStateResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

// SNSPublisher is an interface for directly publishing messages to SNS topics.
type SNSPublisher interface {
	PublishDirect(topicName, message, subject string) bool
}

func handleSetAlarmState(ctx *service.RequestContext, store *Store, locator ServiceLocator) (*service.Response, error) {
	form := parseForm(ctx)
	alarmName := form.Get("AlarmName")
	stateValue := form.Get("StateValue")
	stateReason := form.Get("StateReason")

	if alarmName == "" {
		return xmlErr(service.ErrValidation("AlarmName is required."))
	}
	if stateValue == "" {
		return xmlErr(service.ErrValidation("StateValue is required."))
	}

	// Get alarm actions before updating state.
	alarms := store.DescribeAlarms([]string{alarmName})
	var alarmActions []string
	var alarmArn string
	if len(alarms) > 0 {
		alarmActions = alarms[0].AlarmActions
		alarmArn = alarms[0].AlarmArn
	}

	if !store.SetAlarmState(alarmName, stateValue, stateReason) {
		return xmlErr(service.NewAWSError("ResourceNotFound",
			fmt.Sprintf("Alarm %s does not exist.", alarmName),
			http.StatusNotFound))
	}

	// If transitioning to ALARM state and there are alarm actions, fire them.
	if stateValue == "ALARM" && locator != nil && len(alarmActions) > 0 {
		deliverAlarmActions(locator, alarmName, alarmArn, stateValue, stateReason, alarmActions)
	}

	return xmlOK(&xmlSetAlarmStateResponse{
		Xmlns: cwXmlns,
		Meta:  xmlResponseMetadata{RequestID: newUUID()},
	})
}

// deliverAlarmActions publishes alarm notifications to SNS topics listed in alarm actions.
func deliverAlarmActions(locator ServiceLocator, alarmName, alarmArn, stateValue, stateReason string, actions []string) {
	svc, err := locator.Lookup("sns")
	if err != nil {
		return
	}
	publisher, ok := svc.(SNSPublisher)
	if !ok {
		return
	}

	// Build alarm notification message matching AWS format.
	message := fmt.Sprintf(
		`{"AlarmName":"%s","AlarmArn":"%s","NewStateValue":"%s","NewStateReason":"%s","StateChangeTime":"%s"}`,
		alarmName, alarmArn, stateValue, stateReason, time.Now().UTC().Format(time.RFC3339))

	for _, actionArn := range actions {
		// Extract topic name from SNS ARN: arn:aws:sns:region:account:topic-name
		topicName := extractTopicNameFromARN(actionArn)
		if topicName == "" {
			continue
		}
		publisher.PublishDirect(topicName, message, fmt.Sprintf("ALARM: %q in ALARM", alarmName))
	}
}

// extractTopicNameFromARN extracts the topic name from an SNS ARN.
func extractTopicNameFromARN(arn string) string {
	// arn:aws:sns:region:account:topic-name
	parts := splitARN(arn)
	if len(parts) < 6 {
		return ""
	}
	return parts[5]
}

// splitARN splits an ARN into its components.
func splitARN(arn string) []string {
	result := make([]string, 0, 6)
	s := arn
	for i := 0; i < 5; i++ {
		idx := -1
		for j := 0; j < len(s); j++ {
			if s[j] == ':' {
				idx = j
				break
			}
		}
		if idx < 0 {
			result = append(result, s)
			return result
		}
		result = append(result, s[:idx])
		s = s[idx+1:]
	}
	result = append(result, s)
	return result
}

// ---- DescribeAlarmsForMetric ----

type xmlDescribeAlarmsForMetricResponse struct {
	XMLName xml.Name                         `xml:"DescribeAlarmsForMetricResponse"`
	Xmlns   string                           `xml:"xmlns,attr"`
	Result  xmlDescribeAlarmsForMetricResult `xml:"DescribeAlarmsForMetricResult"`
	Meta    xmlResponseMetadata              `xml:"ResponseMetadata"`
}

type xmlDescribeAlarmsForMetricResult struct {
	MetricAlarms []xmlMetricAlarm `xml:"MetricAlarms>member"`
}

func handleDescribeAlarmsForMetric(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	namespace := form.Get("Namespace")
	metricName := form.Get("MetricName")

	if namespace == "" {
		return xmlErr(service.ErrValidation("Namespace is required."))
	}
	if metricName == "" {
		return xmlErr(service.ErrValidation("MetricName is required."))
	}

	alarms := store.DescribeAlarmsForMetric(namespace, metricName)

	return xmlOK(&xmlDescribeAlarmsForMetricResponse{
		Xmlns:  cwXmlns,
		Result: xmlDescribeAlarmsForMetricResult{MetricAlarms: alarmsToXML(alarms)},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- TagResource ----

type xmlTagResourceResponse struct {
	XMLName xml.Name            `xml:"TagResourceResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	resourceArn := form.Get("ResourceARN")
	if resourceArn == "" {
		return xmlErr(service.ErrValidation("ResourceARN is required."))
	}

	tags := make(map[string]string)
	for i := 1; ; i++ {
		key := form.Get(fmt.Sprintf("Tags.member.%d.Key", i))
		if key == "" {
			break
		}
		val := form.Get(fmt.Sprintf("Tags.member.%d.Value", i))
		tags[key] = val
	}

	if !store.TagResource(resourceArn, tags) {
		return xmlErr(service.NewAWSError("ResourceNotFound",
			"Resource does not exist.", http.StatusNotFound))
	}

	return xmlOK(&xmlTagResourceResponse{
		Xmlns: cwXmlns,
		Meta:  xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- UntagResource ----

type xmlUntagResourceResponse struct {
	XMLName xml.Name            `xml:"UntagResourceResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	resourceArn := form.Get("ResourceARN")
	if resourceArn == "" {
		return xmlErr(service.ErrValidation("ResourceARN is required."))
	}

	keys := make([]string, 0)
	for i := 1; ; i++ {
		key := form.Get(fmt.Sprintf("TagKeys.member.%d", i))
		if key == "" {
			break
		}
		keys = append(keys, key)
	}

	if !store.UntagResource(resourceArn, keys) {
		return xmlErr(service.NewAWSError("ResourceNotFound",
			"Resource does not exist.", http.StatusNotFound))
	}

	return xmlOK(&xmlUntagResourceResponse{
		Xmlns: cwXmlns,
		Meta:  xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- ListTagsForResource ----

type xmlListTagsForResourceResponse struct {
	XMLName xml.Name                       `xml:"ListTagsForResourceResponse"`
	Xmlns   string                         `xml:"xmlns,attr"`
	Result  xmlListTagsForResourceResult   `xml:"ListTagsForResourceResult"`
	Meta    xmlResponseMetadata            `xml:"ResponseMetadata"`
}

type xmlListTagsForResourceResult struct {
	Tags []xmlTagEntry `xml:"Tags>member"`
}

type xmlTagEntry struct {
	Key   string `xml:"Key"`
	Value string `xml:"Value"`
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	resourceArn := form.Get("ResourceARN")
	if resourceArn == "" {
		return xmlErr(service.ErrValidation("ResourceARN is required."))
	}

	tags, ok := store.ListTagsForResource(resourceArn)
	if !ok {
		return xmlErr(service.NewAWSError("ResourceNotFound",
			"Resource does not exist.", http.StatusNotFound))
	}

	entries := make([]xmlTagEntry, 0, len(tags))
	for k, v := range tags {
		entries = append(entries, xmlTagEntry{Key: k, Value: v})
	}

	return xmlOK(&xmlListTagsForResourceResponse{
		Xmlns:  cwXmlns,
		Result: xmlListTagsForResourceResult{Tags: entries},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- helper functions ----

// xmlOK wraps a response body in a 200 XML response.
func xmlOK(body any) (*service.Response, error) {
	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       body,
		Format:     service.FormatXML,
	}, nil
}

// xmlErr wraps an AWSError in an XML error response.
func xmlErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatXML}, awsErr
}

// newUUID returns a random UUID-shaped identifier.
func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
