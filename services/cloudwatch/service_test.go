package cloudwatch_test

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/gateway"
	"github.com/neureaux/cloudmock/pkg/routing"
	cwsvc "github.com/neureaux/cloudmock/services/cloudwatch"
)


// newCWGateway builds a full gateway stack with the CloudWatch service registered and IAM disabled.
func newCWGateway(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"

	reg := routing.NewRegistry()
	reg.Register(cwsvc.New(cfg.AccountID, cfg.Region))

	return gateway.New(cfg, reg)
}

// cwReq builds a form-encoded POST request targeting the CloudWatch service.
func cwReq(t *testing.T, action string, extra url.Values) *http.Request {
	t.Helper()

	form := url.Values{}
	form.Set("Action", action)
	form.Set("Version", "2010-08-01")
	for k, vs := range extra {
		for _, v := range vs {
			form.Add(k, v)
		}
	}

	body := strings.NewReader(form.Encode())
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/monitoring/aws4_request, SignedHeaders=host, Signature=abc123")
	return req
}

// ---- Test 1: PutMetricData + ListMetrics ----

func TestCW_PutMetricData_ListMetrics(t *testing.T) {
	handler := newCWGateway(t)

	// PutMetricData — two metrics in same namespace
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, cwReq(t, "PutMetricData", url.Values{
		"Namespace":                                    {"AWS/EC2"},
		"MetricData.member.1.MetricName":               {"CPUUtilization"},
		"MetricData.member.1.Value":                    {"42.5"},
		"MetricData.member.1.Unit":                     {"Percent"},
		"MetricData.member.1.Dimensions.member.1.Name": {"InstanceId"},
		"MetricData.member.1.Dimensions.member.1.Value": {"i-12345"},
		"MetricData.member.2.MetricName":               {"NetworkIn"},
		"MetricData.member.2.Value":                    {"1024"},
		"MetricData.member.2.Unit":                     {"Bytes"},
	}))
	if w1.Code != http.StatusOK {
		t.Fatalf("PutMetricData: expected 200, got %d\nbody: %s", w1.Code, w1.Body.String())
	}
	body1 := w1.Body.String()
	if !strings.Contains(body1, "PutMetricDataResponse") {
		t.Errorf("PutMetricData: expected PutMetricDataResponse in body\nbody: %s", body1)
	}

	// PutMetricData — metric in second namespace
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, cwReq(t, "PutMetricData", url.Values{
		"Namespace":                      {"Custom/App"},
		"MetricData.member.1.MetricName": {"RequestCount"},
		"MetricData.member.1.Value":      {"100"},
	}))
	if w2.Code != http.StatusOK {
		t.Fatalf("PutMetricData second: expected 200, got %d\nbody: %s", w2.Code, w2.Body.String())
	}

	// ListMetrics — all
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, cwReq(t, "ListMetrics", nil))
	if wl.Code != http.StatusOK {
		t.Fatalf("ListMetrics: expected 200, got %d\nbody: %s", wl.Code, wl.Body.String())
	}
	lBody := wl.Body.String()
	if !strings.Contains(lBody, "CPUUtilization") {
		t.Errorf("ListMetrics: expected CPUUtilization\nbody: %s", lBody)
	}
	if !strings.Contains(lBody, "NetworkIn") {
		t.Errorf("ListMetrics: expected NetworkIn\nbody: %s", lBody)
	}
	if !strings.Contains(lBody, "RequestCount") {
		t.Errorf("ListMetrics: expected RequestCount\nbody: %s", lBody)
	}

	// ListMetrics — namespace filter
	wlf := httptest.NewRecorder()
	handler.ServeHTTP(wlf, cwReq(t, "ListMetrics", url.Values{"Namespace": {"AWS/EC2"}}))
	if wlf.Code != http.StatusOK {
		t.Fatalf("ListMetrics namespace filter: expected 200, got %d\nbody: %s", wlf.Code, wlf.Body.String())
	}
	lfBody := wlf.Body.String()
	if !strings.Contains(lfBody, "CPUUtilization") {
		t.Errorf("ListMetrics filter: expected CPUUtilization\nbody: %s", lfBody)
	}
	if strings.Contains(lfBody, "RequestCount") {
		t.Errorf("ListMetrics filter: RequestCount should be excluded\nbody: %s", lfBody)
	}

	// ListMetrics — metric name filter
	wlm := httptest.NewRecorder()
	handler.ServeHTTP(wlm, cwReq(t, "ListMetrics", url.Values{"MetricName": {"CPUUtilization"}}))
	if wlm.Code != http.StatusOK {
		t.Fatalf("ListMetrics metric filter: expected 200, got %d\nbody: %s", wlm.Code, wlm.Body.String())
	}
	lmBody := wlm.Body.String()
	if !strings.Contains(lmBody, "CPUUtilization") {
		t.Errorf("ListMetrics metric filter: expected CPUUtilization\nbody: %s", lmBody)
	}
	if strings.Contains(lmBody, "NetworkIn") {
		t.Errorf("ListMetrics metric filter: NetworkIn should be excluded\nbody: %s", lmBody)
	}
}

