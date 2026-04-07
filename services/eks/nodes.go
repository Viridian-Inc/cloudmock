package eks

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// launchNodeInstances creates EC2 instances for a node group via the ServiceLocator.
// Returns the instance IDs. Falls back to stub IDs if EC2 is unavailable.
func launchNodeInstances(count int, clusterName, ngName string, locator ServiceLocator) []string {
	ids := make([]string, 0, count)

	if locator == nil {
		for i := 0; i < count; i++ {
			ids = append(ids, fmt.Sprintf("i-%s", randomHex(8)))
		}
		return ids
	}

	ec2Svc, err := locator.Lookup("ec2")
	if err != nil {
		for i := 0; i < count; i++ {
			ids = append(ids, fmt.Sprintf("i-%s", randomHex(8)))
		}
		return ids
	}

	for i := 0; i < count; i++ {
		body, _ := json.Marshal(map[string]any{
			"ImageId":      "ami-eks-node",
			"InstanceType": "t3.medium",
			"MinCount":     1,
			"MaxCount":     1,
			"TagSpecifications": []map[string]any{
				{
					"ResourceType": "instance",
					"Tags": []map[string]string{
						{"Key": "eks:cluster-name", "Value": clusterName},
						{"Key": "eks:nodegroup-name", "Value": ngName},
					},
				},
			},
		})

		resp, err := ec2Svc.HandleRequest(&service.RequestContext{
			Action:     "RunInstances",
			Body:       body,
			RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
		})
		if err != nil || resp == nil {
			ids = append(ids, fmt.Sprintf("i-%s", randomHex(8)))
			continue
		}

		instanceID := extractEC2InstanceID(resp.Body)
		if instanceID != "" {
			ids = append(ids, instanceID)
		} else {
			ids = append(ids, fmt.Sprintf("i-%s", randomHex(8)))
		}
	}

	return ids
}

// terminateNodeInstances terminates EC2 instances via the ServiceLocator.
func terminateNodeInstances(instanceIDs []string, locator ServiceLocator) {
	if locator == nil || len(instanceIDs) == 0 {
		return
	}

	ec2Svc, err := locator.Lookup("ec2")
	if err != nil {
		return
	}

	body, _ := json.Marshal(map[string]any{
		"InstanceIds": instanceIDs,
	})

	_, _ = ec2Svc.HandleRequest(&service.RequestContext{
		Action:     "TerminateInstances",
		Body:       body,
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
	})
}

// extractEC2InstanceID pulls an instance ID from an EC2 RunInstances response.
func extractEC2InstanceID(body any) string {
	if body == nil {
		return ""
	}

	data, err := json.Marshal(body)
	if err != nil {
		return ""
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return ""
	}

	if instances, ok := result["Instances"].([]any); ok && len(instances) > 0 {
		if inst, ok := instances[0].(map[string]any); ok {
			if id, ok := inst["InstanceId"].(string); ok {
				return id
			}
		}
	}
	if instances, ok := result["instancesSet"].([]any); ok && len(instances) > 0 {
		if inst, ok := instances[0].(map[string]any); ok {
			if id, ok := inst["instanceId"].(string); ok {
				return id
			}
		}
	}

	return ""
}
