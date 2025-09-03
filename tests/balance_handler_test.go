package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/example/solapi/internal/cache"
	apihttp "github.com/example/solapi/internal/http"
	"github.com/example/solapi/internal/handlers"
	"github.com/example/solapi/internal/rate"
	"github.com/example/solapi/internal/types"
	sol "github.com/gagliardetto/solana-go"
)

type fakeStore struct{ ok bool }

func (f fakeStore) Validate(_ context.Context, _ string) (bool, error) { return f.ok, nil }
func (f fakeStore) Ping(_ context.Context) error { return nil }

type fakeFetcher struct {
	mu       sync.Mutex
	calls    int
	lamports uint64
	delay    time.Duration
}

func (f *fakeFetcher) GetBalance(_ context.Context, _ sol.PublicKey) (uint64, time.Duration, error) {
	f.mu.Lock()
	f.calls++
	f.mu.Unlock()
	if f.delay > 0 {
		time.Sleep(f.delay)
	}
	return f.lamports, 5 * time.Millisecond, nil
}

func newTestServer(t *testing.T, lamports uint64, delay time.Duration) (*httptest.Server, *fakeFetcher) {
	t.Helper()
	c := cache.New(10 * time.Second)
	ff := &fakeFetcher{lamports: lamports, delay: delay}
	bh := handlers.NewBalanceHandler(handlers.BalanceDeps{
		Cache:          c,
		Fetcher:        ff,
		Timeout:        3 * time.Second,
		MaxConcurrency: 16,
	})
	lm := rate.NewLimiterMap(1000, 1000, time.Minute) // large to avoid test flakiness here
	r := apihttp.NewRouter(bh, lm, fakeStore{ok: true})
	return httptest.NewServer(r), ff
}

func doPost(t *testing.T, ts *httptest.Server, wallets []string, key string) (*http.Response, types.GetBalanceResponse) {
	b, _ := json.Marshal(types.GetBalanceRequest{Wallets: wallets})
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/get-balance", bytes.NewReader(b))
	if key != "" { req.Header.Set("X-API-Key", key) }
	resp, err := ts.Client().Do(req)
	if err != nil { t.Fatalf("request error: %v", err) }
	var out types.GetBalanceResponse
	_ = json.NewDecoder(resp.Body).Decode(&out)
	resp.Body.Close()
	return resp, out
}

func TestSingleWalletSuccess(t *testing.T) {
	ts, ff := newTestServer(t, 2_000_000_000, 0)
	defer ts.Close()
	resp, out := doPost(t, ts, []string{"11111111111111111111111111111111"}, "dev-123")
	if resp.StatusCode != http.StatusOK { t.Fatalf("status=%d", resp.StatusCode) }
	if len(out.Balances) != 1 { t.Fatalf("balances len=%d", len(out.Balances)) }
	if out.Balances[0].Lamports != 2_000_000_000 { t.Fatalf("lamports=%d", out.Balances[0].Lamports) }
	if out.Balances[0].Sol != 2.0 { t.Fatalf("sol=%f", out.Balances[0].Sol) }
	if out.Balances[0].Source != "rpc" { t.Fatalf("source=%s", out.Balances[0].Source) }
	ff.mu.Lock(); calls := ff.calls; ff.mu.Unlock()
	if calls != 1 { t.Fatalf("fetch calls=%d", calls) }
}

func TestMultipleWalletsDeduped(t *testing.T) {
	ts, ff := newTestServer(t, 1_000_000_000, 0)
	defer ts.Close()
	ws := []string{"11111111111111111111111111111111", "11111111111111111111111111111111"}
	resp, out := doPost(t, ts, ws, "dev-123")
	if resp.StatusCode != http.StatusOK { t.Fatalf("status=%d", resp.StatusCode) }
	if len(out.Balances) != 1 { t.Fatalf("balances len=%d", len(out.Balances)) }
	ff.mu.Lock(); calls := ff.calls; ff.mu.Unlock()
	if calls != 1 { t.Fatalf("fetch calls=%d", calls) }
}

func TestCoalescingConcurrentRequests(t *testing.T) {
	ts, ff := newTestServer(t, 1_000_000_000, 50*time.Millisecond)
	defer ts.Close()
	var wg sync.WaitGroup
	type result struct{ status int; balancesLen int; source string }
	resCh := make(chan result, 5)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, out := doPost(t, ts, []string{"11111111111111111111111111111111"}, "dev-123")
			src := ""
			if len(out.Balances) > 0 { src = out.Balances[0].Source }
			resCh <- result{status: resp.StatusCode, balancesLen: len(out.Balances), source: src}
		}()
	}
	wg.Wait()
	close(resCh)
	for r := range resCh {
		if r.status != http.StatusOK { t.Fatalf("status=%d", r.status) }
		if r.balancesLen != 1 { t.Fatalf("balances len=%d", r.balancesLen) }
		if r.source != "rpc" && r.source != "cache" { t.Fatalf("unexpected source %s", r.source) }
	}
	ff.mu.Lock(); calls := ff.calls; ff.mu.Unlock()
	if calls != 1 { t.Fatalf("fetch calls=%d (want 1)", calls) }
}

func TestCachingSecondCallHitsCache(t *testing.T) {
	ts, ff := newTestServer(t, 1_000_000_000, 0)
	defer ts.Close()
	resp1, out1 := doPost(t, ts, []string{"11111111111111111111111111111111"}, "dev-123")
	if resp1.StatusCode != http.StatusOK { t.Fatalf("status1=%d", resp1.StatusCode) }
	if out1.Balances[0].Source != "rpc" { t.Fatalf("src1=%s", out1.Balances[0].Source) }
	resp2, out2 := doPost(t, ts, []string{"11111111111111111111111111111111"}, "dev-123")
	if resp2.StatusCode != http.StatusOK { t.Fatalf("status2=%d", resp2.StatusCode) }
	if out2.Balances[0].Source != "cache" { t.Fatalf("src2=%s", out2.Balances[0].Source) }
	ff.mu.Lock(); calls := ff.calls; ff.mu.Unlock()
	if calls != 1 { t.Fatalf("fetch calls=%d (want 1)", calls) }
}
