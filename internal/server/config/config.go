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
	EnvOAuthEndpoint string //
	HTTPSServer      string // Address of the HTTP server
	ServerCert       string // Path to the server's SSL certificate
	ServerKey        string // Path to the server's SSL key
	ServerCa         string // Path to the server's CA file
	ClientId         string // For access to get token from Yandex Home only to owner
	ClientSecret     string // For access to get token from Yandex Home only to owner
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
	config.HTTPSServer = os.Getenv("HTTPS_SERVER")
	config.ServerCert = os.Getenv("SERVER_CERT_FILE")
	config.ServerKey = os.Getenv("SERVER_KEY_FILE")
	config.ServerCa = os.Getenv("SERVER_CA_FILE")
	config.ClientId = os.Getenv("CLIENT_ID")
	config.ClientSecret = os.Getenv("CLIENT_SECRET")

	return config, nil
}
