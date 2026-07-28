package platform

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenClaims are the JWT claims issued at staff login.
type TokenClaims struct {
	StaffID  string `json:"sub"`
	Hospital string `json:"hospital"`
}

type jwtClaims struct {
	Hospital string `json:"hospital"`
	jwt.RegisteredClaims
}

// Issue creates a signed HS256 access token.
func Issue(claims TokenClaims, secret string, ttl time.Duration, now time.Time) (string, error) {
	if claims.StaffID == "" || claims.Hospital == "" {
		return "", fmt.Errorf("issue token: missing staff id or hospital")
	}
	if len(secret) == 0 {
		return "", fmt.Errorf("issue token: empty secret")
	}
	if ttl <= 0 {
		return "", fmt.Errorf("issue token: ttl must be positive")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtClaims{
		Hospital: claims.Hospital,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   claims.StaffID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	})
	return token.SignedString([]byte(secret))
}

// Parse verifies an HS256 token and returns its claims. Rejects any other algorithm.
func Parse(tokenString, secret string) (TokenClaims, error) {
	parsed, operationError := jwt.ParseWithClaims(tokenString, &jwtClaims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("%w: unexpected signing method", ErrUnauthorized)
		}
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if operationError != nil {
		return TokenClaims{}, fmt.Errorf("%w: %v", ErrUnauthorized, operationError)
	}

	claims, ok := parsed.Claims.(*jwtClaims)
	if !ok || !parsed.Valid {
		return TokenClaims{}, ErrUnauthorized
	}
	if claims.Subject == "" || claims.Hospital == "" {
		return TokenClaims{}, ErrUnauthorized
	}

	return TokenClaims{
		StaffID:  claims.Subject,
		Hospital: claims.Hospital,
	}, nil
}
