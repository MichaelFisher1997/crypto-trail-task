package apihttp

import (
	"net/http"

	"github.com/example/solapi/internal/auth"
	"github.com/example/solapi/internal/handlers"
	"github.com/example/solapi/internal/rate"
	"github.com/go-chi/chi/v5"
)

// NewRouter wires routes and middlewares.
func NewRouter(bh *handlers.BalanceHandler, lm *rate.LimiterMap, store auth.APIKeyStore) http.Handler {
	r := chi.NewRouter()

	r.Use(RequestID)
	r.Use(Logger)
	r.Use(CORS)
	r.Use(RateLimit(lm))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if store != nil {
			if err := store.Ping(r.Context()); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("{\"status\":\"unhealthy\"}"))
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{\"status\":\"ok\"}"))
	})

	r.Route("/api", func(api chi.Router) {
		api.Use(Auth(store))
		api.Post("/get-balance", bh.ServeHTTP)
	})

	return r
}
