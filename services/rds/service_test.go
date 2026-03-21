package rds_test

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
	rdssvc "github.com/neureaux/cloudmock/services/rds"
)

// newRDSGateway builds a full gateway stack with the RDS service registered and IAM disabled.
func newRDSGateway(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"

	reg := routing.NewRegistry()
	reg.Register(rdssvc.New(cfg.AccountID, cfg.Region))

	return gateway.New(cfg, reg)
}

// rdsReq builds a form-encoded POST request targeting the RDS service.
func rdsReq(t *testing.T, action string, extra url.Values) *http.Request {
	t.Helper()

	form := url.Values{}
	form.Set("Action", action)
	form.Set("Version", "2014-10-31")
	for k, vs := range extra {
		for _, v := range vs {
			form.Add(k, v)
		}
	}

	body := strings.NewReader(form.Encode())
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/rds/aws4_request, SignedHeaders=host, Signature=abc123")
	return req
}

// mustCreateInstance creates a DB instance and returns its ARN.
func mustCreateInstance(t *testing.T, handler http.Handler, id, class, engine string) string {
	t.Helper()
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, rdsReq(t, "CreateDBInstance", url.Values{
		"DBInstanceIdentifier": {id},
		"DBInstanceClass":      {class},
		"Engine":               {engine},
		"MasterUsername":       {"admin"},
		"MasterUserPassword":   {"password123"},
		"AllocatedStorage":     {"20"},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateDBInstance %s: expected 200, got %d\nbody: %s", id, w.Code, w.Body.String())
	}
	var resp struct {
		Result struct {
			DBInstance struct {
				DBInstanceArn string `xml:"DBInstanceArn"`
			} `xml:"DBInstance"`
		} `xml:"CreateDBInstanceResult"`
	}
	if err := xml.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("CreateDBInstance: unmarshal: %v\nbody: %s", err, w.Body.String())
	}
	arn := resp.Result.DBInstance.DBInstanceArn
	if arn == "" {
		t.Fatalf("CreateDBInstance: DBInstanceArn is empty\nbody: %s", w.Body.String())
	}
	return arn
}

// mustCreateCluster creates a DB cluster and returns its ARN.
func mustCreateCluster(t *testing.T, handler http.Handler, id, engine string) string {
	t.Helper()
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, rdsReq(t, "CreateDBCluster", url.Values{
		"DBClusterIdentifier": {id},
		"Engine":              {engine},
		"MasterUsername":      {"admin"},
		"MasterUserPassword":  {"password123"},
		"DatabaseName":        {"mydb"},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateDBCluster %s: expected 200, got %d\nbody: %s", id, w.Code, w.Body.String())
	}
	var resp struct {
		Result struct {
			DBCluster struct {
				DBClusterArn string `xml:"DBClusterArn"`
			} `xml:"DBCluster"`
		} `xml:"CreateDBClusterResult"`
	}
	if err := xml.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("CreateDBCluster: unmarshal: %v\nbody: %s", err, w.Body.String())
	}
	arn := resp.Result.DBCluster.DBClusterArn
	if arn == "" {
		t.Fatalf("CreateDBCluster: DBClusterArn is empty\nbody: %s", w.Body.String())
	}
	return arn
}

// ---- Test 1: CreateDBInstance + DescribeDBInstances ----

