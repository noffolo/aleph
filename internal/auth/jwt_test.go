package auth

import (
	"testing"
	"time"
)

func TestGenerateAndValidateToken(t *testing.T) {
	secret := []byte("test-secret-key-32bytes!!!!!")
	claims := SessionToken{
		UserID:    "user-1",
		ProjectID: "proj-abc",
		Role:      "admin",
		Scopes:    "read,write",
	}

	token, err := GenerateToken(claims, secret, JWTTTL)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	validated, err := ValidateToken(token, secret)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if validated.UserID != claims.UserID {
		t.Errorf("UserID: got %q, want %q", validated.UserID, claims.UserID)
	}
	if validated.ProjectID != claims.ProjectID {
		t.Errorf("ProjectID: got %q, want %q", validated.ProjectID, claims.ProjectID)
	}
	if validated.Role != claims.Role {
		t.Errorf("Role: got %q, want %q", validated.Role, claims.Role)
	}
	if validated.Scopes != claims.Scopes {
		t.Errorf("Scopes: got %q, want %q", validated.Scopes, claims.Scopes)
	}
	if validated.Issuer != JWTIssuer {
		t.Errorf("Issuer: got %q, want %q", validated.Issuer, JWTIssuer)
	}
	if validated.Subject != claims.UserID {
		t.Errorf("Subject: got %q, want %q", validated.Subject, claims.UserID)
	}
	if validated.ID == "" {
		t.Error("expected non-empty jti")
	}
	if len(validated.Audience) != 1 || validated.Audience[0] != JWTAudience {
		t.Errorf("Audience: got %v, want [%q]", validated.Audience, JWTAudience)
	}
}

