package memory

import (
	"math"
	"sort"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/rum"
)

// Store is an in-memory circular-buffer RUM event store.
type Store struct {
	mu     sync.RWMutex
	events []rum.RUMEvent
	cap    int
	pos    int // next write position
	full   bool
}

// NewStore creates an in-memory store with the given capacity.
func NewStore(capacity int) *Store {
	if capacity <= 0 {
		capacity = 10000
	}
	return &Store{
		events: make([]rum.RUMEvent, capacity),
		cap:    capacity,
	}
}

// WriteEvent appends a single event to the circular buffer.
func (s *Store) WriteEvent(event rum.RUMEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events[s.pos] = event
	s.pos = (s.pos + 1) % s.cap
	if s.pos == 0 {
		s.full = true
	}
	return nil
}

// WriteBatch appends multiple events.
func (s *Store) WriteBatch(events []rum.RUMEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range events {
		s.events[s.pos] = e
		s.pos = (s.pos + 1) % s.cap
		if s.pos == 0 {
			s.full = true
		}
	}
	return nil
}

// snapshot returns a copy of all stored events (newest last).
func (s *Store) snapshot() []rum.RUMEvent {
	var out []rum.RUMEvent
	if s.full {
		// ring has wrapped — read from pos..end then 0..pos
		out = make([]rum.RUMEvent, s.cap)
		copy(out, s.events[s.pos:])
		copy(out[s.cap-s.pos:], s.events[:s.pos])
	} else {
		out = make([]rum.RUMEvent, s.pos)
		copy(out, s.events[:s.pos])
	}
	return out
}

// WebVitalsOverview computes web vitals aggregation on-the-fly.
func (s *Store) WebVitalsOverview() (*rum.WebVitalsOverview, error) {
	s.mu.RLock()
	events := s.snapshot()
	s.mu.RUnlock()

	type vitalAccum struct {
		good, needsImprovement, poor int
		values                       []float64
	}
	vitals := map[string]*vitalAccum{
		"LCP": {}, "FID": {}, "CLS": {}, "TTFB": {}, "FCP": {}, "INP": {},
	}

	sessions := map[string]struct{}{}
	for _, e := range events {
		sessions[e.SessionID] = struct{}{}
		if e.Type != rum.EventWebVital || e.WebVital == nil {
			continue
		}
		wv := e.WebVital
		acc, ok := vitals[wv.Name]
		if !ok {
			continue
		}
		acc.values = append(acc.values, wv.Value)
		switch wv.Rating {
		case "good":
			acc.good++
		case "needs-improvement":
			acc.needsImprovement++
		case "poor":
			acc.poor++
		}
	}

	p75 := func(vals []float64) float64 {
		if len(vals) == 0 {
			return 0
		}
		sorted := make([]float64, len(vals))
		copy(sorted, vals)
		sort.Float64s(sorted)
		idx := int(math.Ceil(float64(len(sorted))*0.75)) - 1
		if idx < 0 {
			idx = 0
		}
		return sorted[idx]
	}

	toRating := func(acc *vitalAccum) rum.VitalRating {
		return rum.VitalRating{
			Good:             acc.good,
			NeedsImprovement: acc.needsImprovement,
			Poor:             acc.poor,
			P75:              p75(acc.values),
		}
	}

	return &rum.WebVitalsOverview{
		LCP:           toRating(vitals["LCP"]),
		FID:           toRating(vitals["FID"]),
		CLS:           toRating(vitals["CLS"]),
		TTFB:          toRating(vitals["TTFB"]),
		FCP:           toRating(vitals["FCP"]),
		INP:           toRating(vitals["INP"]),
		TotalSessions: len(sessions),
	}, nil
}

