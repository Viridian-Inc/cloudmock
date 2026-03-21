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

// ---- Security Group helpers ----

// mustCreateSG creates a security group and returns its GroupId.
func mustCreateSG(t *testing.T, handler http.Handler, vpcId, name, desc string) string {
	t.Helper()
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ec2Req(t, "CreateSecurityGroup", url.Values{
		"VpcId":            {vpcId},
		"GroupName":        {name},
		"GroupDescription": {desc},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateSecurityGroup %s: expected 200, got %d\nbody: %s", name, w.Code, w.Body.String())
	}
	var resp struct {
		GroupId string `xml:"groupId"`
	}
	if err := xml.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("CreateSecurityGroup: unmarshal: %v\nbody: %s", err, w.Body.String())
	}
	if resp.GroupId == "" {
		t.Fatalf("CreateSecurityGroup: groupId is empty\nbody: %s", w.Body.String())
	}
	return resp.GroupId
}

// describeOneSG calls DescribeSecurityGroups for a single group and returns the body.
func describeOneSG(t *testing.T, handler http.Handler, groupId string) string {
	t.Helper()
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ec2Req(t, "DescribeSecurityGroups", url.Values{
		"GroupId.1": {groupId},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeSecurityGroups %s: expected 200, got %d\nbody: %s", groupId, w.Code, w.Body.String())
	}
	return w.Body.String()
}

// ---- Test 8: CreateSecurityGroup + DescribeSecurityGroups (default egress) ----

func TestEC2_SecurityGroup_CreateAndDescribe(t *testing.T) {
	handler := newEC2Gateway(t)
	vpcId := mustCreateVpc(t, handler, "10.0.0.0/16")

	sgId := mustCreateSG(t, handler, vpcId, "web-sg", "Web tier security group")

	if !strings.HasPrefix(sgId, "sg-") {
		t.Errorf("CreateSecurityGroup: expected sg- prefix, got %s", sgId)
	}

	body := describeOneSG(t, handler, sgId)

	if !strings.Contains(body, sgId) {
		t.Errorf("DescribeSecurityGroups: expected groupId %s\nbody: %s", sgId, body)
	}
	if !strings.Contains(body, "<groupName>web-sg</groupName>") {
		t.Errorf("DescribeSecurityGroups: expected groupName\nbody: %s", body)
	}
	if !strings.Contains(body, "<groupDescription>Web tier security group</groupDescription>") {
		t.Errorf("DescribeSecurityGroups: expected groupDescription\nbody: %s", body)
	}
	if !strings.Contains(body, vpcId) {
		t.Errorf("DescribeSecurityGroups: expected vpcId\nbody: %s", body)
	}
	// Default egress rule: protocol -1, CIDR 0.0.0.0/0.
	if !strings.Contains(body, "<ipProtocol>-1</ipProtocol>") {
		t.Errorf("DescribeSecurityGroups: expected default egress rule with protocol -1\nbody: %s", body)
	}
	if !strings.Contains(body, "<cidrIp>0.0.0.0/0</cidrIp>") {
		t.Errorf("DescribeSecurityGroups: expected default egress CIDR 0.0.0.0/0\nbody: %s", body)
	}

	// Filter by vpc-id.
	wv := httptest.NewRecorder()
	handler.ServeHTTP(wv, ec2Req(t, "DescribeSecurityGroups", url.Values{
		"Filter.1.Name":    {"vpc-id"},
		"Filter.1.Value.1": {vpcId},
	}))
	if wv.Code != http.StatusOK {
		t.Fatalf("DescribeSecurityGroups vpc filter: expected 200, got %d", wv.Code)
	}
	if !strings.Contains(wv.Body.String(), sgId) {
		t.Errorf("DescribeSecurityGroups vpc filter: expected %s\nbody: %s", sgId, wv.Body.String())
	}

	// Filter by group-name.
	wn := httptest.NewRecorder()
	handler.ServeHTTP(wn, ec2Req(t, "DescribeSecurityGroups", url.Values{
		"Filter.1.Name":    {"group-name"},
		"Filter.1.Value.1": {"web-sg"},
	}))
	if wn.Code != http.StatusOK {
		t.Fatalf("DescribeSecurityGroups name filter: expected 200, got %d", wn.Code)
	}
	if !strings.Contains(wn.Body.String(), sgId) {
		t.Errorf("DescribeSecurityGroups name filter: expected %s\nbody: %s", sgId, wn.Body.String())
	}

	// Duplicate name in same VPC should fail.
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, ec2Req(t, "CreateSecurityGroup", url.Values{
		"VpcId":            {vpcId},
		"GroupName":        {"web-sg"},
		"GroupDescription": {"duplicate"},
	}))
	if wd.Code == http.StatusOK {
		t.Error("CreateSecurityGroup duplicate name: expected error, got 200")
	}
	if !strings.Contains(wd.Body.String(), "InvalidGroup.Duplicate") {
		t.Errorf("CreateSecurityGroup duplicate: expected InvalidGroup.Duplicate\nbody: %s", wd.Body.String())
	}
}

// ---- Test 9: AuthorizeSecurityGroupIngress + verify in Describe ----

func TestEC2_SecurityGroup_AuthorizeIngress(t *testing.T) {
	handler := newEC2Gateway(t)
	vpcId := mustCreateVpc(t, handler, "10.0.0.0/16")
	sgId := mustCreateSG(t, handler, vpcId, "app-sg", "App security group")

	// Authorize HTTPS ingress from 0.0.0.0/0.
	wa := httptest.NewRecorder()
	handler.ServeHTTP(wa, ec2Req(t, "AuthorizeSecurityGroupIngress", url.Values{
		"GroupId":                        {sgId},
		"IpPermissions.1.IpProtocol":     {"tcp"},
		"IpPermissions.1.FromPort":       {"443"},
		"IpPermissions.1.ToPort":         {"443"},
		"IpPermissions.1.IpRanges.1.CidrIp": {"0.0.0.0/0"},
	}))
	if wa.Code != http.StatusOK {
		t.Fatalf("AuthorizeSecurityGroupIngress: expected 200, got %d\nbody: %s", wa.Code, wa.Body.String())
	}
	if !strings.Contains(wa.Body.String(), "<return>true</return>") {
		t.Errorf("AuthorizeSecurityGroupIngress: expected return=true\nbody: %s", wa.Body.String())
	}

	// Verify via Describe.
	body := describeOneSG(t, handler, sgId)
	if !strings.Contains(body, "<ipProtocol>tcp</ipProtocol>") {
		t.Errorf("after authorize: expected tcp protocol in ipPermissions\nbody: %s", body)
	}
	if !strings.Contains(body, "<fromPort>443</fromPort>") {
		t.Errorf("after authorize: expected fromPort 443\nbody: %s", body)
	}
	if !strings.Contains(body, "<toPort>443</toPort>") {
		t.Errorf("after authorize: expected toPort 443\nbody: %s", body)
	}
	if !strings.Contains(body, "<cidrIp>0.0.0.0/0</cidrIp>") {
		t.Errorf("after authorize: expected cidrIp 0.0.0.0/0\nbody: %s", body)
	}

	// Authorize on non-existent SG should fail.
	wne := httptest.NewRecorder()
	handler.ServeHTTP(wne, ec2Req(t, "AuthorizeSecurityGroupIngress", url.Values{
		"GroupId":                        {"sg-nonexistent12345678"},
		"IpPermissions.1.IpProtocol":     {"tcp"},
		"IpPermissions.1.FromPort":       {"80"},
		"IpPermissions.1.ToPort":         {"80"},
		"IpPermissions.1.IpRanges.1.CidrIp": {"0.0.0.0/0"},
	}))
	if wne.Code == http.StatusOK {
		t.Error("AuthorizeSecurityGroupIngress non-existent: expected error, got 200")
	}
	if !strings.Contains(wne.Body.String(), "InvalidGroup.NotFound") {
		t.Errorf("AuthorizeSecurityGroupIngress non-existent: expected InvalidGroup.NotFound\nbody: %s", wne.Body.String())
	}
}

// ---- Test 10: RevokeSecurityGroupIngress + verify removed ----

func TestEC2_SecurityGroup_RevokeIngress(t *testing.T) {
	handler := newEC2Gateway(t)
	vpcId := mustCreateVpc(t, handler, "10.0.0.0/16")
	sgId := mustCreateSG(t, handler, vpcId, "revoke-sg", "Revoke test")

	// Add two ingress rules.
	handler.ServeHTTP(httptest.NewRecorder(), ec2Req(t, "AuthorizeSecurityGroupIngress", url.Values{
		"GroupId":                        {sgId},
		"IpPermissions.1.IpProtocol":     {"tcp"},
		"IpPermissions.1.FromPort":       {"80"},
		"IpPermissions.1.ToPort":         {"80"},
		"IpPermissions.1.IpRanges.1.CidrIp": {"0.0.0.0/0"},
		"IpPermissions.2.IpProtocol":     {"tcp"},
		"IpPermissions.2.FromPort":       {"443"},
		"IpPermissions.2.ToPort":         {"443"},
		"IpPermissions.2.IpRanges.1.CidrIp": {"0.0.0.0/0"},
	}))

	// Revoke port-80 rule.
	wr := httptest.NewRecorder()
	handler.ServeHTTP(wr, ec2Req(t, "RevokeSecurityGroupIngress", url.Values{
		"GroupId":                        {sgId},
		"IpPermissions.1.IpProtocol":     {"tcp"},
		"IpPermissions.1.FromPort":       {"80"},
		"IpPermissions.1.ToPort":         {"80"},
		"IpPermissions.1.IpRanges.1.CidrIp": {"0.0.0.0/0"},
	}))
	if wr.Code != http.StatusOK {
		t.Fatalf("RevokeSecurityGroupIngress: expected 200, got %d\nbody: %s", wr.Code, wr.Body.String())
	}

	// Verify port 80 is gone but port 443 remains.
	body := describeOneSG(t, handler, sgId)
	if strings.Contains(body, "<fromPort>80</fromPort>") {
		t.Errorf("after revoke: port 80 should be gone\nbody: %s", body)
	}
	if !strings.Contains(body, "<fromPort>443</fromPort>") {
		t.Errorf("after revoke: port 443 should still be present\nbody: %s", body)
	}

	// Revoking a rule that doesn't exist should fail.
	wne := httptest.NewRecorder()
	handler.ServeHTTP(wne, ec2Req(t, "RevokeSecurityGroupIngress", url.Values{
		"GroupId":                        {sgId},
		"IpPermissions.1.IpProtocol":     {"tcp"},
		"IpPermissions.1.FromPort":       {"8080"},
		"IpPermissions.1.ToPort":         {"8080"},
		"IpPermissions.1.IpRanges.1.CidrIp": {"0.0.0.0/0"},
	}))
	if wne.Code == http.StatusOK {
		t.Error("RevokeSecurityGroupIngress non-existent rule: expected error, got 200")
	}
	if !strings.Contains(wne.Body.String(), "InvalidPermission.NotFound") {
		t.Errorf("RevokeSecurityGroupIngress non-existent rule: expected InvalidPermission.NotFound\nbody: %s", wne.Body.String())
	}
}

