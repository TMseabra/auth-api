package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLimiterBlocksAfterBurstAndRefills(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	l := newLimiter(time.Minute, 3)
	l.now = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		if !l.allow("a") {
			t.Fatalf("request %d blocked within burst", i+1)
		}
	}
	if l.allow("a") {
		t.Fatal("request over burst allowed")
	}
	if !l.allow("b") {
		t.Fatal("limit must be tracked per key")
	}

	now = now.Add(time.Minute)
	if !l.allow("a") {
		t.Fatal("token not refilled after interval")
	}
}

func TestLimiterSweepsIdleEntries(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	l := newLimiter(time.Minute, 1)
	l.now = func() time.Time { return now }

	l.allow("old")
	now = now.Add(entryIdleTTL + 2*sweepInterval)
	l.allow("new")

	if _, ok := l.entries["old"]; ok {
		t.Fatal("idle entry was not removed")
	}
}

func TestLimitByIPReturns429WithRetryAfter(t *testing.T) {
	s, _ := newTestServer()
	l := newLimiter(time.Hour, 2)
	h := s.limitByIP(l, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	do := func(remote, xff string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
		req.RemoteAddr = remote
		if xff != "" {
			req.Header.Set("X-Forwarded-For", xff)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	do("10.0.0.1:1111", "")
	do("10.0.0.1:2222", "")
	rec := do("10.0.0.1:3333", "")
	if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") == "" {
		t.Fatalf("status %d, Retry-After %q", rec.Code, rec.Header().Get("Retry-After"))
	}

	// A spoofed X-Forwarded-For must not give the client a fresh bucket.
	if rec := do("10.0.0.1:4444", "203.0.113.9"); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("X-Forwarded-For bypassed the limit: status %d", rec.Code)
	}
	if rec := do("10.0.0.2:1111", ""); rec.Code != http.StatusOK {
		t.Fatalf("other IP blocked: status %d", rec.Code)
	}
}
