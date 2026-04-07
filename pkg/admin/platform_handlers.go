package admin

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	platformmodel "github.com/Viridian-Inc/cloudmock/pkg/platform/model"
	platformstore "github.com/Viridian-Inc/cloudmock/pkg/platform/store"
	"github.com/Viridian-Inc/cloudmock/pkg/saas/tenant"
)

// PlatformStore is an in-memory fallback for local mode (no Postgres).
type PlatformStore struct {
	mu        sync.RWMutex
	apps      []PlatformApp
	keys      []PlatformKey
	audit     []PlatformAuditEntry
	retention map[string]int
}

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

type PlatformKey struct {
	ID         string  `json:"id"`
	Prefix     string  `json:"prefix"`
	Name       string  `json:"name"`
	Role       string  `json:"role"`
	LastUsedAt *string `json:"last_used_at"`
	CreatedAt  string  `json:"created_at"`
}

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

func newPlatformStore(retentionAudit, retentionReq, retentionSnap int) *PlatformStore {
	return &PlatformStore{
		apps:  []PlatformApp{},
		keys:  []PlatformKey{},
		audit: []PlatformAuditEntry{},
		retention: map[string]int{
			"audit_log":      retentionAudit,
			"request_log":    retentionReq,
			"state_snapshot": retentionSnap,
		},
	}
}

func randHex(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

// hasPlatformDB returns true when the real Postgres stores are wired.
func (a *API) hasPlatformDB() bool {
	return a.platformApps != nil
}

// extractAuth reads auth context from headers set by Clerk middleware.
func extractAuth(r *http.Request) platformmodel.AuthContext {
	return platformmodel.AuthContext{
		TenantID:  r.Header.Get("X-Tenant-ID"),
		ActorID:   r.Header.Get("X-User-ID"),
		ActorType: "user",
		Role:      r.Header.Get("X-Org-Role"),
	}
}

func extractIP(r *http.Request) net.IP {
	ip := r.RemoteAddr
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		ip = strings.SplitN(fwd, ",", 2)[0]
	}
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	return net.ParseIP(strings.TrimSpace(ip))
}

// ── /api/platform/pricing ────────────────────────────────────────────────────

func (a *API) handlePlatformPricing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"free_request_limit": a.cfg.Billing.FreeRequestLimit,
		"price_per_10k":      a.cfg.Billing.PricePerTenK,
		"usage_window_days":  a.cfg.Billing.UsageWindowDays,
		"default_infra_type": a.cfg.Billing.DefaultInfraType,
	})
}

// ── /api/platform/apps ────────────────────────────────────────────────────────

