package generative

import (
	"fmt"
	"github.com/DenisKhanov/TgBOT/internal/tg_bot/api"
	botServ "github.com/DenisKhanov/TgBOT/internal/tg_bot/service"
)

// generativeCreator defines a function to create GenerativeModel
type generativeCreator func(apiKey, modelName string, maxTokens int, temperature float32) (botServ.GenerativeModel, error)

// generativeRegistry stores registered implementations
var generativeRegistry = map[string]generativeCreator{
	"gemini": func(apiKey, modelName string, maxTokens int, temperature float32) (botServ.GenerativeModel, error) {
		return api.NewGeminiAPI(apiKey, modelName, maxTokens, temperature)
	},
	"deepseek": func(apiKey, modelName string, maxTokens int, temperature float32) (botServ.GenerativeModel, error) {
		return api.NewDeepSeekAPI(apiKey, modelName, maxTokens, temperature)
	},
	"openrouter": func(apiKey, modelName string, maxTokens int, temperature float32) (botServ.GenerativeModel, error) {
		return api.NewOpenRouterAPI(apiKey, modelName, maxTokens, temperature)
	},
}

// ModelFactory creates a GenerativeModel implementation based on an environment variable
func ModelFactory(generativeName, apiKey, modelName string, maxTokens int, temperature float32) (botServ.GenerativeModel, error) {
	creator, exists := generativeRegistry[generativeName]
	if !exists {
		return nil, fmt.Errorf("unsupported GENERATIVE_NAME: %s (expected 'gemini' or 'deepseek')", generativeName)
	}
	return creator(apiKey, modelName, maxTokens, temperature)
}
