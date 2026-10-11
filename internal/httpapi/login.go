package httpapi

import (
	"errors"
	"net/http"
	"sync"
	"time"

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
	RefreshToken string `json:"refresh_token"`
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

	if !s.loginEmail.allow(email) {
		tooManyRequests(w)
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

	refresh, hash, err := auth.NewRefreshToken()
	if err != nil {
		serverError(w, err)
		return
	}
	if err := s.store.CreateRefreshToken(r.Context(), user.ID, hash, time.Now().Add(auth.RefreshTokenTTL)); err != nil {
		serverError(w, err)
		return
	}
	s.writeTokens(w, user, refresh)
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// refresh exchanges a valid refresh token for a new access token and a new
// refresh token. The presented token is revoked (rotation); presenting a
// revoked token again revokes the whole session family.
func (s *Server) refresh(w http.ResponseWriter, r *http.Request) {
	var in refreshRequest
	if !decodeJSON(w, r, &in) {
		return
	}
	if in.RefreshToken == "" || len(in.RefreshToken) > 256 {
		writeError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	newRaw, newHash, err := auth.NewRefreshToken()
	if err != nil {
		serverError(w, err)
		return
	}
	userID, err := s.store.RotateRefreshToken(r.Context(),
		auth.HashRefreshToken(in.RefreshToken), newHash, time.Now().Add(auth.RefreshTokenTTL))
	if errors.Is(err, store.ErrTokenInvalid) || errors.Is(err, store.ErrTokenReuse) {
		writeError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}

	user, err := s.store.UserByID(r.Context(), userID)
	if err != nil {
		serverError(w, err)
		return
	}
	s.writeTokens(w, user, newRaw)
}

func (s *Server) writeTokens(w http.ResponseWriter, user store.User, refresh string) {
	access, ttl, err := s.tokens.IssueAccess(user.ID, user.Role)
	if err != nil {
		serverError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, tokenResponse{
		AccessToken:  access,
		RefreshToken: refresh,
		TokenType:    "Bearer",
		ExpiresIn:    int(ttl.Seconds()),
	})
}
