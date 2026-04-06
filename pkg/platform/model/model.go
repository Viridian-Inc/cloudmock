package model

import (
	"net"
	"time"
)

type Tenant struct {
	ID               string    `json:"id"`
	ClerkOrgID       string    `json:"clerk_org_id"`
	Name             string    `json:"name"`
	Slug             string    `json:"slug"`
	Status           string    `json:"status"`
	HasPaymentMethod bool      `json:"has_payment_method"`
	StripeCustomerID string    `json:"stripe_customer_id,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type App struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	Name         string    `json:"name"`
	Slug         string    `json:"slug"`
	Endpoint     string    `json:"endpoint"`
	InfraType    string    `json:"infra_type"`
	FlyAppName   string    `json:"fly_app_name,omitempty"`
	FlyMachineID string    `json:"fly_machine_id,omitempty"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

type APIKey struct {
	ID         string     `json:"id"`
	TenantID   string     `json:"tenant_id"`
	AppID      string     `json:"app_id"`
	KeyHash    string     `json:"-"`
	Prefix     string     `json:"prefix"`
	Name       string     `json:"name"`
	Role       string     `json:"role"`
	LastUsedAt *time.Time `json:"last_used_at"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

type UsageRecord struct {
	ID               string    `json:"id"`
	TenantID         string    `json:"tenant_id"`
	AppID            string    `json:"app_id"`
	PeriodStart      time.Time `json:"period_start"`
	PeriodEnd        time.Time `json:"period_end"`
	RequestCount     int64     `json:"request_count"`
	ReportedToStripe bool      `json:"reported_to_stripe"`
	CreatedAt        time.Time `json:"created_at"`
}

type AuditEntry struct {
	ID           string         `json:"id"`
	TenantID     string         `json:"tenant_id"`
	ActorID      string         `json:"actor_id"`
	ActorType    string         `json:"actor_type"`
	Action       string         `json:"action"`
	ResourceType string         `json:"resource_type"`
	ResourceID   string         `json:"resource_id"`
	IPAddress    net.IP         `json:"ip_address"`
	UserAgent    string         `json:"user_agent"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
}

type DataRetention struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	ResourceType  string    `json:"resource_type"`
	RetentionDays int       `json:"retention_days"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type AuthContext struct {
	TenantID  string
	ActorID   string
	ActorType string
	Role      string
	AppID     string
}
