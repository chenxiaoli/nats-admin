package middleware

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type auditEntry struct {
	adminID  uuid.UUID
	tenantID uuid.UUID
	action   string
	resource string
	detail   any
}

type ctxAuditKey struct{}

func WithAudit(pool *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			entry := &auditEntry{action: r.Method + " " + r.URL.Path}
			ctx := context.WithValue(r.Context(), ctxAuditKey{}, entry)
			rr := &recordingWriter{ResponseWriter: w, status: 200}
			next.ServeHTTP(rr, r.WithContext(ctx))

			entry.adminID = AdminID(r.Context())
			entry.tenantID = TenantID(r.Context())
			detail, _ := json.Marshal(entry.detail)
			if pool == nil {
				// Audit disabled (e.g. unit tests that don't wire a DB).
				return
			}
			_, err := pool.Exec(r.Context(),
				`INSERT INTO audit_logs (admin_id, tenant_id, action, resource, ip_addr, detail) VALUES ($1,$2,$3,$4,$5,$6)`,
				entry.adminID, entry.tenantID, entry.action, entry.resource, r.RemoteAddr, detail)
			if err != nil {
				log.Printf("audit insert failed: %v", err)
			}
		})
	}
}

func AuditSet(r *http.Request, resource string, detail any) {
	v := r.Context().Value(ctxAuditKey{})
	if v == nil {
		return
	}
	e := v.(*auditEntry)
	e.resource = resource
	e.detail = detail
}

type recordingWriter struct {
	http.ResponseWriter
	status int
}

func (r *recordingWriter) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

var _ = time.Now
