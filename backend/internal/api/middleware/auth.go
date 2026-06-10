package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
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

// Authenticator is implemented by the api key service so the middleware
// can resolve a raw key to (key id, admin id) without importing the full
// service struct.
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

// NewAPIKeyAuthenticator returns an Authenticator backed by a pgx pool.
// We define it here so the middleware package has no direct dependency on
// the apikey service struct; it just needs the Validate contract.
func NewAPIKeyAuthenticator(pool *pgxpool.Pool) Authenticator {
	return &poolAuthenticator{pool: pool}
}

type poolAuthenticator struct {
	pool *pgxpool.Pool
}

func (p *poolAuthenticator) Validate(ctx context.Context, raw string) (keyID, adminID uuid.UUID, err error) {
	sum := sha256.Sum256([]byte(raw))
	hash := hex.EncodeToString(sum[:])
	err = p.pool.QueryRow(ctx,
		`SELECT id, admin_id FROM api_keys WHERE key_hash = $1 AND revoked_at IS NULL`,
		hash).Scan(&keyID, &adminID)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	_, _ = p.pool.Exec(ctx, `UPDATE api_keys SET last_used_at = NOW() WHERE id = $1`, keyID)
	return keyID, adminID, nil
}
