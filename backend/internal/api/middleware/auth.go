package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type ctxKey int

const (
	adminIDKey ctxKey = 1
	apiKeyIDKey ctxKey = 2
)

func AdminID(ctx context.Context) uuid.UUID {
	v, _ := ctx.Value(adminIDKey).(uuid.UUID)
	return v
}

// APIKeyID returns the id of the api key used to authenticate, or uuid.Nil
// if the request was authenticated via JWT.
func APIKeyID(ctx context.Context) uuid.UUID {
	v, _ := ctx.Value(apiKeyIDKey).(uuid.UUID)
	return v
}

const (
	wwwAuthSessionExpired = "SessionExpired"
	apiKeyPrefix          = "nak_live_"
)

// Authenticator resolves a raw API key to (key id, admin id). Implemented
// by *apikey.Service.
type Authenticator interface {
	Validate(ctx context.Context, raw string) (keyID, adminID uuid.UUID, err error)
}

func RequireAdmin(secret []byte, authenticator Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			if !strings.HasPrefix(h, "Bearer ") {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			raw := strings.TrimPrefix(h, "Bearer ")

			// API keys do not expire, so no SessionExpired header is emitted
			// for this path — only JWTs can be expired.
			if strings.HasPrefix(raw, apiKeyPrefix) {
				if authenticator == nil {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				keyID, adminID, err := authenticator.Validate(r.Context(), raw)
				if err != nil {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				ctx := context.WithValue(r.Context(), adminIDKey, adminID)
				ctx = context.WithValue(ctx, apiKeyIDKey, keyID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			parsed, err := jwt.Parse(raw, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return secret, nil
			})
			if err != nil {
				if errors.Is(err, jwt.ErrTokenExpired) {
					w.Header().Set("WWW-Authenticate", wwwAuthSessionExpired)
				}
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			if !parsed.Valid {
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
