// Package http provides an HTTP handler for interacting with a server endpoint using TLS.
// It manages client authentication and token retrieval for Telegram bot operations.
package http

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"github.com/DenisKhanov/TgBOT/internal/tg_bot/models"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Handler manages HTTP requests to a server endpoint with TLS client authentication.
// It uses a configured HTTP client for secure communication.
type Handler struct {
	client         *http.Client // HTTP client with TLS configuration.
	serverEndpoint string       // Server endpoint URL (must use HTTPS).
	apiKey         string       // API key for request authentication.
}

// NewHandler creates a new Handler instance with TLS client authentication.
// Arguments:
//   - serverEndpoint: the server endpoint URL (must start with "https://").
//   - cert: path to the TLS client certificate file.
//   - key: path to the TLS client key file.
//   - ca: path to the CA certificate file.
//   - apiKey: API key for securing requests.
//
// Panic if TLS configuration or file loading fails.
// Returns a pointer to a Handler.
func NewHandler(serverEndpoint, cert, key, ca, apiKey string) (*Handler, error) {
	if !strings.HasPrefix(serverEndpoint, "https://") {
		return nil, fmt.Errorf("server endpoint must start with 'https://': %s", serverEndpoint)
	}
	if cert == "" || key == "" || ca == "" || apiKey == "" {
		return nil, fmt.Errorf("certificate, key, CA, and API key must be non-empty")
	}

	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	clientCertPath := filepath.Join(wd, cert)
	clientKeyPath := filepath.Join(wd, key)
	caPath := filepath.Join(wd, ca)

	clientCert, err := tls.LoadX509KeyPair(clientCertPath, clientKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate from %s and %s: %w", clientCertPath, clientKeyPath, err)
	}

	caCert, err := os.ReadFile(caPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load CA certificate from %s: %w", caPath, err)
	}
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to append CA certificate from %s to pool", caPath)
	}

	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{clientCert},
		RootCAs:            caCertPool,
		InsecureSkipVerify: false,
	}

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	h := &Handler{
		client:         client,
		serverEndpoint: serverEndpoint,
		apiKey:         apiKey,
	}
	logrus.Infof("HTTP Handler initialized with endpoint: %s", serverEndpoint)
	return h, nil
}

// GetUserToken retrieves an OAuth token pair for a given chat ID from the server.
// Arguments:
//   - chatID: the Telegram chat ID (int64) used as the state parameter.
//
// Returns a models.ResponseOAuth containing token details or an error if the request fails.
func (h *Handler) GetUserToken(chatID int64) (models.ResponseOAuth, error) {
	if chatID <= 0 {
		err := fmt.Errorf("invalid chatID: %d must be positive", chatID)
		logrus.WithError(err).Error("Failed to process token request")
		return models.ResponseOAuth{}, err
	}

	strChatID := strconv.FormatInt(chatID, 10)
	url := h.serverEndpoint + "?state=" + strChatID
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logrus.WithError(err).Error("Failed to create token request")
		return models.ResponseOAuth{}, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("X-API-Key", h.apiKey)

	resp, err := h.client.Do(req)
	if err != nil {
		logrus.WithError(err).Error("Failed to execute token request")
		return models.ResponseOAuth{}, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {
			logrus.WithError(err).Errorf("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("server returned status: %d", resp.StatusCode)
		logrus.WithError(err).Error("Token request failed")
		return models.ResponseOAuth{}, err
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		logrus.WithError(err).Error("Failed to read token response")
		return models.ResponseOAuth{}, fmt.Errorf("failed to read response body: %w", err)
	}

	var tokenPair models.ResponseOAuth
	if err = json.Unmarshal(data, &tokenPair); err != nil {
		logrus.WithError(err).Error("Failed to unmarshal token response")
		return models.ResponseOAuth{}, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if tokenPair.AccessToken == "" {
		err = fmt.Errorf("empty access token in response")
		logrus.WithError(err).Error("Invalid token response")
		return models.ResponseOAuth{}, err
	}

	logrus.Infof("Successfully retrieved token for chatID: %d", chatID)
	return tokenPair, nil
}
