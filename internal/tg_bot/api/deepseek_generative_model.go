package api

import (
	"context"
	"errors"
	"fmt"
	"github.com/DenisKhanov/TgBOT/internal/tg_bot/models"
	"github.com/go-deepseek/deepseek"
	"github.com/go-deepseek/deepseek/request"
	"github.com/sirupsen/logrus"
	"time"
)

type DeepSeekAPI struct {
	client      deepseek.Client // Клиент для взаимодействия с API
	ctx         context.Context // Контекст для управления запросами
	apiKey      string          // API-ключ (для справки или повторной инициализации)
	modelName   string          // Версия генеративной модели
	maxTokens   int             // Максимальное количество токенов (опционально)
	temperature float32         // Температура для управления креативностью (опционально)
}

// NewDeepSeekAPI создает новый экземпляр NewDeepSeekAPI
func NewDeepSeekAPI(apiKey string, modelName string, maxTokens int, temperature float32) (*DeepSeekAPI, error) {
	// Создаем контекст
	ctx := context.Background()

	// Инициализируем клиент
	client, err := deepseek.NewClient(apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create DeepSeek client: %w", err)
	}

	// Возвращаем структуру
	return &DeepSeekAPI{
		client:      client,
		modelName:   modelName,
		ctx:         ctx,
		apiKey:      apiKey,
		maxTokens:   maxTokens,
		temperature: temperature,
	}, nil
}

func (d *DeepSeekAPI) GenerateStreamTextMsg(text string, history []models.Message) <-chan string {
	_, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return make(<-chan string)
}

// GenerateTextMsg генерирует текст на основе переданного запроса
func (d *DeepSeekAPI) GenerateTextMsg(text string) (string, error) {
	// Создаем контекст с таймаутом 15 секунд
	ctx, cancel := context.WithTimeout(d.ctx, 15*time.Second)
	defer cancel()

	// Формируем запрос к DeepSeek API
	chatReq := &request.ChatCompletionsRequest{
		Model:  d.modelName, // Используем модель чата DeepSeek
		Stream: false,       // Отключаем стриминг
		Messages: []*request.Message{
			{Role: "user", Content: text},
		},
		MaxTokens:   d.maxTokens,    // Устанавливаем максимальное количество токенов
		Temperature: &d.temperature, // Устанавливаем температуру
	}

	// Отправляем запрос
	resp, err := d.client.CallChatCompletionsChat(ctx, chatReq)
	if err != nil {
		err = fmt.Errorf("failed to create request: %w", err)
		logrus.WithError(err).Error("Error creating DeepSeek request")
		return "", err
	}

	// Проверяем наличие ответа
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices returned from DeepSeek API")
	}

	// Извлекаем текст из ответа
	return resp.Choices[0].Message.Content, nil
}

func (d *DeepSeekAPI) ChangeGenerativeModelName(modelName string) error {
	if modelName == "" {
		return errors.New("model name can't be empty")
	}
	ctx, cancel := context.WithTimeout(d.ctx, 15*time.Second)
	defer cancel()

	completionsRequest := &request.ChatCompletionsRequest{
		Model: modelName,
		Messages: []*request.Message{
			{Role: "user", Content: "Hello, are you working?"},
		},
		Stream:    false,
		MaxTokens: 10, // Минимальное количество токенов для теста
	}

	logrus.WithField("model", modelName).Info("Checking if model is working")
	resp, err := d.client.CallChatCompletionsChat(ctx, completionsRequest)
	if err != nil {
		return fmt.Errorf("failed to check model %s: %w", modelName, err)
	}

	if len(resp.Choices) == 0 {
		return fmt.Errorf("no choices returned from model %s", modelName)
	}
	d.modelName = modelName
	logrus.WithField("model", modelName).Info("Model is changed and working")
	return nil

}