func TestRDS_CreateAndDescribeDBInstances(t *testing.T) {
	handler := newRDSGateway(t)

	arn1 := mustCreateInstance(t, handler, "my-mysql-db", "db.t3.micro", "mysql")
	arn2 := mustCreateInstance(t, handler, "my-postgres-db", "db.t3.small", "postgres")

	if !strings.HasPrefix(arn1, "arn:aws:rds:") {
		t.Errorf("CreateDBInstance: expected ARN prefix arn:aws:rds:, got %s", arn1)
	}
	if !strings.Contains(arn1, "my-mysql-db") {
		t.Errorf("CreateDBInstance: expected ARN to contain identifier, got %s", arn1)
	}
	if !strings.Contains(arn2, "my-postgres-db") {
		t.Errorf("CreateDBInstance: expected ARN to contain identifier, got %s", arn2)
	}

	// DescribeDBInstances — all
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, rdsReq(t, "DescribeDBInstances", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeDBInstances: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "my-mysql-db") {
		t.Errorf("DescribeDBInstances: expected my-mysql-db\nbody: %s", body)
	}
	if !strings.Contains(body, "my-postgres-db") {
		t.Errorf("DescribeDBInstances: expected my-postgres-db\nbody: %s", body)
	}
	if !strings.Contains(body, "available") {
		t.Errorf("DescribeDBInstances: expected status available\nbody: %s", body)
	}

	// DescribeDBInstances — by ID
	wf := httptest.NewRecorder()
	handler.ServeHTTP(wf, rdsReq(t, "DescribeDBInstances", url.Values{
		"DBInstanceIdentifier": {"my-mysql-db"},
	}))
	if wf.Code != http.StatusOK {
		t.Fatalf("DescribeDBInstances by ID: expected 200, got %d\nbody: %s", wf.Code, wf.Body.String())
	}
	filterBody := wf.Body.String()
	if !strings.Contains(filterBody, "my-mysql-db") {
		t.Errorf("DescribeDBInstances filter: expected my-mysql-db\nbody: %s", filterBody)
	}
	if strings.Contains(filterBody, "my-postgres-db") {
		t.Errorf("DescribeDBInstances filter: my-postgres-db should be excluded\nbody: %s", filterBody)
	}

	// Verify endpoint and port for mysql (3306)
	var descResp struct {
		Result struct {
			Instances []struct {
				Endpoint struct {
					Port int `xml:"Port"`
				} `xml:"Endpoint"`
				Engine string `xml:"Engine"`
			} `xml:"DBInstances>DBInstance"`
		} `xml:"DescribeDBInstancesResult"`
	}
	if err := xml.Unmarshal([]byte(filterBody), &descResp); err != nil {
		t.Fatalf("DescribeDBInstances: unmarshal: %v", err)
	}
	if len(descResp.Result.Instances) == 0 {
		t.Fatal("DescribeDBInstances: no instances returned")
	}
	inst := descResp.Result.Instances[0]
	if inst.Endpoint.Port != 3306 {
		t.Errorf("CreateDBInstance mysql: expected port 3306, got %d", inst.Endpoint.Port)
	}

	// Describe non-existent instance
	wne := httptest.NewRecorder()
	handler.ServeHTTP(wne, rdsReq(t, "DescribeDBInstances", url.Values{
		"DBInstanceIdentifier": {"no-such-db"},
	}))
	if wne.Code != http.StatusNotFound {
		t.Errorf("DescribeDBInstances non-existent: expected 404, got %d\nbody: %s", wne.Code, wne.Body.String())
	}
}

// ---- Test 2: ModifyDBInstance ----

func TestRDS_ModifyDBInstance(t *testing.T) {
	handler := newRDSGateway(t)

	mustCreateInstance(t, handler, "mod-db", "db.t3.micro", "mysql")

	// ModifyDBInstance — change class and storage
	wm := httptest.NewRecorder()
	handler.ServeHTTP(wm, rdsReq(t, "ModifyDBInstance", url.Values{
		"DBInstanceIdentifier": {"mod-db"},
		"DBInstanceClass":      {"db.t3.medium"},
		"AllocatedStorage":     {"100"},
		"ApplyImmediately":     {"true"},
	}))
	if wm.Code != http.StatusOK {
		t.Fatalf("ModifyDBInstance: expected 200, got %d\nbody: %s", wm.Code, wm.Body.String())
	}
	modBody := wm.Body.String()
	if !strings.Contains(modBody, "db.t3.medium") {
		t.Errorf("ModifyDBInstance: expected db.t3.medium in response\nbody: %s", modBody)
	}
	if !strings.Contains(modBody, "100") {
		t.Errorf("ModifyDBInstance: expected AllocatedStorage 100 in response\nbody: %s", modBody)
	}

	// Verify via DescribeDBInstances
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, rdsReq(t, "DescribeDBInstances", url.Values{
		"DBInstanceIdentifier": {"mod-db"},
	}))
	if wd.Code != http.StatusOK {
		t.Fatalf("DescribeDBInstances after modify: expected 200, got %d", wd.Code)
	}
	descBody := wd.Body.String()
	if !strings.Contains(descBody, "db.t3.medium") {
		t.Errorf("ModifyDBInstance: class not updated in describe response\nbody: %s", descBody)
	}

	// Modify non-existent instance
	wne := httptest.NewRecorder()
	handler.ServeHTTP(wne, rdsReq(t, "ModifyDBInstance", url.Values{
		"DBInstanceIdentifier": {"no-such-db"},
		"DBInstanceClass":      {"db.t3.large"},
	}))
	if wne.Code != http.StatusNotFound {
		t.Errorf("ModifyDBInstance non-existent: expected 404, got %d\nbody: %s", wne.Code, wne.Body.String())
	}
}