func (a *API) handlePlatformApps(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Real DB
		if a.hasPlatformDB() {
			auth := extractAuth(r)
			apps, err := a.platformApps.ListByTenant(r.Context(), auth.TenantID)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			// Also get usage counts per app
			result := make([]map[string]any, 0, len(apps))
			for _, app := range apps {
				count, _ := a.platformUsage.GetCurrentPeriodCount(r.Context(), app.TenantID)
				result = append(result, map[string]any{
					"id":            app.ID,
					"name":          app.Name,
					"slug":          app.Slug,
					"endpoint":      app.Endpoint,
					"infra_type":    app.InfraType,
					"status":        app.Status,
					"request_count": count,
					"created_at":    app.CreatedAt.Format(time.RFC3339),
				})
			}
			writeJSON(w, http.StatusOK, result)
			return
		}
		// Fallback: in-memory
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
			req.InfraType = a.cfg.Billing.DefaultInfraType
		}
		slug := strings.ToLower(strings.ReplaceAll(req.Name, " ", "-"))

		// Real DB
		if a.hasPlatformDB() {
			auth := extractAuth(r)
			app := &platformmodel.App{
				TenantID:  auth.TenantID,
				Name:      req.Name,
				Slug:      slug,
				Endpoint:  fmt.Sprintf("https://%s.cloudmock.app", slug),
				InfraType: req.InfraType,
				Status:    "provisioning",
			}
			if err := a.platformApps.Create(r.Context(), app); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			// Provision infrastructure in background
			if a.orchestrator != nil {
				t := &tenant.Tenant{ID: auth.TenantID, Name: req.Name, Slug: slug, Tier: "hosted", Status: "active"}
				go func() {
					ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
					defer cancel()
					if err := a.orchestrator.Provision(ctx, t); err == nil {
						app.Status = "running"
						app.FlyAppName = t.FlyAppName
						app.FlyMachineID = t.FlyMachineID
						_ = a.platformApps.Update(ctx, app)
					}
				}()
			}
			a.recordAudit(r, "app.create", "app", app.ID)
			writeJSON(w, http.StatusCreated, app)
			return
		}

		// Fallback: in-memory
		app := PlatformApp{
			ID:        fmt.Sprintf("app_%s", randHex(8)),
			Name:      req.Name,
			Slug:      slug,
			Endpoint:  fmt.Sprintf("https://%s.cloudmock.app", slug),
			InfraType: req.InfraType,
			Status:    "running",
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		}
		a.platform.mu.Lock()
		a.platform.apps = append(a.platform.apps, app)
		a.platform.mu.Unlock()
		a.addAuditFromRequest(r, "app.create", fmt.Sprintf("app/%s", slug), app.ID)
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

	if a.hasPlatformDB() {
		if err := a.platformApps.Delete(r.Context(), id); err != nil {
			if err == platformstore.ErrNotFound {
				writeError(w, http.StatusNotFound, "app not found")
			} else {
				writeError(w, http.StatusInternalServerError, err.Error())
			}
			return
		}
		a.recordAudit(r, "app.delete", "app", id)
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id})
		return
	}

	// Fallback
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
	a.addAuditFromRequest(r, "app.delete", fmt.Sprintf("app/%s", id), id)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id})
}

// ── /api/platform/keys ────────────────────────────────────────────────────────

