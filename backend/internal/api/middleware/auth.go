package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type ctxKey int

const adminIDKey ctxKey = 1

func AdminID(ctx context.Context) uuid.UUID {
	v, _ := ctx.Value(adminIDKey).(uuid.UUID)
	return v
}

func RequireAdmin(secret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			if !strings.HasPrefix(h, "Bearer ") {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			raw := strings.TrimPrefix(h, "Bearer ")
			parsed, err := jwt.Parse(raw, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return secret, nil
			})
			if err != nil || !parsed.Valid {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			claims, _ := parsed.Claims.(jwt.MapClaims)
			sub, _ := claims["sub"].(string)
			id, err := uuid.Parse(sub)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), adminIDKey, id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
