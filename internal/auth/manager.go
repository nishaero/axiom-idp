package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

var (
	// ErrInvalidCredentials is returned when credentials are invalid.
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrTokenExpired is returned when the token is expired.
	ErrTokenExpired = errors.New("token expired")
	// ErrTokenInvalid is returned when the token is malformed or has a bad signature.
	ErrTokenInvalid = errors.New("token invalid")
)

const tokenVersion = "v1"

// User represents an authenticated user.
type User struct {
	ID    string            `json:"id"`
	Name  string            `json:"name"`
	Email string            `json:"email"`
	Roles []string          `json:"roles"`
	Meta  map[string]string `json:"metadata"`
}

// Token represents an authentication token.
type Token struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
}

// ValidatedToken contains the decoded token claims.
type ValidatedToken struct {
	UserID    string
	Roles     []string
	IssuedAt  time.Time
	ExpiresAt time.Time
	TokenID   string
}

// Manager handles authentication.
type Manager struct {
	sessionSecret []byte
	issuer        string
}

type tokenClaims struct {
	Subject   string   `json:"sub"`
	Roles     []string `json:"roles,omitempty"`
	IssuedAt  int64    `json:"iat"`
	ExpiresAt int64    `json:"exp"`
	Issuer    string   `json:"iss"`
	TokenID   string   `json:"jti"`
}

// NewManager creates a new auth manager.
func NewManager(sessionSecret string) *Manager {
	return &Manager{
		sessionSecret: []byte(sessionSecret),
		issuer:        "axiom-idp",
	}
}

// GenerateToken generates a new authentication token for a user without roles.
func (m *Manager) GenerateToken(userID string, expiresIn time.Duration) (*Token, error) {
	return m.GenerateTokenWithRoles(userID, nil, expiresIn)
}

// GenerateTokenWithRoles generates a signed authentication token with roles.
func (m *Manager) GenerateTokenWithRoles(userID string, roles []string, expiresIn time.Duration) (*Token, error) {
	if userID == "" {
		return nil, ErrInvalidCredentials
	}
	if expiresIn <= 0 {
		expiresIn = 24 * time.Hour
	}

	normalizedRoles := normalizeRoles(roles)
	claims := tokenClaims{
		Subject:   userID,
		Roles:     normalizedRoles,
		IssuedAt:  time.Now().UTC().Unix(),
		ExpiresAt: time.Now().UTC().Add(expiresIn).Unix(),
		Issuer:    m.issuer,
		TokenID:   randomTokenID(),
	}

	accessToken, err := m.signClaims(claims)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := generateRandomToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Unix(claims.ExpiresAt, 0).UTC(),
		TokenType:    "Bearer",
	}, nil
}

// ValidateToken validates an authentication token and returns the user ID.
func (m *Manager) ValidateToken(token string) (string, error) {
	claims, err := m.ValidateTokenWithClaims(token)
	if err != nil {
		return "", err
	}

	return claims.UserID, nil
}

// ValidateTokenWithClaims validates an authentication token and returns decoded claims.
func (m *Manager) ValidateTokenWithClaims(token string) (*ValidatedToken, error) {
	token = strings.TrimSpace(strings.TrimPrefix(token, "Bearer"))
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, ErrInvalidCredentials
	}

	claims, err := m.verifyToken(token)
	if err != nil {
		return nil, err
	}

	return &ValidatedToken{
		UserID:    claims.Subject,
		Roles:     normalizeRoles(claims.Roles),
		IssuedAt:  time.Unix(claims.IssuedAt, 0).UTC(),
		ExpiresAt: time.Unix(claims.ExpiresAt, 0).UTC(),
		TokenID:   claims.TokenID,
	}, nil
}

func (m *Manager) signClaims(claims tokenClaims) (string, error) {
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	signingInput := tokenVersion + "." + encodedPayload
	mac := hmac.New(sha256.New, m.sessionSecret)
	_, _ = mac.Write([]byte(signingInput))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return signingInput + "." + signature, nil
}

func (m *Manager) verifyToken(token string) (*tokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrTokenInvalid
	}

	if parts[0] != tokenVersion {
		return nil, ErrTokenInvalid
	}

	signingInput := parts[0] + "." + parts[1]
	expectedSignature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, ErrTokenInvalid
	}

	mac := hmac.New(sha256.New, m.sessionSecret)
	_, _ = mac.Write([]byte(signingInput))
	if !hmac.Equal(mac.Sum(nil), expectedSignature) {
		return nil, ErrTokenInvalid
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrTokenInvalid
	}

	var claims tokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, ErrTokenInvalid
	}

	if claims.Subject == "" {
		return nil, ErrTokenInvalid
	}
	if claims.ExpiresAt <= 0 {
		return nil, ErrTokenInvalid
	}
	if time.Now().UTC().Unix() > claims.ExpiresAt {
		return nil, ErrTokenExpired
	}
	if claims.Issuer != "" && claims.Issuer != m.issuer {
		return nil, ErrTokenInvalid
	}

	return &claims, nil
}

// Helper functions.
func generateRandomToken(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func randomTokenID() string {
	tokenID, err := generateRandomToken(16)
	if err != nil {
		return fmt.Sprintf("jti-%d", time.Now().UnixNano())
	}
	return tokenID
}

func normalizeRoles(roles []string) []string {
	if len(roles) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(roles))
	normalized := make([]string, 0, len(roles))
	for _, role := range roles {
		role = strings.TrimSpace(strings.ToLower(role))
		if role == "" {
			continue
		}
		if _, exists := seen[role]; exists {
			continue
		}
		seen[role] = struct{}{}
		normalized = append(normalized, role)
	}

	sort.Strings(normalized)
	return normalized
}
