// Package tbot provides the main application structure and logic for running a Telegram bot.
// It integrates configuration, dependency injection, and graceful shutdown for bot operations.
package tbot

import (
	"context"
	"fmt"
	"github.com/DenisKhanov/TgBOT/internal/logcfg"
	"github.com/DenisKhanov/TgBOT/internal/tg_bot/config"
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
	if err := app.initDeps(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize app: %w", err)
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
		return fmt.Errorf("failed to initialize config: %w", err)
	}
	a.config = cfg
	logcfg.RunLoggerConfig(a.config.EnvLogsLevel, a.config.EnvLogFileName)
	logrus.Infof("Configuration initialized with log level: %s", a.config.EnvLogsLevel)
	return nil
}

// initServiceProvider initializes the service provider for dependency injection.
func (a *App) initServiceProvider(_ context.Context) error {

	a.serviceProvider = NewServiceProvider(
		a.config.EnvTranslateApiEndpoint,
		a.config.EnvDictionaryDetectApiEndpoint,
		a.config.EnvSmartHomeEndpoint,
		a.config.EnvServerEndpoint,
		a.config.EnvTranslateApiKey,
		a.config.EnvGenerativeName,
		a.config.EnvGenerativeApiKey,
		a.config.EnvGenerativeModel,
		a.config.EnvStoragePath,
		a.config.EnvDialogStoragePath,
		a.config.EnvClientCert,
		a.config.EnvClientKey,
		a.config.EnvClientCa,
		a.config.EnvApiKey,
		a.config.EnvClientID,
		a.config.EnvOwnerID,
	)
	logrus.Info("Service provider initialized")
	return nil
}

// runTelegramBot starts the Telegram bot with graceful shutdown.
func (a *App) runTelegramBot() {
	if a.config.EnvBotToken == "" {
		logrus.Fatal("Bot token is not set in configuration")
	}

	botAPI, err := a.serviceProvider.BotAPI(a.config.EnvBotToken)
	if err != nil {
		logrus.Fatalf("Failed to initialize Telegram bot API: %v", err)
	}
	botAPI.Debug = true
	logrus.Infof("Bot API created successfully for %s", botAPI.Self.UserName)

	// Initialize bot service
	myBot, err := a.serviceProvider.BotService(botAPI)
	if err != nil {
		logrus.Fatalf("Failed to initialize Telegram bot service provider: %v", err)
	}
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		for update := range updates {
			if update.InlineQuery != nil {
				myBot.HandleInlineQuery(botAPI, update.InlineQuery)
			} else {
				myBot.UpdateProcessing(&update)
			}
		}
		logrus.Info("Update channel closed")
	}()

	// Main loop
	for {
		select {
		case sig := <-signalChan:
			logrus.Infof("Received signal %v, initiating shutdown", sig)
			cancel()
			if err = myBot.StateRepo.SaveBatchToFile(); err != nil {
				logrus.Errorf("Failed to save state during shutdown: %v", err)
			}
			if err = myBot.AIDialogRepo.SaveBatchToFile(); err != nil {
				logrus.Errorf("Failed to save dialog history during shutdown: %v", err)
			}
			botAPI.StopReceivingUpdates()
			logrus.Info("Telegram bot shut down successfully")
			return

		case <-ticker.C:
			if err = myBot.StateRepo.SaveBatchToFile(); err != nil {
				logrus.Errorf("Failed to save state on ticker: %v", err)
			}
			if err = myBot.AIDialogRepo.SaveBatchToFile(); err != nil {
				logrus.Errorf("Failed to save dialog history on ticker: %v", err)
			}
			if err == nil {
				logrus.Info("User state & AI dialog history saved successfully")
			}
		case <-ctx.Done():
			logrus.Info("Main loop terminated")
			return
		}
	}
}
