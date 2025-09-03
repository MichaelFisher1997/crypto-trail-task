package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/example/solapi/internal/cache"
	apihttp "github.com/example/solapi/internal/http"
	"github.com/example/solapi/internal/handlers"
	"github.com/example/solapi/internal/rate"
	"github.com/example/solapi/internal/types"
	sol "github.com/gagliardetto/solana-go"
)

type noOpFetcher struct{}

func (noOpFetcher) GetBalance(_ context.Context, _ sol.PublicKey) (uint64, time.Duration, error) { return 0, 0, nil }

func routerWithAuth(ok bool) http.Handler {
	c := cache.New(10 * time.Second)
	bh := handlers.NewBalanceHandler(handlers.BalanceDeps{Cache: c, Fetcher: noOpFetcher{}, Timeout: 3 * time.Second, MaxConcurrency: 16})
	lm := rate.NewLimiterMap(1000, 1000, time.Minute)
	return apihttp.NewRouter(bh, lm, fakeStore{ok: ok})
}

func TestAuthMissingKey401(t *testing.T) {
	r := routerWithAuth(true)
	ts := httptest.NewServer(r)
	defer ts.Close()

	b, _ := json.Marshal(types.GetBalanceRequest{Wallets: []string{"11111111111111111111111111111111"}})
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/get-balance", bytes.NewReader(b))
	resp, err := ts.Client().Do(req)
	if err != nil { t.Fatalf("request error: %v", err) }
	if resp.StatusCode != http.StatusUnauthorized { t.Fatalf("status=%d", resp.StatusCode) }
}

func TestAuthInvalidKey403(t *testing.T) {
	r := routerWithAuth(false)
	ts := httptest.NewServer(r)
	defer ts.Close()

	b, _ := json.Marshal(types.GetBalanceRequest{Wallets: []string{"11111111111111111111111111111111"}})
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/get-balance", bytes.NewReader(b))
	req.Header.Set("X-API-Key", "bad-key")
	resp, err := ts.Client().Do(req)
	if err != nil { t.Fatalf("request error: %v", err) }
	if resp.StatusCode != http.StatusForbidden { t.Fatalf("status=%d", resp.StatusCode) }
}

func TestAuthValidKey200(t *testing.T) {
	r := routerWithAuth(true)
	ts := httptest.NewServer(r)
	defer ts.Close()

	b, _ := json.Marshal(types.GetBalanceRequest{Wallets: []string{"11111111111111111111111111111111"}})
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/get-balance", bytes.NewReader(b))
	req.Header.Set("X-API-Key", "dev-123")
	resp, err := ts.Client().Do(req)
	if err != nil { t.Fatalf("request error: %v", err) }
	if resp.StatusCode != http.StatusOK { t.Fatalf("status=%d", resp.StatusCode) }
}
