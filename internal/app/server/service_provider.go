// Package server provides dependency injection and service management for the application's HTTPS server.
// It initializes and provides access to services and handlers required for handling HTTP requests.
package server

import (
	"fmt"
	"github.com/DenisKhanov/TgBOT/internal/server/api/http"
	"github.com/DenisKhanov/TgBOT/internal/server/repository"
	"github.com/DenisKhanov/TgBOT/internal/server/service"
	"sync"
)

// serviceProvider manages dependency injection for components related to the HTTPS server.
// It lazily initializes services and handlers as needed.
type serviceProvider struct {
	service        http.Service // The service instance for business logic.
	serviceErr     error
	handler        *http.Handler // The HTTP handler for routing requests.
	handlerErr     error
	yandexEndpoint string // Yandex OAuth endpoint URL.
	clientID       string // Client ID for Yandex OAuth.
	clientSecret   string // Client secret for Yandex OAuth.
	apiKey         string // API key for securing endpoints.
	tokenStorage   string // Path to persisted token storage.

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
func newServiceProvider(yandexEndpoint, clientID, clientSecret, apiKey, tokenStorage string) (*serviceProvider, error) {
	if yandexEndpoint == "" || clientID == "" || clientSecret == "" || apiKey == "" || tokenStorage == "" {
		return nil, fmt.Errorf("serviceProvider creation failed: all configuration fields (endpoint, clientID, clientSecret, apiKey, tokenStorage) must be non-empty")
	}

	return &serviceProvider{
		yandexEndpoint: yandexEndpoint,
		clientID:       clientID,
		clientSecret:   clientSecret,
		apiKey:         apiKey,
		tokenStorage:   tokenStorage,
	}, nil
}

// Service returns the service instance for business logic operations.
// It lazily initializes the service using Yandex OAuth and repository dependencies if not already created.
// Returns http.Service implementation.
func (s *serviceProvider) Service() (http.Service, error) {
	s.serviceOnce.Do(func() {
		yaOAuth := service.NewYandexAuthAPI(s.yandexEndpoint, s.clientID, s.clientSecret)
		repo, err := repository.NewRepository(s.tokenStorage)
		if err != nil {
			s.serviceErr = fmt.Errorf("initialize token repository: %w", err)
			return
		}
		s.service = service.NewService(yaOAuth, repo)
	})
	if s.serviceErr != nil {
		return nil, s.serviceErr
	}
	if s.service == nil {
		return nil, fmt.Errorf("service not initialized")
	}
	return s.service, nil
}

// Handler returns the HTTP handler for HTTPS endpoints.
// It lazily initializes the handler using the service and API key if not already created.
// Returns a pointer to http.Handler.
func (s *serviceProvider) Handler() (*http.Handler, error) {
	s.handlerOnce.Do(func() {
		service, err := s.Service()
		if err != nil {
			s.handlerErr = err
			return
		}
		s.handler = http.NewHandler(service, s.apiKey)
	})
	if s.handlerErr != nil {
		return nil, s.handlerErr
	}
	if s.handler == nil {
		return nil, fmt.Errorf("handler not initialized")
	}
	return s.handler, nil
}
