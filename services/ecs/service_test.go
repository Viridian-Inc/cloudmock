package ecs_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/gateway"
	"github.com/neureaux/cloudmock/pkg/routing"
	ecssvc "github.com/neureaux/cloudmock/services/ecs"
)

// newECSGateway builds a full gateway stack with the ECS service registered and IAM disabled.
func newECSGateway(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"

	reg := routing.NewRegistry()
	reg.Register(ecssvc.New(cfg.AccountID, cfg.Region))

	return gateway.New(cfg, reg)
}

// ecsReq builds a JSON POST request targeting the ECS service via X-Amz-Target.
func ecsReq(t *testing.T, action string, body interface{}) *http.Request {
	t.Helper()

	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("ecsReq: marshal body: %v", err)
		}
	} else {
		bodyBytes = []byte("{}")
	}

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AmazonEC2ContainerServiceV20141113."+action)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/ecs/aws4_request, SignedHeaders=host;x-amz-target, Signature=abc123")
	return req
}

// decodeJSON is a test helper that unmarshals JSON into a map.
func decodeJSON(t *testing.T, data string) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(data), &m); err != nil {
		t.Fatalf("decodeJSON: %v\nbody: %s", err, data)
	}
	return m
}

// ---- Test 1: CreateCluster + ListClusters + DescribeClusters ----

func TestECS_ClusterLifecycle(t *testing.T) {
	handler := newECSGateway(t)

	// CreateCluster.
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, ecsReq(t, "CreateCluster", map[string]interface{}{
		"clusterName": "my-cluster",
		"tags": []map[string]string{
			{"key": "env", "value": "test"},
		},
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("CreateCluster: expected 200, got %d\nbody: %s", wc.Code, wc.Body.String())
	}

	mc := decodeJSON(t, wc.Body.String())
	cluster, ok := mc["cluster"].(map[string]interface{})
	if !ok {
		t.Fatalf("CreateCluster: missing cluster in response\nbody: %s", wc.Body.String())
	}
	if cluster["clusterName"].(string) != "my-cluster" {
		t.Errorf("CreateCluster: expected clusterName=my-cluster, got %q", cluster["clusterName"])
	}
	arn, _ := cluster["clusterArn"].(string)
	if !strings.Contains(arn, "my-cluster") {
		t.Errorf("CreateCluster: ARN %q does not contain cluster name", arn)
	}
	if cluster["status"].(string) != "ACTIVE" {
		t.Errorf("CreateCluster: expected status=ACTIVE, got %q", cluster["status"])
	}

	// ListClusters.
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, ecsReq(t, "ListClusters", nil))
	if wl.Code != http.StatusOK {
		t.Fatalf("ListClusters: expected 200, got %d\nbody: %s", wl.Code, wl.Body.String())
	}
	ml := decodeJSON(t, wl.Body.String())
	clusterArns, ok := ml["clusterArns"].([]interface{})
	if !ok || len(clusterArns) == 0 {
		t.Fatalf("ListClusters: expected non-empty clusterArns\nbody: %s", wl.Body.String())
	}
	found := false
	for _, a := range clusterArns {
		if strings.Contains(a.(string), "my-cluster") {
			found = true
			break
		}
	}
	if !found {
		t.Error("ListClusters: my-cluster ARN not found")
	}

	// DescribeClusters — by name.
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, ecsReq(t, "DescribeClusters", map[string]interface{}{
		"clusters": []string{"my-cluster"},
	}))
	if wd.Code != http.StatusOK {
		t.Fatalf("DescribeClusters: expected 200, got %d\nbody: %s", wd.Code, wd.Body.String())
	}
	md := decodeJSON(t, wd.Body.String())
	clusters, ok := md["clusters"].([]interface{})
	if !ok || len(clusters) == 0 {
		t.Fatalf("DescribeClusters: expected non-empty clusters\nbody: %s", wd.Body.String())
	}
	entry := clusters[0].(map[string]interface{})
	if entry["clusterName"].(string) != "my-cluster" {
		t.Errorf("DescribeClusters: expected clusterName=my-cluster, got %q", entry["clusterName"])
	}

	// DescribeClusters — all (no filter).
	wda := httptest.NewRecorder()
	handler.ServeHTTP(wda, ecsReq(t, "DescribeClusters", map[string]interface{}{}))
	if wda.Code != http.StatusOK {
		t.Fatalf("DescribeClusters all: expected 200, got %d\nbody: %s", wda.Code, wda.Body.String())
	}
	mda := decodeJSON(t, wda.Body.String())
	allClusters, ok := mda["clusters"].([]interface{})
	if !ok || len(allClusters) == 0 {
		t.Fatalf("DescribeClusters all: expected non-empty clusters")
	}
}

