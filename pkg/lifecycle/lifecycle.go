// Package lifecycle provides a configurable state machine for AWS resource
// lifecycle transitions. Resources progress through a sequence of states
// (e.g., CREATING → AVAILABLE → DELETING → DELETED) either instantly or
// with configurable delays to simulate real AWS async behavior.
package lifecycle

import (
	"sync"
	"time"
)

// State represents a resource lifecycle state (e.g., "CREATING", "AVAILABLE").
type State string

// Transition defines a state change with an optional delay.
type Transition struct {
	From  State
	To    State
	Delay time.Duration // 0 means instant
}

// Config controls whether lifecycle delays are simulated or instant.
type Config struct {
	mu          sync.RWMutex
	enabled     bool // when false, all transitions are instant
	speedFactor float64
}

// DefaultConfig returns a Config with delays disabled (instant transitions).
func DefaultConfig() *Config {
	return &Config{
		enabled:     false,
		speedFactor: 1.0,
	}
}

// SetEnabled enables or disables simulated delays globally.
func (c *Config) SetEnabled(enabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.enabled = enabled
}

// Enabled returns whether simulated delays are active.
func (c *Config) Enabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.enabled
}

// SetSpeedFactor sets a multiplier for all delays (e.g., 0.1 = 10x faster).
func (c *Config) SetSpeedFactor(factor float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if factor <= 0 {
		factor = 1.0
	}
	c.speedFactor = factor
}

// EffectiveDelay returns the actual delay to use for a given nominal delay.
// Returns 0 if delays are disabled.
func (c *Config) EffectiveDelay(nominal time.Duration) time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if !c.enabled {
		return 0
	}
	return time.Duration(float64(nominal) * c.speedFactor)
}

// Machine manages the lifecycle of a single resource through a defined
// set of state transitions.
type Machine struct {
	mu          sync.RWMutex
	state       State
	stateTime   time.Time
	history     []StateRecord
	transitions map[State]Transition // current state → next transition
	config      *Config
	onTransit   func(from, to State) // optional callback when a transition completes
	timer       *time.Timer
}

// StateRecord captures a historical state change.
type StateRecord struct {
	State     State
	EnteredAt time.Time
}

// NewMachine creates a lifecycle machine starting at initialState.
// transitions maps each state to its next automatic transition.
func NewMachine(initialState State, transitions []Transition, cfg *Config) *Machine {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	tmap := make(map[State]Transition, len(transitions))
	for _, t := range transitions {
		tmap[t.From] = t
	}

	now := time.Now().UTC()
	m := &Machine{
		state:       initialState,
		stateTime:   now,
		history:     []StateRecord{{State: initialState, EnteredAt: now}},
		transitions: tmap,
		config:      cfg,
	}

	// Schedule the first automatic transition if one exists.
	m.scheduleNext()
	return m
}

// OnTransition sets a callback invoked after each automatic transition.
func (m *Machine) OnTransition(fn func(from, to State)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onTransit = fn
}

// State returns the current state.
func (m *Machine) State() State {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state
}

// StateTime returns when the current state was entered.
func (m *Machine) StateTime() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stateTime
}

// History returns all state transitions in order.
func (m *Machine) History() []StateRecord {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]StateRecord, len(m.history))
	copy(result, m.history)
	return result
}

// ForceState immediately sets the state, bypassing transitions.
// Use for external triggers (e.g., user-initiated delete).
func (m *Machine) ForceState(s State) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cancelTimer()
	old := m.state
	m.setState(s)
	cb := m.onTransit
	m.mu.Unlock()
	if cb != nil {
		cb(old, s)
	}
	m.mu.Lock()
	m.scheduleNext()
}

// Stop cancels any pending automatic transition.
func (m *Machine) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cancelTimer()
}

func (m *Machine) setState(s State) {
	now := time.Now().UTC()
	m.state = s
	m.stateTime = now
	m.history = append(m.history, StateRecord{State: s, EnteredAt: now})
}

func (m *Machine) cancelTimer() {
	if m.timer != nil {
		m.timer.Stop()
		m.timer = nil
	}
}

// scheduleNext must be called with m.mu held.
func (m *Machine) scheduleNext() {
	m.cancelTimer()
	t, ok := m.transitions[m.state]
	if !ok {
		return // terminal state
	}

	delay := m.config.EffectiveDelay(t.Delay)
	if delay <= 0 {
		// Instant transition.
		old := m.state
		m.setState(t.To)
		cb := m.onTransit
		if cb != nil {
			go cb(old, t.To)
		}
		m.scheduleNext()
		return
	}

	// Delayed transition.
	to := t.To
	m.timer = time.AfterFunc(delay, func() {
		m.mu.Lock()
		old := m.state
		m.setState(to)
		cb := m.onTransit
		m.scheduleNext()
		m.mu.Unlock()
		if cb != nil {
			cb(old, to)
		}
	})
}
