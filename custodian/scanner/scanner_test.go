package main

import (
	"encoding/json"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newMockGateway returns an httptest.Server that mimics cloudmock's API surface
// for the services used by the compliance rules.
func newMockGateway() *httptest.Server {
	mux := http.NewServeMux()

	// S3: ListBuckets (REST GET /)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Route based on Content-Type / headers.
		ct := r.Header.Get("Content-Type")
		target := r.Header.Get("X-Amz-Target")

		// JSON protocol services (DynamoDB, KMS, SecretsManager).
		if strings.Contains(ct, "amz-json") && target != "" {
			handleJSONService(w, r, target)
			return
		}

		// Query protocol (EC2, RDS).
		if ct == "application/x-www-form-urlencoded" {
			r.ParseForm()
			action := r.FormValue("Action")
			handleQueryAction(w, r, action)
			return
		}

		// REST protocol (S3).
		auth := r.Header.Get("Authorization")
		if r.Method == http.MethodGet && strings.Contains(auth, "/s3/") {
			handleS3ListBuckets(w)
			return
		}

		http.Error(w, "not found", http.StatusNotFound)
	})

	// Lambda: ListFunctions (REST GET /2015-03-31/functions/).
	mux.HandleFunc("/2015-03-31/functions/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"Functions": []map[string]interface{}{
				{"FunctionName": "good-func", "Runtime": "nodejs20.x", "FunctionArn": "arn:aws:lambda:us-east-1:123456789012:function:good-func"},
				{"FunctionName": "old-func", "Runtime": "nodejs14.x", "FunctionArn": "arn:aws:lambda:us-east-1:123456789012:function:old-func"},
			},
		})
	})

	return httptest.NewServer(mux)
}

func handleS3ListBuckets(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/xml")
	w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<ListAllMyBucketsResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Owner><ID>123456789012</ID><DisplayName>cloudmock</DisplayName></Owner>
  <Buckets>
    <Bucket><Name>my-private-bucket</Name><CreationDate>2025-01-01T00:00:00.000Z</CreationDate></Bucket>
    <Bucket><Name>public-assets</Name><CreationDate>2025-01-02T00:00:00.000Z</CreationDate></Bucket>
    <Bucket><Name>logs-bucket</Name><CreationDate>2025-01-03T00:00:00.000Z</CreationDate></Bucket>
  </Buckets>
</ListAllMyBucketsResult>`))
}

func handleJSONService(w http.ResponseWriter, r *http.Request, target string) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case strings.HasSuffix(target, ".ListKeys"):
		json.NewEncoder(w).Encode(map[string]interface{}{
			"Keys": []map[string]interface{}{
				{"KeyId": "key-abc123", "KeyArn": "arn:aws:kms:us-east-1:123456789012:key/key-abc123"},
			},
		})
	case strings.HasSuffix(target, ".ListTables"):
		json.NewEncoder(w).Encode(map[string]interface{}{
			"TableNames": []string{"users", "orders"},
		})
	case strings.HasSuffix(target, ".ListSecrets"):
		json.NewEncoder(w).Encode(map[string]interface{}{
			"SecretList": []map[string]interface{}{
				{"Name": "db-password", "ARN": "arn:aws:secretsmanager:us-east-1:123456789012:secret:db-password", "RotationEnabled": false},
				{"Name": "api-key", "ARN": "arn:aws:secretsmanager:us-east-1:123456789012:secret:api-key", "RotationEnabled": true},
			},
		})
	default:
		json.NewEncoder(w).Encode(map[string]interface{}{})
	}
}

func handleQueryAction(w http.ResponseWriter, r *http.Request, action string) {
	w.Header().Set("Content-Type", "application/xml")

	switch action {
	case "DescribeSecurityGroups":
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<DescribeSecurityGroupsResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
  <requestId>req-1</requestId>
  <securityGroupInfo>
    <item>
      <groupId>sg-default</groupId>
      <groupName>default</groupName>
      <groupDescription>default VPC security group</groupDescription>
      <vpcId>vpc-123</vpcId>
      <ipPermissions/>
      <ipPermissionsEgress>
        <item><ipProtocol>-1</ipProtocol><ipRanges><item><cidrIp>0.0.0.0/0</cidrIp></item></ipRanges></item>
      </ipPermissionsEgress>
    </item>
    <item>
      <groupId>sg-open</groupId>
      <groupName>wide-open</groupName>
      <groupDescription>allows all</groupDescription>
      <vpcId>vpc-123</vpcId>
      <ipPermissions>
        <item><ipProtocol>tcp</ipProtocol><fromPort>22</fromPort><toPort>22</toPort><ipRanges><item><cidrIp>0.0.0.0/0</cidrIp></item></ipRanges></item>
        <item><ipProtocol>tcp</ipProtocol><fromPort>443</fromPort><toPort>443</toPort><ipRanges><item><cidrIp>0.0.0.0/0</cidrIp></item></ipRanges></item>
      </ipPermissions>
    </item>
  </securityGroupInfo>
</DescribeSecurityGroupsResponse>`))

	case "DescribeVpcs":
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<DescribeVpcsResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
  <requestId>req-2</requestId>
  <vpcSet>
    <item><vpcId>vpc-default</vpcId><cidrBlock>172.31.0.0/16</cidrBlock><state>available</state><isDefault>true</isDefault></item>
    <item><vpcId>vpc-custom</vpcId><cidrBlock>10.0.0.0/16</cidrBlock><state>available</state><isDefault>false</isDefault></item>
  </vpcSet>
