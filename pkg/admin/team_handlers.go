package admin

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/annotations"
	"github.com/Viridian-Inc/cloudmock/pkg/incident"
)

// handleIncidentComments handles comment operations on incidents.
// POST /api/incidents/{id}/comments — add comment
// GET  /api/incidents/{id}/comments — list comments
func (a *API) handleIncidentComments(w http.ResponseWriter, r *http.Request) {
	if a.incidentService == nil {
		writeError(w, http.StatusServiceUnavailable, "incident service not available")
		return
	}

	// Parse: /api/incidents/{id}/comments
	path := strings.TrimPrefix(r.URL.Path, "/api/incidents/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 || parts[1] != "comments" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	incidentID := parts[0]

	switch r.Method {
	case http.MethodGet:
		comments, err := a.incidentService.Store().GetComments(incidentID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, comments)

	case http.MethodPost:
		var c incident.Comment
		if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		if c.Author == "" {
			writeError(w, http.StatusBadRequest, "author is required")
			return
		}
		if c.Body == "" {
			writeError(w, http.StatusBadRequest, "body is required")
			return
		}
		if c.CreatedAt.IsZero() {
			c.CreatedAt = time.Now()
		}

		if err := a.incidentService.Store().AddComment(incidentID, c); err != nil {
			if err == incident.ErrNotFound {
				writeError(w, http.StatusNotFound, "incident not found")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		a.auditLog(r.Context(), "incident.comment.added", "incident:"+incidentID, map[string]any{
			"author": c.Author,
		})
		writeJSON(w, http.StatusCreated, c)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleAnnotations handles annotation operations.
// GET  /api/annotations — list annotations in time range
// POST /api/annotations — create annotation
func (a *API) handleAnnotations(w http.ResponseWriter, r *http.Request) {
	if a.annotationStore == nil {
		writeError(w, http.StatusServiceUnavailable, "annotation store not available")
		return
	}

	switch r.Method {
	case http.MethodGet:
		q := r.URL.Query()
		service := q.Get("service")

		start := time.Now().Add(-24 * time.Hour)
		end := time.Now()

		if s := q.Get("start"); s != "" {
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				start = t
			}
		}
		if e := q.Get("end"); e != "" {
			if t, err := time.Parse(time.RFC3339, e); err == nil {
				end = t
			}
		}

		results, err := a.annotationStore.List(start, end, service)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, results)

	case http.MethodPost:
		var ann annotations.Annotation
		if err := json.NewDecoder(r.Body).Decode(&ann); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		if ann.Title == "" {
			writeError(w, http.StatusBadRequest, "title is required")
			return
		}

		if err := a.annotationStore.Create(ann); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		a.auditLog(r.Context(), "annotation.created", "annotation:"+ann.ID, map[string]any{
			"title": ann.Title,
		})
		writeJSON(w, http.StatusCreated, ann)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleAnnotationByID handles DELETE /api/annotations/{id}.
func (a *API) handleAnnotationByID(w http.ResponseWriter, r *http.Request) {
	if a.annotationStore == nil {
		writeError(w, http.StatusServiceUnavailable, "annotation store not available")
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/annotations/")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing annotation id")
		return
	}

	switch r.Method {
	case http.MethodDelete:
		if err := a.annotationStore.Delete(id); err != nil {
			if err == annotations.ErrNotFound {
				writeError(w, http.StatusNotFound, "annotation not found")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		a.auditLog(r.Context(), "annotation.deleted", "annotation:"+id, nil)
		w.WriteHeader(http.StatusNoContent)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// ActivityFeedItem represents a single item in the team activity feed.
type ActivityFeedItem struct {
	Type      string    `json:"type"` // "comment", "annotation", "status_change"
	Timestamp time.Time `json:"timestamp"`
	Actor     string    `json:"actor"`
	Summary   string    `json:"summary"`
	Resource  string    `json:"resource"` // e.g., "incident:abc123"
}

// handleActivityFeed returns recent team activity.
// GET /api/activity-feed
func (a *API) handleActivityFeed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var feed []ActivityFeedItem

	// Gather recent incident comments.
	if a.incidentService != nil {
		incidents, err := a.incidentService.Store().List(r.Context(), incident.IncidentFilter{Limit: 20})
		if err == nil {
			for _, inc := range incidents {
				comments, _ := a.incidentService.Store().GetComments(inc.ID)
				for _, c := range comments {
					feed = append(feed, ActivityFeedItem{
						Type:      "comment",
						Timestamp: c.CreatedAt,
						Actor:     c.Author,
						Summary:   "commented on incident: " + inc.Title,
						Resource:  "incident:" + inc.ID,
					})
				}

				// Include status changes as activity.
				if inc.ResolvedAt != nil {
					feed = append(feed, ActivityFeedItem{
						Type:      "status_change",
						Timestamp: *inc.ResolvedAt,
						Actor:     inc.Owner,
						Summary:   "resolved incident: " + inc.Title,
						Resource:  "incident:" + inc.ID,
					})
				}
			}
		}
	}

	// Gather recent annotations.
	if a.annotationStore != nil {
		start := time.Now().Add(-7 * 24 * time.Hour)
		end := time.Now()
		anns, err := a.annotationStore.List(start, end, "")
		if err == nil {
			for _, ann := range anns {
				feed = append(feed, ActivityFeedItem{
					Type:      "annotation",
					Timestamp: ann.Timestamp,
					Actor:     ann.Author,
					Summary:   "created annotation: " + ann.Title,
					Resource:  "annotation:" + ann.ID,
				})
			}
		}
	}

	// Sort by timestamp descending.
	for i := 0; i < len(feed); i++ {
		for j := i + 1; j < len(feed); j++ {
			if feed[j].Timestamp.After(feed[i].Timestamp) {
				feed[i], feed[j] = feed[j], feed[i]
			}
		}
	}

	// Limit to 50 items.
	if len(feed) > 50 {
		feed = feed[:50]
	}

	writeJSON(w, http.StatusOK, feed)
}
