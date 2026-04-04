package admin

import (
	"encoding/csv"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"encoding/json"
)

// PlatformStore is an in-memory store for platform management data.
type PlatformStore struct {
	mu        sync.RWMutex
	apps      []PlatformApp
	keys      []PlatformKey
	audit     []PlatformAuditEntry
	retention map[string]int // resource_type -> days
}

// PlatformApp represents a CloudMock app instance.
type PlatformApp struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Endpoint  string `json:"endpoint"`
	InfraType string `json:"infra_type"`
	Status    string `json:"status"`
	Requests  int64  `json:"request_count"`
	CreatedAt string `json:"created_at"`
}

// PlatformKey represents an API key.
type PlatformKey struct {
	ID         string  `json:"id"`
	Prefix     string  `json:"prefix"`
	Name       string  `json:"name"`
	Role       string  `json:"role"`
	LastUsedAt *string `json:"last_used_at"`
	CreatedAt  string  `json:"created_at"`
}

// PlatformAuditEntry is a single audit log record.
type PlatformAuditEntry struct {
	ID         string `json:"id"`
	Actor      string `json:"actor"`
	ActorType  string `json:"actor_type"`
	Action     string `json:"action"`
	Resource   string `json:"resource"`
	ResourceID string `json:"resource_id"`
	IP         string `json:"ip_address"`
	CreatedAt  string `json:"created_at"`
}

// newPlatformStore initialises a PlatformStore with realistic seed data.
func newPlatformStore() *PlatformStore {
	now := time.Now()
	ts := func(d time.Duration) string { return now.Add(d).UTC().Format(time.RFC3339) }

	lastUsed := ts(-2 * time.Hour)

	apps := []PlatformApp{
		{
			ID:        "app_01",
			Name:      "staging",
			Slug:      "staging",
			Endpoint:  "https://abc123.cloudmock.io",
			InfraType: "shared",
			Status:    "running",
			Requests:  42891,
			CreatedAt: ts(-60 * 24 * time.Hour),
		},
		{
			ID:        "app_02",
			Name:      "ci-tests",
			Slug:      "ci-tests",
			Endpoint:  "https://def456.cloudmock.io",
			InfraType: "dedicated",
			Status:    "running",
			Requests:  1203,
			CreatedAt: ts(-30 * 24 * time.Hour),
		},
	}

	keys := []PlatformKey{
		{
			ID:         "key_01",
			Prefix:     "cm_live_a1b2",
			Name:       "CI Pipeline",
			Role:       "developer",
			LastUsedAt: &lastUsed,
			CreatedAt:  ts(-20 * 24 * time.Hour),
		},
		{
			ID:         "key_02",
			Prefix:     "cm_live_c3d4",
			Name:       "Local Dev",
			Role:       "admin",
			LastUsedAt: nil,
			CreatedAt:  ts(-3 * 24 * time.Hour),
		},
	}

	audit := []PlatformAuditEntry{
		{ID: "aud_01", Actor: "admin@example.com", ActorType: "user", Action: "app.create", Resource: "app/staging", ResourceID: "app_01", IP: "203.0.113.10", CreatedAt: ts(-10 * time.Minute)},
		{ID: "aud_02", Actor: "cm_live_a1b2", ActorType: "key", Action: "aws.request", Resource: "s3/my-bucket", ResourceID: "", IP: "198.51.100.5", CreatedAt: ts(-90 * time.Minute)},
		{ID: "aud_03", Actor: "admin@example.com", ActorType: "user", Action: "key.revoke", Resource: "key/cm_live_e5f6", ResourceID: "key_old", IP: "203.0.113.10", CreatedAt: ts(-30 * time.Hour)},
		{ID: "aud_04", Actor: "admin@example.com", ActorType: "user", Action: "key.create", Resource: "key/cm_live_c3d4", ResourceID: "key_02", IP: "203.0.113.10", CreatedAt: ts(-3 * 24 * time.Hour)},
		{ID: "aud_05", Actor: "ci@example.com", ActorType: "user", Action: "app.update", Resource: "app/ci-tests", ResourceID: "app_02", IP: "192.0.2.42", CreatedAt: ts(-5 * 24 * time.Hour)},
		{ID: "aud_06", Actor: "cm_live_a1b2", ActorType: "key", Action: "aws.request", Resource: "dynamodb/my-table", ResourceID: "", IP: "198.51.100.5", CreatedAt: ts(-6 * 24 * time.Hour)},
		{ID: "aud_07", Actor: "admin@example.com", ActorType: "user", Action: "org.settings.update", Resource: "org/my-org", ResourceID: "org_01", IP: "203.0.113.10", CreatedAt: ts(-7 * 24 * time.Hour)},
	}

	retention := map[string]int{
		"audit_log":      365,
		"request_log":    90,
		"state_snapshot": 30,
	}

	return &PlatformStore{
		apps:      apps,
		keys:      keys,
		audit:     audit,
		retention: retention,
	}
}