// ---- Test 3: CreateDBCluster + DescribeDBClusters ----

func TestRDS_CreateAndDescribeDBClusters(t *testing.T) {
	handler := newRDSGateway(t)

	arn1 := mustCreateCluster(t, handler, "aurora-cluster-1", "aurora-mysql")
	arn2 := mustCreateCluster(t, handler, "aurora-cluster-2", "aurora-postgresql")

	if !strings.HasPrefix(arn1, "arn:aws:rds:") {
		t.Errorf("CreateDBCluster: expected ARN prefix arn:aws:rds:, got %s", arn1)
	}
	if !strings.Contains(arn1, "aurora-cluster-1") {
		t.Errorf("CreateDBCluster: expected ARN to contain identifier, got %s", arn1)
	}
	_ = arn2

	// DescribeDBClusters — all
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, rdsReq(t, "DescribeDBClusters", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeDBClusters: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "aurora-cluster-1") {
		t.Errorf("DescribeDBClusters: expected aurora-cluster-1\nbody: %s", body)
	}
	if !strings.Contains(body, "aurora-cluster-2") {
		t.Errorf("DescribeDBClusters: expected aurora-cluster-2\nbody: %s", body)
	}
	if !strings.Contains(body, "available") {
		t.Errorf("DescribeDBClusters: expected status available\nbody: %s", body)
	}

	// DescribeDBClusters — by ID
	wf := httptest.NewRecorder()
	handler.ServeHTTP(wf, rdsReq(t, "DescribeDBClusters", url.Values{
		"DBClusterIdentifier": {"aurora-cluster-1"},
	}))
	if wf.Code != http.StatusOK {
		t.Fatalf("DescribeDBClusters by ID: expected 200, got %d\nbody: %s", wf.Code, wf.Body.String())
	}
	filterBody := wf.Body.String()
	if !strings.Contains(filterBody, "aurora-cluster-1") {
		t.Errorf("DescribeDBClusters filter: expected aurora-cluster-1\nbody: %s", filterBody)
	}
	if strings.Contains(filterBody, "aurora-cluster-2") {
		t.Errorf("DescribeDBClusters filter: aurora-cluster-2 should be excluded\nbody: %s", filterBody)
	}

	// Verify aurora-mysql uses port 3306
	var clResp struct {
		Result struct {
			Clusters []struct {
				Port int `xml:"Port"`
			} `xml:"DBClusters>DBCluster"`
		} `xml:"DescribeDBClustersResult"`
	}
	if err := xml.Unmarshal([]byte(filterBody), &clResp); err != nil {
		t.Fatalf("DescribeDBClusters: unmarshal: %v", err)
	}
	if len(clResp.Result.Clusters) == 0 {
		t.Fatal("DescribeDBClusters: no clusters returned")
	}
	if clResp.Result.Clusters[0].Port != 3306 {
		t.Errorf("CreateDBCluster aurora-mysql: expected port 3306, got %d", clResp.Result.Clusters[0].Port)
	}

	// Describe non-existent cluster
	wne := httptest.NewRecorder()
	handler.ServeHTTP(wne, rdsReq(t, "DescribeDBClusters", url.Values{
		"DBClusterIdentifier": {"no-such-cluster"},
	}))
	if wne.Code != http.StatusNotFound {
		t.Errorf("DescribeDBClusters non-existent: expected 404, got %d\nbody: %s", wne.Code, wne.Body.String())
	}
}

