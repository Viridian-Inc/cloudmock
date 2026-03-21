package ec2_test

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
	ec2svc "github.com/neureaux/cloudmock/services/ec2"
)

// newEC2Gateway builds a full gateway stack with the EC2 service registered and IAM disabled.
func newEC2Gateway(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"

	reg := routing.NewRegistry()
	reg.Register(ec2svc.New(cfg.AccountID, cfg.Region))

	return gateway.New(cfg, reg)
}

// ec2Req builds a form-encoded POST request targeting the EC2 service.
func ec2Req(t *testing.T, action string, extra url.Values) *http.Request {
	t.Helper()

	form := url.Values{}
	form.Set("Action", action)
	form.Set("Version", "2016-11-15")
	for k, vs := range extra {
		for _, v := range vs {
			form.Add(k, v)
		}
	}

	body := strings.NewReader(form.Encode())
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/ec2/aws4_request, SignedHeaders=host, Signature=abc123")
	return req
}

// mustCreateVpc creates a VPC and returns its VpcId.
func mustCreateVpc(t *testing.T, handler http.Handler, cidr string) string {
	t.Helper()
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ec2Req(t, "CreateVpc", url.Values{
		"CidrBlock": {cidr},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateVpc %s: expected 200, got %d\nbody: %s", cidr, w.Code, w.Body.String())
	}
	var resp struct {
		Vpc struct {
			VpcId string `xml:"vpcId"`
		} `xml:"vpc"`
	}
	if err := xml.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("CreateVpc: unmarshal: %v\nbody: %s", err, w.Body.String())
	}
	if resp.Vpc.VpcId == "" {
		t.Fatalf("CreateVpc: vpcId is empty\nbody: %s", w.Body.String())
	}
	return resp.Vpc.VpcId
}

// mustCreateSubnet creates a subnet and returns its SubnetId.
func mustCreateSubnet(t *testing.T, handler http.Handler, vpcId, cidr string) string {
	t.Helper()
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ec2Req(t, "CreateSubnet", url.Values{
		"VpcId":     {vpcId},
		"CidrBlock": {cidr},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateSubnet %s in %s: expected 200, got %d\nbody: %s", cidr, vpcId, w.Code, w.Body.String())
	}
	var resp struct {
		Subnet struct {
			SubnetId string `xml:"subnetId"`
		} `xml:"subnet"`
	}
	if err := xml.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("CreateSubnet: unmarshal: %v\nbody: %s", err, w.Body.String())
	}
	if resp.Subnet.SubnetId == "" {
		t.Fatalf("CreateSubnet: subnetId is empty\nbody: %s", w.Body.String())
	}
	return resp.Subnet.SubnetId
}

// ---- Test 1: CreateVpc + DescribeVpcs ----

func TestEC2_CreateAndDescribeVpcs(t *testing.T) {
	handler := newEC2Gateway(t)

	// Create two VPCs.
	vpcId1 := mustCreateVpc(t, handler, "10.0.0.0/16")
	vpcId2 := mustCreateVpc(t, handler, "172.16.0.0/16")

	if !strings.HasPrefix(vpcId1, "vpc-") {
		t.Errorf("CreateVpc: expected vpc- prefix, got %s", vpcId1)
	}
	if !strings.HasPrefix(vpcId2, "vpc-") {
		t.Errorf("CreateVpc: expected vpc- prefix, got %s", vpcId2)
	}

	// Verify CreateVpc response fields.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ec2Req(t, "CreateVpc", url.Values{
		"CidrBlock": {"192.168.0.0/16"},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateVpc: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "<state>available</state>") {
		t.Errorf("CreateVpc: expected state=available\nbody: %s", body)
	}
	if !strings.Contains(body, "<isDefault>false</isDefault>") {
		t.Errorf("CreateVpc: expected isDefault=false\nbody: %s", body)
	}
	if !strings.Contains(body, "<ownerId>000000000000</ownerId>") {
		t.Errorf("CreateVpc: expected ownerId\nbody: %s", body)
	}
	if !strings.Contains(body, "dopt-") {
		t.Errorf("CreateVpc: expected dhcpOptionsId\nbody: %s", body)
	}

	// DescribeVpcs — all (should have 3 VPCs now).
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, ec2Req(t, "DescribeVpcs", nil))
	if wd.Code != http.StatusOK {
		t.Fatalf("DescribeVpcs: expected 200, got %d\nbody: %s", wd.Code, wd.Body.String())
	}
	descBody := wd.Body.String()
	if !strings.Contains(descBody, vpcId1) {
		t.Errorf("DescribeVpcs: expected %s\nbody: %s", vpcId1, descBody)
	}
	if !strings.Contains(descBody, vpcId2) {
		t.Errorf("DescribeVpcs: expected %s\nbody: %s", vpcId2, descBody)
	}

	// DescribeVpcs — by ID.
	wf := httptest.NewRecorder()
	handler.ServeHTTP(wf, ec2Req(t, "DescribeVpcs", url.Values{
		"VpcId.1": {vpcId1},
	}))
	if wf.Code != http.StatusOK {
		t.Fatalf("DescribeVpcs by ID: expected 200, got %d\nbody: %s", wf.Code, wf.Body.String())
	}
	filterBody := wf.Body.String()
	if !strings.Contains(filterBody, vpcId1) {
		t.Errorf("DescribeVpcs filter: expected %s\nbody: %s", vpcId1, filterBody)
	}
	if strings.Contains(filterBody, vpcId2) {
		t.Errorf("DescribeVpcs filter: %s should be excluded\nbody: %s", vpcId2, filterBody)
	}
}

