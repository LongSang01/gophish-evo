package auth

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTSecret is the secret key used to sign JWT tokens
// In production, this should be loaded from config via InitJWTSecret.
// If InitJWTSecret is not called, a random key is generated at init time,
// which invalidates all existing tokens on every server restart.
var JWTSecret = []byte(GenerateSecureKey(32))

// InitJWTSecret initializes the JWT secret from the config file.
// This allows tokens to persist across server restarts and enables
// multi-server deployments. If secret is empty, the random key
// generated at init time is retained.
func InitJWTSecret(secret string) {
	if secret != "" {
		JWTSecret = []byte(secret)
	}
}

// JWTClaims represents the claims in a JWT token
type JWTClaims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// TokenExpiration is the duration for which a JWT token is valid
const TokenExpiration = 24 * time.Hour

// ErrInvalidToken is returned when a JWT token is invalid
var ErrInvalidToken = errors.New("invalid or expired token")

// ErrMissingToken is returned when no JWT token is provided
var ErrMissingToken = errors.New("missing authentication token")

// GenerateToken creates a new JWT token for the given user
func GenerateToken(userID int64, username, role string) (string, error) {
	claims := JWTClaims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TokenExpiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "gophish",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JWTSecret)
}

// ValidateToken validates a JWT token string and returns the claims
func ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return JWTSecret, nil
	})

	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// ExtractTokenFromRequest extracts the JWT token from the Authorization header.
func ExtractTokenFromRequest(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			return parts[1], nil
		}
	}
	return "", ErrMissingToken
}
