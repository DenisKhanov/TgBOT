// Package tbot provides dependency injection and service management for Telegram bot components.
// It initializes and provides access to services, repositories, and handlers required for bot operations.
package tbot

import (
	"fmt"
	"github.com/DenisKhanov/TgBOT/internal/tg_bot/api"
	botHand "github.com/DenisKhanov/TgBOT/internal/tg_bot/api/http"
	"github.com/DenisKhanov/TgBOT/internal/tg_bot/infra/generative"
	"github.com/DenisKhanov/TgBOT/internal/tg_bot/models"
	"github.com/DenisKhanov/TgBOT/internal/tg_bot/repository"
	botServ "github.com/DenisKhanov/TgBOT/internal/tg_bot/service"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"sync"
)

// ServiceProvider manages the dependency injection for Telegram bot components.
type ServiceProvider struct {
	// Services
	boringService     botServ.Boring
	translateService  botServ.Translate
	smartHomeService  botServ.SmartHome
	generativeService botServ.GenerativeModel

	// Repository
	repository botServ.Repository

	// Handler
	handler botServ.Handler

	// Bot API
	botAPI *tgbotapi.BotAPI

	// Bot service
	botService *botServ.TgBotServices

	// API endpoints
	translateAPIEndpoint  string
	dictionaryAPIEndpoint string
	smartHomeAPIEndpoint  string

	// Config values
	serverEndpoint   string
	translateApiKey  string
	generativeName   string
	generativeApiKey string
	generativeModel  string
	storagePath      string

	//TLS file path
	clientCert string
	clientKey  string
	clientCa   string

	apiKey   string
	clientID string
	ownerID  int64

	boringOnce     sync.Once
	translateOnce  sync.Once
	smartHomeOnce  sync.Once
	generativeOnce sync.Once
	repoOnce       sync.Once
	handlerOnce    sync.Once
	botAPIOnce     sync.Once
	botServiceOnce sync.Once
}

// NewServiceProvider creates a new instance of the service provider.
func NewServiceProvider(
	translateAPIEndpoint, dictionaryAPIEndpoint, smartHomeAPIEndpoint string,
	serverEndpoint, translateApiKey,
	generativeName, generativeApiKey,
	generativeModel, storagePath, clientCert,
	clientKey, clientCa, apiKey,
	clientID string, ownerID int64,
) *ServiceProvider {
	if translateAPIEndpoint == "" || dictionaryAPIEndpoint == "" || smartHomeAPIEndpoint == "" || serverEndpoint == "" || translateApiKey == "" || generativeApiKey == "" || generativeName == "" || generativeModel == "" || storagePath == "" || clientCert == "" || clientKey == "" || clientCa == "" || apiKey == "" || clientID == "" || ownerID == 0 {
		logrus.Fatal("All ServiceProvider configuration fields must be non-empty")
	}
	return &ServiceProvider{
		translateAPIEndpoint:  translateAPIEndpoint,
		dictionaryAPIEndpoint: dictionaryAPIEndpoint,
		smartHomeAPIEndpoint:  smartHomeAPIEndpoint,
		serverEndpoint:        serverEndpoint,
		translateApiKey:       translateApiKey,
		generativeName:        generativeName,
		generativeApiKey:      generativeApiKey,
		generativeModel:       generativeModel,
		storagePath:           storagePath,
		clientCert:            clientCert,
		clientKey:             clientKey,
		clientCa:              clientCa,
		apiKey:                apiKey,
		clientID:              clientID,
		ownerID:               ownerID,
	}
}

// BoringService returns the service for activity suggestions.
func (s *ServiceProvider) BoringService() botServ.Boring {
	s.boringOnce.Do(func() {
		s.boringService = botServ.NewBoringAPI(models.ActivitiesRU)
		logrus.Info("BoringService initialized")
	})
	return s.boringService
}

// TranslateService returns the service for translation.
func (s *ServiceProvider) TranslateService() botServ.Translate {
	s.translateOnce.Do(func() {
		s.translateService = api.NewYandexAPI(s.translateAPIEndpoint, s.dictionaryAPIEndpoint, s.translateApiKey)
		logrus.Info("TranslateService initialized")
	})
	return s.translateService
}

