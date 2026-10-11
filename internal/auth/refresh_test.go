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
