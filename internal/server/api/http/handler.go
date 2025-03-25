package http

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	"strconv"
)

//go:generate mockgen -source=handlers.go -destination=mocks/handlers_mock.go -package=mocks
type Service interface {
	GetYandexSmartHomeToken(accessCode string, chatID int) error
	GetUserToken(userID int) (accessToken string, err error)
}
type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{
		service: service,
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
	err = h.service.GetYandexSmartHomeToken(accessCode, chatID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{})
}