// PageLoads returns per-route performance aggregations.
func (s *Store) PageLoads() ([]rum.PagePerformance, error) {
	s.mu.RLock()
	events := s.snapshot()
	s.mu.RUnlock()

	type routeAccum struct {
		durations   []float64
		ttfbs       []float64
		transfers   []float64
		count       int
	}

	routes := map[string]*routeAccum{}
	for _, e := range events {
		if e.Type != rum.EventPageLoad || e.PageLoad == nil {
			continue
		}
		pl := e.PageLoad
		route := pl.Route
		if route == "" {
			route = e.URL
		}
		acc, ok := routes[route]
		if !ok {
			acc = &routeAccum{}
			routes[route] = acc
		}
		acc.count++
		acc.durations = append(acc.durations, pl.DurationMs)
		acc.ttfbs = append(acc.ttfbs, pl.TTFB)
		acc.transfers = append(acc.transfers, pl.TransferSizeKB)
	}

	result := make([]rum.PagePerformance, 0, len(routes))
	for route, acc := range routes {
		pp := rum.PagePerformance{
			Route: route,
			Views: acc.count,
		}
		if acc.count > 0 {
			pp.AvgDurationMs = avg(acc.durations)
			pp.P75DurationMs = p75Sorted(acc.durations)
			pp.AvgTTFB = avg(acc.ttfbs)
			pp.AvgTransferSizeKB = avg(acc.transfers)
		}
		result = append(result, pp)
	}

	sort.Slice(result, func(i, j int) bool { return result[i].Views > result[j].Views })
	return result, nil
}

// ErrorGroups aggregates JS errors by fingerprint.
func (s *Store) ErrorGroups() ([]rum.ErrorGroup, error) {
	s.mu.RLock()
	events := s.snapshot()
	s.mu.RUnlock()

	groups := map[string]*rum.ErrorGroup{}
	groupSessions := map[string]map[string]struct{}{}

	for _, e := range events {
		if e.Type != rum.EventJSError || e.JSError == nil {
			continue
		}
		fp := e.JSError.Fingerprint
		if fp == "" {
			continue
		}
		g, ok := groups[fp]
		if !ok {
			g = &rum.ErrorGroup{
				Fingerprint: fp,
				Message:     e.JSError.Message,
				Source:      e.JSError.Source,
				Stack:       e.JSError.Stack,
			}
			groups[fp] = g
			groupSessions[fp] = map[string]struct{}{}
		}
		g.Count++
		if e.Timestamp.After(g.LastSeen) {
			g.LastSeen = e.Timestamp
		}
		groupSessions[fp][e.SessionID] = struct{}{}
	}

	result := make([]rum.ErrorGroup, 0, len(groups))
	for fp, g := range groups {
		g.Sessions = len(groupSessions[fp])
		result = append(result, *g)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Count > result[j].Count })
	return result, nil
}

// PagePerformance returns detailed metrics for a specific route.
func (s *Store) PagePerformance(route string) (*rum.PagePerformance, error) {
	pages, err := s.PageLoads()
	if err != nil {
		return nil, err
	}
	for _, p := range pages {
		if p.Route == route {
			return &p, nil
		}
	}
	return &rum.PagePerformance{Route: route}, nil
}

// Sessions returns recent session summaries.
func (s *Store) Sessions(limit int) ([]rum.SessionSummary, error) {
	s.mu.RLock()
	events := s.snapshot()
	s.mu.RUnlock()

	type sessionAccum struct {
		summary rum.SessionSummary
	}
	sessions := map[string]*sessionAccum{}

	for _, e := range events {
		acc, ok := sessions[e.SessionID]
		if !ok {
			acc = &sessionAccum{
				summary: rum.SessionSummary{
					SessionID: e.SessionID,
					StartedAt: e.Timestamp,
					LastSeen:  e.Timestamp,
					UserAgent: e.UserAgent,
				},
			}
			sessions[e.SessionID] = acc
		}
		if e.Timestamp.Before(acc.summary.StartedAt) {
			acc.summary.StartedAt = e.Timestamp
		}
		if e.Timestamp.After(acc.summary.LastSeen) {
			acc.summary.LastSeen = e.Timestamp
		}
		if e.Type == rum.EventPageLoad {
			acc.summary.PageViews++
		}
		if e.Type == rum.EventJSError {
			acc.summary.ErrorCount++
		}
	}

	result := make([]rum.SessionSummary, 0, len(sessions))
	for _, acc := range sessions {
		result = append(result, acc.summary)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].LastSeen.After(result[j].LastSeen)
	})

	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

