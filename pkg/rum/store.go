package rum

// RUMStore defines the interface for persisting and querying RUM events.
type RUMStore interface {
	// WriteEvent persists a single RUM event.
	WriteEvent(event RUMEvent) error

	// WriteBatch persists multiple RUM events atomically.
	WriteBatch(events []RUMEvent) error

	// WebVitalsOverview returns an aggregate view of core web vitals.
	WebVitalsOverview() (*WebVitalsOverview, error)

	// PageLoads returns per-route performance metrics.
	PageLoads() ([]PagePerformance, error)

	// ErrorGroups returns JS errors grouped by fingerprint.
	ErrorGroups() ([]ErrorGroup, error)

	// PagePerformance returns detailed performance for a specific route.
	PagePerformance(route string) (*PagePerformance, error)

	// Sessions returns a list of recent session summaries.
	Sessions(limit int) ([]SessionSummary, error)

	// SessionDetail returns all events for a given session.
	SessionDetail(sessionID string) ([]RUMEvent, error)

	// RageClicks returns rage click events from the last N minutes.
	RageClicks(minutes int) ([]ClickEvent, error)

	// UserJourneys returns navigation events for a given session, ordered by time.
	UserJourneys(sessionID string) ([]NavigationEvent, error)

	// PerformanceByRoute returns aggregated performance metrics per route.
	PerformanceByRoute() ([]RoutePerformance, error)
}
