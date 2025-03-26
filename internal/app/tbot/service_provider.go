package tbot

import (
	"github.com/DenisKhanov/TgBOT/internal/tg_bot/api"
	botHand "github.com/DenisKhanov/TgBOT/internal/tg_bot/api/http"
	"github.com/DenisKhanov/TgBOT/internal/tg_bot/models"
	"github.com/DenisKhanov/TgBOT/internal/tg_bot/repository"
	botServ "github.com/DenisKhanov/TgBOT/internal/tg_bot/service"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

// ServiceProvider manages the dependency injection for Telegram bot components.
type ServiceProvider struct {
	// Services
	boringService    botServ.Boring
	translateService botServ.YandexTranslate
	smartHomeService botServ.YandexSmartHome

	// Repository
	repository botServ.Repository

	// Handler
	handler botServ.Handler

	// Bot API
	botAPI *tgbotapi.BotAPI

	// Bot service
	botService *botServ.TgBotServices

	// API endpoints
	translateAPI  string
	dictionaryAPI string
	iotAPI        string

	// Config values
	serverEndpoint string
	yandexToken    string
	storagePath    string

	//TLS file path
	clientCert string
	clientKey  string
	clientCa   string

	apiKey string
}

//TODO: разобраться с переносом конфога в сервис провайдер

// NewServiceProvider creates a new instance of the service provider.
func NewServiceProvider(
	translateAPI, dictionaryAPI, iotAPI string,
	serverEndpoint, yandexToken, storagePath, clientCert, clientKey, clientCa, apiKey string,
) *ServiceProvider {
	return &ServiceProvider{
		translateAPI:   translateAPI,
		dictionaryAPI:  dictionaryAPI,
		iotAPI:         iotAPI,
		serverEndpoint: serverEndpoint,
		yandexToken:    yandexToken,
		storagePath:    storagePath,
		clientCert:     clientCert,
		clientKey:      clientKey,
		clientCa:       clientCa,
		apiKey:         apiKey,
	}
}

// BoringService returns the service for activity suggestions.
func (s *ServiceProvider) BoringService() botServ.Boring {
	if s.boringService == nil {
		s.boringService = botServ.NewBoringAPI(models.ActivitiesRU)
	}
	return s.boringService
}

// TranslateService returns the service for translation.
func (s *ServiceProvider) TranslateService() botServ.YandexTranslate {
	if s.translateService == nil {
		s.translateService = api.NewYandexAPI(
			s.translateAPI,
			s.dictionaryAPI,
			s.yandexToken,
		)
	}
	return s.translateService
}

// SmartHomeService returns the service for Yandex smart home integration.
func (s *ServiceProvider) SmartHomeService() botServ.YandexSmartHome {
	if s.smartHomeService == nil {
		s.smartHomeService = api.NewYandexSmartHomeAPI(s.iotAPI)
	}
	return s.smartHomeService
}

// Repository returns the repository for user state management.
func (s *ServiceProvider) Repository() botServ.Repository {
	if s.repository == nil {
		s.repository = repository.NewUsersStateMap(s.storagePath)
		err := s.repository.ReadFileToMemoryURL()
		if err != nil {
			logrus.Error("Failed to read user state from file:", err)
		}
	}
	return s.repository
}

// Handler returns the HTTP handler for OAuth operations.
func (s *ServiceProvider) Handler() botServ.Handler {
	if s.handler == nil {
		s.handler = botHand.NewHandler(s.serverEndpoint, s.clientCert, s.clientKey, s.clientCa, s.apiKey)
	}
	return s.handler
}

// BotAPI returns the Telegram Bot API instance.
func (s *ServiceProvider) BotAPI(token string) (*tgbotapi.BotAPI, error) {
	if s.botAPI == nil {
		botAPI, err := tgbotapi.NewBotAPI(token)
		if err != nil {
			return nil, err
		}
		s.botAPI = botAPI
	}
	return s.botAPI, nil
}

// BotService returns the main Telegram bot service.
func (s *ServiceProvider) BotService(botAPI *tgbotapi.BotAPI) *botServ.TgBotServices {
	if s.botService == nil {
		s.botService = botServ.NewTgBot(
			s.BoringService(),
			s.TranslateService(),
			s.SmartHomeService(),
			s.Repository(),
			botAPI,
			s.Handler(),
		)
	}
	return s.botService
}