func TestValidateTokenMissingSub(t *testing.T) {
	secret := []byte("test-secret-key-32bytes!!!!!")
	claims := SessionToken{
		UserID:    "",
		ProjectID: "proj-abc",
		Role:      "user",
	}

	token, err := GenerateToken(claims, secret, 1*time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	_, err = ValidateToken(token, secret)
	if err == nil {
		t.Error("expected error for missing sub claim")
	}
}

func TestValidateTokenExpired(t *testing.T) {
	secret := []byte("test-secret-key-32bytes!!!!!")
	claims := SessionToken{
		UserID:    "user-1",
		ProjectID: "proj-abc",
		Role:      "user",
	}

	token, err := GenerateToken(claims, secret, 1*time.Millisecond)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	time.Sleep(5 * time.Millisecond)

	_, err = ValidateToken(token, secret)
	if err == nil {
		t.Error("expected error for expired token")
	}
}

func TestValidateTokenWrongSecret(t *testing.T) {
	secret := []byte("test-secret-key-32bytes!!!!!")
	token, err := GenerateToken(SessionToken{
		UserID:    "u1",
		ProjectID: "p1",
		Role:      "user",
	}, secret, 1*time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	_, err = ValidateToken(token, []byte("wrong-secret-key-32bytes!!!"))
	if err == nil {
		t.Error("expected error for wrong secret")
	}
}

func TestGenerateTokenEmptySecret(t *testing.T) {
	_, err := GenerateToken(SessionToken{}, nil, 1*time.Hour)
	if err == nil {
		t.Error("expected error for empty secret")
	}

	_, err = GenerateToken(SessionToken{}, []byte{}, 1*time.Hour)
	if err == nil {
		t.Error("expected error for empty secret")
	}
}

func TestValidateTokenEmptySecret(t *testing.T) {
	_, err := ValidateToken("any.token.string", nil)
	if err == nil {
		t.Error("expected error for empty secret")
	}
}

func TestGenerateJWTSecret(t *testing.T) {
	secret, err := GenerateJWTSecret()
	if err != nil {
		t.Fatalf("GenerateJWTSecret: %v", err)
	}
	if len(secret) != 64 {
		t.Errorf("expected 64 hex characters, got %d", len(secret))
	}

	secret2, err := GenerateJWTSecret()
	if err != nil {
		t.Fatalf("GenerateJWTSecret second call: %v", err)
	}
	if secret == secret2 {
		t.Error("expected different secrets on successive calls")
	}
}

func TestValidateTokenMalformed(t *testing.T) {
	secret := []byte("test-secret-key-32bytes!!!!!")
	_, err := ValidateToken("not.a.jwt", secret)
	if err == nil {
		t.Error("expected error for malformed token")
	}
}

func TestValidateTokenJTIUniqueness(t *testing.T) {
	secret := []byte("test-secret-key-32bytes!!!!!")
	claims := SessionToken{
		UserID:    "u1",
		ProjectID: "p1",
		Role:      "admin",
	}

	token1, err := GenerateToken(claims, secret, 1*time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken 1: %v", err)
	}
	token2, err := GenerateToken(claims, secret, 1*time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken 2: %v", err)
	}

	v1, err := ValidateToken(token1, secret)
	if err != nil {
		t.Fatalf("ValidateToken 1: %v", err)
	}
	v2, err := ValidateToken(token2, secret)
	if err != nil {
		t.Fatalf("ValidateToken 2: %v", err)
	}

	if v1.ID == v2.ID {
		t.Error("expected different JTI for different token generations")
	}
}

func TestValidateTokenAudienceValidation(t *testing.T) {
	secret := []byte("test-secret-key-32bytes!!!!!")
	claims := SessionToken{
		UserID:    "u1",
		ProjectID: "p1",
		Role:      "user",
	}

	token, err := GenerateToken(claims, secret, 1*time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	validated, err := ValidateToken(token, secret)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}

	if len(validated.Audience) == 0 || validated.Audience[0] != JWTAudience {
		t.Errorf("expected audience %q, got %v", JWTAudience, validated.Audience)
	}
}

func TestValidateTokenIssuerValidation(t *testing.T) {
	secret := []byte("test-secret-key-32bytes!!!!!")
	claims := SessionToken{
		UserID:    "u1",
		ProjectID: "p1",
		Role:      "user",
	}

	token, err := GenerateToken(claims, secret, 1*time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	validated, err := ValidateToken(token, secret)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}

	if validated.Issuer != JWTIssuer {
		t.Errorf("expected issuer %q, got %q", JWTIssuer, validated.Issuer)
	}
}

func TestConstants(t *testing.T) {
	if JWTAudience != "aleph-v2-api" {
		t.Errorf("JWTAudience: got %q, want %q", JWTAudience, "aleph-v2-api")
	}
	if JWTIssuer != "aleph-v2" {
		t.Errorf("JWTIssuer: got %q, want %q", JWTIssuer, "aleph-v2")
	}
	if JWTTTL != 1*time.Hour {
		t.Errorf("JWTTTL: got %v, want %v", JWTTTL, 1*time.Hour)
	}
}

func TestValidateTokenMissingJTIClaim(t *testing.T) {
	secret := []byte("test-secret-key-32bytes!!!!!")
	token, err := GenerateToken(SessionToken{
		UserID:    "u1",
		ProjectID: "p1",
		Role:      "user",
	}, secret, 1*time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	validated, err := ValidateToken(token, secret)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if validated.ID == "" {
		t.Error("expected non-empty JTI in validated token")
	}
}

func TestValidateTokenDefaultTTL(t *testing.T) {
	secret := []byte("test-secret-key-32bytes!!!!!")
	claims := SessionToken{
		UserID:    "u1",
		ProjectID: "p1",
		Role:      "user",
	}
	token, err := GenerateToken(claims, secret, 0)
	if err != nil {
		t.Fatalf("GenerateToken with ttl=0: %v", err)
	}
	validated, err := ValidateToken(token, secret)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if validated.UserID != "u1" {
		t.Errorf("expected UserID u1, got %q", validated.UserID)
	}
}
