package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type JetStreamHandler struct{}

func (h *JetStreamHandler) ListStreams(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]any{"streams": []any{}})
}

func (h *JetStreamHandler) CreateStream(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]any{"created": body["name"]})
}

func (h *JetStreamHandler) DeleteStream(w http.ResponseWriter, r *http.Request) {
	_ = chi.URLParam(r, "name")
	w.WriteHeader(http.StatusNoContent)
}

func (h *JetStreamHandler) PurgeStream(w http.ResponseWriter, r *http.Request) {
	_ = chi.URLParam(r, "name")
	w.WriteHeader(http.StatusNoContent)
}

func (h *JetStreamHandler) ListKV(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]any{"buckets": []any{}})
}

func (h *JetStreamHandler) CreateKV(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]any{"created": body["bucket"]})
}

func (h *JetStreamHandler) DeleteKV(w http.ResponseWriter, r *http.Request) {
	_ = chi.URLParam(r, "bucket")
	w.WriteHeader(http.StatusNoContent)
}
