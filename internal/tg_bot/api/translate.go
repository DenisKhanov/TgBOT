// Package api provides an implementation for interacting with the Yandex Translate and Detect Language APIs.
// It supports text translation and language detection.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"time"
)

// Yandex defines the interface for Yandex API operations.
type Yandex interface {
	TranslateAPI(text string) (string, error)  // Translates text to a target language.
	DetectLangAPI(text string) (string, error) // Detects the language of the given text.
}

// YandexAPI manages interactions with the Yandex Translate and Detect Language APIs.
type YandexAPI struct {
	token        string       // Authentication token for API requests.
	endTranslate string       // Endpoint URL for the Translate API.
	endDetect    string       // Endpoint URL for the Detect Language API.
	client       *http.Client //HTTP client
}

// GlossaryPair represents a source and translated text pair for glossary configuration.
type GlossaryPair struct {
	SourceText     string `json:"sourceText"`
	TranslatedText string `json:"translatedText"`
}

// GlossaryData holds an array of GlossaryPair objects.
type GlossaryData struct {
	GlossaryPairs []GlossaryPair `json:"glossaryPairs"`
}

// GlossaryConfig contains GlossaryData.
type GlossaryConfig struct {
	GlossaryData GlossaryData `json:"glossaryData"`
}

// TranslateRequest is the top-level structure for the translation request.
type TranslateRequest struct {
	SourceLanguageCode string         `json:"sourceLanguageCode"` // Source language code (e.g., "en").
	TargetLanguageCode string         `json:"targetLanguageCode"` // Target language code (e.g., "ru").
	Format             string         `json:"format"`             // Format of the text (e.g., "PLAIN_TEXT").
	Texts              []string       `json:"texts"`              // List of texts to translate.
	FolderId           string         `json:"folderId"`           // Folder ID (optional).
	Model              string         `json:"model"`              // Translation model (optional).
	GlossaryConfig     GlossaryConfig `json:"glossaryConfig"`     // Glossary configuration (optional).
	Speller            bool           `json:"speller"`            // Enable spell checking.
}

// Translation represents a single translation result.
type Translation struct {
	Text                 string `json:"text"`
	DetectedLanguageCode string `json:"detectedLanguageCode"`
}

// TranslateResponse contains the response from the Translate API.
type TranslateResponse struct {
	Translations []Translation `json:"translations"`
}

// DetectLangReq is the structure for a language detection request.
type DetectLangReq struct {
	Text              string   `json:"text"`
	LanguageCodeHints []string `json:"language"`
}

// DetectLangRes contains the response from the Detect Language API.
type DetectLangRes struct {
	LanguageCode string `json:"languageCode"`
}

// NewYandexAPI creates a new instance of YandexAPI with the specified endpoints and token.
// Arguments:
//   - endTranslate: endpoint URL for the Translate API.
//   - endDetect: endpoint URL for the Detect Language API.
//   - Token: authentication token for API requests.
//
// Returns a pointer to a YandexAPI.
func NewYandexAPI(endTranslate, endDetect, token string) *YandexAPI {
	return &YandexAPI{
		endTranslate: endTranslate,
		endDetect:    endDetect,
		token:        token,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// TranslateAPI translates the given text to a target language (ru or en based on detection).
// Arguments:
//   - text: the text to translate.
//
// Returns the translated text or an error if the request fails.
func (y *YandexAPI) TranslateAPI(text string) (string, error) {
	detectedLang, err := y.DetectLangAPI(text)
	if err != nil {
		logrus.WithError(err).Error("Failed to detect language")
		return "", fmt.Errorf("language detection failed: %w", err)
	}

	targetLang := "ru"
	if detectedLang == "ru" {
		targetLang = "en"
	}
	reqBody := TranslateRequest{
		SourceLanguageCode: detectedLang,
		TargetLanguageCode: targetLang,
		Format:             "PLAIN_TEXT",
		Texts:              []string{text},
		Speller:            true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), y.client.Timeout)
	defer cancel()

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		err = fmt.Errorf("failed to marshal request body: %w", err)
		logrus.WithError(err).Error("Error preparing TranslateAPI request")
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, y.endTranslate, bytes.NewBuffer(jsonBody))
	if err != nil {
		err = fmt.Errorf("failed to create request: %w", err)
		logrus.WithError(err).Error("Error creating TranslateAPI request")
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", y.token)

	res, err := y.client.Do(req)
	if err != nil {
		logrus.WithError(err).Errorf("Failed to execute TranslateAPI request to %s", y.endTranslate)
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		if err = res.Body.Close(); err != nil {
			logrus.WithError(err).Errorf("Failed to close response body: %v", err)
		}
	}()

	if res.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(res.Body)
		err = fmt.Errorf("unexpected status code: %d, body: %s", res.StatusCode, string(data))
		logrus.WithError(err).Errorf("TranslateAPI failed with status: %s", res.Status)
		return "", err
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		logrus.WithError(err).Error("Failed to read TranslateAPI response")
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	var response TranslateResponse
	if err = json.Unmarshal(data, &response); err != nil {
		logrus.WithError(err).Error("Failed to unmarshal TranslateAPI response")
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(response.Translations) == 0 {
		err = fmt.Errorf("no translations returned")
		logrus.WithError(err).Error("TranslateAPI response is empty")
		return "", err
	}

	result := response.Translations[0].Text
	logrus.Infof("Translated text from %s to %s: %s", detectedLang, targetLang, result)
	return result, nil
}

// DetectLangAPI detects the language of the given text.
// Arguments:
//   - text: the text to detect the language for.
//
// Returns the detected language code or an error if the request fails.
func (y *YandexAPI) DetectLangAPI(text string) (string, error) {
	reqBody := DetectLangReq{
		Text:              text,
		LanguageCodeHints: []string{"ru", "en"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), y.client.Timeout)
	defer cancel()

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		err = fmt.Errorf("failed to marshal request body: %w", err)
		logrus.WithError(err).Error("Error preparing DetectLangAPI request")
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, y.endDetect, bytes.NewBuffer(jsonBody))
	if err != nil {
		err = fmt.Errorf("failed to create request: %w", err)
		logrus.WithError(err).Error("Error creating DetectLangAPI request")
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", y.token)

	res, err := y.client.Do(req)
	if err != nil {
		logrus.WithError(err).Errorf("Failed to execute DetectLangAPI request to %s", y.endDetect)
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		if err = res.Body.Close(); err != nil {
			logrus.WithError(err).Errorf("Failed to close response body: %v", err)
		}
	}()

	if res.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(res.Body)
		err = fmt.Errorf("unexpected status code: %d, body: %s", res.StatusCode, string(data))
		logrus.WithError(err).Errorf("DetectLangAPI failed with status: %s", res.Status)
		return "", err
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		logrus.WithError(err).Error("Failed to read DetectLangAPI response")
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	var response DetectLangRes
	if err = json.Unmarshal(data, &response); err != nil {
		logrus.WithError(err).Error("Failed to unmarshal DetectLangAPI response")
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if response.LanguageCode == "" {
		err = fmt.Errorf("no language detected")
		logrus.WithError(err).Error("DetectLangAPI returned empty language code")
		return "", err
	}

	logrus.Infof("Detected language: %s", response.LanguageCode)
	return response.LanguageCode, nil
}
