package auth

import (
	"bytes"
	"testing"
)

func TestRefreshTokenHashing(t *testing.T) {
	raw, hash, err := NewRefreshToken()
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) < 43 {
		t.Fatalf("token too short: %d", len(raw))
	}
	if !bytes.Equal(hash, HashRefreshToken(raw)) {
		t.Fatal("hash does not match raw token")
	}

	raw2, hash2, _ := NewRefreshToken()
	if raw == raw2 || bytes.Equal(hash, hash2) {
		t.Fatal("tokens must be unique")
	}
}

func TestRefreshTokenIsURLSafe(t *testing.T) {
	for i := 0; i < 50; i++ {
		raw, _, err := NewRefreshToken()
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range raw {
			if r == '+' || r == '/' || r == '=' {
				t.Fatalf("token %q contains a non URL-safe character", raw)
			}
		}
	}
}
