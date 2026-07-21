package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pulse/api/internal/config"
)

type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

type Claims struct {
	UserID string `json:"userId"`
	Role   string `json:"role"`
	Type   TokenType `json:"type"`
	jwt.RegisteredClaims
}

// GenerateAccessToken creates a signed 15-minute access token.
func GenerateAccessToken(userID, role string) (string, error) {
	expiry := time.Duration(config.App.JWTAccessExpiry) * time.Minute
	claims := Claims{
		UserID: userID,
		Role:   role,
		Type:   AccessToken,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   userID,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.App.JWTAccessSecret))
}

// GenerateRefreshToken creates a signed 7-day refresh token.
func GenerateRefreshToken(userID, role string) (string, error) {
	expiry := time.Duration(config.App.JWTRefreshExpiry) * 24 * time.Hour
	claims := Claims{
		UserID: userID,
		Role:   role,
		Type:   RefreshToken,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   userID,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.App.JWTRefreshSecret))
}

// ValidateAccessToken parses and validates an access token.
func ValidateAccessToken(tokenString string) (*Claims, error) {
	return parseToken(tokenString, config.App.JWTAccessSecret, AccessToken)
}

// ValidateRefreshToken parses and validates a refresh token.
func ValidateRefreshToken(tokenString string) (*Claims, error) {
	return parseToken(tokenString, config.App.JWTRefreshSecret, RefreshToken)
}

func parseToken(tokenString, secret string, expectedType TokenType) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	if claims.Type != expectedType {
		return nil, errors.New("wrong token type")
	}
	return claims, nil
}
