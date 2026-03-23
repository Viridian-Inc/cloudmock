package incident

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/neureaux/cloudmock/pkg/regression"
)

// WebhookDispatcher is the interface used by the incident service to fire
// outbound webhooks. It matches *webhook.Dispatcher.
type WebhookDispatcher interface {
	Fire(ctx context.Context, event string, payload interface{}) error
}

// Service correlates regressions and SLO breaches into grouped incidents.
type Service struct {
	store       IncidentStore
	regStore    regression.RegressionStore
	groupWindow time.Duration
	webhooks    WebhookDispatcher
}

// NewService creates an incident service with the given stores and grouping window.
func NewService(store IncidentStore, regStore regression.RegressionStore, groupWindow time.Duration) *Service {
	return &Service{
		store:       store,
		regStore:    regStore,
		groupWindow: groupWindow,
	}
}

// Store returns the underlying incident store.
func (s *Service) Store() IncidentStore { return s.store }

// SetWebhookDispatcher configures an optional dispatcher to fire webhooks on
// incident creation and resolution.
func (s *Service) SetWebhookDispatcher(d WebhookDispatcher) {
	s.webhooks = d
}

// OnRegression handles a regression notification, creating or updating an incident.
func (s *Service) OnRegression(ctx context.Context, r regression.Regression) error {
	if r.Status == "resolved" {
		return s.handleResolved(ctx, r)
	}
	return s.handleActive(ctx, r)
}

// OnSLOBreach handles an SLO burn-rate alert, creating or merging into an incident.
func (s *Service) OnSLOBreach(ctx context.Context, service, action string, burnRate, budgetUsed float64) error {
	title := fmt.Sprintf("SLO burn rate alert: %s/%s (%.0f%% budget consumed)", service, action, budgetUsed*100)

	var severity string
	switch {
	case budgetUsed > 0.9:
		severity = "critical"
	case budgetUsed > 0.5:
		severity = "warning"
	default:
		severity = "info"
	}

	since := time.Now().Add(-s.groupWindow)
	existing, err := s.store.FindActiveByKey(ctx, service, "", since)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return err
	}

	if existing != nil {
		existing.LastSeen = time.Now()
		existing.AlertCount++
		if severityRank(severity) > severityRank(existing.Severity) {
			existing.Severity = severity
		}
		return s.store.Update(ctx, existing)
	}

	now := time.Now()
	inc := &Incident{
		Status:           "active",
		Severity:         severity,
		Title:            title,
		AffectedServices: []string{service},
		AlertCount:       1,
		FirstSeen:        now,
		LastSeen:         now,
	}
	if err := s.store.Save(ctx, inc); err != nil {
		return err
	}
	if s.webhooks != nil {
		s.webhooks.Fire(ctx, "incident.created", inc)
	}
	return nil
}

func (s *Service) handleResolved(ctx context.Context, r regression.Regression) error {
	existing, err := s.store.FindActiveByKey(ctx, r.Service, r.DeployID, time.Time{})
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil // no incident to resolve
		}
		return err
	}

	active, err := s.regStore.ActiveForDeploy(ctx, r.DeployID)
	if err != nil {
		return err
	}

	if len(active) == 0 {
		now := time.Now()
		existing.Status = "resolved"
		existing.ResolvedAt = &now
		if err := s.store.Update(ctx, existing); err != nil {
			return err
		}
		if s.webhooks != nil {
			s.webhooks.Fire(ctx, "incident.resolved", existing)
		}
		return nil
	}

	return nil
}

func (s *Service) handleActive(ctx context.Context, r regression.Regression) error {
	since := time.Now().Add(-s.groupWindow)
	existing, err := s.store.FindActiveByKey(ctx, r.Service, r.DeployID, since)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return err
	}

	if existing != nil {
		existing.LastSeen = r.DetectedAt
		existing.AlertCount++

		if !containsString(existing.AffectedServices, r.Service) {
			existing.AffectedServices = append(existing.AffectedServices, r.Service)
		}
		if r.TenantID != "" && !containsString(existing.AffectedTenants, r.TenantID) {
			existing.AffectedTenants = append(existing.AffectedTenants, r.TenantID)
		}

		newSev := string(r.Severity)
		if severityRank(newSev) > severityRank(existing.Severity) {
			existing.Severity = newSev
		}

		return s.store.Update(ctx, existing)
	}

	var tenants []string
	if r.TenantID != "" {
		tenants = []string{r.TenantID}
	}

	inc := &Incident{
		Status:           "active",
		Severity:         string(r.Severity),
		Title:            r.Title,
		AffectedServices: []string{r.Service},
		AffectedTenants:  tenants,
		RelatedDeployID:  r.DeployID,
		RootCause:        r.Title,
		AlertCount:       1,
		FirstSeen:        r.DetectedAt,
		LastSeen:         r.DetectedAt,
	}
	if err := s.store.Save(ctx, inc); err != nil {
		return err
	}
	if s.webhooks != nil {
		s.webhooks.Fire(ctx, "incident.created", inc)
	}
	return nil
}

// severityRank returns a numeric rank for severity comparison.
func severityRank(s string) int {
	switch s {
	case "critical":
		return 3
	case "warning":
		return 2
	case "info":
		return 1
	default:
		return 0
	}
}

// containsString reports whether ss contains s.
func containsString(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

// Compile-time interface check.
var _ AlertSink = (*Service)(nil)
