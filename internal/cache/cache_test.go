package cache

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCache_GetOrFetch_CacheHitAndError(t *testing.T) {
	c := New(200 * time.Millisecond)
	ctx := context.Background()
	calls := 0
	fetch := func(context.Context) (Value, error) {
		calls++
		return Value{Lamports: 42, FetchedAt: time.Now()}, nil
	}
	// first call -> rpc
	v, src, err := c.GetOrFetch(ctx, "k1", fetch)
	if err != nil || v.Lamports != 42 || src != "rpc" { t.Fatalf("first: v=%v src=%s err=%v", v, src, err) }
	// second call -> cache (no new fetch)
	v2, src2, err := c.GetOrFetch(ctx, "k1", fetch)
	if err != nil || v2.Lamports != 42 || src2 != "cache" { t.Fatalf("second: v=%v src=%s err=%v", v2, src2, err) }
	if calls != 1 { t.Fatalf("fetch calls=%d", calls) }

	// error path: use a different key so it doesn't use existing cache
	badFetch := func(context.Context) (Value, error) { return Value{}, errors.New("fetch-fail") }
	_, src3, err := c.GetOrFetch(ctx, "k2", badFetch)
	if err == nil || src3 != "" { t.Fatalf("expected error, src='%s' err=%v", src3, err) }
}
