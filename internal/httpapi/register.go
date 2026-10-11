package httpapi

import (
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"unicode/utf8"

	"github.com/TMseabra/auth-api/internal/auth"
	"github.com/TMseabra/auth-api/internal/store"
)

const (
	minPasswordLen = 12
	maxPasswordLen = 128
	maxEmailLen    = 254
)

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// normalizeEmail validates and canonicalizes an email address.
func normalizeEmail(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || len(raw) > maxEmailLen {
		return "", false
	}
	addr, err := mail.ParseAddress(raw)
	if err != nil || addr.Address != raw {
		return "", false
	}
	return strings.ToLower(raw), true
}

func validatePassword(p string) string {
	n := utf8.RuneCountInString(p)
	switch {
	case n < minPasswordLen:
		return "password must be at least 12 characters"
	case n > maxPasswordLen:
		return "password must be at most 128 characters"
	}
	return ""
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var in credentials
	if !decodeJSON(w, r, &in) {
		return
	}

	email, ok := normalizeEmail(in.Email)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid email")
		return
	}
	if msg := validatePassword(in.Password); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		serverError(w, err)
		return
	}

	user, err := s.store.CreateUser(r.Context(), email, hash)
	if errors.Is(err, store.ErrEmailTaken) {
		writeError(w, http.StatusConflict, "email already registered")
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, user)
}
