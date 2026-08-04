package managedidentity

// UserAssignedIdentity is the ARM resource shape for Microsoft.ManagedIdentity/userAssignedIdentities.
type UserAssignedIdentity struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Location    string            `json:"location"`
	Tags        map[string]string `json:"tags,omitempty"`
	ClientID    string            `json:"clientId"`
	PrincipalID string            `json:"principalId"`
	TenantID    string            `json:"tenantId"`
}
