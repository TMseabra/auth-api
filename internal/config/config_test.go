package config

import "testing"

const goodSecret = "0123456789abcdef0123456789abcdef"

func TestLoad(t *testing.T) {
	t.Run("defaults port", func(t *testing.T) {
		t.Setenv("PORT", "")
		t.Setenv("DATABASE_URL", "postgres://x")
		t.Setenv("JWT_SECRET", goodSecret)

		cfg, err := Load()
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Port != "8080" {
			t.Fatalf("port = %q, want 8080", cfg.Port)
		}
	})

	t.Run("missing database url", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "")
		t.Setenv("JWT_SECRET", goodSecret)
		if _, err := Load(); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("short jwt secret", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgres://x")
		t.Setenv("JWT_SECRET", "short")
		if _, err := Load(); err == nil {
			t.Fatal("expected error")
		}
	})
}
