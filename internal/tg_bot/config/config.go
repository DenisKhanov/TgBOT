package config

import (
	"fmt"
	"github.com/joho/godotenv"
	"os"
)

// Config holds the application configuration parameters.
// Each field corresponds to an expected environment variable.
type Config struct {
	EnvLogsLevel      string // Log level for the application (e.g., DEBUG, INFO)
	EnvLogFileName    string // File's name for log (e.g., Bot.log)
	EnvStoragePath    string // File's name for log (e.g., VPNServer.log)
	EnvBotToken       string // Connection string for the database
	EnvYandexToken    string // Address of the HTTP server
	EnvServerEndpoint string //
	EnvClientCert     string // Path to the client certificate file
	EnvClientKey      string // Path to the client private key file
	EnvClientCa       string // Path to the client CA certificate file
	EnvApiKey         string // Key for get token from server
}

// NewConfig initializes a new Config instance by loading environment variables from a .env file.
// It returns a pointer to the Config struct and an error if any of the environment variables are missing or invalid.
func NewConfig() (*Config, error) {
	err := godotenv.Load("bot.env")
	if err != nil {
		return nil, fmt.Errorf("new load .env: %w", err)
	}

	config := &Config{}
	config.EnvLogsLevel = os.Getenv("LOG_LEVEL")
	config.EnvLogFileName = os.Getenv("LOG_FILE_NAME")
	config.EnvStoragePath = os.Getenv("FILE_STORAGE_PATH")
	config.EnvBotToken = os.Getenv("TOKEN_BOT")
	config.EnvYandexToken = os.Getenv("TOKEN_YANDEX")
	config.EnvServerEndpoint = os.Getenv("SERVER_ENDPOINT")
	config.EnvClientCert = os.Getenv("CLIENT_CERT_FILE")
	config.EnvClientKey = os.Getenv("CLIENT_KEY_FILE")
	config.EnvClientCa = os.Getenv("CLIENT_CA_FILE")
	config.EnvApiKey = os.Getenv("API_KEY")

	return config, nil
}
