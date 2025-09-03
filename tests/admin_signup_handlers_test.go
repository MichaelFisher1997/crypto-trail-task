package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/solapi/internal/handlers"
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
	if m.fail { return assertErr }
	return nil
}

type errString string
func (e errString) Error() string { return string(e) }
var assertErr = errString("fail")

func TestSignup_CreatesAndReturnsKey_OK(t *testing.T) {
	mc := &mockCreator{}
	h := handlers.NewSignupHandler(mc)
	body, _ := json.Marshal(map[string]any{"owner":"user1","email":"u@example.com"})
	req := httptest.NewRequest(http.MethodPost, "/public/signup", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if mc.calls != 1 { t.Fatalf("Create calls=%d", mc.calls) }
	if mc.last.key == "" { t.Fatalf("expected random key generated") }
	if !mc.last.active { t.Fatalf("expected active=true") }
	if mc.last.owner != "user1" { t.Fatalf("owner=%q", mc.last.owner) }
}

func TestSignup_MethodNotAllowed(t *testing.T) {
	mc := &mockCreator{}
	h := handlers.NewSignupHandler(mc)
	req := httptest.NewRequest(http.MethodGet, "/public/signup", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed { t.Fatalf("status=%d", rec.Code) }
}
