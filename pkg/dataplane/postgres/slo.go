package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/dataplane"
)

// SLOStore implements dataplane.SLOStore against PostgreSQL.
type SLOStore struct {
	pool *pgxpool.Pool
}

// NewSLOStore creates an SLOStore backed by the given pool.
func NewSLOStore(pool *pgxpool.Pool) *SLOStore {
	return &SLOStore{pool: pool}
}

// Rules returns all active SLO rules.
func (s *SLOStore) Rules(ctx context.Context) ([]config.SLORule, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT service, action, p50_ms, p95_ms, p99_ms, error_rate
		 FROM slo_rules WHERE active = true`)
	if err != nil {
		return nil, fmt.Errorf("slo rules query: %w", err)
	}
	defer rows.Close()

	var rules []config.SLORule
	for rows.Next() {
		var r config.SLORule
		if err := rows.Scan(&r.Service, &r.Action, &r.P50Ms, &r.P95Ms, &r.P99Ms, &r.ErrorRate); err != nil {
			return nil, fmt.Errorf("slo rules scan: %w", err)
		}
		rules = append(rules, r)
	}
	return rules, rows.Err()
}

// SetRules deactivates all current rules and inserts new ones in a transaction.
// History entries are recorded for each new rule.
func (s *SLOStore) SetRules(ctx context.Context, rules []config.SLORule) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("slo set rules begin: %w", err)
	}
	defer tx.Rollback(ctx)

	// Deactivate current rules.
	if _, err := tx.Exec(ctx, `UPDATE slo_rules SET active = false WHERE active = true`); err != nil {
		return fmt.Errorf("slo deactivate: %w", err)
	}

	// Insert new rules and history entries.
	for _, r := range rules {
		ruleID := uuid.New()
		if _, err := tx.Exec(ctx,
			`INSERT INTO slo_rules (id, service, action, p50_ms, p95_ms, p99_ms, error_rate, active, created_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, true, now())`,
			ruleID, r.Service, r.Action, r.P50Ms, r.P95Ms, r.P99Ms, r.ErrorRate,
		); err != nil {
			return fmt.Errorf("slo insert rule: %w", err)
		}

		newVals, _ := json.Marshal(r)
		if _, err := tx.Exec(ctx,
			`INSERT INTO slo_rule_history (id, rule_id, change_type, new_values, changed_at)
			 VALUES ($1, $2, 'created', $3, now())`,
			uuid.New(), ruleID, newVals,
		); err != nil {
			return fmt.Errorf("slo insert history: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// Status returns a basic SLOStatus built from active rules. Real-time metrics
// come from the SLO engine at a higher level; here we return zero-value windows.
func (s *SLOStore) Status(ctx context.Context) (*dataplane.SLOStatus, error) {
	rules, err := s.Rules(ctx)
	if err != nil {
		return nil, err
	}

	windows := make([]dataplane.SLOWindowStatus, len(rules))
	for i, r := range rules {
		windows[i] = dataplane.SLOWindowStatus{
			Service: r.Service,
			Action:  r.Action,
		}
	}

	return &dataplane.SLOStatus{
		Windows: windows,
		Healthy: true,
	}, nil
}

// History returns the most recent SLO rule changes.
func (s *SLOStore) History(ctx context.Context, limit int) ([]dataplane.SLORuleChange, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT rule_id, change_type, old_values, new_values, changed_by, changed_at
		 FROM slo_rule_history ORDER BY changed_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("slo history query: %w", err)
	}
	defer rows.Close()

	var changes []dataplane.SLORuleChange
	for rows.Next() {
		var (
			c         dataplane.SLORuleChange
			oldJSON   []byte
			newJSON   []byte
			changedBy *string
			changedAt time.Time
		)
		if err := rows.Scan(&c.RuleID, &c.ChangeType, &oldJSON, &newJSON, &changedBy, &changedAt); err != nil {
			return nil, fmt.Errorf("slo history scan: %w", err)
		}
		c.ChangedAt = changedAt
		if changedBy != nil {
			c.ChangedBy = *changedBy
		}
		if oldJSON != nil {
			var old config.SLORule
			if err := json.Unmarshal(oldJSON, &old); err == nil {
				c.OldValues = &old
			}
		}
		if newJSON != nil {
			var nv config.SLORule
			if err := json.Unmarshal(newJSON, &nv); err == nil {
				c.NewValues = &nv
			}
		}
		changes = append(changes, c)
	}
	return changes, rows.Err()
}

// Compile-time interface check.
var _ dataplane.SLOStore = (*SLOStore)(nil)