// ---- Test 4: CreateDBSnapshot + DescribeDBSnapshots ----

func TestRDS_CreateAndDescribeDBSnapshots(t *testing.T) {
	handler := newRDSGateway(t)

	mustCreateInstance(t, handler, "snap-source-db", "db.t3.micro", "mysql")

	// CreateDBSnapshot
	ws := httptest.NewRecorder()
	handler.ServeHTTP(ws, rdsReq(t, "CreateDBSnapshot", url.Values{
		"DBSnapshotIdentifier": {"my-snapshot-1"},
		"DBInstanceIdentifier": {"snap-source-db"},
	}))
	if ws.Code != http.StatusOK {
		t.Fatalf("CreateDBSnapshot: expected 200, got %d\nbody: %s", ws.Code, ws.Body.String())
	}
	snapBody := ws.Body.String()
	if !strings.Contains(snapBody, "my-snapshot-1") {
		t.Errorf("CreateDBSnapshot: expected identifier in response\nbody: %s", snapBody)
	}
	if !strings.Contains(snapBody, "available") {
		t.Errorf("CreateDBSnapshot: expected status available\nbody: %s", snapBody)
	}
	if !strings.Contains(snapBody, "arn:aws:rds:") {
		t.Errorf("CreateDBSnapshot: expected ARN in response\nbody: %s", snapBody)
	}

	// Create a second snapshot
	ws2 := httptest.NewRecorder()
	handler.ServeHTTP(ws2, rdsReq(t, "CreateDBSnapshot", url.Values{
		"DBSnapshotIdentifier": {"my-snapshot-2"},
		"DBInstanceIdentifier": {"snap-source-db"},
	}))
	if ws2.Code != http.StatusOK {
		t.Fatalf("CreateDBSnapshot 2: expected 200, got %d\nbody: %s", ws2.Code, ws2.Body.String())
	}

	// DescribeDBSnapshots — all
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, rdsReq(t, "DescribeDBSnapshots", nil))
	if wd.Code != http.StatusOK {
		t.Fatalf("DescribeDBSnapshots: expected 200, got %d\nbody: %s", wd.Code, wd.Body.String())
	}
	descBody := wd.Body.String()
	if !strings.Contains(descBody, "my-snapshot-1") {
		t.Errorf("DescribeDBSnapshots: expected my-snapshot-1\nbody: %s", descBody)
	}
	if !strings.Contains(descBody, "my-snapshot-2") {
		t.Errorf("DescribeDBSnapshots: expected my-snapshot-2\nbody: %s", descBody)
	}

	// DescribeDBSnapshots — filter by instance
	wdi := httptest.NewRecorder()
	handler.ServeHTTP(wdi, rdsReq(t, "DescribeDBSnapshots", url.Values{
		"DBInstanceIdentifier": {"snap-source-db"},
	}))
	if wdi.Code != http.StatusOK {
		t.Fatalf("DescribeDBSnapshots by instance: expected 200, got %d", wdi.Code)
	}
	if !strings.Contains(wdi.Body.String(), "my-snapshot-1") {
		t.Errorf("DescribeDBSnapshots by instance: expected my-snapshot-1\nbody: %s", wdi.Body.String())
	}

	// DescribeDBSnapshots — filter by snapshot ID
	wds := httptest.NewRecorder()
	handler.ServeHTTP(wds, rdsReq(t, "DescribeDBSnapshots", url.Values{
		"DBSnapshotIdentifier": {"my-snapshot-1"},
	}))
	if wds.Code != http.StatusOK {
		t.Fatalf("DescribeDBSnapshots by snapshot ID: expected 200, got %d", wds.Code)
	}
	filterSnap := wds.Body.String()
	if !strings.Contains(filterSnap, "my-snapshot-1") {
		t.Errorf("DescribeDBSnapshots filter: expected my-snapshot-1\nbody: %s", filterSnap)
	}
	if strings.Contains(filterSnap, "my-snapshot-2") {
		t.Errorf("DescribeDBSnapshots filter: my-snapshot-2 should be excluded\nbody: %s", filterSnap)
	}

	// Snapshot from non-existent instance
	wne := httptest.NewRecorder()
	handler.ServeHTTP(wne, rdsReq(t, "CreateDBSnapshot", url.Values{
		"DBSnapshotIdentifier": {"bad-snap"},
		"DBInstanceIdentifier": {"no-such-instance"},
	}))
	if wne.Code == http.StatusOK {
		t.Error("CreateDBSnapshot from non-existent instance: expected error, got 200")
	}
}