// randHex returns n random hex characters.
func randHex(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

// ── /api/platform/apps ────────────────────────────────────────────────────────

func (a *API) handlePlatformApps(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.platform.mu.RLock()
		result := make([]PlatformApp, len(a.platform.apps))
		copy(result, a.platform.apps)
		a.platform.mu.RUnlock()
		writeJSON(w, http.StatusOK, result)

	case http.MethodPost:
		var req struct {
			Name      string `json:"name"`
			InfraType string `json:"infra_type"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Name == "" {
			writeError(w, http.StatusBadRequest, "name is required")
			return
		}
		if req.InfraType == "" {
			req.InfraType = "shared"
		}
		slug := strings.ToLower(strings.ReplaceAll(req.Name, " ", "-"))
		app := PlatformApp{
			ID:        fmt.Sprintf("app_%s", randHex(8)),
			Name:      req.Name,
			Slug:      slug,
			Endpoint:  fmt.Sprintf("https://%s.cloudmock.io", randHex(6)),
			InfraType: req.InfraType,
			Status:    "running",
			Requests:  0,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		}
		a.platform.mu.Lock()
		a.platform.apps = append(a.platform.apps, app)
		a.platform.mu.Unlock()

		a.platform.addAudit(r, "app.create", fmt.Sprintf("app/%s", slug), app.ID)
		writeJSON(w, http.StatusCreated, app)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (a *API) handlePlatformAppByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/platform/apps/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	a.platform.mu.Lock()
	found := false
	for i, app := range a.platform.apps {
		if app.ID == id {
			a.platform.apps = append(a.platform.apps[:i], a.platform.apps[i+1:]...)
			found = true
			break
		}
	}
	a.platform.mu.Unlock()

	if !found {
		writeError(w, http.StatusNotFound, "app not found")
		return
	}
	a.platform.addAudit(r, "app.delete", fmt.Sprintf("app/%s", id), id)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id})
}

// ── /api/platform/keys ────────────────────────────────────────────────────────

func (a *API) handlePlatformKeys(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.platform.mu.RLock()
		result := make([]PlatformKey, len(a.platform.keys))
		copy(result, a.platform.keys)
		a.platform.mu.RUnlock()
		writeJSON(w, http.StatusOK, result)

	case http.MethodPost:
		var req struct {
			Name string `json:"name"`
			Role string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Name == "" {
			writeError(w, http.StatusBadRequest, "name is required")
			return
		}
		if req.Role == "" {
			req.Role = "developer"
		}
		prefix := fmt.Sprintf("cm_live_%s", randHex(4))
		plaintext := fmt.Sprintf("%s%s", prefix, randHex(24))
		key := PlatformKey{
			ID:         fmt.Sprintf("key_%s", randHex(8)),
			Prefix:     prefix,
			Name:       req.Name,
			Role:       req.Role,
			LastUsedAt: nil,
			CreatedAt:  time.Now().UTC().Format(time.RFC3339),
		}
		a.platform.mu.Lock()
		a.platform.keys = append([]PlatformKey{key}, a.platform.keys...)
		a.platform.mu.Unlock()

		a.platform.addAudit(r, "key.create", fmt.Sprintf("key/%s", prefix), key.ID)
		// Return key metadata plus the plaintext (shown once)
		writeJSON(w, http.StatusCreated, map[string]any{
			"id":           key.ID,
			"prefix":       key.Prefix,
			"name":         key.Name,
			"role":         key.Role,
			"last_used_at": key.LastUsedAt,
			"created_at":   key.CreatedAt,
			"key":          plaintext,
		})

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (a *API) handlePlatformKeyByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/platform/keys/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	a.platform.mu.Lock()
	found := false
	var prefix string
	for i, k := range a.platform.keys {
		if k.ID == id {
			prefix = k.Prefix
			a.platform.keys = append(a.platform.keys[:i], a.platform.keys[i+1:]...)
			found = true
			break
		}
	}
	a.platform.mu.Unlock()

	if !found {
		writeError(w, http.StatusNotFound, "key not found")
		return
	}
	a.platform.addAudit(r, "key.revoke", fmt.Sprintf("key/%s", prefix), id)
	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked", "id": id})
}

// ── /api/platform/usage ───────────────────────────────────────────────────────

func (a *API) handlePlatformUsage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	a.platform.mu.RLock()
	apps := make([]PlatformApp, len(a.platform.apps))
	copy(apps, a.platform.apps)
	a.platform.mu.RUnlock()

	var total int64
	appBreakdown := make([]map[string]any, 0, len(apps))
	for _, app := range apps {
		total += app.Requests
		appBreakdown = append(appBreakdown, map[string]any{
			"name":     app.name(),
			"requests": app.Requests,
		})
	}

	const freeLimit = 1000
	const pricePerTenK = 0.5
	billable := float64(total) - freeLimit
	if billable < 0 {
		billable = 0
	}
	cost := (billable / 10000.0) * pricePerTenK

	// Build 30-day daily chart data
	now := time.Now()
	daily := make([]map[string]any, 30)
	for i := 29; i >= 0; i-- {
		d := now.AddDate(0, 0, -i)
		label := fmt.Sprintf("%d/%d", int(d.Month()), d.Day())
		// Deterministic-ish fake daily counts derived from total
		count := int64(float64(total)/30) + int64(rand.Intn(500)) - 250
		if count < 0 {
			count = 0
		}
		daily[29-i] = map[string]any{"day": label, "count": count}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"total":      total,
		"free_limit": freeLimit,
		"cost":       cost,
		"apps":       appBreakdown,
		"daily":      daily,
	})
}

// name is a helper for readability in handlePlatformUsage.
func (a *PlatformApp) name() string { return a.Name }

// ── /api/platform/audit ───────────────────────────────────────────────────────

func (a *API) handlePlatformAudit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	q := r.URL.Query()
	actionFilter := q.Get("action")
	offsetStr := q.Get("offset")
	limitStr := q.Get("limit")

	offset := 0
	limit := 50
	if v, err := strconv.Atoi(offsetStr); err == nil && v >= 0 {
		offset = v
	}
	if v, err := strconv.Atoi(limitStr); err == nil && v > 0 && v <= 500 {
		limit = v
	}

	a.platform.mu.RLock()
	all := make([]PlatformAuditEntry, len(a.platform.audit))
	copy(all, a.platform.audit)
	a.platform.mu.RUnlock()

	// Filter
	filtered := all[:0]
	for _, e := range all {
		if actionFilter != "" && e.Action != actionFilter {
			continue
		}
		filtered = append(filtered, e)
	}

	total := len(filtered)
	end := offset + limit
	if end > total {
		end = total
	}
	var page []PlatformAuditEntry
	if offset < total {
		page = filtered[offset:end]
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"entries": page,
		"total":   total,
		"offset":  offset,
		"limit":   limit,
	})
}

func (a *API) handlePlatformAuditExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	a.platform.mu.RLock()
	entries := make([]PlatformAuditEntry, len(a.platform.audit))
	copy(entries, a.platform.audit)
	a.platform.mu.RUnlock()

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="cloudmock-audit-log.csv"`)

	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"Timestamp", "Actor", "ActorType", "Action", "Resource", "ResourceID", "IP"})
	for _, e := range entries {
		_ = cw.Write([]string{e.CreatedAt, e.Actor, e.ActorType, e.Action, e.Resource, e.ResourceID, e.IP})
	}
	cw.Flush()
}

