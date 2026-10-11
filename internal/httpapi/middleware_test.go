package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TMseabra/auth-api/internal/auth"
)

const testSecret = "0123456789abcdef0123456789abcdef"

func newTestServer() (*Server, *auth.TokenIssuer) {
	iss := auth.NewTokenIssuer(testSecret)
	return New(nil, iss), iss
}

func call(h http.Handler, authz string) int {
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	if authz != "" {
		req.Header.Set("Authorization", authz)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code
}

func TestRequireAuth(t *testing.T) {
	s, iss := newTestServer()
	h := s.requireAuth(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	token, _, _ := iss.IssueAccess("u1", "user")

	cases := map[string]int{
		"":                      http.StatusUnauthorized,
		"Bearer":                http.StatusUnauthorized,
		"Basic abc":             http.StatusUnauthorized,
		"Bearer not-a-jwt":      http.StatusUnauthorized,
		"Bearer " + token + "x": http.StatusUnauthorized,
		"Bearer " + token:       http.StatusOK,
		"bearer " + token:       http.StatusOK,
	}
	for authz, want := range cases {
		if got := call(h, authz); got != want {
			t.Errorf("Authorization %q: status %d, want %d", authz, got, want)
		}
	}
}

func TestRequireRole(t *testing.T) {
	s, iss := newTestServer()
	h := s.requireRole("admin", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	user, _, _ := iss.IssueAccess("u1", "user")
	admin, _, _ := iss.IssueAccess("u2", "admin")

	if got := call(h, "Bearer "+user); got != http.StatusForbidden {
		t.Errorf("user token: status %d, want 403", got)
	}
	if got := call(h, "Bearer "+admin); got != http.StatusOK {
		t.Errorf("admin token: status %d, want 200", got)
	}
	if got := call(h, ""); got != http.StatusUnauthorized {
		t.Errorf("no token: status %d, want 401", got)
	}
}

func TestUUIDValidation(t *testing.T) {
	if !uuidRe.MatchString("3f2b8c1e-9a4d-4e6b-8f10-1a2b3c4d5e6f") {
		t.Error("valid uuid rejected")
	}
	for _, bad := range []string{"", "1", "../etc/passwd", "3f2b8c1e-9a4d-4e6b-8f10-1a2b3c4d5e6f; DROP"} {
		if uuidRe.MatchString(bad) {
			t.Errorf("%q accepted", bad)
		}
	}
}
