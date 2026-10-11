package httpapi

import (
	"errors"
	"net/http"
	"regexp"

	"github.com/TMseabra/auth-api/internal/store"
)

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	user, err := s.store.UserByID(r.Context(), claimsFrom(r.Context()).UserID())
	if errors.Is(err, store.ErrNotFound) {
		unauthorized(w)
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if err := s.store.RevokeUserTokens(r.Context(), claimsFrom(r.Context()).UserID()); err != nil {
		serverError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.store.ListUsers(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, users)
}

type roleRequest struct {
	Role string `json:"role"`
}

func (s *Server) setRole(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !uuidRe.MatchString(id) {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	var in roleRequest
	if !decodeJSON(w, r, &in) {
		return
	}
	if in.Role != "user" && in.Role != "admin" {
		writeError(w, http.StatusBadRequest, "role must be user or admin")
		return
	}
	if id == claimsFrom(r.Context()).UserID() {
		writeError(w, http.StatusBadRequest, "you cannot change your own role")
		return
	}

	user, err := s.store.SetUserRole(r.Context(), id, in.Role)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, user)
}