func (a *API) handlePlatformKeys(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if a.hasPlatformDB() {
			appID := r.URL.Query().Get("app_id")
			if appID == "" {
				writeError(w, http.StatusBadRequest, "app_id query parameter required")
				return
			}
			keys, err := a.platformKeys.ListByApp(r.Context(), appID)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, keys)
			return
		}
		// Fallback
		a.platform.mu.RLock()
		result := make([]PlatformKey, len(a.platform.keys))
		copy(result, a.platform.keys)
		a.platform.mu.RUnlock()
		writeJSON(w, http.StatusOK, result)

	case http.MethodPost:
		var req struct {
			Name  string `json:"name"`
			Role  string `json:"role"`
			AppID string `json:"app_id"`
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

		if a.hasPlatformDB() {
			auth := extractAuth(r)
			if req.AppID == "" {
				writeError(w, http.StatusBadRequest, "app_id is required")
				return
			}
			plaintext, key, err := a.platformKeys.Create(r.Context(), auth.TenantID, req.AppID, req.Name, req.Role)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			a.recordAudit(r, "key.create", "api_key", key.ID)
			writeJSON(w, http.StatusCreated, map[string]any{
				"id":           key.ID,
				"prefix":       key.Prefix,
				"name":         key.Name,
				"role":         key.Role,
				"last_used_at": key.LastUsedAt,
				"created_at":   key.CreatedAt,
				"key":          plaintext, // shown once, never stored
			})
			return
		}

		// Fallback: in-memory
		prefix := fmt.Sprintf("cmk_%s", randHex(4))
		plaintext := fmt.Sprintf("%s%s", prefix, randHex(24))
		key := PlatformKey{
			ID:        fmt.Sprintf("key_%s", randHex(8)),
			Prefix:    prefix,
			Name:      req.Name,
			Role:      req.Role,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		}
		a.platform.mu.Lock()
		a.platform.keys = append([]PlatformKey{key}, a.platform.keys...)
		a.platform.mu.Unlock()
		a.addAuditFromRequest(r, "key.create", fmt.Sprintf("key/%s", prefix), key.ID)
		writeJSON(w, http.StatusCreated, map[string]any{
			"id": key.ID, "prefix": key.Prefix, "name": key.Name,
			"role": key.Role, "created_at": key.CreatedAt, "key": plaintext,
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

	if a.hasPlatformDB() {
		if err := a.platformKeys.Revoke(r.Context(), id); err != nil {
			if err == platformstore.ErrNotFound {
				writeError(w, http.StatusNotFound, "key not found")
			} else {
				writeError(w, http.StatusInternalServerError, err.Error())
			}
			return
		}
		a.recordAudit(r, "key.revoke", "api_key", id)
		writeJSON(w, http.StatusOK, map[string]string{"status": "revoked", "id": id})
		return
	}

	// Fallback
	a.platform.mu.Lock()
	found := false
	for i, k := range a.platform.keys {
		if k.ID == id {
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
	a.addAuditFromRequest(r, "key.revoke", fmt.Sprintf("key/%s", id), id)
	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked", "id": id})
}

// ── /api/platform/usage ───────────────────────────────────────────────────────

func (a *API) handlePlatformUsage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	windowDays := a.cfg.Billing.UsageWindowDays
	if windowDays <= 0 {
		windowDays = 30
	}
	freeLimit := a.cfg.Billing.FreeRequestLimit
	pricePerTenK := a.cfg.Billing.PricePerTenK

	if a.hasPlatformDB() {
		auth := extractAuth(r)
		now := time.Now().UTC()
		monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		windowStart := now.AddDate(0, 0, -windowDays)

		// Total for current billing period
		total, _ := a.platformUsage.GetCurrentPeriodCount(r.Context(), auth.TenantID)

		// Per-app breakdown
		apps, _ := a.platformApps.ListByTenant(r.Context(), auth.TenantID)
		appBreakdown := make([]map[string]any, 0, len(apps))
		for _, app := range apps {
			records, _ := a.platformUsage.GetByTenant(r.Context(), auth.TenantID, monthStart, now)
			var appCount int64
			for _, rec := range records {
				if rec.AppID == app.ID {
					appCount += rec.RequestCount
				}
			}
			appBreakdown = append(appBreakdown, map[string]any{
				"name":     app.Name,
				"requests": appCount,
			})
		}

		// Daily breakdown from usage records
		daily := make([]map[string]any, windowDays)
		records, _ := a.platformUsage.GetByTenant(r.Context(), auth.TenantID, windowStart, now)
		dailyCounts := make(map[string]int64)
		for _, rec := range records {
			day := rec.PeriodStart.Format("1/2")
			dailyCounts[day] += rec.RequestCount
		}
		for i := windowDays - 1; i >= 0; i-- {
			d := now.AddDate(0, 0, -i)
			label := fmt.Sprintf("%d/%d", int(d.Month()), d.Day())
			daily[windowDays-1-i] = map[string]any{"day": label, "count": dailyCounts[label]}
		}

		billable := float64(total) - float64(freeLimit)
		if billable < 0 {
			billable = 0
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"total":         total,
			"free_limit":    freeLimit,
			"price_per_10k": pricePerTenK,
			"cost":          (billable / 10000.0) * pricePerTenK,
			"apps":          appBreakdown,
			"daily":         daily,
		})
		return
	}

	// Fallback: trace store or in-memory
	var total int64
	appBreakdown := make([]map[string]any, 0)
	if a.tenantStore != nil {
		tenants, err := a.tenantStore.List(r.Context())
		if err == nil {
			for _, t := range tenants {
				total += t.RequestCount
				appBreakdown = append(appBreakdown, map[string]any{"name": t.Name, "requests": t.RequestCount})
			}
		}
	}
	if len(appBreakdown) == 0 {
		a.platform.mu.RLock()
		for _, app := range a.platform.apps {
			total += app.Requests
			appBreakdown = append(appBreakdown, map[string]any{"name": app.Name, "requests": app.Requests})
		}
		a.platform.mu.RUnlock()
	}

	now := time.Now()
	daily := make([]map[string]any, windowDays)
	if a.traceStore != nil {
		for i := windowDays - 1; i >= 0; i-- {
			d := now.AddDate(0, 0, -i)
			dayStart := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
			daily[windowDays-1-i] = map[string]any{
				"day":   fmt.Sprintf("%d/%d", int(d.Month()), d.Day()),
				"count": a.traceStore.CountInRange(dayStart, dayStart.Add(24*time.Hour)),
			}
		}
	} else {
		avg := total / int64(windowDays)
		for i := windowDays - 1; i >= 0; i-- {
			d := now.AddDate(0, 0, -i)
			daily[windowDays-1-i] = map[string]any{"day": fmt.Sprintf("%d/%d", int(d.Month()), d.Day()), "count": avg}
		}
	}

	billable := float64(total) - float64(freeLimit)
	if billable < 0 {
		billable = 0
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"total": total, "free_limit": freeLimit, "price_per_10k": pricePerTenK,
		"cost": (billable / 10000.0) * pricePerTenK, "apps": appBreakdown, "daily": daily,
	})
}

// ── /api/platform/audit ───────────────────────────────────────────────────────

func (a *API) handlePlatformAudit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	q := r.URL.Query()
	offset, limit := 0, 50
	if v, err := strconv.Atoi(q.Get("offset")); err == nil && v >= 0 {
		offset = v
	}
	if v, err := strconv.Atoi(q.Get("limit")); err == nil && v > 0 && v <= 500 {
		limit = v
	}

	if a.hasPlatformDB() {
		auth := extractAuth(r)
		filter := platformstore.AuditFilter{
			TenantID: auth.TenantID,
			Action:   q.Get("action"),
			Limit:    limit,
			Offset:   offset,
		}
		entries, err := a.platformAudit.Query(r.Context(), filter)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		total, _ := a.platformAudit.Count(r.Context(), filter)
		writeJSON(w, http.StatusOK, map[string]any{
			"entries": entries,
			"total":   total,
			"offset":  offset,
			"limit":   limit,
		})
		return
	}

	// Fallback: in-memory
	actionFilter := q.Get("action")
	a.platform.mu.RLock()
	all := make([]PlatformAuditEntry, len(a.platform.audit))
	copy(all, a.platform.audit)
	a.platform.mu.RUnlock()

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
	writeJSON(w, http.StatusOK, map[string]any{"entries": page, "total": total, "offset": offset, "limit": limit})
}

