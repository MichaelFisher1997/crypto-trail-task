package rate

import (
	"net/http"
	"testing"
	"time"
)

func TestLimiter_AllowAndThrottle(t *testing.T) {
	lm := NewLimiterMap(2, 1, 200*time.Millisecond) // 2 req/min, burst 1 (strict)
	defer lm.Stop()
	ip := "1.2.3.4"
	// First call allowed
	if !lm.Allow(ip) { t.Fatalf("first should allow") }
	// Immediate second call likely denied due to burst=1
	if lm.Allow(ip) { t.Fatalf("second should be throttled") }
}

func TestLimiter_ReaperEvictsIdle(t *testing.T) {
	lm := NewLimiterMap(100, 1, 50*time.Millisecond)
	defer lm.Stop()
	ip := "5.6.7.8"
	if !lm.Allow(ip) { t.Fatalf("allow") }
	// Wait beyond TTL so reaper evicts
	time.Sleep(120 * time.Millisecond)
	// Next allow should still succeed, but importantly not panic and create a fresh limiter
	if !lm.Allow(ip) { t.Fatalf("allow after eviction") }
}

func TestIPFromRequest_HeaderAndRemoteAddr(t *testing.T) {
	r, _ := http.NewRequest(http.MethodGet, "http://x/", nil)
	r.Header.Set("X-Forwarded-For", "203.0.113.1, 10.0.0.1")
	if ip := IPFromRequest(r); ip != "203.0.113.1" { t.Fatalf("xff ip=%s", ip) }

	r2, _ := http.NewRequest(http.MethodGet, "http://x/", nil)
	r2.RemoteAddr = "192.0.2.5:1234"
	if ip := IPFromRequest(r2); ip != "192.0.2.5" { t.Fatalf("remote ip=%s", ip) }
}