</DescribeVpcsResponse>`))

	case "DescribeAddresses":
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<DescribeAddressesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
  <requestId>req-3</requestId>
  <addressesSet>
    <item><publicIp>1.2.3.4</publicIp><allocationId>eipalloc-unused</allocationId><associationId/><instanceId/></item>
    <item><publicIp>5.6.7.8</publicIp><allocationId>eipalloc-used</allocationId><associationId>eipassoc-1</associationId><instanceId>i-123</instanceId></item>
  </addressesSet>
</DescribeAddressesResponse>`))

	case "DescribeInstances":
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<DescribeInstancesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
  <requestId>req-4</requestId>
  <reservationSet/>
</DescribeInstancesResponse>`))

	case "DescribeDBInstances":
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<DescribeDBInstancesResponse xmlns="http://rds.amazonaws.com/doc/2014-10-31/">
  <DescribeDBInstancesResult>
    <DBInstances>
      <DBInstance><DBInstanceIdentifier>my-db</DBInstanceIdentifier><PubliclyAccessible>true</PubliclyAccessible></DBInstance>
      <DBInstance><DBInstanceIdentifier>private-db</DBInstanceIdentifier><PubliclyAccessible>false</PubliclyAccessible></DBInstance>
    </DBInstances>
  </DescribeDBInstancesResult>
</DescribeDBInstancesResponse>`))

	default:
		w.Write([]byte(`<ErrorResponse><Error><Code>InvalidAction</Code><Message>Unknown action</Message></Error></ErrorResponse>`))
	}
}

// --- Rule Tests ---