func (a *API) handlePlatformAuditExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if a.hasPlatformDB() {
		auth := extractAuth(r)
		entries, _ := a.platformAudit.Query(r.Context(), platformstore.AuditFilter{TenantID: auth.TenantID, Limit: 10000})
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", `attachment; filename="cloudmock-audit-log.csv"`)
		cw := csv.NewWriter(w)
		_ = cw.Write([]string{"Timestamp", "Actor", "ActorType", "Action", "ResourceType", "ResourceID", "IP"})
		for _, e := range entries {
			_ = cw.Write([]string{e.CreatedAt.Format(time.RFC3339), e.ActorID, e.ActorType, e.Action, e.ResourceType, e.ResourceID, e.IPAddress.String()})
		}
		cw.Flush()
		return
	}

	// Fallback
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
		if a.hasPlatformDB() {
			auth := extractAuth(r)
			if auth.TenantID != "" {
				// Get tenant info
				var tenantName, tenantSlug, tenantPlan string
				if a.tenantStore != nil {
					t, err := a.tenantStore.Get(r.Context(), auth.TenantID)
					if err == nil {
						tenantName = t.Name
						tenantSlug = t.Slug
						tenantPlan = t.Tier
					}
				}
				// Get retention settings
				retMap := map[string]int{
					"audit_log": a.cfg.Retention.AuditLog, "request_log": a.cfg.Retention.RequestLog,
					"state_snapshot": a.cfg.Retention.StateSnapshot,
				}
				policies, _ := a.platformRetention.GetByTenant(r.Context(), auth.TenantID)
				for _, p := range policies {
					retMap[p.ResourceType] = p.RetentionDays
				}
				// Get usage
				total, _ := a.platformUsage.GetCurrentPeriodCount(r.Context(), auth.TenantID)

				writeJSON(w, http.StatusOK, map[string]any{
					"name":          tenantName,
					"slug":          tenantSlug,
					"plan":          tenantPlan,
					"owner_email":   r.Header.Get("X-User-Email"),
					"request_count": total,
					"retention":     retMap,
				})
				return
			}
		}

		// Fallback for local mode
		a.platform.mu.RLock()
		ret := a.platform.retention
		a.platform.mu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{
			"name": a.cfg.Region, "slug": "local", "plan": "local", "retention": ret,
		})

	case http.MethodPut:
		var req struct {
			Retention map[string]int `json:"retention"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if a.hasPlatformDB() {
			auth := extractAuth(r)
			for resourceType, days := range req.Retention {
				if days > 0 {
					_ = a.platformRetention.Upsert(r.Context(), auth.TenantID, resourceType, days)
				}
			}
			a.recordAudit(r, "settings.update", "retention", "")
			writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
			return
		}

		// Fallback
		a.platform.mu.Lock()
		for k, v := range req.Retention {
			if v > 0 {
				a.platform.retention[k] = v
			}
		}
		a.platform.mu.Unlock()
		a.addAuditFromRequest(r, "org.settings.update", "org/settings", "")
		writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// ── /api/platform/environments ───────────────────────────────────────────────

func (a *API) handlePlatformEnvironments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	envs := []string{"Local"}
	if a.hasPlatformDB() {
		auth := extractAuth(r)
		apps, _ := a.platformApps.ListByTenant(r.Context(), auth.TenantID)
		for _, app := range apps {
			if app.Status == "running" {
				envs = append(envs, app.Name)
			}
		}
	} else if a.tenantStore != nil {
		tenants, _ := a.tenantStore.List(r.Context())
		for _, t := range tenants {
			if t.FlyAppName != "" {
				envs = append(envs, t.Name)
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"environments": envs})
}

// ── audit helpers ────────────────────────────────────────────────────────────

// recordAudit writes to the real audit store (Postgres).
func (a *API) recordAudit(r *http.Request, action, resourceType, resourceID string) {
	if a.platformAudit == nil {
		a.addAuditFromRequest(r, action, resourceType, resourceID)
		return
	}
	auth := extractAuth(r)
	entry := &platformmodel.AuditEntry{
		TenantID:     auth.TenantID,
		ActorID:      auth.ActorID,
		ActorType:    auth.ActorType,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		IPAddress:    extractIP(r),
		UserAgent:    r.UserAgent(),
	}
	_ = a.platformAudit.Append(r.Context(), entry)
}

// addAuditFromRequest is the in-memory fallback.
func (a *API) addAuditFromRequest(r *http.Request, action, resource, resourceID string) {
	actor := r.Header.Get("X-User-Email")
	actorType := "user"
	if actor == "" {
		if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, "Bearer cmk_") {
			actor = authHeader[7:]
			if len(actor) > 15 {
				actor = actor[:15]
			}
			actorType = "key"
		}
	}
	if actor == "" {
		actor = "local"
		actorType = "system"
	}

	maxEntries := a.cfg.Billing.MaxAuditEntries
	if maxEntries <= 0 {
		maxEntries = 1000
	}

	entry := PlatformAuditEntry{
		ID: fmt.Sprintf("aud_%s", randHex(8)), Actor: actor, ActorType: actorType,
		Action: action, Resource: resource, ResourceID: resourceID,
		IP: extractIP(r).String(), CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	a.platform.mu.Lock()
	a.platform.audit = append([]PlatformAuditEntry{entry}, a.platform.audit...)
	if len(a.platform.audit) > maxEntries {
		a.platform.audit = a.platform.audit[:maxEntries]
	}
	a.platform.mu.Unlock()
}

// addAudit is the store-level fallback for backward compat.
func (ps *PlatformStore) addAudit(r *http.Request, action, resource, resourceID string) {
	actor := r.Header.Get("X-User-Email")
	if actor == "" {
		actor = "local"
	}
	entry := PlatformAuditEntry{
		ID: fmt.Sprintf("aud_%s", randHex(8)), Actor: actor, ActorType: "user",
		Action: action, Resource: resource, ResourceID: resourceID,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	ps.mu.Lock()
	ps.audit = append([]PlatformAuditEntry{entry}, ps.audit...)
	if len(ps.audit) > 1000 {
		ps.audit = ps.audit[:1000]
	}
	ps.mu.Unlock()
}
