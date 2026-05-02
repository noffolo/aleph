package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrMissingSubject = errors.New("jwt: missing sub claim")
	ErrInvalidIssuer  = errors.New("jwt: invalid iss claim")
	ErrInvalidAudience = errors.New("jwt: missing aud claim")
	ErrMissingTokenID = errors.New("jwt: missing jti claim")
)

const (
	JWTAudience = "aleph-v2-api"
	JWTIssuer   = "aleph-v2"
	JWTTTL      = 1 * time.Hour
)

type SessionToken struct {
	UserID    string `json:"user_id"`
	ProjectID string `json:"project_id"`
	Role      string `json:"role"`
	Scopes    string `json:"scopes,omitempty"`
	jwt.RegisteredClaims
}

func GenerateToken(claims SessionToken, secret []byte, ttl time.Duration) (string, error) {
	if len(secret) == 0 {
		return "", fmt.Errorf("jwt: secret key is empty")
	}

	if ttl <= 0 {
		ttl = JWTTTL
	}

	now := time.Now()
	jti, err := generateJTI()
	if err != nil {
		return "", fmt.Errorf("jwt: failed to generate jti: %w", err)
	}

	claims.RegisteredClaims = jwt.RegisteredClaims{
		Issuer:    JWTIssuer,
		Subject:   claims.UserID,
		Audience:  jwt.ClaimStrings{JWTAudience},
		ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		NotBefore: jwt.NewNumericDate(now),
		IssuedAt:  jwt.NewNumericDate(now),
		ID:        jti,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(secret)
	if err != nil {
		return "", fmt.Errorf("jwt: sign failed: %w", err)
	}
	return signed, nil
}

func ValidateToken(tokenString string, secret []byte) (*SessionToken, error) {
	if len(secret) == 0 {
		return nil, fmt.Errorf("jwt: secret key is empty")
	}

	claims := &SessionToken{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("jwt: unexpected signing method: %v", t.Header["alg"])
		}
		return secret, nil
	},
		jwt.WithIssuer(JWTIssuer),
		jwt.WithAudience(JWTAudience),
	)
	if err != nil {
		return nil, fmt.Errorf("jwt: validation failed: %w", err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("jwt: token invalid")
	}

	if err := validateClaims(claims); err != nil {
		return nil, err
	}

	return claims, nil
}

func validateClaims(claims *SessionToken) error {
	if claims.Subject == "" {
		return ErrMissingSubject
	}
	if claims.Issuer != JWTIssuer {
		return ErrInvalidIssuer
	}

	validAud := false
	for _, aud := range claims.Audience {
		if aud == JWTAudience {
			validAud = true
			break
		}
	}
	if !validAud {
		return ErrInvalidAudience
	}

	if claims.ID == "" {
		return ErrMissingTokenID
	}

	return nil
}

func generateJTI() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func GenerateJWTSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("jwt secret: rand read failed: %w", err)
	}
	return hex.EncodeToString(b), nil
}