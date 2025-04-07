// Package server provides dependency injection and service management for the application's HTTPS server.
// It initializes and provides access to services and handlers required for handling HTTP requests.
package server

import (
	"github.com/DenisKhanov/TgBOT/internal/server/api/http"
	"github.com/DenisKhanov/TgBOT/internal/server/repository"
	"github.com/DenisKhanov/TgBOT/internal/server/service"
	"github.com/sirupsen/logrus"
	"sync"
)

// serviceProvider manages dependency injection for components related to the HTTPS server.
// It lazily initializes services and handlers as needed.
type serviceProvider struct {
	service        http.Service  // The service instance for business logic.
	handler        *http.Handler // The HTTP handler for routing requests.
	yandexEndpoint string        // Yandex OAuth endpoint URL.
	clientID       string        // Client ID for Yandex OAuth.
	clientSecret   string        // Client secret for Yandex OAuth.
	apiKey         string        // API key for securing endpoints.

	serviceOnce sync.Once // Ensures thread-safe service initialization
	handlerOnce sync.Once // Ensures thread-safe handler initialization
}

// newServiceProvider creates a new instance of serviceProvider with the specified configuration.
// Arguments:
//   - yandexEndpoint: the Yandex OAuth API endpoint.
//   - clientID: the client ID for OAuth authentication.
//   - clientSecret: the client secret for OAuth authentication.
//   - apiKey: the API key for securing HTTP endpoints.
//
// Returns a pointer to a serviceProvider.
func newServiceProvider(yandexEndpoint, clientID, clientSecret, apiKey string) *serviceProvider {
	if yandexEndpoint == "" || clientID == "" || clientSecret == "" || apiKey == "" {
		logrus.Fatal("serviceProvider creation failed: all configuration fields (endpoint, clientID, clientSecret, apiKey) must be non-empty")
	}

	return &serviceProvider{
		yandexEndpoint: yandexEndpoint,
		clientID:       clientID,
		clientSecret:   clientSecret,
		apiKey:         apiKey,
	}
}

// Service returns the service instance for business logic operations.
// It lazily initializes the service using Yandex OAuth and repository dependencies if not already created.
// Returns http.Service implementation.
func (s *serviceProvider) Service() http.Service {
	s.serviceOnce.Do(func() {
		yaOAuth := service.NewYandexAuthAPI(s.yandexEndpoint, s.clientID, s.clientSecret)
		repo := repository.NewRepository()
		s.service = service.NewService(yaOAuth, repo)
		logrus.Info("Service initialized lazily")
	})
	return s.service
}

// Handler returns the HTTP handler for HTTPS endpoints.
// It lazily initializes the handler using the service and API key if not already created.
// Returns a pointer to http.Handler.
func (s *serviceProvider) Handler() *http.Handler {
	s.handlerOnce.Do(func() {
		s.handler = http.NewHandler(s.Service(), s.apiKey)
		logrus.Info("HTTP handler initialized lazily")
	})
	return s.handler
}
