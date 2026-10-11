package httpapi

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	sweepInterval = time.Minute
	entryIdleTTL  = 10 * time.Minute
)

// limiter is an in-memory token bucket per key. State is per process, so with
// several replicas the effective limit is multiplied by the replica count.
type limiter struct {
	mu        sync.Mutex
	limit     rate.Limit
	burst     int
	entries   map[string]*limiterEntry
	lastSweep time.Time
	now       func() time.Time
}

type limiterEntry struct {
	lim  *rate.Limiter
	seen time.Time
}

func newLimiter(every time.Duration, burst int) *limiter {
	return &limiter{
		limit:   rate.Every(every),
		burst:   burst,
		entries: map[string]*limiterEntry{},
		now:     time.Now,
	}
}

func (l *limiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	if now.Sub(l.lastSweep) > sweepInterval {
		for k, e := range l.entries {
			if now.Sub(e.seen) > entryIdleTTL {
				delete(l.entries, k)
			}
		}
		l.lastSweep = now
	}

	e, ok := l.entries[key]
	if !ok {
		e = &limiterEntry{lim: rate.NewLimiter(l.limit, l.burst)}
		l.entries[key] = e
	}
	e.seen = now
	return e.lim.AllowN(now, 1)
}

func tooManyRequests(w http.ResponseWriter) {
	w.Header().Set("Retry-After", "60")
	writeError(w, http.StatusTooManyRequests, "too many requests, try again later")
}

// clientIP uses the TCP peer address. X-Forwarded-For is deliberately ignored:
// it is client-controlled unless set by a trusted proxy.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// limitByIP throttles the wrapped handler per client IP.
func (s *Server) limitByIP(l *limiter, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !l.allow(clientIP(r)) {
			tooManyRequests(w)
			return
		}
		next(w, r)
	}
}
