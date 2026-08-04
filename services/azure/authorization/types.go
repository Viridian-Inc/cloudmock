package authorization

const roleAssignmentsAPIVersion = "2022-04-01"
const managementLocksAPIVersion = "2020-05-01"

type RoleAssignment struct {
	ID         string                   `json:"id"`
	Name       string                   `json:"name"`
	Type       string                   `json:"type"`
	Properties RoleAssignmentProperties `json:"properties"`
}

type RoleAssignmentProperties struct {
	Scope            string `json:"scope"`
	PrincipalID      string `json:"principalId"`
	PrincipalType    string `json:"principalType,omitempty"`
	RoleDefinitionID string `json:"roleDefinitionId"`
	Description      string `json:"description,omitempty"`
	Condition        string `json:"condition,omitempty"`
	ConditionVersion string `json:"conditionVersion,omitempty"`
}

type ManagementLock struct {
	ID         string                   `json:"id"`
	Name       string                   `json:"name"`
	Type       string                   `json:"type"`
	Properties ManagementLockProperties `json:"properties"`
}

type ManagementLockProperties struct {
	Level  string           `json:"level"`
	Notes  string           `json:"notes,omitempty"`
	Owners []map[string]any `json:"owners,omitempty"`
}
