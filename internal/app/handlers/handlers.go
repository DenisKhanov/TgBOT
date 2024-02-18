package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

//go:generate mockgen -source=handlers.go -destination=mocks/handlers_mock.go -package=mocks
type Service interface {
	GetYandexSmartHomeToken(accessCode string)
}
type Handlers struct {
	service Service
}

func NewHandlers(service Service) *Handlers {
	return &Handlers{
		service: service,
	}
}
func (h Handlers) LogIn(c *gin.Context) {
	// Извлечем access_token из url запроса
	accessCode := c.Query("code")
	logrus.Info("Access code:", accessCode)
	h.service.GetYandexSmartHomeToken(accessCode)

}
