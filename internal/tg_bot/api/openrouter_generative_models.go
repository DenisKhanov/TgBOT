package api

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/wojtess/openrouter-api-go"
)

type OpenRouterAPI struct {
	client      *openrouterapigo.OpenRouterClient // Клиент для взаимодействия с API
	ctx         context.Context                   // Контекст для управления запросами
	apiKey      string                            // API-ключ (для справки или повторной инициализации)
	modelName   string                            // Версия генеративной модели
	maxTokens   int                               // Максимальное количество токенов (опционально)
	temperature float32                           // Температура для управления креативностью (опционально)
}

// NewOpenRouterAPI создает новый экземпляр OpenRouterAPI
func NewOpenRouterAPI(apiKey string, modelName string, maxTokens int, temperature float32) (*OpenRouterAPI, error) {
	// Создаем контекст
	ctx := context.Background()

	// Инициализируем клиент
	client := openrouterapigo.NewOpenRouterClient(apiKey)

	// Возвращаем структуру
	return &OpenRouterAPI{
		client:      client,
		modelName:   modelName,
		ctx:         ctx,
		apiKey:      apiKey,
		maxTokens:   maxTokens,
		temperature: temperature,
	}, nil
}

// GenerateTextMsg генерирует текст на основе переданного запроса
func (d *OpenRouterAPI) GenerateTextMsg(text string) (string, error) {

	// Формируем запрос к DeepSeek API
	chatReq := openrouterapigo.Request{
		Model: d.modelName, // Используем модель чата
		Messages: []openrouterapigo.MessageRequest{
			{Role: openrouterapigo.RoleUser, Content: text},
		},
	}

	// Отправляем запрос
	resp, err := d.client.FetchChatCompletions(chatReq)
	if err != nil {
		err = fmt.Errorf("failed to create request: %w", err)
		logrus.WithError(err).Errorf("Error creating %s request", d.modelName)
		return "", err
	}

	// Проверяем наличие ответа
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices returned from DeepSeek API")
	}

	// Извлекаем текст из ответа
	return resp.Choices[0].Message.Content, nil
}