// ---- Test 5: CreateDBSubnetGroup + DescribeDBSubnetGroups ----

func TestRDS_CreateAndDescribeDBSubnetGroups(t *testing.T) {
	handler := newRDSGateway(t)

	// CreateDBSubnetGroup
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, rdsReq(t, "CreateDBSubnetGroup", url.Values{
		"DBSubnetGroupName":        {"my-subnet-group"},
		"DBSubnetGroupDescription": {"Test subnet group"},
		"SubnetIds.member.1":       {"subnet-12345678"},
		"SubnetIds.member.2":       {"subnet-87654321"},
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("CreateDBSubnetGroup: expected 200, got %d\nbody: %s", wc.Code, wc.Body.String())
	}
	createBody := wc.Body.String()
	if !strings.Contains(createBody, "my-subnet-group") {
		t.Errorf("CreateDBSubnetGroup: expected name in response\nbody: %s", createBody)
	}
	if !strings.Contains(createBody, "Test subnet group") {
		t.Errorf("CreateDBSubnetGroup: expected description in response\nbody: %s", createBody)
	}
	if !strings.Contains(createBody, "arn:aws:rds:") {
		t.Errorf("CreateDBSubnetGroup: expected ARN in response\nbody: %s", createBody)
	}
	if !strings.Contains(createBody, "subnet-12345678") {
		t.Errorf("CreateDBSubnetGroup: expected subnet ID in response\nbody: %s", createBody)
	}

	// Create a second subnet group
	wc2 := httptest.NewRecorder()
	handler.ServeHTTP(wc2, rdsReq(t, "CreateDBSubnetGroup", url.Values{
		"DBSubnetGroupName":        {"another-subnet-group"},
		"DBSubnetGroupDescription": {"Another subnet group"},
		"SubnetIds.member.1":       {"subnet-aabbccdd"},
	}))
	if wc2.Code != http.StatusOK {
		t.Fatalf("CreateDBSubnetGroup 2: expected 200, got %d\nbody: %s", wc2.Code, wc2.Body.String())
	}

	// DescribeDBSubnetGroups — all
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, rdsReq(t, "DescribeDBSubnetGroups", nil))
	if wd.Code != http.StatusOK {
		t.Fatalf("DescribeDBSubnetGroups: expected 200, got %d\nbody: %s", wd.Code, wd.Body.String())
	}
	descBody := wd.Body.String()
	if !strings.Contains(descBody, "my-subnet-group") {
		t.Errorf("DescribeDBSubnetGroups: expected my-subnet-group\nbody: %s", descBody)
	}
	if !strings.Contains(descBody, "another-subnet-group") {
		t.Errorf("DescribeDBSubnetGroups: expected another-subnet-group\nbody: %s", descBody)
	}

	// DescribeDBSubnetGroups — filter by name
	wf := httptest.NewRecorder()
	handler.ServeHTTP(wf, rdsReq(t, "DescribeDBSubnetGroups", url.Values{
		"DBSubnetGroupName": {"my-subnet-group"},
	}))
	if wf.Code != http.StatusOK {
		t.Fatalf("DescribeDBSubnetGroups filter: expected 200, got %d", wf.Code)
	}
	filterBody := wf.Body.String()
	if !strings.Contains(filterBody, "my-subnet-group") {
		t.Errorf("DescribeDBSubnetGroups filter: expected my-subnet-group\nbody: %s", filterBody)
	}
	if strings.Contains(filterBody, "another-subnet-group") {
		t.Errorf("DescribeDBSubnetGroups filter: another-subnet-group should be excluded\nbody: %s", filterBody)
	}

	// Duplicate name
	wdup := httptest.NewRecorder()
	handler.ServeHTTP(wdup, rdsReq(t, "CreateDBSubnetGroup", url.Values{
		"DBSubnetGroupName":        {"my-subnet-group"},
		"DBSubnetGroupDescription": {"dupe"},
		"SubnetIds.member.1":       {"subnet-00000000"},
	}))
	if wdup.Code == http.StatusOK {
		t.Error("CreateDBSubnetGroup duplicate: expected error, got 200")
	}
}

