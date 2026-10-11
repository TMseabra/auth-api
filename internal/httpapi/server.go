package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/TMseabra/auth-api/internal/auth"
	"github.com/TMseabra/auth-api/internal/store"
)

type Server struct {
	store  *store.Store
	tokens *auth.TokenIssuer

	authIP     *limiter // register, login and refresh, per client IP
	loginEmail *limiter // login attempts per account, across all IPs
}

func New(s *store.Store, tokens *auth.TokenIssuer) *Server {
	return &Server{
		store:      s,
		tokens:     tokens,
		authIP:     newLimiter(6*time.Second, 10), // 10 burst, then 10/minute
		loginEmail: newLimiter(30*time.Second, 5), // 5 burst, then 2/minute
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.healthz)
	mux.HandleFunc("POST /auth/register", s.limitByIP(s.authIP, s.register))
	mux.HandleFunc("POST /auth/login", s.limitByIP(s.authIP, s.login))
	mux.HandleFunc("POST /auth/refresh", s.limitByIP(s.authIP, s.refresh))
	mux.HandleFunc("POST /auth/logout", s.requireAuth(s.logout))
	mux.HandleFunc("GET /me", s.requireAuth(s.me))
	mux.HandleFunc("GET /admin/users", s.requireRole("admin", s.listUsers))
	mux.HandleFunc("PATCH /admin/users/{id}/role", s.requireRole("admin", s.setRole))
	return mux
}

func (s *Server) healthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func serverError(w http.ResponseWriter, err error) {
	log.Printf("internal error: %v", err)
	writeError(w, http.StatusInternalServerError, "internal error")
}

const maxBodyBytes = 1 << 20

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}
