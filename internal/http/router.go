package apihttp

import (
	"net/http"

	"github.com/example/solapi/internal/auth"
	"github.com/example/solapi/internal/handlers"
	"github.com/example/solapi/internal/rate"
)

// chain applies middlewares in order over a handler.
func chain(h http.Handler, mws ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// NewRouter wires routes and middlewares using the standard library only.
func NewRouter(bh *handlers.BalanceHandler, lm *rate.LimiterMap, store auth.APIKeyStore) http.Handler {
	mux := http.NewServeMux()

	// Health endpoint
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if store != nil {
			if err := store.Ping(r.Context()); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("{\"status\":\"unhealthy\"}"))
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{\"status\":\"ok\"}"))
	})

	// API endpoints (auth-protected)
	mux.Handle("/api/get-balance", Auth(store)(bh))

	// Wrap mux with common middlewares (order: req id -> logger -> cors -> rate)
	return chain(mux, RequestID, Logger, CORS, RateLimit(lm))
}
