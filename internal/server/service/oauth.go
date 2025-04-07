// Package service provides functionality for interacting with the Yandex OAuth API
// to get authorization tokens using client credentials and access codes.
package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/DenisKhanov/TgBOT/internal/server/models"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// YandexAuth represents a client for interacting with the Yandex OAuth authentication API.
// It encapsulates the endpoint and credentials required for token requests.
type YandexAuth struct {
	endpoint     string // The OAuth API endpoint URL.
	clientID     string // The client ID for authentication.
	clientSecret string // The client secret for authentication.
}

// NewYandexAuthAPI creates a new YandexAuth instance with the specified endpoint and credentials.
// Arguments:
//   - endpoint: the URL of the Yandex OAuth token endpoint.
//   - clientID: the client ID issued by Yandex.
//   - clientSecret: the client secret issued by Yandex.
//
// Returns a YandexAuth struct.
func NewYandexAuthAPI(endpoint, clientID, clientSecret string) *YandexAuth {
	return &YandexAuth{
		endpoint:     endpoint,
		clientID:     clientID,
		clientSecret: clientSecret,
	}
}

// GetOAuthToken retrieves an OAuth token from Yandex using the provided access code.
// It sends a POST request to the configured endpoint with the authorization code flow.
// Arguments:
//   - accessCode: the authorization code received from Yandex.
//
// Returns a models.ResponseAUTH struct containing the token details or an error if the request fails.
func (a *YandexAuth) GetOAuthToken(accessCode string) (models.ResponseAUTH, error) {
	if accessCode == "" {
		err := fmt.Errorf("access code is required")
		logrus.WithError(err).Error("invalid input for OAuth token request")
		return models.ResponseAUTH{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	params := url.Values{}
	params.Add("grant_type", "authorization_code")
	params.Add("code", accessCode)
	params.Add("client_id", a.clientID)
	params.Add("client_secret", a.clientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.endpoint, strings.NewReader(params.Encode()))
	if err != nil {
		err = fmt.Errorf("failed to create request with ctx: %w", err)
		logrus.WithError(err).Error("request creation failed")
		return models.ResponseAUTH{}, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// Кодируем строку client_id: client_secret методом base64
	auth := base64.StdEncoding.EncodeToString([]byte(a.clientID + ":" + a.clientSecret))
	// Добавляем заголовок Authorization
	req.Header.Set("Authorization", "Basic "+auth)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		err = fmt.Errorf("failed to execute request: %w", err)
		logrus.WithError(err).Error("HTTP request to Yandex OAuth failed")
		return models.ResponseAUTH{}, err
	}
	defer func() {
		if err = res.Body.Close(); err != nil {
			logrus.WithError(err).Errorf("Failed to close response body: %v", err)
		}
	}()

	if res.StatusCode != http.StatusOK {
		err = fmt.Errorf("unexpected status code: %d", res.StatusCode)
		logrus.WithError(err).Error("Yandex OAuth returned non-200 response")
		return models.ResponseAUTH{}, err
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		err = fmt.Errorf("failed to read response body: %w", err)
		logrus.WithError(err).Error("response body read error")
		return models.ResponseAUTH{}, err
	}

	var response models.ResponseAUTH
	if err = json.Unmarshal(data, &response); err != nil {
		err = fmt.Errorf("failed to unmarshal response: %w", err)
		logrus.WithError(err).Error("JSON parsing error")
		return models.ResponseAUTH{}, err
	}

	if response.AccessToken == "" {
		err = fmt.Errorf("empty access token in response")
		logrus.WithError(err).Error("invalid token response from Yandex")
		return models.ResponseAUTH{}, err
	}

	logrus.Infof("successfully retrieved OAuth token for clientID: %s", a.clientID)
	return response, nil
}