// ---- Test 2: RegisterTaskDefinition + DescribeTaskDefinition + ListTaskDefinitions ----

func TestECS_TaskDefinitionLifecycle(t *testing.T) {
	handler := newECSGateway(t)

	// Register revision 1.
	wr1 := httptest.NewRecorder()
	handler.ServeHTTP(wr1, ecsReq(t, "RegisterTaskDefinition", map[string]interface{}{
		"family": "web-app",
		"containerDefinitions": []map[string]interface{}{
			{
				"name":      "web",
				"image":     "nginx:latest",
				"cpu":       256,
				"memory":    512,
				"essential": true,
				"portMappings": []map[string]interface{}{
					{"containerPort": 80, "hostPort": 80, "protocol": "tcp"},
				},
			},
		},
		"networkMode":             "awsvpc",
		"requiresCompatibilities": []string{"FARGATE"},
		"cpu":    "256",
		"memory": "512",
	}))
	if wr1.Code != http.StatusOK {
		t.Fatalf("RegisterTaskDefinition rev1: expected 200, got %d\nbody: %s", wr1.Code, wr1.Body.String())
	}
	mr1 := decodeJSON(t, wr1.Body.String())
	td1, ok := mr1["taskDefinition"].(map[string]interface{})
	if !ok {
		t.Fatalf("RegisterTaskDefinition rev1: missing taskDefinition\nbody: %s", wr1.Body.String())
	}
	if td1["family"].(string) != "web-app" {
		t.Errorf("RegisterTaskDefinition rev1: expected family=web-app, got %q", td1["family"])
	}
	if td1["revision"].(float64) != 1 {
		t.Errorf("RegisterTaskDefinition rev1: expected revision=1, got %v", td1["revision"])
	}
	if td1["status"].(string) != "ACTIVE" {
		t.Errorf("RegisterTaskDefinition rev1: expected status=ACTIVE, got %q", td1["status"])
	}
	arn1, _ := td1["taskDefinitionArn"].(string)
	if !strings.Contains(arn1, "web-app:1") {
		t.Errorf("RegisterTaskDefinition rev1: ARN %q does not contain web-app:1", arn1)
	}

	// Register revision 2 — verify increment.
	wr2 := httptest.NewRecorder()
	handler.ServeHTTP(wr2, ecsReq(t, "RegisterTaskDefinition", map[string]interface{}{
		"family": "web-app",
		"containerDefinitions": []map[string]interface{}{
			{"name": "web", "image": "nginx:1.25", "cpu": 256, "memory": 512, "essential": true},
		},
		"networkMode": "awsvpc",
		"cpu":         "512",
		"memory":      "1024",
	}))
	if wr2.Code != http.StatusOK {
		t.Fatalf("RegisterTaskDefinition rev2: expected 200, got %d\nbody: %s", wr2.Code, wr2.Body.String())
	}
	mr2 := decodeJSON(t, wr2.Body.String())
	td2 := mr2["taskDefinition"].(map[string]interface{})
	if td2["revision"].(float64) != 2 {
		t.Errorf("RegisterTaskDefinition rev2: expected revision=2, got %v", td2["revision"])
	}
	arn2, _ := td2["taskDefinitionArn"].(string)
	if !strings.Contains(arn2, "web-app:2") {
		t.Errorf("RegisterTaskDefinition rev2: ARN %q does not contain web-app:2", arn2)
	}

	// DescribeTaskDefinition — by family:revision.
	wdesc := httptest.NewRecorder()
	handler.ServeHTTP(wdesc, ecsReq(t, "DescribeTaskDefinition", map[string]interface{}{
		"taskDefinition": "web-app:1",
	}))
	if wdesc.Code != http.StatusOK {
		t.Fatalf("DescribeTaskDefinition: expected 200, got %d\nbody: %s", wdesc.Code, wdesc.Body.String())
	}
	mdesc := decodeJSON(t, wdesc.Body.String())
	tdDesc := mdesc["taskDefinition"].(map[string]interface{})
	if tdDesc["revision"].(float64) != 1 {
		t.Errorf("DescribeTaskDefinition: expected revision=1")
	}

	// ListTaskDefinitions — all.
	wlist := httptest.NewRecorder()
	handler.ServeHTTP(wlist, ecsReq(t, "ListTaskDefinitions", map[string]interface{}{}))
	if wlist.Code != http.StatusOK {
		t.Fatalf("ListTaskDefinitions: expected 200, got %d\nbody: %s", wlist.Code, wlist.Body.String())
	}
	mlist := decodeJSON(t, wlist.Body.String())
	tdArns, ok := mlist["taskDefinitionArns"].([]interface{})
	if !ok || len(tdArns) < 2 {
		t.Fatalf("ListTaskDefinitions: expected 2+ ARNs, got %v\nbody: %s", tdArns, wlist.Body.String())
	}

	// ListTaskDefinitions — family prefix.
	wprefix := httptest.NewRecorder()
	handler.ServeHTTP(wprefix, ecsReq(t, "ListTaskDefinitions", map[string]interface{}{
		"familyPrefix": "web-app",
	}))
	if wprefix.Code != http.StatusOK {
		t.Fatalf("ListTaskDefinitions prefix: expected 200, got %d\nbody: %s", wprefix.Code, wprefix.Body.String())
	}
	mprefix := decodeJSON(t, wprefix.Body.String())
	prefixArns, ok := mprefix["taskDefinitionArns"].([]interface{})
	if !ok || len(prefixArns) == 0 {
		t.Fatalf("ListTaskDefinitions prefix: expected results for web-app prefix\nbody: %s", wprefix.Body.String())
	}
}

