package api

import (
	"context"
	"fmt"
	"github.com/google/generative-ai-go/genai"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"
	"time"
)

// GeminiAPI представляет структуру для работы с Gemini API
type GeminiAPI struct {
	client      *genai.Client          // Клиент для взаимодействия с API
	model       *genai.GenerativeModel // Модель для генерации контента
	ctx         context.Context        // Контекст для управления запросами
	apiKey      string                 // API-ключ (для справки или повторной инициализации)
	maxTokens   int                    // Максимальное количество токенов (опционально)
	temperature float32                // Температура для управления креативностью (опционально)
}

// NewGeminiAPI создает новый экземпляр GeminiAPI
func NewGeminiAPI(apiKey string, modelName string, maxTokens int, temperature float32) (*GeminiAPI, error) {
	// Создаем контекст
	ctx := context.Background()

	// Инициализируем клиент
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}

	// Создаем модель
	model := client.GenerativeModel(modelName)

	// Настраиваем параметры модели (опционально)
	if maxTokens > 0 {
		maxToken := int32(maxTokens)
		model.MaxOutputTokens = &maxToken
	}
	if temperature >= 0 && temperature <= 1 {
		model.Temperature = &temperature
	}

	// Возвращаем структуру
	return &GeminiAPI{
		client:      client,
		model:       model,
		ctx:         ctx,
		apiKey:      apiKey,
		maxTokens:   maxTokens,
		temperature: temperature,
	}, nil
}

func (g GeminiAPI) GenerateTextMsg(text string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	resp, err := g.model.GenerateContent(ctx, genai.Text(text))
	if err != nil {
		err = fmt.Errorf("failed to create request: %w", err)
		logrus.WithError(err).Error("Error creating Gemini request")
		return "", err
	}
	if text, ok := resp.Candidates[0].Content.Parts[0].(genai.Text); ok {
		return string(text), nil
	}
	return "", err
}
