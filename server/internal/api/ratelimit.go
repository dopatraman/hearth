package api

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type ipLimiter struct {
	mu       sync.Mutex
	limiters map[string]*limiterEntry
	rate     rate.Limit
	burst    int
	ttl      time.Duration
}

type limiterEntry struct {
	limiter *rate.Limiter
	lastHit time.Time
}

func newIPLimiter(r rate.Limit, burst int) *ipLimiter {
	l := &ipLimiter{
		limiters: make(map[string]*limiterEntry),
		rate:     r,
		burst:    burst,
		ttl:      10 * time.Minute,
	}
	go l.gc()
	return l
}

func (l *ipLimiter) allow(ip string) bool {
	l.mu.Lock()
	entry, ok := l.limiters[ip]
	if !ok {
		entry = &limiterEntry{limiter: rate.NewLimiter(l.rate, l.burst)}
		l.limiters[ip] = entry
	}
	entry.lastHit = time.Now()
	l.mu.Unlock()
	return entry.limiter.Allow()
}

func (l *ipLimiter) gc() {
	t := time.NewTicker(l.ttl)
	defer t.Stop()
	for range t.C {
		cutoff := time.Now().Add(-l.ttl)
		l.mu.Lock()
		for ip, e := range l.limiters {
			if e.lastHit.Before(cutoff) {
				delete(l.limiters, ip)
			}
		}
		l.mu.Unlock()
	}
}

func clientIP(r *http.Request) string {
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

func rateLimit(l *ipLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !l.allow(clientIP(r)) {
				writeError(w, http.StatusTooManyRequests, "too many requests")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