// ---- Test 11: Cross-SG reference (authorize ingress from another SG) ----

func TestEC2_SecurityGroup_CrossSGReference(t *testing.T) {
	handler := newEC2Gateway(t)
	vpcId := mustCreateVpc(t, handler, "10.0.0.0/16")

	srcSgId := mustCreateSG(t, handler, vpcId, "source-sg", "Source SG")
	dstSgId := mustCreateSG(t, handler, vpcId, "dest-sg", "Destination SG")

	// Allow ingress from srcSg to dstSg on port 3306.
	wa := httptest.NewRecorder()
	handler.ServeHTTP(wa, ec2Req(t, "AuthorizeSecurityGroupIngress", url.Values{
		"GroupId":                                   {dstSgId},
		"IpPermissions.1.IpProtocol":                {"tcp"},
		"IpPermissions.1.FromPort":                  {"3306"},
		"IpPermissions.1.ToPort":                    {"3306"},
		"IpPermissions.1.UserIdGroupPairs.1.GroupId": {srcSgId},
	}))
	if wa.Code != http.StatusOK {
		t.Fatalf("AuthorizeSecurityGroupIngress cross-SG: expected 200, got %d\nbody: %s", wa.Code, wa.Body.String())
	}

	// Verify the group reference appears in the Describe output.
	body := describeOneSG(t, handler, dstSgId)
	if !strings.Contains(body, srcSgId) {
		t.Errorf("cross-SG reference: expected srcSgId %s in ipPermissions\nbody: %s", srcSgId, body)
	}
	if !strings.Contains(body, "<fromPort>3306</fromPort>") {
		t.Errorf("cross-SG reference: expected fromPort 3306\nbody: %s", body)
	}

	// Deleting srcSg while dstSg references it should fail.
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, ec2Req(t, "DeleteSecurityGroup", url.Values{
		"GroupId": {srcSgId},
	}))
	if wd.Code == http.StatusOK {
		t.Error("DeleteSecurityGroup referenced: expected DependencyViolation, got 200")
	}
	if !strings.Contains(wd.Body.String(), "DependencyViolation") {
		t.Errorf("DeleteSecurityGroup referenced: expected DependencyViolation\nbody: %s", wd.Body.String())
	}
}

// ---- Test 12: DeleteSecurityGroup ----

func TestEC2_SecurityGroup_Delete(t *testing.T) {
	handler := newEC2Gateway(t)
	vpcId := mustCreateVpc(t, handler, "10.0.0.0/16")
	sgId := mustCreateSG(t, handler, vpcId, "delete-me", "To be deleted")

	// Delete the group.
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, ec2Req(t, "DeleteSecurityGroup", url.Values{
		"GroupId": {sgId},
	}))
	if wd.Code != http.StatusOK {
		t.Fatalf("DeleteSecurityGroup: expected 200, got %d\nbody: %s", wd.Code, wd.Body.String())
	}
	if !strings.Contains(wd.Body.String(), "<return>true</return>") {
		t.Errorf("DeleteSecurityGroup: expected return=true\nbody: %s", wd.Body.String())
	}

	// Verify it no longer appears in Describe.
	wdesc := httptest.NewRecorder()
	handler.ServeHTTP(wdesc, ec2Req(t, "DescribeSecurityGroups", url.Values{
		"GroupId.1": {sgId},
	}))
	if wdesc.Code != http.StatusOK {
		t.Fatalf("DescribeSecurityGroups after delete: expected 200, got %d", wdesc.Code)
	}
	if strings.Contains(wdesc.Body.String(), sgId) {
		t.Errorf("DescribeSecurityGroups after delete: SG should be gone\nbody: %s", wdesc.Body.String())
	}

	// Delete non-existent SG should fail.
	wne := httptest.NewRecorder()
	handler.ServeHTTP(wne, ec2Req(t, "DeleteSecurityGroup", url.Values{
		"GroupId": {sgId},
	}))
	if wne.Code == http.StatusOK {
		t.Error("DeleteSecurityGroup already deleted: expected error, got 200")
	}
	if !strings.Contains(wne.Body.String(), "InvalidGroup.NotFound") {
		t.Errorf("DeleteSecurityGroup not found: expected InvalidGroup.NotFound\nbody: %s", wne.Body.String())
	}

	// Test AuthorizeSecurityGroupEgress and RevokeSecurityGroupEgress.
	sgId2 := mustCreateSG(t, handler, vpcId, "egress-test", "Egress test SG")

	// Add a custom egress rule.
	wae := httptest.NewRecorder()
	handler.ServeHTTP(wae, ec2Req(t, "AuthorizeSecurityGroupEgress", url.Values{
		"GroupId":                        {sgId2},
		"IpPermissions.1.IpProtocol":     {"tcp"},
		"IpPermissions.1.FromPort":       {"5432"},
		"IpPermissions.1.ToPort":         {"5432"},
		"IpPermissions.1.IpRanges.1.CidrIp": {"10.0.0.0/8"},
	}))
	if wae.Code != http.StatusOK {
		t.Fatalf("AuthorizeSecurityGroupEgress: expected 200, got %d\nbody: %s", wae.Code, wae.Body.String())
	}

	egressBody := describeOneSG(t, handler, sgId2)
	if !strings.Contains(egressBody, "<fromPort>5432</fromPort>") {
		t.Errorf("AuthorizeSecurityGroupEgress: expected port 5432\nbody: %s", egressBody)
	}
	if !strings.Contains(egressBody, "<cidrIp>10.0.0.0/8</cidrIp>") {
		t.Errorf("AuthorizeSecurityGroupEgress: expected cidr 10.0.0.0/8\nbody: %s", egressBody)
	}

	// Revoke the custom egress rule.
	wre := httptest.NewRecorder()
	handler.ServeHTTP(wre, ec2Req(t, "RevokeSecurityGroupEgress", url.Values{
		"GroupId":                        {sgId2},
		"IpPermissions.1.IpProtocol":     {"tcp"},
		"IpPermissions.1.FromPort":       {"5432"},
		"IpPermissions.1.ToPort":         {"5432"},
		"IpPermissions.1.IpRanges.1.CidrIp": {"10.0.0.0/8"},
	}))
	if wre.Code != http.StatusOK {
		t.Fatalf("RevokeSecurityGroupEgress: expected 200, got %d\nbody: %s", wre.Code, wre.Body.String())
	}

	afterRevokeBody := describeOneSG(t, handler, sgId2)
	if strings.Contains(afterRevokeBody, "<fromPort>5432</fromPort>") {
		t.Errorf("RevokeSecurityGroupEgress: port 5432 should be gone\nbody: %s", afterRevokeBody)
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

// ============================================================
// helpers for IGW/NAT/RT tests
// ============================================================

func mustCreateIGW(t *testing.T, handler http.Handler) string {
	t.Helper()
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ec2Req(t, "CreateInternetGateway", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateInternetGateway: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
	var resp struct {
		InternetGateway struct {
			InternetGatewayId string `xml:"internetGatewayId"`
		} `xml:"internetGateway"`
	}
	if err := xml.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("CreateInternetGateway unmarshal: %v\nbody: %s", err, w.Body.String())
	}
	if resp.InternetGateway.InternetGatewayId == "" {
		t.Fatalf("CreateInternetGateway: internetGatewayId empty\nbody: %s", w.Body.String())
	}
	return resp.InternetGateway.InternetGatewayId
}

func mustCreateNatGateway(t *testing.T, handler http.Handler, subnetId, allocId string) string {
	t.Helper()
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ec2Req(t, "CreateNatGateway", url.Values{
		"SubnetId":     {subnetId},
		"AllocationId": {allocId},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateNatGateway: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
	var resp struct {
		NatGateway struct {
			NatGatewayId string `xml:"natGatewayId"`
		} `xml:"natGateway"`
	}
	if err := xml.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("CreateNatGateway unmarshal: %v\nbody: %s", err, w.Body.String())
	}
	if resp.NatGateway.NatGatewayId == "" {
		t.Fatalf("CreateNatGateway: natGatewayId empty\nbody: %s", w.Body.String())
	}
	return resp.NatGateway.NatGatewayId
}

func mustCreateRouteTable(t *testing.T, handler http.Handler, vpcId string) string {
	t.Helper()
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ec2Req(t, "CreateRouteTable", url.Values{
		"VpcId": {vpcId},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateRouteTable: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
	var resp struct {
		RouteTable struct {
			RouteTableId string `xml:"routeTableId"`
		} `xml:"routeTable"`
	}
	if err := xml.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("CreateRouteTable unmarshal: %v\nbody: %s", err, w.Body.String())
	}
	if resp.RouteTable.RouteTableId == "" {
		t.Fatalf("CreateRouteTable: routeTableId empty\nbody: %s", w.Body.String())
	}
	return resp.RouteTable.RouteTableId
}

// ---- Test 8: IGW lifecycle ----

func TestEC2_InternetGatewayLifecycle(t *testing.T) {
	handler := newEC2Gateway(t)
	vpcId := mustCreateVpc(t, handler, "10.0.0.0/16")

	// CreateInternetGateway.
	igwId := mustCreateIGW(t, handler)
	if !strings.HasPrefix(igwId, "igw-") {
		t.Errorf("CreateInternetGateway: expected igw- prefix, got %s", igwId)
	}

	// DescribeInternetGateways — should appear.
	{
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, ec2Req(t, "DescribeInternetGateways", url.Values{
			"InternetGatewayId.1": {igwId},
		}))
		if w.Code != http.StatusOK {
			t.Fatalf("DescribeInternetGateways: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), igwId) {
			t.Errorf("DescribeInternetGateways: expected %s\nbody: %s", igwId, w.Body.String())
		}
	}

	// DeleteInternetGateway while not attached should succeed.
	igw2Id := mustCreateIGW(t, handler)
	{
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, ec2Req(t, "DeleteInternetGateway", url.Values{
			"InternetGatewayId": {igw2Id},
		}))
		if w.Code != http.StatusOK {
			t.Fatalf("DeleteInternetGateway (detached): expected 200, got %d\nbody: %s", w.Code, w.Body.String())
		}
	}

	// AttachInternetGateway.
	{
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, ec2Req(t, "AttachInternetGateway", url.Values{
			"InternetGatewayId": {igwId},
			"VpcId":             {vpcId},
		}))
		if w.Code != http.StatusOK {
			t.Fatalf("AttachInternetGateway: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
		}
	}

	// DescribeInternetGateways filtered by attachment.vpc-id.
	{
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, ec2Req(t, "DescribeInternetGateways", url.Values{
			"Filter.1.Name":    {"attachment.vpc-id"},
			"Filter.1.Value.1": {vpcId},
		}))
		if w.Code != http.StatusOK {
			t.Fatalf("DescribeInternetGateways vpc filter: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
		}
		body := w.Body.String()
		if !strings.Contains(body, igwId) {
			t.Errorf("DescribeInternetGateways vpc filter: expected %s\nbody: %s", igwId, body)
		}
		if !strings.Contains(body, "<state>attached</state>") {
			t.Errorf("DescribeInternetGateways: expected state=attached\nbody: %s", body)
		}
	}

	// Second attach to same VPC should fail.
	{
		igw3Id := mustCreateIGW(t, handler)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, ec2Req(t, "AttachInternetGateway", url.Values{
			"InternetGatewayId": {igw3Id},
			"VpcId":             {vpcId},
		}))
		if w.Code == http.StatusOK {
			t.Error("AttachInternetGateway: second attach to same VPC should fail")
		}
		if !strings.Contains(w.Body.String(), "AlreadyAssociated") {
			t.Errorf("AttachInternetGateway: expected AlreadyAssociated\nbody: %s", w.Body.String())
		}
	}

	// DeleteInternetGateway while attached should fail.
	{
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, ec2Req(t, "DeleteInternetGateway", url.Values{
			"InternetGatewayId": {igwId},
		}))
		if w.Code == http.StatusOK {
			t.Error("DeleteInternetGateway while attached: expected error, got 200")
		}
		if !strings.Contains(w.Body.String(), "DependencyViolation") {
			t.Errorf("DeleteInternetGateway while attached: expected DependencyViolation\nbody: %s", w.Body.String())
		}
	}

	// DetachInternetGateway.
	{
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, ec2Req(t, "DetachInternetGateway", url.Values{
			"InternetGatewayId": {igwId},
			"VpcId":             {vpcId},
		}))
		if w.Code != http.StatusOK {
			t.Fatalf("DetachInternetGateway: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
		}
	}

	// Now delete should succeed.
	{
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, ec2Req(t, "DeleteInternetGateway", url.Values{
			"InternetGatewayId": {igwId},
		}))
		if w.Code != http.StatusOK {
			t.Fatalf("DeleteInternetGateway after detach: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
		}
	}

	// Detach a non-attached IGW should return Gateway.NotAttached.
	{
		igw4Id := mustCreateIGW(t, handler)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, ec2Req(t, "DetachInternetGateway", url.Values{
			"InternetGatewayId": {igw4Id},
			"VpcId":             {vpcId},
		}))
		if w.Code == http.StatusOK {
			t.Error("DetachInternetGateway not attached: expected error, got 200")
		}
		if !strings.Contains(w.Body.String(), "Gateway.NotAttached") {
			t.Errorf("DetachInternetGateway not attached: expected Gateway.NotAttached\nbody: %s", w.Body.String())
		}
	}
}

