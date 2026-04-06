package ecs_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// ---- Capacity Providers ----

func TestECS_CapacityProviderLifecycle(t *testing.T) {
	handler := newECSGateway(t)

	// CreateCapacityProvider
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ecsReq(t, "CreateCapacityProvider", map[string]any{
		"name": "my-cp",
		"autoScalingGroupProvider": map[string]any{
			"autoScalingGroupArn": "arn:aws:autoscaling:us-east-1:000000000000:autoScalingGroup:asg-1",
		},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateCapacityProvider: %d %s", w.Code, w.Body.String())
	}

	// DescribeCapacityProviders
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, ecsReq(t, "DescribeCapacityProviders", map[string]any{
		"capacityProviders": []string{"my-cp"},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeCapacityProviders: %d %s", w.Code, w.Body.String())
	}

	// DeleteCapacityProvider
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, ecsReq(t, "DeleteCapacityProvider", map[string]any{
		"capacityProvider": "my-cp",
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteCapacityProvider: %d %s", w.Code, w.Body.String())
	}
}

// ---- ExecuteCommand ----

func TestECS_ExecuteCommand(t *testing.T) {
	handler := newECSGateway(t)

	// Create cluster first
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, ecsReq(t, "CreateCluster", map[string]string{
		"clusterName": "exec-cluster",
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateCluster: %d %s", w.Code, w.Body.String())
	}

	// ExecuteCommand — stub returns session info
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, ecsReq(t, "ExecuteCommand", map[string]any{
		"cluster":     "exec-cluster",
		"task":        "task-123",
		"command":     "/bin/sh",
		"interactive": true,
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("ExecuteCommand: %d %s", w.Code, w.Body.String())
	}
}
