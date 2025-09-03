package tests

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/example/solapi/internal/cache"
)

func TestCacheSingleflightCoalesces(t *testing.T) {
	c := cache.New(10 * time.Second)
	var mu sync.Mutex
	calls := 0

	fetch := func(ctx context.Context) (cache.Value, error) {
		mu.Lock(); calls++; mu.Unlock()
		time.Sleep(50 * time.Millisecond)
		return cache.Value{Lamports: 1_234, FetchedAt: time.Now()}, nil
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, err := c.GetOrFetch(context.Background(), "k", fetch)
			if err != nil { t.Errorf("err: %v", err) }
		}()
	}
	wg.Wait()
	if calls != 1 { t.Fatalf("fetch calls=%d (want 1)", calls) }
}
