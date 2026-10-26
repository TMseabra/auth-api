package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/TMseabra/auth-api/internal/auth"
	"github.com/TMseabra/auth-api/internal/db"
	"github.com/TMseabra/auth-api/internal/store"
)

const password = "a-long-enough-password"

type env struct {
	t   *testing.T
	srv *httptest.Server
	st  *store.Store
}

// setup returns a server backed by the database in TEST_DATABASE_URL, with
// all data removed. The test is skipped when the variable is not set.
func setup(t *testing.T) *env {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	if err := db.Migrate(url); err != nil {
		t.Fatal(err)
	}
	pool, err := db.Connect(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(context.Background(), `TRUNCATE users CASCADE`); err != nil {
		t.Fatal(err)
	}

	st := store.New(pool)
	srv := httptest.NewServer(New(st, auth.NewTokenIssuer(testSecret)).Routes())
	t.Cleanup(srv.Close)
	return &env{t: t, srv: srv, st: st}
}

func (e *env) do(method, path, token string, body any) (int, map[string]any) {
	e.t.Helper()
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req, _ := http.NewRequest(method, e.srv.URL+path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	defer resp.Body.Close()
	out := map[string]any{}
	json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

func (e *env) register(email string) map[string]any {
	e.t.Helper()
	status, body := e.do("POST", "/auth/register", "", map[string]string{"email": email, "password": password})
	if status != http.StatusCreated {
		e.t.Fatalf("register %s: status %d body %v", email, status, body)
	}
	return body
}

func (e *env) login(email string) map[string]any {
	e.t.Helper()
	status, body := e.do("POST", "/auth/login", "", map[string]string{"email": email, "password": password})
	if status != http.StatusOK {
		e.t.Fatalf("login %s: status %d body %v", email, status, body)
	}
	return body
}

func (e *env) makeAdmin(email string) {
	e.t.Helper()
	u, err := e.st.UserByEmail(context.Background(), email)
	if err != nil {
		e.t.Fatal(err)
	}
	if _, err := e.st.SetUserRole(context.Background(), u.ID, "admin"); err != nil {
		e.t.Fatal(err)
	}
}

func TestRegister(t *testing.T) {
	e := setup(t)

	user := e.register("Alice@Example.com")
	if user["email"] != "alice@example.com" || user["role"] != "user" {
		t.Fatalf("unexpected user: %v", user)
	}
	if _, leaked := user["password_hash"]; leaked {
		t.Fatal("password hash leaked in response")
	}

	if status, _ := e.do("POST", "/auth/register", "", map[string]string{"email": "alice@example.com", "password": password}); status != http.StatusConflict {
		t.Errorf("duplicate email: status %d, want 409", status)
	}
	if status, _ := e.do("POST", "/auth/register", "", map[string]string{"email": "bob@example.com", "password": "short"}); status != http.StatusBadRequest {
		t.Errorf("short password: status %d, want 400", status)
	}
	if status, _ := e.do("POST", "/auth/register", "", map[string]string{"email": "nope", "password": password}); status != http.StatusBadRequest {
		t.Errorf("bad email: status %d, want 400", status)
	}
	if status, _ := e.do("POST", "/auth/register", "", map[string]any{"email": "c@example.com", "password": password, "role": "admin"}); status != http.StatusBadRequest {
		t.Errorf("mass assignment of role: status %d, want 400", status)
	}
}

func TestLoginErrorsAreGeneric(t *testing.T) {
	e := setup(t)
	e.register("alice@example.com")

	s1, b1 := e.do("POST", "/auth/login", "", map[string]string{"email": "alice@example.com", "password": "wrong-password-123"})
	s2, b2 := e.do("POST", "/auth/login", "", map[string]string{"email": "ghost@example.com", "password": "wrong-password-123"})
	if s1 != http.StatusUnauthorized || s2 != http.StatusUnauthorized {
		t.Fatalf("statuses %d, %d; want 401", s1, s2)
	}
	if b1["error"] != b2["error"] {
		t.Fatalf("wrong password and unknown user must look the same: %v vs %v", b1, b2)
	}
}

func TestLoginIsRateLimitedPerAccount(t *testing.T) {
	e := setup(t)
	e.register("alice@example.com")

	bad := map[string]string{"email": "alice@example.com", "password": "wrong-password-123"}
	for i := 0; i < 5; i++ {
		if status, _ := e.do("POST", "/auth/login", "", bad); status != http.StatusUnauthorized {
			t.Fatalf("attempt %d: status %d, want 401", i+1, status)
		}
	}
	if status, _ := e.do("POST", "/auth/login", "", bad); status != http.StatusTooManyRequests {
		t.Errorf("attempt 6: status %d, want 429", status)
	}
	// The correct password is throttled too, otherwise guessing is not slowed down.
	good := map[string]string{"email": "alice@example.com", "password": password}
	if status, _ := e.do("POST", "/auth/login", "", good); status != http.StatusTooManyRequests {
		t.Errorf("correct password while throttled: status %d, want 429", status)
	}
}

func TestMe(t *testing.T) {
	e := setup(t)
	e.register("alice@example.com")
	tokens := e.login("alice@example.com")

	if status, _ := e.do("GET", "/me", "", nil); status != http.StatusUnauthorized {
		t.Errorf("no token: status %d, want 401", status)
	}
	status, me := e.do("GET", "/me", tokens["access_token"].(string), nil)
	if status != http.StatusOK || me["email"] != "alice@example.com" {
		t.Fatalf("me: status %d body %v", status, me)
	}
}

func TestRefreshRotationAndReuseDetection(t *testing.T) {
	e := setup(t)
	e.register("alice@example.com")
	first := e.login("alice@example.com")
	rt1 := first["refresh_token"].(string)

	status, second := e.do("POST", "/auth/refresh", "", map[string]string{"refresh_token": rt1})
	if status != http.StatusOK {
		t.Fatalf("refresh: status %d body %v", status, second)
	}
	rt2 := second["refresh_token"].(string)
	if rt2 == rt1 {
		t.Fatal("refresh token was not rotated")
	}

	// Replaying the old token is rejected and burns the whole family.
	if status, _ := e.do("POST", "/auth/refresh", "", map[string]string{"refresh_token": rt1}); status != http.StatusUnauthorized {
		t.Errorf("reused token: status %d, want 401", status)
	}
	if status, _ := e.do("POST", "/auth/refresh", "", map[string]string{"refresh_token": rt2}); status != http.StatusUnauthorized {
		t.Errorf("token issued before reuse was detected must be revoked: status %d, want 401", status)
	}
	if status, _ := e.do("POST", "/auth/refresh", "", map[string]string{"refresh_token": "garbage"}); status != http.StatusUnauthorized {
		t.Errorf("garbage token: status %d, want 401", status)
	}
}

func TestLogoutRevokesRefreshTokens(t *testing.T) {
	e := setup(t)
	e.register("alice@example.com")
	tokens := e.login("alice@example.com")

	if status, _ := e.do("POST", "/auth/logout", "", nil); status != http.StatusUnauthorized {
		t.Errorf("logout without token: status %d, want 401", status)
	}
	if status, _ := e.do("POST", "/auth/logout", tokens["access_token"].(string), nil); status != http.StatusNoContent {
		t.Fatalf("logout: status %d, want 204", status)
	}
	if status, _ := e.do("POST", "/auth/refresh", "", map[string]string{"refresh_token": tokens["refresh_token"].(string)}); status != http.StatusUnauthorized {
		t.Errorf("refresh after logout: status %d, want 401", status)
	}
}

func TestAdminRoutes(t *testing.T) {
	e := setup(t)
	e.register("admin@example.com")
	bob := e.register("bob@example.com")
	e.makeAdmin("admin@example.com")

	adminToken := e.login("admin@example.com")["access_token"].(string)
	userToken := e.login("bob@example.com")["access_token"].(string)
	rolePath := "/admin/users/" + bob["id"].(string) + "/role"

	// Regular users and anonymous callers are kept out.
	if status, _ := e.do("GET", "/admin/users", userToken, nil); status != http.StatusForbidden {
		t.Errorf("user listing users: status %d, want 403", status)
	}
	if status, _ := e.do("PATCH", rolePath, userToken, map[string]string{"role": "admin"}); status != http.StatusForbidden {
		t.Errorf("user escalating own role: status %d, want 403", status)
	}
	if status, _ := e.do("GET", "/admin/users", "", nil); status != http.StatusUnauthorized {
		t.Errorf("anonymous listing users: status %d, want 401", status)
	}

	// Admins can list and change roles.
	if status, _ := e.do("GET", "/admin/users", adminToken, nil); status != http.StatusOK {
		t.Errorf("admin listing users: status %d, want 200", status)
	}
	status, updated := e.do("PATCH", rolePath, adminToken, map[string]string{"role": "admin"})
	if status != http.StatusOK || updated["role"] != "admin" {
		t.Fatalf("set role: status %d body %v", status, updated)
	}
	if status, _ := e.do("PATCH", rolePath, adminToken, map[string]string{"role": "root"}); status != http.StatusBadRequest {
		t.Errorf("invalid role: status %d, want 400", status)
	}
	if status, _ := e.do("PATCH", "/admin/users/not-a-uuid/role", adminToken, map[string]string{"role": "user"}); status != http.StatusBadRequest {
		t.Errorf("invalid id: status %d, want 400", status)
	}
	if status, _ := e.do("PATCH", "/admin/users/3f2b8c1e-9a4d-4e6b-8f10-1a2b3c4d5e6f/role", adminToken, map[string]string{"role": "user"}); status != http.StatusNotFound {
		t.Errorf("unknown user: status %d, want 404", status)
	}
}

func TestAdminCannotChangeOwnRole(t *testing.T) {
	e := setup(t)
	admin := e.register("admin@example.com")
	e.makeAdmin("admin@example.com")
	token := e.login("admin@example.com")["access_token"].(string)

	status, body := e.do("PATCH", "/admin/users/"+admin["id"].(string)+"/role", token, map[string]string{"role": "user"})
	if status != http.StatusBadRequest {
		t.Fatalf("status %d body %v, want 400", status, body)
	}
}

func TestRefreshRejectsOversizedAndEmptyTokens(t *testing.T) {
	e := setup(t)

	for name, token := range map[string]string{
		"empty":     "",
		"oversized": strings.Repeat("a", 300),
	} {
		if status, _ := e.do("POST", "/auth/refresh", "", map[string]string{"refresh_token": token}); status != http.StatusUnauthorized {
			t.Errorf("%s token: status %d, want 401", name, status)
		}
	}
}
