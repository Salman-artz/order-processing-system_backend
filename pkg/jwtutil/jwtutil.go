// Package jwtutil provides shared JWT token generation and validation helpers.
// All services that need to verify customer/admin identity use this package.
package jwtutil

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Role represents user role in the system.
type Role string

const (
	RoleCustomer Role = "customer"
	RoleAdmin    Role = "admin"
)

// Claims is the custom JWT claims structure.
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   Role   `json:"role"`
	jwt.RegisteredClaims
}

// Config holds JWT configuration.
type Config struct {
	// Secret is the HMAC signing key. Load from JWT_SECRET env var.
	Secret string
	// TokenTTL is how long an access token is valid. Default: 24h.
	TokenTTL time.Duration
}

func (c *Config) applyDefaults() {
	if c.TokenTTL == 0 {
		c.TokenTTL = 24 * time.Hour
	}
}

// GenerateToken creates and signs a new JWT for the given user.
func GenerateToken(cfg Config, userID, email string, role Role) (string, error) {
	cfg.applyDefaults()

	if cfg.Secret == "" {
		return "", errors.New("jwtutil: JWT_SECRET is empty")
	}

	claims := Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(cfg.TokenTTL)),
			Issuer:    "order-processing-system",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(cfg.Secret))
	if err != nil {
		return "", fmt.Errorf("jwtutil: sign token: %w", err)
	}

	return signed, nil
}

// ValidateToken parses and validates a JWT string.
// Returns the claims if valid, or an error if invalid/expired.
func ValidateToken(cfg Config, tokenStr string) (*Claims, error) {
	if cfg.Secret == "" {
		return nil, errors.New("jwtutil: JWT_SECRET is empty")
	}

	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("jwtutil: unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(cfg.Secret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("jwtutil: parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("jwtutil: invalid token claims")
	}

	return claims, nil
}
