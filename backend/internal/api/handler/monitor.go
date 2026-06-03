package handler

import "net/http"

type MonitorHandler struct{}

func (h *MonitorHandler) Servers(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]any{"servers": []any{}})
}
func (h *MonitorHandler) Tenants(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]any{"accounts": []any{}})
}
func (h *MonitorHandler) TenantStats(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]any{"stats": map[string]any{}})
}
func (h *MonitorHandler) WebSocket(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "ws not implemented", http.StatusNotImplemented)
}
