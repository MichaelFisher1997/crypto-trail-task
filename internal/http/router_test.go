package apihttp_test

import (
    "context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/example/solapi/internal/cache"
	apihttp "github.com/example/solapi/internal/http"
	"github.com/example/solapi/internal/handlers"
	"github.com/example/solapi/internal/rate"
)

type fakeStorePing struct{ pingErr error }

func (f fakeStorePing) Validate(_ context.Context, _ string) (bool, error) { return true, nil }
func (f fakeStorePing) Ping(_ context.Context) error { return f.pingErr }

// Test /healthz returns ok when Ping() is nil or when store is nil
func TestHealthz_OK(t *testing.T) {
	c := cache.New(10 * time.Second)
	bh := handlers.NewBalanceHandler(handlers.BalanceDeps{Cache: c, Fetcher: nil, Timeout: 3 * time.Second, MaxConcurrency: 16})
	lm := rate.NewLimiterMap(1000, 1000, time.Minute)
	r := apihttp.NewRouter(bh, lm, nil)
	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil { t.Fatalf("request error: %v", err) }
	if resp.StatusCode != http.StatusOK { t.Fatalf("status=%d", resp.StatusCode) }
}

func TestHealthz_StorePingOK(t *testing.T) {
	c := cache.New(10 * time.Second)
	bh := handlers.NewBalanceHandler(handlers.BalanceDeps{Cache: c, Fetcher: nil, Timeout: 3 * time.Second, MaxConcurrency: 16})
	lm := rate.NewLimiterMap(1000, 1000, time.Minute)
	r := apihttp.NewRouter(bh, lm, fakeStorePing{})
	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil { t.Fatalf("request error: %v", err) }
	if resp.StatusCode != http.StatusOK { t.Fatalf("status=%d", resp.StatusCode) }
}

func TestHealthz_StorePingError(t *testing.T) {
	c := cache.New(10 * time.Second)
	bh := handlers.NewBalanceHandler(handlers.BalanceDeps{Cache: c, Fetcher: nil, Timeout: 3 * time.Second, MaxConcurrency: 16})
	lm := rate.NewLimiterMap(1000, 1000, time.Minute)
	r := apihttp.NewRouter(bh, lm, fakeStorePing{pingErr: errString("down")})
	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil { t.Fatalf("request error: %v", err) }
	if resp.StatusCode != http.StatusInternalServerError { t.Fatalf("status=%d", resp.StatusCode) }
}

type errString string
func (e errString) Error() string { return string(e) }
