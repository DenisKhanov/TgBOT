package api

import (
	"context"
	"errors"
	"fmt"
	"github.com/DenisKhanov/TgBOT/internal/tg_bot/models"
	"github.com/sirupsen/logrus"
	"github.com/wojtess/openrouter-api-go"
	"time"
)

// OpenRouterAPI provides an interface for interacting with the OpenRouter API to generate text responses.
//
// It manages the configuration for API requests, including the API key, model name, maximum tokens, and
// temperature for controlling creativity. The struct supports both streaming and non-streaming text generation,
// as well as changing the generative model dynamically.
type OpenRouterAPI struct {
	client      *openrouterapigo.OpenRouterClient // Клиент для взаимодействия с API
	ctx         context.Context                   // Контекст для управления запросами
	apiKey      string                            // API-ключ (для справки или повторной инициализации)
	modelName   string                            // Версия генеративной модели
	maxTokens   int                               // Максимальное количество токенов (опционально)
	temperature float32                           // Температура для управления креативностью (опционально)
}

// NewOpenRouterAPI creates a new instance of OpenRouterAPI with the specified configuration.
//
// It initializes the OpenRouter client with the provided API key and sets up the context, model name,
// maximum tokens, and temperature for text generation requests.
//
// Parameters:
//   - apiKey: The API key for authenticating with the OpenRouter API.
//   - modelName: The name of the generative model to use (e.g., "deepseek-coder").
//   - maxTokens: The maximum number of tokens for generated responses (optional).
//   - temperature: The temperature value to control the creativity of the model (optional, typically between 0 and 1).
//
// Returns:
//   - *OpenRouterAPI: A pointer to the initialized OpenRouterAPI instance.
//   - error: An error if initialization fails; nil otherwise.
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

// GenerateStreamTextMsg generates a streaming text response based on the user's input and dialog history.
//
// It constructs a request with the provided text and dialog history, sends it to the OpenRouter API in streaming
// mode, and returns a channel that streams the generated text chunks. The method includes a timeout of 1 minute;
// if the API does not respond within this time, the streaming is stopped, and an error message is sent to the channel.
//
// Parameters:
//   - text: The user's input text to generate a response for.
//   - history: A slice of models.Message representing the dialog history to provide context.
//
// Returns:
//   - <-chan string: A channel that streams the generated text chunks. The channel is closed when streaming is complete
//     or an error occurs.
func (d *OpenRouterAPI) GenerateStreamTextMsg(text string, history []models.Message) <-chan string {
	// Формируем список сообщений для API, начиная с истории
	messages := make([]openrouterapigo.MessageRequest, 0, len(history)+1)
	for _, msg := range history {
		messages = append(messages, openrouterapigo.MessageRequest{
			Role:    openrouterapigo.MessageRole(msg.Role), // "user" или "assistant"
			Content: msg.Content,
		})
	}
	// Добавляем текущее сообщение пользователя
	messages = append(messages, openrouterapigo.MessageRequest{
		Role:    openrouterapigo.RoleUser,
		Content: text,
	})

	// Формируем запрос к DeepSeek API
	chatReq := openrouterapigo.Request{
		Model:    d.modelName, // Используем модель чата
		Stream:   true,
		Messages: messages,
	}

	outputChan := make(chan openrouterapigo.Response)
	processingChan := make(chan interface{})
	errChan := make(chan error)

	// Async send request
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)

	go d.client.FetchChatCompletionsStream(chatReq, outputChan, processingChan, errChan, ctx)

	textChan := make(chan string)

	go func() {
		defer cancel()
		defer close(textChan)
		for {
			select {
			case <-ctx.Done():
				err := ctx.Err()
				logrus.WithError(err).Error("Streaming stopped due to context completion")
				textChan <- fmt.Sprint("Ошибка: вышло время ожидания ответа от ИИ")
				return
			case err := <-errChan:
				if err != nil {
					logrus.WithError(err).Error("Error during streaming from OpenRouter")
					textChan <- fmt.Sprintf("Ошибка: %v", err)
					return
				}
			case <-processingChan:
			case output, ok := <-outputChan:
				if !ok {
					logrus.Info("Streaming completed")
					return
				}
				if len(output.Choices) > 0 {
					content := output.Choices[0].Delta.Content
					if content != "" {
						logrus.WithField("chunk", content).Debug("Received stream chunk")
					}
					textChan <- content
				}
			}
		}
	}()
	return textChan
}

// GenerateTextMsg generates a non-streaming text response based on the user's input.
//
// It sends a request to the OpenRouter API with the user's text and returns the generated response as a single string.
// The request is made in non-streaming mode, meaning the entire response is generated and returned at once.
//
// Parameters:
//   - text: The user's input text to generate a response for.
//
// Returns:
//   - string: The generated text response.
//   - error: An error if the request fails or no response is returned from the API; nil otherwise.
func (d *OpenRouterAPI) GenerateTextMsg(text string) (string, error) {
	// Формируем запрос к DeepSeek API
	chatReq := openrouterapigo.Request{
		Model:  d.modelName, // Используем модель чата
		Stream: false,
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

// ChangeGenerativeModelName changes the generative model used by the OpenRouterAPI instance.
//
// It validates the new model name by sending a test request to the OpenRouter API with a simple message.
// If the model is accessible and responds successfully, the model name is updated. The test request uses
// minimal tokens (MaxTokens: 10) and a temperature of 0.7 to ensure a quick response. If the model name is
// empty or the test request fails, an error is returned.
//
// Parameters:
//   - modelName: The name of the new generative model to use (e.g., "deepseek-coder").
//
// Returns:
//   - error: An error if the model name is empty, the test request fails, or the model does not respond; nil on success.
func (d *OpenRouterAPI) ChangeGenerativeModelName(modelName string) error {
	if modelName == "" {
		return errors.New("model name can't be empty")
	}
	request := openrouterapigo.Request{
		Model: modelName,
		Messages: []openrouterapigo.MessageRequest{
			{Role: openrouterapigo.RoleUser, Content: "Hello, are you working?"},
		},
		Stream:      false,
		MaxTokens:   10, // Минимальное количество токенов для теста
		Temperature: 0.7,
	}

	logrus.WithField("model", modelName).Info("Checking if model is working")
	resp, err := d.client.FetchChatCompletions(request)
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
