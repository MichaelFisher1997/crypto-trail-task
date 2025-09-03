package main

import (
    "context"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"

    "github.com/example/solapi/internal/cache"
    "github.com/example/solapi/internal/config"
    apihttp "github.com/example/solapi/internal/http"
    "github.com/example/solapi/internal/handlers"
    "github.com/example/solapi/internal/rate"
)

func TestAdd(t *testing.T) {
    if add(2, 3) != 5 { t.Fatalf("bad add") }
}

func TestHelpersAndCrossPackageSmoke(t *testing.T) {
    if sanitizePort("") != "8080" { t.Fatalf("sanitize") }
    if sanitizePort("9090") != "9090" { t.Fatalf("sanitize pass") }
    if sumN(-1) != 0 || sumN(3) != 6 { t.Fatalf("sumN") }
    if chooseCommitment("") != "finalized" || chooseCommitment("processed") != "processed" { t.Fatalf("commit") }

    _ = config.Load()
    c := cache.New(50 * time.Millisecond)
    // use cache quickly
    v, src, err := c.GetOrFetch(context.Background(), "k", func(ctx context.Context) (cache.Value, error) {
        return cache.Value{Lamports: 1, FetchedAt: time.Now()}, nil
    })
    if err != nil || v.Lamports != 1 || src == "" { t.Fatalf("cache path") }

    // build a minimal router to hit /healthz
    bh := handlers.NewBalanceHandler(handlers.BalanceDeps{Cache: c, Fetcher: nil, Timeout: 100 * time.Millisecond, MaxConcurrency: 1})
    lm := rate.NewLimiterMap(100, 1, time.Second)
    defer lm.Stop()
    r := apihttp.NewRouter(bh, lm, nil)
    ts := httptest.NewServer(r)
    defer ts.Close()
    resp, err := http.Get(ts.URL + "/healthz")
    if err != nil { t.Fatalf("health get: %v", err) }
    if resp.StatusCode != http.StatusOK { t.Fatalf("status=%d", resp.StatusCode) }
}
