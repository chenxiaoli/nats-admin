package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const testSecret = "test-secret-test-secret-test-secret-1234"

func mintToken(t *testing.T, exp time.Time, sub string) string {
	t.Helper()
	if sub == "" {
		sub = uuid.New().String()
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  sub,
		"role": "admin",
		"exp":  exp.Unix(),
		"iat":  time.Now().Unix(),
	})
	signed, err := tok.SignedString([]byte(testSecret))
	if err != nil {
		t.Fatal(err)
	}
	return signed
}

func protectedHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func runMiddleware(req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	mw := RequireAdmin([]byte(testSecret), nil)(http.HandlerFunc(protectedHandler))
	mw.ServeHTTP(rr, req)
	return rr
}

func TestRequireAdmin_MissingHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	rr := runMiddleware(req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d", rr.Code)
	}
	if got := rr.Header().Get("WWW-Authenticate"); got != "" {
		t.Fatalf("expected no WWW-Authenticate, got %q", got)
	}
}

func TestRequireAdmin_MalformedHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer not-a-jwt")
	rr := runMiddleware(req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d", rr.Code)
	}
	if got := rr.Header().Get("WWW-Authenticate"); got != "" {
		t.Fatalf("expected no WWW-Authenticate, got %q", got)
	}
}

func TestRequireAdmin_ExpiredToken(t *testing.T) {
	tok := mintToken(t, time.Now().Add(-1*time.Hour), "")
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := runMiddleware(req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d", rr.Code)
	}
	if got := rr.Header().Get("WWW-Authenticate"); got != "SessionExpired" {
		t.Fatalf("WWW-Authenticate: got %q, want SessionExpired", got)
	}
}

func TestRequireAdmin_BadSignature(t *testing.T) {
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  uuid.New().String(),
		"exp":  time.Now().Add(time.Hour).Unix(),
	})
	signed, _ := tok.SignedString([]byte("wrong-secret-wrong-secret-wrong-secret-123"))
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	rr := runMiddleware(req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d", rr.Code)
	}
	if got := rr.Header().Get("WWW-Authenticate"); got != "" {
		t.Fatalf("expected no WWW-Authenticate, got %q", got)
	}
}

func TestRequireAdmin_NonUUIDSub(t *testing.T) {
	tok := mintToken(t, time.Now().Add(time.Hour), "not-a-uuid")
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := runMiddleware(req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d", rr.Code)
	}
	if got := rr.Header().Get("WWW-Authenticate"); got != "" {
		t.Fatalf("expected no WWW-Authenticate, got %q", got)
	}
}

func TestRequireAdmin_ValidToken(t *testing.T) {
	tok := mintToken(t, time.Now().Add(time.Hour), "")
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := runMiddleware(req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rr.Code)
	}
	if body := rr.Body.String(); body != "ok" {
		t.Fatalf("body: got %q", body)
	}
}

type stubAuthenticator struct {
	keyID, adminID uuid.UUID
	err            error
}

func (s *stubAuthenticator) Validate(ctx context.Context, raw string) (keyID, adminID uuid.UUID, err error) {
	return s.keyID, s.adminID, s.err
}

func TestRequireAdmin_APIKey_Valid(t *testing.T) {
	adminID := uuid.New()
	keyID := uuid.New()
	stub := &stubAuthenticator{keyID: keyID, adminID: adminID}
	var captured uuid.UUID
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = AdminID(r.Context())
		w.WriteHeader(http.StatusOK)
	})
	mw := RequireAdmin([]byte(testSecret), stub)(handler)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer nak_live_somekeyvalue")
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rr.Code)
	}
	if captured != adminID {
		t.Fatalf("AdminID not set: got %v, want %v", captured, adminID)
	}
}

func TestRequireAdmin_APIKey_Invalid(t *testing.T) {
	stub := &stubAuthenticator{err: errors.New("not found")}
	mw := RequireAdmin([]byte(testSecret), stub)(http.HandlerFunc(protectedHandler))
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer nak_live_bogus")
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d, want 401", rr.Code)
	}
}

func TestRequireAdmin_APIKey_NoAuthenticator(t *testing.T) {
	mw := RequireAdmin([]byte(testSecret), nil)(http.HandlerFunc(protectedHandler))
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer nak_live_anything")
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d, want 401", rr.Code)
	}
}
