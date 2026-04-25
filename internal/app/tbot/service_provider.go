// Package tbot provides dependency injection and service management for Telegram bot components.
// It initializes and provides access to services, repositories, and handlers required for bot operations.
package tbot

import (
	"fmt"
	"sync"

	"github.com/DenisKhanov/TgBOT/internal/tg_bot/api"
	botHand "github.com/DenisKhanov/TgBOT/internal/tg_bot/api/http"
	"github.com/DenisKhanov/TgBOT/internal/tg_bot/infra/generative"
	"github.com/DenisKhanov/TgBOT/internal/tg_bot/models"
	"github.com/DenisKhanov/TgBOT/internal/tg_bot/repository"
	botServ "github.com/DenisKhanov/TgBOT/internal/tg_bot/service"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

// ServiceProvider manages the dependency injection for Telegram bot components.
type ServiceProvider struct {
	// Services
	boringService     botServ.Boring
	translateService  botServ.Translate
	smartHomeService  botServ.SmartHome
	generativeService botServ.GenerativeModel

	// ChatStateRepository
	usersStateRepo  botServ.UsersChatStateRepository
	aiDialogHistory botServ.AIDialogHistoryRepository

	// Handler
	handler    botServ.Handler
	handlerErr error

	// Bot API
	botAPI    *tgbotapi.BotAPI
	botAPIErr error

	// Bot service
	botService    *botServ.TgBotServices
	botServiceErr error

	// API endpoints
	translateAPIEndpoint  string
	dictionaryAPIEndpoint string
	smartHomeAPIEndpoint  string

	// Config values
	serverEndpoint    string
	translateApiKey   string
	generativeName    string
	generativeApiKey  string
	generativeModel   string
	storagePath       string
	dialogStoragePath string

	//TLS file path
	clientCert string
	clientKey  string
	clientCa   string

	apiKey    string
	clientID  string
	ownerID   int64
	moviesURL string

	boringOnce       sync.Once
	translateOnce    sync.Once
	smartHomeOnce    sync.Once
	generativeOnce   sync.Once
	stateRepoOnce    sync.Once
	aiDialogRepoOnce sync.Once
	handlerOnce      sync.Once
	botAPIOnce       sync.Once
	botServiceOnce   sync.Once
}

// NewServiceProvider creates a new instance of the service provider.
func NewServiceProvider(
	translateAPIEndpoint, dictionaryAPIEndpoint, smartHomeAPIEndpoint string,
	serverEndpoint, translateApiKey,
	generativeName, generativeApiKey,
	generativeModel, storagePath, dialogStoragePath, clientCert,
	clientKey, clientCa, apiKey,
	clientID string, ownerID int64, moviesURL string,
) (*ServiceProvider, error) {
	switch {
	case translateAPIEndpoint == "":
		return nil, fmt.Errorf("translateAPIEndpoint is required")
	case dictionaryAPIEndpoint == "":
		return nil, fmt.Errorf("dictionaryAPIEndpoint is required")
	case smartHomeAPIEndpoint == "":
		return nil, fmt.Errorf("smartHomeAPIEndpoint is required")
	case serverEndpoint == "":
		return nil, fmt.Errorf("serverEndpoint is required")
	case translateApiKey == "":
		return nil, fmt.Errorf("translateApiKey is required")
	case generativeName == "":
		return nil, fmt.Errorf("generativeName is required")
	case generativeApiKey == "":
		return nil, fmt.Errorf("generativeApiKey is required")
	case generativeModel == "":
		return nil, fmt.Errorf("generativeModel is required")
	case storagePath == "":
		return nil, fmt.Errorf("storagePath is required")
	case dialogStoragePath == "":
		return nil, fmt.Errorf("dialogStoragePath is required")
	case clientCert == "":
		return nil, fmt.Errorf("clientCert is required")
	case clientKey == "":
		return nil, fmt.Errorf("clientKey is required")
	case clientCa == "":
		return nil, fmt.Errorf("clientCa is required")
	case apiKey == "":
		return nil, fmt.Errorf("apiKey is required")
	case clientID == "":
		return nil, fmt.Errorf("clientID is required")
	case ownerID == 0:
		return nil, fmt.Errorf("ownerID is required")
	case moviesURL == "":
		return nil, fmt.Errorf("moviesURL is required")
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
		dialogStoragePath:     dialogStoragePath,
		clientCert:            clientCert,
		clientKey:             clientKey,
		clientCa:              clientCa,
		apiKey:                apiKey,
		clientID:              clientID,
		ownerID:               ownerID,
		moviesURL:             moviesURL,
	}, nil
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

// The ChatStateRepository returns the usersStateRepo for user state management.
func (s *ServiceProvider) ChatStateRepository() botServ.UsersChatStateRepository {
	s.stateRepoOnce.Do(func() {
		s.usersStateRepo = repository.NewUsersStateMap(s.storagePath)
		if err := s.usersStateRepo.ReadFileToMemoryURL(); err != nil {
			logrus.Errorf("Failed to read user state from file: %v", err)
		} else {
			logrus.Info("AIDialogHistoryRepository initialized and state loaded")
		}
	})
	return s.usersStateRepo
}

// The AiDialogHistoryRepository returns the aiDialogRepo for dialog with AI history management.
func (s *ServiceProvider) AiDialogHistoryRepository() botServ.AIDialogHistoryRepository {
	s.aiDialogRepoOnce.Do(func() {
		s.aiDialogHistory = repository.NewAiDialogHistory(s.dialogStoragePath)
		if err := s.aiDialogHistory.LoadDialogFromFile(); err != nil {
			logrus.Errorf("Failed to read AI dialog history from file: %v", err)
		} else {
			logrus.Info("AIDialogHistoryRepository initialized and state loaded")
		}
	})
	return s.aiDialogHistory
}

// Handler returns the HTTP handler for OAuth operations.
func (s *ServiceProvider) Handler() (botServ.Handler, error) {
	s.handlerOnce.Do(func() {
		s.handler, s.handlerErr = botHand.NewHandler(s.serverEndpoint+"/login", s.clientCert, s.clientKey, s.clientCa, s.apiKey)
		if s.handlerErr != nil {
			s.handler = nil
		}
	})
	if s.handlerErr != nil {
		return nil, fmt.Errorf("initialize handler: %w", s.handlerErr)
	}
	if s.handler == nil {
		return nil, fmt.Errorf("handler not initialized")
	}
	return s.handler, nil
}

// BotAPI returns the Telegram Bot API instance.
func (s *ServiceProvider) BotAPI(token string) (*tgbotapi.BotAPI, error) {
	s.botAPIOnce.Do(func() {
		s.botAPI, s.botAPIErr = tgbotapi.NewBotAPI(token)
		if s.botAPIErr != nil {
			s.botAPI = nil
		}
	})
	if s.botAPIErr != nil {
		return nil, fmt.Errorf("initialize bot API: %w", s.botAPIErr)
	}
	if s.botAPI == nil {
		return nil, fmt.Errorf("bot API not initialized")
	}

	return s.botAPI, nil
}

// BotService returns the main Telegram bot service.
func (s *ServiceProvider) BotService(botAPI *tgbotapi.BotAPI) (*botServ.TgBotServices, error) {
	s.botServiceOnce.Do(func() {
		handler, err := s.Handler()
		if err != nil {
			s.botServiceErr = err
			return
		}
		generativeService, err := s.GenerativeService()
		if err != nil {
			s.botServiceErr = err
			return
		}
		AuthURL := fmt.Sprintf("https://oauth.yandex.ru/authorize?response_type=code&client_id=%s&redirect_uri=%s/callback&state=", s.clientID, s.serverEndpoint)
		s.botService = botServ.NewTgBot(
			s.BoringService(),
			s.TranslateService(),
			s.SmartHomeService(),
			generativeService,
			s.ChatStateRepository(),
			s.AiDialogHistoryRepository(),
			botAPI,
			handler,
			AuthURL,
			s.ownerID,
			s.moviesURL,
		)
	})
	if s.botServiceErr != nil {
		return nil, fmt.Errorf("initialize bot service: %w", s.botServiceErr)
	}
	if s.botService == nil {
		return nil, fmt.Errorf("bot service not initialized")
	}
	return s.botService, nil
}
