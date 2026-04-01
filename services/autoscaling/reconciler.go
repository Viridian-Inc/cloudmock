package autoscaling

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/neureaux/cloudmock/pkg/eventbus"
	"github.com/neureaux/cloudmock/pkg/service"
)

// reconcileInstancesViaEC2 adjusts ASG instances by creating/terminating real
// EC2 instances through the ServiceLocator. Falls back to stub instance IDs
// if the locator or EC2 service is unavailable.
// Must be called with s.mu held.
func (s *Store) reconcileInstancesViaEC2(asg *AutoScalingGroup, locator ServiceLocator, bus *eventbus.Bus) {
	current := len(asg.Instances)
	desired := asg.DesiredCapacity

	if desired > current {
		// Scale up: launch new instances
		for i := current; i < desired; i++ {
			az := ""
			if len(asg.AvailabilityZones) > 0 {
				az = asg.AvailabilityZones[i%len(asg.AvailabilityZones)]
			}
			s.instanceSeq++

			instanceID := s.launchEC2Instance(asg, az, locator)

			inst := &AutoScalingInstance{
				InstanceID:           instanceID,
				AutoScalingGroupName: asg.Name,
				AvailabilityZone:     az,
				LifecycleState:       "InService",
				HealthStatus:         "Healthy",
				LaunchConfigName:     asg.LaunchConfigName,
			}
			asg.Instances = append(asg.Instances, inst)

			if bus != nil {
				bus.Publish(&eventbus.Event{
					Source: "autoscaling",
					Type:   "autoscaling:EC2_INSTANCE_LAUNCH",
					Detail: map[string]any{
						"AutoScalingGroupName": asg.Name,
						"EC2InstanceId":        instanceID,
						"AvailabilityZone":     az,
					},
				})
			}
		}
	} else if desired < current {
		// Scale down: terminate excess instances
		toRemove := asg.Instances[desired:]
		asg.Instances = asg.Instances[:desired]

		for _, inst := range toRemove {
			s.terminateEC2Instance(inst.InstanceID, locator)

			if bus != nil {
				bus.Publish(&eventbus.Event{
					Source: "autoscaling",
					Type:   "autoscaling:EC2_INSTANCE_TERMINATE",
					Detail: map[string]any{
						"AutoScalingGroupName": asg.Name,
						"EC2InstanceId":        inst.InstanceID,
					},
				})
			}
		}
	}
}

// launchEC2Instance attempts to create an EC2 instance via the locator.
// Returns the instance ID. Falls back to a random stub ID if EC2 is unavailable.
func (s *Store) launchEC2Instance(asg *AutoScalingGroup, az string, locator ServiceLocator) string {
	if locator == nil {
		return fmt.Sprintf("i-%s", randomHex(8))
	}

	ec2Svc, err := locator.Lookup("ec2")
	if err != nil {
		return fmt.Sprintf("i-%s", randomHex(8))
	}

	// Build RunInstances request body
	body, _ := json.Marshal(map[string]any{
		"ImageId":      "ami-stub",
		"InstanceType": "t3.micro",
		"MinCount":     1,
		"MaxCount":     1,
		"Placement": map[string]any{
			"AvailabilityZone": az,
		},
		"TagSpecifications": []map[string]any{
			{
				"ResourceType": "instance",
				"Tags": []map[string]string{
					{"Key": "aws:autoscaling:groupName", "Value": asg.Name},
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
		return fmt.Sprintf("i-%s", randomHex(8))
	}

	// Try to extract instance ID from response
	instanceID := extractInstanceID(resp.Body)
	if instanceID != "" {
		return instanceID
	}
	return fmt.Sprintf("i-%s", randomHex(8))
}

// terminateEC2Instance attempts to terminate an EC2 instance via the locator.
// Silently ignores errors for graceful degradation.
func (s *Store) terminateEC2Instance(instanceID string, locator ServiceLocator) {
	if locator == nil {
		return
	}

	ec2Svc, err := locator.Lookup("ec2")
	if err != nil {
		return
	}

	body, _ := json.Marshal(map[string]any{
		"InstanceIds": []string{instanceID},
	})

	_, _ = ec2Svc.HandleRequest(&service.RequestContext{
		Action:     "TerminateInstances",
		Body:       body,
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
	})
}

// extractInstanceID tries to pull an instance ID from an EC2 RunInstances response.
func extractInstanceID(body any) string {
	if body == nil {
		return ""
	}

	// The response could be a map or a struct; try JSON round-trip
	data, err := json.Marshal(body)
	if err != nil {
		return ""
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return ""
	}

	// Try common EC2 response shapes
	if instances, ok := result["Instances"].([]any); ok && len(instances) > 0 {
		if inst, ok := instances[0].(map[string]any); ok {
			if id, ok := inst["InstanceId"].(string); ok {
				return id
			}
		}
	}

	// Try nested reservationSet shape
	if reservationSet, ok := result["instancesSet"].([]any); ok && len(reservationSet) > 0 {
		if inst, ok := reservationSet[0].(map[string]any); ok {
			if id, ok := inst["instanceId"].(string); ok {
				return id
			}
		}
	}

	return ""
}
