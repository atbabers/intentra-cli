// Package api provides HTTP client functionality for communicating with the
// Intentra server. It supports multiple authentication modes including HMAC,
// API key, and mTLS for enterprise deployments.
package api

import (
	"bytes"
	"crypto/hmac"
	cryptoRand "crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/atbabers/intentra-cli/internal/config"
	"github.com/atbabers/intentra-cli/internal/device"
	"github.com/atbabers/intentra-cli/pkg/models"
)

// Client handles communication with the Intentra API.
type Client struct {
	cfg        *config.Config
	httpClient *http.Client
}

// NewClient creates a new API client configured with the provided settings.
func NewClient(cfg *config.Config) (*Client, error) {
	if !cfg.Server.Enabled {
		return nil, fmt.Errorf("server sync is not enabled")
	}

	transport := &http.Transport{}

	// Configure mTLS if enabled
	if cfg.Server.Auth.Mode == "mtls" {
		tlsConfig, err := configureMTLS(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to configure mTLS: %w", err)
		}
		transport.TLSClientConfig = tlsConfig
	}

	httpClient := &http.Client{
		Timeout:   cfg.Server.Timeout,
		Transport: transport,
	}

	return &Client{
		cfg:        cfg,
		httpClient: httpClient,
	}, nil
}

// configureMTLS sets up mTLS client certificates for JAMF/MDM deployments.
func configureMTLS(cfg *config.Config) (*tls.Config, error) {
	mtlsCfg := cfg.Server.Auth.MTLS

	// Load client certificate and key
	cert, err := tls.LoadX509KeyPair(mtlsCfg.CertFile, mtlsCfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate: %w", err)
	}

	// SECURITY: InsecureSkipVerify is only allowed in development environments.
	// In production, always validate server certificates to prevent MITM attacks.
	skipVerify := mtlsCfg.SkipVerify
	if skipVerify {
		env := os.Getenv("INTENTRA_ENV")
		if env != "development" && env != "dev" && env != "local" {
			return nil, fmt.Errorf("InsecureSkipVerify is not allowed in production environments (INTENTRA_ENV=%s). Set INTENTRA_ENV=development to enable for testing only", env)
		}
	}

	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: skipVerify,
	}

	// Load CA certificate if provided
	if mtlsCfg.CAFile != "" {
		caCert, err := os.ReadFile(mtlsCfg.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		tlsConfig.RootCAs = caCertPool
	}

	return tlsConfig, nil
}

// SendScan sends a single scan to the API.
func (c *Client) SendScan(scan *models.Scan) error {
	return c.SendScans([]*models.Scan{scan})
}

// SendScans sends a batch of scans to the API.
func (c *Client) SendScans(scans []*models.Scan) error {
	if len(scans) == 0 {
		return nil
	}

	// Get device ID (auto-generated if not configured)
	deviceID, err := c.getDeviceID()
	if err != nil {
		return fmt.Errorf("failed to get device ID: %w", err)
	}

	// Get device metadata for every request
	metadata := device.GetMetadata()

	// Prepare request body with device ID and metadata
	body := map[string]any{
		"device_id":        deviceID,
		"hostname":         metadata.Hostname,
		"username":         metadata.Username,
		"platform":         metadata.Platform,
		"os_version":       metadata.OSVersion,
		"intentra_version": metadata.IntentraVersion,
		"scans":            scans,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal scans: %w", err)
	}

	// Create request
	url := c.cfg.Server.Endpoint + "/scans"
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "intentra-cli/1.0")

	// Add authentication
	if err := c.addAuth(req, jsonBody); err != nil {
		return fmt.Errorf("failed to add auth: %w", err)
	}

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Health checks API connectivity.
func (c *Client) Health() error {
	url := c.cfg.Server.Endpoint + "/health"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned: %d", resp.StatusCode)
	}

	return nil
}

