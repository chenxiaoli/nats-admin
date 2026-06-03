package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/allen/nats-admin/internal/api/middleware"
	"github.com/allen/nats-admin/internal/tenant"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type TenantsHandler struct {
	svc *tenant.Service
}

func NewTenantsHandler(svc *tenant.Service) *TenantsHandler { return &TenantsHandler{svc: svc} }

type createTenantReq struct {
	Name             string `json:"name"`
	Slug             string `json:"slug"`
	JSMemoryStorage  int64  `json:"js_max_memory_storage"`
	JSDiskStorage    int64  `json:"js_max_disk_storage"`
	JSMaxStreams     int32  `json:"js_max_streams"`
	JSMaxConsumers   int32  `json:"js_max_consumers"`
	MaxConnections   int32  `json:"max_connections"`
	MaxSubscriptions int32  `json:"max_subscriptions"`
}

type createTenantResp struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	Slug          string    `json:"slug"`
	AccountPubKey string    `json:"account_public_key"`
	AccountSeed   string    `json:"account_seed,omitempty"`
	AccountJWT    string    `json:"account_jwt"`
}

func (h *TenantsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createTenantReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	out, err := h.svc.Create(r.Context(), tenant.CreateRequest{
		Name: req.Name, Slug: req.Slug,
		JSMemoryStorage: req.JSMemoryStorage, JSDiskStorage: req.JSDiskStorage,
		JSMaxStreams: req.JSMaxStreams, JSMaxConsumers: req.JSMaxConsumers,
		MaxConnections: req.MaxConnections, MaxSubscriptions: req.MaxSubscriptions,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	middleware.AuditSet(r, "tenant", map[string]any{"id": out.Tenant.ID, "name": req.Name})
	writeJSON(w, createTenantResp{
		ID: out.Tenant.ID, Name: out.Tenant.Name, Slug: out.Tenant.Slug,
		AccountPubKey: out.Tenant.AccountPublicKey, AccountSeed: out.Seed,
		AccountJWT: out.Tenant.AccountJWT,
	})
}

func (h *TenantsHandler) List(w http.ResponseWriter, r *http.Request) {
	ts, err := h.svc.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, ts)
}

func (h *TenantsHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	t, err := h.svc.Get(r.Context(), id)
	if errors.Is(err, tenant.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, t)
}

func (h *TenantsHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	var req createTenantReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	t, err := h.svc.UpdateLimits(r.Context(), id, tenant.CreateRequest{
		Name: req.Name, Slug: req.Slug,
		JSMemoryStorage: req.JSMemoryStorage, JSDiskStorage: req.JSDiskStorage,
		JSMaxStreams: req.JSMaxStreams, JSMaxConsumers: req.JSMaxConsumers,
		MaxConnections: req.MaxConnections, MaxSubscriptions: req.MaxSubscriptions,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	middleware.AuditSet(r, "tenant", map[string]any{"id": id, "action": "update_limits"})
	writeJSON(w, t)
}

func (h *TenantsHandler) Suspend(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	if err := h.svc.Suspend(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	middleware.AuditSet(r, "tenant", map[string]any{"id": id, "action": "suspend"})
	w.WriteHeader(http.StatusNoContent)
}

func (h *TenantsHandler) Activate(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	if err := h.svc.Activate(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	middleware.AuditSet(r, "tenant", map[string]any{"id": id, "action": "activate"})
	w.WriteHeader(http.StatusNoContent)
}

func (h *TenantsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := uuid.Parse(chi.URLParam(r, "id"))
	if err := h.svc.Delete(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	middleware.AuditSet(r, "tenant", map[string]any{"id": id, "action": "delete"})
	w.WriteHeader(http.StatusNoContent)
}