// ---- Test 2: PutMetricAlarm + DescribeAlarms ----

func TestCW_PutMetricAlarm_DescribeAlarms(t *testing.T) {
	handler := newCWGateway(t)

	// PutMetricAlarm
	wput := httptest.NewRecorder()
	handler.ServeHTTP(wput, cwReq(t, "PutMetricAlarm", url.Values{
		"AlarmName":          {"high-cpu"},
		"Namespace":          {"AWS/EC2"},
		"MetricName":         {"CPUUtilization"},
		"ComparisonOperator": {"GreaterThanThreshold"},
		"Threshold":          {"80"},
		"EvaluationPeriods":  {"2"},
		"Period":             {"300"},
		"Statistic":          {"Average"},
		"AlarmActions.member.1": {"arn:aws:sns:us-east-1:000000000000:alerts"},
	}))
	if wput.Code != http.StatusOK {
		t.Fatalf("PutMetricAlarm: expected 200, got %d\nbody: %s", wput.Code, wput.Body.String())
	}
	if !strings.Contains(wput.Body.String(), "PutMetricAlarmResponse") {
		t.Errorf("PutMetricAlarm: expected PutMetricAlarmResponse\nbody: %s", wput.Body.String())
	}

	// Create a second alarm
	wput2 := httptest.NewRecorder()
	handler.ServeHTTP(wput2, cwReq(t, "PutMetricAlarm", url.Values{
		"AlarmName":          {"high-memory"},
		"Namespace":          {"AWS/EC2"},
		"MetricName":         {"MemoryUtilization"},
		"ComparisonOperator": {"GreaterThanThreshold"},
		"Threshold":          {"90"},
		"EvaluationPeriods":  {"1"},
		"Period":             {"60"},
		"Statistic":          {"Maximum"},
	}))
	if wput2.Code != http.StatusOK {
		t.Fatalf("PutMetricAlarm second: expected 200, got %d\nbody: %s", wput2.Code, wput2.Body.String())
	}

	// DescribeAlarms — all
	wdesc := httptest.NewRecorder()
	handler.ServeHTTP(wdesc, cwReq(t, "DescribeAlarms", nil))
	if wdesc.Code != http.StatusOK {
		t.Fatalf("DescribeAlarms: expected 200, got %d\nbody: %s", wdesc.Code, wdesc.Body.String())
	}
	descBody := wdesc.Body.String()
	if !strings.Contains(descBody, "high-cpu") {
		t.Errorf("DescribeAlarms: expected high-cpu\nbody: %s", descBody)
	}
	if !strings.Contains(descBody, "high-memory") {
		t.Errorf("DescribeAlarms: expected high-memory\nbody: %s", descBody)
	}
	if !strings.Contains(descBody, "INSUFFICIENT_DATA") {
		t.Errorf("DescribeAlarms: expected default state INSUFFICIENT_DATA\nbody: %s", descBody)
	}
	if !strings.Contains(descBody, "arn:aws:cloudwatch:") {
		t.Errorf("DescribeAlarms: expected AlarmArn in response\nbody: %s", descBody)
	}

	// DescribeAlarms — by name filter
	wname := httptest.NewRecorder()
	handler.ServeHTTP(wname, cwReq(t, "DescribeAlarms", url.Values{
		"AlarmNames.member.1": {"high-cpu"},
	}))
	if wname.Code != http.StatusOK {
		t.Fatalf("DescribeAlarms by name: expected 200, got %d\nbody: %s", wname.Code, wname.Body.String())
	}
	nameBody := wname.Body.String()
	if !strings.Contains(nameBody, "high-cpu") {
		t.Errorf("DescribeAlarms by name: expected high-cpu\nbody: %s", nameBody)
	}
	if strings.Contains(nameBody, "high-memory") {
		t.Errorf("DescribeAlarms by name: high-memory should be excluded\nbody: %s", nameBody)
	}

	// Verify alarm ARN format
	var resp struct {
		Result struct {
			Alarms []struct {
				AlarmArn string `xml:"AlarmArn"`
			} `xml:"MetricAlarms>member"`
		} `xml:"DescribeAlarmsResult"`
	}
	if err := xml.Unmarshal([]byte(nameBody), &resp); err != nil {
		t.Fatalf("DescribeAlarms: unmarshal: %v\nbody: %s", err, nameBody)
	}
	if len(resp.Result.Alarms) == 0 {
		t.Fatal("DescribeAlarms: no alarms in response")
	}
	arn := resp.Result.Alarms[0].AlarmArn
	if !strings.HasPrefix(arn, "arn:aws:cloudwatch:") {
		t.Errorf("DescribeAlarms: unexpected ARN format: %s", arn)
	}
	if !strings.Contains(arn, "high-cpu") {
		t.Errorf("DescribeAlarms: ARN should contain alarm name, got: %s", arn)
	}
}