// ── /api/platform/settings ────────────────────────────────────────────────────

func (a *API) handlePlatformSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.platform.mu.RLock()
		ret := map[string]int{
			"audit_log":      a.platform.retention["audit_log"],
			"request_log":    a.platform.retention["request_log"],
			"state_snapshot": a.platform.retention["state_snapshot"],
		}
		a.platform.mu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{
			"name":        "My Organization",
			"slug":        "my-org",
			"plan":        "Free",
			"owner_email": "admin@example.com",
			"retention":   ret,
		})

	case http.MethodPut:
		var req struct {
			Retention map[string]int `json:"retention"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		a.platform.mu.Lock()
		for k, v := range req.Retention {
			if v > 0 {
				a.platform.retention[k] = v
			}
		}
		ret := map[string]int{
			"audit_log":      a.platform.retention["audit_log"],
			"request_log":    a.platform.retention["request_log"],
			"state_snapshot": a.platform.retention["state_snapshot"],
		}
		a.platform.mu.Unlock()

		a.platform.addAudit(r, "org.settings.update", "org/my-org", "org_01")
		writeJSON(w, http.StatusOK, map[string]any{
			"name":        "My Organization",
			"slug":        "my-org",
			"plan":        "Free",
			"owner_email": "admin@example.com",
			"retention":   ret,
		})

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

// addAudit appends a new audit entry derived from the HTTP request.
func (ps *PlatformStore) addAudit(r *http.Request, action, resource, resourceID string) {
	ip := r.RemoteAddr
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		ip = strings.SplitN(fwd, ",", 2)[0]
	}
	// Strip port from RemoteAddr if present
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}

	entry := PlatformAuditEntry{
		ID:         fmt.Sprintf("aud_%s", randHex(8)),
		Actor:      "admin@example.com",
		ActorType:  "user",
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		IP:         ip,
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}

	ps.mu.Lock()
	ps.audit = append([]PlatformAuditEntry{entry}, ps.audit...)
	if len(ps.audit) > 1000 {
		ps.audit = ps.audit[:1000]
	}
	ps.mu.Unlock()
}
