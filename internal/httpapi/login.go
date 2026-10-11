package httpapi

import (
	"errors"
	"net/http"
	"sync"

	"github.com/TMseabra/auth-api/internal/auth"
	"github.com/TMseabra/auth-api/internal/store"
)

const invalidCredentials = "invalid credentials"

var (
	dummyHashOnce sync.Once
	dummyHash     string
)

// burnPasswordHash spends the same time as a real verification so that a
// login attempt for an unknown email is not distinguishable by latency.
func burnPasswordHash(password string) {
	dummyHashOnce.Do(func() { dummyHash, _ = auth.HashPassword("dummy-password-for-timing") })
	auth.VerifyPassword(password, dummyHash)
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var in credentials
	if !decodeJSON(w, r, &in) {
		return
	}

	email, ok := normalizeEmail(in.Email)
	if !ok || in.Password == "" || len(in.Password) > 4*maxPasswordLen {
		burnPasswordHash(in.Password)
		writeError(w, http.StatusUnauthorized, invalidCredentials)
		return
	}

	user, err := s.store.UserByEmail(r.Context(), email)
	if errors.Is(err, store.ErrNotFound) {
		burnPasswordHash(in.Password)
		writeError(w, http.StatusUnauthorized, invalidCredentials)
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}

	match, err := auth.VerifyPassword(in.Password, user.PasswordHash)
	if err != nil {
		serverError(w, err)
		return
	}
	if !match {
		writeError(w, http.StatusUnauthorized, invalidCredentials)
		return
	}

	s.issueTokens(w, r, user)
}

func (s *Server) issueTokens(w http.ResponseWriter, r *http.Request, user store.User) {
	access, ttl, err := s.tokens.IssueAccess(user.ID, user.Role)
	if err != nil {
		serverError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, tokenResponse{
		AccessToken: access,
		TokenType:   "Bearer",
		ExpiresIn:   int(ttl.Seconds()),
	})
}
