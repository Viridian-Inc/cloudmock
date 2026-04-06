package middleware

import (
	"context"
	"net/http"

	"github.com/neureaux/cloudmock/pkg/platform/model"
)

// AuditWriter can persist audit log entries.
type AuditWriter interface {
	Append(ctx context.Context, e *model.AuditEntry) error
}

// Audit is the audit-logging middleware.
type Audit struct {
	writer AuditWriter
}

// NewAudit creates an Audit middleware backed by the given writer.
func NewAudit(writer AuditWriter) *Audit {
	return &Audit{writer: writer}
}

// Handler returns an http.Handler that appends an audit entry for each
// authenticated request. Requests with no auth context are passed through
// without logging.
func (a *Audit) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ac := AuthFromContext(r.Context())
		if ac != nil {
			entry := &model.AuditEntry{
				TenantID:  ac.TenantID,
				ActorID:   ac.ActorID,
				ActorType: ac.ActorType,
				Action:    r.Method + " " + r.URL.Path,
			}
			go func() {
				_ = a.writer.Append(context.Background(), entry)
			}()
		}
		next.ServeHTTP(w, r)
	})
}
