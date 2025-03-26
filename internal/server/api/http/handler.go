package http

import (
	"fmt"
	"github.com/DenisKhanov/TgBOT/internal/server/models"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	"strconv"
)

//go:generate mockgen -source=handlers.go -destination=mocks/handlers_mock.go -package=mocks
type Service interface {
	GetYandexSmartHomeToken(accessCode string, chatID int64) error
	GetUserToken(userID int64) (models.Tokens, error)
}
type Handler struct {
	service Service
	apiKey  string
}

func NewHandler(service Service, apiKey string) *Handler {
	return &Handler{
		service: service,
		apiKey:  apiKey,
	}
}

func (h Handler) GetTokenFromYandex(c *gin.Context) {
	// Извлечем code из url запроса
	accessCode := c.Query("code")
	state := c.Query("state")
	chatID, err := strconv.Atoi(state)
	if err != nil {
		logrus.Fatalf("can't convert chatID to int: %v", err)
	}
	userID := int64(chatID)
	err = h.service.GetYandexSmartHomeToken(accessCode, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"msg": "Authorisation success"})
}

func (h Handler) GetSavedToken(c *gin.Context) {
	// Проверка API-ключа
	if c.GetHeader("X-API-Key") != h.apiKey {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User wasn't authorized"})
		return
	}
	// Извлечем code из url запроса
	state := c.Query("state")
	chatID, err := strconv.Atoi(state)
	if err != nil {
		logrus.Fatalf("can't convert chatID to int: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
	userID := int64(chatID)
	tokenPair, err := h.service.GetUserToken(userID)
	if err != nil {
		logrus.Info(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fmt.Println("TOKEN PAIR")
	fmt.Println(tokenPair.AccessToken, "\n", tokenPair.RefreshToken, "\n", tokenPair.ExpiresIn)
	c.JSON(http.StatusOK, gin.H{
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
		"expires_in":    tokenPair.ExpiresIn,
	})
}
