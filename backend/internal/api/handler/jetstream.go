package handler

import (
	"encoding/json"
	"net/http"

	"github.com/chenxiaoli/nats-admin/internal/api/middleware"
	"github.com/chenxiaoli/nats-admin/internal/jetstream"
	"github.com/chenxiaoli/nats-admin/internal/monitor"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type JetStreamHandler struct {
	admin *jetstream.Admin
}

func NewJetStreamHandler(admin *jetstream.Admin) *JetStreamHandler {
	return &JetStreamHandler{admin: admin}
}

func (h *JetStreamHandler) ListStreams(w http.ResponseWriter, r *http.Request) {
	tid := middleware.TenantID(r.Context())
	streams, err := h.admin.ListStreams(r.Context(), tid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if streams == nil {
		streams = []jetstream.StreamInfo{}
	}
	writeJSON(w, map[string]any{"streams": streams})
}

func (h *JetStreamHandler) CreateStream(w http.ResponseWriter, r *http.Request) {
	tid := middleware.TenantID(r.Context())
	var req jetstream.CreateStreamReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	si, err := h.admin.CreateStream(r.Context(), tid, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, si)
}

func (h *JetStreamHandler) DeleteStream(w http.ResponseWriter, r *http.Request) {
	tid := middleware.TenantID(r.Context())
	name := chi.URLParam(r, "name")
	if err := h.admin.DeleteStream(r.Context(), tid, name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *JetStreamHandler) PurgeStream(w http.ResponseWriter, r *http.Request) {
	tid := middleware.TenantID(r.Context())
	name := chi.URLParam(r, "name")
	if err := h.admin.PurgeStream(r.Context(), tid, name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *JetStreamHandler) ListKV(w http.ResponseWriter, r *http.Request) {
	tid := middleware.TenantID(r.Context())
	buckets, err := h.admin.ListKV(r.Context(), tid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if buckets == nil {
		buckets = []jetstream.KVBucketInfo{}
	}
	writeJSON(w, map[string]any{"buckets": buckets})
}

func (h *JetStreamHandler) CreateKV(w http.ResponseWriter, r *http.Request) {
	tid := middleware.TenantID(r.Context())
	var req jetstream.CreateKVReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	ki, err := h.admin.CreateKV(r.Context(), tid, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, ki)
}

func (h *JetStreamHandler) DeleteKV(w http.ResponseWriter, r *http.Request) {
	tid := middleware.TenantID(r.Context())
	bucket := chi.URLParam(r, "bucket")
	if err := h.admin.DeleteKV(r.Context(), tid, bucket); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type MonitorHandler struct {
	mon *monitor.Monitor
	hub *monitor.Hub
}

func NewMonitorHandler(mon *monitor.Monitor, hub *monitor.Hub) *MonitorHandler {
	return &MonitorHandler{mon: mon, hub: hub}
}

func (h *MonitorHandler) Servers(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]any{"servers": h.mon.Servers()})
}

func (h *MonitorHandler) Tenants(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]any{"accounts": h.mon.Accounts()})
}

func (h *MonitorHandler) TenantStats(w http.ResponseWriter, r *http.Request) {
	pubKey := chi.URLParam(r, "id")
	stats, err := h.mon.RequestAccountStats(r.Context(), pubKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"stats": stats})
}

func (h *MonitorHandler) WebSocket(w http.ResponseWriter, r *http.Request) {
	if h.hub == nil {
		http.Error(w, "ws not configured", http.StatusNotImplemented)
		return
	}
	h.hub.HandleWS(w, r)
}

// unused but keeps uuid import
var _ = uuid.UUID{}
