package auth

import (
	"testing"
	"time"
)

func TestGenerateToken(t *testing.T) {
	manager := NewManager("test-secret")

	token, err := manager.GenerateToken("user-123", time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	if token.AccessToken == "" {
		t.Error("AccessToken should not be empty")
	}

	if token.RefreshToken == "" {
		t.Error("RefreshToken should not be empty")
	}

	if token.TokenType != "Bearer" {
		t.Errorf("Expected token type Bearer, got %s", token.TokenType)
	}

	if token.ExpiresAt.Before(time.Now()) {
		t.Error("Token should not be expired")
	}
}

func TestValidateToken(t *testing.T) {
	manager := NewManager("test-secret")

	token, err := manager.GenerateTokenWithRoles("user-123", []string{RoleViewer}, time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	userID, err := manager.ValidateToken(token.AccessToken)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	if userID != "user-123" {
		t.Fatalf("Expected user-123, got %s", userID)
	}

	_, err = manager.ValidateToken("")
	if err == nil {
		t.Error("Should return error for empty token")
	}

	_, err = manager.ValidateToken(token.AccessToken + "tampered")
	if err == nil {
		t.Error("Should reject tampered token")
	}
}

func TestValidateTokenWithClaims(t *testing.T) {
	manager := NewManager("test-secret")

	token, err := manager.GenerateTokenWithRoles("user-456", []string{RoleEngineer, RoleViewer}, time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	claims, err := manager.ValidateTokenWithClaims(token.AccessToken)
	if err != nil {
		t.Fatalf("Failed to validate token with claims: %v", err)
	}

	if claims.UserID != "user-456" {
		t.Fatalf("Expected user-456, got %s", claims.UserID)
	}

	if len(claims.Roles) != 2 {
		t.Fatalf("Expected 2 roles, got %d", len(claims.Roles))
	}
}

func TestValidateExpiredToken(t *testing.T) {
	manager := NewManager("test-secret")

	token, err := manager.signClaims(tokenClaims{
		Subject:   "user-789",
		IssuedAt:  time.Now().UTC().Add(-2 * time.Hour).Unix(),
		ExpiresAt: time.Now().UTC().Add(-time.Hour).Unix(),
		Issuer:    "axiom-idp",
		TokenID:   "expired-token",
	})
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	_, err = manager.ValidateToken(token)
	if err == nil {
		t.Fatal("Expected token validation to fail after expiry")
	}
}