// ---- Test 2: CreateSubnet + DescribeSubnets ----

func TestEC2_CreateAndDescribeSubnets(t *testing.T) {
	handler := newEC2Gateway(t)

	vpcId := mustCreateVpc(t, handler, "10.0.0.0/16")

	// Create two subnets.
	subId1 := mustCreateSubnet(t, handler, vpcId, "10.0.1.0/24")
	subId2 := mustCreateSubnet(t, handler, vpcId, "10.0.2.0/24")

	if !strings.HasPrefix(subId1, "subnet-") {
		t.Errorf("CreateSubnet: expected subnet- prefix, got %s", subId1)
	}

	// Verify CreateSubnet response fields.
	ws := httptest.NewRecorder()
	handler.ServeHTTP(ws, ec2Req(t, "CreateSubnet", url.Values{
		"VpcId":            {vpcId},
		"CidrBlock":        {"10.0.3.0/24"},
		"AvailabilityZone": {"us-east-1b"},
	}))
	if ws.Code != http.StatusOK {
		t.Fatalf("CreateSubnet: expected 200, got %d\nbody: %s", ws.Code, ws.Body.String())
	}
	subBody := ws.Body.String()
	if !strings.Contains(subBody, "<state>available</state>") {
		t.Errorf("CreateSubnet: expected state=available\nbody: %s", subBody)
	}
	if !strings.Contains(subBody, "<availabilityZone>us-east-1b</availabilityZone>") {
		t.Errorf("CreateSubnet: expected AZ us-east-1b\nbody: %s", subBody)
	}
	if !strings.Contains(subBody, vpcId) {
		t.Errorf("CreateSubnet: expected vpcId in response\nbody: %s", subBody)
	}
	// /24 = 256 IPs - 5 reserved = 251
	if !strings.Contains(subBody, "<availableIpAddressCount>251</availableIpAddressCount>") {
		t.Errorf("CreateSubnet: expected 251 available IPs\nbody: %s", subBody)
	}

	// DescribeSubnets — all.
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, ec2Req(t, "DescribeSubnets", nil))
	if wd.Code != http.StatusOK {
		t.Fatalf("DescribeSubnets: expected 200, got %d\nbody: %s", wd.Code, wd.Body.String())
	}
	descBody := wd.Body.String()
	if !strings.Contains(descBody, subId1) {
		t.Errorf("DescribeSubnets: expected %s\nbody: %s", subId1, descBody)
	}
	if !strings.Contains(descBody, subId2) {
		t.Errorf("DescribeSubnets: expected %s\nbody: %s", subId2, descBody)
	}

	// DescribeSubnets — by SubnetId.
	wf := httptest.NewRecorder()
	handler.ServeHTTP(wf, ec2Req(t, "DescribeSubnets", url.Values{
		"SubnetId.1": {subId1},
	}))
	if wf.Code != http.StatusOK {
		t.Fatalf("DescribeSubnets by ID: expected 200, got %d\nbody: %s", wf.Code, wf.Body.String())
	}
	filterBody := wf.Body.String()
	if !strings.Contains(filterBody, subId1) {
		t.Errorf("DescribeSubnets filter: expected %s\nbody: %s", subId1, filterBody)
	}
	if strings.Contains(filterBody, subId2) {
		t.Errorf("DescribeSubnets filter: %s should be excluded\nbody: %s", subId2, filterBody)
	}

	// DescribeSubnets — filter by vpc-id.
	wvf := httptest.NewRecorder()
	handler.ServeHTTP(wvf, ec2Req(t, "DescribeSubnets", url.Values{
		"Filter.1.Name":    {"vpc-id"},
		"Filter.1.Value.1": {vpcId},
	}))
	if wvf.Code != http.StatusOK {
		t.Fatalf("DescribeSubnets by vpc-id: expected 200, got %d\nbody: %s", wvf.Code, wvf.Body.String())
	}
	vpcFilterBody := wvf.Body.String()
	if !strings.Contains(vpcFilterBody, subId1) {
		t.Errorf("DescribeSubnets vpc filter: expected %s\nbody: %s", subId1, vpcFilterBody)
	}

	// Default AZ should be us-east-1a when not specified.
	var descResp struct {
		SubnetSet []struct {
			AvailabilityZone string `xml:"availabilityZone"`
		} `xml:"subnetSet>item"`
	}
	wAz := httptest.NewRecorder()
	handler.ServeHTTP(wAz, ec2Req(t, "DescribeSubnets", url.Values{
		"SubnetId.1": {subId1},
	}))
	if err := xml.Unmarshal(wAz.Body.Bytes(), &descResp); err != nil {
		t.Fatalf("DescribeSubnets unmarshal: %v", err)
	}
	if len(descResp.SubnetSet) != 1 {
		t.Fatalf("DescribeSubnets: expected 1 subnet, got %d", len(descResp.SubnetSet))
	}
	if descResp.SubnetSet[0].AvailabilityZone != "us-east-1a" {
		t.Errorf("CreateSubnet default AZ: expected us-east-1a, got %s", descResp.SubnetSet[0].AvailabilityZone)
	}
}

