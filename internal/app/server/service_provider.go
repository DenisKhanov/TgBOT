package server

import (
	"github.com/DenisKhanov/TgBOT/internal/server/api/http"
	"github.com/DenisKhanov/TgBOT/internal/server/service"
)

// serviceProvider manages the dependency injection for http_shortener-related components.
type serviceProvider struct {
	service http.Service  // Service for
	handler *http.Handler // Handler for
}

// newServiceProvider creates a new instance of the service provider.
func newServiceProvider() *serviceProvider {
	return &serviceProvider{}
}

// Service returns the service for user-related operations.
func (s *serviceProvider) Service(yaEndpoint, clientID, clientSecret string) http.Service {
	yaOAuth := service.NewYandexAuthAPI(yaEndpoint, clientID, clientSecret)
	if s.service == nil {
		s.service = service.NewService(yaOAuth)
	}
	return s.service
}

// Handler returns the http for user-related HTTP endpoints.
func (s *serviceProvider) Handler(yaEndpoint, clientID, clientSecret string) *http.Handler {
	if s.handler == nil {
		s.handler = http.NewHandler(s.Service(yaEndpoint, clientID, clientSecret))
	}
	return s.handler
}
