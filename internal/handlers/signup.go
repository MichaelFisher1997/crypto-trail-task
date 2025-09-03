package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/example/solapi/internal/auth"
)

// SignupHandler issues an API key without admin auth. For testing only.
type SignupHandler struct {
	Store auth.APIKeyCreator
}

func NewSignupHandler(store auth.APIKeyCreator) *SignupHandler {
	return &SignupHandler{Store: store}
}

type signupRequest struct {
	Owner string `json:"owner"`
	Email string `json:"email"`
}

type signupResponse struct {
	Key     string `json:"key"`
	Active  bool   `json:"active"`
	Owner   string `json:"owner,omitempty"`
	Email   string `json:"email,omitempty"`
	Created string `json:"created_at"`
}

func (h *SignupHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte(`{"error":"method not allowed"}`))
		return
	}
	var req signupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"bad request"}`))
		return
	}
	// generate a random API key
	var b [32]byte
	_, _ = rand.Read(b[:])
	key := hex.EncodeToString(b[:])
	if err := h.Store.Create(r.Context(), key, true, req.Owner); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(signupResponse{
		Key:     key,
		Active:  true,
		Owner:   req.Owner,
		Email:   req.Email,
		Created: time.Now().UTC().Format(time.RFC3339),
	})
}
