package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// OAuth2Provider represents an OAuth2/OIDC provider
type OAuth2Provider struct {
	provider     *oidc.Provider
	oauth2Config *oauth2.Config
	issuer       string
	clientID     string
	clientSecret string
}

// NewOAuth2Provider creates a new OAuth2 provider
func NewOAuth2Provider(issuer, clientID, clientSecret, redirectURL string) (*OAuth2Provider, error) {
	ctx := context.Background()

	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC provider: %w", err)
	}

	oauth2Config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}

	return &OAuth2Provider{
		provider:     provider,
		oauth2Config: oauth2Config,
		issuer:       issuer,
		clientID:     clientID,
		clientSecret: clientSecret,
	}, nil
}

// GetAuthURL returns the OAuth2 authorization URL
func (op *OAuth2Provider) GetAuthURL(state string) string {
	return op.oauth2Config.AuthCodeURL(state)
}

// ExchangeCode exchanges an auth code for a token
func (op *OAuth2Provider) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	return op.oauth2Config.Exchange(ctx, code)
}

// VerifyToken verifies and parses an ID token
func (op *OAuth2Provider) VerifyToken(ctx context.Context, idToken string) (*oidc.IDToken, error) {
	verifier := op.provider.Verifier(&oidc.Config{ClientID: op.clientID})
	return verifier.Verify(ctx, idToken)
}

// GetUserInfo extracts user information from token claims
func (op *OAuth2Provider) GetUserInfo(ctx context.Context, accessToken *oauth2.Token) (*User, error) {
	idToken, ok := accessToken.Extra("id_token").(string)
	if !ok {
		return nil, fmt.Errorf("no id_token in response")
	}

	token, err := op.VerifyToken(ctx, idToken)
	if err != nil {
		return nil, err
	}

	var claims struct {
		Email       string `json:"email"`
		Name        string `json:"name"`
		Verified    bool   `json:"email_verified"`
		Picture     string `json:"picture"`
	}

	if err := token.Claims(&claims); err != nil {
		return nil, err
	}

	return &User{
		ID:    token.Subject,
		Name:  claims.Name,
		Email: claims.Email,
		Meta: map[string]string{
			"picture": claims.Picture,
		},
	}, nil
}

// OAuth2Handler handles OAuth2 authorization
type OAuth2Handler struct {
	provider   *OAuth2Provider
	authManager *Manager
	redirectURL string
}

// NewOAuth2Handler creates a new OAuth2 handler
func NewOAuth2Handler(provider *OAuth2Provider, authManager *Manager, redirectURL string) *OAuth2Handler {
	return &OAuth2Handler{
		provider:    provider,
		authManager: authManager,
		redirectURL: redirectURL,
	}
}

// HandleLogin initiates OAuth2 flow
func (h *OAuth2Handler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	state := generateRandomString(32)
	authURL := h.provider.GetAuthURL(state)

	// Store state in session (in production, use secure session storage)
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600, // 10 minutes
	})

	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// HandleCallback handles OAuth2 callback
func (h *OAuth2Handler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	// Verify state
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil {
		http.Error(w, "Missing state cookie", http.StatusBadRequest)
		return
	}

	if r.FormValue("state") != stateCookie.Value {
		http.Error(w, "State mismatch", http.StatusBadRequest)
		return
	}

	code := r.FormValue("code")
	if code == "" {
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		return
	}

	// Exchange code for token
	token, err := h.provider.ExchangeCode(r.Context(), code)
	if err != nil {
		http.Error(w, "Token exchange failed", http.StatusInternalServerError)
		return
	}

	// Get user information
	user, err := h.provider.GetUserInfo(r.Context(), token)
	if err != nil {
		http.Error(w, "Failed to get user info", http.StatusInternalServerError)
		return
	}

	// Generate session token
	sessionToken, err := h.authManager.GenerateToken(user.ID, 24*time.Hour)
	if err != nil {
		http.Error(w, "Failed to generate session token", http.StatusInternalServerError)
		return
	}

	// Redirect with token
	redirectURL := url.URL{
		Scheme:   "http",
		Host:     "localhost:3000",
		Path:     "/auth/callback",
		RawQuery: fmt.Sprintf("token=%s", sessionToken.AccessToken),
	}

	http.Redirect(w, r, redirectURL.String(), http.StatusTemporaryRedirect)
}

// Helper function
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[i%len(charset)]
	}
	return string(b)
}
