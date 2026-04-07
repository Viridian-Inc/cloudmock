package ecs

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// CapacityProvider holds ECS capacity provider metadata.
type CapacityProvider struct {
	Name                     string
	ARN                      string
	Status                   string
	AutoScalingGroupProvider map[string]any
}

// capacityProviderStore is the package-level in-memory store for capacity providers.
var (
	cpMu                sync.RWMutex
	capacityProviders   = make(map[string]*CapacityProvider) // keyed by name
	cpAccountID         string
	cpRegion            string
)

// initCPStore sets the account and region for ARN generation.
// Called by the Store constructor.
func initCPStore(accountID, region string) {
	cpMu.Lock()
	defer cpMu.Unlock()
	cpAccountID = accountID
	cpRegion = region
}

func capacityProviderARN(name string) string {
	return fmt.Sprintf("arn:aws:ecs:%s:%s:capacity-provider/%s", cpRegion, cpAccountID, name)
}

// ---- JSON request/response types ----

type createCapacityProviderRequest struct {
	Name                     string         `json:"name"`
	AutoScalingGroupProvider map[string]any `json:"autoScalingGroupProvider"`
}

type capacityProviderJSON struct {
	CapacityProviderArn      string         `json:"capacityProviderArn"`
	Name                     string         `json:"name"`
	Status                   string         `json:"status"`
	AutoScalingGroupProvider map[string]any `json:"autoScalingGroupProvider,omitempty"`
}

func cpToJSON(cp *CapacityProvider) capacityProviderJSON {
	return capacityProviderJSON{
		CapacityProviderArn:      cp.ARN,
		Name:                     cp.Name,
		Status:                   cp.Status,
		AutoScalingGroupProvider: cp.AutoScalingGroupProvider,
	}
}

// ---- Capacity Provider handlers ----

func handleCreateCapacityProvider(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createCapacityProviderRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Name == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"name is required.", http.StatusBadRequest))
	}

	cpMu.Lock()
	defer cpMu.Unlock()

	// Initialise account/region from the store if not yet set.
	if cpAccountID == "" {
		cpAccountID = store.accountID
		cpRegion = store.region
	}

	if _, exists := capacityProviders[req.Name]; exists {
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidParameterException",
				fmt.Sprintf("A capacity provider with the name %s already exists.", req.Name),
				http.StatusBadRequest)
	}

	cp := &CapacityProvider{
		Name:                     req.Name,
		ARN:                      capacityProviderARN(req.Name),
		Status:                   "ACTIVE",
		AutoScalingGroupProvider: req.AutoScalingGroupProvider,
	}
	capacityProviders[req.Name] = cp

	return jsonOK(map[string]any{
		"capacityProvider": cpToJSON(cp),
	})
}

type describeCapacityProvidersRequest struct {
	CapacityProviders []string `json:"capacityProviders"`
}

func handleDescribeCapacityProviders(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeCapacityProvidersRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	cpMu.RLock()
	defer cpMu.RUnlock()

	// Initialise account/region from the store if not yet set.
	if cpAccountID == "" {
		cpAccountID = store.accountID
		cpRegion = store.region
	}

	var out []capacityProviderJSON

	if len(req.CapacityProviders) == 0 {
		// Return all.
		out = make([]capacityProviderJSON, 0, len(capacityProviders))
		for _, cp := range capacityProviders {
			out = append(out, cpToJSON(cp))
		}
	} else {
		out = make([]capacityProviderJSON, 0, len(req.CapacityProviders))
		for _, name := range req.CapacityProviders {
			if cp, ok := capacityProviders[name]; ok {
				out = append(out, cpToJSON(cp))
			}
		}
	}

	return jsonOK(map[string]any{
		"capacityProviders": out,
	})
}

type deleteCapacityProviderRequest struct {
	CapacityProvider string `json:"capacityProvider"`
}

func handleDeleteCapacityProvider(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteCapacityProviderRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.CapacityProvider == "" {
		return jsonErr(service.NewAWSError("InvalidParameterException",
			"capacityProvider is required.", http.StatusBadRequest))
	}

	cpMu.Lock()
	defer cpMu.Unlock()

	cp, ok := capacityProviders[req.CapacityProvider]
	if !ok {
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidParameterException",
				fmt.Sprintf("The specified capacity provider does not exist: %s", req.CapacityProvider),
				http.StatusBadRequest)
	}

	delete(capacityProviders, req.CapacityProvider)

	return jsonOK(map[string]any{
		"capacityProvider": cpToJSON(cp),
	})
}

// ---- ExecuteCommand handler ----

type executeCommandRequest struct {
	Cluster   string `json:"cluster"`
	Container string `json:"container"`
	Command   string `json:"command"`
	Task      string `json:"task"`
}

func handleExecuteCommand(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req executeCommandRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(map[string]any{
		"session": map[string]any{
			"sessionId":  "ecs-execute-command-" + newUUID(),
			"streamUrl":  "wss://ssmmessages.us-east-1.amazonaws.com/v1/data-channel/session-" + newUUID(),
			"tokenValue": "mock-token-" + newUUID(),
		},
		"clusterArn":   req.Cluster,
		"containerName": req.Container,
		"command":       req.Command,
		"taskArn":       req.Task,
		"interactive":   true,
	})
}
