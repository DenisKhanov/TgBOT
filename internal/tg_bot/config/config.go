package config

import (
	"fmt"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"os"
	"strconv"
)

// Config holds the application configuration parameters.
// Each field corresponds to an expected environment variable.
type Config struct {
	EnvLogsLevel                   string // Log level for the application (e.g., DEBUG, INFO)
	EnvLogFileName                 string // File's name for log (e.g., Bot.log)
	EnvStoragePath                 string // File's name for log (e.g., VPNServer.log)
	EnvBotToken                    string // Telegram Bot Token for authentication with the Telegram API
	EnvTranslateApiEndpoint        string // Endpoint URL for the translation API (e.g., Yandex Translate API)
	EnvDictionaryDetectApiEndpoint string // Endpoint URL for the dictionary/detect language API (e.g., for language detection)
	EnvSmartHomeEndpoint           string // Endpoint URL for the smart home API (e.g., Yandex Smart Home API)
	EnvTranslateApiKey             string // API Key for the translation service (e.g., Yandex Translate API)
	EnvGenerativeName              string // Name of the generative AI provider to use (e.g., "gemini" or "deepseek")
	EnvGenerativeApiKey            string // API Key for the generative AI service (e.g., Gemini or DeepSeek API)
	EnvGenerativeModel             string // Model name for the generative AI (e.g., "gemini-2.0-flash" for Gemini)
	EnvServerEndpoint              string // Server endpoint URL for external API or service communication
	EnvClientCert                  string // Path to the client certificate file
	EnvClientKey                   string // Path to the client private key file
	EnvClientCa                    string // Path to the client CA certificate file
	EnvApiKey                      string // Key for get access to get token from server
	EnvClientID                    string // Program ID for OAUth URL
	EnvOwnerID                     int64  // TG owner's ID for get access to using smart home
}

// NewConfig initializes a new Config instance by loading environment variables from a .env file.
// It returns a pointer to the Config struct and an error if any of the environment variables are missing or invalid.
func NewConfig() (*Config, error) {
	var id int64
	err := godotenv.Load("bot.env")
	if err != nil {
		return nil, fmt.Errorf("new load .env: %w", err)
	}

	config := &Config{}
	config.EnvLogsLevel = os.Getenv("LOG_LEVEL")
	config.EnvLogFileName = os.Getenv("LOG_FILE_NAME")
	config.EnvStoragePath = os.Getenv("FILE_STORAGE_PATH")
	config.EnvBotToken = os.Getenv("TOKEN_BOT")
	config.EnvTranslateApiEndpoint = os.Getenv("TRANSLATE_API_ENDPOINT")
	config.EnvDictionaryDetectApiEndpoint = os.Getenv("DICTIONARY_DETECT_API_ENDPOINT")
	config.EnvSmartHomeEndpoint = os.Getenv("SMART_HOME_ENDPOINT")
	config.EnvTranslateApiKey = os.Getenv("TRANSLATE_API_KEY")
	config.EnvGenerativeName = os.Getenv("GENERATIVE_NAME")
	config.EnvGenerativeApiKey = os.Getenv("GENERATIVE_API_KEY")
	config.EnvGenerativeModel = os.Getenv("GENERATIVE_MODEL")
	config.EnvServerEndpoint = os.Getenv("SERVER_ENDPOINT")
	config.EnvClientCert = os.Getenv("CLIENT_CERT_FILE")
	config.EnvClientKey = os.Getenv("CLIENT_KEY_FILE")
	config.EnvClientCa = os.Getenv("CLIENT_CA_FILE")
	config.EnvApiKey = os.Getenv("API_KEY")
	config.EnvClientID = os.Getenv("CLIENT_ID")
	if id, err = strconv.ParseInt(os.Getenv("OWNER_ID"), 10, 64); err == nil {
		config.EnvOwnerID = id
	} else {
		logrus.WithError(err).Error("Failed to parse OWNER_ID from environment")
		return nil, err
	}

	return config, nil
}
