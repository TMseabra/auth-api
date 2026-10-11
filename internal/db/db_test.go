package db

import (
	"context"
	"os"
	"testing"
)

func TestMigrate(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	// Running twice must be idempotent.
	for i := 0; i < 2; i++ {
		if err := Migrate(url); err != nil {
			t.Fatal(err)
		}
	}

	pool, err := Connect(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	for _, table := range []string{"users", "refresh_tokens"} {
		var exists bool
		err := pool.QueryRow(context.Background(),
			`SELECT to_regclass('public.' || $1) IS NOT NULL`, table).Scan(&exists)
		if err != nil || !exists {
			t.Fatalf("table %s missing (err=%v)", table, err)
		}
	}
}
