package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"time"
)

const RefreshTokenTTL = 7 * 24 * time.Hour

// NewRefreshToken returns a random opaque token and the SHA-256 hash that is
// stored in the database. The raw token is never persisted.
func NewRefreshToken() (raw string, hash []byte, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", nil, err
	}
	raw = base64.RawURLEncoding.EncodeToString(b)
	return raw, HashRefreshToken(raw), nil
}

func HashRefreshToken(raw string) []byte {
	sum := sha256.Sum256([]byte(raw))
	return sum[:]
}
