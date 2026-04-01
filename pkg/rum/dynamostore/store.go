// Package dynamostore implements rum.RUMStore backed by DynamoDB
// via the generic dynamostore package. RUM events use TTL for auto-expiry.
package dynamostore

import (
	"encoding/json"
	"sort"
	"time"

	ds "github.com/neureaux/cloudmock/pkg/dynamostore"
	"github.com/neureaux/cloudmock/pkg/rum"
)

const featureRUM = "RUM"

// defaultTTLDays is the default number of days before RUM events expire.
const defaultTTLDays = 7

// Store implements rum.RUMStore backed by DynamoDB.
type Store struct {
	db      *ds.Store
	ttlDays int
}

// New creates a DynamoDB-backed RUM store with the given TTL in days.
func New(db *ds.Store, ttlDays int) *Store {
	if ttlDays <= 0 {
		ttlDays = defaultTTLDays
	}
	return &Store{db: db, ttlDays: ttlDays}
}

func (s *Store) ttlEpoch() *int64 {
	t := time.Now().Add(time.Duration(s.ttlDays) * 24 * time.Hour).Unix()
	return &t
}

func (s *Store) WriteEvent(event rum.RUMEvent) error {
	return s.db.PutWithTTL(nil, featureRUM, event.ID, event, s.ttlEpoch())
}