// ---- Test 3: CreateService + ListServices + DescribeServices ----

func TestECS_ServiceLifecycle(t *testing.T) {
	handler := newECSGateway(t)

	// Need a cluster first.
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, ecsReq(t, "CreateCluster", map[string]interface{}{
		"clusterName": "svc-cluster",
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("setup CreateCluster: %d %s", wc.Code, wc.Body.String())
	}

	// Register a task definition.
	wt := httptest.NewRecorder()
	handler.ServeHTTP(wt, ecsReq(t, "RegisterTaskDefinition", map[string]interface{}{
		"family": "svc-task",
		"containerDefinitions": []map[string]interface{}{
			{"name": "app", "image": "myapp:latest", "cpu": 256, "memory": 512, "essential": true},
		},
	}))
	if wt.Code != http.StatusOK {
		t.Fatalf("setup RegisterTaskDefinition: %d %s", wt.Code, wt.Body.String())
	}

	// CreateService.
	ws := httptest.NewRecorder()
	handler.ServeHTTP(ws, ecsReq(t, "CreateService", map[string]interface{}{
		"cluster":        "svc-cluster",
		"serviceName":    "my-service",
		"taskDefinition": "svc-task:1",
		"desiredCount":   3,
		"launchType":     "FARGATE",
	}))
	if ws.Code != http.StatusOK {
		t.Fatalf("CreateService: expected 200, got %d\nbody: %s", ws.Code, ws.Body.String())
	}
	ms := decodeJSON(t, ws.Body.String())
	svc, ok := ms["service"].(map[string]interface{})
	if !ok {
		t.Fatalf("CreateService: missing service\nbody: %s", ws.Body.String())
	}
	if svc["serviceName"].(string) != "my-service" {
		t.Errorf("CreateService: expected serviceName=my-service, got %q", svc["serviceName"])
	}
	if svc["desiredCount"].(float64) != 3 {
		t.Errorf("CreateService: expected desiredCount=3, got %v", svc["desiredCount"])
	}
	if svc["status"].(string) != "ACTIVE" {
		t.Errorf("CreateService: expected status=ACTIVE, got %q", svc["status"])
	}
	svcARN, _ := svc["serviceArn"].(string)
	if !strings.Contains(svcARN, "my-service") {
		t.Errorf("CreateService: ARN %q does not contain service name", svcARN)
	}

	// ListServices.
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, ecsReq(t, "ListServices", map[string]interface{}{
		"cluster": "svc-cluster",
	}))
	if wl.Code != http.StatusOK {
		t.Fatalf("ListServices: expected 200, got %d\nbody: %s", wl.Code, wl.Body.String())
	}
	ml := decodeJSON(t, wl.Body.String())
	svcARNs, ok := ml["serviceArns"].([]interface{})
	if !ok || len(svcARNs) == 0 {
		t.Fatalf("ListServices: expected non-empty serviceArns\nbody: %s", wl.Body.String())
	}

	// DescribeServices.
	wdesc := httptest.NewRecorder()
	handler.ServeHTTP(wdesc, ecsReq(t, "DescribeServices", map[string]interface{}{
		"cluster":  "svc-cluster",
		"services": []string{"my-service"},
	}))
	if wdesc.Code != http.StatusOK {
		t.Fatalf("DescribeServices: expected 200, got %d\nbody: %s", wdesc.Code, wdesc.Body.String())
	}
	mdesc := decodeJSON(t, wdesc.Body.String())
	svcs, ok := mdesc["services"].([]interface{})
	if !ok || len(svcs) == 0 {
		t.Fatalf("DescribeServices: expected non-empty services\nbody: %s", wdesc.Body.String())
	}
	svcEntry := svcs[0].(map[string]interface{})
	if svcEntry["serviceName"].(string) != "my-service" {
		t.Errorf("DescribeServices: expected serviceName=my-service")
	}
}