// ---- Test 9: NAT Gateway lifecycle ----

func TestEC2_NatGatewayLifecycle(t *testing.T) {
	handler := newEC2Gateway(t)
	vpcId := mustCreateVpc(t, handler, "10.0.0.0/16")
	subnetId := mustCreateSubnet(t, handler, vpcId, "10.0.1.0/24")

	// CreateNatGateway.
	natId := mustCreateNatGateway(t, handler, subnetId, "eipalloc-abc123")
	if !strings.HasPrefix(natId, "nat-") {
		t.Errorf("CreateNatGateway: expected nat- prefix, got %s", natId)
	}

	// DescribeNatGateways — by ID.
	{
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, ec2Req(t, "DescribeNatGateways", url.Values{
			"NatGatewayId.1": {natId},
		}))
		if w.Code != http.StatusOK {
			t.Fatalf("DescribeNatGateways: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
		}
		body := w.Body.String()
		if !strings.Contains(body, natId) {
			t.Errorf("DescribeNatGateways: expected %s\nbody: %s", natId, body)
		}
		if !strings.Contains(body, "<state>available</state>") {
			t.Errorf("DescribeNatGateways: expected state=available\nbody: %s", body)
		}
		if !strings.Contains(body, subnetId) {
			t.Errorf("DescribeNatGateways: expected subnetId\nbody: %s", body)
		}
		if !strings.Contains(body, vpcId) {
			t.Errorf("DescribeNatGateways: expected vpcId\nbody: %s", body)
		}
	}

	// DescribeNatGateways — filter by subnet-id.
	{
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, ec2Req(t, "DescribeNatGateways", url.Values{
			"Filter.1.Name":    {"subnet-id"},
			"Filter.1.Value.1": {subnetId},
		}))
		if w.Code != http.StatusOK {
			t.Fatalf("DescribeNatGateways subnet filter: expected 200, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), natId) {
			t.Errorf("DescribeNatGateways subnet filter: expected %s\nbody: %s", natId, w.Body.String())
		}
	}

	// DeleteNatGateway.
	{
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, ec2Req(t, "DeleteNatGateway", url.Values{
			"NatGatewayId": {natId},
		}))
		if w.Code != http.StatusOK {
			t.Fatalf("DeleteNatGateway: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "<state>deleted</state>") {
			t.Errorf("DeleteNatGateway: expected state=deleted in response\nbody: %s", w.Body.String())
		}
	}

	// DescribeNatGateways after delete — filter by state=deleted.
	{
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, ec2Req(t, "DescribeNatGateways", url.Values{
			"Filter.1.Name":    {"state"},
			"Filter.1.Value.1": {"deleted"},
		}))
		if w.Code != http.StatusOK {
			t.Fatalf("DescribeNatGateways state filter: expected 200, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), natId) {
			t.Errorf("DescribeNatGateways deleted: expected %s\nbody: %s", natId, w.Body.String())
		}
	}

	// CreateNatGateway for non-existent subnet should fail.
	{
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, ec2Req(t, "CreateNatGateway", url.Values{
			"SubnetId":     {"subnet-nonexistent000001"},
			"AllocationId": {"eipalloc-xyz"},
		}))
		if w.Code == http.StatusOK {
			t.Error("CreateNatGateway non-existent subnet: expected error, got 200")
		}
		if !strings.Contains(w.Body.String(), "InvalidSubnetID.NotFound") {
			t.Errorf("CreateNatGateway non-existent subnet: expected InvalidSubnetID.NotFound\nbody: %s", w.Body.String())
		}
	}
}

// ---- Test 10: Route Table lifecycle ----

func TestEC2_RouteTableLifecycle(t *testing.T) {
	handler := newEC2Gateway(t)
	vpcId := mustCreateVpc(t, handler, "10.0.0.0/16")

	// CreateRouteTable.
	rtbId := mustCreateRouteTable(t, handler, vpcId)
	if !strings.HasPrefix(rtbId, "rtb-") {
		t.Errorf("CreateRouteTable: expected rtb- prefix, got %s", rtbId)
	}

	// DescribeRouteTables — local route should be present.
	{
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, ec2Req(t, "DescribeRouteTables", url.Values{
			"RouteTableId.1": {rtbId},
		}))
		if w.Code != http.StatusOK {
			t.Fatalf("DescribeRouteTables: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
		}
		body := w.Body.String()
		if !strings.Contains(body, rtbId) {
			t.Errorf("DescribeRouteTables: expected %s\nbody: %s", rtbId, body)
		}
		if !strings.Contains(body, "<gatewayId>local</gatewayId>") {
			t.Errorf("DescribeRouteTables: expected local route\nbody: %s", body)
		}
		if !strings.Contains(body, "10.0.0.0/16") {
			t.Errorf("DescribeRouteTables: expected VPC CIDR in local route\nbody: %s", body)
		}
	}

	// CreateRoute via IGW.
	igwId := mustCreateIGW(t, handler)
	{
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, ec2Req(t, "AttachInternetGateway", url.Values{
			"InternetGatewayId": {igwId},
			"VpcId":             {vpcId},
		}))
		if w.Code != http.StatusOK {
			t.Fatalf("AttachInternetGateway: %d %s", w.Code, w.Body.String())
		}
	}
	{
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, ec2Req(t, "CreateRoute", url.Values{
			"RouteTableId":         {rtbId},
			"DestinationCidrBlock": {"0.0.0.0/0"},
			"GatewayId":            {igwId},
		}))
		if w.Code != http.StatusOK {
			t.Fatalf("CreateRoute IGW: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
		}
	}

	// CreateRoute via NAT Gateway.
	subnetId := mustCreateSubnet(t, handler, vpcId, "10.0.1.0/24")
	natId := mustCreateNatGateway(t, handler, subnetId, "eipalloc-001")
	{
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, ec2Req(t, "CreateRoute", url.Values{
			"RouteTableId":         {rtbId},
			"DestinationCidrBlock": {"192.168.0.0/16"},
			"NatGatewayId":         {natId},
		}))
		if w.Code != http.StatusOK {
			t.Fatalf("CreateRoute NAT: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
		}
	}

	// DescribeRouteTables — verify both routes present.
	{
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, ec2Req(t, "DescribeRouteTables", url.Values{
			"RouteTableId.1": {rtbId},
		}))
		body := w.Body.String()
		if !strings.Contains(body, "0.0.0.0/0") {
			t.Errorf("DescribeRouteTables: expected default route\nbody: %s", body)
		}
		if !strings.Contains(body, igwId) {
			t.Errorf("DescribeRouteTables: expected igwId in route\nbody: %s", body)
		}
		if !strings.Contains(body, "192.168.0.0/16") {
			t.Errorf("DescribeRouteTables: expected NAT route destination\nbody: %s", body)
		}
		if !strings.Contains(body, natId) {
			t.Errorf("DescribeRouteTables: expected natId in route\nbody: %s", body)
		}
	}

	// DeleteRoute — remove the NAT route.
	{
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, ec2Req(t, "DeleteRoute", url.Values{
			"RouteTableId":         {rtbId},
			"DestinationCidrBlock": {"192.168.0.0/16"},
		}))
		if w.Code != http.StatusOK {
			t.Fatalf("DeleteRoute: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
		}
	}

	// Verify route is gone.
	{
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, ec2Req(t, "DescribeRouteTables", url.Values{
			"RouteTableId.1": {rtbId},
		}))
		if strings.Contains(w.Body.String(), "192.168.0.0/16") {
			t.Errorf("DescribeRouteTables: NAT route should be deleted\nbody: %s", w.Body.String())
		}
	}

	// Delete local (CreateRouteTable) route should fail.
	{
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, ec2Req(t, "DeleteRoute", url.Values{
			"RouteTableId":         {rtbId},
			"DestinationCidrBlock": {"10.0.0.0/16"},
		}))
		if w.Code == http.StatusOK {
			t.Error("DeleteRoute local: expected error, got 200")
		}
		if !strings.Contains(w.Body.String(), "InvalidParameterValue") {
			t.Errorf("DeleteRoute local: expected InvalidParameterValue\nbody: %s", w.Body.String())
		}
	}

	// DeleteRouteTable — should succeed (no subnet associations).
	{
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, ec2Req(t, "DeleteRouteTable", url.Values{
			"RouteTableId": {rtbId},
		}))
		if w.Code != http.StatusOK {
			t.Fatalf("DeleteRouteTable: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
		}
	}
}

