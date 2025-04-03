// Package http provides HTTP handlers for interacting with the Yandex Smart Home service
// and managing authorization tokens via a REST API.
package http

import (
	"github.com/DenisKhanov/TgBOT/internal/server/models"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"strconv"
)

// Service defines an interface for handling authorization tokens with Yandex Smart Home.
type Service interface {
	// GetYandexSmartHomeToken retrieves a Yandex Smart Home token using an access code
	// and saves it for the specified user.
	// Arguments:
	//   - accessCode: the access code received from Yandex.
	//   - chatID: the user's chat ID (int64).
	// Returns an error if the token cannot be retrieved or saved.
	GetYandexSmartHomeToken(accessCode string, chatID int64) error
	// GetUserToken retrieves the saved token pair (access and refresh) for the specified user.
	// Arguments:
	//   - userID: the user ID (int64).
	// Returns a Tokens struct and an error if the tokens are not found.
	GetUserToken(userID int64) (models.Tokens, error)
}

// Handler represents a structure for handling HTTP requests using a service and API key.
type Handler struct {
	service Service // Service for token operations.
	apiKey  string  // API key for request authorization.
}

// NewHandler creates a new Handler instance with the provided service and API key.
// Arguments:
//   - service: an implementation of the Service interface.
//   - apiKey: a string containing the API key for authorization.
//
// Returns a pointer to a Handler.
func NewHandler(service Service, apiKey string) *Handler {
	return &Handler{
		service: service,
		apiKey:  apiKey,
	}
}

// GetTokenFromYandex handles a request to retrieve a Yandex Smart Home token.
// Extracts the access code and chatID from query parameters, calls the service to get the token,
// and returns an HTML success page.
// Returns an appropriate HTTP status and message if errors occur.
func (h Handler) GetTokenFromYandex(c *gin.Context) {
	accessCode := c.Query("code")
	if accessCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "access code is required"})
		return
	}

	state := c.Query("state")
	chatID, err := strconv.Atoi(state)
	if err != nil {
		logrus.WithError(err).Error("failed to convert state to chatID")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state parameter"})
		return
	}

	userID := int64(chatID)
	err = h.service.GetYandexSmartHomeToken(accessCode, userID)
	if err != nil {
		logrus.WithError(err).Error("failed to get Yandex Smart Home token")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	file, err := os.Open("success.html")
	if err != nil {
		logrus.WithError(err).Error("failed to open success.html")
		c.String(http.StatusInternalServerError, "Internal server error")
		return
	}
	defer func() {
		if err = file.Close(); err != nil {
			logrus.WithError(err).Errorf("Failed to close file: %v", err)
		}
	}()

	html, err := io.ReadAll(file)
	if err != nil {
		logrus.WithError(err).Error("failed to read success.html")
		c.String(http.StatusInternalServerError, "Internal server error")
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", html)
}

// GetSavedToken retrieves the saved tokens for a user based on the chatID from the request.
// Verifies the API key in the X-API-Key header.
// On success, returns a JSON response with
// access_token, refresh_token, and expires_in.
// Returns an HTTP error status on failure.
func (h Handler) GetSavedToken(c *gin.Context) {
	// Проверка API-ключа
	if c.GetHeader("X-API-Key") != h.apiKey {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or missing API key"})
		return
	}
	// Извлечем code из url запроса
	state := c.Query("state")
	if state == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "state parameter is required"})
		return
	}

	chatID, err := strconv.Atoi(state)
	if err != nil {
		logrus.WithError(err).Error("can't convert chatID to int")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	userID := int64(chatID)
	tokenPair, err := h.service.GetUserToken(userID)
	if err != nil {
		logrus.WithError(err).Info("failed to retrieve user token")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
		"expires_in":    tokenPair.ExpiresIn,
	})
}

// Hello returns a greeting message in JSON format.
// Used to verify API functionality.
func (h Handler) Hello(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"msg": "Hello bro"})
}
