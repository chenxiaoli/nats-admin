package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	pool      *pgxpool.Pool
	jwtSecret []byte
	expiry    time.Duration
}

func NewAuthHandler(pool *pgxpool.Pool, secret []byte, expiry time.Duration) *AuthHandler {
	return &AuthHandler{pool: pool, jwtSecret: secret, expiry: expiry}
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResp struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	AdminID   uuid.UUID `json:"admin_id"`
	Role      string    `json:"role"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	var (
		id   uuid.UUID
		hash string
		role string
	)
	err := h.pool.QueryRow(r.Context(),
		`SELECT id, password_hash, role FROM admin_users WHERE email = $1`, req.Email,
	).Scan(&id, &hash, &role)
	if err == pgx.ErrNoRows {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)) != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	exp := time.Now().Add(h.expiry)
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  id.String(),
		"role": role,
		"exp":  exp.Unix(),
		"iat":  time.Now().Unix(),
	})
	signed, err := tok.SignedString(h.jwtSecret)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, loginResp{Token: signed, ExpiresAt: exp, AdminID: id, Role: role})
}

func writeJSON(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(body)
}