// ---- Test 3: DeleteAlarms ----

func TestCW_DeleteAlarms(t *testing.T) {
	handler := newCWGateway(t)

	// Create two alarms
	for _, name := range []string{"alarm-to-delete", "alarm-to-keep"} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, cwReq(t, "PutMetricAlarm", url.Values{
			"AlarmName":          {name},
			"Namespace":          {"Test/NS"},
			"MetricName":         {"TestMetric"},
			"ComparisonOperator": {"GreaterThanThreshold"},
			"Threshold":          {"10"},
			"EvaluationPeriods":  {"1"},
			"Period":             {"60"},
			"Statistic":          {"Sum"},
		}))
		if w.Code != http.StatusOK {
			t.Fatalf("PutMetricAlarm %s: expected 200, got %d", name, w.Code)
		}
	}

	// Verify both exist
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, cwReq(t, "DescribeAlarms", nil))
	if !strings.Contains(wl.Body.String(), "alarm-to-delete") || !strings.Contains(wl.Body.String(), "alarm-to-keep") {
		t.Fatalf("setup: both alarms should exist\nbody: %s", wl.Body.String())
	}

	// DeleteAlarms
	wdel := httptest.NewRecorder()
	handler.ServeHTTP(wdel, cwReq(t, "DeleteAlarms", url.Values{
		"AlarmNames.member.1": {"alarm-to-delete"},
	}))
	if wdel.Code != http.StatusOK {
		t.Fatalf("DeleteAlarms: expected 200, got %d\nbody: %s", wdel.Code, wdel.Body.String())
	}
	if !strings.Contains(wdel.Body.String(), "DeleteAlarmsResponse") {
		t.Errorf("DeleteAlarms: expected DeleteAlarmsResponse\nbody: %s", wdel.Body.String())
	}

	// Verify deleted alarm gone and other still present
	wl2 := httptest.NewRecorder()
	handler.ServeHTTP(wl2, cwReq(t, "DescribeAlarms", nil))
	if strings.Contains(wl2.Body.String(), "alarm-to-delete") {
		t.Errorf("DeleteAlarms: alarm-to-delete should be gone\nbody: %s", wl2.Body.String())
	}
	if !strings.Contains(wl2.Body.String(), "alarm-to-keep") {
		t.Errorf("DeleteAlarms: alarm-to-keep should still exist\nbody: %s", wl2.Body.String())
	}

	// DeleteAlarms is idempotent — deleting non-existent alarm should succeed
	wdel2 := httptest.NewRecorder()
	handler.ServeHTTP(wdel2, cwReq(t, "DeleteAlarms", url.Values{
		"AlarmNames.member.1": {"alarm-to-delete"},
	}))
	if wdel2.Code != http.StatusOK {
		t.Errorf("DeleteAlarms idempotent: expected 200, got %d\nbody: %s", wdel2.Code, wdel2.Body.String())
	}
}

