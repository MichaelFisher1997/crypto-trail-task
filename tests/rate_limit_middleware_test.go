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

type fakeFetcherRL struct{}

func (f fakeFetcherRL) GetBalance(_ context.Context, _ sol.PublicKey) (uint64, time.Duration, error) { return 0, 0, nil }

func newRouterForRateLimit(t *testing.T, rpm int) http.Handler {
	c := cache.New(10 * time.Second)
	bh := handlers.NewBalanceHandler(handlers.BalanceDeps{Cache: c, Fetcher: fakeFetcherRL{}, Timeout: 3 * time.Second, MaxConcurrency: 16})
	lm := rate.NewLimiterMap(rpm, rpm, time.Minute)
	return apihttp.NewRouter(bh, lm, fakeStore{ok: true})
}

func TestRateLimit429(t *testing.T) {
	r := newRouterForRateLimit(t, 10)
	ts := httptest.NewServer(r)
	defer ts.Close()

	b, _ := json.Marshal(types.GetBalanceRequest{Wallets: []string{"11111111111111111111111111111111"}})
	var got429 int
	for i := 0; i < 11; i++ {
		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/get-balance", bytes.NewReader(b))
		req.Header.Set("X-API-Key", "dev-123")
		resp, err := ts.Client().Do(req)
		if err != nil { t.Fatalf("request error: %v", err) }
		if resp.StatusCode == http.StatusTooManyRequests { got429++ }
		resp.Body.Close()
	}
	if got429 != 1 { t.Fatalf("got429=%d want 1", got429) }
}