// getDeviceID returns the device ID (configured or auto-generated).
func (c *Client) getDeviceID() (string, error) {
	// Use configured device ID if provided
	if c.cfg.Server.Auth.HMAC.DeviceID != "" {
		return c.cfg.Server.Auth.HMAC.DeviceID, nil
	}
	// Auto-generate immutable device ID from hardware
	return device.GetDeviceID()
}

// addAuth adds authentication headers based on config.
func (c *Client) addAuth(req *http.Request, body []byte) error {
	switch c.cfg.Server.Auth.Mode {
	case "hmac":
		return c.addHMACAuth(req, body)
	case "api_key":
		return c.addAPIKeyAuth(req, body)
	case "mtls":
		// mTLS authentication is handled at transport level
		// Just add device ID header
		deviceID, err := c.getDeviceID()
		if err != nil {
			return err
		}
		req.Header.Set("X-Device-ID", deviceID)
		return nil
	default:
		return fmt.Errorf("unknown auth mode: %s", c.cfg.Server.Auth.Mode)
	}
}

// signRequest generates HMAC-SHA256 signature with replay protection (nonce + timestamp).
// Returns timestamp, nonce, and signature for use in auth headers.
func signRequest(secret string, body []byte) (timestamp, nonce, signature string, err error) {
	timestamp = strconv.FormatInt(time.Now().Unix(), 10)

	// Generate nonce for replay protection (16 bytes = 32 hex chars)
	nonceBytes := make([]byte, 16)
	if _, err = io.ReadFull(cryptoRand.Reader, nonceBytes); err != nil {
		return "", "", "", fmt.Errorf("failed to generate nonce: %w", err)
	}
	nonce = hex.EncodeToString(nonceBytes)

	// Create signature: HMAC-SHA256(secret, "timestamp:nonce:body")
	// Including nonce prevents replay attacks even within timestamp window
	message := timestamp + ":" + nonce + ":" + string(body)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(message))
	signature = hex.EncodeToString(h.Sum(nil))

	return timestamp, nonce, signature, nil
}

// setAuthHeaders sets the common HMAC auth headers on the request.
func setAuthHeaders(req *http.Request, keyID, timestamp, nonce, signature string) {
	req.Header.Set("X-API-Key-ID", keyID)
	req.Header.Set("X-API-Timestamp", timestamp)
	req.Header.Set("X-API-Nonce", nonce)
	req.Header.Set("X-API-Signature", signature)
}

// addHMACAuth adds HMAC authentication headers with replay protection.
// Server expects: X-API-Key-ID, X-API-Timestamp, X-API-Nonce, X-API-Signature, X-Device-ID
func (c *Client) addHMACAuth(req *http.Request, body []byte) error {
	secret := c.cfg.Server.Auth.HMAC.Secret
	if secret == "" {
		return fmt.Errorf("HMAC auth requires secret")
	}

	keyID := c.cfg.Server.Auth.HMAC.KeyID
	if keyID == "" {
		return fmt.Errorf("HMAC auth requires key_id to be set")
	}

	deviceID, err := c.getDeviceID()
	if err != nil {
		return fmt.Errorf("failed to get device ID: %w", err)
	}

	timestamp, nonce, signature, err := signRequest(secret, body)
	if err != nil {
		return err
	}

	setAuthHeaders(req, keyID, timestamp, nonce, signature)
	req.Header.Set("X-Device-ID", deviceID)

	return nil
}

// addAPIKeyAuth adds API key authentication headers with replay protection.
// Server expects same format as HMAC: X-API-Key-ID, X-API-Timestamp, X-API-Nonce, X-API-Signature
func (c *Client) addAPIKeyAuth(req *http.Request, body []byte) error {
	keyID := c.cfg.Server.Auth.APIKey.KeyID
	secret := c.cfg.Server.Auth.APIKey.Secret

	if keyID == "" || secret == "" {
		return fmt.Errorf("API key auth requires key_id and secret")
	}

	timestamp, nonce, signature, err := signRequest(secret, body)
	if err != nil {
		return err
	}

	setAuthHeaders(req, keyID, timestamp, nonce, signature)

	return nil
}