// ---- Test 6: DeleteDBInstance + DeleteDBCluster ----

func TestRDS_DeleteDBInstanceAndCluster(t *testing.T) {
	handler := newRDSGateway(t)

	// Create resources
	mustCreateInstance(t, handler, "delete-me-db", "db.t3.micro", "postgres")
	mustCreateInstance(t, handler, "keep-me-db", "db.t3.micro", "mysql")
	mustCreateCluster(t, handler, "delete-me-cluster", "aurora-mysql")
	mustCreateCluster(t, handler, "keep-me-cluster", "aurora-postgresql")

	// DeleteDBInstance
	wdi := httptest.NewRecorder()
	handler.ServeHTTP(wdi, rdsReq(t, "DeleteDBInstance", url.Values{
		"DBInstanceIdentifier": {"delete-me-db"},
		"SkipFinalSnapshot":    {"true"},
	}))
	if wdi.Code != http.StatusOK {
		t.Fatalf("DeleteDBInstance: expected 200, got %d\nbody: %s", wdi.Code, wdi.Body.String())
	}
	if !strings.Contains(wdi.Body.String(), "delete-me-db") {
		t.Errorf("DeleteDBInstance: expected identifier in response\nbody: %s", wdi.Body.String())
	}

	// Verify instance is gone
	wcheck := httptest.NewRecorder()
	handler.ServeHTTP(wcheck, rdsReq(t, "DescribeDBInstances", url.Values{
		"DBInstanceIdentifier": {"delete-me-db"},
	}))
	if wcheck.Code != http.StatusNotFound {
		t.Errorf("DescribeDBInstances after delete: expected 404, got %d", wcheck.Code)
	}

	// Verify keep-me-db still exists
	wkeep := httptest.NewRecorder()
	handler.ServeHTTP(wkeep, rdsReq(t, "DescribeDBInstances", url.Values{
		"DBInstanceIdentifier": {"keep-me-db"},
	}))
	if wkeep.Code != http.StatusOK {
		t.Errorf("DescribeDBInstances keep-me-db: expected 200, got %d", wkeep.Code)
	}

	// Delete again — should fail
	wdi2 := httptest.NewRecorder()
	handler.ServeHTTP(wdi2, rdsReq(t, "DeleteDBInstance", url.Values{
		"DBInstanceIdentifier": {"delete-me-db"},
		"SkipFinalSnapshot":    {"true"},
	}))
	if wdi2.Code == http.StatusOK {
		t.Error("DeleteDBInstance second time: expected error, got 200")
	}

	// DeleteDBCluster
	wdc := httptest.NewRecorder()
	handler.ServeHTTP(wdc, rdsReq(t, "DeleteDBCluster", url.Values{
		"DBClusterIdentifier": {"delete-me-cluster"},
		"SkipFinalSnapshot":   {"true"},
	}))
	if wdc.Code != http.StatusOK {
		t.Fatalf("DeleteDBCluster: expected 200, got %d\nbody: %s", wdc.Code, wdc.Body.String())
	}
	if !strings.Contains(wdc.Body.String(), "delete-me-cluster") {
		t.Errorf("DeleteDBCluster: expected identifier in response\nbody: %s", wdc.Body.String())
	}

	// Verify cluster is gone
	wcc := httptest.NewRecorder()
	handler.ServeHTTP(wcc, rdsReq(t, "DescribeDBClusters", url.Values{
		"DBClusterIdentifier": {"delete-me-cluster"},
	}))
	if wcc.Code != http.StatusNotFound {
		t.Errorf("DescribeDBClusters after delete: expected 404, got %d", wcc.Code)
	}

	// Verify keep-me-cluster still exists
	wkc := httptest.NewRecorder()
	handler.ServeHTTP(wkc, rdsReq(t, "DescribeDBClusters", url.Values{
		"DBClusterIdentifier": {"keep-me-cluster"},
	}))
	if wkc.Code != http.StatusOK {
		t.Errorf("DescribeDBClusters keep-me-cluster: expected 200, got %d", wkc.Code)
	}

	// Delete non-existent cluster
	wdc2 := httptest.NewRecorder()
	handler.ServeHTTP(wdc2, rdsReq(t, "DeleteDBCluster", url.Values{
		"DBClusterIdentifier": {"no-such-cluster"},
		"SkipFinalSnapshot":   {"true"},
	}))
	if wdc2.Code == http.StatusOK {
		t.Error("DeleteDBCluster non-existent: expected error, got 200")
	}
}

