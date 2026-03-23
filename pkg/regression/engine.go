package regression

import (
	"context"
	"math"
	"time"

	"github.com/neureaux/cloudmock/pkg/dataplane"
)

type pendingDeploy struct {
	DeployID  string
	Service   string
	DeployAt  time.Time
	EvalTimes []time.Duration
}

// Engine orchestrates continuous regression detection and deploy-triggered evaluation.
type Engine struct {
	source   MetricSource
	store    RegressionStore
	deploys  dataplane.ConfigStore
	config   AlgorithmConfig
	interval time.Duration
	window   time.Duration
	pending  chan pendingDeploy
	stop     chan struct{}
	done     chan struct{}
}

// New creates a new regression detection engine.
func New(source MetricSource, store RegressionStore, deploys dataplane.ConfigStore, cfg AlgorithmConfig, interval, window time.Duration) *Engine {
	return &Engine{
		source:   source,
		store:    store,
		deploys:  deploys,
		config:   cfg,
		interval: interval,
		window:   window,
		pending:  make(chan pendingDeploy, 64),
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
}

// Store exposes the regression store for API handlers.
func (e *Engine) Store() RegressionStore {
	return e.store
}

// Start launches the scan ticker and deploy consumer goroutines.
func (e *Engine) Start(ctx context.Context) {
	go e.scanLoop(ctx)
	go e.deployConsumer(ctx)
}

// Stop signals the engine to shut down and waits for goroutines to finish.
func (e *Engine) Stop() {
	close(e.stop)
	<-e.done
}

func (e *Engine) scanLoop(ctx context.Context) {
	ticker := time.NewTicker(e.interval)
	defer ticker.Stop()

	for {
		select {
		case <-e.stop:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = e.Scan(ctx)
			_ = e.checkResolutions(ctx)
		}
	}
}

func (e *Engine) deployConsumer(ctx context.Context) {
	defer close(e.done)

	for {
		select {
		case <-e.stop:
			return
		case <-ctx.Done():
			return
		case pd := <-e.pending:
			for _, evalDelay := range pd.EvalTimes {
				pd := pd // capture
				delay := evalDelay
				time.AfterFunc(delay, func() {
					_ = e.evaluateDeploy(ctx, pd)
				})
			}
		}
	}
}

// OnDeploy enqueues a deploy event for evaluation at predefined intervals.
func (e *Engine) OnDeploy(deploy dataplane.DeployEvent) {
	e.pending <- pendingDeploy{
		DeployID:  deploy.ID,
		Service:   deploy.Service,
		DeployAt:  deploy.DeployedAt,
		EvalTimes: []time.Duration{10 * time.Millisecond, 50 * time.Millisecond, 100 * time.Millisecond},
	}
}

// Scan performs continuous regression detection across all services.
func (e *Engine) Scan(ctx context.Context) error {
	services, err := e.source.ListServices(ctx)
	if err != nil {
		return err
	}

	now := time.Now()
	afterWindow := TimeWindow{Start: now.Add(-e.window), End: now}
	beforeWindow := TimeWindow{Start: now.Add(-2 * e.window), End: now.Add(-e.window)}

	for _, service := range services {
		afterMetrics, err := e.source.WindowMetrics(ctx, service, "", afterWindow)
		if err != nil {
			continue
		}
		beforeMetrics, err := e.source.WindowMetrics(ctx, service, "", beforeWindow)
		if err != nil {
			continue
		}

		e.runAlgorithms(ctx, beforeMetrics, afterMetrics, "", beforeWindow, afterWindow)
	}

	// Tenant outlier detection
	for _, service := range services {
		tenants, err := e.source.ListTenants(ctx, service)
		if err != nil {
			continue
		}

		fleetMetrics, err := e.source.FleetWindowMetrics(ctx, service, afterWindow)
		if err != nil {
			continue
		}

		for _, tenantID := range tenants {
			tenantMetrics, err := e.source.TenantWindowMetrics(ctx, service, tenantID, afterWindow)
			if err != nil {
				continue
			}

			r := detectTenantOutlier(tenantMetrics, fleetMetrics, e.config.TenantOutlier)
			if r != nil {
				r.TenantID = tenantID
				r.Service = service
				r.DetectedAt = now
				r.WindowAfter = afterWindow
				_ = e.store.Save(ctx, r)
			}
		}
	}

	return nil
}

// runAlgorithms runs all 6 detection algorithms (except tenant outlier) and saves any results.
func (e *Engine) runAlgorithms(ctx context.Context, before, after *WindowMetrics, deployID string, beforeWindow, afterWindow TimeWindow) {
	now := time.Now()

	checks := []*Regression{
		detectLatencyRegression(before, after, e.config.LatencyRegression),
		detectErrorRate(before, after, e.config.ErrorRate),
		detectCacheMiss(before, after, e.config.CacheMiss),
		detectDBFanout(before, after, e.config.DBFanout),
		detectPayloadGrowth(before, after, e.config.PayloadGrowth),
	}

	for _, r := range checks {
		if r == nil {
			continue
		}
		r.DeployID = deployID
		r.DetectedAt = now
		r.WindowBefore = beforeWindow
		r.WindowAfter = afterWindow
		_ = e.store.Save(ctx, r)
	}
}

// evaluateDeploy runs all algorithms comparing pre-deploy vs post-deploy metrics.
func (e *Engine) evaluateDeploy(ctx context.Context, pd pendingDeploy) error {
	beforeWindow := TimeWindow{Start: pd.DeployAt.Add(-e.window), End: pd.DeployAt}
	afterWindow := TimeWindow{Start: pd.DeployAt, End: time.Now()}

	beforeMetrics, err := e.source.WindowMetrics(ctx, pd.Service, "", beforeWindow)
	if err != nil {
		return err
	}

	afterMetrics, err := e.source.WindowMetrics(ctx, pd.Service, "", afterWindow)
	if err != nil {
		return err
	}

	e.runAlgorithmsForDeploy(ctx, beforeMetrics, afterMetrics, pd.DeployID, beforeWindow, afterWindow)
	return nil
}

// runAlgorithmsForDeploy runs algorithms and handles deduplication for deploy-triggered evaluations.
func (e *Engine) runAlgorithmsForDeploy(ctx context.Context, before, after *WindowMetrics, deployID string, beforeWindow, afterWindow TimeWindow) {
	now := time.Now()

	checks := []struct {
		result *Regression
		algo   AlgorithmType
	}{
		{detectLatencyRegression(before, after, e.config.LatencyRegression), AlgoLatencyRegression},
		{detectErrorRate(before, after, e.config.ErrorRate), AlgoErrorRate},
		{detectCacheMiss(before, after, e.config.CacheMiss), AlgoCacheMiss},
		{detectDBFanout(before, after, e.config.DBFanout), AlgoDBFanout},
		{detectPayloadGrowth(before, after, e.config.PayloadGrowth), AlgoPayloadGrowth},
	}

	// Get existing regressions for this deploy to handle deduplication
	existing, _ := e.store.ActiveForDeploy(ctx, deployID)
	existingMap := make(map[AlgorithmType]Regression)
	for _, r := range existing {
		existingMap[r.Algorithm] = r
	}

	for _, c := range checks {
		if c.result == nil {
			continue
		}
		c.result.DeployID = deployID
		c.result.DetectedAt = now
		c.result.WindowBefore = beforeWindow
		c.result.WindowAfter = afterWindow

		if prev, ok := existingMap[c.algo]; ok {
			// Update only if higher confidence
			if c.result.Confidence > prev.Confidence {
				_ = e.store.Save(ctx, c.result)
			}
		} else {
			_ = e.store.Save(ctx, c.result)
		}
	}
}

// checkResolutions auto-resolves active regressions when metrics recover.
func (e *Engine) checkResolutions(ctx context.Context) error {
	active, err := e.store.List(ctx, RegressionFilter{Status: "active"})
	if err != nil {
		return err
	}

	now := time.Now()
	currentWindow := TimeWindow{Start: now.Add(-e.window), End: now}

	for _, r := range active {
		currentMetrics, err := e.source.WindowMetrics(ctx, r.Service, r.Action, currentWindow)
		if err != nil {
			continue
		}

		var currentValue float64
		switch r.Algorithm {
		case AlgoLatencyRegression:
			currentValue = currentMetrics.P99Ms
		case AlgoErrorRate:
			currentValue = currentMetrics.ErrorRate
		case AlgoCacheMiss:
			currentValue = currentMetrics.CacheMissRate
		case AlgoDBFanout:
			currentValue = currentMetrics.AvgSpanCount
		case AlgoPayloadGrowth:
			currentValue = currentMetrics.AvgRespSize
		case AlgoTenantOutlier:
			currentValue = currentMetrics.P99Ms
		default:
			continue
		}

		// If current value is within 10% of the before value, resolve
		if r.BeforeValue == 0 {
			continue
		}
		deviation := math.Abs(currentValue-r.BeforeValue) / r.BeforeValue
		if deviation <= 0.10 {
			_ = e.store.UpdateStatus(ctx, r.ID, "resolved")
		}
	}

	return nil
}
