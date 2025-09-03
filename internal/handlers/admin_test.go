package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockCreator struct{
	calls int
	last struct{ key string; active bool; owner string }
	fail bool
}

func (m *mockCreator) Create(_ context.Context, key string, active bool, owner string) error {
	m.calls++
	m.last.key = key
	m.last.active = active
	m.last.owner = owner
	if m.fail { return errString("fail") }
	return nil
}

type errString string
func (e errString) Error() string { return string(e) }

func TestAdmin_Unauthorized(t *testing.T) {
	mc := &mockCreator{}
	h := NewAdminHandler(mc, "secret")
	req := httptest.NewRequest(http.MethodPost, "/admin/create-key", bytes.NewReader([]byte(`{"owner":"acme"}`)))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestAdmin_GenerateAndStore_OK(t *testing.T) {
	mc := &mockCreator{}
	h := NewAdminHandler(mc, "secret")
	body, _ := json.Marshal(map[string]any{"owner":"acme"})
	req := httptest.NewRequest(http.MethodPost, "/admin/create-key", bytes.NewReader(body))
	req.Header.Set("X-Admin-Token", "secret")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if mc.calls != 1 { t.Fatalf("Create calls=%d", mc.calls) }
	if mc.last.key == "" { t.Fatalf("expected random key generated") }
	if !mc.last.active { t.Fatalf("expected active=true") }
	if mc.last.owner != "acme" { t.Fatalf("owner=%q", mc.last.owner) }
}

func TestAdmin_MethodNotAllowed(t *testing.T) {
	mc := &mockCreator{}
	h := NewAdminHandler(mc, "secret")
	req := httptest.NewRequest(http.MethodGet, "/admin/create-key", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed { t.Fatalf("status=%d", rec.Code) }
}

func TestAdmin_BadJSON(t *testing.T) {
	mc := &mockCreator{}
	h := NewAdminHandler(mc, "secret")
	req := httptest.NewRequest(http.MethodPost, "/admin/create-key", bytes.NewReader([]byte("{bad")))
	req.Header.Set("X-Admin-Token", "secret")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest { t.Fatalf("status=%d", rec.Code) }
}

func TestAdmin_StoreError(t *testing.T) {
	mc := &mockCreator{fail: true}
	h := NewAdminHandler(mc, "secret")
	body, _ := json.Marshal(map[string]any{"owner":"acme"})
	req := httptest.NewRequest(http.MethodPost, "/admin/create-key", bytes.NewReader(body))
	req.Header.Set("X-Admin-Token", "secret")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError { t.Fatalf("status=%d", rec.Code) }
}
