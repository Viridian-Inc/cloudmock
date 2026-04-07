package dataplane

import (
	"context"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/config"
)

type SLOWindowStatus struct {
	Service    string
	Action     string
	Total      int64
	Violations int64
	Errors     int64
	ErrorRate  float64
	BudgetUsed float64
	BurnRate   float64
	P50Ms      float64
	P95Ms      float64
	P99Ms      float64
	Breaching  bool
}

type SLOAlert struct {
	Service string
	Action  string
	Message string
}

type SLOStatus struct {
	Windows []SLOWindowStatus
	Healthy bool
	Alerts  []SLOAlert
}

type SLORuleChange struct {
	RuleID     string
	ChangeType string
	OldValues  *config.SLORule
	NewValues  *config.SLORule
	ChangedBy  string
	ChangedAt  time.Time
}

type SLOStore interface {
	Rules(ctx context.Context) ([]config.SLORule, error)
	SetRules(ctx context.Context, rules []config.SLORule) error
	Status(ctx context.Context) (*SLOStatus, error)
	History(ctx context.Context, limit int) ([]SLORuleChange, error)
}
