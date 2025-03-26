package tbot

import (
	"context"
	"github.com/DenisKhanov/TgBOT/internal/logcfg"
	"github.com/DenisKhanov/TgBOT/internal/tg_bot/config"
	"github.com/DenisKhanov/TgBOT/internal/tg_bot/repository"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// App represents the application structure responsible for initializing dependencies
// and running the Telegram bot.
type App struct {
	serviceProvider *ServiceProvider // The service provider for dependency injection
	config          *config.Config   // The configuration object for the application
}

// NewApp creates a new instance of the application.
func NewApp(ctx context.Context) (*App, error) {
	app := &App{}
	err := app.initDeps(ctx)
	if err != nil {
		return nil, err
	}
	return app, nil
}

// Run starts the application and runs the Telegram bot.
func (a *App) Run() {
	a.runTelegramBot()
}

// initDeps initializes all dependencies required by the application.
func (a *App) initDeps(ctx context.Context) error {
	inits := []func(context.Context) error{
		a.initConfig,
		a.initServiceProvider,
	}

	for _, f := range inits {
		err := f(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

// initConfig initializes the application configuration.
func (a *App) initConfig(_ context.Context) error {
	cfg, err := config.NewConfig()
	if err != nil {
		return err
	}
	a.config = cfg
	logcfg.RunLoggerConfig(a.config.EnvLogsLevel, a.config.EnvLogFileName)
	return nil
}

// initServiceProvider initializes the service provider for dependency injection.
func (a *App) initServiceProvider(_ context.Context) error {
	const (
		YandexTranslateAPI  = "https://translate.api.cloud.yandex.net/translate/v2/translate" // translate
		YandexDictionaryAPI = "https://translate.api.cloud.yandex.net/translate/v2/detect"    // detect language
		YandexIOTAPI        = "https://api.iot.yandex.net"
	)

	a.serviceProvider = NewServiceProvider(
		YandexTranslateAPI,
		YandexDictionaryAPI,
		YandexIOTAPI,
		a.config.EnvServerEndpoint,
		a.config.EnvYandexToken,
		a.config.EnvStoragePath,
		a.config.EnvClientCert,
		a.config.EnvClientKey,
		a.config.EnvClientCa,
		a.config.EnvApiKey,
	)
	return nil
}

// runTelegramBot starts the Telegram bot with graceful shutdown.
func (a *App) runTelegramBot() {
	// Initialize bot API
	botAPI, err := a.serviceProvider.BotAPI(a.config.EnvBotToken)
	if err != nil {
		logrus.Fatalf("[ERROR] can't make telegram bot, %v", err)
	}
	botAPI.Debug = true
	logrus.Infof("Bot API created successfully for %s", botAPI.Self.UserName)

	// Initialize bot service
	myBot := a.serviceProvider.BotService(botAPI)

	// Setup ticker for periodic state saving
	ticker := time.NewTicker(time.Minute * 5) // Ticker for saving user state to file every 5 minutes
	defer ticker.Stop()

	// Setup signal handling for graceful shutdown
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// Configure updates channel
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60 // seconds timeout
	updates := botAPI.GetUpdatesChan(updateConfig)

	// Get repository as UsersState for type assertion
	usersState, ok := a.serviceProvider.Repository().(*repository.UsersState)
	if !ok {
		logrus.Fatal("Failed to cast repository to UsersState")
	}

	// Main loop
	for {
		select {
		case sig := <-signalChan: // Wait for shutdown signal
			logrus.Infof("Received %v signal, shutting down bot...", sig)
			if err = myBot.Repository.SaveBatchToFile(); err != nil {
				logrus.Error("Error while saving state on shutdown: ", err)
			}
			logrus.Info("Shutting down main loop...")
			os.Exit(1)

		case <-ticker.C: // Ticker event
			if err = myBot.Repository.SaveBatchToFile(); err != nil {
				logrus.Error("Error while saving state on ticker: ", err)
			}
		case update, ok := <-updates: // Telegram updates
			if !ok {
				logrus.Errorf("telegram update chan closed")
			}
			if update.InlineQuery != nil {
				myBot.HandleInlineQuery(botAPI, update.InlineQuery)
			} else {
				myBot.UpdateProcessing(&update, usersState)
			}
		}
	}
}
