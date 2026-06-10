package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/chenxiaoli/nats-admin/internal/apikey"
	"github.com/chenxiaoli/nats-admin/internal/api/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type APIKeysHandler struct {
	svc *apikey.Service
}

func NewAPIKeysHandler(svc *apikey.Service) *APIKeysHandler { return &APIKeysHandler{svc: svc} }

type createKeyReq struct {
	Name string `json:"name"`
}

type createKeyResp struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Key       string    `json:"key"`
	CreatedAt string    `json:"created_at"`
}

type keySummary struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	KeyPrefix  string    `json:"key_prefix"`
	LastUsedAt *string   `json:"last_used_at"`
	CreatedAt  string    `json:"created_at"`
	RevokedAt  *string   `json:"revoked_at"`
}

func (h *APIKeysHandler) Create(w http.ResponseWriter, r *http.Request) {
	adminID := middleware.AdminID(r.Context())
	var req createKeyReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	created, err := h.svc.Create(r.Context(), adminID, req.Name)
	if err != nil {
		http.Error(w, "create failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(createKeyResp{
		ID:        created.Key.ID,
		Name:      created.Key.Name,
		Key:       created.Raw,
		CreatedAt: created.Key.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

func (h *APIKeysHandler) List(w http.ResponseWriter, r *http.Request) {
	adminID := middleware.AdminID(r.Context())
	keys, err := h.svc.List(r.Context(), adminID)
	if err != nil {
		http.Error(w, "list failed", http.StatusInternalServerError)
		return
	}
	out := make([]keySummary, 0, len(keys))
	for _, k := range keys {
		out = append(out, keySummary{
			ID:         k.ID,
			Name:       k.Name,
			KeyPrefix:  k.KeyPrefix,
			LastUsedAt: formatTimePtr(k.LastUsedAt),
			CreatedAt:  k.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			RevokedAt:  formatTimePtr(k.RevokedAt),
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (h *APIKeysHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	adminID := middleware.AdminID(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	if err := h.svc.Revoke(r.Context(), adminID, id); err != nil {
		if errors.Is(err, apikey.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, apikey.ErrNotActive) {
			http.Error(w, "already revoked", http.StatusConflict)
			return
		}
		http.Error(w, "revoke failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func formatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format("2006-01-02T15:04:05Z07:00")
	return &s
}