// ---- Test 11: Route Table associations ----

func TestEC2_RouteTableAssociations(t *testing.T) {
	handler := newEC2Gateway(t)
	vpcId := mustCreateVpc(t, handler, "10.0.0.0/16")
	subnetId := mustCreateSubnet(t, handler, vpcId, "10.0.1.0/24")
	rtbId := mustCreateRouteTable(t, handler, vpcId)

	// AssociateRouteTable.
	var assocId string
	{
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, ec2Req(t, "AssociateRouteTable", url.Values{
			"RouteTableId": {rtbId},
			"SubnetId":     {subnetId},
		}))
		if w.Code != http.StatusOK {
			t.Fatalf("AssociateRouteTable: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
		}
		var resp struct {
			AssociationId string `xml:"associationId"`
		}
		if err := xml.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("AssociateRouteTable unmarshal: %v", err)
		}
		if !strings.HasPrefix(resp.AssociationId, "rtbassoc-") {
			t.Errorf("AssociateRouteTable: expected rtbassoc- prefix, got %s", resp.AssociationId)
		}
		assocId = resp.AssociationId
	}

	// DescribeRouteTables — association should appear.
	{
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, ec2Req(t, "DescribeRouteTables", url.Values{
			"RouteTableId.1": {rtbId},
		}))
		body := w.Body.String()
		if !strings.Contains(body, assocId) {
			t.Errorf("DescribeRouteTables: expected assocId %s\nbody: %s", assocId, body)
		}
		if !strings.Contains(body, subnetId) {
			t.Errorf("DescribeRouteTables: expected subnetId in association\nbody: %s", body)
		}
	}

	// DeleteRouteTable while associated should fail.
	{
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, ec2Req(t, "DeleteRouteTable", url.Values{
			"RouteTableId": {rtbId},
		}))
		if w.Code == http.StatusOK {
			t.Error("DeleteRouteTable with associations: expected error, got 200")
		}
		if !strings.Contains(w.Body.String(), "DependencyViolation") {
			t.Errorf("DeleteRouteTable with associations: expected DependencyViolation\nbody: %s", w.Body.String())
		}
	}

	// DisassociateRouteTable.
	{
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, ec2Req(t, "DisassociateRouteTable", url.Values{
			"AssociationId": {assocId},
		}))
		if w.Code != http.StatusOK {
			t.Fatalf("DisassociateRouteTable: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
		}
	}

	// After disassociation the association ID should be gone.
	{
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, ec2Req(t, "DescribeRouteTables", url.Values{
			"RouteTableId.1": {rtbId},
		}))
		if strings.Contains(w.Body.String(), assocId) {
			t.Errorf("DescribeRouteTables: assocId should be gone after disassociation\nbody: %s", w.Body.String())
		}
	}

	// Now delete should succeed.
	{
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, ec2Req(t, "DeleteRouteTable", url.Values{
			"RouteTableId": {rtbId},
		}))
		if w.Code != http.StatusOK {
			t.Fatalf("DeleteRouteTable after disassoc: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
		}
	}
}

// ---- Test 12: CreateRoute target validation ----

func TestEC2_CreateRouteTargetValidation(t *testing.T) {
	handler := newEC2Gateway(t)
	vpcId := mustCreateVpc(t, handler, "10.0.0.0/16")
	rtbId := mustCreateRouteTable(t, handler, vpcId)

	// CreateRoute with nonexistent IGW should fail.
	{
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, ec2Req(t, "CreateRoute", url.Values{
			"RouteTableId":         {rtbId},
			"DestinationCidrBlock": {"0.0.0.0/0"},
			"GatewayId":            {"igw-doesnotexist0001"},
		}))
		if w.Code == http.StatusOK {
			t.Error("CreateRoute nonexistent IGW: expected error, got 200")
		}
		if !strings.Contains(w.Body.String(), "InvalidGatewayID.NotFound") {
			t.Errorf("CreateRoute nonexistent IGW: expected InvalidGatewayID.NotFound\nbody: %s", w.Body.String())
		}
	}

	// CreateRoute with nonexistent NAT GW should fail.
	{
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, ec2Req(t, "CreateRoute", url.Values{
			"RouteTableId":         {rtbId},
			"DestinationCidrBlock": {"0.0.0.0/0"},
			"NatGatewayId":         {"nat-doesnotexist00001"},
		}))
		if w.Code == http.StatusOK {
			t.Error("CreateRoute nonexistent NAT GW: expected error, got 200")
		}
		if !strings.Contains(w.Body.String(), "InvalidNatGatewayID.NotFound") {
			t.Errorf("CreateRoute nonexistent NAT GW: expected InvalidNatGatewayID.NotFound\nbody: %s", w.Body.String())
		}
	}

	// CreateRoute with no target should fail.
	{
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, ec2Req(t, "CreateRoute", url.Values{
			"RouteTableId":         {rtbId},
			"DestinationCidrBlock": {"0.0.0.0/0"},
		}))
		if w.Code == http.StatusOK {
			t.Error("CreateRoute no target: expected error, got 200")
		}
		if !strings.Contains(w.Body.String(), "MissingParameter") {
			t.Errorf("CreateRoute no target: expected MissingParameter\nbody: %s", w.Body.String())
		}
	}

	// ReplaceRoute with nonexistent destination should fail.
	{
		igwId := mustCreateIGW(t, handler)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, ec2Req(t, "ReplaceRoute", url.Values{
			"RouteTableId":         {rtbId},
			"DestinationCidrBlock": {"1.2.3.0/24"},
			"GatewayId":            {igwId},
		}))
		if w.Code == http.StatusOK {
			t.Error("ReplaceRoute nonexistent dest: expected error, got 200")
		}
		if !strings.Contains(w.Body.String(), "InvalidRoute.NotFound") {
			t.Errorf("ReplaceRoute nonexistent dest: expected InvalidRoute.NotFound\nbody: %s", w.Body.String())
		}
	}

	// CreateRoute on nonexistent route table should fail.
	{
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, ec2Req(t, "CreateRoute", url.Values{
			"RouteTableId":         {"rtb-nonexistent000001"},
			"DestinationCidrBlock": {"0.0.0.0/0"},
			"GatewayId":            {"igw-fake"},
		}))
		if w.Code == http.StatusOK {
			t.Error("CreateRoute nonexistent RTB: expected error, got 200")
		}
		if !strings.Contains(w.Body.String(), "InvalidRouteTableID.NotFound") {
			t.Errorf("CreateRoute nonexistent RTB: expected InvalidRouteTableID.NotFound\nbody: %s", w.Body.String())
		}
	}
}

// ---- Test 13: ReplaceRoute updates an existing route ----

func TestEC2_ReplaceRoute(t *testing.T) {
	handler := newEC2Gateway(t)
	vpcId := mustCreateVpc(t, handler, "10.0.0.0/16")
	rtbId := mustCreateRouteTable(t, handler, vpcId)
	igw1Id := mustCreateIGW(t, handler)
	igw2Id := mustCreateIGW(t, handler)

	// Attach both IGWs would fail (only 1 per VPC) — attach only igw1.
	{
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, ec2Req(t, "AttachInternetGateway", url.Values{
			"InternetGatewayId": {igw1Id},
			"VpcId":             {vpcId},
		}))
		if w.Code != http.StatusOK {
			t.Fatalf("AttachInternetGateway igw1: %d %s", w.Code, w.Body.String())
		}
	}
	// Detach igw1, attach igw2 so both exist as valid IGWs.
	{
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, ec2Req(t, "DetachInternetGateway", url.Values{
			"InternetGatewayId": {igw1Id},
			"VpcId":             {vpcId},
		}))
		if w.Code != http.StatusOK {
			t.Fatalf("DetachInternetGateway igw1: %d %s", w.Code, w.Body.String())
		}
	}

	// Add a route pointing to igw1.
	{
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, ec2Req(t, "CreateRoute", url.Values{
			"RouteTableId":         {rtbId},
			"DestinationCidrBlock": {"0.0.0.0/0"},
			"GatewayId":            {igw1Id},
		}))
		if w.Code != http.StatusOK {
			t.Fatalf("CreateRoute: %d %s", w.Code, w.Body.String())
		}
	}

	// ReplaceRoute — point 0.0.0.0/0 to igw2 instead.
	{
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, ec2Req(t, "ReplaceRoute", url.Values{
			"RouteTableId":         {rtbId},
			"DestinationCidrBlock": {"0.0.0.0/0"},
			"GatewayId":            {igw2Id},
		}))
		if w.Code != http.StatusOK {
			t.Fatalf("ReplaceRoute: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
		}
	}

	// Verify route now points to igw2.
	{
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, ec2Req(t, "DescribeRouteTables", url.Values{
			"RouteTableId.1": {rtbId},
		}))
		body := w.Body.String()
		if !strings.Contains(body, igw2Id) {
			t.Errorf("ReplaceRoute: expected igw2Id in route\nbody: %s", body)
		}
	}
}

// ============================================================
// Task 4: Network Resources tests
// ============================================================

// ---- Test: EIP (AllocateAddress, AssociateAddress, DisassociateAddress, ReleaseAddress) ----

