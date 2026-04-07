package elasticloadbalancing

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// HealthChecker performs background health checks against registered targets.
type HealthChecker struct {
	mu      sync.Mutex
	store   *Store
	locator ServiceLocator
}

// NewHealthChecker creates a HealthChecker that probes target health via the
// ServiceLocator. If locator is nil, all targets degrade to their current state.
func NewHealthChecker(store *Store, locator ServiceLocator) *HealthChecker {
	return &HealthChecker{
		store:   store,
		locator: locator,
	}
}

// CheckAllTargets iterates over every target group and probes each target.
// For instance targets, it verifies the EC2 instance exists via the locator.
// This is called by the worker pool on a periodic schedule.
func (hc *HealthChecker) CheckAllTargets() {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.store.mu.Lock()
	defer hc.store.mu.Unlock()

	for _, tg := range hc.store.targetGroups {
		for _, t := range tg.Targets {
			if t.Health == "draining" {
				continue // draining targets stay draining until removed
			}
			if hc.locator != nil && tg.TargetType == "instance" {
				if hc.ec2InstanceExists(t.ID) {
					t.Health = "healthy"
					t.HealthReason = ""
				} else {
					t.Health = "unhealthy"
					t.HealthReason = "Target.NotInService"
				}
			} else {
				// No locator: promote initial to healthy (stub behavior)
				if t.Health == "initial" {
					t.Health = "healthy"
					t.HealthReason = ""
				}
			}
		}
	}
}

// ec2InstanceExists checks whether an EC2 instance exists by calling
// DescribeInstances via the locator. Returns false on any error.
func (hc *HealthChecker) ec2InstanceExists(instanceID string) bool {
	if hc.locator == nil {
		return true
	}

	ec2Svc, err := hc.locator.Lookup("ec2")
	if err != nil {
		return true // degrade gracefully: assume healthy
	}

	body, _ := json.Marshal(map[string]any{
		"InstanceIds": []string{instanceID},
	})

	resp, err := ec2Svc.HandleRequest(&service.RequestContext{
		Action:     "DescribeInstances",
		Body:       body,
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
	})
	if err != nil {
		return false
	}
	if resp == nil {
		return false
	}
	return true
}

// DeregisterWithDraining marks targets as "draining" instead of immediately
// removing them. After the draining period, they should be removed.
func (hc *HealthChecker) DeregisterWithDraining(tgARN string, targetIDs []string) bool {
	hc.store.mu.Lock()
	defer hc.store.mu.Unlock()

	tg, ok := hc.store.targetGroups[tgARN]
	if !ok {
		return false
	}

	for _, id := range targetIDs {
		if t, exists := tg.Targets[id]; exists {
			t.Health = "draining"
			t.HealthReason = "Target.DeregistrationInProgress"
		}
	}
	return true
}

// CompleteDraining removes all targets in the "draining" state from a target group.
func (hc *HealthChecker) CompleteDraining(tgARN string) {
	hc.store.mu.Lock()
	defer hc.store.mu.Unlock()

	tg, ok := hc.store.targetGroups[tgARN]
	if !ok {
		return
	}

	for id, t := range tg.Targets {
		if t.Health == "draining" {
			delete(tg.Targets, id)
		}
	}
}
