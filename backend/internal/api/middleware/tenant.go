package middleware

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type tenantKey struct{}

func TenantID(ctx context.Context) uuid.UUID {
	v, _ := ctx.Value(tenantKey{}).(uuid.UUID)
	return v
}

func InjectTenant() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, err := uuid.Parse(chi.URLParam(r, "id"))
			if err != nil {
				http.Error(w, "invalid tenant id", http.StatusBadRequest)
				return
			}
			ctx := context.WithValue(r.Context(), tenantKey{}, id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
