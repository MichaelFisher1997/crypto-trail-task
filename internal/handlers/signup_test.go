package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockCreator2 struct{
	calls int
	last struct{ key string; active bool; owner string }
	fail bool
}

func (m *mockCreator2) Create(_ context.Context, key string, active bool, owner string) error {
	m.calls++
	m.last.key = key
	m.last.active = active
	m.last.owner = owner
	if m.fail { return errors.New("fail") }
	return nil
}

func TestSignup_CreatesAndReturnsKey_OK(t *testing.T) {
	mc := &mockCreator2{}
	h := NewSignupHandler(mc)
	body, _ := json.Marshal(map[string]any{"owner":"user1","email":"u@example.com"})
	req := httptest.NewRequest(http.MethodPost, "/public/signup", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK { t.Fatalf("status=%d", rec.Code) }
	if mc.calls != 1 { t.Fatalf("Create calls=%d", mc.calls) }
	if mc.last.key == "" { t.Fatalf("expected random key generated") }
	if !mc.last.active { t.Fatalf("expected active=true") }
	if mc.last.owner != "user1" { t.Fatalf("owner=%q", mc.last.owner) }
}

func TestSignup_MethodNotAllowed(t *testing.T) {
	mc := &mockCreator2{}
	h := NewSignupHandler(mc)
	req := httptest.NewRequest(http.MethodGet, "/public/signup", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed { t.Fatalf("status=%d", rec.Code) }
}

func TestSignup_BadJSON(t *testing.T) {
	mc := &mockCreator2{}
	h := NewSignupHandler(mc)
	req := httptest.NewRequest(http.MethodPost, "/public/signup", bytes.NewReader([]byte("{bad")))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest { t.Fatalf("status=%d", rec.Code) }
}

func TestSignup_StoreError(t *testing.T) {
	mc := &mockCreator2{fail: true}
	h := NewSignupHandler(mc)
	body, _ := json.Marshal(map[string]any{"owner":"user1"})
	req := httptest.NewRequest(http.MethodPost, "/public/signup", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError { t.Fatalf("status=%d", rec.Code) }
}
