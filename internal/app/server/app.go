// Package server provides the main application structure and logic for running an HTTPS server.
// It integrates configuration, dependency injection, and graceful shutdown for handling Yandex OAuth requests.
package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/DenisKhanov/TgBOT/internal/logcfg"
	"github.com/DenisKhanov/TgBOT/internal/server/api/http/middleware"
	"github.com/DenisKhanov/TgBOT/internal/server/config"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

// App represents the application structure responsible for initializing dependencies
// and running the HTTPS server.
type App struct {
	serviceProvider *serviceProvider // Service provider for dependency injection.
	config          *config.Config   // Configuration object for the application.
	httpsServer     *http.Server     // HTTPS server instance.
}

// NewApp creates a new instance of the App with initialized dependencies.
// Arguments:
//   - ctx: context for dependency initialization.
//
// Returns a pointer to App and an error if initialization fails.
func NewApp(ctx context.Context) (*App, error) {
	app := &App{}
	if err := app.initDeps(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize app: %w", err)
	}
	return app, nil
}

// Run starts the application and runs the HTTPS server.
func (a *App) Run() {
	a.runServer()
}

// initDeps initializes all dependencies required by the application.
// Arguments:
//   - ctx: context for initialization operations.
//
// Returns an error if any dependency initialization fails.
func (a *App) initDeps(ctx context.Context) error {
	inits := []func(context.Context) error{
		a.initConfig,
		a.initServiceProvider,
		a.initHTTPSServer,
	}

	for _, f := range inits {
		if err := f(ctx); err != nil {
			return err
		}
	}
	return nil
}

// initConfig initializes the application configuration.
// Arguments:
//   - ctx: context (unused but included for consistency).
//
// Returns an error if configuration loading fails.
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
// Arguments:
//   - ctx: context (unused but included for consistency).
//
// Returns an error if service provider initialization fails.
func (a *App) initServiceProvider(_ context.Context) error {
	a.serviceProvider = newServiceProvider(a.config.EnvOAuthEndpoint, a.config.EnvClientId, a.config.EnvClientSecret, a.config.EnvApiKey)
	logrus.Info("Service provider initialized")
	return nil
}

// initHTTPSServer initializes the HTTPS server with middleware and routes.
// It configures endpoints for Yandex OAuth 2.0 handling.
// Arguments:
//   - ctx: context (unused but included for consistency).
//
// Returns an error if server initialization fails.
func (a *App) initHTTPSServer(_ context.Context) error {
	myHandler := a.serviceProvider.Handler()

	ginMode := gin.DebugMode
	if a.config.EnvLogsLevel == "prod" {
		ginMode = gin.ReleaseMode
	}

	gin.SetMode(ginMode)

	router := gin.Default()
	publicRoutes := router.Group("/")
	publicRoutes.Use(middleware.LogrusLog())

	publicRoutes.GET("/callback", myHandler.GetTokenFromYandex)
	publicRoutes.GET("/login", myHandler.GetSavedToken)
	publicRoutes.GET("/", myHandler.Hello) // The endpoint for test

	a.httpsServer = &http.Server{
		Addr:    a.config.EnvHTTPSServer,
		Handler: router,
	}

	logrus.Info("HTTPS server initialized with routes")
	return nil
}

// runServer starts the HTTPS server with TLS and handles graceful shutdown.
// It listens for termination signals and ensures proper server closure.
func (a *App) runServer() {
	wd, err := os.Getwd()
	if err != nil {
		logrus.WithError(err).Error("Failed to get working directory, using relative paths")
		wd = "."
	}

	certPath := filepath.Join(wd, a.config.EnvServerCert)
	keyPath := filepath.Join(wd, a.config.EnvServerKey)

	if _, err = os.Stat(certPath); os.IsNotExist(err) {
		logrus.Fatalf("Certificate file not found: %s", certPath)
	}
	if _, err = os.Stat(keyPath); os.IsNotExist(err) {
		logrus.Fatalf("Key file not found: %s", keyPath)
	}

	go func() {
		logrus.Infof("Starting HTTPS server on %s with TLS", a.config.EnvHTTPSServer)
		if err = a.httpsServer.ListenAndServeTLS(certPath, keyPath); err != nil && !errors.Is(http.ErrServerClosed, err) {
			logrus.Fatalf("Failed to start HTTPS server: %v", err)
		}
	}()

	// Shutdown signal with a grace period of 5 seconds
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	sig := <-signalChan
	logrus.Infof("Received shutdown signal: %v", sig)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := a.httpsServer.Shutdown(shutdownCtx); err != nil {
		logrus.WithError(err).Error("HTTPS server shutdown failed")
	} else {
		logrus.Info("HTTPS server shut down successfully")
	}

	logrus.Info("Application terminated")
}