// ---- Test 4: SetAlarmState ----

func TestCW_SetAlarmState(t *testing.T) {
	handler := newCWGateway(t)

	// Create alarm
	wput := httptest.NewRecorder()
	handler.ServeHTTP(wput, cwReq(t, "PutMetricAlarm", url.Values{
		"AlarmName":          {"state-alarm"},
		"Namespace":          {"AWS/EC2"},
		"MetricName":         {"CPUUtilization"},
		"ComparisonOperator": {"GreaterThanThreshold"},
		"Threshold":          {"80"},
		"EvaluationPeriods":  {"1"},
		"Period":             {"60"},
		"Statistic":          {"Average"},
	}))
	if wput.Code != http.StatusOK {
		t.Fatalf("PutMetricAlarm: expected 200, got %d", wput.Code)
	}

	// Verify initial state
	wdesc1 := httptest.NewRecorder()
	handler.ServeHTTP(wdesc1, cwReq(t, "DescribeAlarms", url.Values{
		"AlarmNames.member.1": {"state-alarm"},
	}))
	if !strings.Contains(wdesc1.Body.String(), "INSUFFICIENT_DATA") {
		t.Errorf("initial state: expected INSUFFICIENT_DATA\nbody: %s", wdesc1.Body.String())
	}

	// SetAlarmState to ALARM
	wset := httptest.NewRecorder()
	handler.ServeHTTP(wset, cwReq(t, "SetAlarmState", url.Values{
		"AlarmName":   {"state-alarm"},
		"StateValue":  {"ALARM"},
		"StateReason": {"CPU spike detected"},
	}))
	if wset.Code != http.StatusOK {
		t.Fatalf("SetAlarmState: expected 200, got %d\nbody: %s", wset.Code, wset.Body.String())
	}
	if !strings.Contains(wset.Body.String(), "SetAlarmStateResponse") {
		t.Errorf("SetAlarmState: expected SetAlarmStateResponse\nbody: %s", wset.Body.String())
	}

	// Verify new state
	wdesc2 := httptest.NewRecorder()
	handler.ServeHTTP(wdesc2, cwReq(t, "DescribeAlarms", url.Values{
		"AlarmNames.member.1": {"state-alarm"},
	}))
	descBody := wdesc2.Body.String()
	if !strings.Contains(descBody, "ALARM") {
		t.Errorf("SetAlarmState: expected ALARM state\nbody: %s", descBody)
	}
	if !strings.Contains(descBody, "CPU spike detected") {
		t.Errorf("SetAlarmState: expected StateReason in response\nbody: %s", descBody)
	}

	// SetAlarmState to OK
	wsetOK := httptest.NewRecorder()
	handler.ServeHTTP(wsetOK, cwReq(t, "SetAlarmState", url.Values{
		"AlarmName":   {"state-alarm"},
		"StateValue":  {"OK"},
		"StateReason": {"Resolved"},
	}))
	if wsetOK.Code != http.StatusOK {
		t.Fatalf("SetAlarmState OK: expected 200, got %d", wsetOK.Code)
	}

	// SetAlarmState on non-existent alarm
	wne := httptest.NewRecorder()
	handler.ServeHTTP(wne, cwReq(t, "SetAlarmState", url.Values{
		"AlarmName":   {"no-such-alarm"},
		"StateValue":  {"OK"},
		"StateReason": {"test"},
	}))
	if wne.Code != http.StatusNotFound {
		t.Errorf("SetAlarmState non-existent: expected 404, got %d\nbody: %s", wne.Code, wne.Body.String())
	}
}

