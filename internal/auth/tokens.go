package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	issuer         = "auth-api"
	AccessTokenTTL = 15 * time.Minute
)

var ErrInvalidToken = errors.New("invalid token")

type Claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

// UserID returns the subject of the token.
func (c Claims) UserID() string { return c.Subject }

type TokenIssuer struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

func NewTokenIssuer(secret string) *TokenIssuer {
	return &TokenIssuer{secret: []byte(secret), ttl: AccessTokenTTL, now: time.Now}
}

// IssueAccess returns a signed access JWT and its lifetime.
func (t *TokenIssuer) IssueAccess(userID, role string) (string, time.Duration, error) {
	now := t.now()
	claims := Claims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(t.ttl)),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.secret)
	return signed, t.ttl, err
}

// ParseAccess validates signature, algorithm, issuer and expiry.
func (t *TokenIssuer) ParseAccess(raw string) (Claims, error) {
	var claims Claims
	token, err := jwt.ParseWithClaims(raw, &claims,
		func(*jwt.Token) (any, error) { return t.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(issuer),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(t.now),
	)
	if err != nil || !token.Valid || claims.Subject == "" {
		return Claims{}, ErrInvalidToken
	}
	return claims, nil
}
