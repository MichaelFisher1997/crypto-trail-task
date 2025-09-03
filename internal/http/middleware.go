package apihttp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"time"

	"github.com/example/solapi/internal/auth"
	"github.com/example/solapi/internal/rate"
	"github.com/example/solapi/pkg/jsonutil"
)

type ctxKey string

const (
	ctxKeyRequestID ctxKey = "req_id"
	ctxKeyAPIKeyHP  ctxKey = "api_key_hp"
)

// RequestID middleware injects a random request id into context and response header.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b [8]byte
		_, _ = rand.Read(b[:])
		reqID := hex.EncodeToString(b[:])
		r = r.WithContext(context.WithValue(r.Context(), ctxKeyRequestID, reqID))
		w.Header().Set("X-Request-ID", reqID)
		next.ServeHTTP(w, r)
	})
}

// Logger middleware logs minimal structured info per request.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rlw := &respLogger{ResponseWriter: w, status: 200}
		next.ServeHTTP(rlw, r)
		reqID, _ := r.Context().Value(ctxKeyRequestID).(string)
		ip := rate.IPFromRequest(r)
		apiHP, _ := r.Context().Value(ctxKeyAPIKeyHP).(string)
		log.Printf("event=request method=%s path=%s status=%d dur_ms=%d ip=%s req_id=%s api=%s", r.Method, r.URL.Path, rlw.status, time.Since(start).Milliseconds(), ip, reqID, apiHP)
	})
}

type respLogger struct{ http.ResponseWriter; status int }

func (r *respLogger) WriteHeader(code int) { r.status = code; r.ResponseWriter.WriteHeader(code) }

// CORS middleware: allows cross-origin requests for demo/testing UI.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RateLimit middleware enforces per-IP rate limiting.
func RateLimit(lm *rate.LimiterMap) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := rate.IPFromRequest(r)
			if !lm.Allow(ip) {
				jsonutil.JSON(w, http.StatusTooManyRequests, map[string]string{"error": "rate limited"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Auth middleware validates the X-API-Key header using the provided store.
func Auth(store auth.APIKeyStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("X-API-Key")
			if key == "" {
				jsonutil.JSON(w, http.StatusUnauthorized, map[string]string{"error": "missing api key"})
				return
			}
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			ok, err := store.Validate(ctx, key)
			if err != nil {
				jsonutil.JSON(w, http.StatusForbidden, map[string]string{"error": "invalid api key"})
				return
			}
			if !ok {
				jsonutil.JSON(w, http.StatusForbidden, map[string]string{"error": "invalid or inactive api key"})
				return
			}
			// store hash prefix in context for logging
			r = r.WithContext(context.WithValue(r.Context(), ctxKeyAPIKeyHP, auth.HashPrefix(key)))
			next.ServeHTTP(w, r)
		})
	}
}
