package auth

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"golang.org/x/oauth2"

	"github.com/dedene/frontapp-cli/internal/config"
)

var ErrNotAuthenticated = errors.New("not authenticated")

// TokenSource provides OAuth2 tokens with lazy refresh on 401.
// Access tokens are kept in memory only; refresh tokens are stored in keyring.
type TokenSource struct {
	mu           sync.Mutex
	client       string
	email        string
	store        Store
	accessToken  string
	accessExpiry time.Time
}

func NewTokenSource(client, email string, store Store) *TokenSource {
	return &TokenSource{
		client: client,
		email:  email,
		store:  store,
	}
}

// RefreshTokenSource is a simple token source that uses a refresh token directly.
// Used during initial login before we know the user's email.
type RefreshTokenSource struct {
	client       string
	refreshToken string
}

// NewRefreshTokenSource creates a token source from a refresh token directly.
func NewRefreshTokenSource(client, refreshToken string) *RefreshTokenSource {
	return &RefreshTokenSource{
		client:       client,
		refreshToken: refreshToken,
	}
}

// Token returns an access token by exchanging the refresh token.
func (ts *RefreshTokenSource) Token() (*oauth2.Token, error) {
	creds, err := config.ReadClientCredentials(ts.client)
	if err != nil {
		return nil, fmt.Errorf("read credentials: %w", err)
	}

	cfg := oauth2.Config{
		ClientID:     creds.ClientID,
		ClientSecret: creds.ClientSecret,
		Endpoint:     frontEndpoint,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tok, err := cfg.TokenSource(ctx, &oauth2.Token{
		RefreshToken: ts.refreshToken,
	}).Token()
	if err != nil {
		return nil, fmt.Errorf("refresh token: %w", err)
	}

	return tok, nil
}

// Token returns a valid access token, refreshing if necessary.
func (ts *TokenSource) Token() (*oauth2.Token, error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	// Return cached access token if still valid
	if ts.accessToken != "" && time.Now().Before(ts.accessExpiry) {
		return &oauth2.Token{
			AccessToken: ts.accessToken,
			Expiry:      ts.accessExpiry,
		}, nil
	}

	// Refresh the token
	if err := ts.refresh(); err != nil {
		return nil, err
	}

	return &oauth2.Token{
		AccessToken: ts.accessToken,
		Expiry:      ts.accessExpiry,
	}, nil
}

// Invalidate marks the current access token as invalid, forcing a refresh on next Token() call.
func (ts *TokenSource) Invalidate() {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.accessToken = ""
	ts.accessExpiry = time.Time{}
}

func (ts *TokenSource) refresh() error {
	// Get refresh token from keyring
	tok, err := ts.store.GetToken(ts.client, ts.email)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrNotAuthenticated, err)
	}

	if tok.RefreshToken == "" {
		return ErrNotAuthenticated
	}

	// Get OAuth credentials
	creds, err := config.ReadClientCredentials(ts.client)
	if err != nil {
		return fmt.Errorf("read credentials: %w", err)
	}

	cfg := oauth2.Config{
		ClientID:     creds.ClientID,
		ClientSecret: creds.ClientSecret,
		Endpoint:     frontEndpoint,
	}

	// Use refresh token to get new access token
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	newTok, err := cfg.TokenSource(ctx, &oauth2.Token{
		RefreshToken: tok.RefreshToken,
	}).Token()
	if err != nil {
		return fmt.Errorf("refresh token: %w", err)
	}

	ts.accessToken = newTok.AccessToken
	ts.accessExpiry = newTok.Expiry

	// If we got a new refresh token, store it
	if newTok.RefreshToken != "" && newTok.RefreshToken != tok.RefreshToken {
		tok.RefreshToken = newTok.RefreshToken
		if err := ts.store.SetToken(ts.client, ts.email, tok); err != nil {
			// Log but don't fail - we still have a working access token
			fmt.Printf("Warning: failed to store new refresh token: %v\n", err)
		}
	}

	return nil
}

// GetAuthenticatedEmail returns the email for the authenticated account,
// or error if not authenticated.
func GetAuthenticatedEmail(client string) (string, error) {
	store, err := OpenDefault()
	if err != nil {
		return "", err
	}

	tokens, err := store.ListTokens()
	if err != nil {
		return "", fmt.Errorf("list tokens: %w", err)
	}

	normalizedClient, err := config.NormalizeClientNameOrDefault(client)
	if err != nil {
		return "", fmt.Errorf("normalize client: %w", err)
	}

	for _, tok := range tokens {
		if tok.Client == normalizedClient {
			return tok.Email, nil
		}
	}

	return "", ErrNotAuthenticated
}
