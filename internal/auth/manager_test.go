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

	userID, err := manager.ValidateToken("valid-token")
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	if userID == "" {
		t.Error("UserID should not be empty")
	}

	_, err = manager.ValidateToken("")
	if err == nil {
		t.Error("Should return error for empty token")
	}
}