// ---- Test 3: Subnet CIDR validation ----

func TestEC2_SubnetCIDRValidation(t *testing.T) {
	handler := newEC2Gateway(t)

	vpcId := mustCreateVpc(t, handler, "10.0.0.0/16")

	// Subnet CIDR outside VPC CIDR should fail.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ec2Req(t, "CreateSubnet", url.Values{
		"VpcId":     {vpcId},
		"CidrBlock": {"172.16.0.0/24"},
	}))
	if w.Code == http.StatusOK {
		t.Error("CreateSubnet with out-of-range CIDR: expected error, got 200")
	}
	if !strings.Contains(w.Body.String(), "InvalidSubnet.Range") {
		t.Errorf("CreateSubnet out-of-range: expected InvalidSubnet.Range error\nbody: %s", w.Body.String())
	}

	// Invalid CIDR syntax should fail.
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, ec2Req(t, "CreateSubnet", url.Values{
		"VpcId":     {vpcId},
		"CidrBlock": {"not-a-cidr"},
	}))
	if w2.Code == http.StatusOK {
		t.Error("CreateSubnet with invalid CIDR: expected error, got 200")
	}

	// Subnet for non-existent VPC should fail.
	w3 := httptest.NewRecorder()
	handler.ServeHTTP(w3, ec2Req(t, "CreateSubnet", url.Values{
		"VpcId":     {"vpc-nonexistent12345678"},
		"CidrBlock": {"10.0.1.0/24"},
	}))
	if w3.Code == http.StatusOK {
		t.Error("CreateSubnet with non-existent VPC: expected error, got 200")
	}
	if !strings.Contains(w3.Body.String(), "InvalidVpcID.NotFound") {
		t.Errorf("CreateSubnet non-existent VPC: expected InvalidVpcID.NotFound\nbody: %s", w3.Body.String())
	}

	// Valid subnet within VPC should succeed.
	w4 := httptest.NewRecorder()
	handler.ServeHTTP(w4, ec2Req(t, "CreateSubnet", url.Values{
		"VpcId":     {vpcId},
		"CidrBlock": {"10.0.1.0/24"},
	}))
	if w4.Code != http.StatusOK {
		t.Errorf("CreateSubnet valid CIDR: expected 200, got %d\nbody: %s", w4.Code, w4.Body.String())
	}
}

// ---- Test 4: DeleteVpc fails if subnets exist ----

func TestEC2_DeleteVpcDependencyViolation(t *testing.T) {
	handler := newEC2Gateway(t)

	vpcId := mustCreateVpc(t, handler, "10.0.0.0/16")
	_ = mustCreateSubnet(t, handler, vpcId, "10.0.1.0/24")

	// Delete VPC should fail with DependencyViolation.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ec2Req(t, "DeleteVpc", url.Values{
		"VpcId": {vpcId},
	}))
	if w.Code == http.StatusOK {
		t.Error("DeleteVpc with subnets: expected error, got 200")
	}
	if !strings.Contains(w.Body.String(), "DependencyViolation") {
		t.Errorf("DeleteVpc with subnets: expected DependencyViolation\nbody: %s", w.Body.String())
	}
}

// ---- Test 5: DeleteSubnet then DeleteVpc succeeds ----

