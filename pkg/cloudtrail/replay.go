package cloudtrail

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

// ReplayConfig controls how events are replayed against a CloudMock endpoint.
type ReplayConfig struct {
	Endpoint    string   // CloudMock gateway URL
	Speed       float64  // 0 = instant, 1.0 = realtime
	FilterWrite bool     // If true, skip read-only events
	Services    []string // If non-empty, only replay matching services
}

// ReplayResult summarizes the outcome of a replay run.
type ReplayResult struct {
	TotalEvents int           `json:"total_events"`
	Replayed    int           `json:"replayed"`
	Skipped     int           `json:"skipped"`
	Succeeded   int           `json:"succeeded"`
	Failed      int           `json:"failed"`
	Errors      []ReplayError `json:"errors,omitempty"`
	Duration    time.Duration `json:"duration"`
}

// ReplayError records a single failed replay attempt.
type ReplayError struct {
	EventName string `json:"event_name"`
	Service   string `json:"service"`
	Error     string `json:"error"`
	Status    int    `json:"status"`
}

// Replay sends CloudTrail events to a CloudMock endpoint according to the
// provided configuration and returns a summary of the results.
func Replay(events []CloudTrailEvent, cfg ReplayConfig) (*ReplayResult, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("endpoint is required")
	}

	start := time.Now()
	totalEvents := len(events)

	// Apply filters.
	filtered := make([]CloudTrailEvent, len(events))
	copy(filtered, events)

	if cfg.FilterWrite {
		filtered = FilterWriteEvents(filtered)
	}
	if len(cfg.Services) > 0 {
		filtered = FilterByServices(filtered, cfg.Services)
	}

	// Sort chronologically.
	SortByTime(filtered)

	skipped := totalEvents - len(filtered)

	result := &ReplayResult{
		TotalEvents: totalEvents,
		Skipped:     skipped,
	}

	client := &http.Client{Timeout: 30 * time.Second}

	for i, event := range filtered {
		// Respect speed: sleep based on time delta between consecutive events.
		if cfg.Speed > 0 && i > 0 {
			prev := filtered[i-1].ParsedTime()
			curr := event.ParsedTime()
			if !prev.IsZero() && !curr.IsZero() {
				delta := curr.Sub(prev)
				if delta > 0 {
					sleepDuration := time.Duration(float64(delta) / cfg.Speed)
					time.Sleep(sleepDuration)
				}
			}
		}

		req, err := ConvertToRequest(event, cfg.Endpoint)
		if err != nil {
			result.Skipped++
			continue
		}

		resp, err := client.Do(req)
		result.Replayed++
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, ReplayError{
				EventName: event.EventName,
				Service:   event.ServiceName(),
				Error:     err.Error(),
				Status:    0,
			})
			continue
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			result.Succeeded++
		} else {
			result.Failed++
			result.Errors = append(result.Errors, ReplayError{
				EventName: event.EventName,
				Service:   event.ServiceName(),
				Error:     fmt.Sprintf("HTTP %d", resp.StatusCode),
				Status:    resp.StatusCode,
			})
		}
	}

	result.Duration = time.Since(start)

	return result, nil
}