// ---- Test 5: GetMetricData ----

func TestCW_GetMetricData(t *testing.T) {
	handler := newCWGateway(t)

	// Put some metric data points
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, cwReq(t, "PutMetricData", url.Values{
		"Namespace":                      {"AWS/Lambda"},
		"MetricData.member.1.MetricName": {"Invocations"},
		"MetricData.member.1.Value":      {"50"},
		"MetricData.member.1.Unit":       {"Count"},
		"MetricData.member.2.MetricName": {"Errors"},
		"MetricData.member.2.Value":      {"2"},
		"MetricData.member.2.Unit":       {"Count"},
	}))
	if w1.Code != http.StatusOK {
		t.Fatalf("PutMetricData: expected 200, got %d\nbody: %s", w1.Code, w1.Body.String())
	}

	// GetMetricData — query for Invocations
	wget := httptest.NewRecorder()
	handler.ServeHTTP(wget, cwReq(t, "GetMetricData", url.Values{
		"StartTime": {"2000-01-01T00:00:00Z"},
		"EndTime":   {"2099-12-31T23:59:59Z"},
		"MetricDataQueries.member.1.Id":                                    {"m1"},
		"MetricDataQueries.member.1.MetricStat.Metric.Namespace":           {"AWS/Lambda"},
		"MetricDataQueries.member.1.MetricStat.Metric.MetricName":          {"Invocations"},
		"MetricDataQueries.member.1.MetricStat.Period":                     {"60"},
		"MetricDataQueries.member.1.MetricStat.Stat":                       {"Sum"},
	}))
	if wget.Code != http.StatusOK {
		t.Fatalf("GetMetricData: expected 200, got %d\nbody: %s", wget.Code, wget.Body.String())
	}
	getBody := wget.Body.String()
	if !strings.Contains(getBody, "GetMetricDataResponse") {
		t.Errorf("GetMetricData: expected GetMetricDataResponse\nbody: %s", getBody)
	}
	if !strings.Contains(getBody, "m1") {
		t.Errorf("GetMetricData: expected query ID m1 in response\nbody: %s", getBody)
	}
	if !strings.Contains(getBody, "50") {
		t.Errorf("GetMetricData: expected value 50 in response\nbody: %s", getBody)
	}

	// GetMetricData — no data for query (wrong metric name)
	wempty := httptest.NewRecorder()
	handler.ServeHTTP(wempty, cwReq(t, "GetMetricData", url.Values{
		"StartTime": {"2000-01-01T00:00:00Z"},
		"EndTime":   {"2099-12-31T23:59:59Z"},
		"MetricDataQueries.member.1.Id":                           {"m2"},
		"MetricDataQueries.member.1.MetricStat.Metric.Namespace":  {"AWS/Lambda"},
		"MetricDataQueries.member.1.MetricStat.Metric.MetricName": {"NoSuchMetric"},
		"MetricDataQueries.member.1.MetricStat.Period":            {"60"},
		"MetricDataQueries.member.1.MetricStat.Stat":              {"Sum"},
	}))
	if wempty.Code != http.StatusOK {
		t.Fatalf("GetMetricData empty: expected 200, got %d\nbody: %s", wempty.Code, wempty.Body.String())
	}
	if !strings.Contains(wempty.Body.String(), "m2") {
		t.Errorf("GetMetricData empty: expected query ID m2 in response\nbody: %s", wempty.Body.String())
	}
}