// SmartHomeService returns the service for Yandex smart home integration.
func (s *ServiceProvider) SmartHomeService() botServ.SmartHome {
	s.smartHomeOnce.Do(func() {
		s.smartHomeService = api.NewYandexSmartHomeAPI(s.smartHomeAPIEndpoint)
		logrus.Info("SmartHomeService initialized")
	})
	return s.smartHomeService
}

// GenerativeService returns the service for GenerativeModel generative model integration.
func (s *ServiceProvider) GenerativeService() (botServ.GenerativeModel, error) {
	var err error
	s.generativeOnce.Do(func() {
		s.generativeService, err = generative.ModelFactory(s.generativeName, s.generativeApiKey, s.generativeModel, 0, 1.0)
		if err != nil {
			logrus.Errorf("Failed to initialize Generative service: %v", err)
			s.generativeService = nil // Сброс при ошибке
		}
	})
	if s.generativeService == nil {
		return nil, fmt.Errorf("generative service not initialized")
	}
	logrus.Info("Generative model initialized")
	return s.generativeService, nil
}

// The Repository returns the repository for user state management.
func (s *ServiceProvider) Repository() botServ.Repository {
	s.repoOnce.Do(func() {
		s.repository = repository.NewUsersStateMap(s.storagePath)
		if err := s.repository.ReadFileToMemoryURL(); err != nil {
			logrus.Errorf("Failed to read user state from file: %v", err)
		} else {
			logrus.Info("Repository initialized and state loaded")
		}
	})
	return s.repository
}

// Handler returns the HTTP handler for OAuth operations.
func (s *ServiceProvider) Handler() (botServ.Handler, error) {
	var err error
	s.handlerOnce.Do(func() {
		s.handler, err = botHand.NewHandler(s.serverEndpoint+"/login", s.clientCert, s.clientKey, s.clientCa, s.apiKey)
		if err != nil {
			logrus.Errorf("Failed to initialize Handler: %v", err)
			s.handler = nil // Сброс при ошибке
		}
	})
	if s.handler == nil {
		return nil, fmt.Errorf("handler not initialized")
	}
	logrus.Info("Handler initialized")
	return s.handler, nil
}

// BotAPI returns the Telegram Bot API instance.
func (s *ServiceProvider) BotAPI(token string) (*tgbotapi.BotAPI, error) {
	var err error
	s.botAPIOnce.Do(func() {
		s.botAPI, err = tgbotapi.NewBotAPI(token)
		if err != nil {
			logrus.Errorf("Failed to initialize BotAPI: %v", err)
			s.botAPI = nil
		}
	})
	if s.botAPI == nil {
		return nil, fmt.Errorf("bot API not initialized")
	}

	logrus.Info("BotApi initialized")
	return s.botAPI, nil
}

// BotService returns the main Telegram bot service.
func (s *ServiceProvider) BotService(botAPI *tgbotapi.BotAPI) (*botServ.TgBotServices, error) {
	handler, err := s.Handler()
	if err != nil {
		logrus.Errorf("Failed to get handler: %v", err)
		return nil, fmt.Errorf("bot service not initialized")
	}
	generativeService, err := s.GenerativeService()
	if err != nil {
		logrus.Errorf("Failed to get generative service: %v", err)
		return nil, fmt.Errorf("bot service not initialized")
	}
	AuthURL := fmt.Sprintf("https://oauth.yandex.ru/authorize?response_type=code&client_id=%s&redirect_uri=%s/callback&state=", s.clientID, s.serverEndpoint)

	s.botServiceOnce.Do(func() {
		s.botService = botServ.NewTgBot(
			s.BoringService(),
			s.TranslateService(),
			s.SmartHomeService(),
			generativeService,
			s.Repository(),
			botAPI,
			handler,
			AuthURL,
			s.ownerID,
		)
		logrus.Info("BotService initialized")
	})
	return s.botService, nil
}