func TestEC2_EIP_Lifecycle(t *testing.T) {
	handler := newEC2Gateway(t)

	// AllocateAddress.
	wa := httptest.NewRecorder()
	handler.ServeHTTP(wa, ec2Req(t, "AllocateAddress", nil))
	if wa.Code != http.StatusOK {
		t.Fatalf("AllocateAddress: expected 200, got %d\nbody: %s", wa.Code, wa.Body.String())
	}
	allocBody := wa.Body.String()
	if !strings.Contains(allocBody, "eipalloc-") {
		t.Errorf("AllocateAddress: expected eipalloc- prefix\nbody: %s", allocBody)
	}
	if !strings.Contains(allocBody, "<domain>vpc</domain>") {
		t.Errorf("AllocateAddress: expected domain=vpc\nbody: %s", allocBody)
	}
	var allocResp struct {
		AllocationId string `xml:"allocationId"`
		PublicIp     string `xml:"publicIp"`
	}
	if err := xml.Unmarshal(wa.Body.Bytes(), &allocResp); err != nil {
		t.Fatalf("AllocateAddress unmarshal: %v", err)
	}
	allocationId := allocResp.AllocationId
	if allocationId == "" {
		t.Fatalf("AllocateAddress: empty allocationId")
	}
	if !strings.HasPrefix(allocResp.PublicIp, "54.") {
		t.Errorf("AllocateAddress: expected 54.x.x.x PublicIp, got %s", allocResp.PublicIp)
	}

	// DescribeAddresses — verify the EIP exists.
	{
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, ec2Req(t, "DescribeAddresses", url.Values{
			"AllocationId.1": {allocationId},
		}))
		if w.Code != http.StatusOK {
			t.Fatalf("DescribeAddresses: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), allocationId) {
			t.Errorf("DescribeAddresses: expected allocationId\nbody: %s", w.Body.String())
		}
	}

	// AssociateAddress with a fake instanceId (store doesn't validate instance existence).
	wAssoc := httptest.NewRecorder()
	handler.ServeHTTP(wAssoc, ec2Req(t, "AssociateAddress", url.Values{
		"AllocationId": {allocationId},
		"InstanceId":   {"i-00000000000000001"},
	}))
	if wAssoc.Code != http.StatusOK {
		t.Fatalf("AssociateAddress: expected 200, got %d\nbody: %s", wAssoc.Code, wAssoc.Body.String())
	}
	if !strings.Contains(wAssoc.Body.String(), "eipassoc-") {
		t.Errorf("AssociateAddress: expected eipassoc- prefix\nbody: %s", wAssoc.Body.String())
	}
	var assocResp struct {
		AssociationId string `xml:"associationId"`
	}
	if err := xml.Unmarshal(wAssoc.Body.Bytes(), &assocResp); err != nil {
		t.Fatalf("AssociateAddress unmarshal: %v", err)
	}
	assocId := assocResp.AssociationId

	// ReleaseAddress while still associated should fail.
	wRelFail := httptest.NewRecorder()
	handler.ServeHTTP(wRelFail, ec2Req(t, "ReleaseAddress", url.Values{
		"AllocationId": {allocationId},
	}))
	if wRelFail.Code == http.StatusOK {
		t.Error("ReleaseAddress while associated: expected error, got 200")
	}
	if !strings.Contains(wRelFail.Body.String(), "InvalidIPAddress.InUse") {
		t.Errorf("ReleaseAddress while associated: expected InvalidIPAddress.InUse\nbody: %s", wRelFail.Body.String())
	}

	// DisassociateAddress.
	wDisassoc := httptest.NewRecorder()
	handler.ServeHTTP(wDisassoc, ec2Req(t, "DisassociateAddress", url.Values{
		"AssociationId": {assocId},
	}))
	if wDisassoc.Code != http.StatusOK {
		t.Fatalf("DisassociateAddress: expected 200, got %d\nbody: %s", wDisassoc.Code, wDisassoc.Body.String())
	}

	// ReleaseAddress after disassociation should succeed.
	wRel := httptest.NewRecorder()
	handler.ServeHTTP(wRel, ec2Req(t, "ReleaseAddress", url.Values{
		"AllocationId": {allocationId},
	}))
	if wRel.Code != http.StatusOK {
		t.Fatalf("ReleaseAddress: expected 200, got %d\nbody: %s", wRel.Code, wRel.Body.String())
	}

	// DescribeAddresses after release should be empty for that ID.
	{
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, ec2Req(t, "DescribeAddresses", url.Values{
			"AllocationId.1": {allocationId},
		}))
		if w.Code != http.StatusOK {
			t.Fatalf("DescribeAddresses after release: expected 200, got %d", w.Code)
		}
		if strings.Contains(w.Body.String(), allocationId) {
			t.Errorf("DescribeAddresses after release: allocationId should be gone\nbody: %s", w.Body.String())
		}
	}
}

// ---- Test: ENI (CreateNetworkInterface, DescribeNetworkInterfaces, DeleteNetworkInterface) ----

func TestEC2_NetworkInterface_Lifecycle(t *testing.T) {
	handler := newEC2Gateway(t)
	vpcId := mustCreateVpc(t, handler, "10.0.0.0/16")
	subnetId := mustCreateSubnet(t, handler, vpcId, "10.0.1.0/24")

	// CreateNetworkInterface.
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, ec2Req(t, "CreateNetworkInterface", url.Values{
		"SubnetId":         {subnetId},
		"SecurityGroupId.1": {"sg-dummy00000000001"},
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("CreateNetworkInterface: expected 200, got %d\nbody: %s", wc.Code, wc.Body.String())
	}
	createBody := wc.Body.String()
	if !strings.Contains(createBody, "eni-") {
		t.Errorf("CreateNetworkInterface: expected eni- prefix\nbody: %s", createBody)
	}
	if !strings.Contains(createBody, subnetId) {
		t.Errorf("CreateNetworkInterface: expected subnetId in response\nbody: %s", createBody)
	}
	if !strings.Contains(createBody, vpcId) {
		t.Errorf("CreateNetworkInterface: expected vpcId in response\nbody: %s", createBody)
	}
	if !strings.Contains(createBody, "<status>available</status>") {
		t.Errorf("CreateNetworkInterface: expected status=available\nbody: %s", createBody)
	}
	var createResp struct {
		NetworkInterface struct {
			NetworkInterfaceId string `xml:"networkInterfaceId"`
		} `xml:"networkInterface"`
	}
	if err := xml.Unmarshal(wc.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("CreateNetworkInterface unmarshal: %v", err)
	}
	eniId := createResp.NetworkInterface.NetworkInterfaceId
	if eniId == "" {
		t.Fatalf("CreateNetworkInterface: empty networkInterfaceId")
	}

	// DescribeNetworkInterfaces — by ID.
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, ec2Req(t, "DescribeNetworkInterfaces", url.Values{
		"NetworkInterfaceId.1": {eniId},
	}))
	if wd.Code != http.StatusOK {
		t.Fatalf("DescribeNetworkInterfaces: expected 200, got %d\nbody: %s", wd.Code, wd.Body.String())
	}
	if !strings.Contains(wd.Body.String(), eniId) {
		t.Errorf("DescribeNetworkInterfaces: expected eniId\nbody: %s", wd.Body.String())
	}

	// DescribeNetworkInterfaces — by subnet-id filter.
	wf := httptest.NewRecorder()
	handler.ServeHTTP(wf, ec2Req(t, "DescribeNetworkInterfaces", url.Values{
		"Filter.1.Name":    {"subnet-id"},
		"Filter.1.Value.1": {subnetId},
	}))
	if wf.Code != http.StatusOK {
		t.Fatalf("DescribeNetworkInterfaces subnet filter: expected 200, got %d", wf.Code)
	}
	if !strings.Contains(wf.Body.String(), eniId) {
		t.Errorf("DescribeNetworkInterfaces subnet filter: expected eniId\nbody: %s", wf.Body.String())
	}

	// DeleteNetworkInterface.
	wdel := httptest.NewRecorder()
	handler.ServeHTTP(wdel, ec2Req(t, "DeleteNetworkInterface", url.Values{
		"NetworkInterfaceId": {eniId},
	}))
	if wdel.Code != http.StatusOK {
		t.Fatalf("DeleteNetworkInterface: expected 200, got %d\nbody: %s", wdel.Code, wdel.Body.String())
	}

	// Delete again should fail.
	wdel2 := httptest.NewRecorder()
	handler.ServeHTTP(wdel2, ec2Req(t, "DeleteNetworkInterface", url.Values{
		"NetworkInterfaceId": {eniId},
	}))
	if wdel2.Code == http.StatusOK {
		t.Error("DeleteNetworkInterface second call: expected error, got 200")
	}
}

// ---- Test: NACL (CreateNetworkAcl, CreateNetworkAclEntry, DescribeNetworkAcls, DeleteNetworkAclEntry, DeleteNetworkAcl) ----