// SessionDetail returns all events for a given session.
func (s *Store) SessionDetail(sessionID string) ([]rum.RUMEvent, error) {
	s.mu.RLock()
	events := s.snapshot()
	s.mu.RUnlock()

	var result []rum.RUMEvent
	for _, e := range events {
		if e.SessionID == sessionID {
			result = append(result, e)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.Before(result[j].Timestamp)
	})
	return result, nil
}

// RageClicks returns click events marked as rage clicks from the last N minutes.
func (s *Store) RageClicks(minutes int) ([]rum.ClickEvent, error) {
	s.mu.RLock()
	events := s.snapshot()
	s.mu.RUnlock()

	cutoff := time.Now().Add(-time.Duration(minutes) * time.Minute)
	var result []rum.ClickEvent
	for _, e := range events {
		if e.Type != rum.EventClick || e.Click == nil {
			continue
		}
		if e.Timestamp.Before(cutoff) {
			continue
		}
		if e.Click.IsRage {
			result = append(result, *e.Click)
		}
	}
	return result, nil
}

// UserJourneys returns navigation events for a given session, ordered by time.
func (s *Store) UserJourneys(sessionID string) ([]rum.NavigationEvent, error) {
	s.mu.RLock()
	events := s.snapshot()
	s.mu.RUnlock()

	// Collect navigation events for this session, in order.
	type timestampedNav struct {
		ts  time.Time
		nav rum.NavigationEvent
	}
	var navs []timestampedNav
	for _, e := range events {
		if e.SessionID != sessionID {
			continue
		}
		if e.Type != rum.EventNavigation || e.Navigation == nil {
			continue
		}
		navs = append(navs, timestampedNav{ts: e.Timestamp, nav: *e.Navigation})
	}
	sort.Slice(navs, func(i, j int) bool { return navs[i].ts.Before(navs[j].ts) })

	result := make([]rum.NavigationEvent, len(navs))
	for i, n := range navs {
		result[i] = n.nav
	}
	return result, nil
}

// PerformanceByRoute returns aggregated performance metrics per route.
func (s *Store) PerformanceByRoute() ([]rum.RoutePerformance, error) {
	s.mu.RLock()
	events := s.snapshot()
	s.mu.RUnlock()

	type routeAccum struct {
		durations []float64
		ttfbs     []float64
		count     int
	}

	routes := map[string]*routeAccum{}
	for _, e := range events {
		if e.Type != rum.EventPageLoad || e.PageLoad == nil {
			continue
		}
		pl := e.PageLoad
		route := pl.Route
		if route == "" {
			route = e.URL
		}
		acc, ok := routes[route]
		if !ok {
			acc = &routeAccum{}
			routes[route] = acc
		}
		acc.count++
		acc.durations = append(acc.durations, pl.DurationMs)
		acc.ttfbs = append(acc.ttfbs, pl.TTFB)
	}

	result := make([]rum.RoutePerformance, 0, len(routes))
	for route, acc := range routes {
		rp := rum.RoutePerformance{
			Route: route,
			Views: acc.count,
		}
		if acc.count > 0 {
			rp.AvgDurationMs = avg(acc.durations)
			rp.P75DurationMs = p75Sorted(acc.durations)
			rp.AvgTTFB = avg(acc.ttfbs)
		}
		result = append(result, rp)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Views > result[j].Views })
	return result, nil
}

// --- helpers ---

func avg(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	var sum float64
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

func p75Sorted(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sorted := make([]float64, len(vals))
	copy(sorted, vals)
	sort.Float64s(sorted)
	idx := int(math.Ceil(float64(len(sorted))*0.75)) - 1
	if idx < 0 {
		idx = 0
	}
	return sorted[idx]
}
