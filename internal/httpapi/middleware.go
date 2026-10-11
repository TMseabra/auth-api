package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/TMseabra/auth-api/internal/auth"
)

type ctxKey struct{}

func claimsFrom(ctx context.Context) auth.Claims {
	c, _ := ctx.Value(ctxKey{}).(auth.Claims)
	return c
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Bearer`)
	writeError(w, http.StatusUnauthorized, "unauthorized")
}

// requireAuth validates the Bearer access token and stores its claims in the
// request context.
func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		scheme, token, found := strings.Cut(r.Header.Get("Authorization"), " ")
		if !found || !strings.EqualFold(scheme, "Bearer") || token == "" {
			unauthorized(w)
			return
		}
		claims, err := s.tokens.ParseAccess(strings.TrimSpace(token))
		if err != nil {
			unauthorized(w)
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, claims)))
	}
}

// requireRole allows the request only for users with the given role.
func (s *Server) requireRole(role string, next http.HandlerFunc) http.HandlerFunc {
	return s.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		if claimsFrom(r.Context()).Role != role {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}
		next(w, r)
	})
}