// ---- Test 7: Tags — AddTagsToResource / ListTagsForResource / RemoveTagsFromResource ----

func TestRDS_Tags(t *testing.T) {
	handler := newRDSGateway(t)

	instARN := mustCreateInstance(t, handler, "tag-test-db", "db.t3.micro", "mysql")

	// AddTagsToResource
	wt := httptest.NewRecorder()
	handler.ServeHTTP(wt, rdsReq(t, "AddTagsToResource", url.Values{
		"ResourceName":        {instARN},
		"Tags.member.1.Key":   {"env"},
		"Tags.member.1.Value": {"production"},
		"Tags.member.2.Key":   {"team"},
		"Tags.member.2.Value": {"platform"},
	}))
	if wt.Code != http.StatusOK {
		t.Fatalf("AddTagsToResource: expected 200, got %d\nbody: %s", wt.Code, wt.Body.String())
	}

	// ListTagsForResource
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, rdsReq(t, "ListTagsForResource", url.Values{
		"ResourceName": {instARN},
	}))
	if wl.Code != http.StatusOK {
		t.Fatalf("ListTagsForResource: expected 200, got %d\nbody: %s", wl.Code, wl.Body.String())
	}
	listBody := wl.Body.String()
	if !strings.Contains(listBody, "env") || !strings.Contains(listBody, "production") {
		t.Errorf("ListTagsForResource: expected env=production\nbody: %s", listBody)
	}
	if !strings.Contains(listBody, "team") || !strings.Contains(listBody, "platform") {
		t.Errorf("ListTagsForResource: expected team=platform\nbody: %s", listBody)
	}

	// RemoveTagsFromResource
	wr := httptest.NewRecorder()
	handler.ServeHTTP(wr, rdsReq(t, "RemoveTagsFromResource", url.Values{
		"ResourceName":      {instARN},
		"TagKeys.member.1":  {"env"},
	}))
	if wr.Code != http.StatusOK {
		t.Fatalf("RemoveTagsFromResource: expected 200, got %d\nbody: %s", wr.Code, wr.Body.String())
	}

	// Verify tag removed
	wl2 := httptest.NewRecorder()
	handler.ServeHTTP(wl2, rdsReq(t, "ListTagsForResource", url.Values{
		"ResourceName": {instARN},
	}))
	if wl2.Code != http.StatusOK {
		t.Fatalf("ListTagsForResource after remove: expected 200, got %d", wl2.Code)
	}
	list2Body := wl2.Body.String()
	if strings.Contains(list2Body, ">env<") {
		t.Errorf("RemoveTagsFromResource: env tag should be gone\nbody: %s", list2Body)
	}
	if !strings.Contains(list2Body, "team") {
		t.Errorf("RemoveTagsFromResource: team tag should still be present\nbody: %s", list2Body)
	}

	// ListTagsForResource on non-existent ARN
	wne := httptest.NewRecorder()
	handler.ServeHTTP(wne, rdsReq(t, "ListTagsForResource", url.Values{
		"ResourceName": {"arn:aws:rds:us-east-1:000000000000:db:no-such"},
	}))
	if wne.Code != http.StatusNotFound {
		t.Errorf("ListTagsForResource non-existent: expected 404, got %d\nbody: %s", wne.Code, wne.Body.String())
	}
}

// ---- Test 8: Unknown action ----

func TestRDS_UnknownAction(t *testing.T) {
	handler := newRDSGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, rdsReq(t, "NonExistentAction", nil))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("unknown action: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
}
