package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFormatTimePtr_Nil(t *testing.T) {
	if got := formatTimePtr(nil); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestCreate_RequiresName(t *testing.T) {
	h := &APIKeysHandler{}
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()
	h.Create(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", rr.Code)
	}
}

func TestList_EmptyResponse(t *testing.T) {
	out := []keySummary{}
	b, _ := json.Marshal(out)
	if string(b) != "[]" {
		t.Fatalf("expected empty array, got %s", b)
	}
}

func TestRevoke_BadID(t *testing.T) {
	h := &APIKeysHandler{}
	req := httptest.NewRequest("DELETE", "/x", nil)
	rr := httptest.NewRecorder()
	h.Revoke(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", rr.Code)
	}
}
