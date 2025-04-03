package config

import (
	"fmt"
	"github.com/joho/godotenv"
	"os"
)

// Config holds the application configuration parameters.
// Each field corresponds to an expected environment variable.
type Config struct {
	EnvLogsLevel     string // Log level for the application (e.g., DEBUG, INFO)
	EnvLogFileName   string // File's name for log (e.g., Server.log)
	EnvOAuthEndpoint string // Yandex's endpoint for token request
	EnvHTTPSServer   string // Address of the HTTP server
	EnvServerCert    string // Path to the server's SSL certificate
	EnvServerKey     string // Path to the server's SSL key
	EnvServerCa      string // Path to the server's CA file
	EnvClientId      string // For access to request token from Yandex Home, only to an owner
	EnvClientSecret  string // For access to request token from Yandex Home, only to an owner
	EnvApiKey        string // Key for take token TGBot agent
}

// NewConfig initializes a new Config instance by loading environment variables from a .env file.
// It returns a pointer to the Config struct and an error if any of the environment variables are missing or invalid.
func NewConfig() (*Config, error) {
	err := godotenv.Load("server.env")
	if err != nil {
		return nil, fmt.Errorf("new load .env: %w", err)
	}

	config := &Config{}
	config.EnvLogsLevel = os.Getenv("LOG_LEVEL")
	config.EnvLogFileName = os.Getenv("LOG_FILE_NAME")
	config.EnvOAuthEndpoint = os.Getenv("OAUTH_ENDPOINT")
	config.EnvHTTPSServer = os.Getenv("HTTPS_SERVER")
	config.EnvServerCert = os.Getenv("SERVER_CERT_FILE")
	config.EnvServerKey = os.Getenv("SERVER_KEY_FILE")
	config.EnvServerCa = os.Getenv("SERVER_CA_FILE")
	config.EnvClientId = os.Getenv("CLIENT_ID")
	config.EnvClientSecret = os.Getenv("CLIENT_SECRET")
	config.EnvApiKey = os.Getenv("API_KEY")

	return config, nil
}