func (s *Store) WriteBatch(events []rum.RUMEvent) error {
	for _, e := range events {
		if err := s.WriteEvent(e); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) allEvents() ([]rum.RUMEvent, error) {
	var events []rum.RUMEvent
	if err := s.db.List(nil, featureRUM, &events); err != nil {
		return nil, err
	}
	return events, nil
}

func (s *Store) WebVitalsOverview() (*rum.WebVitalsOverview, error) {
	events, err := s.allEvents()
	if err != nil {
		return nil, err
	}

	overview := &rum.WebVitalsOverview{}
	sessions := make(map[string]bool)
	vitalValues := map[string][]float64{}

	for _, e := range events {
		sessions[e.SessionID] = true
		if e.WebVital != nil {
			vitalValues[e.WebVital.Name] = append(vitalValues[e.WebVital.Name], e.WebVital.Value)
			rating := e.WebVital.Rating
			switch e.WebVital.Name {
			case "LCP":
				addRating(&overview.LCP, rating)
			case "FID":
				addRating(&overview.FID, rating)
			case "CLS":
				addRating(&overview.CLS, rating)
			case "TTFB":
				addRating(&overview.TTFB, rating)
			case "FCP":
				addRating(&overview.FCP, rating)
			case "INP":
				addRating(&overview.INP, rating)
			}
		}
	}

	overview.TotalSessions = len(sessions)
	setP75(&overview.LCP, vitalValues["LCP"])
	setP75(&overview.FID, vitalValues["FID"])
	setP75(&overview.CLS, vitalValues["CLS"])
	setP75(&overview.TTFB, vitalValues["TTFB"])
	setP75(&overview.FCP, vitalValues["FCP"])
	setP75(&overview.INP, vitalValues["INP"])

	return overview, nil
}

func (s *Store) PageLoads() ([]rum.PagePerformance, error) {
	events, err := s.allEvents()
	if err != nil {
		return nil, err
	}
	return computePagePerformance(events), nil
}

func (s *Store) ErrorGroups() ([]rum.ErrorGroup, error) {
	events, err := s.allEvents()
	if err != nil {
		return nil, err
	}
	return computeErrorGroups(events), nil
}

func (s *Store) PagePerformance(route string) (*rum.PagePerformance, error) {
	perfs, err := s.PageLoads()
	if err != nil {
		return nil, err
	}
	for _, p := range perfs {
		if p.Route == route {
			return &p, nil
		}
	}
	return nil, nil
}

func (s *Store) Sessions(limit int) ([]rum.SessionSummary, error) {
	events, err := s.allEvents()
	if err != nil {
		return nil, err
	}
	return computeSessions(events, limit), nil
}

func (s *Store) SessionDetail(sessionID string) ([]rum.RUMEvent, error) {
	events, err := s.allEvents()
	if err != nil {
		return nil, err
	}
	var result []rum.RUMEvent
	for _, e := range events {
		if e.SessionID == sessionID {
			result = append(result, e)
		}
	}
	return result, nil
}

func (s *Store) RageClicks(minutes int) ([]rum.ClickEvent, error) {
	events, err := s.allEvents()
	if err != nil {
		return nil, err
	}
	cutoff := time.Now().Add(-time.Duration(minutes) * time.Minute)
	var result []rum.ClickEvent
	for _, e := range events {
		if e.Click != nil && e.Click.IsRage && e.Timestamp.After(cutoff) {
			result = append(result, *e.Click)
		}
	}
	return result, nil
}

func (s *Store) UserJourneys(sessionID string) ([]rum.NavigationEvent, error) {
	events, err := s.allEvents()
	if err != nil {
		return nil, err
	}
	var result []rum.NavigationEvent
	for _, e := range events {
		if e.SessionID == sessionID && e.Navigation != nil {
			result = append(result, *e.Navigation)
		}
	}
	return result, nil
}

func (s *Store) PerformanceByRoute() ([]rum.RoutePerformance, error) {
	events, err := s.allEvents()
	if err != nil {
		return nil, err
	}
	return computeRoutePerformance(events), nil
}

// --- helpers ---

func addRating(vr *rum.VitalRating, rating string) {
	switch rating {
	case "good":
		vr.Good++
	case "needs-improvement":
		vr.NeedsImprovement++
	case "poor":
		vr.Poor++
	}
}

func setP75(vr *rum.VitalRating, vals []float64) {
	if len(vals) == 0 {
		return
	}
	sort.Float64s(vals)
	idx := int(float64(len(vals)) * 0.75)
	if idx >= len(vals) {
		idx = len(vals) - 1
	}
	vr.P75 = vals[idx]
}

func computePagePerformance(events []rum.RUMEvent) []rum.PagePerformance {
	type acc struct {
		views    int
		durSum   float64
		ttfbSum  float64
		sizeSum  float64
		durVals  []float64
	}
	routes := map[string]*acc{}
	for _, e := range events {
		if e.PageLoad == nil {
			continue
		}
		a, ok := routes[e.PageLoad.Route]
		if !ok {
			a = &acc{}
			routes[e.PageLoad.Route] = a
		}
		a.views++
		a.durSum += e.PageLoad.DurationMs
		a.ttfbSum += e.PageLoad.TTFB
		a.sizeSum += e.PageLoad.TransferSizeKB
		a.durVals = append(a.durVals, e.PageLoad.DurationMs)
	}

	var result []rum.PagePerformance
	for route, a := range routes {
		sort.Float64s(a.durVals)
		p75Idx := int(float64(len(a.durVals)) * 0.75)
		if p75Idx >= len(a.durVals) {
			p75Idx = len(a.durVals) - 1
		}
		result = append(result, rum.PagePerformance{
			Route:             route,
			Views:             a.views,
			AvgDurationMs:     a.durSum / float64(a.views),
			P75DurationMs:     a.durVals[p75Idx],
			AvgTTFB:           a.ttfbSum / float64(a.views),
			AvgTransferSizeKB: a.sizeSum / float64(a.views),
		})
	}
	return result
}

func computeErrorGroups(events []rum.RUMEvent) []rum.ErrorGroup {
	type errAcc struct {
		message  string
		source   string
		stack    string
		count    int
		sessions map[string]bool
		lastSeen time.Time
	}
	groups := map[string]*errAcc{}
	for _, e := range events {
		if e.JSError == nil {
			continue
		}
		fp := e.JSError.Fingerprint
		if fp == "" {
			fp = e.JSError.Message
		}
		a, ok := groups[fp]
		if !ok {
			a = &errAcc{
				message:  e.JSError.Message,
				source:   e.JSError.Source,
				stack:    e.JSError.Stack,
				sessions: map[string]bool{},
			}
			groups[fp] = a
		}
		a.count++
		a.sessions[e.SessionID] = true
		if e.Timestamp.After(a.lastSeen) {
			a.lastSeen = e.Timestamp
		}
	}

	var result []rum.ErrorGroup
	for fp, a := range groups {
		result = append(result, rum.ErrorGroup{
			Fingerprint: fp,
			Message:     a.message,
			Source:      a.source,
			Count:       a.count,
			Sessions:    len(a.sessions),
			LastSeen:    a.lastSeen,
			Stack:       a.stack,
		})
	}
	return result
}

func computeSessions(events []rum.RUMEvent, limit int) []rum.SessionSummary {
	type sessAcc struct {
		startedAt  time.Time
		lastSeen   time.Time
		pageViews  int
		errorCount int
		userAgent  string
	}
	sessions := map[string]*sessAcc{}
	for _, e := range events {
		a, ok := sessions[e.SessionID]
		if !ok {
			a = &sessAcc{startedAt: e.Timestamp, userAgent: e.UserAgent}
			sessions[e.SessionID] = a
		}
		if e.Timestamp.Before(a.startedAt) {
			a.startedAt = e.Timestamp
		}
		if e.Timestamp.After(a.lastSeen) {
			a.lastSeen = e.Timestamp
		}
		if e.Type == rum.EventPageLoad {
			a.pageViews++
		}
		if e.Type == rum.EventJSError {
			a.errorCount++
		}
	}

	var result []rum.SessionSummary
	for id, a := range sessions {
		result = append(result, rum.SessionSummary{
			SessionID:  id,
			StartedAt:  a.startedAt,
			LastSeen:   a.lastSeen,
			PageViews:  a.pageViews,
			ErrorCount: a.errorCount,
			UserAgent:  a.userAgent,
		})
	}

	// Sort by last_seen descending.
	sort.Slice(result, func(i, j int) bool {
		return result[i].LastSeen.After(result[j].LastSeen)
	})

	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result
}

func computeRoutePerformance(events []rum.RUMEvent) []rum.RoutePerformance {
	type acc struct {
		durSum  float64
		ttfbSum float64
		views   int
		durVals []float64
	}
	routes := map[string]*acc{}
	for _, e := range events {
		if e.PageLoad == nil {
			continue
		}
		a, ok := routes[e.PageLoad.Route]
		if !ok {
			a = &acc{}
			routes[e.PageLoad.Route] = a
		}
		a.views++
		a.durSum += e.PageLoad.DurationMs
		a.ttfbSum += e.PageLoad.TTFB
		a.durVals = append(a.durVals, e.PageLoad.DurationMs)
	}

	var result []rum.RoutePerformance
	for route, a := range routes {
		sort.Float64s(a.durVals)
		p75Idx := int(float64(len(a.durVals)) * 0.75)
		if p75Idx >= len(a.durVals) {
			p75Idx = len(a.durVals) - 1
		}
		result = append(result, rum.RoutePerformance{
			Route:         route,
			AvgDurationMs: a.durSum / float64(a.views),
			P75DurationMs: a.durVals[p75Idx],
			AvgTTFB:       a.ttfbSum / float64(a.views),
			Views:         a.views,
		})
	}
	return result
}

// Compile-time interface check.
var _ rum.RUMStore = (*Store)(nil)

// Ensure json is imported (used indirectly by dynamostore).
var _ = json.Marshal
