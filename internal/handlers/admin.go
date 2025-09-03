package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/example/solapi/internal/auth"
)

// AdminHandler provides admin-only endpoints like creating API keys.
type AdminHandler struct {
	Store      auth.APIKeyCreator
	AdminToken string
}

func NewAdminHandler(store auth.APIKeyCreator, adminToken string) *AdminHandler {
	return &AdminHandler{Store: store, AdminToken: adminToken}
}

// createKeyRequest is the request payload for creating a key.
// If Key is empty, a random 32-byte hex string will be generated.
// Owner is optional.
type createKeyRequest struct {
	Key   string `json:"key"`
	Owner string `json:"owner"`
}

type createKeyResponse struct {
	Key     string `json:"key"`
	Active  bool   `json:"active"`
	Owner   string `json:"owner,omitempty"`
	Created string `json:"created_at"`
}

// ServeHTTP handles POST /admin/create-key
func (h *AdminHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte(`{"error":"method not allowed"}`))
		return
	}
	if h.AdminToken == "" || r.Header.Get("X-Admin-Token") != h.AdminToken {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"unauthorized"}`))
		return
	}
	var req createKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"bad request"}`))
		return
	}
	key := req.Key
	if key == "" {
		var b [32]byte
		_, _ = rand.Read(b[:])
		key = hex.EncodeToString(b[:])
	}
	ctx := r.Context()
	if err := h.Store.Create(ctx, key, true, req.Owner); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(createKeyResponse{
		Key:     key,
		Active:  true,
		Owner:   req.Owner,
		Created: time.Now().UTC().Format(time.RFC3339),
	})
}
