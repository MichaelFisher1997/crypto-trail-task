package rate

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type entry struct {
	limiter *rate.Limiter
	last    time.Time
}

// LimiterMap provides per-IP rate limiting with TTL eviction.
type LimiterMap struct {
	mu       sync.Mutex
	limiters map[string]*entry
	rpm      int
	burst    int
	ttl      time.Duration
	stopCh   chan struct{}
}

// NewLimiterMap creates a LimiterMap with cleanup goroutine.
func NewLimiterMap(rpm, burst int, ttl time.Duration) *LimiterMap {
	lm := &LimiterMap{
		limiters: make(map[string]*entry),
		rpm:      rpm,
		burst:    burst,
		ttl:      ttl,
		stopCh:   make(chan struct{}),
	}
	go lm.reaper()
	return lm
}

func (l *LimiterMap) reaper() {
	t := time.NewTicker(l.ttl)
	defer t.Stop()
	for {
		select {
		case <-l.stopCh:
			return
		case now := <-t.C:
			l.mu.Lock()
			for ip, e := range l.limiters {
				if now.Sub(e.last) > l.ttl {
					delete(l.limiters, ip)
				}
			}
			l.mu.Unlock()
		}
	}
}

// Stop stops the cleanup goroutine.
func (l *LimiterMap) Stop() { close(l.stopCh) }

func (l *LimiterMap) get(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()
	if e, ok := l.limiters[ip]; ok {
		e.last = time.Now()
		return e.limiter
	}
	lim := rate.NewLimiter(rate.Every(time.Minute/time.Duration(l.rpm)), l.burst)
	l.limiters[ip] = &entry{limiter: lim, last: time.Now()}
	return lim
}

// Allow returns true if the request from given IP should be allowed.
func (l *LimiterMap) Allow(ip string) bool {
	return l.get(ip).Allow()
}

// IPFromRequest extracts the client IP from the request.
func IPFromRequest(r *http.Request) string {
	// Prefer X-Forwarded-For first element if present
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// take first value
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
