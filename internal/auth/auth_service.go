package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {}

func NewAuthService() *AuthService { return &AuthService{} }

func (s *AuthService) ValidateAPIKey(apiKey string) (bool, string, error) {
	// Implementazione mock: valida solo se inizia con "aleph_"
	if len(apiKey) > 6 { return true, "default-project-id", nil }
	return false, "", nil
}
