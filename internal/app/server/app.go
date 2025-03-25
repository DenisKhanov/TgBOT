package server

import (
	"context"
	"github.com/DenisKhanov/TgBOT/internal/logcfg"
	"github.com/DenisKhanov/TgBOT/internal/server/api/http/middleware"
	"github.com/DenisKhanov/TgBOT/internal/server/config"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// App represents the application structure responsible for initializing dependencies
// and running the serverHTTPS and serverGRPC.
type App struct {
	serviceProvider *serviceProvider // The service provider for dependency injection
	config          *config.Config   // The configuration object for the application
	//tls             *tls.Config
	serverHTTPS *http.Server // The serverHTTPS instance
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

// Run starts the application and runs the serverHTTPS and serverGRPC.
func (a *App) Run() {
	a.runServer()
}

// initDeps initializes all dependencies required by the application.
func (a *App) initDeps(ctx context.Context) error {
	inits := []func(context.Context) error{
		a.initConfig,
		a.initServiceProvider,
		a.initHTTPSServer,
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
	a.serviceProvider = newServiceProvider()
	return nil
}

// initHTTPSServer initializes the http_shortener serverHTTPS with middleware and routes.
func (a *App) initHTTPSServer(_ context.Context) error {
	myHandler := a.serviceProvider.Handler(a.config.EnvOAuthEndpoint, a.config.ClientId, a.config.ClientSecret)

	// Установка переменной окружения для включения режима разработки
	gin.SetMode(gin.DebugMode)
	router := gin.Default()
	//Public middleware routers group
	publicRoutes := router.Group("/")
	publicRoutes.Use(middleware.LogrusLog())

	publicRoutes.GET("/callback", myHandler.GetTokenFromYandex)

	a.serverHTTPS = &http.Server{
		Addr:    a.config.HTTPSServer,
		Handler: router,
	}

	return nil
}

// runServer starts the gRPC + HTTP servers with graceful shutdown.
func (a *App) runServer() {
	// run HTTP server
	wd, err := os.Getwd()
	if err != nil {
		logrus.Fatal(err)
	}
	if err := a.serverHTTPS.ListenAndServeTLS(wd+a.config.ServerCert, wd+a.config.ServerKey); err != nil {
		logrus.Fatalf("Failed to start HTTPS server: %v", err)
	}
	logrus.Infof("HTTPS server started on: %s", a.config.HTTPSServer)

	// Shutdown signal with grace period of 5 seconds
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	sig := <-signalChan
	logrus.Infof("Shutting down HTTPS servers with signal : %v...", sig)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := a.serverHTTPS.Shutdown(shutdownCtx); err != nil {
		logrus.WithError(err).Error("HTTP server shutdown error")
	}

	logrus.Info("Server exited")
}