// ---- Test 6: TagResource on alarm ----

func TestCW_TagResource(t *testing.T) {
	handler := newCWGateway(t)

	// Create alarm
	wput := httptest.NewRecorder()
	handler.ServeHTTP(wput, cwReq(t, "PutMetricAlarm", url.Values{
		"AlarmName":          {"tag-alarm"},
		"Namespace":          {"AWS/EC2"},
		"MetricName":         {"CPUUtilization"},
		"ComparisonOperator": {"GreaterThanThreshold"},
		"Threshold":          {"80"},
		"EvaluationPeriods":  {"1"},
		"Period":             {"60"},
		"Statistic":          {"Average"},
	}))
	if wput.Code != http.StatusOK {
		t.Fatalf("PutMetricAlarm: expected 200, got %d", wput.Code)
	}

	// Get the alarm ARN
	wdesc := httptest.NewRecorder()
	handler.ServeHTTP(wdesc, cwReq(t, "DescribeAlarms", url.Values{
		"AlarmNames.member.1": {"tag-alarm"},
	}))
	var descResp struct {
		Result struct {
			Alarms []struct {
				AlarmArn string `xml:"AlarmArn"`
			} `xml:"MetricAlarms>member"`
		} `xml:"DescribeAlarmsResult"`
	}
	if err := xml.Unmarshal(wdesc.Body.Bytes(), &descResp); err != nil {
		t.Fatalf("DescribeAlarms: unmarshal: %v", err)
	}
	if len(descResp.Result.Alarms) == 0 {
		t.Fatal("DescribeAlarms: no alarms returned")
	}
	alarmArn := descResp.Result.Alarms[0].AlarmArn
	if alarmArn == "" {
		t.Fatal("DescribeAlarms: AlarmArn is empty")
	}

	// TagResource
	wtag := httptest.NewRecorder()
	handler.ServeHTTP(wtag, cwReq(t, "TagResource", url.Values{
		"ResourceARN":         {alarmArn},
		"Tags.member.1.Key":   {"env"},
		"Tags.member.1.Value": {"production"},
		"Tags.member.2.Key":   {"team"},
		"Tags.member.2.Value": {"platform"},
	}))
	if wtag.Code != http.StatusOK {
		t.Fatalf("TagResource: expected 200, got %d\nbody: %s", wtag.Code, wtag.Body.String())
	}
	if !strings.Contains(wtag.Body.String(), "TagResourceResponse") {
		t.Errorf("TagResource: expected TagResourceResponse\nbody: %s", wtag.Body.String())
	}

	// ListTagsForResource
	wlist := httptest.NewRecorder()
	handler.ServeHTTP(wlist, cwReq(t, "ListTagsForResource", url.Values{
		"ResourceARN": {alarmArn},
	}))
	if wlist.Code != http.StatusOK {
		t.Fatalf("ListTagsForResource: expected 200, got %d\nbody: %s", wlist.Code, wlist.Body.String())
	}
	listBody := wlist.Body.String()
	if !strings.Contains(listBody, "env") || !strings.Contains(listBody, "production") {
		t.Errorf("ListTagsForResource: expected env=production\nbody: %s", listBody)
	}
	if !strings.Contains(listBody, "team") || !strings.Contains(listBody, "platform") {
		t.Errorf("ListTagsForResource: expected team=platform\nbody: %s", listBody)
	}

	// UntagResource
	wuntag := httptest.NewRecorder()
	handler.ServeHTTP(wuntag, cwReq(t, "UntagResource", url.Values{
		"ResourceARN":      {alarmArn},
		"TagKeys.member.1": {"env"},
	}))
	if wuntag.Code != http.StatusOK {
		t.Fatalf("UntagResource: expected 200, got %d\nbody: %s", wuntag.Code, wuntag.Body.String())
	}

	// Verify tag removed
	wlist2 := httptest.NewRecorder()
	handler.ServeHTTP(wlist2, cwReq(t, "ListTagsForResource", url.Values{
		"ResourceARN": {alarmArn},
	}))
	if wlist2.Code != http.StatusOK {
		t.Fatalf("ListTagsForResource after untag: expected 200, got %d", wlist2.Code)
	}
	list2Body := wlist2.Body.String()
	if strings.Contains(list2Body, ">env<") {
		t.Errorf("UntagResource: env tag should be gone\nbody: %s", list2Body)
	}
	if !strings.Contains(list2Body, "team") {
		t.Errorf("UntagResource: team tag should still be present\nbody: %s", list2Body)
	}

	// TagResource on non-existent ARN
	wne := httptest.NewRecorder()
	handler.ServeHTTP(wne, cwReq(t, "TagResource", url.Values{
		"ResourceARN":         {"arn:aws:cloudwatch:us-east-1:000000000000:alarm:no-such-alarm"},
		"Tags.member.1.Key":   {"k"},
		"Tags.member.1.Value": {"v"},
	}))
	if wne.Code != http.StatusNotFound {
		t.Errorf("TagResource non-existent: expected 404, got %d\nbody: %s", wne.Code, wne.Body.String())
	}
}