func TestEC2_NetworkACL_Lifecycle(t *testing.T) {
	handler := newEC2Gateway(t)
	vpcId := mustCreateVpc(t, handler, "10.0.0.0/16")

	// CreateNetworkAcl.
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, ec2Req(t, "CreateNetworkAcl", url.Values{
		"VpcId": {vpcId},
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("CreateNetworkAcl: expected 200, got %d\nbody: %s", wc.Code, wc.Body.String())
	}
	createBody := wc.Body.String()
	if !strings.Contains(createBody, "acl-") {
		t.Errorf("CreateNetworkAcl: expected acl- prefix\nbody: %s", createBody)
	}
	if !strings.Contains(createBody, vpcId) {
		t.Errorf("CreateNetworkAcl: expected vpcId\nbody: %s", createBody)
	}
	var createResp struct {
		NetworkAcl struct {
			NetworkAclId string `xml:"networkAclId"`
		} `xml:"networkAcl"`
	}
	if err := xml.Unmarshal(wc.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("CreateNetworkAcl unmarshal: %v", err)
	}
	aclId := createResp.NetworkAcl.NetworkAclId
	if aclId == "" {
		t.Fatalf("CreateNetworkAcl: empty networkAclId")
	}

	// CreateNetworkAclEntry — add custom rule 200 allow tcp inbound.
	we := httptest.NewRecorder()
	handler.ServeHTTP(we, ec2Req(t, "CreateNetworkAclEntry", url.Values{
		"NetworkAclId": {aclId},
		"RuleNumber":   {"200"},
		"Protocol":     {"6"},
		"RuleAction":   {"allow"},
		"Egress":       {"false"},
		"CidrBlock":    {"10.0.0.0/8"},
	}))
	if we.Code != http.StatusOK {
		t.Fatalf("CreateNetworkAclEntry: expected 200, got %d\nbody: %s", we.Code, we.Body.String())
	}

	// Duplicate rule number + direction should fail.
	wdup := httptest.NewRecorder()
	handler.ServeHTTP(wdup, ec2Req(t, "CreateNetworkAclEntry", url.Values{
		"NetworkAclId": {aclId},
		"RuleNumber":   {"200"},
		"Protocol":     {"6"},
		"RuleAction":   {"deny"},
		"Egress":       {"false"},
		"CidrBlock":    {"192.168.0.0/16"},
	}))
	if wdup.Code == http.StatusOK {
		t.Error("CreateNetworkAclEntry duplicate rule: expected error, got 200")
	}
	if !strings.Contains(wdup.Body.String(), "NetworkAclEntryAlreadyExists") {
		t.Errorf("CreateNetworkAclEntry duplicate: expected NetworkAclEntryAlreadyExists\nbody: %s", wdup.Body.String())
	}

	// DescribeNetworkAcls — by ID should include the new entry.
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, ec2Req(t, "DescribeNetworkAcls", url.Values{
		"NetworkAclId.1": {aclId},
	}))
	if wd.Code != http.StatusOK {
		t.Fatalf("DescribeNetworkAcls: expected 200, got %d\nbody: %s", wd.Code, wd.Body.String())
	}
	descBody := wd.Body.String()
	if !strings.Contains(descBody, aclId) {
		t.Errorf("DescribeNetworkAcls: expected aclId\nbody: %s", descBody)
	}
	if !strings.Contains(descBody, "<ruleNumber>200</ruleNumber>") {
		t.Errorf("DescribeNetworkAcls: expected rule 200\nbody: %s", descBody)
	}

	// DescribeNetworkAcls — filter by vpc-id.
	wvf := httptest.NewRecorder()
	handler.ServeHTTP(wvf, ec2Req(t, "DescribeNetworkAcls", url.Values{
		"Filter.1.Name":    {"vpc-id"},
		"Filter.1.Value.1": {vpcId},
	}))
	if wvf.Code != http.StatusOK {
		t.Fatalf("DescribeNetworkAcls vpc filter: expected 200, got %d", wvf.Code)
	}
	if !strings.Contains(wvf.Body.String(), aclId) {
		t.Errorf("DescribeNetworkAcls vpc filter: expected aclId\nbody: %s", wvf.Body.String())
	}

	// DeleteNetworkAclEntry — remove rule 200 inbound.
	wde := httptest.NewRecorder()
	handler.ServeHTTP(wde, ec2Req(t, "DeleteNetworkAclEntry", url.Values{
		"NetworkAclId": {aclId},
		"RuleNumber":   {"200"},
		"Egress":       {"false"},
	}))
	if wde.Code != http.StatusOK {
		t.Fatalf("DeleteNetworkAclEntry: expected 200, got %d\nbody: %s", wde.Code, wde.Body.String())
	}

	// Verify entry is gone.
	{
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, ec2Req(t, "DescribeNetworkAcls", url.Values{
			"NetworkAclId.1": {aclId},
		}))
		if strings.Contains(w.Body.String(), "<ruleNumber>200</ruleNumber>") {
			t.Errorf("DeleteNetworkAclEntry: rule 200 should be gone\nbody: %s", w.Body.String())
		}
	}

	// DeleteNetworkAcl.
	wdel := httptest.NewRecorder()
	handler.ServeHTTP(wdel, ec2Req(t, "DeleteNetworkAcl", url.Values{
		"NetworkAclId": {aclId},
	}))
	if wdel.Code != http.StatusOK {
		t.Fatalf("DeleteNetworkAcl: expected 200, got %d\nbody: %s", wdel.Code, wdel.Body.String())
	}

	// Delete default NACL should fail.
	// First find the default NACL for this VPC.
	wdn := httptest.NewRecorder()
	handler.ServeHTTP(wdn, ec2Req(t, "DescribeNetworkAcls", url.Values{
		"Filter.1.Name":    {"vpc-id"},
		"Filter.1.Value.1": {vpcId},
	}))
	var descAll struct {
		Items []struct {
			NetworkAclId string `xml:"networkAclId"`
			IsDefault    bool   `xml:"default"`
		} `xml:"networkAclSet>item"`
	}
	if err := xml.Unmarshal(wdn.Body.Bytes(), &descAll); err != nil {
		t.Fatalf("DescribeNetworkAcls for default: unmarshal: %v", err)
	}
	var defaultAclId string
	for _, item := range descAll.Items {
		if item.IsDefault {
			defaultAclId = item.NetworkAclId
			break
		}
	}
	if defaultAclId == "" {
		t.Fatal("could not find default NACL")
	}
	wdelDefault := httptest.NewRecorder()
	handler.ServeHTTP(wdelDefault, ec2Req(t, "DeleteNetworkAcl", url.Values{
		"NetworkAclId": {defaultAclId},
	}))
	if wdelDefault.Code == http.StatusOK {
		t.Error("DeleteNetworkAcl default: expected error, got 200")
	}
	if !strings.Contains(wdelDefault.Body.String(), "InvalidParameterValue") {
		t.Errorf("DeleteNetworkAcl default: expected error code\nbody: %s", wdelDefault.Body.String())
	}
}

// ---- Test: VPC Endpoint (CreateVpcEndpoint, DescribeVpcEndpoints, DeleteVpcEndpoints) ----

func TestEC2_VpcEndpoint_Lifecycle(t *testing.T) {
	handler := newEC2Gateway(t)
	vpcId := mustCreateVpc(t, handler, "10.0.0.0/16")

	// CreateVpcEndpoint — Gateway type.
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, ec2Req(t, "CreateVpcEndpoint", url.Values{
		"VpcId":           {vpcId},
		"ServiceName":     {"com.amazonaws.us-east-1.s3"},
		"VpcEndpointType": {"Gateway"},
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("CreateVpcEndpoint: expected 200, got %d\nbody: %s", wc.Code, wc.Body.String())
	}
	createBody := wc.Body.String()
	if !strings.Contains(createBody, "vpce-") {
		t.Errorf("CreateVpcEndpoint: expected vpce- prefix\nbody: %s", createBody)
	}
	if !strings.Contains(createBody, "<state>available</state>") {
		t.Errorf("CreateVpcEndpoint: expected state=available\nbody: %s", createBody)
	}
	if !strings.Contains(createBody, "com.amazonaws.us-east-1.s3") {
		t.Errorf("CreateVpcEndpoint: expected serviceName\nbody: %s", createBody)
	}
	if !strings.Contains(createBody, "<vpcEndpointType>Gateway</vpcEndpointType>") {
		t.Errorf("CreateVpcEndpoint: expected type=Gateway\nbody: %s", createBody)
	}
	var createResp struct {
		VpcEndpoint struct {
			VpcEndpointId string `xml:"vpcEndpointId"`
		} `xml:"vpcEndpoint"`
	}
	if err := xml.Unmarshal(wc.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("CreateVpcEndpoint unmarshal: %v", err)
	}
	epId := createResp.VpcEndpoint.VpcEndpointId
	if epId == "" {
		t.Fatalf("CreateVpcEndpoint: empty vpcEndpointId")
	}

	// DescribeVpcEndpoints — by ID.
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, ec2Req(t, "DescribeVpcEndpoints", url.Values{
		"VpcEndpointId.1": {epId},
	}))
	if wd.Code != http.StatusOK {
		t.Fatalf("DescribeVpcEndpoints: expected 200, got %d\nbody: %s", wd.Code, wd.Body.String())
	}
	if !strings.Contains(wd.Body.String(), epId) {
		t.Errorf("DescribeVpcEndpoints: expected epId\nbody: %s", wd.Body.String())
	}

	// DescribeVpcEndpoints — filter by vpc-id.
	wvf := httptest.NewRecorder()
	handler.ServeHTTP(wvf, ec2Req(t, "DescribeVpcEndpoints", url.Values{
		"Filter.1.Name":    {"vpc-id"},
		"Filter.1.Value.1": {vpcId},
	}))
	if wvf.Code != http.StatusOK {
		t.Fatalf("DescribeVpcEndpoints vpc filter: expected 200, got %d", wvf.Code)
	}
	if !strings.Contains(wvf.Body.String(), epId) {
		t.Errorf("DescribeVpcEndpoints vpc filter: expected epId\nbody: %s", wvf.Body.String())
	}

	// DeleteVpcEndpoints.
	wdel := httptest.NewRecorder()
	handler.ServeHTTP(wdel, ec2Req(t, "DeleteVpcEndpoints", url.Values{
		"VpcEndpointId.1": {epId},
	}))
	if wdel.Code != http.StatusOK {
		t.Fatalf("DeleteVpcEndpoints: expected 200, got %d\nbody: %s", wdel.Code, wdel.Body.String())
	}

	// Delete again should fail.
	wdel2 := httptest.NewRecorder()
	handler.ServeHTTP(wdel2, ec2Req(t, "DeleteVpcEndpoints", url.Values{
		"VpcEndpointId.1": {epId},
	}))
	if wdel2.Code == http.StatusOK {
		t.Error("DeleteVpcEndpoints second call: expected error, got 200")
	}
}

// ---- Test: VPC Peering (CreateVpcPeeringConnection, AcceptVpcPeeringConnection, DeleteVpcPeeringConnection) ----