func TestRuleS3PublicAccess(t *testing.T) {
	srv := newMockGateway()
	defer srv.Close()

	rule := ruleS3PublicAccess()
	findings, err := rule.Check(srv.Client(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].ResourceID != "public-assets" {
		t.Errorf("expected resource 'public-assets', got %q", findings[0].ResourceID)
	}
	if findings[0].Severity != "critical" {
		t.Errorf("expected severity 'critical', got %q", findings[0].Severity)
	}
}

func TestRuleSGOpenIngress(t *testing.T) {
	srv := newMockGateway()
	defer srv.Close()

	rule := ruleSGOpenIngress()
	findings, err := rule.Check(srv.Client(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// sg-open has 2 ingress rules with 0.0.0.0/0 (port 22 and 443).
	if len(findings) != 2 {
		t.Fatalf("expected 2 findings, got %d", len(findings))
	}
	for _, f := range findings {
		if f.ResourceID != "sg-open" {
			t.Errorf("expected resource 'sg-open', got %q", f.ResourceID)
		}
	}
}

func TestRuleSGUnrestrictedSSH(t *testing.T) {
	srv := newMockGateway()
	defer srv.Close()

	rule := ruleSGUnrestrictedSSH()
	findings, err := rule.Check(srv.Client(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only port 22 rule on sg-open should match.
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].ResourceID != "sg-open" {
		t.Errorf("expected resource 'sg-open', got %q", findings[0].ResourceID)
	}
	if findings[0].Severity != "critical" {
		t.Errorf("expected severity 'critical', got %q", findings[0].Severity)
	}
}

func TestRuleLambdaRuntime(t *testing.T) {
	srv := newMockGateway()
	defer srv.Close()

	rule := ruleLambdaRuntime()
	findings, err := rule.Check(srv.Client(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only old-func (nodejs14.x) should be flagged.
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].ResourceID != "old-func" {
		t.Errorf("expected resource 'old-func', got %q", findings[0].ResourceID)
	}
}

// --- Report Format Tests ---

func TestRenderJSON(t *testing.T) {
	findings := []Finding{
		{RuleID: "test-rule", ResourceID: "res-1", ResourceType: "AWS::Test::Thing", Severity: "high", Message: "test finding", Remediation: "fix it"},
	}
	rules := []Rule{
		{ID: "test-rule", Name: "Test Rule", Severity: "high"},
		{ID: "passing-rule", Name: "Passing Rule", Severity: "low"},
	}
	report := GenerateReport(findings, rules)

	data, err := RenderJSON(report)
	if err != nil {
		t.Fatalf("RenderJSON failed: %v", err)
	}

	var parsed Report
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if parsed.Summary.TotalRules != 2 {
		t.Errorf("expected 2 total rules, got %d", parsed.Summary.TotalRules)
	}
	if parsed.Summary.Passed != 1 {
		t.Errorf("expected 1 passed, got %d", parsed.Summary.Passed)
	}
	if parsed.Summary.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", parsed.Summary.Failed)
	}
	if parsed.Summary.High != 1 {
		t.Errorf("expected 1 high, got %d", parsed.Summary.High)
	}
}

func TestRenderJUnit(t *testing.T) {
	findings := []Finding{
		{RuleID: "test-rule", ResourceID: "res-1", Severity: "critical", Message: "bad thing"},
	}
	rules := []Rule{
		{ID: "test-rule", Name: "Test Rule", Severity: "critical"},
		{ID: "ok-rule", Name: "OK Rule", Severity: "low"},
	}
	report := GenerateReport(findings, rules)

	data, err := RenderJUnit(report)
	if err != nil {
		t.Fatalf("RenderJUnit failed: %v", err)
	}

	// Verify it's valid XML.
	var suites struct {
		XMLName xml.Name `xml:"testsuites"`
		Suites  []struct {
			Tests    int `xml:"tests,attr"`
			Failures int `xml:"failures,attr"`
		} `xml:"testsuite"`
	}
	if err := xml.Unmarshal(data, &suites); err != nil {
		t.Fatalf("output is not valid JUnit XML: %v", err)
	}
	if len(suites.Suites) != 1 {
		t.Fatalf("expected 1 suite, got %d", len(suites.Suites))
	}
	if suites.Suites[0].Tests != 2 {
		t.Errorf("expected 2 tests, got %d", suites.Suites[0].Tests)
	}
	if suites.Suites[0].Failures != 1 {
		t.Errorf("expected 1 failure, got %d", suites.Suites[0].Failures)
	}
}

func TestRenderHTML(t *testing.T) {
	findings := []Finding{
		{RuleID: "test-rule", ResourceID: "res-1", Severity: "high", Message: "found issue"},
	}
	rules := []Rule{
		{ID: "test-rule", Name: "Test Rule", Severity: "high"},
	}
	report := GenerateReport(findings, rules)

	data, err := RenderHTML(report)
	if err != nil {
		t.Fatalf("RenderHTML failed: %v", err)
	}

	html := string(data)
	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("HTML output missing DOCTYPE")
	}
	if !strings.Contains(html, "cloudmock Compliance Report") {
		t.Error("HTML output missing title")
	}
	if !strings.Contains(html, "test-rule") {
		t.Error("HTML output missing rule ID")
	}
	if !strings.Contains(html, "found issue") {
		t.Error("HTML output missing finding message")
	}
}