// ---- Test 7: DescribeAlarmsForMetric ----

func TestCW_DescribeAlarmsForMetric(t *testing.T) {
	handler := newCWGateway(t)

	// Create alarms for different metrics
	for _, tc := range []struct{ name, ns, metric string }{
		{"cpu-alarm-1", "AWS/EC2", "CPUUtilization"},
		{"cpu-alarm-2", "AWS/EC2", "CPUUtilization"},
		{"net-alarm", "AWS/EC2", "NetworkIn"},
	} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, cwReq(t, "PutMetricAlarm", url.Values{
			"AlarmName":          {tc.name},
			"Namespace":          {tc.ns},
			"MetricName":         {tc.metric},
			"ComparisonOperator": {"GreaterThanThreshold"},
			"Threshold":          {"10"},
			"EvaluationPeriods":  {"1"},
			"Period":             {"60"},
			"Statistic":          {"Average"},
		}))
		if w.Code != http.StatusOK {
			t.Fatalf("PutMetricAlarm %s: expected 200, got %d", tc.name, w.Code)
		}
	}

	// DescribeAlarmsForMetric — CPUUtilization
	wdesc := httptest.NewRecorder()
	handler.ServeHTTP(wdesc, cwReq(t, "DescribeAlarmsForMetric", url.Values{
		"Namespace":  {"AWS/EC2"},
		"MetricName": {"CPUUtilization"},
	}))
	if wdesc.Code != http.StatusOK {
		t.Fatalf("DescribeAlarmsForMetric: expected 200, got %d\nbody: %s", wdesc.Code, wdesc.Body.String())
	}
	descBody := wdesc.Body.String()
	if !strings.Contains(descBody, "cpu-alarm-1") {
		t.Errorf("DescribeAlarmsForMetric: expected cpu-alarm-1\nbody: %s", descBody)
	}
	if !strings.Contains(descBody, "cpu-alarm-2") {
		t.Errorf("DescribeAlarmsForMetric: expected cpu-alarm-2\nbody: %s", descBody)
	}
	if strings.Contains(descBody, "net-alarm") {
		t.Errorf("DescribeAlarmsForMetric: net-alarm should be excluded\nbody: %s", descBody)
	}
}

// ---- Test 8: PutMetricData — missing namespace returns error ----

func TestCW_PutMetricData_MissingNamespace(t *testing.T) {
	handler := newCWGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, cwReq(t, "PutMetricData", url.Values{
		"MetricData.member.1.MetricName": {"CPUUtilization"},
		"MetricData.member.1.Value":      {"50"},
	}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("PutMetricData no namespace: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
}
