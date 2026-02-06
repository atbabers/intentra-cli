// Package auth provides authentication and token management for the Intentra CLI.
// It handles OAuth 2.0 device flow authentication with Auth0, secure token storage,
// and automatic token refresh.
package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/atbabers/intentra-cli/internal/config"
)

const defaultAPIEndpoint = "https://api.intentra.sh"

// Credentials represents stored authentication credentials.
type Credentials struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	IDToken      string    `json:"id_token,omitempty"`
	TokenType    string    `json:"token_type"`
	ExpiresAt    time.Time `json:"expires_at"`
	UserID       string    `json:"user_id,omitempty"`
	Email        string    `json:"email,omitempty"`
}

// DeviceCodeResponse represents the response from the device code endpoint.
type DeviceCodeResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

// TokenResponse represents the response from the token endpoint.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Error        string `json:"error,omitempty"`
	ErrorDesc    string `json:"error_description,omitempty"`
}

// IsExpired returns true if the credentials have expired or will expire within the buffer period.
func (c *Credentials) IsExpired() bool {
	buffer := 5 * time.Minute
	return time.Now().Add(buffer).After(c.ExpiresAt)
}

// IsValid returns true if credentials exist and are not expired.
func (c *Credentials) IsValid() bool {
	return c.AccessToken != "" && !c.IsExpired()
}

// LoadCredentials loads credentials from the credentials file.
func LoadCredentials() (*Credentials, error) {
	credFile := config.GetCredentialsFile()

	data, err := os.ReadFile(credFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read credentials: %w", err)
	}

	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	return &creds, nil
}

// SaveCredentials saves credentials to the credentials file with secure permissions.
func SaveCredentials(creds *Credentials) error {
	if err := config.EnsureDirectories(); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	credFile := config.GetCredentialsFile()
	if err := os.WriteFile(credFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write credentials: %w", err)
	}

	return nil
}

// DeleteCredentials removes the credentials file.
func DeleteCredentials() error {
	credFile := config.GetCredentialsFile()

	err := os.Remove(credFile)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete credentials: %w", err)
	}

	return nil
}

// CredentialsFromTokenResponse creates Credentials from a TokenResponse.
func CredentialsFromTokenResponse(resp *TokenResponse) *Credentials {
	expiresAt := time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second)

	return &Credentials{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		IDToken:      resp.IDToken,
		TokenType:    resp.TokenType,
		ExpiresAt:    expiresAt,
	}
}

// GetValidCredentials loads credentials from secure storage, refreshes if needed, and returns them if valid.
func GetValidCredentials() *Credentials {
	creds, err := LoadCredentialsFromKeyring()
	if err != nil || creds == nil {
		return nil
	}

	if creds.IsValid() {
		return creds
	}

	if creds.RefreshToken == "" {
		return nil
	}

	refreshed, err := RefreshCredentials(creds)
	if err != nil {
		return nil
	}

	return refreshed
}

// GetValidCredentialsSecure is an alias for GetValidCredentials using secure storage.
func GetValidCredentialsSecure() *Credentials {
	return GetValidCredentials()
}

// RefreshCredentials uses the refresh token to obtain new credentials.
func RefreshCredentials(creds *Credentials) (*Credentials, error) {
	if creds.RefreshToken == "" {
		return nil, fmt.Errorf("no refresh token available")
	}

	currentCreds, _ := LoadCredentialsFromKeyring()
	if currentCreds != nil && currentCreds.IsValid() && currentCreds.AccessToken != creds.AccessToken {
		return currentCreds, nil
	}

	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	endpoint := cfg.Server.Endpoint
	if endpoint == "" {
		endpoint = defaultAPIEndpoint
	}

	url := endpoint + "/oauth/refresh"
	payload := map[string]string{
		"refresh_token": creds.RefreshToken,
	}
	payloadBytes, _ := json.Marshal(payload)

	resp, err := http.Post(url, "application/json", bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("refresh failed (%d): %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("no access token in response")
	}

	newCreds := CredentialsFromTokenResponse(&tokenResp)
	newCreds.UserID = creds.UserID
	newCreds.Email = creds.Email

	if newCreds.RefreshToken == "" {
		newCreds.RefreshToken = creds.RefreshToken
	}

	err = WithCredentialLock(func() error {
		latestCreds, _ := LoadCredentialsFromKeyring()
		if latestCreds != nil && latestCreds.IsValid() && latestCreds.AccessToken != creds.AccessToken {
			newCreds = latestCreds
			return nil
		}

		return storeCredentialsInKeyringUnlocked(newCreds)
	})

	if err != nil {
		if err := SaveCredentials(newCreds); err != nil {
			return nil, fmt.Errorf("failed to save refreshed credentials: %w", err)
		}
	}

	return newCreds, nil
}
