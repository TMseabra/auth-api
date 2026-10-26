package store

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/TMseabra/auth-api/internal/db"
)

// setup returns a store backed by TEST_DATABASE_URL with all data removed.
// The test is skipped when the variable is not set.
func setup(t *testing.T) *Store {
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
	return New(pool)
}

func TestCreateUserEmailIsUniqueIgnoringCase(t *testing.T) {
	s := setup(t)
	ctx := context.Background()

	if _, err := s.CreateUser(ctx, "a@example.com", "hash"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateUser(ctx, "A@Example.COM", "hash"); !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("err = %v, want ErrEmailTaken", err)
	}
}

func TestListUsersReturnsEveryUser(t *testing.T) {
	s := setup(t)
	ctx := context.Background()

	for _, email := range []string{"a@example.com", "b@example.com", "c@example.com"} {
		if _, err := s.CreateUser(ctx, email, "hash"); err != nil {
			t.Fatal(err)
		}
	}
	users, err := s.ListUsers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 3 {
		t.Fatalf("got %d users, want 3", len(users))
	}
	for _, u := range users {
		if u.Role != "user" {
			t.Errorf("%s has role %q, want the default role", u.Email, u.Role)
		}
	}
}

func TestRotateRefreshTokenRejectsExpiredToken(t *testing.T) {
	s := setup(t)
	ctx := context.Background()
	user, err := s.CreateUser(ctx, "a@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}

	old := []byte("old-token-hash-32-bytes-long-000")
	if err := s.CreateRefreshToken(ctx, user.ID, old, time.Now().Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	_, err = s.RotateRefreshToken(ctx, old, []byte("new-token-hash-32-bytes-long-000"), time.Now().Add(time.Hour))
	if !errors.Is(err, ErrTokenInvalid) {
		t.Fatalf("err = %v, want ErrTokenInvalid", err)
	}
}
