package incident

import (
	"context"
	"time"

	"github.com/neureaux/cloudmock/pkg/regression"
)

// AlertSink receives notifications when regressions or SLO breaches are
// detected, allowing the incident service to correlate them into incidents.
type AlertSink interface {
	OnRegression(ctx context.Context, r regression.Regression) error
	OnSLOBreach(ctx context.Context, service, action string, burnRate, budgetUsed float64) error
}

// Incident represents a grouped set of alerts affecting one or more services.
type Incident struct {
	ID               string     `json:"id"`
	Status           string     `json:"status"`
	Severity         string     `json:"severity"`
	Title            string     `json:"title"`
	AffectedServices []string   `json:"affected_services"`
	AffectedTenants  []string   `json:"affected_tenants"`
	AlertCount       int        `json:"alert_count"`
	RootCause        string     `json:"root_cause,omitempty"`
	RelatedDeployID  string     `json:"related_deploy_id,omitempty"`
	FirstSeen        time.Time  `json:"first_seen"`
	LastSeen         time.Time  `json:"last_seen"`
	ResolvedAt       *time.Time `json:"resolved_at,omitempty"`
	Owner            string     `json:"owner,omitempty"`
}

// IncidentFilter controls which incidents are returned by List.
type IncidentFilter struct {
	Status   string
	Severity string
	Service  string
	Limit    int
}
