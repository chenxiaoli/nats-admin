package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/chenxiaoli/nats-admin/internal/api/middleware"
	"github.com/chenxiaoli/nats-admin/internal/credential"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type CredentialsHandler struct {
	svc *credential.Service
}

func NewCredentialsHandler(svc *credential.Service) *CredentialsHandler {
	return &CredentialsHandler{svc: svc}
}

type issueReq struct {
	Name      string     `json:"name"`
	PubAllow  []string   `json:"pub_allow"`
	SubAllow  []string   `json:"sub_allow"`
	ExpiresAt *time.Time `json:"expires_at"`
}

func (h *CredentialsHandler) List(w http.ResponseWriter, r *http.Request) {
	tid, _ := uuid.Parse(chi.URLParam(r, "id"))
	creds, err := h.svc.List(r.Context(), tid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	type out struct {
		ID         uuid.UUID  `json:"id"`
		Name       string     `json:"name"`
		UserPubKey string     `json:"user_public_key"`
		PubAllow   []string   `json:"pub_allow"`
		SubAllow   []string   `json:"sub_allow"`
		RevokedAt  *time.Time `json:"revoked_at"`
		ExpiresAt  *time.Time `json:"expires_at"`
		CreatedAt  time.Time  `json:"created_at"`
	}
	resp := make([]out, 0, len(creds))
	for _, c := range creds {
		resp = append(resp, out{
			ID: c.ID, Name: c.Name, UserPubKey: c.UserPublicKey,
			PubAllow: c.PubAllow, SubAllow: c.SubAllow,
			RevokedAt: c.RevokedAt, ExpiresAt: c.ExpiresAt, CreatedAt: c.CreatedAt,
		})
	}
	writeJSON(w, resp)
}

func (h *CredentialsHandler) Issue(w http.ResponseWriter, r *http.Request) {
	tid, _ := uuid.Parse(chi.URLParam(r, "id"))
	var req issueReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	issued, err := h.svc.Issue(r.Context(), tid, credential.IssueRequest{
		Name: req.Name, PubAllow: req.PubAllow, SubAllow: req.SubAllow, ExpiresAt: req.ExpiresAt,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	middleware.AuditSet(r, "credential", map[string]any{
		"tenant_id": tid, "credential_id": issued.ID, "name": issued.Name,
	})
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Credential-Id", issued.ID.String())
	w.Write([]byte(issued.Creds))
}

func (h *CredentialsHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	tid, _ := uuid.Parse(chi.URLParam(r, "id"))
	cid, _ := uuid.Parse(chi.URLParam(r, "cid"))
	if err := h.svc.Revoke(r.Context(), tid, cid); err != nil {
		if errors.Is(err, credential.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	middleware.AuditSet(r, "credential", map[string]any{"tenant_id": tid, "credential_id": cid, "action": "revoke"})
	w.WriteHeader(http.StatusNoContent)
}
