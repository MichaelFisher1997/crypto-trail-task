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

type dummyFetcher struct{}

func (dummyFetcher) GetBalance(_ context.Context, _ sol.PublicKey) (uint64, time.Duration, error) { return 0, 0, nil }

func newRouterForValidation() http.Handler {
	c := cache.New(10 * time.Second)
	bh := handlers.NewBalanceHandler(handlers.BalanceDeps{Cache: c, Fetcher: dummyFetcher{}, Timeout: 3 * time.Second, MaxConcurrency: 16})
	lm := rate.NewLimiterMap(1000, 1000, time.Minute)
	return apihttp.NewRouter(bh, lm, fakeStore{ok: true})
}

func TestEmptyWalletsReturns400(t *testing.T) {
	r := newRouterForValidation()
	ts := httptest.NewServer(r)
	defer ts.Close()

	b, _ := json.Marshal(types.GetBalanceRequest{Wallets: []string{}})
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/get-balance", bytes.NewReader(b))
	req.Header.Set("X-API-Key", "dev-123")
	resp, err := ts.Client().Do(req)
	if err != nil { t.Fatalf("request error: %v", err) }
	if resp.StatusCode != http.StatusBadRequest { t.Fatalf("status=%d", resp.StatusCode) }
}

func TestInvalidBase58ProducesErrorsArray200(t *testing.T) {
	r := newRouterForValidation()
	ts := httptest.NewServer(r)
	defer ts.Close()

	b, _ := json.Marshal(types.GetBalanceRequest{Wallets: []string{"not-a-key"}})
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/get-balance", bytes.NewReader(b))
	req.Header.Set("X-API-Key", "dev-123")
	resp, err := ts.Client().Do(req)
	if err != nil { t.Fatalf("request error: %v", err) }
	if resp.StatusCode != http.StatusOK { t.Fatalf("status=%d", resp.StatusCode) }
	var out types.GetBalanceResponse
	_ = json.NewDecoder(resp.Body).Decode(&out)
	resp.Body.Close()
	if len(out.Errors) != 1 { t.Fatalf("errors len=%d", len(out.Errors)) }
}