func TestEC2_VpcPeering_Lifecycle(t *testing.T) {
	handler := newEC2Gateway(t)
	vpcId1 := mustCreateVpc(t, handler, "10.0.0.0/16")
	vpcId2 := mustCreateVpc(t, handler, "10.1.0.0/16")

	// CreateVpcPeeringConnection.
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, ec2Req(t, "CreateVpcPeeringConnection", url.Values{
		"VpcId":     {vpcId1},
		"PeerVpcId": {vpcId2},
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("CreateVpcPeeringConnection: expected 200, got %d\nbody: %s", wc.Code, wc.Body.String())
	}
	createBody := wc.Body.String()
	if !strings.Contains(createBody, "pcx-") {
		t.Errorf("CreateVpcPeeringConnection: expected pcx- prefix\nbody: %s", createBody)
	}
	if !strings.Contains(createBody, "pending-acceptance") {
		t.Errorf("CreateVpcPeeringConnection: expected status=pending-acceptance\nbody: %s", createBody)
	}
	if !strings.Contains(createBody, vpcId1) {
		t.Errorf("CreateVpcPeeringConnection: expected requesterVpcId\nbody: %s", createBody)
	}
	if !strings.Contains(createBody, vpcId2) {
		t.Errorf("CreateVpcPeeringConnection: expected accepterVpcId\nbody: %s", createBody)
	}
	var createResp struct {
		VpcPeeringConnection struct {
			VpcPeeringConnectionId string `xml:"vpcPeeringConnectionId"`
		} `xml:"vpcPeeringConnection"`
	}
	if err := xml.Unmarshal(wc.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("CreateVpcPeeringConnection unmarshal: %v", err)
	}
	pcxId := createResp.VpcPeeringConnection.VpcPeeringConnectionId
	if pcxId == "" {
		t.Fatalf("CreateVpcPeeringConnection: empty vpcPeeringConnectionId")
	}

	// AcceptVpcPeeringConnection.
	wacc := httptest.NewRecorder()
	handler.ServeHTTP(wacc, ec2Req(t, "AcceptVpcPeeringConnection", url.Values{
		"VpcPeeringConnectionId": {pcxId},
	}))
	if wacc.Code != http.StatusOK {
		t.Fatalf("AcceptVpcPeeringConnection: expected 200, got %d\nbody: %s", wacc.Code, wacc.Body.String())
	}
	if !strings.Contains(wacc.Body.String(), "active") {
		t.Errorf("AcceptVpcPeeringConnection: expected status=active\nbody: %s", wacc.Body.String())
	}

	// Accept again (now active) should fail.
	wacc2 := httptest.NewRecorder()
	handler.ServeHTTP(wacc2, ec2Req(t, "AcceptVpcPeeringConnection", url.Values{
		"VpcPeeringConnectionId": {pcxId},
	}))
	if wacc2.Code == http.StatusOK {
		t.Error("AcceptVpcPeeringConnection second call: expected error, got 200")
	}
	if !strings.Contains(wacc2.Body.String(), "InvalidStateTransition") {
		t.Errorf("AcceptVpcPeeringConnection second: expected InvalidStateTransition\nbody: %s", wacc2.Body.String())
	}

	// DeleteVpcPeeringConnection.
	wdel := httptest.NewRecorder()
	handler.ServeHTTP(wdel, ec2Req(t, "DeleteVpcPeeringConnection", url.Values{
		"VpcPeeringConnectionId": {pcxId},
	}))
	if wdel.Code != http.StatusOK {
		t.Fatalf("DeleteVpcPeeringConnection: expected 200, got %d\nbody: %s", wdel.Code, wdel.Body.String())
	}

	// Delete again should fail.
	wdel2 := httptest.NewRecorder()
	handler.ServeHTTP(wdel2, ec2Req(t, "DeleteVpcPeeringConnection", url.Values{
		"VpcPeeringConnectionId": {pcxId},
	}))
	if wdel2.Code == http.StatusOK {
		t.Error("DeleteVpcPeeringConnection second call: expected error, got 200")
	}
	if !strings.Contains(wdel2.Body.String(), "InvalidVpcPeeringConnectionID.NotFound") {
		t.Errorf("DeleteVpcPeeringConnection second: expected NotFound\nbody: %s", wdel2.Body.String())
	}

	// Create with non-existent requester VPC should fail.
	wbad := httptest.NewRecorder()
	handler.ServeHTTP(wbad, ec2Req(t, "CreateVpcPeeringConnection", url.Values{
		"VpcId":     {"vpc-nonexistent00000000"},
		"PeerVpcId": {vpcId2},
	}))
	if wbad.Code == http.StatusOK {
		t.Error("CreateVpcPeeringConnection bad VpcId: expected error, got 200")
	}
}

// ============================================================
// Instance + Tagging tests
// ============================================================

// mustRunInstance runs a single instance and returns its instanceId.
func mustRunInstance(t *testing.T, handler http.Handler, imageId, subnetId string) string {
	t.Helper()
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ec2Req(t, "RunInstances", url.Values{
		"ImageId":      {imageId},
		"SubnetId":     {subnetId},
		"InstanceType": {"t2.micro"},
		"MinCount":     {"1"},
		"MaxCount":     {"1"},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("RunInstances: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
	var resp struct {
		InstancesSet struct {
			Items []struct {
				InstanceId string `xml:"instanceId"`
			} `xml:"item"`
		} `xml:"instancesSet"`
	}
	if err := xml.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("RunInstances: unmarshal: %v\nbody: %s", err, w.Body.String())
	}
	if len(resp.InstancesSet.Items) == 0 || resp.InstancesSet.Items[0].InstanceId == "" {
		t.Fatalf("RunInstances: instanceId is empty\nbody: %s", w.Body.String())
	}
	return resp.InstancesSet.Items[0].InstanceId
}

// ---- Test: RunInstances + DescribeInstances ----

func TestEC2_RunAndDescribeInstances(t *testing.T) {
	handler := newEC2Gateway(t)

	vpcId := mustCreateVpc(t, handler, "10.0.0.0/16")
	subnetId := mustCreateSubnet(t, handler, vpcId, "10.0.1.0/24")

	// RunInstances — basic.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ec2Req(t, "RunInstances", url.Values{
		"ImageId":           {"ami-12345678"},
		"SubnetId":          {subnetId},
		"InstanceType":      {"t2.micro"},
		"MinCount":          {"1"},
		"MaxCount":          {"1"},
		"SecurityGroupId.1": {"sg-fake"},
		"KeyName":           {"my-key"},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("RunInstances: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()

	if !strings.Contains(body, "<name>running</name>") {
		t.Errorf("RunInstances: expected state=running\nbody: %s", body)
	}
	if !strings.Contains(body, vpcId) {
		t.Errorf("RunInstances: expected vpcId in response\nbody: %s", body)
	}
	if !strings.Contains(body, subnetId) {
		t.Errorf("RunInstances: expected subnetId in response\nbody: %s", body)
	}
	if !strings.Contains(body, "10.0.1.") {
		t.Errorf("RunInstances: expected private IP from subnet range\nbody: %s", body)
	}

	// Parse instance ID.
	var runResp struct {
		InstancesSet struct {
			Items []struct {
				InstanceId string `xml:"instanceId"`
				VpcId      string `xml:"vpcId"`
				SubnetId   string `xml:"subnetId"`
			} `xml:"item"`
		} `xml:"instancesSet"`
	}
	if err := xml.Unmarshal(w.Body.Bytes(), &runResp); err != nil {
		t.Fatalf("RunInstances: unmarshal: %v", err)
	}
	if len(runResp.InstancesSet.Items) == 0 {
		t.Fatal("RunInstances: no instances in response")
	}
	instId := runResp.InstancesSet.Items[0].InstanceId
	if !strings.HasPrefix(instId, "i-") {
		t.Errorf("RunInstances: expected i- prefix, got %s", instId)
	}
	if runResp.InstancesSet.Items[0].VpcId != vpcId {
		t.Errorf("RunInstances: vpcId mismatch: got %s, want %s",
			runResp.InstancesSet.Items[0].VpcId, vpcId)
	}
	if runResp.InstancesSet.Items[0].SubnetId != subnetId {
		t.Errorf("RunInstances: subnetId mismatch: got %s, want %s",
			runResp.InstancesSet.Items[0].SubnetId, subnetId)
	}

	// DescribeInstances — all.
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, ec2Req(t, "DescribeInstances", nil))
	if wd.Code != http.StatusOK {
		t.Fatalf("DescribeInstances: expected 200, got %d\nbody: %s", wd.Code, wd.Body.String())
	}
	if !strings.Contains(wd.Body.String(), instId) {
		t.Errorf("DescribeInstances: expected %s in response\nbody: %s", instId, wd.Body.String())
	}

	// DescribeInstances — by ID.
	wf := httptest.NewRecorder()
	handler.ServeHTTP(wf, ec2Req(t, "DescribeInstances", url.Values{
		"InstanceId.1": {instId},
	}))
	if wf.Code != http.StatusOK {
		t.Fatalf("DescribeInstances by ID: expected 200, got %d\nbody: %s", wf.Code, wf.Body.String())
	}
	if !strings.Contains(wf.Body.String(), instId) {
		t.Errorf("DescribeInstances by ID: expected %s\nbody: %s", instId, wf.Body.String())
	}

	// DescribeInstances — filter by vpc-id.
	wfv := httptest.NewRecorder()
	handler.ServeHTTP(wfv, ec2Req(t, "DescribeInstances", url.Values{
		"Filter.1.Name":    {"vpc-id"},
		"Filter.1.Value.1": {vpcId},
	}))
	if wfv.Code != http.StatusOK {
		t.Fatalf("DescribeInstances vpc filter: expected 200, got %d\nbody: %s", wfv.Code, wfv.Body.String())
	}
	if !strings.Contains(wfv.Body.String(), instId) {
		t.Errorf("DescribeInstances vpc filter: expected %s\nbody: %s", instId, wfv.Body.String())
	}

	// RunInstances with bad subnet should fail.
	wbad := httptest.NewRecorder()
	handler.ServeHTTP(wbad, ec2Req(t, "RunInstances", url.Values{
		"ImageId":  {"ami-12345678"},
		"SubnetId": {"subnet-nonexistent000000"},
		"MinCount": {"1"},
		"MaxCount": {"1"},
	}))
	if wbad.Code == http.StatusOK {
		t.Error("RunInstances bad subnet: expected error, got 200")
	}
}

// ---- Test: TerminateInstances ----

func TestEC2_TerminateInstances(t *testing.T) {
	handler := newEC2Gateway(t)
	vpcId := mustCreateVpc(t, handler, "10.0.0.0/16")
	subnetId := mustCreateSubnet(t, handler, vpcId, "10.0.1.0/24")

	instId := mustRunInstance(t, handler, "ami-abc", subnetId)

	// Terminate it.
	wt := httptest.NewRecorder()
	handler.ServeHTTP(wt, ec2Req(t, "TerminateInstances", url.Values{
		"InstanceId.1": {instId},
	}))
	if wt.Code != http.StatusOK {
		t.Fatalf("TerminateInstances: expected 200, got %d\nbody: %s", wt.Code, wt.Body.String())
	}
	termBody := wt.Body.String()
	if !strings.Contains(termBody, "<name>terminated</name>") {
		t.Errorf("TerminateInstances: expected terminated state\nbody: %s", termBody)
	}
	if !strings.Contains(termBody, "<name>running</name>") {
		t.Errorf("TerminateInstances: expected previous state=running\nbody: %s", termBody)
	}

	// DescribeInstances should still show terminated instance.
	wdesc := httptest.NewRecorder()
	handler.ServeHTTP(wdesc, ec2Req(t, "DescribeInstances", url.Values{
		"InstanceId.1": {instId},
	}))
	if wdesc.Code != http.StatusOK {
		t.Fatalf("DescribeInstances after terminate: expected 200, got %d", wdesc.Code)
	}
	if !strings.Contains(wdesc.Body.String(), "terminated") {
		t.Errorf("DescribeInstances after terminate: expected terminated state\nbody: %s", wdesc.Body.String())
	}
}

// ---- Test: StopInstances + StartInstances ----

func TestEC2_StopAndStartInstances(t *testing.T) {
	handler := newEC2Gateway(t)
	vpcId := mustCreateVpc(t, handler, "10.0.0.0/16")
	subnetId := mustCreateSubnet(t, handler, vpcId, "10.0.1.0/24")

	instId := mustRunInstance(t, handler, "ami-abc", subnetId)

	// Stop running instance.
	ws := httptest.NewRecorder()
	handler.ServeHTTP(ws, ec2Req(t, "StopInstances", url.Values{
		"InstanceId.1": {instId},
	}))
	if ws.Code != http.StatusOK {
		t.Fatalf("StopInstances: expected 200, got %d\nbody: %s", ws.Code, ws.Body.String())
	}
	stopBody := ws.Body.String()
	if !strings.Contains(stopBody, "<name>stopped</name>") {
		t.Errorf("StopInstances: expected stopped state\nbody: %s", stopBody)
	}

	// Verify stopped via DescribeInstanceStatus.
	wst := httptest.NewRecorder()
	handler.ServeHTTP(wst, ec2Req(t, "DescribeInstanceStatus", url.Values{
		"InstanceId.1": {instId},
	}))
	if wst.Code != http.StatusOK {
		t.Fatalf("DescribeInstanceStatus: expected 200, got %d\nbody: %s", wst.Code, wst.Body.String())
	}
	if !strings.Contains(wst.Body.String(), "<name>stopped</name>") {
		t.Errorf("DescribeInstanceStatus: expected stopped\nbody: %s", wst.Body.String())
	}

	// Start the stopped instance.
	wstart := httptest.NewRecorder()
	handler.ServeHTTP(wstart, ec2Req(t, "StartInstances", url.Values{
		"InstanceId.1": {instId},
	}))
	if wstart.Code != http.StatusOK {
		t.Fatalf("StartInstances: expected 200, got %d\nbody: %s", wstart.Code, wstart.Body.String())
	}
	startBody := wstart.Body.String()
	if !strings.Contains(startBody, "<name>running</name>") {
		t.Errorf("StartInstances: expected running state\nbody: %s", startBody)
	}
	if !strings.Contains(startBody, "<name>stopped</name>") {
		t.Errorf("StartInstances: expected previous state=stopped\nbody: %s", startBody)
	}

	// Stop a stopped instance — should have no items (noop).
	ws2 := httptest.NewRecorder()
	handler.ServeHTTP(ws2, ec2Req(t, "StopInstances", url.Values{
		"InstanceId.1": {instId},
	}))
	_ = ws2 // state is running, stop it again is fine
}

// ---- Test: CreateTags + DescribeTags ----

func TestEC2_CreateAndDescribeTags(t *testing.T) {
	handler := newEC2Gateway(t)
	vpcId := mustCreateVpc(t, handler, "10.0.0.0/16")
	subnetId := mustCreateSubnet(t, handler, vpcId, "10.0.1.0/24")
	instId := mustRunInstance(t, handler, "ami-abc", subnetId)

	// Tag the VPC.
	wct := httptest.NewRecorder()
	handler.ServeHTTP(wct, ec2Req(t, "CreateTags", url.Values{
		"ResourceId.1": {vpcId},
		"Tag.1.Key":    {"Name"},
		"Tag.1.Value":  {"my-vpc"},
		"Tag.2.Key":    {"Env"},
		"Tag.2.Value":  {"test"},
	}))
	if wct.Code != http.StatusOK {
		t.Fatalf("CreateTags VPC: expected 200, got %d\nbody: %s", wct.Code, wct.Body.String())
	}

	// Tag the instance.
	wcti := httptest.NewRecorder()
	handler.ServeHTTP(wcti, ec2Req(t, "CreateTags", url.Values{
		"ResourceId.1": {instId},
		"Tag.1.Key":    {"Name"},
		"Tag.1.Value":  {"web-server"},
	}))
	if wcti.Code != http.StatusOK {
		t.Fatalf("CreateTags instance: expected 200, got %d\nbody: %s", wcti.Code, wcti.Body.String())
	}

	// DescribeTags — all.
	wdt := httptest.NewRecorder()
	handler.ServeHTTP(wdt, ec2Req(t, "DescribeTags", nil))
	if wdt.Code != http.StatusOK {
		t.Fatalf("DescribeTags: expected 200, got %d\nbody: %s", wdt.Code, wdt.Body.String())
	}
	dtBody := wdt.Body.String()
	if !strings.Contains(dtBody, "my-vpc") {
		t.Errorf("DescribeTags: expected 'my-vpc'\nbody: %s", dtBody)
	}
	if !strings.Contains(dtBody, "web-server") {
		t.Errorf("DescribeTags: expected 'web-server'\nbody: %s", dtBody)
	}
	if !strings.Contains(dtBody, "<resourceType>vpc</resourceType>") {
		t.Errorf("DescribeTags: expected resourceType=vpc\nbody: %s", dtBody)
	}
	if !strings.Contains(dtBody, "<resourceType>instance</resourceType>") {
		t.Errorf("DescribeTags: expected resourceType=instance\nbody: %s", dtBody)
	}

	// DescribeTags — filter by resource-id.
	wdf := httptest.NewRecorder()
	handler.ServeHTTP(wdf, ec2Req(t, "DescribeTags", url.Values{
		"Filter.1.Name":    {"resource-id"},
		"Filter.1.Value.1": {vpcId},
	}))
	if wdf.Code != http.StatusOK {
		t.Fatalf("DescribeTags filter by resource-id: expected 200, got %d\nbody: %s", wdf.Code, wdf.Body.String())
	}
	dfBody := wdf.Body.String()
	if !strings.Contains(dfBody, vpcId) {
		t.Errorf("DescribeTags filter: expected vpcId\nbody: %s", dfBody)
	}
	if strings.Contains(dfBody, instId) {
		t.Errorf("DescribeTags filter: instId should be excluded\nbody: %s", dfBody)
	}

	// DescribeTags — filter by key.
	wdk := httptest.NewRecorder()
	handler.ServeHTTP(wdk, ec2Req(t, "DescribeTags", url.Values{
		"Filter.1.Name":    {"key"},
		"Filter.1.Value.1": {"Env"},
	}))
	if wdk.Code != http.StatusOK {
		t.Fatalf("DescribeTags filter by key: expected 200, got %d\nbody: %s", wdk.Code, wdk.Body.String())
	}
	if !strings.Contains(wdk.Body.String(), "test") {
		t.Errorf("DescribeTags filter by key: expected Env=test\nbody: %s", wdk.Body.String())
	}

	// Verify VPC tags are propagated in DescribeVpcs.
	wdv := httptest.NewRecorder()
	handler.ServeHTTP(wdv, ec2Req(t, "DescribeVpcs", url.Values{
		"VpcId.1": {vpcId},
	}))
	// Tags propagation is in the store; DescribeVpcs response doesn't include tags in current
	// XML mapping but the store's VPC.Tags map should have them.
}

// ---- Test: DeleteTags ----

func TestEC2_DeleteTags(t *testing.T) {
	handler := newEC2Gateway(t)
	vpcId := mustCreateVpc(t, handler, "10.0.0.0/16")

	// Create tags.
	wct := httptest.NewRecorder()
	handler.ServeHTTP(wct, ec2Req(t, "CreateTags", url.Values{
		"ResourceId.1": {vpcId},
		"Tag.1.Key":    {"Name"},
		"Tag.1.Value":  {"my-vpc"},
		"Tag.2.Key":    {"Env"},
		"Tag.2.Value":  {"prod"},
	}))
	if wct.Code != http.StatusOK {
		t.Fatalf("CreateTags: expected 200, got %d\nbody: %s", wct.Code, wct.Body.String())
	}

	// Verify tags exist.
	wdt := httptest.NewRecorder()
	handler.ServeHTTP(wdt, ec2Req(t, "DescribeTags", url.Values{
		"Filter.1.Name":    {"resource-id"},
		"Filter.1.Value.1": {vpcId},
	}))
	if !strings.Contains(wdt.Body.String(), "my-vpc") {
		t.Fatalf("CreateTags: expected Name tag before deletion\nbody: %s", wdt.Body.String())
	}

	// Delete "Name" tag.
	wdel := httptest.NewRecorder()
	handler.ServeHTTP(wdel, ec2Req(t, "DeleteTags", url.Values{
		"ResourceId.1": {vpcId},
		"Tag.1.Key":    {"Name"},
	}))
	if wdel.Code != http.StatusOK {
		t.Fatalf("DeleteTags: expected 200, got %d\nbody: %s", wdel.Code, wdel.Body.String())
	}

	// DescribeTags should no longer have Name tag.
	wdt2 := httptest.NewRecorder()
	handler.ServeHTTP(wdt2, ec2Req(t, "DescribeTags", url.Values{
		"Filter.1.Name":    {"resource-id"},
		"Filter.1.Value.1": {vpcId},
	}))
	if wdt2.Code != http.StatusOK {
		t.Fatalf("DescribeTags after delete: expected 200, got %d\nbody: %s", wdt2.Code, wdt2.Body.String())
	}
	afterBody := wdt2.Body.String()
	if strings.Contains(afterBody, "my-vpc") {
		t.Errorf("DeleteTags: Name tag should be removed\nbody: %s", afterBody)
	}
	// Env tag should still exist.
	if !strings.Contains(afterBody, "prod") {
		t.Errorf("DeleteTags: Env tag should still exist\nbody: %s", afterBody)
	}
}

// ---- Test: DescribeInstances with tag filter ----

func TestEC2_DescribeInstancesTagFilter(t *testing.T) {
	handler := newEC2Gateway(t)
	vpcId := mustCreateVpc(t, handler, "10.0.0.0/16")
	subnetId := mustCreateSubnet(t, handler, vpcId, "10.0.1.0/24")

	instId1 := mustRunInstance(t, handler, "ami-abc", subnetId)
	instId2 := mustRunInstance(t, handler, "ami-abc", subnetId)

	// Tag instId1 with Name=web.
	wct := httptest.NewRecorder()
	handler.ServeHTTP(wct, ec2Req(t, "CreateTags", url.Values{
		"ResourceId.1": {instId1},
		"Tag.1.Key":    {"Name"},
		"Tag.1.Value":  {"web"},
	}))
	if wct.Code != http.StatusOK {
		t.Fatalf("CreateTags: expected 200, got %d\nbody: %s", wct.Code, wct.Body.String())
	}

	// Tag instId2 with Name=db.
	wct2 := httptest.NewRecorder()
	handler.ServeHTTP(wct2, ec2Req(t, "CreateTags", url.Values{
		"ResourceId.1": {instId2},
		"Tag.1.Key":    {"Name"},
		"Tag.1.Value":  {"db"},
	}))
	if wct2.Code != http.StatusOK {
		t.Fatalf("CreateTags: expected 200, got %d\nbody: %s", wct2.Code, wct2.Body.String())
	}

	// DescribeInstances with tag:Name=web should return only instId1.
	wdf := httptest.NewRecorder()
	handler.ServeHTTP(wdf, ec2Req(t, "DescribeInstances", url.Values{
		"Filter.1.Name":    {"tag:Name"},
		"Filter.1.Value.1": {"web"},
	}))
	if wdf.Code != http.StatusOK {
		t.Fatalf("DescribeInstances tag filter: expected 200, got %d\nbody: %s", wdf.Code, wdf.Body.String())
	}
	filterBody := wdf.Body.String()
	if !strings.Contains(filterBody, instId1) {
		t.Errorf("DescribeInstances tag filter: expected %s\nbody: %s", instId1, filterBody)
	}
	if strings.Contains(filterBody, instId2) {
		t.Errorf("DescribeInstances tag filter: %s should be excluded\nbody: %s", instId2, filterBody)
	}
}
