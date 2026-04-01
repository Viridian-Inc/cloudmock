package organizations

import (
	"encoding/json"
)

// SCPStatement represents a single statement in a service control policy.
type SCPStatement struct {
	Effect   string   `json:"Effect"`
	Action   []string `json:"Action"`
	Resource []string `json:"Resource"`
}

// SCPDocument represents a parsed SCP policy document.
type SCPDocument struct {
	Version   string         `json:"Version"`
	Statement []SCPStatement `json:"Statement"`
}

// EvaluateSCP checks whether an action on a resource is allowed by the SCPs
// attached to the given target and its parents (up to root).
// Returns true if the action is allowed (not denied by any SCP), false if denied.
// If no SCPs are attached, the action is allowed by default.
func (s *Store) EvaluateSCP(targetID, action, resource string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Collect all SCP policy IDs that apply to this target and its ancestors
	var applicablePolicies []*Policy
	currentID := targetID
	for {
		// Find SCPs attached to this target
		for policyID, targets := range s.policyAttachments {
			for _, t := range targets {
				if t.TargetId == currentID {
					if policy, ok := s.policies[policyID]; ok {
						if policy.PolicySummary.Type == "SERVICE_CONTROL_POLICY" {
							applicablePolicies = append(applicablePolicies, policy)
						}
					}
				}
			}
		}

		// Walk up the hierarchy
		if ou, ok := s.ous[currentID]; ok {
			currentID = ou.ParentId
			continue
		}
		if acct, ok := s.accounts[currentID]; ok {
			currentID = acct.ParentId
			continue
		}
		// We're at a root or unknown, stop
		break
	}

	if len(applicablePolicies) == 0 {
		return true // No SCPs means allow
	}

	// Evaluate SCPs: explicit deny wins, then require explicit allow
	hasAllow := false
	for _, policy := range applicablePolicies {
		var doc SCPDocument
		if err := json.Unmarshal([]byte(policy.Content), &doc); err != nil {
			continue
		}

		for _, stmt := range doc.Statement {
			if matchesSCPAction(stmt.Action, action) && matchesSCPResource(stmt.Resource, resource) {
				if stmt.Effect == "Deny" {
					return false
				}
				if stmt.Effect == "Allow" {
					hasAllow = true
				}
			}
		}
	}

	return hasAllow
}

func matchesSCPAction(actions []string, action string) bool {
	for _, a := range actions {
		if a == "*" || a == action {
			return true
		}
	}
	return false
}

func matchesSCPResource(resources []string, resource string) bool {
	if len(resources) == 0 {
		return true
	}
	for _, r := range resources {
		if r == "*" || r == resource {
			return true
		}
	}
	return false
}