// ---- Test 4: RunTask + DescribeTasks + StopTask ----

func TestECS_TaskLifecycle(t *testing.T) {
	handler := newECSGateway(t)

	// Setup cluster + task definition.
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, ecsReq(t, "CreateCluster", map[string]interface{}{
		"clusterName": "task-cluster",
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("setup CreateCluster: %d %s", wc.Code, wc.Body.String())
	}

	wt := httptest.NewRecorder()
	handler.ServeHTTP(wt, ecsReq(t, "RegisterTaskDefinition", map[string]interface{}{
		"family": "runner-task",
		"containerDefinitions": []map[string]interface{}{
			{"name": "runner", "image": "myrunner:latest", "cpu": 256, "memory": 512, "essential": true},
		},
	}))
	if wt.Code != http.StatusOK {
		t.Fatalf("setup RegisterTaskDefinition: %d %s", wt.Code, wt.Body.String())
	}

	// RunTask.
	wr := httptest.NewRecorder()
	handler.ServeHTTP(wr, ecsReq(t, "RunTask", map[string]interface{}{
		"cluster":        "task-cluster",
		"taskDefinition": "runner-task:1",
		"count":          2,
	}))
	if wr.Code != http.StatusOK {
		t.Fatalf("RunTask: expected 200, got %d\nbody: %s", wr.Code, wr.Body.String())
	}
	mr := decodeJSON(t, wr.Body.String())
	tasks, ok := mr["tasks"].([]interface{})
	if !ok || len(tasks) != 2 {
		t.Fatalf("RunTask: expected 2 tasks, got %v\nbody: %s", tasks, wr.Body.String())
	}
	task0 := tasks[0].(map[string]interface{})
	taskARN, _ := task0["taskArn"].(string)
	if taskARN == "" {
		t.Fatal("RunTask: missing taskArn")
	}
	if task0["lastStatus"].(string) != "RUNNING" {
		t.Errorf("RunTask: expected lastStatus=RUNNING, got %q", task0["lastStatus"])
	}
	if task0["desiredStatus"].(string) != "RUNNING" {
		t.Errorf("RunTask: expected desiredStatus=RUNNING")
	}

	// ListTasks.
	wlist := httptest.NewRecorder()
	handler.ServeHTTP(wlist, ecsReq(t, "ListTasks", map[string]interface{}{
		"cluster": "task-cluster",
	}))
	if wlist.Code != http.StatusOK {
		t.Fatalf("ListTasks: expected 200, got %d\nbody: %s", wlist.Code, wlist.Body.String())
	}
	mlist := decodeJSON(t, wlist.Body.String())
	taskARNs, ok := mlist["taskArns"].([]interface{})
	if !ok || len(taskARNs) < 2 {
		t.Fatalf("ListTasks: expected 2+ task ARNs, got %v\nbody: %s", taskARNs, wlist.Body.String())
	}

	// DescribeTasks.
	wdesc := httptest.NewRecorder()
	handler.ServeHTTP(wdesc, ecsReq(t, "DescribeTasks", map[string]interface{}{
		"cluster": "task-cluster",
		"tasks":   []string{taskARN},
	}))
	if wdesc.Code != http.StatusOK {
		t.Fatalf("DescribeTasks: expected 200, got %d\nbody: %s", wdesc.Code, wdesc.Body.String())
	}
	mdesc := decodeJSON(t, wdesc.Body.String())
	describedTasks, ok := mdesc["tasks"].([]interface{})
	if !ok || len(describedTasks) == 0 {
		t.Fatalf("DescribeTasks: expected non-empty tasks\nbody: %s", wdesc.Body.String())
	}
	dt := describedTasks[0].(map[string]interface{})
	if dt["taskArn"].(string) != taskARN {
		t.Errorf("DescribeTasks: ARN mismatch")
	}

	// StopTask.
	wstop := httptest.NewRecorder()
	handler.ServeHTTP(wstop, ecsReq(t, "StopTask", map[string]interface{}{
		"cluster": "task-cluster",
		"task":    taskARN,
		"reason":  "testing stop",
	}))
	if wstop.Code != http.StatusOK {
		t.Fatalf("StopTask: expected 200, got %d\nbody: %s", wstop.Code, wstop.Body.String())
	}
	mstop := decodeJSON(t, wstop.Body.String())
	stoppedTask := mstop["task"].(map[string]interface{})
	if stoppedTask["lastStatus"].(string) != "STOPPED" {
		t.Errorf("StopTask: expected lastStatus=STOPPED, got %q", stoppedTask["lastStatus"])
	}
	if stoppedTask["stoppedReason"].(string) != "testing stop" {
		t.Errorf("StopTask: expected stoppedReason=%q, got %q", "testing stop", stoppedTask["stoppedReason"])
	}
}

// ---- Test 5: UpdateService ----

func TestECS_UpdateService(t *testing.T) {
	handler := newECSGateway(t)

	// Setup.
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, ecsReq(t, "CreateCluster", map[string]interface{}{
		"clusterName": "update-cluster",
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("setup CreateCluster: %d %s", wc.Code, wc.Body.String())
	}

	wt := httptest.NewRecorder()
	handler.ServeHTTP(wt, ecsReq(t, "RegisterTaskDefinition", map[string]interface{}{
		"family": "update-task",
		"containerDefinitions": []map[string]interface{}{
			{"name": "app", "image": "myapp:v1", "cpu": 256, "memory": 512, "essential": true},
		},
	}))
	if wt.Code != http.StatusOK {
		t.Fatalf("setup RegisterTaskDefinition: %d %s", wt.Code, wt.Body.String())
	}

	ws := httptest.NewRecorder()
	handler.ServeHTTP(ws, ecsReq(t, "CreateService", map[string]interface{}{
		"cluster":        "update-cluster",
		"serviceName":    "update-svc",
		"taskDefinition": "update-task:1",
		"desiredCount":   1,
	}))
	if ws.Code != http.StatusOK {
		t.Fatalf("setup CreateService: %d %s", ws.Code, ws.Body.String())
	}

	// Register a new task definition revision.
	wt2 := httptest.NewRecorder()
	handler.ServeHTTP(wt2, ecsReq(t, "RegisterTaskDefinition", map[string]interface{}{
		"family": "update-task",
		"containerDefinitions": []map[string]interface{}{
			{"name": "app", "image": "myapp:v2", "cpu": 256, "memory": 512, "essential": true},
		},
	}))
	if wt2.Code != http.StatusOK {
		t.Fatalf("setup RegisterTaskDefinition rev2: %d %s", wt2.Code, wt2.Body.String())
	}

	// UpdateService — change desired count and task definition.
	desiredCount := 5
	wu := httptest.NewRecorder()
	handler.ServeHTTP(wu, ecsReq(t, "UpdateService", map[string]interface{}{
		"cluster":        "update-cluster",
		"service":        "update-svc",
		"desiredCount":   desiredCount,
		"taskDefinition": "update-task:2",
	}))
	if wu.Code != http.StatusOK {
		t.Fatalf("UpdateService: expected 200, got %d\nbody: %s", wu.Code, wu.Body.String())
	}
	mu := decodeJSON(t, wu.Body.String())
	updatedSvc, ok := mu["service"].(map[string]interface{})
	if !ok {
		t.Fatalf("UpdateService: missing service\nbody: %s", wu.Body.String())
	}
	if updatedSvc["desiredCount"].(float64) != float64(desiredCount) {
		t.Errorf("UpdateService: expected desiredCount=%d, got %v", desiredCount, updatedSvc["desiredCount"])
	}
	if updatedSvc["taskDefinition"].(string) != "update-task:2" {
		t.Errorf("UpdateService: expected taskDefinition=update-task:2, got %q", updatedSvc["taskDefinition"])
	}
}

// ---- Test 6: DeleteService + DeleteCluster ----

func TestECS_DeleteServiceAndCluster(t *testing.T) {
	handler := newECSGateway(t)

	// Setup cluster.
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, ecsReq(t, "CreateCluster", map[string]interface{}{
		"clusterName": "del-cluster",
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("setup CreateCluster: %d %s", wc.Code, wc.Body.String())
	}

	// Setup service.
	ws := httptest.NewRecorder()
	handler.ServeHTTP(ws, ecsReq(t, "CreateService", map[string]interface{}{
		"cluster":        "del-cluster",
		"serviceName":    "del-svc",
		"taskDefinition": "some-task:1",
		"desiredCount":   1,
	}))
	if ws.Code != http.StatusOK {
		t.Fatalf("setup CreateService: %d %s", ws.Code, ws.Body.String())
	}

	// DeleteService.
	wds := httptest.NewRecorder()
	handler.ServeHTTP(wds, ecsReq(t, "DeleteService", map[string]interface{}{
		"cluster": "del-cluster",
		"service": "del-svc",
		"force":   true,
	}))
	if wds.Code != http.StatusOK {
		t.Fatalf("DeleteService: expected 200, got %d\nbody: %s", wds.Code, wds.Body.String())
	}
	mds := decodeJSON(t, wds.Body.String())
	delSvc, ok := mds["service"].(map[string]interface{})
	if !ok {
		t.Fatalf("DeleteService: missing service in response\nbody: %s", wds.Body.String())
	}
	if delSvc["serviceName"].(string) != "del-svc" {
		t.Errorf("DeleteService: expected serviceName=del-svc")
	}

	// Verify service is gone — ListServices should be empty.
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, ecsReq(t, "ListServices", map[string]interface{}{
		"cluster": "del-cluster",
	}))
	if wl.Code != http.StatusOK {
		t.Fatalf("ListServices after delete: expected 200, got %d\nbody: %s", wl.Code, wl.Body.String())
	}
	ml := decodeJSON(t, wl.Body.String())
	svcARNs, _ := ml["serviceArns"].([]interface{})
	if len(svcARNs) != 0 {
		t.Errorf("ListServices: expected 0 services after delete, got %d", len(svcARNs))
	}

	// DeleteCluster.
	wdc := httptest.NewRecorder()
	handler.ServeHTTP(wdc, ecsReq(t, "DeleteCluster", map[string]interface{}{
		"cluster": "del-cluster",
	}))
	if wdc.Code != http.StatusOK {
		t.Fatalf("DeleteCluster: expected 200, got %d\nbody: %s", wdc.Code, wdc.Body.String())
	}
	mdc := decodeJSON(t, wdc.Body.String())
	delCluster, ok := mdc["cluster"].(map[string]interface{})
	if !ok {
		t.Fatalf("DeleteCluster: missing cluster in response\nbody: %s", wdc.Body.String())
	}
	if delCluster["clusterName"].(string) != "del-cluster" {
		t.Errorf("DeleteCluster: expected clusterName=del-cluster")
	}

	// Verify cluster is gone.
	wlc := httptest.NewRecorder()
	handler.ServeHTTP(wlc, ecsReq(t, "DescribeClusters", map[string]interface{}{
		"clusters": []string{"del-cluster"},
	}))
	if wlc.Code != http.StatusOK {
		t.Fatalf("DescribeClusters after delete: expected 200, got %d\nbody: %s", wlc.Code, wlc.Body.String())
	}
	mlc := decodeJSON(t, wlc.Body.String())
	remainingClusters, _ := mlc["clusters"].([]interface{})
	failures, _ := mlc["failures"].([]interface{})
	if len(remainingClusters) != 0 || len(failures) == 0 {
		t.Errorf("DescribeClusters after delete: expected 0 clusters and 1 failure, got clusters=%d failures=%d",
			len(remainingClusters), len(failures))
	}
}

// ---- Test 7: TagResource / UntagResource / ListTagsForResource ----

func TestECS_TagOperations(t *testing.T) {
	handler := newECSGateway(t)

	// Create a cluster and use its ARN for tagging.
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, ecsReq(t, "CreateCluster", map[string]interface{}{
		"clusterName": "tag-cluster",
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("setup CreateCluster: %d %s", wc.Code, wc.Body.String())
	}
	mc := decodeJSON(t, wc.Body.String())
	clusterARN := mc["cluster"].(map[string]interface{})["clusterArn"].(string)

	// TagResource.
	wt := httptest.NewRecorder()
	handler.ServeHTTP(wt, ecsReq(t, "TagResource", map[string]interface{}{
		"resourceArn": clusterARN,
		"tags": []map[string]string{
			{"key": "team", "value": "platform"},
			{"key": "project", "value": "cloudmock"},
		},
	}))
	if wt.Code != http.StatusOK {
		t.Fatalf("TagResource: expected 200, got %d\nbody: %s", wt.Code, wt.Body.String())
	}

	// ListTagsForResource.
	wlt := httptest.NewRecorder()
	handler.ServeHTTP(wlt, ecsReq(t, "ListTagsForResource", map[string]interface{}{
		"resourceArn": clusterARN,
	}))
	if wlt.Code != http.StatusOK {
		t.Fatalf("ListTagsForResource: expected 200, got %d\nbody: %s", wlt.Code, wlt.Body.String())
	}
	mlt := decodeJSON(t, wlt.Body.String())
	tags, ok := mlt["tags"].([]interface{})
	if !ok {
		t.Fatalf("ListTagsForResource: missing tags\nbody: %s", wlt.Body.String())
	}
	tagMap := make(map[string]string)
	for _, tg := range tags {
		entry := tg.(map[string]interface{})
		tagMap[entry["key"].(string)] = entry["value"].(string)
	}
	if tagMap["team"] != "platform" {
		t.Errorf("ListTagsForResource: expected team=platform, got %q", tagMap["team"])
	}
	if tagMap["project"] != "cloudmock" {
		t.Errorf("ListTagsForResource: expected project=cloudmock, got %q", tagMap["project"])
	}

	// UntagResource.
	wu := httptest.NewRecorder()
	handler.ServeHTTP(wu, ecsReq(t, "UntagResource", map[string]interface{}{
		"resourceArn": clusterARN,
		"tagKeys":     []string{"team"},
	}))
	if wu.Code != http.StatusOK {
		t.Fatalf("UntagResource: expected 200, got %d\nbody: %s", wu.Code, wu.Body.String())
	}

	// Verify tag removed.
	wlt2 := httptest.NewRecorder()
	handler.ServeHTTP(wlt2, ecsReq(t, "ListTagsForResource", map[string]interface{}{
		"resourceArn": clusterARN,
	}))
	mlt2 := decodeJSON(t, wlt2.Body.String())
	tags2, _ := mlt2["tags"].([]interface{})
	for _, tg := range tags2 {
		entry := tg.(map[string]interface{})
		if entry["key"].(string) == "team" {
			t.Error("UntagResource: team tag should have been removed")
		}
	}
	// project tag should still exist.
	found := false
	for _, tg := range tags2 {
		entry := tg.(map[string]interface{})
		if entry["key"].(string) == "project" {
			found = true
		}
	}
	if !found {
		t.Error("UntagResource: project tag should still exist")
	}
}

// ---- Test 8: Unknown action ----

func TestECS_UnknownAction(t *testing.T) {
	handler := newECSGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ecsReq(t, "NonExistentAction", nil))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("unknown action: expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
}
