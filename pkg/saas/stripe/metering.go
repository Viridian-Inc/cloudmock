package stripe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/neureaux/cloudmock/pkg/saas/tenant"
)

const stripeAPIBase = "https://api.stripe.com/v1"

// UsageReporter reads unreported usage records from the tenant store
// and reports them to Stripe's metering API.
type UsageReporter struct {
	tenants    tenant.Store
	apiKey     string
	httpClient *http.Client
	logger     *slog.Logger
}

// NewUsageReporter creates a new Stripe usage reporter.
func NewUsageReporter(tenants tenant.Store, stripeAPIKey string, logger *slog.Logger) *UsageReporter {
	if logger == nil {
		logger = slog.Default()
	}
	return &UsageReporter{
		tenants:    tenants,
		apiKey:     stripeAPIKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		logger:     logger,
	}
}

// meterEvent is the payload sent to Stripe's v2 billing meter events endpoint.
type meterEvent struct {
	EventName string            `json:"event_name"`
	Payload   meterEventPayload `json:"payload"`
	Timestamp string            `json:"timestamp,omitempty"`
}

// meterEventPayload holds the metering dimensions.
type meterEventPayload struct {
	StripeCustomerID string `json:"stripe_customer_id"`
	Value            string `json:"value"`
}

// ReportUsage reads unreported usage records and sends them to Stripe's
// billing meter events API. Each record is sent individually and marked
// as reported on success.
func (u *UsageReporter) ReportUsage(ctx context.Context) error {
	records, err := u.tenants.GetUnreportedUsage(ctx)
	if err != nil {
		return fmt.Errorf("get unreported usage: %w", err)
	}

	if len(records) == 0 {
		u.logger.Debug("stripe metering: no unreported usage records")
		return nil
	}

	u.logger.Info("stripe metering: reporting usage", "record_count", len(records))

	var reportErrors int
	for _, rec := range records {
		if err := u.reportSingleRecord(ctx, rec); err != nil {
			u.logger.Error("stripe metering: failed to report record",
				"record_id", rec.ID,
				"tenant_id", rec.TenantID,
				"error", err,
			)
			reportErrors++
			continue
		}

		if err := u.tenants.MarkUsageReported(ctx, rec.ID); err != nil {
			u.logger.Error("stripe metering: failed to mark record as reported",
				"record_id", rec.ID,
				"error", err,
			)
			reportErrors++
		}
	}

	if reportErrors > 0 {
		return fmt.Errorf("stripe metering: %d/%d records failed", reportErrors, len(records))
	}

	u.logger.Info("stripe metering: all records reported", "count", len(records))
	return nil
}

// reportSingleRecord sends one usage record to the Stripe billing meter events API.
func (u *UsageReporter) reportSingleRecord(ctx context.Context, rec tenant.UsageRecord) error {
	// Look up the tenant to get the Stripe customer ID.
	t, err := u.tenants.Get(ctx, rec.TenantID)
	if err != nil {
		return fmt.Errorf("get tenant %s: %w", rec.TenantID, err)
	}

	if t.StripeCustomerID == "" {
		// Tenant has no Stripe customer — skip (probably on the free tier).
		u.logger.Debug("stripe metering: skipping tenant without stripe customer",
			"tenant_id", t.ID,
		)
		return nil
	}

	event := meterEvent{
		EventName: "api_requests",
		Payload: meterEventPayload{
			StripeCustomerID: t.StripeCustomerID,
			Value:            fmt.Sprintf("%d", rec.RequestCount),
		},
		Timestamp: rec.PeriodEnd.UTC().Format(time.RFC3339),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal meter event: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		stripeAPIBase+"/billing/meter_events", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+u.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send meter event: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("stripe API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// RunPeriodicReporting starts a goroutine that reports usage every interval.
// It blocks until the context is cancelled. Intended to be run as:
//
//	go reporter.RunPeriodicReporting(ctx, 1*time.Hour)
func (u *UsageReporter) RunPeriodicReporting(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	u.logger.Info("stripe metering: started periodic reporting", "interval", interval)

	for {
		select {
		case <-ctx.Done():
			u.logger.Info("stripe metering: stopped periodic reporting")
			return
		case <-ticker.C:
			if err := u.ReportUsage(ctx); err != nil {
				u.logger.Error("stripe metering: periodic report failed", "error", err)
			}
		}
	}
}