func TestEC2_DeleteSubnetThenDeleteVpc(t *testing.T) {
	handler := newEC2Gateway(t)

	vpcId := mustCreateVpc(t, handler, "10.0.0.0/16")
	subId := mustCreateSubnet(t, handler, vpcId, "10.0.1.0/24")

	// Delete subnet first.
	ws := httptest.NewRecorder()
	handler.ServeHTTP(ws, ec2Req(t, "DeleteSubnet", url.Values{
		"SubnetId": {subId},
	}))
	if ws.Code != http.StatusOK {
		t.Fatalf("DeleteSubnet: expected 200, got %d\nbody: %s", ws.Code, ws.Body.String())
	}
	if !strings.Contains(ws.Body.String(), "<return>true</return>") {
		t.Errorf("DeleteSubnet: expected return=true\nbody: %s", ws.Body.String())
	}

	// Verify subnet is gone.
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, ec2Req(t, "DescribeSubnets", url.Values{
		"SubnetId.1": {subId},
	}))
	if wd.Code != http.StatusOK {
		t.Fatalf("DescribeSubnets after delete: expected 200, got %d", wd.Code)
	}
	// Should return empty set, not an error.
	if strings.Contains(wd.Body.String(), subId) {
		t.Errorf("DescribeSubnets after delete: subnet should be gone\nbody: %s", wd.Body.String())
	}

	// Now delete VPC should succeed.
	wv := httptest.NewRecorder()
	handler.ServeHTTP(wv, ec2Req(t, "DeleteVpc", url.Values{
		"VpcId": {vpcId},
	}))
	if wv.Code != http.StatusOK {
		t.Fatalf("DeleteVpc: expected 200, got %d\nbody: %s", wv.Code, wv.Body.String())
	}
	if !strings.Contains(wv.Body.String(), "<return>true</return>") {
		t.Errorf("DeleteVpc: expected return=true\nbody: %s", wv.Body.String())
	}

	// Verify VPC is gone from DescribeVpcs.
	wdv := httptest.NewRecorder()
	handler.ServeHTTP(wdv, ec2Req(t, "DescribeVpcs", url.Values{
		"VpcId.1": {vpcId},
	}))
	if wdv.Code != http.StatusOK {
		t.Fatalf("DescribeVpcs after delete: expected 200, got %d", wdv.Code)
	}
	if strings.Contains(wdv.Body.String(), vpcId) {
		t.Errorf("DescribeVpcs after delete: VPC should be gone\nbody: %s", wdv.Body.String())
	}

	// Delete non-existent VPC should fail.
	wne := httptest.NewRecorder()
	handler.ServeHTTP(wne, ec2Req(t, "DeleteVpc", url.Values{
		"VpcId": {vpcId},
	}))
	if wne.Code == http.StatusOK {
		t.Error("DeleteVpc already deleted: expected error, got 200")
	}

	// Delete non-existent subnet should fail.
	wnes := httptest.NewRecorder()
	handler.ServeHTTP(wnes, ec2Req(t, "DeleteSubnet", url.Values{
		"SubnetId": {subId},
	}))
	if wnes.Code == http.StatusOK {
		t.Error("DeleteSubnet already deleted: expected error, got 200")
	}
}

// ---- Test 6: ModifyVpcAttribute ----

func TestEC2_ModifyVpcAttribute(t *testing.T) {
	handler := newEC2Gateway(t)

	vpcId := mustCreateVpc(t, handler, "10.0.0.0/16")

	// Enable DNS hostnames.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ec2Req(t, "ModifyVpcAttribute", url.Values{
		"VpcId":                     {vpcId},
		"EnableDnsHostnames.Value":  {"true"},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("ModifyVpcAttribute: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "<return>true</return>") {
		t.Errorf("ModifyVpcAttribute: expected return=true\nbody: %s", w.Body.String())
	}

	// Verify via DescribeVpcs.
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, ec2Req(t, "DescribeVpcs", url.Values{
		"VpcId.1": {vpcId},
	}))
	if wd.Code != http.StatusOK {
		t.Fatalf("DescribeVpcs: expected 200, got %d", wd.Code)
	}
	descBody := wd.Body.String()
	if !strings.Contains(descBody, "<enableDnsHostnames>true</enableDnsHostnames>") {
		t.Errorf("ModifyVpcAttribute: DNS hostnames should be true\nbody: %s", descBody)
	}

	// Modify non-existent VPC.
	wne := httptest.NewRecorder()
	handler.ServeHTTP(wne, ec2Req(t, "ModifyVpcAttribute", url.Values{
		"VpcId":                    {"vpc-nonexistent12345678"},
		"EnableDnsSupport.Value":   {"false"},
	}))
	if wne.Code == http.StatusOK {
		t.Error("ModifyVpcAttribute non-existent: expected error, got 200")
	}
}

// ---- Test 7: Unknown action ----

func TestEC2_UnknownAction(t *testing.T) {
	handler := newEC2Gateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ec2Req(t, "NonExistentAction", nil))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("unknown action: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
}
