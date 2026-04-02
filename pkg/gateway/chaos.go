package gateway

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

// ChaosRule defines a fault injection rule that the ChaosEngine applies to matching requests.
type ChaosRule struct {
	ID         string `json:"id"`
	Service    string `json:"service"`    // target service ("dynamodb", "s3", "*" for all)
	Action     string `json:"action"`     // target action ("*" for all)
	Enabled    bool   `json:"enabled"`
	Type       string `json:"type"`       // "error", "latency", "timeout", "blackhole"
	ErrorCode  int    `json:"errorCode"`  // HTTP status code for "error" type
	ErrorMsg   string `json:"errorMsg"`   // error message
	LatencyMs  int    `json:"latencyMs"`  // added latency for "latency" type
	Percentage int    `json:"percentage"` // 0-100, probability of applying
}

// ChaosEngine manages a set of chaos/fault injection rules.
type ChaosEngine struct {
	mu          sync.RWMutex
	rules       []ChaosRule
	seq         int
	PersistFunc func(rules []ChaosRule) // called after any mutation, if non-nil
}

// NewChaosEngine creates a new ChaosEngine with no rules.
func NewChaosEngine() *ChaosEngine {
	return &ChaosEngine{}
}

// NewChaosEngineWithRules creates a ChaosEngine pre-loaded with rules.
// The seq counter is set to len(rules) so new IDs don't collide.
func NewChaosEngineWithRules(rules []ChaosRule) *ChaosEngine {
	return &ChaosEngine{
		rules: rules,
		seq:   len(rules),
	}
}

// Rules returns all configured chaos rules.
func (ce *ChaosEngine) Rules() []ChaosRule {
	ce.mu.RLock()
	defer ce.mu.RUnlock()
	out := make([]ChaosRule, len(ce.rules))
	copy(out, ce.rules)
	return out
}

// AddRule adds a new chaos rule and returns it with its assigned ID.
func (ce *ChaosEngine) AddRule(rule ChaosRule) ChaosRule {
	ce.mu.Lock()
	ce.seq++
	rule.ID = fmt.Sprintf("chaos-%d", ce.seq)
	ce.rules = append(ce.rules, rule)
	snapshot := make([]ChaosRule, len(ce.rules))
	copy(snapshot, ce.rules)
	persist := ce.PersistFunc
	ce.mu.Unlock()
	if persist != nil {
		persist(snapshot)
	}
	return rule
}

// UpdateRule updates a rule by ID, returning the updated rule and whether it was found.
func (ce *ChaosEngine) UpdateRule(id string, update ChaosRule) (ChaosRule, bool) {
	ce.mu.Lock()
	for i, r := range ce.rules {
		if r.ID == id {
			update.ID = id
			ce.rules[i] = update
			result := ce.rules[i]
			snapshot := make([]ChaosRule, len(ce.rules))
			copy(snapshot, ce.rules)
			persist := ce.PersistFunc
			ce.mu.Unlock()
			if persist != nil {
				persist(snapshot)
			}
			return result, true
		}
	}
	ce.mu.Unlock()
	return ChaosRule{}, false
}

// DeleteRule removes a rule by ID, returning whether it was found.
func (ce *ChaosEngine) DeleteRule(id string) bool {
	ce.mu.Lock()
	for i, r := range ce.rules {
		if r.ID == id {
			ce.rules = append(ce.rules[:i], ce.rules[i+1:]...)
			snapshot := make([]ChaosRule, len(ce.rules))
			copy(snapshot, ce.rules)
			persist := ce.PersistFunc
			ce.mu.Unlock()
			if persist != nil {
				persist(snapshot)
			}
			return true
		}
	}
	ce.mu.Unlock()
	return false
}

// DisableAll disables all rules.
func (ce *ChaosEngine) DisableAll() {
	ce.mu.Lock()
	for i := range ce.rules {
		ce.rules[i].Enabled = false
	}
	snapshot := make([]ChaosRule, len(ce.rules))
	copy(snapshot, ce.rules)
	persist := ce.PersistFunc
	ce.mu.Unlock()
	if persist != nil {
		persist(snapshot)
	}
}

// HasActiveRules returns true if any rule is enabled.
func (ce *ChaosEngine) HasActiveRules() bool {
	ce.mu.RLock()
	defer ce.mu.RUnlock()
	for _, r := range ce.rules {
		if r.Enabled {
			return true
		}
	}
	return false
}

// Match finds the first enabled rule that matches the given service and action,
// applies the percentage check, and returns the rule (or nil if no match/no fire).
func (ce *ChaosEngine) Match(svcName, action string) *ChaosRule {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	for _, r := range ce.rules {
		if !r.Enabled {
			continue
		}
		if r.Service != "*" && r.Service != svcName {
			continue
		}
		if r.Action != "*" && r.Action != action {
			continue
		}
		// Percentage check
		if r.Percentage <= 0 {
			continue
		}
		if r.Percentage < 100 && rand.Intn(100) >= r.Percentage {
			continue
		}
		matched := r
		return &matched
	}
	return nil
}

// ChaosMiddleware wraps a gateway handler and applies chaos/fault injection rules
// before forwarding requests.
func ChaosMiddleware(next http.Handler, engine *ChaosEngine) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Fast path: skip chaos matching when no rules are enabled.
		if !engine.HasActiveRules() {
			next.ServeHTTP(w, r)
			return
		}

		svcName := detectServiceFromRequest(r)
		action := detectActionFromRequest(r)

		rule := engine.Match(svcName, action)
		if rule != nil {
			switch rule.Type {
			case "error":
				code := rule.ErrorCode
				if code == 0 {
					code = http.StatusInternalServerError
				}
				msg := rule.ErrorMsg
				if msg == "" {
					msg = "ChaosEngineInjectedFault"
				}
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Cloudmock-Chaos", rule.ID)
				w.WriteHeader(code)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"__type":  "ChaosEngineError",
					"message": msg,
					"chaosId": rule.ID,
				})
				return

			case "latency":
				if rule.LatencyMs > 0 {
					time.Sleep(time.Duration(rule.LatencyMs) * time.Millisecond)
				}
				w.Header().Set("X-Cloudmock-Chaos", rule.ID)
				// Fall through to normal handler.

			case "timeout":
				w.Header().Set("X-Cloudmock-Chaos", rule.ID)
				time.Sleep(30 * time.Second)
				w.WriteHeader(http.StatusGatewayTimeout)
				_, _ = w.Write([]byte(`{"__type":"ChaosTimeout","message":"Request timed out (chaos injection)"}`))
				return

			case "blackhole":
				// Hijack the connection and close it without sending a response.
				hj, ok := w.(http.Hijacker)
				if ok {
					conn, _, err := hj.Hijack()
					if err == nil {
						conn.Close()
						return
					}
				}
				// Fallback if hijack is not supported: just close with no response body.
				w.Header().Set("Connection", "close")
				w.WriteHeader(http.StatusBadGateway)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}
