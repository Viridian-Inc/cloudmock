package tenant

import "time"

// Tenant represents a hosted SaaS tenant (organization).
type Tenant struct {
	ID                   string
	ClerkOrgID           string
	Name                 string
	Slug                 string
	StripeCustomerID     string
	StripeSubscriptionID string
	Tier                 string // "free", "pro", "team"
	Status               string // "active", "suspended", "canceled"
	FlyMachineID         string
	FlyAppName           string
	RequestCount         int64
	RequestLimit         int64
	DataRetentionDays    int
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// UsageRecord represents a billing usage record for a tenant.
type UsageRecord struct {
	ID               string
	TenantID         string
	PeriodStart      time.Time
	PeriodEnd        time.Time
	RequestCount     int64
	TotalCost        float64
	ReportedToStripe bool
	CreatedAt        time.Time
}
