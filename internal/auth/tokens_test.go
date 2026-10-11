package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "0123456789abcdef0123456789abcdef"

func TestAccessTokenRoundTrip(t *testing.T) {
	iss := NewTokenIssuer(testSecret)
	raw, ttl, err := iss.IssueAccess("user-1", "admin")
	if err != nil {
		t.Fatal(err)
	}
	if ttl != 15*time.Minute {
		t.Fatalf("ttl = %v", ttl)
	}
	claims, err := iss.ParseAccess(raw)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID() != "user-1" || claims.Role != "admin" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestAccessTokenExpires(t *testing.T) {
	iss := NewTokenIssuer(testSecret)
	raw, _, _ := iss.IssueAccess("u", "user")
	iss.now = func() time.Time { return time.Now().Add(16 * time.Minute) }
	if _, err := iss.ParseAccess(raw); err == nil {
		t.Fatal("expired token accepted")
	}
}

func TestAccessTokenRejectsTampering(t *testing.T) {
	iss := NewTokenIssuer(testSecret)
	raw, _, _ := iss.IssueAccess("u", "user")

	other := NewTokenIssuer("another-secret-another-secret-12345")
	if _, err := other.ParseAccess(raw); err == nil {
		t.Fatal("token signed with a different secret accepted")
	}

	parts := strings.Split(raw, ".")
	if _, err := iss.ParseAccess(parts[0] + "." + parts[1] + ".AAAA"); err == nil {
		t.Fatal("token with bad signature accepted")
	}
}

func TestAccessTokenRejectsNoneAlgorithm(t *testing.T) {
	iss := NewTokenIssuer(testSecret)
	unsigned := jwt.NewWithClaims(jwt.SigningMethodNone, Claims{
		Role: "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   "u",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})
	raw, err := unsigned.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := iss.ParseAccess(raw); err == nil {
		t.Fatal("alg=none token accepted")
	}
}
